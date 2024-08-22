package sqlite

import (
	"database/sql"
	"fmt"

	"github.com/lnksnk/lnksnk/database"
	//helper registration sqlite driver

	//_ "github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"
	_ "modernc.org/sqlite"
)

// Open -wrap sql.Open("sqlite", datasource)
func Open(datasource string) (*sql.DB, error) {
	return sql.Open("sqlite", datasource)
}

func parseSqlParam(totalArgs int) (s string) {
	return "$" + fmt.Sprintf("%d", totalArgs+1)
}

func init() {
	database.GLOBALDBMS().RegisterDriver("sqlite", func(datasource string, a ...interface{}) (db *sql.DB, err error) {
		if datasource == ":memory:" {
			datasource = "file::memory:?mode=memory"
		}
		db, err = Open(datasource)
		if err != nil {
			return nil, errors.Wrap(err, "create db conn")
		}
		return
	}, parseSqlParam)
}
