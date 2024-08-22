package mssql

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/lnksnk/lnksnk/database"
	//helper registration sql server driver

	_ "github.com/microsoft/go-mssqldb"
	"github.com/microsoft/go-mssqldb/azuread"
	"github.com/pkg/errors"
)

// Open -wrap sql.Open("sqlserver", datasource)
func Open(datasource string) (*sql.DB, error) {
	var tlsversion = ""
	for _, dtasrc := range strings.Split(datasource, ";") {
		if strings.HasPrefix(dtasrc, "tlsmin=") {
			if tlsversion = strings.TrimSpace(dtasrc[len("tlsmin="):]); tlsversion == "" {
				tlsversion = "1.0"
				datasource = strings.Replace(datasource, "tlsmin=", "tlsmin="+tlsversion, 1)
			}
		}
	}
	if tlsversion == "" {
		tlsversion = "1.0"
		datasource += ";" + "tlsmin=" + tlsversion
	}

	return sql.Open("sqlserver", datasource)
}

// Open -wrap sql.Open("azure", datasource)
func OpenAzure(datasource string) (*sql.DB, error) {
	var tlsversion = ""
	for _, dtasrc := range strings.Split(datasource, ";") {
		if strings.HasPrefix(dtasrc, "tlsmin=") {
			if tlsversion = strings.TrimSpace(dtasrc[len("tlsmin="):]); tlsversion == "" {
				tlsversion = "1.0"
				datasource = strings.Replace(datasource, "tlsmin=", "tlsmin="+tlsversion, 1)
			}
		}
	}
	if tlsversion == "" {
		tlsversion = "1.0"
		datasource += ";" + "tlsmin=" + tlsversion
	}
	return sql.Open(azuread.DriverName, datasource)
}

func parseSqlParam(totalArgs int) (s string) {
	return ("@p" + fmt.Sprintf("%d", totalArgs))
}

func init() {
	//fmt.Println(mssql.CopyIn("test_table", mssql.BulkOptions{CheckConstraints: true}, "test_varchar", "test_nvarchar", "test_float", "test_bigint"))
	database.GLOBALDBMS().RegisterDriver("sqlserver", func(datasource string, a ...interface{}) (db *sql.DB, err error) {
		db, err = Open(datasource)
		if err != nil {
			return nil, errors.Wrap(err, "create db conn pool")
		}
		return
	}, parseSqlParam)

	database.GLOBALDBMS().RegisterDriver("mssql", func(datasource string, a ...interface{}) (db *sql.DB, err error) {
		db, err = Open(datasource)
		if err != nil {
			return nil, errors.Wrap(err, "create db conn pool")
		}
		return
	}, parseSqlParam)

	database.GLOBALDBMS().RegisterDriver("sqlserver-azure", func(datasource string, a ...interface{}) (db *sql.DB, err error) {
		db, err = OpenAzure(datasource)
		if err != nil {
			return nil, errors.Wrap(err, "create db conn pool")
		}
		return
	}, parseSqlParam)

	database.GLOBALDBMS().RegisterDriver("mssql-azure", func(datasource string, a ...interface{}) (db *sql.DB, err error) {
		db, err = OpenAzure(datasource)
		if err != nil {
			return nil, errors.Wrap(err, "create db conn pool")
		}
		return
	}, parseSqlParam)
}
