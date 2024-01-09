package ora

import (
	"database/sql"
	"fmt"
	"net/url"

	"github.com/evocert/lnksnk/database"
	//helper registration oracle server driver
	"github.com/pkg/errors"
	_ "github.com/sijms/go-ora/v2"
)

// Open -wrap sql.Open("oracle", datasource)
func Open(oraname, datasource string) (*sql.DB, error) {
	if url, _ := url.ParseRequestURI(datasource); url != nil {
		return sql.Open(oraname, datasource)
	}
	return nil, nil
}
func parseSqlParam(totalArgs int) string {
	return ":" + fmt.Sprintf("%d", totalArgs+1)
}
func init() {
	database.GLOBALDBMS().RegisterDriver("oracle", func(datasource string, a ...interface{}) (db *sql.DB, err error) {
		db, err = Open("oracle", datasource)
		if err != nil {
			return nil, errors.Wrap(err, "create db conn pool")
		}
		return
	}, parseSqlParam)
	database.GLOBALDBMS().RegisterDriver("oracle:ext", func(datasource string, a ...interface{}) (db *sql.DB, err error) {
		db, err = Open("oracle", datasource)
		if err != nil {
			return nil, errors.Wrap(err, "create db conn pool")
		}
		return
	}, parseSqlParam)
}
