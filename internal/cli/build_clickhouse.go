// +build clickhouse

package cli

import (
	_ "github.com/shaoding/migrate/database/clickhouse"
	_ "github.com/kshvakov/clickhouse"
)
