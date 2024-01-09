package database

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/evocert/lnksnk/iorw"
	"github.com/evocert/lnksnk/iorw/active"

	"reflect"
	"strings"
	"time"
)

type streamType int

const (
	csvStrm streamType = iota
	jsonStrm
	xmlStrm
)

type Reader struct {
	nextreaders []*Reader
	strmrdr     *iorw.EOFCloseSeekReader
	//preStrmRdr       *iorw.BuffReader
	strmsttngs       map[string]interface{}
	strmtpe          streamType
	stmnt            *Statement
	rows             RowsAPI
	RowNr            int64
	started          bool
	first            bool
	last             bool
	EventPrepColumns PrepColumnsFunc
	EventPrepData    PrepDataFunc
	EventInit        ReaderInitFunc
	EventError       ErrorFunc
	EventNext        RowNextFunc
	EventSelect      RowSelectFunc
	EventRow         RowFunc
	EventRowError    RowErrorFunc
	EventFinalize    ReaderFinalizeFunc
	EventClose       func(*Reader)
	exctrs           map[*Executor]*Executor
	orderedexctrs    []*Executor
	CastTypeValue    func(interface{}, interface{}) (interface{}, bool)
}

type PrepColumnsFunc func(*Reader, func(...string)) error
type PrepDataFunc func(*Reader, func(...interface{})) error
type ReaderInitFunc func(*Reader) error
type ErrorFunc func(error)
type RowFunc func(*Reader) error
type RowErrorFunc func(error, *Reader) (bool, error)
type RowNextFunc func(*Reader) (bool, error)
type RowSelectFunc func(*Reader) (bool, error)
type ReaderFinalizeFunc func(*Reader) error

func NewReader(strmrdrd interface{}, strmtpe string, strmsttngs map[string]interface{}, stmnt *Statement, oninit interface{}, onprepcolumns interface{}, onprepdata interface{}, onnext interface{}, onselect interface{}, onrow interface{}, onrowerror interface{}, onerror interface{}, onfinalize interface{}, runtime active.Runtime /*, logger logging.Logger*/) (reader *Reader) {
	var strmtype = func() streamType {
		if strings.EqualFold(strmtpe, "csv") {
			return csvStrm
		} else if strings.EqualFold(strmtpe, "json") {
			return jsonStrm
		} else if strings.EqualFold(strmtpe, "xml") {
			return xmlStrm
		} else {
			return csvStrm
		}
	}
	var strmrdr *iorw.EOFCloseSeekReader = nil
	if strmrdr, _ = strmrdrd.(*iorw.EOFCloseSeekReader); strmrdr == nil {
		if rdr, _ := strmrdrd.(io.Reader); rdr != nil {
			strmrdr = iorw.NewEOFCloseSeekReader(rdr, true)
		} else if rdrs, _ := strmrdrd.(string); rdrs != "" {
			strmrdr = iorw.NewEOFCloseSeekReader(strings.NewReader(rdrs), true)
		}
	}
	reader = &Reader{strmrdr: strmrdr, stmnt: stmnt, strmtpe: strmtype(), strmsttngs: strmsttngs /*, runtime: runtime /*, LOG: logger*/}
	if onerror == nil {
		reader.EventError = func(err error) {}
	} else {
		if donerror, _ := onerror.(ErrorFunc); donerror != nil {
			reader.EventError = donerror
		} else if donerror, _ := onerror.(func(error)); donerror != nil {
			reader.EventError = donerror
		} else if runtime != nil && onerror != nil {
			reader.EventError = func(err error) {
				runtime.InvokeFunction(onerror, err)
			}
		}
	}
	if onfinalize == nil {
		reader.EventFinalize = func(*Reader) error { return nil }
	} else {
		if donfinalize, _ := onfinalize.(ReaderFinalizeFunc); donfinalize != nil {
			reader.EventFinalize = donfinalize
		} else if donfinalize, _ := onfinalize.(func(*Reader) error); donfinalize != nil {
			reader.EventFinalize = donfinalize
		} else if runtime != nil && onfinalize != nil {
			reader.EventFinalize = func(rdr *Reader) (err error) {
				runtime.InvokeFunction(onfinalize, rdr)
				return
			}
		}
	}
	if oninit == nil {
		reader.EventInit = func(*Reader) error { return nil }
	} else {
		if doninit, _ := oninit.(ReaderInitFunc); doninit != nil {
			reader.EventInit = doninit
		} else if doninit, _ := oninit.(func(*Reader) error); doninit != nil {
			reader.EventInit = doninit
		} else if runtime != nil && oninit != nil {
			reader.EventInit = func(rdr *Reader) (err error) {
				runtime.InvokeFunction(oninit, rdr)
				return
			}
		}
	}
	if onprepcolumns == nil {
		reader.EventPrepColumns = func(*Reader, func(...string)) error { return nil }
	} else {
		if donprepcolumns, _ := onprepcolumns.(PrepColumnsFunc); donprepcolumns != nil {
			reader.EventPrepColumns = donprepcolumns
		} else if donprepcolumns, _ := onprepcolumns.(func(*Reader, func(...string)) error); donprepcolumns != nil {
			reader.EventPrepColumns = donprepcolumns
		} else if runtime != nil && onprepcolumns != nil {
			reader.EventPrepColumns = func(rdr *Reader, prepcols func(...string)) (err error) {
				runtime.InvokeFunction(onprepcolumns, rdr, prepcols)
				return
			}
		}
	}
	if onprepdata == nil {
		reader.EventPrepData = func(*Reader, func(...interface{})) error { return nil }
	} else {
		if donprepdata, _ := onprepdata.(PrepDataFunc); donprepdata != nil {
			reader.EventPrepData = donprepdata
		} else if donprepdata, _ := onprepdata.(func(*Reader, func(...interface{})) error); donprepdata != nil {
			reader.EventPrepData = donprepdata
		} else if runtime != nil && onprepdata != nil {
			reader.EventPrepData = func(rdr *Reader, prepData func(...interface{})) (err error) {
				runtime.InvokeFunction(onprepdata, rdr, prepData)
				return
			}
		}
	}
	if onnext == nil {
		reader.EventNext = func(rdr *Reader) (cannext bool, err error) {
			return err == nil, err
		}
	} else {
		if donnext, _ := onnext.(RowNextFunc); donnext != nil {
			reader.EventNext = func(rdr *Reader) (cannext bool, err error) {
				cannext, err = donnext(rdr)

				return cannext, err
			}
		} else if donnext, _ := onnext.(func(*Reader) (bool, error)); donnext != nil {
			reader.EventNext = func(rdr *Reader) (cannext bool, err error) {
				cannext, err = donnext(rdr)
				return cannext, err
			}
		} else if runtime != nil && onnext != nil {
			reader.EventNext = func(rdr *Reader) (cannext bool, err error) {
				if err = rdr.rows.Scan(rdr.CastTypeValue); err == nil {
					invkresult := runtime.InvokeFunction(onnext, rdr)
					if cannextdb, ignrerrok := invkresult.(bool); ignrerrok {
						cannext = cannextdb
					} else if ignrerre, _ := invkresult.(error); ignrerre != nil {
						err = ignrerre
					}
				}
				return
			}
		}
	}
	if onselect == nil {
		reader.EventSelect = func(rdr *Reader) (selected bool, err error) {
			return err == nil, err
		}
	} else {
		if donselect, _ := onselect.(RowSelectFunc); donselect != nil {
			reader.EventSelect = func(rdr *Reader) (selected bool, err error) {
				selected, err = donselect(rdr)
				return
			}
		} else if donselect, _ := onselect.(func(*Reader) (bool, error)); donselect != nil {
			reader.EventSelect = func(rdr *Reader) (selected bool, err error) {
				selected, err = donselect(rdr)
				return
			}
		} else if runtime != nil && onselect != nil {
			reader.EventSelect = func(rdr *Reader) (selected bool, err error) {
				invkresult := runtime.InvokeFunction(onselect, rdr)
				if selectedb, ignrerrok := invkresult.(bool); ignrerrok {
					selected = selectedb
				} else if ignrerre, _ := invkresult.(error); ignrerre != nil {
					err = ignrerre
				}
				return
			}
		}
	}
	if onrow == nil {
		reader.EventRow = func(rdr *Reader) error {
			processReaderExecutors(rdr)
			return nil
		}
	} else {
		if donrow, _ := onrow.(RowFunc); donrow != nil {
			reader.EventRow = func(rdr *Reader) error {
				processReaderExecutors(rdr)
				return donrow(rdr)
			}
		} else if donrow, _ := onrow.(func(*Reader) error); donrow != nil {
			reader.EventRow = func(rdr *Reader) error {
				processReaderExecutors(rdr)
				return donrow(rdr)
			}
		} else if runtime != nil && onrow != nil {
			reader.EventRow = func(rdr *Reader) (err error) {
				processReaderExecutors(rdr)
				if invkresult := runtime.InvokeFunction(onrow, reader); invkresult != nil {
					if invrsltb, _ := invkresult.(bool); invrsltb {
						rdr.last = false
					}
				}
				return
			}
		}
	}
	if onrowerror == nil {
		reader.EventRowError = func(err error, r *Reader) (bool, error) { return false, nil }
	} else {
		if donrowerror, _ := onrowerror.(RowErrorFunc); donrowerror != nil {
			reader.EventRowError = donrowerror
		} else if donrowerror, _ := onrowerror.(func(error, *Reader) (bool, error)); donrowerror != nil {
			reader.EventRowError = donrowerror
		} else if runtime != nil && onrowerror != nil {
			reader.EventRowError = func(rdrerr error, rdr *Reader) (ignrerr bool, err error) {
				if invkresult := runtime.InvokeFunction(onrowerror, rdrerr, rdr); invkresult != nil {
					if ignrerrb, ignrerrok := invkresult.(bool); ignrerrok {
						ignrerr = ignrerrb
					} else if ignrerre, _ := invkresult.(error); ignrerre != nil {
						err = ignrerre
					}
				}
				return
			}
		}
	}

	return
}

func (rdr *Reader) callFinalise() {
	if rdr != nil {
		if EventFinalize := rdr.EventFinalize; EventFinalize != nil {
			rdr.EventFinalize = nil
			EventFinalize(rdr)
		}
	}
}

func (rdr *Reader) AppendReader(rdrs ...*Reader) (appended bool) {
	if rdr != nil {
		appended = rdr.InsertAfterReader(nil, rdrs...)
	}
	return
}

func (rdr *Reader) RemoveReader(rdrstormv ...*Reader) (removed bool) {
	if rdr != nil {
		if nxtrdrsl, nextreaders := len(rdr.nextreaders), rdr.nextreaders; nxtrdrsl > 0 {
			if rmvrdsl := len(rdrstormv); rmvrdsl > 0 {
				for rdri, rd := 0, rdrstormv[0]; rdri < rmvrdsl; rd = rdrstormv[rdri] {
					if rd == nil {
						rdrstormv = append(rdrstormv[:rdri], rdrstormv[rdri+1:]...)
						rmvrdsl--
						continue
					}
					rdri++
				}
				if rmvrdsl > 0 {
					for nxtri, nxtrd := 0, nextreaders[0]; nxtri < nxtrdrsl && rmvrdsl > 0; nxtrd = nextreaders[nxtri] {
						for rdri, rd := 0, rdrstormv[0]; rdri < rmvrdsl && nxtrdrsl > 0; rd = rdrstormv[rdri] {
							if rd == nxtrd {
								rdrstormv = append(nextreaders[:nxtri], nextreaders[nxtri+1:]...)
								nxtrdrsl--
								rdrstormv = append(rdrstormv[:rdri], rdrstormv[rdri+1:]...)
								rmvrdsl--
								continue
							}
							rdri++
						}
					}
					rdr.nextreaders = nextreaders
				}
			}
		}
	}
	return
}

func (rdr *Reader) PreAppendReader(rdrs ...*Reader) (preappended bool) {
	if rdr != nil {
		preappended = rdr.InsertBeforeReader(nil, rdrs...)
	}
	return
}

func (rdr *Reader) InsertBeforeReader(bfrrdr *Reader, rdrs ...*Reader) (inserted bool) {
	if rdr != nil {
		if rdrsl := len(rdrs); rdrsl > 0 {
			for rdri, rd := 0, rdrs[0]; rdri < rdrsl; rd = rdrs[rdri] {
				if rd == nil {
					rdrs = append(rdrs[:rdri], rdrs[rdri+1:]...)
					rdrsl--
					continue
				}
				rdri++
			}
			if rdrsl > 0 {
				if bfrrdr == nil {
					rdr.nextreaders = append(rdrs, rdr.nextreaders...)
				} else if rdrsl = len(rdr.nextreaders); rdrsl > 0 {
					rdrti := rdrsl - 1
					for rdri, rd := range rdr.nextreaders {
						if rd == bfrrdr {
							if rdri == 0 {
								rdr.nextreaders = append(rdrs, rdr.nextreaders...)
							} else {
								rdr.nextreaders = append(rdr.nextreaders[:rdri], append(rdrs, rdr.nextreaders[rdri:]...)...)
							}
							break
						} else if rdrti > rdri {
							if rd = rdrs[rdrti]; rd == bfrrdr {
								rdr.nextreaders = append(rdr.nextreaders[:rdri], append(rdrs, rdr.nextreaders[rdri:]...)...)
								break
							}
							rdrti--
						}
					}
				}
			}
		}
	}
	return
}

func (rdr *Reader) InsertAfterReader(aftrrdr *Reader, rdrs ...*Reader) (inserted bool) {
	if rdr != nil {
		if rdrsl := len(rdrs); rdrsl > 0 {
			for rdri, rd := 0, rdrs[0]; rdri < rdrsl; rd = rdrs[rdri] {
				if rd == nil {
					rdrs = append(rdrs[:rdri], rdrs[rdri+1:]...)
					rdrsl--
					continue
				}
				rdri++
			}
			if rdrsl > 0 {
				if aftrrdr == nil {
					rdr.nextreaders = append(rdr.nextreaders, rdrs...)
				} else if rdrsl = len(rdr.nextreaders); rdrsl > 0 {
					rdrti := rdrsl - 1
					for rdri, rd := range rdr.nextreaders {
						if rd == aftrrdr {
							if rdri == rdrsl-1 {
								rdr.nextreaders = append(rdr.nextreaders, rdrs...)
							} else {
								rdr.nextreaders = append(rdr.nextreaders[:rdri+1], append(rdrs, rdr.nextreaders[rdri+1:]...)...)
							}
							break
						} else if rdrti > rdri {
							if rd := rdrs[rdrti]; rd == aftrrdr {
								rdr.nextreaders = append(rdr.nextreaders[:rdri+1], append(rdrs, rdr.nextreaders[rdri+1:]...)...)
								break
							}
							rdrti--
						}
					}
				}
			}
		}
	}
	return
}

func (rdr *Reader) NextReader() (nxtrdr *Reader) {
	if rdr != nil {
		if nxtrdrsl, nextreaders := len(nxtrdr.nextreaders), nxtrdr.nextreaders; nxtrdrsl > 0 {
			nxtrdr = nextreaders[nxtrdrsl-1]
		} else {
			nxtrdr = rdr
		}
	}
	return
}

func (rdr *Reader) IsFirst() (first bool) {
	if rdr != nil {
		first = rdr.first
	}
	return
}

func (rdr *Reader) IsLast() (last bool) {
	if rdr != nil {
		last = rdr.last
	}
	return
}

func (rdr *Reader) CSVReader(a ...interface{}) (eofr *iorw.EOFCloseSeekReader) {
	if rdr != nil {
		pi, pw := io.Pipe()
		ctx, ctxcnl := context.WithCancel(context.Background())
		go func() {
			pwerr := error(nil)
			defer func() {
				if pwerr != nil {
					pw.CloseWithError(pwerr)
				} else {
					pw.Close()
				}
			}()
			ctxcnl()
			pwerr = rdr.ToCSV(pw, a...)
		}()
		<-ctx.Done()
		eofr = iorw.NewEOFCloseSeekReader(pi)
	}
	return
}

func (rdr *Reader) JSONReader(layout string, cols ...string) (eofr *iorw.EOFCloseSeekReader) {
	if rdr != nil {
		pi, pw := io.Pipe()
		ctx, ctxcnl := context.WithCancel(context.Background())
		go func() {
			pwerr := error(nil)
			defer func() {
				if pwerr != nil {
					pw.CloseWithError(pwerr)
				} else {
					pw.Close()
				}
			}()
			ctxcnl()
			pwerr = rdr.ToJSON(pw, layout, cols...)
		}()
		<-ctx.Done()
		eofr = iorw.NewEOFCloseSeekReader(pi)
	}
	return
}

func (rdr *Reader) ToJSON(w io.Writer, layout string, cols ...string) (err error) {
	if rdr != nil {
		buffenc := iorw.NewBuffer()
		enc := json.NewEncoder(buffenc)

		var wrtenc = func(v any) (encerr error) {
			if encerr = enc.Encode(v); encerr != nil {
				return
			}
			_, encerr = w.Write([]byte(strings.TrimFunc(buffenc.String(), iorw.IsSpace)))
			buffenc.Clear()
			return
		}
		if cols := rdr.Columns(cols...); len(cols) > 0 {
			if strings.EqualFold(layout, "datatables") {
				if _, err = w.Write([]byte("{")); err != nil {
					return
				}
				if err = iorw.Fprint(w, `"colummns":`); err != nil {
					return
				}
				if _, err = w.Write([]byte("[")); err != nil {
					return
				}
				for cn, c := range cols {
					if _, err = w.Write([]byte("{")); err != nil {
						return
					}
					if err = iorw.Fprint(w, `"title":`); err != nil {
						return
					}
					if err = wrtenc(c); err != nil {
						return
					}
					if _, err = w.Write([]byte("}")); err != nil {
						return
					}
					if cn < len(cols)-1 {
						if _, err = w.Write([]byte(",")); err != nil {
							return
						}
					}
				}
				if _, err = w.Write([]byte("]")); err != nil {
					return
				}
				if err = iorw.Fprint(w, `,"data":`); err != nil {
					return
				}
				if _, err = w.Write([]byte("[")); err != nil {
					return
				}
				for nxt, nxterr := rdr.Next(); nxt && nxterr == nil; nxt, nxterr = rdr.Next() {
					if err = wrtenc(rdr.Data(cols...)); err != nil {
						return
					}
					if !rdr.IsFirst() {
						if _, err = w.Write([]byte(",")); err != nil {
							return
						}
					}
				}
				if _, err = w.Write([]byte("]")); err != nil {
					return
				}
				if _, err = w.Write([]byte("}")); err != nil {
					return
				}
			} else if strings.EqualFold(layout, "dataset") {
				if _, err = w.Write([]byte("{")); err != nil {
					return
				}
				if err = iorw.Fprint(w, `"colummns":`); err != nil {
					return
				}
				if _, err = w.Write([]byte("[")); err != nil {
					return
				}
				for cn, c := range cols {
					if err = wrtenc(c); err != nil {
						return
					}
					if cn < len(cols)-1 {
						if _, err = w.Write([]byte(",")); err != nil {
							return
						}
					}
				}
				if _, err = w.Write([]byte("]")); err != nil {
					return
				}
				if err = iorw.Fprint(w, `,"data":`); err != nil {
					return
				}
				if _, err = w.Write([]byte("[")); err != nil {
					return
				}
				for nxt, nxterr := rdr.Next(); nxt && nxterr == nil; nxt, nxterr = rdr.Next() {
					if err = wrtenc(rdr.Data(cols...)); err != nil {
						return
					}
					if !rdr.IsFirst() {
						if _, err = w.Write([]byte(",")); err != nil {
							return
						}
					}
				}
				if _, err = w.Write([]byte("]")); err != nil {
					return
				}
				if _, err = w.Write([]byte("}")); err != nil {
					return
				}
			} else if strings.EqualFold(layout, "array") {
				if _, err = w.Write([]byte("[")); err != nil {
					return
				}
				if err = wrtenc(cols); err != nil {
					return
				}
				if err = iorw.Fprint(w, `,`); err != nil {
					return
				}
				for nxt, nxterr := rdr.Next(); nxt && nxterr == nil; nxt, nxterr = rdr.Next() {
					if err = wrtenc(rdr.Data(cols...)); err != nil {
						return
					}
					if !rdr.IsFirst() {
						if _, err = w.Write([]byte(",")); err != nil {
							return
						}
					}
				}
				if _, err = w.Write([]byte("]")); err != nil {
					return
				}
			} else {
				if _, err = w.Write([]byte("[")); err != nil {
					return
				}
				for nxt, nxterr := rdr.Next(); nxt && nxterr == nil; nxt, nxterr = rdr.Next() {
					if data := rdr.Data(cols...); len(data) == len(cols) {
						if _, err = w.Write([]byte("{")); err != nil {
							return
						}
						for cn, c := range cols {

							if err = wrtenc(c); err != nil {
								return
							}
							if _, err = w.Write([]byte(":")); err != nil {
								return
							}
							if err = wrtenc(data[cn]); err != nil {
								return
							}

							if cn < len(cols)-1 {
								if _, err = w.Write([]byte(",")); err != nil {
									return
								}
							}
						}
						if _, err = w.Write([]byte("}")); err != nil {
							return
						}
					}
					if !rdr.IsFirst() {
						if _, err = w.Write([]byte(",")); err != nil {
							return
						}
					}
				}
				if _, err = w.Write([]byte("]")); err != nil {
					return
				}
			}
		}
	}
	return
}

func (rdr *Reader) ExecEachPrepaired(execerr func(*Executor, int, error), execs ...*Executor) (err error) {
	if rdr != nil && len(execs) > 0 && execerr != nil {
		for execn, exec := range execs {
			if stmnt := exec.stmnt; stmnt != nil {
				if stmnt.rdr == nil {
					stmnt.rdr = rdr
				}
			}
			execerr(exec, execn, exec.Exec())
		}
	}
	return
}

func (rdr *Reader) ForEachColumnType(eachitem func(*ColumnType, int, bool, bool), cols ...string) (err error) {
	if rdr != nil && eachitem != nil {
		if len(cols) == 1 && strings.Contains(cols[0], ",") {
			cols = strings.Split(cols[0], ",")
		}
		if clstpes := rdr.ColumnTypes(cols...); len(clstpes) > 0 {
			clstpesl := len(clstpes)
			for clstpen, clstpe := range clstpes {
				eachitem(clstpe, clstpen, clstpen == 0, clstpen == clstpesl-1)
			}
		}
	}
	return
}

func (rdr *Reader) ForEachColumn(eachitem func(string, int, bool, bool), cols ...string) (err error) {
	if rdr != nil && eachitem != nil {
		if len(cols) == 1 && strings.Contains(cols[0], ",") {
			cols = strings.Split(cols[0], ",")
		}
		if cls := rdr.Columns(cols...); len(cls) > 0 {
			clsl := len(cls)
			for cln, cl := range cls {
				eachitem(cl, cln, cln == 0, cln == clsl-1)
			}
		}
	}
	return
}

func (rdr *Reader) ForEach(eachitem func(*Reader, bool, bool) bool) (err error) {
	if rdr != nil && eachitem != nil {
		var eachdone = false
		var nxt = false
		var prvEventRow = rdr.EventRow
		rdr.EventRow = func(r *Reader) (evterr error) {
			eachdone = eachitem(rdr, rdr.first, rdr.last)
			return
		}
		for err == nil {
			if nxt, err = rdr.Next(); nxt && err == nil && !eachdone {
				continue
			} else if !nxt || eachdone {
				break
			}
		}
		rdr.EventRow = prvEventRow
		eachitem = nil
	}
	return
}

func (rdr *Reader) ForEachData(eachitem func([]interface{}, int64, bool, bool) bool, cols ...string) (err error) {
	if rdr != nil && eachitem != nil {
		if cls := rdr.Columns(cols...); len(cls) > 0 {
			var eachdone = false
			var nxt = false
			var prvEventRow = rdr.EventRow
			rdr.EventRow = func(r *Reader) (evterr error) {
				eachdone = eachitem(rdr.Data(cls...), rdr.RowNr, rdr.first, rdr.last)
				return
			}
			for err == nil {
				if nxt, err = rdr.Next(); nxt && err == nil && !eachdone {
					continue
				} else if !nxt || eachdone {
					break
				}
			}
			rdr.EventRow = prvEventRow
			cls = nil
		}
		eachitem = nil
		cols = nil
	}
	return
}

func (rdr *Reader) ForEachDataMap(eachitem func(map[string]interface{}, int64, bool, bool) bool, cols ...string) (err error) {
	if rdr != nil && eachitem != nil {
		if cls := rdr.Columns(cols...); len(cls) > 0 {
			var eachdone = false
			var nxt = false
			var prvEventRow = rdr.EventRow
			rdr.EventRow = func(r *Reader) (evterr error) {
				eachdone = eachitem(rdr.DataMap(cls...), rdr.RowNr, rdr.first, rdr.last)
				return
			}
			for err == nil {
				if nxt, err = rdr.Next(); nxt && err == nil && !eachdone {
					continue
				} else if !nxt || eachdone {
					break
				}
			}
			rdr.EventRow = prvEventRow
			cls = nil
		}
	}
	return
}

func (rdr *Reader) ForEachDataMapSet(eachitem func([]map[string]interface{}, int64, bool, bool) bool, cols ...string) (err error) {
	if rdr != nil && eachitem != nil {
		if cls := rdr.Columns(cols...); len(cls) > 0 {
			var eachdone = false
			var nxt = false
			var prvEventRow = rdr.EventRow
			rdr.EventRow = func(r *Reader) (evterr error) {
				eachdone = eachitem(rdr.DataSetMap(cls...), rdr.RowNr, rdr.first, rdr.last)
				return
			}
			for err == nil {
				if nxt, err = rdr.Next(); nxt && err == nil && !eachdone {
					continue
				} else if !nxt || eachdone {
					break
				}
			}
			rdr.EventRow = prvEventRow
			cls = nil
		}
	}
	return
}

func (rdr *Reader) ToCSV(w io.Writer, a ...interface{}) (err error) {
	if rdr != nil {
		var incldhdrs = true
		var colsep = ","
		var txtpar = "\""
		var rowsep = "\r\n"
		for _, d := range a {
			if mpd, _ := d.(map[string]interface{}); len(mpd) > 0 {
				for mk, mv := range mpd {
					if mvs, _ := mv.(string); mvs != "" {
						if strings.EqualFold("col-sep", mk) {
							colsep = mvs
						} else if strings.EqualFold("row-sep", mk) {
							rowsep = mvs
						} else if strings.EqualFold("headers", mk) {
							if strings.EqualFold(mvs, "true") || strings.EqualFold(mvs, "false") {
								incldhdrs = strings.EqualFold(mvs, "true")
							}
						}
					} else if mvb, mvbok := mv.(bool); mvbok {
						if strings.EqualFold("headers", mk) {
							incldhdrs = mvb
						}
					}
				}
			}
		}
		if err = rdr.Prep(); err == nil {
			var wbuf = iorw.NewBuffer()
			defer wbuf.Close()
			var bufens = iorw.NewBuffer()
			defer bufens.Close()
			colsepi := 0
			colsepL := len(txtpar)
			txtpari := 0
			txtparL := len(txtpar)

			var wrteval = func(v interface{}) (werr error) {
				if werr = iorw.Fprint(wbuf, fmt.Sprintf("%v", v)); werr == nil {
					var sv = strings.TrimSpace(wbuf.String())
					var svl = len(sv)
					var svn = 0
					hsColSep, hsNl, hsTxtPar := false, false, false
					colsepi = 0
					txtpari = 0
					for svn < svl {
						sr := rune(sv[svn])
						if !hsColSep && rune(colsep[colsepi]) == sr {
							colsepi++
							if colsepi == colsepL {
								hsColSep = true
								colsepi = 0
							}
						} else if !hsColSep && colsepi > 0 {
							colsepi = 0
						}
						if rune(txtpar[txtpari]) == sr {
							txtpari++
							if txtpari == txtparL {
								if !hsTxtPar {
									hsTxtPar = true
								}
								txtpari = 0
								sv = sv[:svn+1] + txtpar + sv[svn+1:]
								svl += txtparL
								svn += txtparL
								svn++
								continue
							}
						}
						if !hsNl && sr == '\n' {
							hsNl = true
						}
						svn++
					}
					if hsColSep || hsNl || hsTxtPar {
						if hsColSep || hsNl || hsTxtPar {
							bufens.Print(txtpar, sv, txtpar)
						} else {
							bufens.Print(sv)
						}
					} else {
						wbuf.WriteTo(bufens)
					}
					wbuf.Clear()
				}
				return
			}
			cls := rdr.Columns()
			if clsl := len(cls); clsl > 0 {
				if incldhdrs {
					for cn, c := range cls {
						if err = iorw.Fprint(bufens, txtpar); err == nil {
							if err = wrteval(c); err != nil {
								rdr.Close()
								return
							}
							if err = iorw.Fprint(bufens, txtpar); err == nil {
								if cn < clsl-1 {
									if err = iorw.Fprint(bufens, colsep); err != nil {
										rdr.Close()
										return
									}
								}
							} else {
								rdr.Close()
								return
							}
						} else {
							rdr.Close()
							return
						}
					}
					if err = iorw.Fprint(bufens, rowsep); err != nil {
						rdr.Close()
						return
					}
				}

				for {
					if nxt, nxterr := rdr.Next(); nxterr == nil && nxt {
						data := rdr.Data()
						for dtan, dta := range data {
							if err = wrteval(dta); err == nil {
								if dtan < clsl-1 {
									if err = iorw.Fprint(bufens, colsep); err != nil {
										rdr.Close()
										return
									}
								} else if dtan == clsl-1 {
									break
								}
							} else {
								rdr.Close()
								return
							}
						}
						if err = iorw.Fprint(bufens, rowsep); err == nil {
							if bufens.Size() >= 1024*1204 {
								if _, err = bufens.WriteTo(w); err != nil {
									rdr.Close()
									return
								}
								bufens.Clear()
							}
						} else {
							rdr.Close()
							return
						}
					} else if nxterr != nil {
						err = nxterr
						rdr.Close()
						return
					} else {
						break
					}
				}
			}
			if _, err = bufens.WriteTo(w); err != nil {
				rdr.Close()
			}
		}
	}
	return
}

func (rdr *Reader) Prep() (err error) {
	if rdr != nil {
		if rows := rdr.rows; rows == nil {
			if rdr.stmnt != nil {
				if cn := rdr.stmnt.cn; cn != nil {
					if cn.isRemote() {

					} else if rdr.stmnt.prepstmnt != nil {
						if rows, err = rdr.stmnt.Query(); err == nil {
							if _, err = rows.Columns(); err == nil {
								rdr.rows = rows
							} else {
								rdr.Close()
							}
						} else {
							if rdr.EventError != nil {
								rdr.EventError(err)
							}
							rdr.Close()
						}
					}
				}
			} else if rdr.strmrdr != nil {
				if datardr := newDataReader("csv", rdr.strmsttngs, rdr.strmrdr); datardr != nil {
					if rows := newSqlRows(nil, datardr, nil); rows != nil {
						if _, err = rows.Columns(); err == nil {
							rdr.rows = rows
						} else {
							rdr.Close()
						}
					} else {
						if rdr.EventError != nil {
							rdr.EventError(err)
						}
						rdr.Close()
					}
				}
			} else if eventprepcols, eventprepdata := rdr.EventPrepColumns, rdr.EventPrepData; eventprepcols != nil && eventprepdata != nil {
				if strmrdr := newStreamReader(rdr, eventprepcols, eventprepdata); strmrdr != nil {
					if rows := newSqlRows(nil, nil, strmrdr); rows != nil {
						if _, err = rows.Columns(); err == nil {
							rdr.rows = rows
						} else {
							rdr.Close()
						}
					} else {
						if rdr.EventError != nil {
							rdr.EventError(err)
						}
						rdr.Close()
					}
				}
			}
		}
	}
	return
}

func (rdr *Reader) Columns(cols ...string) (cls []string) {
	if rdr != nil {
		if rows := rdr.rows; rows != nil {
			cls, _ = rows.Columns(cols...)
		}
	}
	return
}

func (rdr *Reader) ColumnTypes(cols ...string) (clstpes []*ColumnType) {
	if rdr != nil {
		if rows := rdr.rows; rows != nil {
			clstpes, _ = rows.ColumnTypes(cols...)
		}
	}
	return
}

var emptymap = map[string]interface{}{}
var emptysetmap = []map[string]interface{}{}

// DataMap return Displayable data in the form of a map[string]interface{} column and values
func (rdr *Reader) DataMap(cols ...string) (datamap map[string]interface{}) {
	if rdr != nil {
		if len(cols) == 1 && strings.Contains(cols[0], ",") {
			cols = strings.Split(cols[0], ",")
		}
		if cls, data := rdr.Columns(cols...), rdr.Data(cols...); len(cls) > 0 && len(data) == len(cls) {
			if datamap == nil {
				datamap = map[string]interface{}{}
			}
			for cn := range cls {
				datamap[cls[cn]] = data[cn]
			}
			return datamap
		}
	}
	return emptymap
}

func (rdr *Reader) DataSetMap(cols ...string) (datasetmap []map[string]interface{}) {
	if rdr != nil {
		if len(cols) == 1 && strings.Contains(cols[0], ",") {
			cols = strings.Split(cols[0], ",")
		}
		if cls, data := rdr.Columns(cols...), rdr.Data(cols...); len(cls) > 0 && len(data) == len(cls) {
			datasetmap = make([]map[string]interface{}, len(cls))
			for cn, c := range cls {
				datasetmap[cn] = map[string]interface{}{c: data[cn]}
			}
			return datasetmap
		}
	}
	return emptysetmap
}

func (rdr *Reader) Data(cols ...string) (dspdata []interface{}) {
	if rdr != nil {
		if rows := rdr.rows; rows != nil {
			dspdata = rows.DisplayData(cols...)
		}
	}
	return
}

func (rdr *Reader) Field(name string) (val interface{}) {
	if rdr != nil && name != "" {
		if rows := rdr.rows; rows != nil {
			val = rows.Field(name)
		}
	}
	return
}

func cleanupStringData(str string) string {
	if cleanedrns := []rune(strings.TrimFunc(str, iorw.IsSpace)); len(cleanedrns) > 0 {
		for rn, r := range cleanedrns {
			if r == 2 || r == 3 {
				cleanedrns[rn] = ' '
			} else {
				cleanedrns[rn] = r
			}
		}
		return string(cleanedrns)
	} else {
		return ""
	}
}

func castSQLTypeValue(valToCast interface{}, colType *ColumnType) (castedVal interface{}) {
	if valToCast != nil {
		if d, dok := valToCast.([]uint8); dok {
			castedVal = cleanupStringData(string(d))
		} else if sd, dok := valToCast.(string); dok {
			castedVal = cleanupStringData(sd)
		} else if dtime, dok := valToCast.(time.Time); dok {
			castedVal = dtime.Format("2006-01-02T15:04:05")
		} else if djsn, djsnok := valToCast.([]byte); djsnok {
			if dv, dverr := json.Marshal(djsn); dverr == nil {
				castedVal = dv
			} else {
				castedVal = djsn
			}
		} else {
			castedVal = valToCast
		}
	} else {
		if valToCast == nil {
			if strings.Contains(strings.ToLower(colType.databaseType), "char") {
				castedVal = ""
			} else if strings.Contains(strings.ToLower(colType.databaseType), "int") {
				castedVal = ""
			} else {
				castedVal = valToCast
			}
		} else {
			castedVal = valToCast
		}
	}
	return castedVal
}

func processReaderExecutors(rdr *Reader) {
	for _, exctr := range rdr.exctrs {
		if exctrerr := exctr.Exec(); exctrerr != nil {
			exctr.Close()
		}
	}
}

func (rdr *Reader) Next() (next bool, err error) {
	if rdr != nil {
		defer func() {
			if err != nil {
				rdr.Close()
			}
		}()

		rows := rdr.rows
		if rows == nil {
			if err = rdr.Prep(); err != nil {
				return
			}
			if rows = rdr.rows; rows == nil {
				return
			}
		}

		if stmnt := rdr.stmnt; stmnt != nil && rdr.stmnt.isRemote {

		} else if rows != nil {
			if stmnt := rdr.stmnt; stmnt != nil {
				if ctx := stmnt.ctx; rdr.rows != nil {
					select {
					case <-ctx.Done():
						if err = ctx.Err(); err != nil {
							rdr.Close()
							return
						}
					default:
					}
				}
			}
			ctx := rows.Context()
			for err == nil {
				if ctx != nil {
					select {
					case <-ctx.Done():
						if err = ctx.Err(); err != nil {
							next = false
							rdr.callFinalise()
							break
						}
					default:
					}
				}
				selected := false
				if !rdr.started {
					rdr.started = true
					rdr.RowNr = 0
					if next, err = rows.Next(), rows.Err(); err == nil && next {
						if err = rows.Scan(rdr.CastTypeValue); err == nil {
							rdr.RowNr++
							rdr.first = true
							if next, err = rdr.EventNext(rdr); next && err == nil {
								if selected, err = rdr.EventSelect(rdr); err == nil {
									if next, err = rows.Next(), rows.Err(); !next && err == nil {
										rdr.last = true
										next = true
									} else if err == nil {
										rdr.last = false
										next = true
									}
									if next && err == nil && selected {
										if err = rdr.EventRow(rdr); err != nil {
											next = false
										}
										break
									}
								} else if err == nil {
									rdr.callFinalise()
									break
								}
							} else if next && err != nil {
								next = false
								break
							}
						} else {
							next = false
						}
					} else {
						next = false
						break
					}
				} else if !rdr.last {
					if err = rows.Scan(rdr.CastTypeValue); err == nil {
						rdr.RowNr++
						if next, err = rdr.EventNext(rdr); next && err == nil {
							rdr.first = false
							if selected, err = rdr.EventSelect(rdr); err == nil {
								if next, err = rows.Next(), rows.Err(); !next && err == nil {
									rdr.last = true
									if next = true; err == nil {
										if selected {
											if err = rdr.EventRow(rdr); err != nil {
												next = false
											}
										}
										break
									}
								} else {
									rdr.last = false
									if next && err == nil && selected {
										if err = rdr.EventRow(rdr); err != nil {
											next = false
										}
										break
									}
								}
							} else if err == nil {
								rdr.callFinalise()
								break
							}
						} else if next && err != nil {
							next = false
						}
					} else {
						next = false
					}
				} else if rdr.last {
					rdr.callFinalise()
					break
				}
			}
		}
	}
	return
}

func (rdr *Reader) Close() (err error) {
	if rdr != nil {
		if sqlrws := rdr.rows; sqlrws != nil {
			rdr.rows = nil
			err = sqlrws.Close()
		}
		if stmnt := rdr.stmnt; stmnt != nil {
			rdr.stmnt = nil
			stmnt.Close()
		}
		if rdr.strmrdr != nil {
			rdr.strmrdr.Close()
			rdr.strmrdr = nil
		}
		if rdr.EventInit != nil {
			rdr.EventInit = nil
		}
		if rdr.EventSelect != nil {
			rdr.EventSelect = nil
		}
		if rdr.EventRow != nil {
			rdr.EventRow = nil
		}
		if rdr.EventRowError != nil {
			rdr.EventRowError = nil
		}
		if rdr.EventError != nil {
			rdr.EventRowError = nil
		}
		rdr.callFinalise()
		if close := rdr.EventClose; close != nil {
			rdr.EventClose = nil
			close(rdr)
		}
		if exctrs := rdr.exctrs; exctrs != nil {
			for exectr := range exctrs {
				exectr.Close()
			}
			rdr.exctrs = nil
		}
		if rdr.orderedexctrs != nil {
			rdr.orderedexctrs = nil
		}
		rdr = nil
	}
	return
}

// ColumnType structure defining column definition
type ColumnType struct {
	name              string
	hasNullable       bool
	hasLength         bool
	hasPrecisionScale bool
	nullable          bool
	length            int64
	databaseType      string
	precision         int64
	scale             int64
	scanType          reflect.Type
}

// Name ColumnType.Name()
func (colType *ColumnType) Name() string {
	return colType.name
}

// Numeric ColumnType is Numeric() bool
func (colType *ColumnType) Numeric() bool {
	if colType.hasPrecisionScale {
		return true
	}
	return !strings.Contains(colType.databaseType, "CHAR") && !strings.Contains(colType.databaseType, "DATE") && !strings.Contains(colType.databaseType, "TIME")
}

// HasNullable ColumnType content has NULL able content
func (colType *ColumnType) HasNullable() bool {
	return colType.hasNullable
}

// HasLength ColumnType content has Length definition
func (colType *ColumnType) HasLength() bool {
	return colType.hasLength
}

// HasPrecisionScale ColumnType content has PrecisionScale
func (colType *ColumnType) HasPrecisionScale() bool {
	return colType.hasPrecisionScale
}

// Nullable ColumnType content is Nullable
func (colType *ColumnType) Nullable() bool {
	return colType.nullable
}

// Length ColumnType content lenth must be used in conjunction with HasLength
func (colType *ColumnType) Length() int64 {
	return colType.length
}

// DatabaseType ColumnType underlying db type as defined by driver of Connection
func (colType *ColumnType) DatabaseType() string {
	return colType.databaseType
}

// Precision ColumnType numeric Precision. Used in conjunction with HasPrecisionScale
func (colType *ColumnType) Precision() int64 {
	return colType.precision
}

// Scale ColumnType Scale. Used in conjunction with HasPrecisionScale
func (colType *ColumnType) Scale() int64 {
	return colType.scale
}

// Type ColumnType reflect.Type as specified by golang sql/database
func (colType *ColumnType) Type() reflect.Type {
	return colType.scanType
}
