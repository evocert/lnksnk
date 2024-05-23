package database

import (
	"context"
	"database/sql"
	"io"

	//"github.com/evocert/lnksnk/caching"
	"strings"
	"sync"

	"github.com/evocert/lnksnk/concurrent"
	"github.com/evocert/lnksnk/fsutils"
	"github.com/evocert/lnksnk/iorw"
	"github.com/evocert/lnksnk/parameters"
)

type Statement struct {
	ctx           context.Context
	cn            *Connection
	isRemote      bool
	prepstmnt     *sql.Stmt
	prms          *parameters.Parameters
	rdr           *Reader
	args          *sync.Map
	stmntlck      *sync.RWMutex
	stmnt         string
	argnames      []string
	argtypes      []int
	parseSqlParam func(totalArgs int) (s string)
}

func NewStatement(cn *Connection) (stmnt *Statement) {
	if cn != nil {
		stmnt = &Statement{cn: cn, isRemote: cn.isRemote(), stmntlck: &sync.RWMutex{}, args: &sync.Map{}}
	}
	return
}

func parseParam(parseSqlParam func(totalArgs int) (s string), totalArgs int) (s string) {
	if parseSqlParam != nil {
		s = parseSqlParam(totalArgs)
	} else {
		s = "?"
	}

	return
}

type StatementHandler interface {
	Prepair(...interface{}) []interface{}
}

type StatementHandlerFunc func(a ...interface{}) []interface{}

func (stmnthndlfnc StatementHandlerFunc) Prepair(a ...interface{}) []interface{} {
	return stmnthndlfnc(a...)
}

func (stmnt *Statement) Prepair(prms *parameters.Parameters, rdr *Reader, args map[string]interface{}, a ...interface{}) (preperr error) {
	if stmnt != nil {
		defer func() {

			if preperr != nil && stmnt != nil {
				stmnt.Close()
			}
		}()
		var rnrr io.RuneReader = nil
		var qrybuf = iorw.NewBuffer()
		var validNames []string
		var validNameType []int
		var fs *fsutils.FSUtils = nil
		var al = len(a)
		var ai = 0
		stmntref := &stmnt.stmnt
		var cchng *concurrent.Map = nil
		var ctx context.Context = nil
		var stmnthndlr StatementHandler = nil
		for ai < al {
			if d := a[ai]; d != nil {
				if stmnthndld, _ := d.(StatementHandler); stmnthndld != nil {
					if stmnthndlr == nil {
						stmnthndlr = stmnthndld
					}
					a = append(a[:ai], a[ai+1:]...)
					al--
					continue
				} else if fsd, _ := d.(*fsutils.FSUtils); fsd != nil {
					if fs == nil {
						fs = fsd
					}
					a = append(a[:ai], a[ai+1:]...)
					al--
					continue
				} else if ccnngd, _ := d.(*concurrent.Map); ccnngd != nil {
					if cchng == nil {
						cchng = ccnngd
					}
					a = append(a[:ai], a[ai+1:]...)
					al--
					continue
				} else if ctxd, _ := d.(context.Context); ctxd != nil {
					if ctx == nil {
						ctx = ctxd
					}
					a = append(a[:ai], a[ai+1:]...)
					al--
					continue
				}
			}
			ai++
		}
		if vqry, vqryfnd := cchng.Find(a...); vqryfnd && vqry != nil {
			qrybuf.Print(vqry)
		} else {
			qrybuf.Print(a...)
		}

		if fs != nil {
			if fi := func() fsutils.FileInfo {
				if tstsql := qrybuf.String() + func() string {
					if !qrybuf.HasSuffix(".sql") {
						return ".sql"
					}
					return ""
				}(); tstsql != "" {
					if fio := fs.LS(tstsql); len(fio) == 1 {
						return fio[0]
					}
					if fio := fs.LS(tstsql[:len(tstsql)-len(".sql")] + "." + stmnt.cn.driverName + ".sql"); len(fio) == 1 {
						return fio[0]
					}
				}
				return nil
			}(); fi != nil && stmnthndlr != nil {
				qrybuf.Clear()
				qrybuf.Print(stmnthndlr.Prepair(fi))
			}
		}

		/*if stmnthndlr != nil {
			qrybuf.Print(stmnthndlr.Prepair(qrybuf.Clone(true).Reader(true)))
		}*/

		if qrybuf.HasPrefix("#") && qrybuf.HasSuffix("#") {
			if substrqry, _ := qrybuf.SubString(1, qrybuf.Size()-1); substrqry != "" {
				subqryarr := strings.Split(substrqry, "=>")
				subqry := make([]interface{}, len(subqryarr))
				for subn, sub := range subqryarr {
					subqry[subn] = strings.TrimSpace(sub)
				}
				if valfnd, valfndok := cchng.Find(subqry...); valfndok && valfnd != nil {
					qrybuf.Clear()
					qrybuf.Print(valfnd)
				} else {
					qrybuf.Clear()
					qrybuf.Print(substrqry)
				}
			}
		}
		defer qrybuf.Close()
		//if stmnthndlr != nil {
		//	qrybuf.Print(stmnthndlr.Prepair(qrybuf.Clone(true).Reader(true)))
		//}
		rnrr = qrybuf.Reader()
		var foundTxt = false

		var prvr = rune(0)
		var prmslbl = [][]rune{[]rune("@"), []rune("@")}
		var prmslbli = []int{0, 0}

		rnsbuf := []rune{}

		var appr = func(r rune) {
			rnsbuf = append(rnsbuf, r)
		}

		var apprs = func(p []rune) {
			if pl := len(p); pl > 0 {
				rnsbuf = append(rnsbuf, p...)
			}
		}

		var psblprmnme = make([]rune, 8192)
		var psblprmnmei = 0

		var possibleArgName map[string]int = map[string]int{}
		if len(args) > 0 {
			for dfltk := range args {
				possibleArgName[dfltk] = 0
			}
		}

		if prms != nil {
			for _, dfltk := range prms.StandardKeys() {
				possibleArgName[dfltk] = 1
			}
		}

		if rdr != nil {
			for _, dfltk := range rdr.Columns() {
				possibleArgName[dfltk] = 2
			}
		}

		var parseRune = func(r rune) {
			if foundTxt {
				appr(r)
				if r == '\'' {
					foundTxt = false
					prvr = rune(0)
				} else {
					prvr = r
				}
			} else {
				if prmslbli[1] == 0 && prmslbli[0] < len(prmslbl[0]) {
					if prmslbli[0] > 0 && prmslbl[0][prmslbli[0]-1] == prvr && prmslbl[0][prmslbli[0]] != r {
						if prmsl := prmslbli[0]; prmsl > 0 {
							prmslbli[0] = 0
							apprs(prmslbl[0][:prmsl])
						}
					}
					if prmslbl[0][prmslbli[0]] == r {
						prmslbli[0]++
						if prmslbli[0] == len(prmslbl[0]) {
							prvr = rune(0)
						} else {
							prvr = r
						}
					} else {
						if prmsl := prmslbli[0]; prmsl > 0 {
							prmslbli[0] = 0
							apprs(prmslbl[0][:prmsl])
						}
						appr(r)
						if r == '\'' {
							foundTxt = true
							prvr = rune(0)
						} else {
							prvr = r
						}
					}
				} else if prmslbli[0] == len(prmslbl[0]) && prmslbli[1] < len(prmslbl[1]) {
					if prmslbl[1][prmslbli[1]] == r {
						prmslbli[1]++
						if prmslbli[1] == len(prmslbl[1]) {
							if psblprmnmei > 0 {
								if psbprmnme := string(psblprmnme[:psblprmnmei]); psbprmnme != "" {
									fndprm := true
									for mpvk, mpkv := range possibleArgName {
										if fndprm = strings.EqualFold(psbprmnme, mpvk); fndprm {
											if validNames == nil {
												validNames = []string{}
											}
											if validNameType == nil {
												validNameType = []int{}
											}
											apprs([]rune(parseParam(stmnt.cn.dbParseSqlParam, len(validNames))))
											validNames = append(validNames, mpvk)
											validNameType = append(validNameType, mpkv)
											break
										}
									}
									if !fndprm {
										apprs(prmslbl[0])
										apprs(psblprmnme[:psblprmnmei])
									}
								} else {
									apprs(prmslbl[0])
									apprs(prmslbl[1])
								}
								psblprmnmei = 0
							} else {
								apprs(prmslbl[0])
								apprs(prmslbl[1])
							}
							prmslbli[1] = 0
							prvr = rune(0)
							prmslbli[0] = 0
						}
					} else {
						if prmsl := prmslbli[1]; prmsl > 0 {
							prmslbli[1] = 0
							prvr = rune(0)
							prmslbli[0] = 0
							apprs(prmslbl[0])
							if psblprmnmei > 0 {
								apprs(psblprmnme[:psblprmnmei])
								psblprmnmei = 0
							}
							apprs(prmslbl[1][:prmsl])
						} else {
							psblprmnme[psblprmnmei] = r
							psblprmnmei++
							prvr = r
							if psblprmnmei == len(psblprmnme) {
								prmslbli[1] = 0
								prvr = rune(0)
								prmslbli[0] = 0
								apprs(prmslbl[0])
								if psblprmnmei > 0 {
									apprs(psblprmnme[:psblprmnmei])
									psblprmnmei = 0
								}
							}
						}
					}
				}
			}
		}
		*stmntref = ""
		for r, rs, rerr := rnrr.ReadRune(); rs > 0 && rerr == nil; r, rs, rerr = rnrr.ReadRune() {
			if rs > 0 {
				parseRune(r)
			} else {
				if rerr != io.EOF {
					preperr = rerr
					return
				}
			}
		}
		if rnsbufl := len(rnsbuf); rnsbufl > 0 {
			*stmntref += string(rnsbuf[:rnsbufl])
			rnsbuf = nil
			//rsnsbfi = 0
		}
		if refrdr := stmnt.rdr; rdr != nil && refrdr != rdr {
			stmnt.rdr = rdr
		}

		if len(args) > 0 {
			if argssnc := stmnt.args; argssnc != nil {
				for ak, av := range args {
					argssnc.Store(ak, av)
				}
			}
		}
		if len(validNames) > 0 {
			stmnt.argnames = validNames[:]
			stmnt.argtypes = validNameType[:]
		}

		if refprms := stmnt.prms; prms != nil && prms != refprms {
			stmnt.prms = prms
		}
		if stmnt.prepstmnt == nil && stmnt.cn.isRemote() {

		} else {
			if ctx != nil && stmnt.ctx != ctx {
				stmnt.ctx = ctx
			}
			if stmnt.prepstmnt == nil {
				if db, dberr := stmnt.cn.DbInvoke(); dberr == nil && db != nil {
					if stmnt.ctx != nil {
						if stmnt.prepstmnt, preperr = db.PrepareContext(stmnt.ctx, stmnt.stmnt); preperr != nil {
							return
						}
					} else if stmnt.prepstmnt, preperr = db.Prepare(stmnt.stmnt); preperr != nil {
						return
					}
				} else if dberr != nil {
					preperr = dberr
				}
			}
		}
	}
	return
}

func (stmnt *Statement) Arguments() (args []interface{}) {
	if stmnt != nil && stmnt.cn != nil && len(stmnt.argnames) > 0 {
		if argssnc, argnames, argtypes, rdr := stmnt.args, stmnt.argnames, stmnt.argtypes, stmnt.rdr; argssnc != nil && len(argnames) > 0 && len(argnames) == len(argtypes) {
			for argn, argnme := range argnames {
				if argtpe := argtypes[argn]; argtpe == 0 {
					if argv, argvok := argssnc.Load(argnme); argvok {
						args = append(args, argv)
					}
				} else if prms := stmnt.prms; prms != nil && argtpe == 1 {
					args = append(args, strings.Join(prms.Parameter(argnme), ""))
				} else if rdr != nil && argtpe == 2 {
					if rows := rdr.rows; rows != nil {
						if clsi := rows.FieldIndex(argnme); clsi > -1 {
							args = append(args, rows.FieldByIndex(clsi))
						}
					}
				}
			}
		}
	}
	return
}

func (stmnt *Statement) Query() (rows RowsAPI, err error) {
	if stmnt != nil {
		if ctx, prep := stmnt.ctx, stmnt.prepstmnt; prep != nil {
			var sqlrw *sql.Rows = nil
			if ctx != nil {
				if sqlrw, err = prep.QueryContext(ctx, stmnt.Arguments()...); err == nil && sqlrw != nil {
					rows = newSqlRows(sqlrw, nil, nil)
				}
			} else {
				if sqlrw, err = prep.Query(stmnt.Arguments()...); err == nil && sqlrw != nil {
					rows = newSqlRows(sqlrw, nil, nil)
				}
			}
		}
	}
	return
}

func (stmnt *Statement) Close() (err error) {
	if stmnt != nil {
		if prms := stmnt.prms; prms != nil {
			stmnt.prms = nil
		}

		if args := stmnt.args; args != nil {
			stmnt.args = nil
		}

		if rdr := stmnt.rdr; rdr != nil {
			stmnt.rdr = nil
		}
		if prepstmnt := stmnt.prepstmnt; prepstmnt != nil {
			stmnt.prepstmnt = nil
			err = prepstmnt.Close()
		}
		if cn := stmnt.cn; cn != nil {
			stmnt.cn = nil
		}
	}
	return
}
