package oracle

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"io/ioutil"
	nurl "net/url"

	"github.com/shaoding/migrate"
	"github.com/shaoding/migrate/database"
	multierror "github.com/hashicorp/go-multierror"
)

type OraErr interface {
	Code() int
	Error() string
	Message() string
}

var b2i = map[bool]int8{false: 0, true: 1}

var i2b = []bool{false, true}

func init() {
	db := Oracle{}
	database.Register("goracle", &db)
}

var DefaultMigrationsTable = "schema_migrations"

var (
	ErrNilConfig     = fmt.Errorf("no config")
	ErrNoSchema      = fmt.Errorf("no schema")
	ErrDatabaseDirty = fmt.Errorf("database is dirty")
)

type Config struct {
	MigrationsTable string
	SchemaName      string
}

type Oracle struct {
	// Locking and unlocking need to use the same connection
	conn     *sql.Conn
	db       *sql.DB
	isLocked bool

	// Open and WithInstance need to guarantee that config is never nil
	config *Config
}

func WithInstance(instance *sql.DB, config *Config) (database.Driver, error) {
	if config == nil {
		return nil, ErrNilConfig
	}

	if err := instance.Ping(); err != nil {
		return nil, err
	}

	query := `select sys_context('userenv','db_name') from dual`
	var schemaName string
	if err := instance.QueryRow(query).Scan(&schemaName); err != nil {
		return nil, &database.Error{OrigErr: err, Query: []byte(query)}
	}

	if len(schemaName) == 0 {
		return nil, ErrNoSchema
	}

	config.SchemaName = schemaName

	if len(config.MigrationsTable) == 0 {
		config.MigrationsTable = DefaultMigrationsTable
	}

	conn, err := instance.Conn(context.Background())

	if err != nil {
		return nil, err
	}

	ora := &Oracle{
		conn:   conn,
		db:     instance,
		config: config,
	}

	if err := ora.ensureVersionTable(); err != nil {
		return nil, err
	}

	return ora, nil
}

func (ora *Oracle) Open(url string) (database.Driver, error) {
	purl, err := nurl.Parse(url)
	if err != nil {
		return nil, err
	}

	db, err := sql.Open("goracle", migrate.FilterCustomQuery(purl).String())
	if err != nil {
		return nil, err
	}

	migrationsTable := purl.Query().Get("x-migrations-table")

	orai, err := WithInstance(db, &Config{
		SchemaName:      purl.Path,
		MigrationsTable: migrationsTable,
	})

	if err != nil {
		return nil, err
	}

	return orai, nil
}

func (ora *Oracle) Close() error {
	connErr := ora.conn.Close()
	dbErr := ora.db.Close()
	if connErr != nil || dbErr != nil {
		return fmt.Errorf("conn: %v, db: %v", connErr, dbErr)
	}
	return nil
}

func (ora *Oracle) Lock() error {
	if ora.isLocked {
		return database.ErrLocked
	}

	_, err := database.GenerateAdvisoryLockId(ora.config.SchemaName)
	if err != nil {
		return err
	}

	// This will either obtain the lock immediately and return true,
	// or return false if the lock cannot be acquired immediately.
	// query := `SELECT pg_advisory_lock($1)`
	// if _, err := p.conn.ExecContext(context.Background(), query, aid); err != nil {
	// 	return &database.Error{OrigErr: err, Err: "try lock failed", Query: []byte(query)}
	// }

	ora.isLocked = true
	return nil
}

func (ora *Oracle) Unlock() error {
	if !ora.isLocked {
		return nil
	}

	_, err := database.GenerateAdvisoryLockId(ora.config.SchemaName)
	if err != nil {
		return err
	}

	// query := `SELECT pg_advisory_unlock($1)`
	// if _, err := p.conn.ExecContext(context.Background(), query, aid); err != nil {
	// 	return &database.Error{OrigErr: err, Query: []byte(query)}
	// }
	ora.isLocked = false
	return nil
}

func (ora *Oracle) Run(migration io.Reader) error {
	migr, err := ioutil.ReadAll(migration)
	if err != nil {
		return err
	}

	query := string(migr[:])
	if _, err := ora.conn.ExecContext(context.Background(), query); err != nil {
		if sqlError, ok := err.(OraErr); ok {
			return database.Error{OrigErr: err, Err: sqlError.Message(), Query: migr}
		}
		return database.Error{OrigErr: err, Err: "migration failed", Query: migr}
	}

	return nil
}

func (ora *Oracle) SetVersion(version int, dirty bool) error {
	tx, err := ora.conn.BeginTx(context.Background(), &sql.TxOptions{})
	if err != nil {
		return &database.Error{OrigErr: err, Err: "transaction start failed"}
	}

	query := "TRUNCATE TABLE \"" + ora.config.MigrationsTable + "\""
	if _, err := tx.Exec(query); err != nil {
		tx.Rollback()
		return &database.Error{OrigErr: err, Query: []byte(query)}
	}

	if version >= 0 {
		query = fmt.Sprintf(`INSERT INTO "%s" (version, dirty) VALUES (%d, '%d')`, ora.config.MigrationsTable, version, b2i[dirty])
		if _, err := tx.Exec(query); err != nil {
			tx.Rollback()
			return &database.Error{OrigErr: err, Query: []byte(query)}
		}
	}

	if err := tx.Commit(); err != nil {
		return &database.Error{OrigErr: err, Err: "transaction commit failed"}
	}

	return nil
}

func (ora *Oracle) Version() (version int, dirty bool, err error) {
	query := "SELECT version, dirty FROM \"" + ora.config.MigrationsTable + "\" WHERE ROWNUM = 1"
	err = ora.conn.QueryRowContext(context.Background(), query).Scan(&version, &dirty)
	switch {
	case err == sql.ErrNoRows:
		return database.NilVersion, false, nil

	case err != nil:
		if _, ok := err.(OraErr); ok {
			return database.NilVersion, false, nil
		}
		return 0, false, &database.Error{OrigErr: err, Query: []byte(query)}

	default:
		return version, dirty, nil
	}
}

func (ora *Oracle) Drop() error {
	// select all tables in current schema
	query := fmt.Sprintf(`SELECT TABLE_NAME FROM USER_TABLES`)
	tables, err := ora.conn.QueryContext(context.Background(), query)
	if err != nil {
		return &database.Error{OrigErr: err, Query: []byte(query)}
	}
	defer tables.Close()

	// delete one table after another
	tableNames := make([]string, 0)
	for tables.Next() {
		var tableName string
		if err := tables.Scan(&tableName); err != nil {
			return err
		}
		if len(tableName) > 0 {
			tableNames = append(tableNames, tableName)
		}
	}

	// store porcedure
	query = `CREATE OR REPLACE
procedure proc_dropifexists(
    TABLE_NAME in VARCHAR2
) is
 l_count NUMBER;
BEGIN
 SELECT COUNT(1)
 INTO l_count
 FROM USER_TABLES
 WHERE table_name = TABLE_NAME;
 IF l_count > 0 THEN
 EXECUTE IMMEDIATE 'Drop table '||'"'||TABLE_NAME||'"'||' CASCADE CONSTRAINTS';
END IF;
END;`
	if _, err = ora.conn.ExecContext(context.Background(), query); err != nil {
		return &database.Error{OrigErr: err, Query: []byte(query)}
	}

	if len(tableNames) > 0 {
		// delete one by one ...
		for _, t := range tableNames {
			query = "BEGIN proc_dropifexists('" + t + "'); END;"
			if _, err := ora.conn.ExecContext(context.Background(), query); err != nil {
				return &database.Error{OrigErr: err, Query: []byte(query)}
			}
		}
	}

	return nil
}

// ensureVersionTable checks if versions table exists and, if not, creates it.
// Note that this function locks the database, which deviates from the usual
// convention of "caller locks" in the Postgres type.
func (ora *Oracle) ensureVersionTable() (err error) {
	if err = ora.Lock(); err != nil {
		return err
	}

	defer func() {
		if e := ora.Unlock(); e != nil {
			if err == nil {
				err = e
			} else {
				err = multierror.Append(err, e)
			}
		}
	}()

	// store porcedure
	query := `CREATE OR REPLACE
	procedure proc_createifnotexists(
		TABLE_NAME in VARCHAR2
	) authid current_user is
	nCount NUMBER;
	v_sql LONG;
	begin
	SELECT count(*) into nCount FROM user_tables where table_name = TABLE_NAME;
	IF(nCount <= 0)
	THEN
	v_sql:='create table '||'"'||TABLE_NAME||'"'||' (VERSION NUMBER(20) NOT NULL PRIMARY KEY, DIRTY NUMBER(1) NOT NULL)';
	execute immediate v_sql;
	END IF;
	end;`
	if _, err = ora.conn.ExecContext(context.Background(), query); err != nil {
		return &database.Error{OrigErr: err, Query: []byte(query)}
	}

	query = "BEGIN proc_createifnotexists('" + ora.config.MigrationsTable + "'); END;"
	if _, err = ora.conn.ExecContext(context.Background(), query); err != nil {
		return &database.Error{OrigErr: err, Query: []byte(query)}
	}

	return nil
}
