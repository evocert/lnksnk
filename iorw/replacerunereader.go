package iorw

import (
	"bufio"
	"io"
	"sort"
	"strings"
)

type ReplaceRuneReader struct {
	rnstst    []rune
	rmnrnsl   int
	orgrdr    *RuneReaderSlice
	crntrdr   io.RuneReader
	crntrns   []rune
	crntrnsl  int
	rplcewith map[string]interface{}
	rplcekeys []string
	mxlrplcL  int
	OnClose   func(*ReplaceRuneReader, error) (err error)
}

func NewReplaceRuneReader(orgrdr interface{}, rplwiths ...interface{}) (rplcerrdr *ReplaceRuneReader) {
	orgrnrdr, _ := orgrdr.(io.RuneReader)
	if orgrnrdr == nil {
		if orgr, _ := orgrdr.(io.Reader); orgr != nil {
			orgrnrdr = bufio.NewReaderSize(orgr, 1)
		}
	}
	rplcerrdr = &ReplaceRuneReader{orgrdr: NewRuneReaderSlice(orgrnrdr), rnstst: make([]rune, 8192)}
	rplwithsl := len(rplwiths)
	for rplwithsl > 0 && rplwithsl%2 == 0 {
		rplcerrdr.ReplaceWith(rplwiths[0], rplwiths[1])
		rplwiths = rplwiths[2:]
		rplwithsl -= 2
	}
	return
}

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

func replacedWithReader(rplcerrdr *ReplaceRuneReader, rplcewith map[string]interface{}, phrase string, isrepeatable bool, rnstopreappend ...rune) bool {
	if len(rplcewith) > 0 && phrase != "" {
		var preappndrdr io.RuneReader = nil
		if len(rnstopreappend) > 0 {
			preappndrdr = strings.NewReader(string(rnstopreappend))
		}
		if phrsev, phrsok := rplcewith[phrase]; phrsok {
			if phrss, phrssok := phrsev.(string); phrssok && phrss != "" {
				if preappndrdr != nil {
					rplcerrdr.crntrdr = NewRuneReaderSlice(preappndrdr, strings.NewReader(phrss))
					return true
				}
				rplcerrdr.crntrdr = strings.NewReader(phrss)
				return true
			}
			if phrsr, _ := phrsev.(io.Reader); phrsr != nil {
				if rdr, _ := phrsr.(io.RuneReader); rdr == nil {
					if preappndrdr != nil {
						rplcerrdr.crntrdr = NewRuneReaderSlice(preappndrdr, bufio.NewReader(phrsr))
						return true
					}
					rplcerrdr.crntrdr = bufio.NewReader(phrsr)
					return true
				}
				if preappndrdr != nil {
					rplcerrdr.crntrdr = NewRuneReaderSlice(preappndrdr, bufio.NewReader(phrsr))
					return true
				}
				rplcerrdr.crntrdr, _ = phrsr.(io.RuneReader)
				return true
			}
			if phrsbf, _ := phrsev.(*Buffer); phrsbf != nil {
				if isrepeatable {
					if preappndrdr != nil {
						rplcerrdr.crntrdr = NewRuneReaderSlice(preappndrdr, phrsbf.Clone(true).Reader(true))
						return true
					}
					rplcerrdr.crntrdr = phrsbf.Clone(true).Reader(true)
					return true
				}
				if preappndrdr != nil {
					rplcerrdr.crntrdr = NewRuneReaderSlice(preappndrdr, phrsbf.Reader())
					return true
				}
				rplcerrdr.crntrdr = phrsbf.Reader()
				return true
			}
		}
	}
	return false
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
	if orgrdr, rmngl := rplcerrdr.orgrdr, rplcerrdr.rmnrnsl; orgrdr != nil || rmngl > 0 {
		var rnstst []rune = rplcerrdr.rnstst
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
				lstphrn := -1
				lstphrsi := -1
				for prsn, prsphrase := range rplcerrdr.rplcekeys {
					if prsn > lstphrn {
						if phrsi := IndexOfRunes(rnstst[:rnststL], []rune(prsphrase)...); (phrsi > -1 && lstphrsi == -1) || (phrsi > -1 && phrsi < lstphrsi) {
							subphrsn := -1
							for subn, sbphrs := range rplcerrdr.rplcekeys[prsn+1:] {
								if !RunesHasPrefix([]rune(sbphrs), []rune(prsphrase)...) {
									if sbphrsi := IndexOfRunes(rnstst[:rnststL], []rune(sbphrs)...); sbphrsi < phrsi && (sbphrsi > -1 && lstphrsi == -1) || (sbphrsi > -1 && phrsi < lstphrsi) {
										lstphrsi = sbphrsi
										lstphrn = subn + prsn + 1
										subphrsn = subn
										break
									}
									continue
								}
								if sbphrsi := IndexOfRunes(rnstst[:rnststL], []rune(sbphrs)...); sbphrsi < phrsi && (sbphrsi > -1 && lstphrsi == -1) || (sbphrsi > -1 && phrsi < lstphrsi) {
									lstphrsi = sbphrsi
									lstphrn = subn + prsn + 1
									subphrsn = subn
									break
								}
							}
							if subphrsn > -1 {
								continue
							}
							if lstphrsi == -1 || lstphrsi > phrsi {
								lstphrsi = phrsi
								lstphrn = prsn
							}
						}
					}
				}
				if lstphrn == -1 {
					rplcerrdr.rmnrnsl = 0
					rplcerrdr.crntrdr = strings.NewReader(string(rnstst[:rnststL]))
					//rplcerrdr.crntrns = append(rplcerrdr.crntrns, rnstst[:rnststL]...)
					//rplcerrdr.crntrnsl += rnststL
					return underlineReadRune(rplcerrdr)
				}
				mtchdprhase := rplcerrdr.rplcekeys[lstphrn]
				var preappendrunes []rune = nil
				if lstphrsi > 0 {
					//rplcerrdr.crntrns = append(rplcerrdr.crntrns, rnstst[:lstphrsi]...)
					//rplcerrdr.crntrnsl += len(rnstst[:lstphrsi])
					preappendrunes = make([]rune, len(rnstst[:lstphrsi]))
					copy(preappendrunes, rnstst[:lstphrsi])
				}
				rplcerrdr.rmnrnsl = copy(rnstst, rnstst[lstphrsi+len(mtchdprhase):rnststL])
				replacedWithReader(rplcerrdr, rplcerrdr.rplcewith, mtchdprhase, true, preappendrunes...)
				return underlineReadRune(rplcerrdr)
			}
		}
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
