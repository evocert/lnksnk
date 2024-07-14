package iorw

import (
	"bufio"
	"io"
	"sort"
	"strings"
	"unicode/utf8"
)

type ReplaceRuneReader struct {
	eoffnd bool
	//rnstst        []rune
	//rnststi       int
	//rmnrnsl       int
	orgrdr        *RuneReaderSlice
	crntrdr       io.RuneReader
	crntrns       []rune
	crntrnsl      int
	rplcewith     map[string]interface{}
	rplcekeys     []string
	crntrplcekeys map[int]string
	mxlrplcL      int
	OnClose       func(*ReplaceRuneReader, error) (err error)
	undrlyingrdr  ReadRuneFunc
}

func NewReplaceRuneReader(orgrdr interface{}, rplwiths ...interface{}) (rplcerrdr *ReplaceRuneReader) {
	orgrnrdr, _ := orgrdr.(io.RuneReader)
	if orgrnrdr == nil {
		if orgr, _ := orgrdr.(io.Reader); orgr != nil {
			orgrnrdr = bufio.NewReaderSize(orgr, 1)
		}
	}
	rplcerrdr = &ReplaceRuneReader{orgrdr: NewRuneReaderSlice(orgrnrdr)}
	rplwithsl := len(rplwiths)
	for rplwithsl > 0 && rplwithsl%2 == 0 {
		rplcerrdr.ReplaceWith(rplwiths[0], rplwiths[1])
		rplwiths = rplwiths[2:]
		rplwithsl -= 2
	}
	rplcerrdr.undrlyingrdr = func() (rune, int, error) {
		return rplcerrdr.ReadUnderlyingRune()
	}
	return
}

func (rplcerrdr *ReplaceRuneReader) UnderlyingReader() ReadRuneFunc {
	if rplcerrdr == nil || rplcerrdr.undrlyingrdr == nil {
		return func() (rune, int, error) {
			return 0, 0, io.EOF
		}
	}
	return rplcerrdr.undrlyingrdr
}

func (rplcerrdr *ReplaceRuneReader) ReadUnderlyingRune() (r rune, size int, err error) {
	if rplcerrdr == nil {
		return 0, 0, io.EOF
	}
	if crntrdr := rplcerrdr.crntrdr; crntrdr != nil {
		r, size, err = crntrdr.ReadRune()
		if size == 0 && err == io.EOF {
			if rplcerrdr.crntrdr == rplcerrdr.orgrdr {
				rplcerrdr.orgrdr = nil
			}
			rplcerrdr.crntrdr = nil
		}
	}
	if size == 0 && (err == nil || err == io.EOF) && rplcerrdr.orgrdr != nil {
		r, size, err = rplcerrdr.orgrdr.ReadRune()
		return
	}
	if size == 0 && err == nil {
		err = io.EOF
	}
	return
}

func (rplcerrdr *ReplaceRuneReader) FoundEOF() bool {
	if rplcerrdr == nil {
		return false
	}
	return rplcerrdr.eoffnd
}

func (rplcerrdr *ReplaceRuneReader) ReadRunesUntil(eof ...interface{}) io.RuneReader {
	if rplcerrdr == nil {
		return nil
	}

	var eofrunes []rune = nil
	if len(eof) == 1 {
		if s, sok := eof[0].(string); sok && s != "" {
			eofrunes = []rune(s)
		} else {
			eofrunes, _ = eof[0].([]rune)
		}
	}
	if eofl := len(eofrunes); eofl > 0 {
		rplcerrdr.eoffnd = false
		eofi := 0
		prveofr := rune(0)
		bfrdrns := []rune{}
		var rnsrdr ReadRuneFunc = nil
		rnsrdr = func() (r rune, size int, err error) {
			if len(bfrdrns) > 0 {
				r = bfrdrns[0]
				bfrdrns = bfrdrns[1:]
				size = utf8.RuneLen(r)
				return
			}
			for !rplcerrdr.eoffnd {
				r, size, err = rplcerrdr.ReadUnderlyingRune()
				if size > 0 {
					if err == nil || err == io.EOF {
						if eofi > 0 && eofrunes[eofi-1] == prveofr && eofrunes[eofi] != r {
							bfrdrns = append(bfrdrns, eofrunes[:eofi]...)
							eofi = 0
						}
						if eofrunes[eofi] == r {
							eofi++
							if eofi == eofl {
								rplcerrdr.eoffnd = true
								err = io.EOF
								r = 0
								size = 0
								prveofr = 0
								return
							}
							prveofr = r
							continue
						}
						if len(bfrdrns) > 0 {
							bfrdrns = append(bfrdrns, r)
							return rnsrdr.ReadRune()
						}
						return
					}
					if err != io.EOF {
						return
					}
				}
			}
			if size == 0 && err == nil {
				err = io.EOF
			}
			return
		}
		return rnsrdr
	}
	return nil
}

type ReplaceRunesEvent func(matchphrase string, rplcerrdr *ReplaceRuneReader) interface{}

func (rplcerrdr *ReplaceRuneReader) ReplaceWith(phrase, replacewith interface{}) {
	if rplcerrdr != nil {
		if sphrase, _ := phrase.(string); sphrase != "" {
			if rplcerrdr.rplcewith == nil {
				rplcerrdr.rplcewith = map[string]interface{}{}
				rplcerrdr.mxlrplcL = 0
			}
			if phrsl := len(sphrase); phrsl > rplcerrdr.mxlrplcL {
				rplcerrdr.mxlrplcL = phrsl
			}
			rplcerrdr.rplcewith[sphrase] = replacewith
			rplcerrdr.rplcekeys = []string{}
			for rplckey := range rplcerrdr.rplcewith {
				rplcerrdr.rplcekeys = append(rplcerrdr.rplcekeys, rplckey)
			}
			if len(rplcerrdr.rplcekeys) > 0 {
				sort.Slice(rplcerrdr.rplcekeys, func(i, j int) bool {
					return len(rplcerrdr.rplcekeys[i]) < len(rplcerrdr.rplcekeys[j])
				})
			}
		}
	}
}

func replacedWithReader(rplcerrdr *ReplaceRuneReader, rplcewith map[string]interface{}, phrase string, isrepeatable bool, prerns ...rune) (bool, error) {
	if len(rplcewith) > 0 && phrase != "" {
		var appndrns = func(postrdr io.RuneReader) io.RuneReader {
			if len(prerns) > 0 {
				return NewRuneReaderSlice(NewRunesReader(prerns...), postrdr)
			}
			return postrdr
		}
		if phrsev, phrsok := rplcewith[phrase]; phrsok {
			if phrss, phrssok := phrsev.(string); phrssok && phrss != "" {
				rplcerrdr.crntrdr = appndrns(strings.NewReader(phrss))
				return true, nil
			}
			if phrsr, _ := phrsev.(io.Reader); phrsr != nil {
				if rdr, _ := phrsr.(io.RuneReader); rdr == nil {
					rplcerrdr.crntrdr = appndrns(bufio.NewReader(phrsr))
					return true, nil
				}
				rplcerrdr.crntrdr, _ = phrsr.(io.RuneReader)
				rplcerrdr.crntrdr = appndrns(rplcerrdr.crntrdr)
				return true, nil
			}
			if phrsbf, _ := phrsev.(*Buffer); !phrsbf.Empty() {
				if isrepeatable {
					rplcerrdr.crntrdr = appndrns(phrsbf.Clone(true).Reader(true))
					return true, nil
				}
				rplcerrdr.crntrdr = appndrns(phrsbf.Reader())
				return true, nil
			}
			if subReplaceRdrEvent, _ := phrsev.(func(string, *ReplaceRuneReader) interface{}); subReplaceRdrEvent != nil {
				if nxtrdr := subReplaceRdrEvent(phrase, rplcerrdr); nxtrdr != nil {
					if errnxtrdr, _ := nxtrdr.(error); errnxtrdr != nil {
						return false, errnxtrdr
					}

					if nxtvrnr, _ := nxtrdr.(io.RuneReader); nxtvrnr != nil {
						rplcerrdr.crntrdr = appndrns(nxtvrnr)
						return true, nil
					}
					if nxtvs, _ := nxtrdr.(string); nxtvs != "" {
						rplcerrdr.crntrdr = appndrns(strings.NewReader(nxtvs))
						return true, nil
					}
					if nxtvrns, _ := nxtrdr.([]int32); len(nxtvrns) > 0 {
						rplcerrdr.crntrdr = appndrns(strings.NewReader(string(nxtvrns)))
						return true, nil
					}
					rplcerrdr.crntrdr = appndrns(NewMultiArgsReader(nxtrdr))
					return true, nil
				}
			}
			if subReplaceRdrEvent, _ := phrsev.(ReplaceRunesEvent); subReplaceRdrEvent != nil {
				if nxtrdr := subReplaceRdrEvent(phrase, rplcerrdr); nxtrdr != nil {
					if errnxtrdr, _ := nxtrdr.(error); errnxtrdr != nil {
						return false, errnxtrdr
					}

					if nxtvrnr, _ := nxtrdr.(io.RuneReader); nxtvrnr != nil {
						rplcerrdr.crntrdr = appndrns(nxtvrnr)
						return true, nil
					}
					if nxtvs, _ := nxtrdr.(string); nxtvs != "" {
						rplcerrdr.crntrdr = appndrns(strings.NewReader(nxtvs))
						return true, nil
					}
					if nxtvrns, _ := nxtrdr.([]int32); len(nxtvrns) > 0 {
						rplcerrdr.crntrdr = appndrns(strings.NewReader(string(nxtvrns)))
						return true, nil
					}
					rplcerrdr.crntrdr = appndrns(NewMultiArgsReader(nxtrdr))
					return true, nil
				}
			}
		}
	}
	return false, nil
}

func underlineReadRune(rplcerrdr *ReplaceRuneReader) (r rune, size int, err error) {
	if rplcerrdr.crntrnsl > 0 {
		r, size, err = rplcerrdr.crntrns[0], 1, nil
		rplcerrdr.crntrns = rplcerrdr.crntrns[1:]
		rplcerrdr.crntrnsl--
		return
	}
	if crntrdr := rplcerrdr.crntrdr; crntrdr != nil {
		r, size, err = crntrdr.ReadRune()
		if size == 0 && err == io.EOF {
			if rplcerrdr.crntrdr == rplcerrdr.orgrdr {
				rplcerrdr.orgrdr = nil
			}
			rplcerrdr.crntrdr = nil
			r, size, err = underlineReadRune(rplcerrdr)
			return
		}
		return
	}
	if len(rplcerrdr.rplcekeys) == 0 && rplcerrdr.orgrdr != nil {
		r, size, err = rplcerrdr.orgrdr.ReadRune()
		return
	}
	if orgrdr := rplcerrdr.orgrdr; orgrdr != nil {
		mxkl := 0
		knsfnd := []int{}
		var mtchrns []rune = nil
		unmathcedrns := make([]rune, 4096)
		unmathcl := 0
		var flushUnmatched = func() io.RuneReader {
			if unmathcl > 0 {
				unmtchl := unmathcl
				unmathcl = 0
				return NewRunesReader(unmathcedrns[:unmtchl]...)
			}
			return nil
		}
		for err == nil {
			r, size, err = orgrdr.ReadRune()
			if size > 0 && (err == nil || err == io.EOF) {
				for kn, kk := range rplcerrdr.rplcekeys {
					if rune(kk[0]) == r {
						knsfnd = append(knsfnd, kn)
						if len([]rune(kk)) > mxkl {
							mxkl = len([]rune(kk))
						}
					}
				}
				if mxkl > 0 {
					mtchrns = make([]rune, mxkl)
					mtchrns[0] = r
					n, nerr := ReadRunes(mtchrns[1:], orgrdr)
					if n > 0 && (nerr == nil || nerr == io.EOF) {
						mxkl = n + 1
						knfndl := len(knsfnd)
					doagain:
						mxfndkl := 0
						mxfndkn := 0
						var mxfndkrns []rune = nil
						for ki, kn := range knsfnd {
							kkrns := []rune(rplcerrdr.rplcekeys[kn])
							if len(kkrns) > mxkl {
								knsfnd = append(knsfnd[:ki], knsfnd[ki+1:]...)
								knfndl--
								if knfndl > 0 {
									goto doagain
								}
								if rplcerrdr.crntrdr == nil {
									rplcerrdr.crntrdr = NewRunesReader(append(unmathcedrns[:unmathcl], mtchrns[:mxkl]...)...)
									return underlineReadRune(rplcerrdr)
								}
							}
							if strings.HasPrefix(string(mtchrns[:mxkl]), string(kkrns)) {
								if mxfndkl < len(kkrns) {
									mxfndkl = len(kkrns)
									mxfndkn = kn
									mxfndkrns = kkrns
								}
								if ki+1 < knfndl {
									continue
								}

								mtchdprhase := rplcerrdr.rplcekeys[mxfndkn]
								if rmndrns := mtchrns[len(mxfndkrns):]; len(rmndrns) > 0 {
									rplcerrdr.PreAppend(NewRunesReader(rmndrns...))
								}
								if _, err = replacedWithReader(rplcerrdr, rplcerrdr.rplcewith, mtchdprhase, true, unmathcedrns[:unmathcl]...); err != nil {
									return 0, 0, err
								}
								return underlineReadRune(rplcerrdr)
							}
							knsfnd = append(knsfnd[:ki], knsfnd[ki+1:]...)
							knfndl--
							if knfndl > 0 {
								goto doagain
							}
							if rplcerrdr.crntrdr == nil {
								rplcerrdr.crntrdr = NewRunesReader(append(unmathcedrns[:unmathcl], mtchrns[:mxkl]...)...)
								return underlineReadRune(rplcerrdr)
							}
						}

						mtchrns = nil
						return underlineReadRune(rplcerrdr)
					}
					if nerr != nil {
						err = nerr
					}
				}
				unmathcedrns[unmathcl] = r
				unmathcl++
				if unmathcl == 4096 {
					break
				}
			}
		}

		if flshd := flushUnmatched(); flshd != nil {
			if rplcerrdr.crntrdr == nil {
				rplcerrdr.crntrdr = flshd
				return underlineReadRune(rplcerrdr)
			}
		}

		/*var rnstst []rune = rplcerrdr.rnstst

		for err == nil {
			rnststL := rmngl
			if rplcerrdr.orgrdr != nil {
				for rnsi := range rnstst {
					if r, size, err = rplcerrdr.orgrdr.ReadRune(); size > 0 && (err == io.EOF || err == nil) {
						rnstst[rnsi+rmngl] = r
						rnststL += (rmngl + 1)
						continue
					}
					if err != nil {
						if err != io.EOF {
							return
						}
					}
					break
				}
			}

			if rnststL > 0 {
				rplcerrdr.rnststi = 0
				lstphrn := -1
				lstphrsi := -1
				crntrplcekeys := rplcerrdr.crntrplcekeys
				ki := 0
				mxkl := 0

				for tn, tr := range rnstst[:rnststL] {
					if len(crntrplcekeys) == 0 {
						for kn, kk := range rplcerrdr.rplcekeys {
							if rune(kk[ki]) == tr {
								if lstphrsi == -1 {
									lstphrsi = tn
								}
								lstphrn = kn
								if crntrplcekeys == nil {
									crntrplcekeys = map[int]string{}
									rplcerrdr.crntrplcekeys = crntrplcekeys
								}
								if crntrplcekeys[kn] == "" {
									crntrplcekeys[kn] = kk
									if mxkl < len(kk) {
										mxkl = len(kk)
									}
								}
							}
						}
						if len(crntrplcekeys) > 0 {
							if ki+1 == mxkl && len(crntrplcekeys) == 1 {
								goto foundmtch
							}
							ki++
						}
						continue
					}

					for kn, kk := range crntrplcekeys {
						if ki <= len(kk)-1 && rune(kk[ki]) == tr {
							lstphrn = kn
							continue
						}
						delete(crntrplcekeys, kn)
					}
					if len(crntrplcekeys) == 0 {
						lstphrn = -1
						lstphrsi = -1
						ki = 0
						mxkl = 0
						continue
					}
				foundmtch:
					if ki+1 == mxkl && len(crntrplcekeys) == 1 {
						clear(crntrplcekeys)
						mtchdprhase := rplcerrdr.rplcekeys[lstphrn]
						var preappendrunes []rune = nil
						if lstphrsi > 0 {
							preappendrunes = make([]rune, len(rnstst[:lstphrsi]))
							copy(preappendrunes, rnstst[:lstphrsi])
						}
						rplcerrdr.rmnrnsl = copy(rnstst, rnstst[lstphrsi+len(mtchdprhase):rnststL])
						if _, err = replacedWithReader(rplcerrdr, rplcerrdr.rplcewith, mtchdprhase, true, preappendrunes...); err != nil {
							return 0, 0, err
						}
						return underlineReadRune(rplcerrdr)
					}
					ki++
				}
				if lstphrn == -1 {
					rplcerrdr.rmnrnsl = 0
					rplcerrdr.crntrdr = strings.NewReader(string(rnstst[:rnststL]))
					return underlineReadRune(rplcerrdr)
				}
			}
		}*/
	}

	if size == 0 && err == nil {
		err = io.EOF
	}
	return
}

func (rplcerrdr *ReplaceRuneReader) ReadRune() (r rune, size int, err error) {
	if rplcerrdr != nil {
		r, size, err = underlineReadRune(rplcerrdr)
	}
	if size == 0 && err == nil {
		err = io.EOF
	}
	if err != nil {
		if onclose := rplcerrdr.OnClose; onclose != nil {
			rplcerrdr.OnClose = nil
			if err == io.EOF {
				if clserr := onclose(rplcerrdr, nil); clserr != nil {
					err = clserr
				}
			} else {
				if clserr := onclose(rplcerrdr, err); clserr != nil {
					err = clserr
				}
			}
		}
	}
	return
}

func (rplcerrdr *ReplaceRuneReader) PreAppend(rdrs ...io.RuneReader) {
	if rplcerrdr != nil && rplcerrdr.orgrdr != nil {
		rplcerrdr.orgrdr.PreAppend(rdrs...)
	}
}

func (rplcerrdr *ReplaceRuneReader) PostAppend(rdrs ...io.RuneReader) {
	if rplcerrdr != nil && rplcerrdr.orgrdr != nil {
		rplcerrdr.orgrdr.PostAppend(rdrs...)
	}
}

func (rplcerrdr *ReplaceRuneReader) Close() (err error) {
	if rplcerrdr != nil {
		if rplcerrdr.crntrdr != nil {
			rplcerrdr.crntrdr = nil
		}
		if rplcerrdr.orgrdr != nil {
			rplcerrdr.orgrdr.Close()
			rplcerrdr.orgrdr = nil
		}
		if len(rplcerrdr.rplcewith) > 0 {
			for rplck, rplv := range rplcerrdr.rplcewith {
				if rplv != "" {
					rplcerrdr.rplcewith[rplck] = ""
				}
				delete(rplcerrdr.rplcewith, rplck)
			}
		}
	}
	return
}

func (rplcerrdr *ReplaceRuneReader) Phrases() (phrases []string) {
	if rplcerrdr != nil {
		phrases = append(phrases, rplcerrdr.rplcekeys...)
	}
	return
}

func (rplcerrdr *ReplaceRuneReader) WriteTo(wtr io.Writer) (n int64, err error) {
	if rplcerrdr != nil && wtr != nil {
		if bfwtr, _ := wtr.(*Buffer); bfwtr != nil {
			for err == nil {
				if r, s, rerr := rplcerrdr.ReadRune(); s > 0 {
					bfwtr.WriteRune(r)
					n += int64(s)
					if rerr != nil {
						err = rerr
					}
				} else if rerr != nil {
					err = rerr
				}
			}
		}
	}
	return
}
