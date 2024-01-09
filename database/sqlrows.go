package database

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"strings"

	"github.com/evocert/lnksnk/iorw"
)

type datareadertype int

const (
	dataRdrUnknown datareadertype = iota
	dataRdrJson
	dataRdrCsv
)

type DataReader struct {
	rdr        *iorw.RuneReaderSlice
	rdrtpe     datareadertype
	colsep     []rune
	colTxtr    rune
	hasheaders bool
	rowsep     []rune
	cls        []string
	clstpes    []*ColumnType
	data       []interface{}
}

func (datardr *DataReader) Close() (err error) {
	if datardr != nil {
		if rdrtpe := datardr.rdrtpe; rdrtpe != dataRdrUnknown {
			datardr.rdrtpe = dataRdrUnknown
			if rdrtpe == dataRdrCsv {

				datardr.colsep = nil
				datardr.colTxtr = 0
				datardr.rowsep = nil
				//datardr.prerdr = nil
			}

			if datardr.cls != nil {
				datardr.cls = nil
			}
			if datardr.clstpes != nil {
				datardr.clstpes = nil
			}
			if datardr.data != nil {
				datardr.data = nil
			}
			if datardr.data != nil {
				datardr.data = nil
			}
			datardr.hasheaders = false
		}
	}
	return
}

func (datardr *DataReader) Next() (next bool, err error) {
	if datardr != nil {
		if rdr := datardr.rdr; rdr != nil {
			if datardr.rdrtpe == dataRdrCsv {
				if len(datardr.cls) == 0 {
					if _, err = readCsv(datardr, rdr); err != nil {
						return
					}
				}
				if next, err = readCsv(datardr, rdr); err == nil {
					next = !next
				}
			}
		}
	}
	return
}

func (datardr *DataReader) Columns() (cols []string, err error) {
	if datardr != nil {
		if cls, rdr, rdrtpe := datardr.cls, datardr.rdr, datardr.rdrtpe; rdr != nil {
			if len(cls) == 0 {
				if rdrtpe == dataRdrCsv {
					if _, err = readCsv(datardr, rdr); err == nil {
						cols = datardr.cls[:]
					}
				}
			} else {
				cols = datardr.cls[:]
			}
		}
	}
	return
}

func readCsv(datardr *DataReader, rdr io.RuneReader) (done bool, err error) {
	var csvTxtr = rune(0)
	var colsep = datardr.colsep
	var clspl = len(colsep)
	var rowsep = datardr.rowsep
	var hasHdrs = datardr.hasheaders

	var prvr = rune(0)
	rwspl := len(rowsep)
	var rwspi = 0
	if rwspl == 0 {
		rowsep = append(rowsep, '\n')
		rwspl = 1
	}

	if clspl == 0 {
		var psblclseps = map[rune]int{}
		tmpbuf := iorw.NewBuffer()
		for {
			if r, rs, rerr := rdr.ReadRune(); rerr == nil || rerr == io.EOF {
				if rs > 0 {
					if r == 2 {
						r = 32
					}
					tmpbuf.WriteRune(r)
					if csvTxtr == 0 && (r == '"') {
						csvTxtr = r
						prvr = 0
						continue
					} else if csvTxtr > 0 && csvTxtr == r {
						if prvr != csvTxtr {
							csvTxtr = 0
							prvr = 0
							continue
						}
					}
					if csvTxtr == 0 && rwspi < rwspl {
						if rowsep[rwspi] == r {
							rwspi++
							if rwspi == rwspl {
								if rwspl == 1 && prvr == '\r' {
									rowsep = append([]rune{prvr}, rowsep...)
								}
								datardr.rowsep = rowsep[:]
								lstcnt := 0
								lstspr := rune(0)
								for psblr, cnt := range psblclseps {
									if lstcnt < cnt {
										lstspr = psblr
									}
								}
								if lstspr > 0 {
									colsep = []rune{lstspr}
									datardr.colsep = colsep[:]
								}
								if hasHdrs {
									datardr.rdr.PreAppend(tmpbuf.Clone(true).Reader(true))
									done, err = readCsv(datardr, datardr.rdr)
								} else if lstcnt > 0 {
									//datardr.prerdr = tmpbuf.Clone(true).Reader(true)
									datardr.rdr.PreAppend(tmpbuf.Clone(true).Reader(true))
									datardr.cls = make([]string, lstcnt+1)
									datardr.clstpes = make([]*ColumnType, lstcnt+1)
									datardr.data = make([]interface{}, lstcnt+1)
									for lstcnt > 0 {
										colnme := fmt.Sprintf("column%d", lstcnt)
										datardr.cls[lstcnt-1] = colnme
										cltp := &ColumnType{}
										cltp.databaseType = "VARCHAR"
										cltp.hasLength = true
										cltp.hasNullable = true
										cltp.hasPrecisionScale = false
										cltp.name = colnme
										datardr.clstpes[lstcnt-1] = cltp
										lstcnt--
									}
								}
								break
							}
						} else if rwspi > 0 {
							err = fmt.Errorf("%s", "Invalid EOF")
							break
						} else if csvTxtr == 0 && ((32 < r && r <= 126) || r == 9) {
							if (65 <= r && r <= 122) || (46 <= r && r <= 58) || (33 <= r && r <= 43) {
								continue
							}
							psblclseps[r] += 1
							prvr = r
						} else {
							prvr = r
						}
					}
				}
				if rerr != nil {
					break
				}
			} else if rerr != nil {
				if rerr != io.EOF {
					err = rerr
				}
				break
			}
		}
	} else {

		var clspi = 0
		var dataval = []rune{}
		var datai = 0
		var datal = len(datardr.data)
		var appndcols = len(datardr.cls) == 0 && hasHdrs

		var toDataVal = func(dtar ...rune) (str string) {
			for n, r := range dtar {
				if !iorw.IsSpace(r) {
					mxn := len(dtar)
					for mxn >= n {
						if !iorw.IsSpace(dtar[mxn-1]) {
							str = string(dtar[n:mxn])
							return
						}
						mxn--
					}
					break
				}
			}
			return
		}
		for {
			if r, rs, rerr := rdr.ReadRune(); rerr == nil || rerr == io.EOF {
				if rs > 0 {
					if r == 2 {
						r = 32
					}
					if csvTxtr == 0 && (r == '"') {
						csvTxtr = r
						prvr = 0
						dataval = nil
						continue
					} else if csvTxtr > 0 && csvTxtr == r {
						if prvr != csvTxtr {
							csvTxtr = 0
							prvr = 0
							continue
						}
						dataval = append(dataval, r)
						continue
					}
					if csvTxtr == 0 && rwspi < rwspl {
						if rowsep[rwspi] == r {
							rwspi++
							if rwspi == rwspl {
								if appndcols {
									datardr.cls = append(datardr.cls, toDataVal(dataval...))
									datal = len(datardr.cls)
									datardr.clstpes = make([]*ColumnType, datal)

									for cn, cl := range datardr.cls {
										cltp := &ColumnType{}
										cltp.databaseType = "VARCHAR"
										cltp.hasLength = true
										cltp.hasNullable = true
										cltp.hasPrecisionScale = false
										cltp.name = cl
										datardr.clstpes[cn] = cltp
									}
									datardr.data = make([]interface{}, datal)
									appndcols = false
								} else {
									if datai == datal-1 {
										datardr.data[datai] = toDataVal(dataval...)
										datai++
										datal = 0

									} else if datai < datal {
										datardr.data[datai] = toDataVal(dataval...)
										datai++
										datal = 0
									}
								}
								dataval = nil
								clspi = 0
								prvr = 0
								break
							}
						} else if rwspi > 0 {
							err = fmt.Errorf("%s", "Invalid EOF")
							break
						} else if csvTxtr == 0 && clspi < clspl {
							if colsep[clspi] == r {
								clspi++
								if clspi == clspl {
									if appndcols {
										datardr.cls = append(datardr.cls, toDataVal(dataval...))
									} else {
										if datai < datal {
											datardr.data[datai] = toDataVal(dataval...)
											datai++
										}
									}
									dataval = nil
									clspi = 0
									prvr = 0
								} else {
									prvr = r
								}
							} else {
								if clspi > 0 {
									dataval = append(dataval, colsep[:clspi]...)
									clspi = 0
								}
								dataval = append(dataval, r)
								prvr = r
							}
						} else {
							dataval = append(dataval, r)
							prvr = r
						}
					}
				}
				if rerr != nil {
					if done = rerr == io.EOF; !done {
						err = rerr
					} else {
						if appndcols {
							datardr.cls = append(datardr.cls, toDataVal(dataval...))
							datal = len(datardr.cls)
							datardr.clstpes = make([]*ColumnType, datal)

							for cn, cl := range datardr.cls {
								cltp := &ColumnType{}
								cltp.databaseType = "VARCHAR"
								cltp.hasLength = true
								cltp.hasNullable = true
								cltp.hasPrecisionScale = false
								cltp.name = cl
								datardr.clstpes[cn] = cltp
							}
							datardr.data = make([]interface{}, datal)
							appndcols = false
						} else {
							if datai == datal-1 {
								datardr.data[datai] = toDataVal(dataval...)
								datai++
							}
						}
					}
					break
				}
			} else if rerr != nil {
				if done = rerr == io.EOF; !done {
					err = rerr
				} else {
					if appndcols {
						datardr.cls = append(datardr.cls, toDataVal(dataval...))
						datal = len(datardr.cls)
						datardr.clstpes = make([]*ColumnType, datal)

						for cn, cl := range datardr.cls {
							cltp := &ColumnType{}
							cltp.databaseType = "VARCHAR"
							cltp.hasLength = true
							cltp.hasNullable = true
							cltp.hasPrecisionScale = false
							cltp.name = cl
							datardr.clstpes[cn] = cltp
						}
						datardr.data = make([]interface{}, datal)
						appndcols = false
					} else {
						if datai == datal-1 {
							datardr.data[datai] = toDataVal(dataval...)
							datai++
						}
					}
				}
				break
			}
		}
	}
	return
}

func newDataReader(rdrtype string, sttngs map[string]interface{}, rdr *iorw.EOFCloseSeekReader) (datardr *DataReader) {
	datardr = &DataReader{rdrtpe: func() datareadertype {
		if strings.EqualFold(rdrtype, "csv") {
			return dataRdrCsv
		} else if strings.EqualFold(rdrtype, "json") {
			return dataRdrJson
		}
		return dataRdrUnknown
	}(), rdr: iorw.NewRuneReaderSlice(rdr), hasheaders: true}
	return
}

type SqlRows struct {
	ctx         context.Context
	ctxcnl      context.CancelCauseFunc
	rows        *sql.Rows
	datardr     *DataReader
	strmrdr     *StreamReader
	cls         []string
	clstpes     []*ColumnType
	lsterr      error
	dataref     []interface{}
	displaydata []interface{}
	data        []interface{}
	clsimap     map[string]int
}

type StreamReader struct {
	cls           []string
	clstpes       []*ColumnType
	data          []interface{}
	eventprepcols func()
	eventprepdata func()
}

func newStreamReader(rdr *Reader, prepCols PrepColumnsFunc, prepData PrepDataFunc) (streamrdr *StreamReader) {
	if rdr == nil || prepCols == nil || prepData == nil {
		return
	}
	streamrdr = &StreamReader{}
	streamrdr.eventprepcols = func() {
		prepCols(rdr, streamrdr.PrepColumns)
	}
	streamrdr.eventprepdata = func() {
		prepData(rdr, streamrdr.PrepData)
	}
	return
}

func (strmrdr *StreamReader) Next() (next bool, err error) {
	if strmrdr != nil {
		if eventprepcols, eventprepdata := strmrdr.eventprepcols, strmrdr.eventprepdata; eventprepdata != nil && eventprepcols != nil {
			clsl := len(strmrdr.cls)
			if clsl == 0 {
				eventprepcols()
				clsl = len(strmrdr.cls)
			}
			if clsl > 0 {
				eventprepdata()
				dtal := len(strmrdr.data)
				if next = dtal > 0; next {

				}
			}
		}
	}
	return
}

func (strmrdr *StreamReader) Columns() (cols []string, err error) {
	if strmrdr != nil {
		if cls, eventprepcols, eventprepdata := strmrdr.cls, strmrdr.eventprepcols, strmrdr.eventprepdata; eventprepcols != nil || eventprepdata != nil {
			if len(cls) == 0 {
				eventprepcols()
				cols = strmrdr.cls[:]
			} else {
				cols = strmrdr.cls[:]
			}
		}
	}
	return
}

func (strmrdr *StreamReader) PrepColumns(cols ...string) {
	if strmrdr != nil {
		if colsl := len(cols); colsl > 0 {
			if colsl == 1 && strings.Contains(cols[0], ",") {
				cols = strings.Split(cols[0], ",")
				colsl = len(cols)
			}
			for coli := 0; coli < colsl; {
				if cols[coli] = strings.TrimFunc(cols[coli], iorw.IsSpace); cols[coli] == "" {
					cols = append(cols[:coli], cols[coli+1:]...)
					colsl--
					continue
				}
				coli++
			}
			if colsl > 0 {
				if clsl := len(strmrdr.cls); clsl == colsl {
					for cn, col := range cols {
						if strmrdr.cls[cn] != col {
							strmrdr.cls[cn] = col
						}
					}
				} else {
					strmrdr.cls = make([]string, colsl)
					copy(strmrdr.cls, cols)
				}
			}
		}
	}
}

func (strmrdr *StreamReader) PrepData(data ...interface{}) {
	if strmrdr != nil {
		if clsl := len(strmrdr.cls); clsl > 0 && len(data) == clsl {
			if dtal := len(strmrdr.data); dtal != clsl {
				strmrdr.data = make([]interface{}, clsl)
				copy(strmrdr.data, data)
			} else {
				copy(strmrdr.data, data)
			}
		}
	}
}

func newSqlRows(rows *sql.Rows, datardr *DataReader, strmrdr *StreamReader) (sqlrws *SqlRows) {
	ctx, ctxcnlcause := context.WithCancelCause(context.TODO())
	sqlrws = &SqlRows{rows: rows, datardr: datardr, strmrdr: strmrdr, ctx: ctx, ctxcnl: ctxcnlcause}
	return
}

func (sqlrws *SqlRows) Context() (ctx context.Context) {
	if sqlrws != nil {
		ctx = sqlrws.ctx
	}
	return
}

func (sqlrws *SqlRows) Data(cols ...string) (data []interface{}) {
	if sqlrws != nil {
		if colsl := len(cols); colsl > 0 {
			if colsl == 1 && strings.Contains(cols[0], ",") {
				cols = strings.Split(cols[0], ",")
			}
			cls := sqlrws.cls
			var mstcdcols = []int{}
			var ccls = make([]string, len(cls))
			copy(ccls, cls)
			var clsignr = map[string]bool{}
			for _, cl := range cols {
				if cl == "" {
					continue
				}
				if ci, clok := sqlrws.clsimap[cl]; clok {
					mstcdcols = append(mstcdcols, ci)
					clsignr[cl] = true
					continue
				}
				for cn, c := range ccls {
					for cig, cgb := range clsignr {
						if strings.EqualFold(c, cig) && cgb {
							delete(clsignr, cig)
							ccls = append(ccls[:cn], ccls[cn+1:]...)
							break
						}
					}
					if len(clsignr) == 0 {
						break
					}
				}
				for cn, c := range ccls {
					if strings.EqualFold(c, cl) {
						mstcdcols = append(mstcdcols, sqlrws.clsimap[c])
						ccls = append(ccls[:cn], ccls[cn+1:]...)
						break
					}
				}
			}

			if mstcdcolsl := len(mstcdcols); mstcdcolsl > 0 {
				data = make([]interface{}, mstcdcolsl)
				for mstcdcolsn, mstcdcolsp := range mstcdcols {
					data[mstcdcolsn] = sqlrws.data[mstcdcolsp]
				}
			}
		} else {
			data = sqlrws.data[:]
		}
	}
	return
}

func (sqlrws *SqlRows) DisplayData(cols ...string) (displaydata []interface{}) {
	if sqlrws != nil {
		if colsl := len(cols); colsl > 0 {
			if colsl == 1 && strings.Contains(cols[0], ",") {
				cols = strings.Split(cols[0], ",")
			}
			cls := sqlrws.cls
			var mstcdcols = []int{}
			var ccls = make([]string, len(cls))
			copy(ccls, cls)
			var clsignr = map[string]bool{}
			for _, cl := range cols {
				if cl == "" {
					continue
				}
				if ci, clok := sqlrws.clsimap[cl]; clok {
					mstcdcols = append(mstcdcols, ci)
					clsignr[cl] = true
					continue
				}
				for cn, c := range ccls {
					for cig, cgb := range clsignr {
						if strings.EqualFold(c, cig) && cgb {
							delete(clsignr, cig)
							ccls = append(ccls[:cn], ccls[cn+1:]...)
							break
						}
					}
					if len(clsignr) == 0 {
						break
					}
				}
				for cn, c := range ccls {
					if strings.EqualFold(c, cl) {
						mstcdcols = append(mstcdcols, sqlrws.clsimap[c])
						ccls = append(ccls[:cn], ccls[cn+1:]...)
						break
					}
				}
			}

			if mstcdcolsl := len(mstcdcols); mstcdcolsl > 0 {
				displaydata = make([]interface{}, mstcdcolsl)
				for mstcdcolsn, mstcdcolsp := range mstcdcols {
					displaydata[mstcdcolsn] = sqlrws.displaydata[mstcdcolsp]
				}
			}
		} else {
			displaydata = sqlrws.displaydata[:]
		}
	}
	return
}

func (sqlrws *SqlRows) Field(name string) (val interface{}) {
	if sqlrws != nil {
		if name != "" && sqlrws.clsimap != nil {
			for c, ci := range sqlrws.clsimap {
				if strings.EqualFold(c, name) {
					val = sqlrws.FieldByIndex(ci)
					return
				}
			}
		}
	}
	return
}

func (sqlrws *SqlRows) FieldIndex(name string) (index int) {
	index = -1
	if sqlrws != nil && name != "" {
		if clsimap := sqlrws.clsimap; clsimap != nil {
			if cli, cliok := clsimap[name]; cliok {
				index = cli
			} else {
				for c, ci := range clsimap {
					if strings.EqualFold(name, c) {
						index = ci
						return
					}
				}
			}
		}
	}
	return
}

func (sqlrws *SqlRows) FieldByIndex(index int) (val interface{}) {
	if sqlrws != nil {
		if sqlrws.clsimap != nil && index >= 0 && index < len(sqlrws.cls) && len(sqlrws.cls) == len(sqlrws.displaydata) {
			val = sqlrws.displaydata[index]
		}
	}
	return
}

func (sqlrws *SqlRows) cancelContext(err error) {
	if sqlrws != nil {
		if ctxcnclerr := sqlrws.ctxcnl; ctxcnclerr != nil {
			sqlrws.ctxcnl = nil
			ctxcnclerr(err)
		}
	}
}

func (sqlrws *SqlRows) Scan(castTypeVal func(valToCast interface{}, colType interface{}) (val interface{}, scanned bool)) (err error) {
	if sqlrws != nil {
		clsl := 0
		if cls, clstpes, rows, datardr, strmrdr := sqlrws.cls, sqlrws.clstpes, sqlrws.rows, sqlrws.datardr, sqlrws.strmrdr; rows != nil || datardr != nil || strmrdr != nil {
			if clsl = len(cls); clsl == 0 {
				if cls, err = sqlrws.Columns(); err != nil {
					sqlrws.cancelContext(err)
					return
				}
				clsl = len(cls)
				clstpes = sqlrws.clstpes
			}

			if clsl > 0 {
				if rows != nil {
					if err = rows.Scan(sqlrws.dataref...); err != nil {
						sqlrws.cancelContext(err)
						return
					}
					dspok := false
					for cn, cltpe := range clstpes {
						if castTypeVal == nil {
							sqlrws.displaydata[cn] = castSQLTypeValue(sqlrws.data[cn], cltpe)
						} else if sqlrws.displaydata[cn], dspok = castTypeVal(sqlrws.data[cn], cltpe); !dspok {
							sqlrws.displaydata[cn] = castSQLTypeValue(sqlrws.data[cn], cltpe)
						}
					}
				} else if datardr != nil {
					copy(sqlrws.data, datardr.data)
					copy(sqlrws.displaydata, datardr.data)
				} else if strmrdr != nil {
					copy(sqlrws.data, strmrdr.data)
					copy(sqlrws.displaydata, strmrdr.data)
				}
			}
		}
	}
	return
}

func (sqlrws *SqlRows) Err() (err error) {
	if sqlrws != nil {
		err = sqlrws.lsterr
	}
	return
}

func (sqlrws *SqlRows) ColumnTypes(cols ...string) (coltypes []*ColumnType, err error) {
	if sqlrws != nil {
		cls, cltpes := sqlrws.cls, sqlrws.clstpes
		if len(cls) == 0 {
			if cls, err = sqlrws.Columns(); err != nil {
				return
			}
			cltpes = sqlrws.clstpes[:]
			if len(cls) > 0 && len(cls) == len(cltpes) {
				coltypes = cltpes[:]
			}
		} else if len(cls) > 0 && len(cls) == len(cltpes) {
			coltypes = cltpes[:]
		}

		if colsl := len(cols); colsl > 0 {
			if colsl == 1 && strings.Contains(cols[0], ",") {
				cols = strings.Split(cols[0], ",")
			}
			var mstcdcols = []int{}
			var ccls = make([]string, len(cltpes))
			copy(ccls, cls)
			var clsignr = map[string]bool{}
			for _, cl := range cols {
				if cl == "" {
					continue
				}
				if ci, clok := sqlrws.clsimap[cl]; clok {
					mstcdcols = append(mstcdcols, ci)
					clsignr[cl] = true
					continue
				}
				for cn, c := range ccls {
					for cig, cgb := range clsignr {
						if strings.EqualFold(c, cig) && cgb {
							delete(clsignr, cig)
							ccls = append(ccls[:cn], ccls[cn+1:]...)
							break
						}
					}
					if len(clsignr) == 0 {
						break
					}
				}
				for cn, c := range ccls {
					if strings.EqualFold(c, cl) {
						mstcdcols = append(mstcdcols, sqlrws.clsimap[c])
						ccls = append(ccls[:cn], ccls[cn+1:]...)
						break
					}
				}
			}

			if mstcdcolsl := len(mstcdcols); mstcdcolsl > 0 {
				coltypes = make([]*ColumnType, mstcdcolsl)
				for mstcdcolsn, mstcdcolsp := range mstcdcols {
					coltypes[mstcdcolsn] = sqlrws.clstpes[mstcdcolsp]
				}
			}
		}
	}
	return
}

func (sqlrws *SqlRows) Columns(col ...string) (cols []string, err error) {
	if sqlrws != nil {
		if cls, clstpes, rows, datardr, strmrdr := sqlrws.cls, sqlrws.clstpes, sqlrws.rows, sqlrws.datardr, sqlrws.strmrdr; len(cls) == 0 {
			if rows != nil {
				if cls, err = rows.Columns(); err == nil {
					if rwclstpes, rwerr := rows.ColumnTypes(); rwerr == nil {
						clsimap := sqlrws.clsimap
						if clsimap == nil {
							clsimap = map[string]int{}
							sqlrws.clsimap = clsimap
						}
						for cn, c := range cls {
							clsimap[c] = cn
						}
						sqlrws.cls = make([]string, len(cls))
						copy(sqlrws.cls, cls)
						cols = cls[:]
						if len(rwclstpes) == len(cls) {
							clstpes = make([]*ColumnType, len(rwclstpes))
							for rwcltpn, rwcltpe := range rwclstpes {
								ctype := rwcltpe
								coltype := &ColumnType{}
								coltype.databaseType = ctype.DatabaseTypeName()
								coltype.length, coltype.hasLength = ctype.Length()
								coltype.name = ctype.Name()
								coltype.databaseType = ctype.DatabaseTypeName()
								coltype.nullable, coltype.hasNullable = ctype.Nullable()
								coltype.precision, coltype.scale, coltype.hasPrecisionScale = ctype.DecimalSize()
								coltype.scanType = ctype.ScanType()
								clstpes[rwcltpn] = coltype

							}
							sqlrws.clstpes = clstpes[:]
						}
						if clsl := len(cls); clsl > 0 {
							dtal, dspdtal, dtarefL := len(sqlrws.data), len(sqlrws.displaydata), len(sqlrws.dataref)
							if dtal < clsl {
								sqlrws.data = make([]interface{}, clsl)
							}
							if dtarefL < clsl {
								sqlrws.dataref = make([]interface{}, clsl)
								for dtan := range sqlrws.data[:clsl] {
									sqlrws.dataref[dtan] = &sqlrws.data[dtan]
								}
							}
							if dspdtal < clsl {
								sqlrws.displaydata = make([]interface{}, clsl)
							}
							if len(cls) > 0 && sqlrws.clsimap == nil {
								sqlrws.clsimap = map[string]int{}
								for cn, c := range cls {
									sqlrws.clsimap[c] = cn
								}
							}
						}
					} else if rwerr != nil {
						err = rwerr
						sqlrws.lsterr = err
						sqlrws.cancelContext(err)
					}
				} else if err != nil {
					sqlrws.lsterr = err
					sqlrws.cancelContext(err)
				}
			} else if datardr != nil {
				if cls, err = datardr.Columns(); err != nil {
					sqlrws.lsterr = err
					sqlrws.cancelContext(err)
				} else if clsl := len(cls); clsl > 0 {
					dtal, dspdtal, dtarefL := len(sqlrws.data), len(sqlrws.displaydata), len(sqlrws.dataref)
					if dtal < clsl {
						sqlrws.data = make([]interface{}, clsl)
					}
					if dtarefL < clsl {
						sqlrws.dataref = make([]interface{}, clsl)
					}
					if dspdtal < clsl {
						sqlrws.displaydata = make([]interface{}, clsl)
					}
					sqlrws.cls = cls[:]
					sqlrws.clstpes = datardr.clstpes[:]
					cols = cls[:]
					if len(cols) > 0 && sqlrws.clsimap == nil {
						sqlrws.clsimap = map[string]int{}
						for cn, c := range cols {
							sqlrws.clsimap[c] = cn
						}
					}
				}
			} else if strmrdr != nil {
				if cls, err = strmrdr.Columns(); err != nil {
					sqlrws.lsterr = err
					sqlrws.cancelContext(err)
				} else if clsl := len(cls); clsl > 0 {
					dtal, dspdtal, dtarefL := len(sqlrws.data), len(sqlrws.displaydata), len(sqlrws.dataref)
					if dtal < clsl {
						sqlrws.data = make([]interface{}, clsl)
					}
					if dtarefL < clsl {
						sqlrws.dataref = make([]interface{}, clsl)
					}
					if dspdtal < clsl {
						sqlrws.displaydata = make([]interface{}, clsl)
					}
					sqlrws.cls = cls[:]
					sqlrws.clstpes = strmrdr.clstpes[:]
					cols = cls[:]
					if len(cols) > 0 && sqlrws.clsimap == nil {
						sqlrws.clsimap = map[string]int{}
						for cn, c := range cols {
							sqlrws.clsimap[c] = cn
						}
					}
				}
			}
		} else if len(clstpes) == len(cls) {
			cols = cls[:]
		}
		if colsl := len(col); colsl > 0 {
			if colsl == 1 && strings.Contains(col[0], ",") {
				col = strings.Split(col[0], ",")
			}
			var mstcdcols = []int{}
			var ccls = make([]string, len(cols))
			copy(ccls, cols)
			var clsignr = map[string]bool{}
			for _, cl := range col {
				if cl == "" {
					continue
				}
				if ci, clok := sqlrws.clsimap[cl]; clok {
					mstcdcols = append(mstcdcols, ci)
					clsignr[cl] = true
					continue
				}
				for cn, c := range ccls {
					for cig, cgb := range clsignr {
						if strings.EqualFold(c, cig) && cgb {
							delete(clsignr, cig)
							ccls = append(ccls[:cn], ccls[cn+1:]...)
							break
						}
					}
					if len(clsignr) == 0 {
						break
					}
				}
				for cn, c := range ccls {
					if strings.EqualFold(c, cl) {
						mstcdcols = append(mstcdcols, sqlrws.clsimap[c])
						ccls = append(ccls[:cn], ccls[cn+1:]...)
						break
					}
				}
			}

			if mstcdcolsl := len(mstcdcols); mstcdcolsl > 0 {
				cols = make([]string, mstcdcolsl)
				for mstcdcolsn, mstcdcolsp := range mstcdcols {
					cols[mstcdcolsn] = sqlrws.cls[mstcdcolsp]
				}
				mstcdcols = nil
			}
		}
	}
	return
}

func (sqlrws *SqlRows) Next() (next bool) {
	if sqlrws != nil {
		cls, rows, datardr, strmrdr := sqlrws.cls, sqlrws.rows, sqlrws.datardr, sqlrws.strmrdr
		if len(cls) == 0 {
			sqlrws.Columns()
			if sqlrws.lsterr = rows.Err(); sqlrws.lsterr != nil {
				next = false
				sqlrws.cancelContext(sqlrws.lsterr)
				return
			}
		}
		if rows != nil {
			next = rows.Next()
			if sqlrws.lsterr = rows.Err(); sqlrws.lsterr != nil {
				next = false
				sqlrws.cancelContext(sqlrws.lsterr)
			}
		} else if datardr != nil {
			next, sqlrws.lsterr = datardr.Next()
			if sqlrws.lsterr != nil {
				next = false
				sqlrws.cancelContext(sqlrws.lsterr)
			}
		} else if strmrdr != nil {
			next, sqlrws.lsterr = strmrdr.Next()
			if sqlrws.lsterr != nil {
				next = false
				sqlrws.cancelContext(sqlrws.lsterr)
			}
		}
	}

	return
}

func (sqlrws *SqlRows) Close() (err error) {
	if sqlrws != nil {
		if rows := sqlrws.rows; rows != nil {
			sqlrws.rows = nil
			err = rows.Close()
		}
		if datardr := sqlrws.datardr; datardr != nil {
			sqlrws.datardr = nil
			err = datardr.Close()
		}
		if cls := sqlrws.cls; cls != nil {
			sqlrws.cls = nil
		}
		if clstpes := sqlrws.clstpes; clstpes != nil {
			sqlrws.clstpes = nil
		}
		if sqlrws.data != nil {
			sqlrws.data = nil
		}
		if sqlrws.dataref != nil {
			sqlrws.dataref = nil
		}
		if sqlrws.displaydata != nil {
			sqlrws.displaydata = nil
		}
		if sqlrws.ctxcnl != nil {
			sqlrws.ctxcnl = nil
		}
		if sqlrws.ctx != nil {
			sqlrws.ctx = nil
		}
	}
	return
}
