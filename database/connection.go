package database

import (
	"database/sql"
	"strings"
	"sync"
	"time"
)

type Connection struct {
	lstcheck               time.Time
	suggestedmaxcnss       int64
	cnsccount              int64
	dbms                   *DBMS
	driverName, dataSource string
	args                   []interface{}
	dbinvklck              *sync.RWMutex
	dbParseSqlParam        func(totalArgs int) (s string)
	dbinvoker              func(string, ...interface{}) (*sql.DB, error)
	sqldb                  *sql.DB
}

func NewConnection(dbms *DBMS, driverName, dataSource string) (cn *Connection) {
	if dbms != nil {
		cn = &Connection{dbms: dbms, driverName: driverName, dataSource: dataSource, dbinvklck: &sync.RWMutex{}}
		cn.dbinvoker, cn.dbParseSqlParam = dbms.DriverCnInvoker(driverName)
	}
	return
}

func (cn *Connection) isRemote() (isremote bool) {
	if cn != nil {
		isremote = strings.HasPrefix(cn.dataSource, "http://") || strings.HasPrefix(cn.dataSource, "https://") || strings.HasPrefix(cn.dataSource, "ws://") || strings.HasPrefix(cn.dataSource, "wss://")
	}
	return
}

func (cn *Connection) Stmnt() (stmnt *Statement) {
	if cn != nil {
		stmnt = NewStatement(cn)
	}
	return
}

func (cn *Connection) Dispose() (err error) {
	if cn != nil {
		if cn.sqldb != nil {
			err = cn.sqldb.Close()
			cn.sqldb = nil
		}
		if cn.args != nil {
			cn.args = nil
		}
		if cn.dbinvoker != nil {
			cn.dbinvoker = nil
		}
		if cn.dbms != nil {
			cn.dbms = nil
		}
		cn = nil
	}
	return
}

func (cn *Connection) DbInvoke() (db *sql.DB, dberr error) {
	if cn != nil {
		if dbms, sqldb, dbinvoker := cn.dbms, cn.sqldb, cn.dbinvoker; dbms != nil {
			if db = sqldb; db == nil {
				if dbinvoker != nil {
					if db, dberr = dbinvoker(cn.dataSource); dberr == nil {
						cn.sqldb = db
						cn.cnsccount = 0
						cn.lstcheck = time.Now()
						cn.suggestedmaxcnss = 0
						if sqldb != nil {
							sqldb.Close()
						}
					}
				}
			}
		}
	}
	return
}
