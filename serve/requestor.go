package serve

import (
	"bufio"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/evocert/lnksnk/concurrent"
	"github.com/evocert/lnksnk/database"
	"github.com/evocert/lnksnk/email/emailing"
	"github.com/evocert/lnksnk/fsutils"
	"github.com/evocert/lnksnk/iorw"
	"github.com/evocert/lnksnk/iorw/active"
	"github.com/evocert/lnksnk/iorw/parsing"
	"github.com/evocert/lnksnk/mimes"
	"github.com/evocert/lnksnk/parameters"
	"github.com/evocert/lnksnk/resources"
	"github.com/evocert/lnksnk/scheduling"
	"github.com/evocert/lnksnk/stdio/command"
	"github.com/evocert/lnksnk/ws"
)

var lastserial int64 = time.Now().UnixNano()

func nextserial() (nxsrl int64) {
	for {
		if nxsrl = time.Now().UnixNano(); atomic.CompareAndSwapInt64(&lastserial, atomic.LoadInt64(&lastserial), nxsrl) {
			break
		}
		time.Sleep(1 * time.Nanosecond)
	}
	return
}

func ProcessRequesterConn(conn net.Conn, activemap map[string]interface{}) {
	if conn != nil {
		if rqst, rqsterr := http.ReadRequest(bufio.NewReaderSize(conn, 65536)); rqsterr != nil {
			conn.Close()
			return
		} else if rqst != nil {
			ProcessRequest("", rqst, NewResponseWriter(rqst, conn), activemap)
		}
	}

}

func ServeHTTPRequest(w http.ResponseWriter, r *http.Request) {
	ProcessRequest("", r, w, nil)
}

func ProcessRequestPath(path string, activemap map[string]interface{}) {
	internalServeRequest(path, nil, nil, nil, nil, fs, activemap)
}

func ProcessRequest(path string, httprqst *http.Request, httprspns http.ResponseWriter, activemap map[string]interface{}) {
	if httprqst != nil && httprspns != nil {
		if ws, wserr := ws.NewServerReaderWriter(httprspns, httprqst); wserr == nil && ws != nil {

			return
		}
		internalServeRequest("", newReader(httprqst), newWriter(httprspns), httprspns, httprqst, fs, activemap)
	}
}

var fs = resources.GLOBALRSNG().FS()

func ParseEval(evalcode func(a ...interface{}) (val interface{}, err error), path, pathext string, pathmodified time.Time, Out io.Writer, In io.Reader, fs *fsutils.FSUtils, invertactive bool, fi fsutils.FileInfo, fnmodified func(modified time.Time), fnactiveraw func(rsraw bool, rsactive bool)) (err error) {
	if fi != nil && path != "" && path[0:1] != "/" {
		path += fi.Path()
	}
	err = parsing.Parse(true, pathmodified, path, pathext, Out, func() (f io.Reader, ferr error) {
		if fi != nil {
			f, ferr = fi.Open(fnactiveraw, fnmodified)
			return
		}
		return In, nil
	}, fs, invertactive, evalcode)
	return
}

func internalServeRequest(path string, In *reader, Out *writer, httpw http.ResponseWriter, httpr *http.Request, fs *fsutils.FSUtils, activemap map[string]interface{}, a ...interface{}) (err error) {
	params := parameters.NewParameters()
	defer params.CleanupParameters()
	var ctx context.Context = nil
	if httpr != nil {
		ctx = httpr.Context()
		parameters.LoadParametersFromHTTPRequest(params, httpr)
		if path == "" {
			path = httpr.URL.Path
		}
	} else {
		if path != "" {
			path = strings.Replace(path, "\\", "/", -1)
		}
		parameters.LoadParametersFromRawURL(params, path)
	}

	defer In.Close()
	defer Out.Close()

	var prsevalbuf *iorw.Buffer = nil
	defer prsevalbuf.Close()
	var terminal *terminals = nil
	defer terminal.Close()

	var fi fsutils.FileInfo = nil

	var dbclsrs = newdbclosers()
	defer dbclsrs.Close()
	var vm *active.VM = nil
	var dbhnlr *database.DBMSHandler = DBMS.DBMSHandler(ctx, active.RuntimeFunc(func(functocall interface{}, args ...interface{}) interface{} {
		return vm.InvokeFunction(functocall, args...)
	}), params, CHACHING, fs, func(ina ...interface{}) (a []interface{}) {
		if vm != nil && len(ina) == 1 {
			if fia, _ := ina[0].(fsutils.FileInfo); fia != nil {
				stmntoutbuf := iorw.NewBuffer()
				defer stmntoutbuf.Close()
				vmw := vm.W
				vm.W = stmntoutbuf
				if evalerr := ParseEval(vm.Eval, fia.Path(), fia.PathExt(), fia.ModTime(), stmntoutbuf, nil, fs, false, fia, nil, nil); evalerr == nil {
					a = append(a, stmntoutbuf.Clone(true).Reader(true))
				}
				vm.W = vmw
			}
		} else {
			a = append(a, ina...)
		}
		return
	})
	defer dbhnlr.Dispose()
	invokevm := func() *active.VM {
		if vm != nil {
			return vm
		}
		vm = func() (nvm *active.VM) {
			select {
			case nvm = <-chnvms:
				if nvm == nil {
					nvm = active.NewVM()
				}
			default:
				nvm = active.NewVM()
			}
			nvm.ErrPrint = func(a ...interface{}) (vmerr error) {
				if Out != nil {
					Out.Print("<pre>ERR:\r\n")
					Out.Print(a...)
					Out.Print("\r\n</pre>")
				}
				return
			}
			nvm.Set("fs", fs)
			nvm.Set("listen", LISTEN)
			nvm.Set("lstn", LISTEN)
			if terminal == nil {
				terminal = newTerminal()
			}
			nvm.Set("terminal", terminal)
			nvm.Set("trm", terminal)
			nvm.Set("command", terminal)
			nvm.Set("cmd", terminal)
			nvm.Set("faf", func(rqstpath string) {
				go ProcessRequestPath(rqstpath, nil)
			})
			var fparseEval = func(prsout io.Writer, evalrt interface{}, a ...interface{}) (prsevalerr error) {
				var invert bool = false
				var fitouse = fi
				if len(a) > 0 {
					if inv, invok := a[0].(bool); invok {
						invert = inv
						a = a[1:]
					}
				}
				if prsout == nil {
					prsout = Out
				} else if prsout != Out {
					if nvm.W == Out {
						nvm.SetPrinter(prsout)
						defer func() {
							nvm.SetPrinter(Out)
						}()
					}
				}
				var evalroot, _ = evalrt.(string)
				if evalroot != "" {
					if fios := fs.LS(evalroot); len(fios) == 1 {
						fitouse = fios[0]
						evalroot = fitouse.PathRoot()
					}
				}
				var prin, _ = evalrt.(io.Reader)
				var suggestedroot = "/"
				if fitouse != nil {
					suggestedroot = fitouse.PathRoot()
					if fitouse.IsDir() {
						if strings.HasSuffix(evalroot, "/") {
							for _, evlpth := range []string{"index.html", "index.js"} {
								if fis := fs.LS(evalroot + evlpth); len(fis) == 1 {
									prsevalerr = ParseEval(nvm.Eval, fis[0].Path(), ".js", fis[0].ModTime(), prsout, nil, fs, invert, fis[0], nil, nil)
									return
								}
							}
						}
						return
					}
					prsevalerr = ParseEval(nvm.Eval, fitouse.Path(), fitouse.PathExt(), fitouse.ModTime(), prsout, nil, fs, invert, fitouse, nil, nil)
					return
				}

				if fis := fs.LS(evalroot + ".js"); len(fis) == 1 {
					prsevalerr = ParseEval(nvm.Eval, fis[0].Path(), ".js", fis[0].ModTime(), prsout, nil, fs, invert, fis[0], nil, nil)
				} else if len(fis) == 0 {
					if evalroot != "" || (evalroot == "" && evalrt != nil) {
						a = append([]interface{}{evalrt}, a...)
					}
					if prin == nil && len(a) > 0 {
						func() {
							defer prsevalbuf.Clear()
							if prsevalbuf == nil {
								prsevalbuf = iorw.NewBuffer()
								prsevalbuf.Print(a...)
							} else {
								prsevalbuf.Clear()
								prsevalbuf.Print(a...)
							}
							if prsevalbuf.Size() > 0 {
								prsevalerr = ParseEval(nvm.Eval, ":no-cache/"+suggestedroot, ".js", time.Now(), prsout, prsevalbuf.Clone(true).Reader(true), fs, invert, nil, nil, nil)
							}
						}()
					} else {
						if fitouse.PathRoot() != suggestedroot && fitouse.PathExt() != ".js" {
							prsevalerr = ParseEval(nvm.Eval, ":no-cache/"+suggestedroot, ".js", time.Now(), prsout, prin, fs, invert, nil, nil, nil)
						}
					}
				}
				return prsevalerr
			}

			nvm.Set("parseEval", fparseEval)

			nvm.Set("scheduling", SCHEDULING)
			nvm.Set("schdlng", SCHEDULING)
			nvm.Set("caching", CHACHING)
			nvm.Set("cchng", CHACHING)
			nvm.Set("db", dbhnlr)
			nvm.Set("dbqry", func(alias string, a ...interface{}) (reader *database.Reader) {
				a = append([]interface{}{nvm}, a...)

				if params != nil {
					a = append(a, params)
				}
				if fs != nil {
					a = append(a, fs)
				}
				if reader = DBMS.Query(alias, a...); reader != nil {
					reader.EventClose = func(r *database.Reader) {
						dbclsrs.clsrs.Delete(r)
					}
					dbclsrs.clsrs.Store(reader, reader)
				}
				return
			})

			nvm.Set("dbexec", func(alias string, a ...interface{}) (exectr *database.Executor) {
				a = append([]interface{}{nvm}, a...)

				if params != nil {
					a = append(a, params)
				}
				if fs != nil {
					a = append(a, fs)
				}
				if exectr = DBMS.Execute(alias, a...); exectr != nil {
					exectr.EventClose = func(ex *database.Executor) {
						dbclsrs.clsrs.Delete(ex)
					}
					dbclsrs.clsrs.Store(exectr, exectr)
				}
				return
			})

			nvm.Set("dbreg", func(alias string, driver string, datasource string, a ...interface{}) bool {
				return DBMS.Register(alias, driver, datasource, a...)
			})

			nvm.Set("dbunreg", func(alias string, a ...interface{}) bool {
				return DBMS.Unregister(alias, a...)
			})

			nvm.Set("dbprep", func(alias string, a ...interface{}) (exectr *database.Executor) {
				a = append([]interface{}{nvm}, a...)

				if params != nil {
					a = append(a, params)
				}
				if fs != nil {
					a = append(a, fs)
				}
				if exectr = DBMS.Prepair(alias, a...); exectr != nil {
					exectr.EventClose = func(ex *database.Executor) {
						dbclsrs.clsrs.Delete(ex)
					}
					dbclsrs.clsrs.Store(exectr, exectr)
				}
				return
			})

			nvm.Set("email", EMAILING.ActiveEmailManager(nvm, func() parameters.ParametersAPI {
				return params
			}, fs))
			for actvkey, actvval := range activemap {
				nvm.Set(actvkey, actvval)
			}

			var vmparam = map[string]interface{}{
				"set":       params.SetParameter,
				"get":       params.Parameter,
				"exist":     params.ContainsParameter,
				"fileExist": params.ContainsFileParameter,
				"setFile":   params.SetFileParameter,
				"getFile":   params.FileParameter,
				"keys":      params.StandardKeys,
				"fileKeys":  params.FileKeys,
				"fileName":  params.FileName,
			}

			nvm.Set("param", vmparam)
			nvm.Set("_in", In)
			nvm.Set("_out", Out)
			nvm.R = In
			nvm.W = Out

			return nvm
		}()
		return vm
	}
	var rangeOffset = In.RangeOffset()
	var rangeType = In.RangeType()
	defer func() {
		if vm != nil {
			chnvms <- vm
		}
	}()

	var pathext = filepath.Ext(path)
	var pathmodified time.Time = time.Now()
	var fnmodified = func(modified time.Time) {
		pathmodified = modified
	}

	var israw = false
	var convertactive = false

	var mimetipe, istexttype, ismedia = mimes.FindMimeType(pathext, "text/plain")
	var isactive = istexttype

	var fnactiveraw = func(rsraw bool, rsactive bool) {
		if israw = rsraw; !israw {
			if isactive {
				if !convertactive {
					convertactive = rsactive
				}
			}
		} else {
			isactive = false
		}
	}

	var invertactive = false
	if strings.Contains(path, "/active:") {
		for strings.Contains(path, "/active:") {
			prepath := path[:strings.Index(path, "/active:")+1]
			path = prepath + path[strings.Index(path, "/active:")+len("/active:"):]
		}
		invertactive = true
	}
	if pathext != "" {
		if fis := fs.LS(path); len(fis) == 1 {
			mimetipe, istexttype, ismedia = mimes.FindMimeType(pathext, "text/plain")
			fi = fis[0]
			fnactiveraw(fi.IsRaw(), fi.IsActive())
			fnmodified(fi.ModTime())
		}
	}
	if fi == nil && pathext == "" && strings.HasSuffix(path, "/") {
		for _, psblexts := range []string{".html", ".js"} {
			isactive = true
			if fis := fs.LS(path + "index" + psblexts); len(fis) == 1 {
				fi = fis[0]
				path = fi.Path()
				mimetipe, istexttype, ismedia = mimes.FindMimeType(psblexts, "text/plain")
				pathext = fi.PathExt()

				fnactiveraw(fi.IsRaw(), fi.IsActive())
				fnmodified(fi.ModTime())
				break
			}
		}
	}

	if istexttype || strings.Contains(mimetipe, "text/plain") {
		mimetipe += "; charset=utf-8"
	}
	if fi != nil {
		if pathext != "" {

			if invertactive {
				if !israw && !ismedia {
					invertactive = true
					if !isactive {
						isactive = true
					}
				}
			}
			if Out != nil {
				Out.Header().Set("Content-Type", mimetipe)
			}
			if !isactive && convertactive {
				isactive = true
			}

			if !israw && isactive {
				err = ParseEval(invokevm().Eval, path, pathext, pathmodified, Out, nil, fs, invertactive, fi, fnmodified, fnactiveraw)
			} else if israw || ismedia {
				if ismedia {
					if f, ferr := fi.Open(); ferr == nil {
						defer f.Close()
						if rssize := fi.Size(); rssize > 0 {
							var eofrs *iorw.EOFCloseSeekReader = nil
							if eofrs, _ = f.(*iorw.EOFCloseSeekReader); eofrs == nil {
								eofrs = iorw.NewEOFCloseSeekReader(f, false)
							}
							if eofrs != nil {
								if rangeOffset == -1 {
									rangeOffset = 0
								} else {
									eofrs.Seek(rangeOffset, 0)
								}
								if rssize > 0 {
									if rangeType == "bytes" && rangeOffset > -1 {
										maxoffset := int64(0)
										maxlen := int64(0)
										if maxoffset = rangeOffset + (rssize - rangeOffset); maxoffset > 0 {
											maxlen = maxoffset - rangeOffset
											maxoffset--
										}

										if maxoffset < rangeOffset {
											maxoffset = rangeOffset
											maxlen = 0
										}

										if maxlen > 1024*1024 {
											maxlen = 1024 * 1024
											maxoffset = rangeOffset + (maxlen - 1)
										}
										contentrange := fmt.Sprintf("%s %d-%d/%d", In.RangeType(), rangeOffset, maxoffset, rssize)
										if Out != nil {
											Out.Header().Set("Accept-Ranges", "bytes")
											Out.Header().Set("Content-Range", contentrange)
											Out.Header().Set("Content-Length", fmt.Sprintf("%d", maxlen))
										}
										eofrs.SetMaxRead(maxlen)
										if httpw != nil {
											httpw.WriteHeader(206)
										}
									} else {
										if Out != nil {
											Out.Header().Set("Accept-Ranges", "bytes")
											Out.Header().Set("Content-Length", fmt.Sprintf("%d", rssize))
										}
										eofrs.SetMaxRead(rssize)
									}
								}
								Out.Print(eofrs)
							}
						}
					}
				} else {
					if Out != nil {
						if fi != nil {
							Out.Header().Set("Content-Length", fmt.Sprintf("%d", fi.Size()))
						}
						if f, ferr := fi.Open(); ferr == nil {
							if f != nil {
								defer f.Close()
								Out.Print(f)
							}
						}
					}
				}
			} else {
				if Out != nil {
					if f, ferr := fi.Open(); ferr == nil {
						if f != nil {
							defer f.Close()
							Out.Print(f)
						}
					}
				}
			}
		}
	} else {
		if Out != nil {
			Out.Header().Set("Content-Type", mimetipe)
		}
	}
	return
}

type dbclosers struct {
	clsrs *sync.Map
}

func newdbclosers() *dbclosers {
	return &dbclosers{clsrs: &sync.Map{}}
}

func (dbcls *dbclosers) Close() {
	if dbcls != nil {
		if clsrs := dbcls.clsrs; clsrs != nil {
			clsrs.Range(func(key, value any) bool {
				if exctr, _ := value.(*database.Executor); exctr != nil {
					exctr.EventClose = nil
					clsrs.Delete(key)
					exctr.Close()
				} else if dbrdr, _ := value.(*database.Reader); dbrdr != nil {
					dbrdr.EventClose = nil
					clsrs.Delete(key)
					dbrdr.Close()
				}
				return true
			})
			dbcls.clsrs = nil
		}
	}
}

type terminals struct {
	cmdprscs    *sync.Map
	cmdprscrefs *sync.Map
}

func newTerminal() (terms *terminals) {
	terms = &terminals{cmdprscs: &sync.Map{}, cmdprscrefs: &sync.Map{}}
	return
}

func (terms *terminals) Command(alias string, execargs ...string) (cmd *command.Command, err error) { // (cmd *osprc.Command, err error) {
	if terms != nil && alias != "" {
		execpath := ""
		if len(execargs) > 0 {
			execpath = execargs[0]
			execargs = execargs[1:]
		}
		if execpath != "" {
			if cmd, err = command.NewCommand(execpath, os.Environ(), execargs...); err == nil && cmd != nil {
				if terms.cmdprscs == nil {
					terms.cmdprscs = &sync.Map{}
				}
				if terms.cmdprscrefs != nil {
					if cmpiv, cmpivok := terms.cmdprscrefs.Load(alias); cmpivok {
						cmpi, _ := cmpiv.(int)
						if cmv, cmvok := terms.cmdprscs.Load(cmpi); cmvok {
							if cmpref, _ := cmv.(*command.Command); cmpref != nil {
								cmpref.Close()
							}
						}
					}
				} else {
					terms.cmdprscrefs = &sync.Map{}
				}
				terms.cmdprscs.Store(cmd.Pid, cmd)
				cmd.OnClose = func(prcid int) {
					if cmpiv, cmpivok := terms.cmdprscrefs.Load(alias); cmpivok {
						if cmpi, _ := cmpiv.(int); cmpi == prcid {
							terms.cmdprscrefs.Delete(alias)
						}
						if _, cmpvok := terms.cmdprscs.Load(prcid); cmpvok {
							terms.cmdprscs.Delete(prcid)
						}
					}
				}
			} else {
				if terms.cmdprscrefs != nil {
					if cmpiv, cmpivok := terms.cmdprscrefs.Load(alias); cmpivok {
						cmpi, _ := cmpiv.(int)
						if cmpv, cmpvok := terms.cmdprscs.Load(cmpi); cmpvok {
							cmd, _ = cmpv.(*command.Command)
						}
					}
				}
			}
		}
	}
	return
}

func (terms *terminals) Close() {
	if terms != nil {
		if cmdprscs := terms.cmdprscs; cmdprscs != nil {
			cmdprscs.Range(func(key, value any) bool {
				if cmd, _ := value.(*command.Command); cmd != nil {
					cmd.Close()
				}
				return true
			})
			terms.cmdprscs = nil
			if terms.cmdprscrefs != nil {
				terms.cmdprscrefs = nil
			}
		}
		if terms.cmdprscrefs != nil {
			terms.cmdprscrefs = nil
		}
	}
}

//var requests = concurrent.NewMap()

var chnvms = make(chan *active.VM)
var chndbmshnds = make(chan *database.DBMSHandler)
var chnemailing = make(chan *emailing.ActiveEmailManager)
var chnterms = make(chan *terminals)

var DBMS = database.GLOBALDBMS()
var SCHEDULING = scheduling.GLOBALSCHEDULING()
var EMAILING = emailing.GLOBALEMAILMNGR()

type ListenApi interface {
	Serve(network string, addr string, tlsconf ...*tls.Config)
	ServeTLS(network string, addr string, orgname string, tlsconf ...*tls.Config)
	Shutdown(...interface{})
}

var LISTEN ListenApi = nil

var CHACHING = concurrent.NewMap()

func init() {
	go func() {
		for vmref := range chnvms {
			go func(vm *active.VM) { vm.Close() }(vmref)
		}
	}()
	go func() {
		for dbmsref := range chndbmshnds {
			go func(dbms *database.DBMSHandler) { dbms.Dispose() }(dbmsref)
		}
	}()
	go func() {
		for emailref := range chnemailing {
			go func(email *emailing.ActiveEmailManager) { email.Close() }(emailref)
		}
	}()
	go func() {
		for termref := range chnterms {
			go func(term *terminals) { term.Close() }(termref)
		}
	}()
}
