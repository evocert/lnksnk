package dbserve

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"sync"

	"github.com/lnksnk/lnksnk/database"
	"github.com/lnksnk/lnksnk/fsutils"
	"github.com/lnksnk/lnksnk/iorw"
	"github.com/lnksnk/lnksnk/iorw/active"
	"github.com/lnksnk/lnksnk/parameters"
	"github.com/lnksnk/lnksnk/serve/serveio"
)

func ServeRequest(prefix string, w serveio.Writer, r serveio.Reader, a ...interface{}) (bool, error) {
	if prefix = strings.TrimFunc(prefix, iorw.IsSpace); prefix == "" {
		return false, nil
	}

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
				if path = strings.TrimFunc(dpath, iorw.IsSpace); !strings.Contains(path, prefix) {
					return false, nil
				}
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
	if path == "" {
		if path = strings.TrimFunc(r.Path(), iorw.IsSpace); path == "" {
			return false, nil
		}
	}
	if strings.Contains(path, prefix) {
		ctx := r.Context()
		pathext := filepath.Ext(path)
		if pathext == "" {
			pathext = ".json"
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
			if path = path[strings.Index(path, prefix)+len(prefix):]; path != "" && validdbrequestexts[pathext] {
				pthsepi := strings.Index(path, "/")

				if pthsepi == -1 {
					pthsepi = len(path)
				}
				rmndrpath := path[pthsepi:]
				path = path[:pthsepi]
				if w != nil {
					if pathext == ".json" {
						w.Header().Set("Content-Type", "application/json; charset=utf-8")
					}
					if pathext == ".js" {
						w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
					}
				}

				if cmdv, cmdvok := cmnds.Load(path); cmdvok {
					cmdhndlr := cmdv.(CommandFunc)
					if err := cmdhndlr.ExecuteCmd(rmndrpath, pathext, dbhndl, w, r, fs); err != nil {
						enc := json.NewEncoder(w)
						enc.Encode(map[string]interface{}{"err": err.Error()})
					}
					return true, nil
				}
				if rmndrpath != "" && dbhndl.Exists(path) {
					if pathext != "" && strings.HasSuffix(rmndrpath, pathext) {
						rmndrpath = rmndrpath[:len(rmndrpath)-len(pathext)]
					}
					rmngpthi := strings.Index(rmndrpath, "/")
					if rmngpthi == -1 {
						rmngpthi = len(rmndrpath)
					}
					if rmngpthi == 0 {
						rmndrpath = rmndrpath[1:]
					}
					subrmng := ""
					subrmngi := strings.Index(rmndrpath, "/")
					if subrmngi > -1 {
						subrmng = rmndrpath[subrmngi:]
						rmndrpath = rmndrpath[:subrmngi]
					}
					if rmndrpath != "" {
						if aliascmdv, aliascmdvok := aliascmnds.Load(rmndrpath); aliascmdvok {
							aliascmd := aliascmdv.(AliasCommandFunc)
							if err := aliascmd.ExecuteCmd(path, subrmng, pathext, dbhndl, w, r, fs); err != nil {

							}
						}
						return true, nil
					}
				}
			}
		}
	}
	return false, nil
}

type HandlerAliasCommand interface {
	ExecuteCmd(string, string, string, *database.DBMSHandler, serveio.Writer, serveio.Reader, *fsutils.FSUtils) error
}

type AliasCommandFunc func(alias, path string, ext string, dbhnl *database.DBMSHandler, w serveio.Writer, r serveio.Reader, fs *fsutils.FSUtils) (err error)

func (aliascmdfunc AliasCommandFunc) ExecuteCmd(alias, cmdpath, cmdpathext string, dbhndl *database.DBMSHandler, w serveio.Writer, r serveio.Reader, fs *fsutils.FSUtils) (err error) {
	if err = aliascmdfunc(alias, cmdpath, cmdpathext, dbhndl, w, r, fs); err != nil {

	}
	return
}

type HandlerCommand interface {
	ExecuteCmd(string, string, *database.DBMSHandler, serveio.Writer, serveio.Reader, *fsutils.FSUtils) error
}

type CommandFunc func(path string, ext string, dbhnl *database.DBMSHandler, w serveio.Writer, r serveio.Reader, fs *fsutils.FSUtils) (err error)

func (cmdfunc CommandFunc) ExecuteCmd(cmdpath, cmdpathext string, dbhndl *database.DBMSHandler, w serveio.Writer, r serveio.Reader, fs *fsutils.FSUtils) (err error) {
	if err = cmdfunc(cmdpath, cmdpathext, dbhndl, w, r, fs); err != nil {

	}
	return
}

var validdbrequestexts = map[string]bool{".js": true, ".json": true}

var cmdcommands CommandFunc = func(path, ext string, dbhnl *database.DBMSHandler, w serveio.Writer, r serveio.Reader, fs *fsutils.FSUtils) (err error) {
	return
}

var aliascmdexec AliasCommandFunc = func(alias, path, ext string, dbhnl *database.DBMSHandler, w serveio.Writer, r serveio.Reader, fs *fsutils.FSUtils) (err error) {

	return
}

func HandleCommand(a ...interface{}) {

	if al := len(a); al > 1 {
		for al > 0 {
			if cmds, _ := a[0].(string); cmds != "" {
				if cmdfunc, _ := a[1].(CommandFunc); cmdfunc != nil {
					cmnds.Store(cmds, cmdfunc)
					al -= 2
					a = a[2:]
					continue
				}
				if aliascmdfunc, _ := a[1].(AliasCommandFunc); aliascmdfunc != nil {
					aliascmnds.Store(cmds, aliascmdfunc)
					al -= 2
					a = a[2:]
					continue
				}
			}
			al -= 2
			a = a[2:]
		}
	}
}

var cmnds = &sync.Map{}
var aliascmnds = &sync.Map{}
