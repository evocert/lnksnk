package query

import (
	"encoding/json"
	"path/filepath"
	"strings"

	"github.com/lnksnk/lnksnk/database"
	"github.com/lnksnk/lnksnk/database/dbserve"
	"github.com/lnksnk/lnksnk/fsutils"
	"github.com/lnksnk/lnksnk/iorw"
	"github.com/lnksnk/lnksnk/serve/serveio"
)

var aliascmdquery dbserve.AliasCommandFunc = func(alias, path, ext string, dbhnl *database.DBMSHandler, w serveio.Writer, r serveio.Reader, fs *fsutils.FSUtils) (err error) {
	var qryarr []interface{} = nil
	var layout = ""
	var cols []string = nil
	var errorsfnd []interface{} = nil
	var warnings []interface{} = nil
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
							if qryk == "cols" || qryk == "columns" {
								if colsd, _ := qryv.([]string); len(colsd) > 0 && len(cols) == 0 {
									cols = append(cols, colsd...)
									delete(qrymp, qryk)
									continue
								}
								if colsd, _ := qryv.([]interface{}); len(colsd) > 0 && len(cols) == 0 {
									for _, cold := range colsd {
										if col, _ := cold.(string); col != "" {
											cols = append(cols, col)
										}
									}
									delete(qrymp, qryk)
									continue
								}
								delete(qrymp, qryk)
								continue
							}
							if qryk == "layout" {
								if layoutd, _ := qryv.(string); layoutd != "" && layout == "" {
									layout = layoutd
								}
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
				for _, col := range append(params.Parameter("col"), append(params.Parameter("cols"), params.Parameter("columns")...)...) {
					cols = append(cols, strings.Split(col, ",")...)
				}
				if (params.ContainsParameter("qry") && params.Type("qry") == "std") || (params.ContainsParameter("query") && params.Type("query") == "std") {
					for _, qry := range append(params.Parameter("qry"), params.Parameter("query")...) {
						qryarr = append(qryarr, qry)
					}
				}
				layout = strings.TrimFunc(params.StringParameter("layout", ""), iorw.IsSpace)
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
		var rec = dbhnl.Query(alias, qryarr...)
		if rec != nil && errfound == nil {
			err = rec.ToJSON(w, layout, cols...)
			return
		}
		if errfound != nil {
			err = errfound
			errorsfnd = append(errorsfnd, err.Error())
		}
		if len(errorsfnd) > 0 || len(warnings) > 0 {
			mp := map[string]interface{}{}
			if len(errorsfnd) > 0 {
				mp["err"] = errorsfnd
			}
			if len(warnings) > 0 {
				mp["warn"] = warnings
			}
			enc := json.NewEncoder(w)
			err = enc.Encode(mp)
		}
	}
	return
}

func init() {
	dbserve.HandleCommand("query", aliascmdquery)
}
