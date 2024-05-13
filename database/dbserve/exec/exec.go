package exec

import (
	"encoding/json"
	"path/filepath"
	"strings"

	"github.com/evocert/lnksnk/database"
	"github.com/evocert/lnksnk/database/dbserve"
	"github.com/evocert/lnksnk/fsutils"
	"github.com/evocert/lnksnk/serve/serveio"
)

var aliascmdexec dbserve.AliasCommandFunc = func(alias, path, ext string, dbhnl *database.DBMSHandler, w serveio.Writer, r serveio.Reader, fs *fsutils.FSUtils) (err error) {
	var qryarr []interface{} = nil
	if httpr := r.HttpR(); httpr != nil {
		if cnttype := httpr.Header.Get("Content-Type"); strings.Contains(cnttype, "application/json") {
			if bdy := httpr.Body; bdy != nil {
				var qryref interface{} = nil

				if err = json.NewDecoder(bdy).Decode(&qryref); err == nil {
					if qryarrd, _ := qryref.([]interface{}); len(qryarrd) > 0 {
						qryarr = append(qryarr, qryarrd...)
						dbhnl.Query(alias, qryarr...)
					}
					if qrymp, _ := qryref.(map[string]interface{}); len(qrymp) > 0 {
						var qryarr []interface{} = nil
						for qryk, qryv := range qrymp {
							if qryk == "query" {
								qryarr = append(qryarr, qryv)
								delete(qrymp, qryk)
								continue
							}
						}
						if len(qrymp) > 0 {
							qryarr = append(qryarr, qrymp)
						}
					}
				}
			}
		} else {
			if params := dbhnl.Params(); params != nil {
				if (params.ContainsParameter("qry") && params.Type("qry") == "std") || (params.ContainsParameter("query") && params.Type("query") == "std") {
					for _, qry := range append(params.Parameter("qry"), params.Parameter("query")...) {
						qryarr = append(qryarr, qry)
					}
				}
			}
		}
		if path != "" && path[len(path)-1] != '/' && len(qryarr) == 0 {
			pathext := filepath.Ext(path)
			if pathext != "" {
				if pathext != ".sql" {
					path = path[:len(path)-len(pathext)] + ".sql"
				}
			} else {
				path = path + ".sql"
			}
			qryarr = append(qryarr, path)
		}
	}
	if len(qryarr) > 0 {
		var errfound error = nil
		qryarr = append(qryarr, map[string]interface{}{
			"error": func(err error) {
				errfound = err
			},
		})
		exec := dbhnl.Execute(alias, qryarr...)
		defer exec.Close()
		if errfound == nil {
			err = w.Print("{}")
			return
		}
		if errfound != nil {
			enc := json.NewEncoder(w)
			err = enc.Encode(map[string]interface{}{"err": errfound.Error()})
			return
		}
	}
	return
}

func init() {
	dbserve.HandleCommand("exec", aliascmdexec)
}
