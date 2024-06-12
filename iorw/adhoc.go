package iorw

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Printer - interface
type Printer interface {
	Print(a ...interface{}) error
	Println(a ...interface{}) error
	Write(p []byte) (int, error)
}

// Reader - interface
type Reader interface {
	Seek(int64, int) (int64, error)
	SetMaxRead(int64) (err error)
	Read([]byte) (int, error)
	ReadRune() (rune, int, error)
	Readln() (string, error)
	ReadLines() ([]string, error)
	ReadAll() (string, error)
}

// PrinterReader - interface
type PrinterReader interface {
	Printer
	Reader
}

// Fprint - refer to fmt.Fprint
func Fprint(w io.Writer, a ...interface{}) (err error) {
	if len(a) > 0 && w != nil {
		for dn := range a {
			if s, sok := a[dn].(string); sok {
				if _, err = w.Write(RunesToUTF8([]rune(s)...)); err != nil {
					break
				}
			} else if ir, irok := a[dn].(io.Reader); irok {
				if wfrom, _ := w.(io.ReaderFrom); wfrom != nil {
					if _, err = wfrom.ReadFrom(ir); err == io.EOF {
						err = nil
					}
				} else if wto, _ := ir.(io.WriterTo); wto != nil {
					_, err = wto.WriteTo(w)
				} else if _, err = WriteToFunc(ir, func(b []byte) (int, error) {
					return w.Write(b)
				}); err != nil {
					break
				} else if ir, irok := a[dn].(io.RuneReader); irok {
					for err == nil {
						pr, prs, prserr := ir.ReadRune()
						if prs > 0 && (prserr == nil || prserr == io.EOF) {
							_, err = w.Write(RunesToUTF8(pr))
						}
						if prserr != nil && err == nil {
							if prserr != io.EOF {
								err = prserr
							}
							break
						}
					}
				} else {
					break
				}
			} else if bf, irok := a[dn].(*Buffer); irok {
				_, err = bf.WriteTo(w)
			} else if ir, irok := a[dn].(io.RuneReader); irok {
				for err == nil {
					pr, prs, prserr := ir.ReadRune()
					if prs > 0 && (prserr == nil || prserr == io.EOF) {
						_, err = w.Write(RunesToUTF8(pr))
					}
					if prserr != nil && err == nil {
						if prserr != io.EOF {
							err = prserr
						}
						break
					}
				}
			} else if aa, aaok := a[dn].([]interface{}); aaok {
				if len(aa) > 0 {
					if err = Fprint(w, aa...); err != nil {
						break
					}
				}
			} else if sa, saok := a[dn].([]string); saok {
				if len(sa) > 0 {
					if _, err = w.Write(RunesToUTF8([]rune(strings.Join(sa, ""))...)); err != nil {
						break
					}
				}
			} else {
				if a[dn] != nil {
					if _, err = fmt.Fprint(w, a[dn]); err != nil {
						break
					}
				}
			}
		}
	}
	return
}

func IsSpace(r rune) bool {
	return (asciiSpace[r] == 1) || (r > 128 && unicode.IsSpace(r))
}

var asciiSpace = map[rune]uint8{'\t': 1, '\n': 1, '\v': 1, '\f': 1, '\r': 1, ' ': 1}

func IsTxtPar(r rune) bool {
	return (txtpars[r] == 1)
}

var txtpars = map[rune]uint8{'\'': 1, '"': 1, '`': 1}

func CopyBytes(dest []byte, desti int, src []byte, srci int) (lencopied int, destn int, srcn int) {
	if destl, srcl := len(dest), len(src); (destl > 0 && desti < destl) && (srcl > 0 && srci < srcl) {
		if (srcl - srci) <= (destl - desti) {
			cpyl := copy(dest[desti:desti+(srcl-srci)], src[srci:srci+(srcl-srci)])
			srcn = srci + cpyl
			destn = desti + cpyl
			lencopied = cpyl
		} else if (destl - desti) < (srcl - srci) {
			cpyl := copy(dest[desti:desti+(destl-desti)], src[srci:srci+(destl-desti)])
			srcn = srci + cpyl
			destn = desti + cpyl
			lencopied = cpyl
		}
	}
	return
}

// FReadRunesEOL
func FReadRunesEOL(rdr io.RuneReader, txtpar rune, eolrns ...rune) (rnsline []rune, err error) {
	if eolrnsl := len(eolrns); rdr != nil && eolrnsl > 0 {
		func() {
			eofri := 0
			prvr := rune(0)
			txtr := rune(0)
			for {
				if r, s, rerr := rdr.ReadRune(); rerr == nil || rerr == io.EOF {
					if s > 0 {
						if txtr == 0 {
							if txtpar > 0 && r > 0 && r == txtpar {
								txtr = r
								prvr = r
							} else {
								if eofri > 0 && eolrns[eofri-1] == prvr && eolrns[eofri] != r {
									rnsline = append(rnsline, eolrns[:eofri]...)
									eofri = 0
								}
								if eolrns[eofri] == r {
									eofri++
									if eofri == eolrnsl {
										break
									} else {
										prvr = r
									}
								} else {
									if eofri > 0 {
										rnsline = append(rnsline, eolrns[:eofri]...)
										eofri = 0
									}
									prvr = r
									rnsline = append(rnsline, r)
								}
							}
						} else if txtr > 0 && txtpar > 0 && r > 0 {
							if txtr == r {
								if prvr == r {
									rnsline = append(rnsline, r)
								} else {
									txtr = 0
								}
							} else {
								rnsline = append(rnsline, r)
								prvr = r
							}
						}
					} else {
						if rerr != nil {
							if rerr != io.EOF {
								err = nil
							}
						}
						break
					}
				} else if rerr != nil {
					if rerr != io.EOF {
						err = nil
					}
					break
				}
			}
		}()
	}
	return
}

func ReadRunesEOL(rdr func() (r rune, size int, err error), txtpar rune, foundRunes func(bool, bool, ...rune), eolrns ...rune) (err error) {
	if eolrnsl := len(eolrns); rdr != nil && eolrnsl > 0 && foundRunes != nil {
		func() {
			eofri := 0
			prvr := rune(0)
			txtr := rune(0)
			var rnsline []rune
			for {
				if r, s, rerr := rdr(); rerr == nil || rerr == io.EOF {
					if s > 0 {
						if txtr == 0 {
							if txtpar > 0 && r > 0 && r == txtpar {
								txtr = r
								prvr = r
							} else {
								if eofri > 0 && eolrns[eofri-1] == prvr && eolrns[eofri] != r {
									rnsline = append(rnsline, eolrns[:eofri]...)
									eofri = 0
								}
								if eolrns[eofri] == r {
									eofri++
									if eofri == eolrnsl {
										if len(rnsline) > 0 {
											foundRunes(true, false, rnsline...)
											rnsline = nil
										}
										return
									} else {
										prvr = r
									}
								} else {
									if eofri > 0 {
										rnsline = append(rnsline, eolrns[:eofri]...)
										eofri = 0
									}
									prvr = r
									rnsline = append(rnsline, r)
								}
							}
						} else if txtr > 0 && txtpar > 0 && r > 0 {
							if txtr == r {
								if prvr == r {
									rnsline = append(rnsline, r)
								} else {
									txtr = 0
								}
							} else {
								rnsline = append(rnsline, r)
								prvr = r
							}
						}
					} else {
						if rerr != nil {
							err = rerr
							if rerr == io.EOF {
								foundRunes(true, true, rnsline...)
								if len(rnsline) > 0 {
									rnsline = nil
								}
							}
						}
						return
					}
				} else if rerr != nil {
					err = rerr
					if rerr == io.EOF {
						foundRunes(true, true, rnsline...)
						if len(rnsline) > 0 {
							rnsline = nil
						}
					}
					return
				}
			}
		}()
	}
	return
}

// Fprintln - refer to fmt.Fprintln
func Fprintln(w io.Writer, a ...interface{}) (err error) {
	if len(a) > 0 && w != nil {
		err = Fprint(w, a...)
	}
	if err == nil {
		err = Fprint(w, "\r\n")
	}
	return
}

// ReadLines from r io.Reader as lines []string
func ReadLines(r interface{}) (lines []string, err error) {
	if r != nil {
		var rnrd io.RuneReader = nil
		if rnr, rnrok := r.(io.RuneReader); rnrok {
			rnrd = rnr
		} else {
			if rd, _ := r.(io.Reader); rd != nil {
				rnrd = bufio.NewReader(rd)
			}
		}
		if rnrd == nil {
			return
		}
		rns := make([]rune, 1024)
		rnsi := 0
		s := ""
		for {
			rn, size, rnerr := rnrd.ReadRune()
			if size > 0 {
				if rn == '\n' {
					if rnsi > 0 {
						s += string(rns[:rnsi])
						rnsi = 0
					}
					if s != "" {
						s = strings.TrimSpace(s)
						if lines == nil {
							lines = []string{}
						}
						lines = append(lines, s)
						s = ""
					}
					continue
				}
				rns[rnsi] = rn
				rnsi++
				if rnsi == len(rns) {
					s += string(rns[:rnsi])
					rnsi = 0
				}
			}
			if rnerr != nil {
				err = rnerr
				if rnsi > 0 {
					s += string(rns[:rnsi])
					rnsi = 0
				}
				if s != "" {
					s = strings.TrimSpace(s)
					if lines == nil {
						lines = []string{}
					}
					lines = append(lines, s)
					s = ""
				}
				if err == io.EOF {
					err = nil
				}
				break
			}
		}
	}
	return
}

func ReadWriteEof(readfunc func([]byte) (int, error), writefunc func([]byte) (int, error), nexteof func() []byte, foundeeof func() error) (n int64, err error) {
	for err == nil {
		oefbytes := nexteof()
		if eofl := len(oefbytes); eofl > 0 {
			eofl := len(oefbytes)
			eofi := 0
			wn := 0
			prveofb := byte(0)
			rdrbytes := make([]byte, 8192)
			rdrn, rdrerr := readfunc(rdrbytes)
			if rdrbts := rdrbytes[:rdrn]; len(rdrbts) > 0 {
				for bn, bb := range rdrbts {
					if eofi > 0 && oefbytes[eofi-1] == prveofb && oefbytes[eofi] != bb {
						ei := eofi
						eofi = 0
						prveofb = 0
						if wn, err = writefunc(oefbytes[:ei]); err != nil {
							return
						}
						n += int64(wn)
					}
					if oefbytes[eofi] == bb {
						eofi++
						if eofi == eofl {
							eofi = 0
							eofl = 0
							prveofb = 0
							if err = foundeeof(); err != nil {
								return
							}
							oefbytes = nexteof()
							if eofl = len(oefbytes); eofl == 0 {
								return
							}
						} else {
							prveofb = bb
						}
					} else {
						if eofi > 0 {
							ei := eofi
							eofi = 0
							prveofb = 0
							if wn, err = writefunc(oefbytes[:ei]); err != nil {
								return
							}
							n += int64(wn)
						}
						prveofb = bb
						if wn, err = writefunc(rdrbts[bn : bn+1]); err != nil {
							return
						}
						n += int64(wn)
					}
				}
			}
			if rdrerr == io.EOF {

				rdrerr = nil
			}
		} else {
			break
		}
	}
	return
}

// ReadLine from r io.Reader as s string
func ReadLine(rs ...interface{}) (s string, err error) {
	if rsl := len(rs); rsl >= 1 {
		var r interface{} = rs[0]
		var cantrim = false
		if rsl > 1 {
			cantrim, _ = rs[1].(bool)
		}
		if r != nil {
			var rnrd io.RuneReader = nil
			if rnr, rnrok := r.(io.RuneReader); rnrok {
				rnrd = rnr
			} else if rr, rrok := r.(io.Reader); rrok {
				rnrd = bufio.NewReader(rr)
			}
			rns := make([]rune, 1024)
			rnsi := 0
			for {
				rn, size, rnerr := rnrd.ReadRune()
				if size > 0 {
					if rn == '\n' {
						if rnsi > 0 {
							s += strings.TrimFunc(string(rns[:rnsi]), IsSpace)
							rnsi = 0
						}
						break
					}
					rns[rnsi] = rn
					rnsi++
					if rnsi == len(rns) {
						s += string(rns[:rnsi])
						rnsi = 0
					}
				}
				if rnerr != nil {
					err = rnerr
					if rnsi > 0 && (err == nil || err == io.EOF) {
						if err == io.EOF {
							err = nil
						}
						s += string(rns[:rnsi])
						rnsi = 0
					}
					break
				}
			}
		}
		if cantrim {
			s = strings.TrimFunc(s, IsSpace)
		}
		//
	}
	return
}

// ReaderToString read reader and return content as string
func ReaderToString(r interface{}) (s string, err error) {
	runes := make([]rune, 1024)
	runesi := 0
	if err = ReadRunesEOFFunc(r, func(rn rune) error {
		runes[runesi] = rn
		runesi++
		if runesi == len(runes) {
			s += string(runes[:runesi])
			runesi = 0
		}
		return nil
	}); err == nil || err == io.EOF {
		if runesi > 0 {
			s += string(runes[:runesi])
			runesi = 0
		}
	}
	if err == io.EOF {
		err = nil
	}
	return
}

// ReadRunesEOFFunc read runes from r io.Reader and call fncrne func(rune) error
func ReadRunesEOFFunc(r interface{}, fncrne func(rune) error) (err error) {
	if r != nil && fncrne != nil {
		var rnrd io.RuneReader = nil
		if rnr, rnrok := r.(io.RuneReader); rnrok {
			rnrd = rnr
		} else if rdr, rdrok := r.(io.Reader); rdrok {
			rnrd = bufio.NewReader(rdr)
		}
		if rnrd != nil {
			for {
				rn, size, rnerr := rnrd.ReadRune()
				if size > 0 {
					if err = fncrne(rn); err != nil {
						break
					}
				}
				if err == nil && rnerr != nil {
					if rnerr != io.EOF {
						err = rnerr
					}
					break
				}
			}
		}
	}
	return
}

func RunesToUTF8(rs ...rune) []byte {
	size := 0
	for rn := range rs {
		size += utf8.RuneLen(rs[rn])
	}
	bs := make([]byte, size)
	count := 0
	for rn := range rs {
		count += utf8.EncodeRune(bs[count:], rs[rn])
	}

	return bs
}

type funcrdrwtr struct {
	funcw func([]byte) (int, error)
	funcr func([]byte) (int, error)
}

func (fncrw *funcrdrwtr) Close() (err error) {
	if fncrw != nil {
		if fncrw.funcr != nil {
			fncrw.funcr = nil
		}
		if fncrw.funcw != nil {
			fncrw.funcw = nil
		}
		fncrw = nil
	}
	return
}

func (fncrw *funcrdrwtr) Write(p []byte) (n int, err error) {
	if fncrw != nil && fncrw.funcw != nil {
		n, err = fncrw.funcw(p)
	}
	return
}

func (fncrw *funcrdrwtr) Read(p []byte) (n int, err error) {
	if fncrw != nil && fncrw.funcr != nil {
		n, err = fncrw.funcr(p)
	}
	return
}

func WriteToFunc(r io.Reader, funcw func([]byte) (int, error), bufsize ...int) (n int64, err error) {
	if r != nil && funcw != nil {
		func() {
			n, err = ReadWriteToFunc(funcw, func(b []byte) (int, error) {
				return r.Read(b)
			}, bufsize...)
		}()
	}
	return
}

func ReadToFunc(w io.Writer, funcr func([]byte) (int, error)) (n int64, err error) {
	if w != nil && funcr != nil {
		func() {
			n, err = ReadWriteToFunc(func(b []byte) (int, error) {
				return w.Write(b)
			}, funcr)
		}()
	}
	return
}

func ReadHandle(r io.Reader, handle func([]byte), maxrlen int) (n int, err error) {
	if maxrlen < 4096 {
		maxrlen = 4096
	}
	s := make([]byte, maxrlen)
	sn := 0
	si := 0
	sl := len(s)
	serr := error(nil)
	for n < maxrlen && err == nil {
		switch sn, serr = r.Read(s[si : si+(sl-si)]); true {
		case sn < 0:
			err = serr
		case sn == 0: // EOF
			if si > 0 {
				handle(s[:si])
				si = 0
			}
			err = serr
		case sn > 0:
			si += sn
			n += sn
			err = serr
		}
	}
	if si > 0 {
		handle(s[:si])
	}
	if n == 0 && err == nil {
		err = io.EOF
	}
	return
}

func ReadWriteToFunc(funcw func([]byte) (int, error), funcr func([]byte) (int, error), bufsize ...int) (n int64, err error) {
	if funcw != nil && funcr != nil {
		fncrw := &funcrdrwtr{funcr: funcr, funcw: funcw}
		func() {
			defer func() {
				if rv := recover(); rv != nil {
					switch x := rv.(type) {
					case string:
						err = errors.New(x)
					case error:
						err = x
					default:
						err = errors.New("unknown panic")
					}
				}
				fncrw.Close()
			}()
			if len(bufsize) > 0 {
				if bufsize[0] < 8912 {
					n, err = io.Copy(fncrw, fncrw)
				} else {
					n, err = io.CopyBuffer(fncrw, fncrw, make([]byte, bufsize[0]))
				}
			} else {
				n, err = io.Copy(fncrw, fncrw)
			}
		}()
	}
	return
}

func RunesToBytes(r ...rune) (bts []byte, rl int) {
	return RunesToUTF8(r...), len(r)
}

func ToData(format string, a ...interface{}) (data interface{}, err error) {
	mltiargrdr := NewMultiArgsReader(a...)
	defer mltiargrdr.Close()
	if format = strings.TrimFunc(format, IsSpace); format == "" || strings.EqualFold(format, "json") {
		dec := json.NewDecoder(mltiargrdr)
		tknlvl := -1
		mps := map[int]map[string]interface{}{}
		mpdkeys := map[int]string{}
		lsts := map[int][]interface{}{}
		tknlvltpes := map[int]rune{}
		lsttkntpe := rune(0)

		lstkey := ""
		for {
			if tkn, tknerr := dec.Token(); tknerr == nil {
				if tok, ok := tkn.(json.Delim); ok {
					if tok == '{' {
						lstmap := map[string]interface{}{}
						tknlvl++
						lsttkntpe = 'O'
						tknlvltpes[tknlvl] = lsttkntpe
						mps[tknlvl] = lstmap
					} else if tok == '}' {
						lstmap := mps[tknlvl]
						delete(tknlvltpes, tknlvl)
						delete(mps, tknlvl)
						delete(mpdkeys, tknlvl)
						tknlvl--
						if tknlvl <= -1 {
							data = lstmap
							lsttkntpe = rune(0)
						} else {
							if lsttkntpe, lstkey = tknlvltpes[tknlvl], mpdkeys[tknlvl]; lsttkntpe == 'O' {
								if lstkey != "" {
									mps[tknlvl][lstkey] = lstmap
									mpdkeys[tknlvl] = ""
								}
							} else if lsttkntpe == 'A' {
								lsts[tknlvl] = append(lsts[tknlvl], lstmap)
							}
						}
					} else if tok == '[' {
						lstlst := []interface{}{}
						tknlvl++
						lsts[tknlvl] = lstlst
						lsttkntpe = 'A'
						tknlvltpes[tknlvl] = lsttkntpe
					} else if tok == ']' {
						lstlst := lsts[tknlvl]
						delete(tknlvltpes, tknlvl)
						delete(lsts, tknlvl)
						tknlvl--
						if tknlvl <= -1 {
							data = lstlst
							lsttkntpe = rune(0)
						} else {
							if lsttkntpe, lstkey = tknlvltpes[tknlvl], mpdkeys[tknlvl]; lsttkntpe == 'O' {
								if lstkey != "" {
									mps[tknlvl][lstkey] = lstlst
									mpdkeys[tknlvl] = ""
								}
							} else if lsttkntpe == 'A' {
								lsts[tknlvl] = append(lsts[tknlvl], lstlst)
							}
						}
					}
				} else if lsttkntpe == 'O' {
					if lstkey = mpdkeys[tknlvl]; lstkey != "" {
						if tkn != nil {
							if flt, fltok := tkn.(float64); fltok {
								mps[tknlvl][lstkey] = flt
							} else if blnt, blntok := tkn.(bool); blntok {
								mps[tknlvl][lstkey] = blnt
							} else if strt, strtok := tkn.(string); strtok {
								mps[tknlvl][lstkey] = strt
							} else if jsnnr, jsnnrtok := tkn.(json.Number); jsnnrtok {
								if flt, _ := jsnnr.Float64(); flt >= 0.0 {
									mps[tknlvl][lstkey], _ = jsnnr.Int64()
								} else {
									mps[tknlvl][lstkey] = flt
								}
							}
						} else {
							mps[tknlvl][lstkey] = nil
						}
						mpdkeys[tknlvl] = ""
					} else if lstkey, _ = tkn.(string); lstkey != "" {
						mpdkeys[tknlvl] = lstkey
					}
				} else if lsttkntpe == 'A' {
					if tkn != nil {
						if jsnnr, jsnnrtok := tkn.(json.Number); jsnnrtok {
							if flt, flterr := jsnnr.Float64(); flterr == nil {
								if flt == 0 || flt-float64(int64(flt)) == 0 {
									lsts[tknlvl] = append(lsts[tknlvl], int64(flt))
								} else {
									lsts[tknlvl] = append(lsts[tknlvl], int64(flt))
								}
							} else if intt, intterr := jsnnr.Int64(); intterr == nil {
								lsts[tknlvl] = append(lsts[tknlvl], intt)
							}
						} else if flt, fltok := tkn.(float64); fltok {
							if flt == 0 || flt-float64(int64(flt)) == 0 {
								lsts[tknlvl] = append(lsts[tknlvl], int64(flt))
							} else {
								lsts[tknlvl] = append(lsts[tknlvl], int64(flt))
							}
						} else if blnt, blntok := tkn.(bool); blntok {
							lsts[tknlvl] = append(lsts[tknlvl], blnt)
						} else if strt, strtok := tkn.(string); strtok {
							lsts[tknlvl] = append(lsts[tknlvl], strt)
						}
					} else {
						lsts[tknlvl] = append(lsts[tknlvl], nil)
					}
				}
			} else if tknerr != nil {
				if tknerr == io.EOF {
					if tknlvl > -1 {
						data = nil
						if lstkey != "" {
							lstkey = ""
						}
						var clearmap func(map[string]interface{}) = nil
						var cleararr func(arr []interface{}) = nil

						cleararr = func(arr []interface{}) {
							if arrl := len(arr); arrl > 0 {
								for arrl > 0 {
									arrl--
									if vm, _ := arr[0].(map[string]interface{}); vm != nil {
										clearmap(vm)
									} else if va, _ := arr[0].([]interface{}); va != nil {
										cleararr(va)
									}
									arr = arr[1:]
								}
							}
						}

						clearmap = func(m map[string]interface{}) {
							for k, v := range m {
								if vm, _ := v.(map[string]interface{}); vm != nil {
									clearmap(vm)
								} else if va, _ := v.([]interface{}); va != nil {
									cleararr(va)
								}
								m[k] = nil
								delete(m, k)
							}
						}
						for tknlvl > -1 {
							if lsttkntpe = tknlvltpes[tknlvl]; lsttkntpe == 'O' {
								lstmp := mps[tknlvl]
								clearmap(lstmp)
								delete(mps, tknlvl)
							} else if lsttkntpe == 'A' {
								lstls := lsts[tknlvl]
								cleararr(lstls)
								delete(lsts, tknlvl)
							}
							tknlvl--
						}
					}
				}
				break
			}
		}
	} else if strings.EqualFold(format, "raw") {
		data, err = mltiargrdr.ReadAll()
	}
	return
}

func RunesHasPrefix(runes []rune, subrunes ...rune) bool {
	if lnrns, lnsubrns := len(runes), len(subrunes); lnrns >= lnsubrns {
		lnrns = lnsubrns
		for _, r := range runes[:lnsubrns] {
			for srn, sr := range subrunes {
				if sr != r {
					break
				}
				if srn == lnsubrns-1 {
					return true
				}
			}
		}
	}
	return false
}

func RunesHasSuffix(runes []rune, subrunes ...rune) bool {
	if lnrns, lnsubrns := len(runes), len(subrunes); lnrns >= lnsubrns {
		maxrns := lnrns
		lnrns = lnsubrns
		for _, r := range runes[maxrns-lnsubrns:] {
			for srn, sr := range subrunes {
				if sr != r {
					break
				}
				if srn == lnsubrns-1 {
					return true
				}
			}
		}
	}
	return false
}

func IndexOfRunes(runes []rune, subrunes ...rune) int {
	if lnrns, lnsubrns := len(runes), len(subrunes); lnrns >= lnsubrns {
		srn := 0
		for rn, r := range runes {
			if subrunes[srn] == r {
				srn++
				if srn == lnsubrns {
					return rn - (srn - 1)
				}
				continue
			}
			if srn > 0 && subrunes[srn-1] == r {
				continue
			}
			srn = 0
			continue
		}
	}
	return -1
}

func LastIndexOfRunes(runes []rune, subrunes ...rune) int {
	if lnrns, lnsubrns := len(runes), len(subrunes); lnrns >= lnsubrns {
		for rn := range runes {
			tstrn := lnrns - (rn + lnsubrns)
			r := runes[tstrn]
			srn := 0
			if sr := subrunes[srn]; sr == r {
				srn++
				if srn == lnsubrns {
					return tstrn - (srn - 1)
				}
				continue
			}
			srn = 0
		}
	}
	return -1
}

func ReadRunes(p []rune, rds ...interface{}) (n int, err error) {
	if pl := len(p); pl > 0 {
		var rd io.RuneReader = nil
		if len(rds) == 1 {
			if rd, _ = rds[0].(io.RuneReader); rd == nil {
				if r, _ := rds[0].(io.Reader); r != nil {
					rd = bufio.NewReader(r)
				}
			}
			if rd != nil {
				pi := 0
				for pi < pl {
					pr, ps, perr := rd.ReadRune()
					if ps > 0 {
						p[pi] = pr
						pi++
					}
					if perr != nil || ps == 0 {
						if perr == nil {
							perr = io.EOF
						}
						err = perr
						break
					}
				}
				if n = pi; n > 0 && err == io.EOF {
					err = nil
				}
			}
		}
	}
	return
}
