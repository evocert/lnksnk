package sessioning

import (
	"github.com/lnksnk/lnksnk/database"
	"github.com/lnksnk/lnksnk/iorw/active"
	"github.com/lnksnk/lnksnk/parameters"
)

type Session struct {
	active.Runtime
	DBMS   *database.DBMSHandler
	PARAMS *parameters.Parameters
}

func NewSession() (ssn *Session) {
	ssn = &Session{}
	return
}

func (ssn *Session) InvokeFunction(v interface{}, a ...interface{}) (result interface{}) {
	if ssn != nil {
		if rntime := ssn.Runtime; rntime != nil {
			result = rntime.InvokeFunction(v, a...)
		}
	}
	return
}

func (ssn *Session) Close() (err error) {
	if ssn != nil {
		if dbms := ssn.DBMS; dbms != nil {
			ssn.DBMS = nil
			dbms.Dispose()
		}
		if prms := ssn.PARAMS; prms != nil {
			ssn.PARAMS = nil
			prms.CleanupParameters()
		}
	}
	return
}
