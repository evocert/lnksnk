package database

import (
	"context"
	"database/sql"
	"io"

	"github.com/lnksnk/lnksnk/concurrent"
	"github.com/lnksnk/lnksnk/fsutils"
	"github.com/lnksnk/lnksnk/iorw/active"

	//"lnksnk/logging"
	"strings"
	"sync"

	"github.com/lnksnk/lnksnk/parameters"
)

type DBMSHandler struct {
	ctx               context.Context
	fs                *fsutils.FSUtils
	dbms              *DBMS
	runtime           active.Runtime
	prms              parameters.ParametersAPI
	readers           *sync.Map
	exctrs            *sync.Map
	cchng             *concurrent.Map
	CallPrepStatement StatementHandlerFunc
}

func (dbmshndlr *DBMSHandler) Params() (prms parameters.ParametersAPI) {
	if dbmshndlr != nil {
		prms = dbmshndlr.prms
	}
	return
}

func (dbmshndlr *DBMSHandler) Exists(alias string) (exists bool) {
	if dbmshndlr != nil && alias != "" {
		if dbms := dbmshndlr.dbms; dbms != nil {
			if cnctns := dbms.cnctns; cnctns != nil {
				_, exists = cnctns.Load(alias)
			}
		}
	}
	return
}

func (dbmshndlr *DBMSHandler) Status(alias string) (status map[string]interface{}, err error) {
	if dbmshndlr != nil {
		if dbms := dbmshndlr.dbms; dbms != nil {
			if cnctns := dbms.cnctns; cnctns != nil {
				if cnv, _ := cnctns.Load(alias); cnv != nil {
					if cn, _ := cnv.(*Connection); cn != nil {
						dbstats, dbstatserr := cn.Status()
						if dbstatserr != nil {
							err = dbstatserr
							return
						}
						if status == nil {
							status = map[string]interface{}{}
						}
						status["idle"] = dbstats.Idle
						status["inuse"] = dbstats.InUse
						status["open"] = dbstats.OpenConnections
					}
				}
			}
		}
	}
	return
}

func (dbmshndlr *DBMSHandler) TryConnect(dbtype string, datasource string) (connected interface{}) {
	if dbmshndlr != nil && dbtype != "" {
		if dbms := dbmshndlr.dbms; dbms != nil {
			if datasource != "" {
				if dbinvkr, _ := dbms.DriverCnInvoker(dbtype); dbinvkr != nil {
					if db, dberr := dbinvkr(datasource); dberr != nil {
						connected = dberr
					} else if db != nil {
						defer db.Close()
						if dberr = db.Ping(); dberr != nil {
							connected = dberr
						} else {
							connected = true
						}
					}
				}
			} else {
				if cnctns := dbms.cnctns; cnctns != nil {
					if cn, _ := cnctns.Load(dbtype); cn != nil {
						if db, dberr := (cn.(*Connection)).DbInvoke(); dberr != nil {
							connected = dberr
						} else if db != nil {
							defer db.Close()
							if dberr = db.Ping(); dberr != nil {
								connected = dberr
							} else {
								connected = true
							}
						}
					}
				}
			}
		}
	}
	return
}

func (dbmshndlr *DBMSHandler) DriverName(alias string) (driver string) {
	if dbmshndlr != nil && alias != "" {
		if dbms := dbmshndlr.dbms; dbms != nil {
			if cntns := dbms.cnctns; cntns != nil {
				if cnv, _ := cntns.Load(alias); cnv != nil {
					driver = (cnv.(*Connection)).driverName
				}
			}
		}
	}
	return
}

func (dbmshndlr *DBMSHandler) InOut(in interface{}, out io.Writer, ioargs ...interface{}) (err error) {

	return
}

func (dbmshndlr *DBMSHandler) Connections() (cns []string) {
	if dbmshndlr != nil {
		if dbms := dbmshndlr.dbms; dbms != nil {
			if cnctns := dbms.cnctns; cnctns != nil {
				cnctns.Range(func(key, value any) bool {
					cns = append(cns, key.(string))
					return true
				})
			}
		}
	}
	return
}

func (dbmshndlr *DBMSHandler) Drivers() (drvrs []string) {
	if dbmshndlr != nil {
		if dbms := dbmshndlr.dbms; dbms != nil {
			if drivers := dbms.drivers; drivers != nil {
				drivers.Range(func(key, value any) bool {
					drvrs = append(drvrs, key.(string))
					return true
				})
			}
		}
	}
	return
}

func (dbmshndlr *DBMSHandler) Dispose() {
	if dbmshndlr != nil {
		if dbmshndlr.runtime != nil {
			dbmshndlr.runtime = nil
		}
		if dbmshndlr.dbms != nil {
			dbmshndlr.dbms = nil
		}
		if dbmshndlr.prms != nil {
			dbmshndlr.prms = nil
		}
		if readers := dbmshndlr.readers; readers != nil {
			dbmshndlr.readers = nil
			readers.Range(func(key, value any) bool {
				if rdr, _ := key.(*Reader); rdr != nil && rdr == value {
					rdr.Close()
				}
				return true
			})
		}
		if exctrs := dbmshndlr.exctrs; exctrs != nil {
			dbmshndlr.exctrs = nil
			exctrs.Range(func(key, value any) bool {
				if exctr := key.(*Executor); exctr != nil && exctr == value {
					exctr.Close()
				}
				return true
			})
		}
		dbmshndlr = nil
	}
}

func (dbmshndlr *DBMSHandler) QryArray(alias string, a ...interface{}) []interface{} {
	return dbmshndlr.QueryArray(alias, a...)
}

func (dbmshndlr *DBMSHandler) QueryArray(alias string, a ...interface{}) (arr []interface{}) {
	if rdr := dbmshndlr.Query(alias, a...); rdr != nil {
		arr = rdr.AsArray()
	}
	return
}

func (dbmshndlr *DBMSHandler) QryMap(alias string, a ...interface{}) map[string]interface{} {
	return dbmshndlr.QueryMap(alias, a...)
}

func (dbmshndlr *DBMSHandler) QueryMap(alias string, a ...interface{}) (mp map[string]interface{}) {
	if rdr := dbmshndlr.Query(alias, a...); rdr != nil {
		mp = rdr.AsMap()
	}
	return
}

func (dbmshndlr *DBMSHandler) Qry(alias string, a ...interface{}) *Reader {
	return dbmshndlr.Query(alias, a...)
}

func (dbmshndlr *DBMSHandler) Query(alias string, a ...interface{}) (reader *Reader) {
	if dbmshndlr != nil {
		if dbms := dbmshndlr.dbms; dbms != nil {
			if dbmshndlr.ctx != nil {
				a = append(a, dbmshndlr.ctx)
			}
			if dbmshndlr.runtime != nil {
				a = append([]interface{}{dbmshndlr.runtime}, a...)
			}
			if dbmshndlr.prms != nil {
				a = append(a, dbmshndlr.prms)
			}
			if dbmshndlr.fs != nil {
				a = append(a, dbmshndlr.fs)
			}
			if dbmshndlr.CallPrepStatement != nil {
				a = append(a, dbmshndlr.CallPrepStatement)
			}
			if reader = dbms.Query(alias, a...); reader != nil {
				readers := dbmshndlr.readers
				if readers == nil {
					readers = &sync.Map{}
					dbmshndlr.readers = readers
				}
				readers.Store(reader, reader)
				if reader.EventClose == nil {
					reader.EventClose = func(r *Reader) {
						readers.CompareAndDelete(r, r)
					}
				}
			}
		}
	}
	return
}

func (dbmshndlr *DBMSHandler) Exec(alias string, a ...interface{}) *Executor {
	return dbmshndlr.Execute(alias, a...)
}

func (dbmshndlr *DBMSHandler) Execute(alias string, a ...interface{}) (exctr *Executor) {
	if dbmshndlr != nil {
		if dbms := dbmshndlr.dbms; dbms != nil {
			if dbmshndlr.ctx != nil {
				a = append(a, dbmshndlr.ctx)
			}
			if dbmshndlr.runtime != nil {
				a = append([]interface{}{dbmshndlr.runtime}, a...)
			}
			if dbmshndlr.prms != nil {
				a = append(a, dbmshndlr.prms)
			}
			if dbmshndlr.fs != nil {
				a = append(a, dbmshndlr.fs)
			}
			if dbmshndlr.cchng != nil {
				a = append(a, dbmshndlr.cchng)
			}
			if dbmshndlr.CallPrepStatement != nil {
				a = append(a, dbmshndlr.CallPrepStatement)
			}
			if exctr = dbms.Execute(alias, a...); exctr != nil {
				exctrs := dbmshndlr.exctrs
				if exctrs == nil {
					dbmshndlr.exctrs = &sync.Map{}
					exctrs = dbmshndlr.exctrs
				}
				exctrs.Store(exctr, exctr)
				if exctr.EventClose == nil {
					exctr.EventClose = func(exc *Executor) {
						exctrs.CompareAndDelete(exc, exc)
					}
				}
			}
		}
	}
	return
}

func (dbmshndlr *DBMSHandler) Prep(alias string, a ...interface{}) *Executor {
	return dbmshndlr.Prepair(alias, a...)
}

func (dbmshndlr *DBMSHandler) Prepair(alias string, a ...interface{}) (exctr *Executor) {
	if dbmshndlr != nil {
		if dbms := dbmshndlr.dbms; dbms != nil {
			if dbmshndlr.ctx != nil {
				a = append(a, dbmshndlr.ctx)
			}
			if dbmshndlr.runtime != nil {
				a = append([]interface{}{dbmshndlr.runtime}, a...)
			}
			if dbmshndlr.prms != nil {
				a = append(a, dbmshndlr.prms)
			}
			if dbmshndlr.fs != nil {
				a = append(a, dbmshndlr.fs)
			}
			if dbmshndlr.cchng != nil {
				a = append(a, dbmshndlr.cchng)
			}
			if dbmshndlr.CallPrepStatement != nil {
				a = append(a, dbmshndlr.CallPrepStatement)
			}
			if exctr = dbms.Prepair(alias, a...); exctr != nil {
				exctrs := dbmshndlr.exctrs
				if exctrs == nil {
					dbmshndlr.exctrs = &sync.Map{}
					exctrs = dbmshndlr.exctrs
				}
				exctrs.Store(exctr, exctr)
				if exctr.EventClose == nil {
					exctr.EventClose = func(exc *Executor) {
						exctrs.CompareAndDelete(exc, exc)
					}
				}
			}
		}
	}
	return
}

func (dbmshndlr *DBMSHandler) Unreg(alias string) bool {
	return dbmshndlr.Unregister(alias)
}

func (dbmshndlr *DBMSHandler) Unregister(alias string) (unregistered bool) {
	if dbmshndlr != nil {
		if dbms := dbmshndlr.dbms; dbms != nil {
			unregistered = dbms.Unregister(alias)
		}
	}
	return
}

func (dbmshndlr *DBMSHandler) Reg(alias string, driver string, datasource string, a ...interface{}) bool {
	return dbmshndlr.Register(alias, driver, datasource, a...)
}

func (dbmshndlr *DBMSHandler) Register(alias string, driver string, datasource string, a ...interface{}) (registered bool) {
	if dbmshndlr != nil {
		if dbms := dbmshndlr.dbms; dbms != nil {
			registered = dbms.Register(alias, driver, datasource, a...)
		}
	}
	return
}

func (dbms *DBMS) DBMSHandler(ctx context.Context, runtime active.Runtime, prms parameters.ParametersAPI, cchng *concurrent.Map, fs *fsutils.FSUtils, callprepstmnt StatementHandlerFunc) (dbmshndlr *DBMSHandler) {
	dbmshndlr = &DBMSHandler{ctx: ctx, dbms: dbms, runtime: runtime, prms: prms, cchng: cchng, fs: fs, CallPrepStatement: callprepstmnt}
	return
}

type DBMS struct {
	cnctns  *sync.Map
	drivers *sync.Map
}

func (dbms *DBMS) DriverCnInvoker(driver string) (dbinvoker func(string, ...interface{}) (*sql.DB, error), dbsqlparseparam func(int) string) {
	if dbms != nil && driver != "" {
		if drivers := dbms.drivers; drivers != nil {
			if dvrd, dvrdok := drivers.Load(driver); dvrdok {
				if dvr, _ := dvrd.(*dbdriver); dvr != nil {
					dbinvoker = dvr.dbInvoker
					dbsqlparseparam = dvr.dbParseSqlParam
				}
			}
		}
	}
	return
}

func NewDBMS() (dbms *DBMS) {
	dbms = &DBMS{cnctns: &sync.Map{}, drivers: &sync.Map{}}
	return
}

var glbdbms *DBMS

// GLOBALDBMS - Global DBMS instance
func GLOBALDBMS() *DBMS {
	return glbdbms
}

func init() {
	if glbdbms == nil {
		glbdbms = NewDBMS()
	}
}

func (dbms *DBMS) Unregister(alias string, a ...interface{}) (unregistered bool) {
	if cnctns := dbms.cnctns; cnctns != nil && alias != "" {
		if cnv, cnvok := cnctns.Load(alias); cnvok {
			cnctns.CompareAndDelete(alias, cnv)
			if cn, _ := cnv.(*Connection); cn != nil {
				cn.Dispose()
			}
			unregistered = true
		}
	}
	return
}

func (dbms *DBMS) Register(alias string, driver string, datasource string, a ...interface{}) (registered bool) {
	if alias != "" && driver != "" && datasource != "" {
		if cnctns, drivers := dbms.cnctns, dbms.drivers; cnctns != nil && drivers != nil {
			var cn *Connection = nil
			var drvdbinvoker func(string, ...interface{}) (*sql.DB, error) = nil
			var drvdbsqlPrameParse func(int) string = nil
			var doesExits = func() bool {
				if cnd, cnvok := cnctns.Load(alias); cnvok {
					if cn, _ = cnd.(*Connection); cn != nil {
						return true
					}
				}
				return false
			}
			var doesDriverExits = func() (exists bool) {
				if dvrd, dvrok := drivers.Load(driver); dvrok {
					if dvr, _ := dvrd.(*dbdriver); dvr != nil {
						drvdbinvoker = dvr.dbInvoker
						drvdbsqlPrameParse = dvr.dbParseSqlParam
					}
					return drvdbinvoker != nil
				}
				return
			}
			func() {
				if strings.HasPrefix(datasource, "http://") || strings.HasPrefix(datasource, "https://") || strings.HasPrefix(datasource, "ws://") || strings.HasPrefix(datasource, "wss://") {
					if doesExits() {
						if cn.driverName != driver {
							cn.driverName = driver
							cn.dataSource = datasource
							registered = true
						}
					} else if cn = NewConnection(dbms, driver, datasource); cn != nil {
						func() {
							cnctns.Store(alias, cn)
						}()
						registered = true
					}
				} else if doesDriverExits() {
					if doesExits() {
						if cn.driverName != driver {
							cn.driverName = driver
							cn.dataSource = datasource
							cn.dbParseSqlParam = drvdbsqlPrameParse
							cn.dbinvoker = drvdbinvoker
							registered = true
						}
					} else if cn = NewConnection(dbms, driver, datasource); cn != nil {
						func() {
							cnctns.Store(alias, cn)
						}()
						registered = true
					}
				}
			}()
		}
	}
	return
}

type dbdriver struct {
	dbInvoker       func(string, ...interface{}) (*sql.DB, error)
	dbParseSqlParam func(int) string
}

func (dbms *DBMS) RegisterDriver(driver string, invokedbcall func(string, ...interface{}) (*sql.DB, error), parseDbSqlParamCall func(int) string) {
	if driver != "" && invokedbcall != nil {
		if drivers := dbms.drivers; drivers != nil {
			if parseDbSqlParamCall == nil {
				parseDbSqlParamCall = func(i int) string { return "?" }
			}
			drivers.Store(driver, &dbdriver{dbInvoker: invokedbcall, dbParseSqlParam: parseDbSqlParamCall})
		}
	}
}

func (dbms *DBMS) Query(alias string, a ...interface{}) (reader *Reader) {
	if dbms != nil && alias != "" {
		if cnctns, drivers := dbms.cnctns, dbms.drivers; cnctns != nil && drivers != nil {
			var oninit interface{} = nil
			var onprepcolumns interface{} = nil
			var onprepdata interface{} = nil
			var onerror interface{} = nil

			var onfinalize interface{} = nil
			var onrow interface{} = nil
			var onrowerror interface{} = nil
			var onselect interface{} = nil
			var onnext interface{} = nil
			var rdr *Reader = nil
			var prms *parameters.Parameters = nil
			var args map[string]interface{} = nil
			var runtime active.Runtime = nil
			var preprdsexctrs [][]interface{} = nil
			var al = 0

			if al = len(a); al > 0 {
				var ai = 0
				for ai < al {
					if a[ai] == nil {
						a = append(a[:ai], a[ai+1:]...)
						al--
						continue
					} else {
						if runtimed, _ := a[ai].(active.Runtime); runtimed != nil {
							if runtime == nil {
								runtime = runtimed
							}
							a = append(a[:ai], a[ai+1:]...)
							al--
							continue
						} else if mpd, _ := a[ai].(map[string]interface{}); mpd != nil {
							for mk, mv := range mpd {
								if mk == "error" {
									if runtime != nil && mv != nil && onerror == nil {
										onerror = mv
									}
									delete(mpd, mk)
								} else if mk == "init" {
									if runtime != nil && mv != nil && oninit == nil {
										oninit = mv
									}
									delete(mpd, mk)
								} else if mk == "prep-columns" {
									if runtime != nil && mv != nil && onprepcolumns == nil {
										onprepcolumns = mv
									}
									delete(mpd, mk)
								} else if mk == "prep-data" {
									if runtime != nil && mv != nil && onprepdata == nil {
										onprepdata = mv
									}
									delete(mpd, mk)
								} else if mk == "finalize" {
									if runtime != nil && onfinalize == nil {
										onfinalize = mv
									}
									delete(mpd, mk)
								} else if mk == "row" {
									if runtime != nil && mv != nil && onrow == nil {
										onrow = mv
									}
									delete(mpd, mk)
								} else if mk == "row-error" {
									if runtime != nil && mv != nil && onrowerror == nil {
										onrowerror = mv
									}
									delete(mpd, mk)
								} else if mk == "next" {
									if runtime != nil && mv != nil && onnext == nil {
										onnext = mv
									}
									delete(mpd, mk)
								} else if mk == "select" {
									if runtime != nil && mv != nil && onselect == nil {
										onselect = mv
									}
									delete(mpd, mk)
								} else {
									if args == nil {
										args = map[string]interface{}{}
									}
									args[mk] = mv
								}
							}
							a = append(a[:ai], a[ai+1:]...)
							al--
							continue
						} else if mpd, _ := a[ai].(map[string]string); mpd != nil {
							for mk, mv := range mpd {
								if args == nil {
									args = map[string]interface{}{}
								}
								args[mk] = mv
							}
							a = append(a[:ai], a[ai+1:]...)
							al--
							continue
						} else if rdrd, _ := a[ai].(*Reader); rdrd != nil {
							if rdr == nil {
								rdr = rdrd
							}
							a = append(a[:ai], a[ai+1:]...)
							al--
							continue
						} else if prmsd, _ := a[ai].(*parameters.Parameters); prmsd != nil {
							if prms == nil {
								prms = prmsd
							}
							a = append(a[:ai], a[ai+1:]...)
							al--
							continue
						} else if arrd, _ := a[ai].([]interface{}); arrd != nil {
							if len(arrd) >= 2 {
								if aliasd, _ := arrd[0].(string); aliasd != "" {
									if !(strings.EqualFold(aliasd, "csv") || strings.EqualFold(aliasd, "json") || strings.EqualFold(aliasd, "xml")) {
										func() {
											if _, hascn := cnctns.Load(aliasd); hascn {
												preprdsexctrs = append(preprdsexctrs, arrd)
											}
										}()
									}
								}
							}
							a = append(a[:ai], a[ai+1:]...)
							al--
							continue
						}
					}
					ai++
				}
			}
			if al > 0 {
				if alias == "csv" || alias == "json" || alias == "xml" {
					if alias == "csv" {
						if reader = NewReader(a[0], alias, args, nil, oninit, onprepcolumns, onprepdata, onnext, onselect, onrow, onrowerror, onerror, onfinalize, runtime); reader != nil {
							if err := reader.Prep(); err != nil {
								if reader.EventError != nil {
									reader.EventError(err)
								}
								reader.Close()
								reader = nil
							} else {
								if len(preprdsexctrs) > 0 {
									for _, exctrargs := range preprdsexctrs {
										exctralias, _ := exctrargs[0].(string)
										exctrargs = append(exctrargs[1:], reader)
										if reader.stmnt != nil && reader.stmnt.prms != nil {
											exctrargs = append(exctrargs, reader.stmnt.prms)
										}
										if prprdectr := dbms.Prepair(exctralias, exctrargs...); prprdectr != nil {
											if reader.exctrs == nil {
												reader.exctrs = map[*Executor]*Executor{}
											}
											reader.exctrs[prprdectr] = prprdectr
											reader.orderedexctrs = append(reader.orderedexctrs, prprdectr)
											prprdectr.EventClose = func(exec *Executor) {
												delete(reader.exctrs, exec)
												for execn, execref := range reader.orderedexctrs {
													if exec == execref {
														reader.orderedexctrs = append(reader.orderedexctrs[:execn], reader.orderedexctrs[execn+1:]...)
														break
													}
												}
											}
										}
									}
								}
								reader.EventInit(reader)
							}
						}
					}
				} else if alias == "stream" && onprepcolumns != nil && onprepdata != nil {
					if reader = NewReader(a[0], alias, args, nil, oninit, onprepcolumns, onprepdata, onnext, onselect, onrow, onrowerror, onerror, onfinalize, runtime); reader != nil {
						if err := reader.Prep(); err != nil {
							if reader.EventError != nil {
								reader.EventError(err)
							}
							reader.Close()
							reader = nil
						} else {
							if len(preprdsexctrs) > 0 {
								for _, exctrargs := range preprdsexctrs {
									exctralias, _ := exctrargs[0].(string)
									exctrargs = append(exctrargs[1:], reader)
									if reader.stmnt != nil && reader.stmnt.prms != nil {
										exctrargs = append(exctrargs, reader.stmnt.prms)
									}
									if prprdectr := dbms.Prepair(exctralias, exctrargs...); prprdectr != nil {
										if reader.exctrs == nil {
											reader.exctrs = map[*Executor]*Executor{}
										}
										reader.exctrs[prprdectr] = prprdectr
										reader.orderedexctrs = append(reader.orderedexctrs, prprdectr)
										prprdectr.EventClose = func(exec *Executor) {
											delete(reader.exctrs, exec)
											for execn, execref := range reader.orderedexctrs {
												if exec == execref {
													reader.orderedexctrs = append(reader.orderedexctrs[:execn], reader.orderedexctrs[execn+1:]...)
													break
												}
											}
										}
									}
								}
							}
							reader.EventInit(reader)
						}
					}
				} else if alias != "" {
					var cn *Connection = nil
					if cn = func() *Connection {
						if cnv, cnok := cnctns.Load(alias); cnok {
							return cnv.(*Connection)
						}
						return nil
					}(); cn != nil {
						var stmnt = cn.Stmnt()
						if stmnt != nil {
							if err := stmnt.Prepair(prms, rdr, args, a...); err == nil {
								if reader = NewReader(nil, "", nil, stmnt, oninit, onprepcolumns, onprepdata, onnext, onselect, onrow, onrowerror, onerror, onfinalize, runtime); reader != nil {
									if err = reader.Prep(); err != nil {
										if reader.EventError != nil {
											reader.EventError(err)
										}
										reader.Close()
										reader = nil
									} else {
										if len(preprdsexctrs) > 0 {
											for _, exctrargs := range preprdsexctrs {
												exctralias, _ := exctrargs[0].(string)
												exctrargs = append(exctrargs[1:], reader)
												if reader.stmnt != nil && reader.stmnt.prms != nil {
													exctrargs = append(exctrargs, reader.stmnt.prms)
												}
												if prprdectr := dbms.Prepair(exctralias, exctrargs...); prprdectr != nil {
													if reader.exctrs == nil {
														reader.exctrs = map[*Executor]*Executor{}
													}
													reader.exctrs[prprdectr] = prprdectr
													reader.orderedexctrs = append(reader.orderedexctrs, prprdectr)
													prprdectr.EventClose = func(exec *Executor) {
														delete(reader.exctrs, exec)
														for execn, execref := range reader.orderedexctrs {
															if exec == execref {
																reader.orderedexctrs = append(reader.orderedexctrs[:execn], reader.orderedexctrs[execn+1:]...)
																break
															}
														}
													}
												}
											}
										}
										reader.EventInit(reader)
									}
								}
							} else {
								defer stmnt.Close()
								invokeErrorEvent(onerror, err, runtime)
							}
						}
					}
				}
			}
		}
	}
	return
}

func (dbms *DBMS) Execute(alias string, a ...interface{}) (exectr *Executor) {
	if dbms != nil && alias != "" {
		if cnctns := dbms.cnctns; cnctns != nil {
			var oninit interface{} = nil
			var onerror interface{} = nil

			var onfinalize interface{} = nil
			var onexec interface{} = nil
			var onexecerror interface{} = nil
			var rdr *Reader = nil
			var prms *parameters.Parameters = nil
			var args map[string]interface{} = nil
			var runtime active.Runtime = nil
			var al = 0
			if al = len(a); al > 0 {
				var ai = 0
				for ai < al {
					if a[ai] == nil {
						a = append(a[:ai], a[ai+1:]...)
						al--
						continue
					} else {
						if runtimed, _ := a[ai].(active.Runtime); runtimed != nil {
							if runtime == nil {
								runtime = runtimed
							}
							a = append(a[:ai], a[ai+1:]...)
							al--
							continue
						} else if mpd, _ := a[ai].(map[string]interface{}); mpd != nil {
							for mk, mv := range mpd {
								if mk == "error" {
									if runtime != nil && mv != nil && onerror == nil {
										onerror = mv
									}
									delete(mpd, mk)
								} else if mk == "init" {
									if runtime != nil && mv != nil && oninit == nil {
										oninit = mv
									}
									delete(mpd, mk)
								} else if mk == "finalize" {
									if runtime != nil && onfinalize == nil {
										onfinalize = mv
									}
									delete(mpd, mk)
								} else if mk == "exec" {
									if runtime != nil && mv != nil && onexec == nil {
										onexec = mv
									}
									delete(mpd, mk)
								} else if mk == "exec-error" {
									if runtime != nil && mv != nil && onexecerror == nil {
										onexecerror = mv
									}
									delete(mpd, mk)
								} else {
									if args == nil {
										args = map[string]interface{}{}
									}
									args[mk] = mv
								}
							}
							a = append(a[:ai], a[ai+1:]...)
							al--
							continue
						} else if mpd, _ := a[ai].(map[string]string); mpd != nil {
							for mk, mv := range mpd {
								if args == nil {
									args = map[string]interface{}{}
								}
								args[mk] = mv
							}
							a = append(a[:ai], a[ai+1:]...)
							al--
							continue
						} else if rdrd, _ := a[ai].(*Reader); rdrd != nil {
							if rdr == nil {
								rdr = rdrd
							}
							a = append(a[:ai], a[ai+1:]...)
							al--
							continue
						} else if prmsd, _ := a[ai].(*parameters.Parameters); prmsd != nil {
							if prms == nil {
								prms = prmsd
							}
							a = append(a[:ai], a[ai+1:]...)
							al--
							continue
						}
					}
					ai++
				}
			}
			if al > 0 {
				if alias != "" {
					var cn *Connection = nil
					if cn = func() *Connection {
						if cnv, cnok := cnctns.Load(alias); cnok {
							return cnv.(*Connection)
						}
						return nil
					}(); cn != nil {
						var stmnt = cn.Stmnt()
						if stmnt != nil {
							if err := stmnt.Prepair(prms, rdr, args, a...); err == nil {
								if exectr = NewExecutor(stmnt, false, oninit, onexec, onexecerror, onerror, onfinalize, runtime /*, logger*/); exectr != nil {
									if err = exectr.Exec(); err != nil {
										exectr.Close()
										invokeErrorEvent(onerror, err, runtime)
									}
								}
							} else {
								invokeErrorEvent(onerror, err, runtime)
							}
						}
					}
				}
			}
		}
	}
	return
}

func invokeErrorEvent(event interface{}, err error, runtime active.Runtime) {
	if err != nil && event != nil {
		if eventd, _ := event.(func(error)); eventd != nil {
			eventd(err)
			return
		}
		if runtime != nil {
			runtime.InvokeFunction(event, err)
		}
	}
}

func (dbms *DBMS) Prepair(alias string, a ...interface{}) (exectr *Executor) {
	if dbms != nil && alias != "" {
		if cnctns := dbms.cnctns; cnctns != nil {
			var oninit interface{} = nil
			var onerror interface{} = nil

			var onfinalize interface{} = nil
			var onexec interface{} = nil
			var onexecerror interface{} = nil
			var rdr *Reader = nil
			var prms *parameters.Parameters = nil
			var args map[string]interface{} = nil
			var runtime active.Runtime = nil
			var al = 0
			if al = len(a); al > 0 {
				var ai = 0
				for ai < al {
					if a[ai] == nil {
						a = append(a[:ai], a[ai+1:]...)
						al--
						continue
					} else {
						if runtimed, _ := a[ai].(active.Runtime); runtimed != nil {
							if runtime == nil {
								runtime = runtimed
							}
							a = append(a[:ai], a[ai+1:]...)
							al--
							continue
						} else if mpd, _ := a[ai].(map[string]interface{}); mpd != nil {
							for mk, mv := range mpd {
								if mk == "error" {
									if runtime != nil && mv != nil && onerror == nil {
										onerror = mv
									}
									delete(mpd, mk)
								} else if mk == "init" {
									if runtime != nil && mv != nil && oninit == nil {
										oninit = mv
									}
									delete(mpd, mk)
								} else if mk == "finalize" {
									if runtime != nil && onfinalize == nil {
										onfinalize = mv
									}
									delete(mpd, mk)
								} else if mk == "exec" {
									if runtime != nil && mv != nil && onexec == nil {
										onexec = mv
									}
									delete(mpd, mk)
								} else if mk == "exec-error" {
									if runtime != nil && mv != nil && onexecerror == nil {
										onexecerror = mv
									}
									delete(mpd, mk)
								} else {
									if args == nil {
										args = map[string]interface{}{}
									}
									args[mk] = mv
								}
							}
							a = append(a[:ai], a[ai+1:]...)
							al--
							continue
						} else if mpd, _ := a[ai].(map[string]string); mpd != nil {
							for mk, mv := range mpd {
								if args == nil {
									args = map[string]interface{}{}
								}
								args[mk] = mv
							}
							a = append(a[:ai], a[ai+1:]...)
							al--
							continue
						} else if rdrd, _ := a[ai].(*Reader); rdrd != nil {
							if rdr == nil {
								rdr = rdrd
							}
							a = append(a[:ai], a[ai+1:]...)
							al--
							continue
						} else if prmsd, _ := a[ai].(*parameters.Parameters); prmsd != nil {
							if prms == nil {
								prms = prmsd
							}
							a = append(a[:ai], a[ai+1:]...)
							al--
							continue
						}
					}
					ai++
				}
			}
			if al > 0 {
				if alias != "" {
					var cn *Connection = nil
					if cn = func() *Connection {
						if cnv, cnok := cnctns.Load(alias); cnok {
							return cnv.(*Connection)
						}
						return nil
					}(); cn != nil {
						var stmnt = cn.Stmnt()
						if stmnt != nil {
							if err := stmnt.Prepair(prms, rdr, args, a...); err == nil {
								if exectr = NewExecutor(stmnt, true, oninit, onexec, onexecerror, onerror, onfinalize, runtime /*, logger*/); exectr != nil {
									if err = exectr.Exec(); err != nil {
										defer exectr.Close()
										invokeErrorEvent(onerror, err, runtime)
									} else {
										exectr.prpOnly = false
									}
								}
							} else {
								invokeErrorEvent(onerror, err, runtime)
							}
						}
					}
				}
			}
		}
	}
	return
}

func (dbms *DBMS) Exists(alias string) (exists bool) {
	if dbms != nil {
		if cnctns := dbms.cnctns; cnctns != nil {
			_, exists = cnctns.Load(alias)
		}
	}
	return
}
