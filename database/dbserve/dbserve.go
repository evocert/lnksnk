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
		if strings.Contains(path, "/db:") {
			pathext := filepath.Ext(path)
			if pathext == "" {
				pathext = ".json"
			}
			if path = path[strings.Index(path, "/db:")+len("/db:"):]; path != "" && validdbrequestexts[pathext] {
				pthsepi := strings.Index(path, "/")

				if pthsepi == -1 {
					pthsepi = len(path)
				}
				rmndrpath := path[pthsepi:]
				if w != nil {
					if pathext == ".json" {
						w.Header().Set("Content-Type", "application/json; charset=utf-8")
					}
					if pathext == ".js" {
						w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
					}
				}

				if cmdhndlr := validdbcommands[path]; cmdhndlr != nil {
					if err := cmdhndlr.ExecuteCmd(rmndrpath, pathext, dbhndl, w, r, fs); err != nil {
						enc := json.NewEncoder(w)
						enc.Encode(map[string]interface{}{"err": err.Error()})
					}
					return
				}
				if rmndrpath != "" && dbhndl.Exists(path) {
					if pathext != "" && strings.HasSuffix(rmndrpath, pathext) {
						rmndrpath = rmndrpath[:len(rmndrpath)-len(pathext)]
					}
					rmngpthi := strings.Index(rmndrpath, "/")
					if rmngpthi == -1 {
						rmngpthi = len(rmndrpath)
					}
					if rmndrpath = rmndrpath[:rmngpthi]; rmndrpath != "" {
						if aliascmd := validdbaliascommands[rmndrpath]; aliascmd != nil {
							if err := aliascmd.ExecuteCmd(path, rmndrpath, pathext, dbhndl, w, r, fs); err != nil {

							}
						}
					}

				}
			}
		}
	}
}

type HandlerAliasCommand interface {
	ExecuteCmd(string, string, string, *database.DBMSHandler, serveio.Writer, serveio.Reader, *fsutils.FSUtils) error
}

type AliasCommandFunc func(alias, path string, ext string, dbhnl *database.DBMSHandler, w serveio.Writer, r serveio.Reader, fs *fsutils.FSUtils) (err error)

func (aliascmdfunc AliasCommandFunc) ExecuteCmd(alias, cmdpath, cmdpathext string, dbhndl *database.DBMSHandler, w serveio.Writer, r serveio.Reader, fs *fsutils.FSUtils) error {
	return aliascmdfunc(alias, cmdpath, cmdpathext, dbhndl, w, r, fs)
}

type HandlerCommand interface {
	ExecuteCmd(string, string, *database.DBMSHandler, serveio.Writer, serveio.Reader, *fsutils.FSUtils) error
}

type CommandFunc func(path string, ext string, dbhnl *database.DBMSHandler, w serveio.Writer, r serveio.Reader, fs *fsutils.FSUtils) (err error)

func (cmdfunc CommandFunc) ExecuteCmd(cmdpath, cmdpathext string, dbhndl *database.DBMSHandler, w serveio.Writer, r serveio.Reader, fs *fsutils.FSUtils) error {
	return cmdfunc(cmdpath, cmdpathext, dbhndl, w, r, fs)
}

var validdbrequestexts = map[string]bool{".js": true, ".json": true}
var validdbcommands = map[string]HandlerCommand{"connections": cmdconnections,
	"drivers":    cmddrivers,
	"register":   cmdregister,
	"unregister": cmdregister,
	"connection": cmdconnection,
	"driver":     cmddriver,
	"commands":   cmdcommands}

var validdbaliascommands = map[string]HandlerAliasCommand{
	"query":  aliascmdquery,
	"exec":   aliascmdexec,
	"status": aliascmdstatus}

var cmdcommands CommandFunc = func(path, ext string, dbhnl *database.DBMSHandler, w serveio.Writer, r serveio.Reader, fs *fsutils.FSUtils) (err error) {
	return
}

var cmdconnections CommandFunc = func(path, ext string, dbhnl *database.DBMSHandler, w serveio.Writer, r serveio.Reader, fs *fsutils.FSUtils) (err error) {
	encd := json.NewEncoder(w)
	err = encd.Encode(dbhnl.Connections())
	return
}

var cmdconnection CommandFunc = func(path, ext string, dbhnl *database.DBMSHandler, w serveio.Writer, r serveio.Reader, fs *fsutils.FSUtils) (err error) {
	return
}

var cmddrivers CommandFunc = func(path, ext string, dbhnl *database.DBMSHandler, w serveio.Writer, r serveio.Reader, fs *fsutils.FSUtils) (err error) {
	encd := json.NewEncoder(w)
	err = encd.Encode(dbhnl.Drivers())
	return
}

var cmddriver CommandFunc = func(path, ext string, dbhnl *database.DBMSHandler, w serveio.Writer, r serveio.Reader, fs *fsutils.FSUtils) (err error) {

	return
}

var cmdregister CommandFunc = func(path, ext string, dbhnl *database.DBMSHandler, w serveio.Writer, r serveio.Reader, fs *fsutils.FSUtils) (err error) {

	return
}

var cmdunregister CommandFunc = func(path, ext string, dbhnl *database.DBMSHandler, w serveio.Writer, r serveio.Reader, fs *fsutils.FSUtils) (err error) {
	return
}

var aliascmdquery AliasCommandFunc = func(alias, path, ext string, dbhnl *database.DBMSHandler, w serveio.Writer, r serveio.Reader, fs *fsutils.FSUtils) (err error) {

	return
}

var aliascmdexec AliasCommandFunc = func(alias, path, ext string, dbhnl *database.DBMSHandler, w serveio.Writer, r serveio.Reader, fs *fsutils.FSUtils) (err error) {

	return
}

var aliascmdstatus AliasCommandFunc = func(alias, path, ext string, dbhnl *database.DBMSHandler, w serveio.Writer, r serveio.Reader, fs *fsutils.FSUtils) (err error) {

	return
}
