package mssql

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"io/ioutil"
	nurl "net/url"

	multierror "github.com/hashicorp/go-multierror"
	"github.com/shaoding/migrate"
	"github.com/shaoding/migrate/database"
)

type SQLError interface {
	SQLErrorNumber() int32
	SQLErrorMessage() string
}

var b2i = map[bool]int8{false: 0, true: 1}

var i2b = []bool{false, true}

func init() {
	fmt.Println("register sqlserver")
	db := Mssql{}
	database.Register("sqlserver", &db)
}

var DefaultMigrationsTable = "schema_migrations"

var (
	ErrNilConfig      = fmt.Errorf("no config")
	ErrNoDatabaseName = fmt.Errorf("no database name")
	ErrNoSchema       = fmt.Errorf("no schema")
	ErrDatabaseDirty  = fmt.Errorf("database is dirty")
)

type Config struct {
	MigrationsTable string
	DatabaseName    string
	SchemaName      string
}

type Mssql struct {
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

	query := `SELECT DB_NAME()`
	var databaseName string
	if err := instance.QueryRow(query).Scan(&databaseName); err != nil {
		return nil, &database.Error{OrigErr: err, Query: []byte(query)}
	}

	if len(databaseName) == 0 {
		return nil, ErrNoDatabaseName
	}

	config.DatabaseName = databaseName

	query = `SELECT SCHEMA_NAME()`
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

	ms := &Mssql{
		conn:   conn,
		db:     instance,
		config: config,
	}

	if err := ms.ensureVersionTable(); err != nil {
		return nil, err
	}

	return ms, nil
}

func (ms *Mssql) Open(url string) (database.Driver, error) {
	purl, err := nurl.Parse(url)
	if err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlserver", migrate.FilterCustomQuery(purl).String())
	if err != nil {
		return nil, err
	}

	migrationsTable := purl.Query().Get("x-migrations-table")

	msi, err := WithInstance(db, &Config{
		DatabaseName:    purl.Path,
		MigrationsTable: migrationsTable,
	})

	if err != nil {
		return nil, err
	}

	return msi, nil
}

func (ms *Mssql) Close() error {
	connErr := ms.conn.Close()
	dbErr := ms.db.Close()
	if connErr != nil || dbErr != nil {
		return fmt.Errorf("conn: %v, db: %v", connErr, dbErr)
	}
	return nil
}

// https://www.postgresql.org/docs/9.6/static/explicit-locking.html#ADVISORY-LOCKS
func (ms *Mssql) Lock() error {
	if ms.isLocked {
		return database.ErrLocked
	}

	_, err := database.GenerateAdvisoryLockId(ms.config.DatabaseName, ms.config.SchemaName)
	if err != nil {
		return err
	}

	// This will either obtain the lock immediately and return true,
	// or return false if the lock cannot be acquired immediately.
	// query := `SELECT pg_advisory_lock($1)`
	// if _, err := p.conn.ExecContext(context.Background(), query, aid); err != nil {
	// 	return &database.Error{OrigErr: err, Err: "try lock failed", Query: []byte(query)}
	// }

	ms.isLocked = true
	return nil
}

func (ms *Mssql) Unlock() error {
	if !ms.isLocked {
		return nil
	}

	_, err := database.GenerateAdvisoryLockId(ms.config.DatabaseName, ms.config.SchemaName)
	if err != nil {
		return err
	}

	// query := `SELECT pg_advisory_unlock($1)`
	// if _, err := p.conn.ExecContext(context.Background(), query, aid); err != nil {
	// 	return &database.Error{OrigErr: err, Query: []byte(query)}
	// }
	ms.isLocked = false
	return nil
}

func (ms *Mssql) Run(migration io.Reader) error {
	migr, err := ioutil.ReadAll(migration)
	if err != nil {
		return err
	}

	query := string(migr[:])
	if _, err := ms.conn.ExecContext(context.Background(), query); err != nil {
		if sqlError, ok := err.(SQLError); ok {
			return database.Error{OrigErr: err, Err: sqlError.SQLErrorMessage(), Query: migr}
		}
		return database.Error{OrigErr: err, Err: "migration failed", Query: migr}
	}

	return nil
}

func (ms *Mssql) SetVersion(version int, dirty bool) error {
	tx, err := ms.conn.BeginTx(context.Background(), &sql.TxOptions{})
	if err != nil {
		return &database.Error{OrigErr: err, Err: "transaction start failed"}
	}

	query := "TRUNCATE TABLE " + ms.config.MigrationsTable
	if _, err := tx.Exec(query); err != nil {
		tx.Rollback()
		return &database.Error{OrigErr: err, Query: []byte(query)}
	}

	if version >= 0 {
		query = fmt.Sprintf(`INSERT INTO %s (version, dirty) VALUES (%d, '%d')`, ms.config.MigrationsTable, version, b2i[dirty])
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

func (ms *Mssql) Version() (version int, dirty bool, err error) {
	query := "SELECT TOP 1 version, dirty FROM " + ms.config.MigrationsTable
	err = ms.conn.QueryRowContext(context.Background(), query).Scan(&version, &dirty)
	switch {
	case err == sql.ErrNoRows:
		return database.NilVersion, false, nil

	case err != nil:
		if _, ok := err.(SQLError); ok {
			return database.NilVersion, false, nil
		}
		return 0, false, &database.Error{OrigErr: err, Query: []byte(query)}

	default:
		return version, dirty, nil
	}
}

func (ms *Mssql) Drop() error {
	// select all tables in current schema
	query := fmt.Sprintf(`SELECT TABLE_NAME FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_TYPE = 'BASE TABLE' AND TABLE_CATALOG='%s'`, ms.config.DatabaseName)
	tables, err := ms.conn.QueryContext(context.Background(), query)
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

	if len(tableNames) > 0 {
		// delete one by one ...
		for _, t := range tableNames {
			query = "DROP TABLE IF EXISTS " + t
			if _, err := ms.conn.ExecContext(context.Background(), query); err != nil {
				return &database.Error{OrigErr: err, Query: []byte(query)}
			}
		}
	}

	return nil
}

// ensureVersionTable checks if versions table exists and, if not, creates it.
// Note that this function locks the database, which deviates from the usual
// convention of "caller locks" in the Postgres type.
func (ms *Mssql) ensureVersionTable() (err error) {
	if err = ms.Lock(); err != nil {
		return err
	}

	defer func() {
		if e := ms.Unlock(); e != nil {
			if err == nil {
				err = e
			} else {
				err = multierror.Append(err, e)
			}
		}
	}()

	query := "IF NOT EXISTS (SELECT * FROM sysobjects WHERE name='" + ms.config.MigrationsTable + "' and xtype='U') " + "CREATE TABLE " + ms.config.MigrationsTable + "(version bigint not null, dirty bit not null, primary key (version))"
	if _, err = ms.conn.ExecContext(context.Background(), query); err != nil {
		return &database.Error{OrigErr: err, Query: []byte(query)}
	}

	return nil
}
