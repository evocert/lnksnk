package status

import (
	"encoding/json"

	"github.com/lnksnk/lnksnk/database"
	"github.com/lnksnk/lnksnk/database/dbserve"
	"github.com/lnksnk/lnksnk/fsutils"
	"github.com/lnksnk/lnksnk/serve/serveio"
)

var aliascmdstatus dbserve.AliasCommandFunc = func(alias, path, ext string, dbhnl *database.DBMSHandler, w serveio.Writer, r serveio.Reader, fs *fsutils.FSUtils) (err error) {
	if prms := dbhnl.Params(); prms != nil {
		var errorsfnd []interface{} = nil
		var warnings []interface{} = nil
		isalias := alias != ""
		if isalias {
			sts, stserr := dbhnl.Status(alias)
			if stserr == nil {
				if len(sts) > 0 {
					enc := json.NewEncoder(w)
					enc.Encode(sts)
					return
				}
				warnings = append(warnings, "No status info")
			} else if stserr != nil {
				errorsfnd = append(errorsfnd, "Status-err:"+stserr.Error())
			}
		}
		if !isalias {
			errorsfnd = append(errorsfnd, "No alias provided")
		}
		enc := json.NewEncoder(w)
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

func init() {
	dbserve.HandleCommand("status", aliascmdstatus)
}
