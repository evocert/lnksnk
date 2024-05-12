package register

import (
	"encoding/json"

	"github.com/evocert/lnksnk/database"
	"github.com/evocert/lnksnk/database/dbserve"
	"github.com/evocert/lnksnk/fsutils"
	"github.com/evocert/lnksnk/serve/serveio"
)

var cmdregister dbserve.CommandFunc = func(path, ext string, dbhnl *database.DBMSHandler, w serveio.Writer, r serveio.Reader, fs *fsutils.FSUtils) (err error) {
	if prms := dbhnl.Params(); prms != nil {
		isname, name, isdatasource, datasource, isdriver, driver, connect := prms.ContainsParameter("name") && prms.StringParameter("name", "") != "", prms.StringParameter("name", ""), prms.ContainsParameter("datasource") && prms.StringParameter("datasource", "") != "", prms.StringParameter("datasource", ""), prms.ContainsParameter("driver") && prms.StringParameter("driver", "") != "", prms.StringParameter("driver", ""), prms.StringParameter("connect", "") == "Y"
		var errorsfnd []interface{} = nil
		var warnings []interface{} = nil
		if isname && isdatasource && isdriver {
			if connect {
				if err, _ = dbhnl.TryConnect(driver, datasource).(error); err != nil {
					errorsfnd = append(errorsfnd, "connect-err: "+err.Error())
					connect = false
				}
				if connect {
					if dbhnl.Register(name, driver, datasource) {
						err = w.Print("{}")
						return
					}
					warnings = append(warnings, "Unable to register")
				}
			} else {
				if dbhnl.Register(name, driver, datasource) {
					err = w.Print("{}")
					return
				}
				warnings = append(warnings, "Unable to register")
			}
		}
		enc := json.NewEncoder(w)

		if !isname {
			errorsfnd = append(errorsfnd, "No db-alias provided")
		}
		if !isdriver {
			errorsfnd = append(errorsfnd, "No driver selected")
		}
		if !isdatasource {
			errorsfnd = append(errorsfnd, "No datasource")
		}
		mp := map[string]interface{}{}
		if len(errorsfnd) > 0 {
			mp["err"] = errorsfnd
		}
		if len(warnings) > 0 {
			mp["warn"] = warnings
		}
		err = enc.Encode(mp)
	}
	return
}

var cmdunregister dbserve.CommandFunc = func(path, ext string, dbhnl *database.DBMSHandler, w serveio.Writer, r serveio.Reader, fs *fsutils.FSUtils) (err error) {
	if prms := dbhnl.Params(); prms != nil {
		isname, name := prms.ContainsParameter("name") && prms.StringParameter("name", "") != "", prms.StringParameter("name", "")
		var errorsfnd []interface{} = nil
		if isname {
			if dbhnl.Unregister(name) {
				err = w.Print("[]")
				return
			}
			errorsfnd = append(errorsfnd, "fialed to unregister")
		}
		enc := json.NewEncoder(w)

		if !isname {
			errorsfnd = append(errorsfnd, "No db-alias provided")
		}
		err = enc.Encode(map[string]interface{}{"err": errorsfnd})
	}
	return
}

func init() {
	dbserve.HandleCommand("register", cmdregister)
	dbserve.HandleCommand("unregister", cmdunregister)
}
