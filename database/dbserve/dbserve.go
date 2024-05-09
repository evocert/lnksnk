package dbserve

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"

	"github.com/evocert/lnksnk/database"
	"github.com/evocert/lnksnk/fsutils"
	"github.com/evocert/lnksnk/iorw/active"
	"github.com/evocert/lnksnk/parameters"
	"github.com/evocert/lnksnk/serve/serveio"
)

func ServeRequest(w serveio.Writer, r serveio.Reader, a ...interface{}) {
	ctx := r.Context()
	var callPrepStatement database.StatementHandlerFunc = nil
	var runtime active.Runtime = nil
	var params *parameters.Parameters = nil
	var fs *fsutils.FSUtils = nil
	var dbhndl *database.DBMSHandler
	var path string = ""
	if al := len(a); al > 0 {
		ai := 0
		for ai < al {
			d := a[ai]
			if dpath, _ := d.(string); dpath != "" && path == "" {
				path = dpath
				a = append(a[:ai], a[ai+1:]...)
				al--
				continue
			}
			if ddbhndl, _ := d.(*database.DBMSHandler); ddbhndl != nil && dbhndl == nil {
				dbhndl = ddbhndl
				a = append(a[:ai], a[ai+1:]...)
				al--
				continue
			}
			if dparams, _ := d.(*parameters.Parameters); dparams != nil && params == nil {
				params = dparams
				a = append(a[:ai], a[ai+1:]...)
				al--
				continue
			}
			if dcallPrepStmnt, _ := d.(database.StatementHandlerFunc); dcallPrepStmnt != nil && callPrepStatement == nil {
				callPrepStatement = dcallPrepStmnt
				a = append(a[:ai], a[ai+1:]...)
				al--
				continue
			}
			if druntime, _ := d.(active.Runtime); druntime != nil && runtime == nil {
				runtime = druntime
				a = append(a[:ai], a[ai+1:]...)
				al--
				continue
			}
			if dfs, _ := d.(*fsutils.FSUtils); dfs != nil && fs == nil {
				fs = dfs
				a = append(a[:ai], a[ai+1:]...)
				al--
				continue
			}
			ai++
		}
	}

	defer params.CleanupParameters()

	if dbhndl == nil {
		if ctx == nil {
			ctx = context.Background()
		}
		if params == nil {
			params = parameters.NewParameters()
			parameters.LoadParametersFromHTTPRequest(params, r.HttpR())
			parameters.LoadParametersFromRawURL(params, path)
		}
		dbhndl = database.GLOBALDBMS().DBMSHandler(ctx, runtime, params, nil, fs, callPrepStatement)
	}
	if dbhndl != nil {
		if strings.Contains(path, "/db/") || strings.Contains(path, "/db-") {
			pathext := filepath.Ext(path)
			if pathext == "" {
				pathext = ".json"
			}
			if path = path[strings.Index(path, "/db")+len("/db"):]; path != "" && validdbrequestexts[pathext] {
				if w != nil {
					if pathext == ".json" {
						w.Header().Set("Content-Type", "application/json; charset=utf-8")
					}
					if pathext == ".js" {
						w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
					}
				}
				if pthsepi := strings.Index(path, "/"); path[0:1] == "-" && pthsepi == -1 {
					if path = path[1:]; validdbcommands[path] {
						if path == "commands" {
							cmds := []string{}
							for cmd := range validdbcommands {
								if cmd == "commands" {
									continue
								}
								cmds = append(cmds, cmd)
							}
							enc := json.NewEncoder(w)
							enc.Encode(cmds)
							return
						}
						if path == "connections" {
							enc := json.NewEncoder(w)
							enc.Encode(dbhndl.Connections())
							return
						}
						if path == "drivers" {
							enc := json.NewEncoder(w)
							enc.Encode(dbhndl.Drivers())
							return
						}
					}
					return
				}
				if pthi := strings.Index(path, "/"); pthi > 2 {
					if path = path[pthi+1:]; path != "" {

					}
				}
			}
		}
	}
}

var validdbrequestexts = map[string]bool{".js": true, ".json": true}
var validdbcommands = map[string]bool{"connections": true, "drivers": true, "commands": true}
