package iorw

import (
	"bufio"
	"io"
	"sort"
	"strings"
)

type ReplaceRuneReader struct {
	orgrdr    *RuneReaderSlice
	org, crnt bool
	crntrdr   io.RuneReader
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
	rplcerrdr = &ReplaceRuneReader{orgrdr: NewRuneReaderSlice(orgrnrdr)}
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
			sort.Strings(rplcerrdr.rplcekeys)
		}
	}
}

func replacedWithReader(rplcewith map[string]interface{}, phrase string, isrepeatable bool) (rdr io.RuneReader) {
	if len(rplcewith) > 0 && phrase != "" {
		if phrsev, phrsok := rplcewith[phrase]; phrsok {
			if phrss, phrssok := phrsev.(string); phrssok {
				rdr = strings.NewReader(phrss)
			} else if phrsr, _ := phrsev.(io.Reader); phrsr != nil {
				if rdr, _ = phrsr.(io.RuneReader); rdr == nil {
					rdr = bufio.NewReader(phrsr)
				}
			} else if phrsbf, _ := phrsev.(*Buffer); phrsbf != nil {
				if isrepeatable {
					rdr = phrsbf.Clone(true).Reader(true)
				} else {
					rdr = phrsbf.Reader()
				}
			}
		}
	}
	return
}

func underlineReadRune(rplcerrdr *ReplaceRuneReader) (r rune, size int, err error) {
	if rplcerrdr.crnt = (rplcerrdr.crntrdr != nil); rplcerrdr.crnt {
		r, size, err = rplcerrdr.crntrdr.ReadRune()
		if size == 0 && err == io.EOF {
			if rplcerrdr.crntrdr == rplcerrdr.orgrdr {
				rplcerrdr.orgrdr = nil
			}
			rplcerrdr.crntrdr = nil
			r, size, err = underlineReadRune(rplcerrdr)
			return
		}
		if err == io.EOF {
			rplcerrdr.crntrdr = nil
			if rplcerrdr.crntrdr == rplcerrdr.orgrdr {
				rplcerrdr.orgrdr = nil
				rplcerrdr.crnt = false
				rplcerrdr.org = false
			} else {
				err = nil
			}
			return
		}
	} else if rplcerrdr.org = (rplcerrdr.orgrdr != nil); rplcerrdr.org {
		if r, size, err = rplcerrdr.orgrdr.ReadRune(); size == 0 {
			rplcerrdr.orgrdr = nil
			rplcerrdr.org = false
		}
		return
	}
	if size == 0 {
		err = io.EOF
	}
	return
}

func (rplcerrdr *ReplaceRuneReader) ReadRune() (r rune, size int, err error) {
	if rplcerrdr != nil {
		if rplcerrdr.mxlrplcL == 0 {
			if rplcerrdr.orgrdr != nil {
				if r, size, err = rplcerrdr.orgrdr.ReadRune(); size == 0 && err == nil {
					err = io.EOF
				}
			}
			return
		}
		for err == nil {
			if r, size, err = underlineReadRune(rplcerrdr); rplcerrdr.crnt || (!rplcerrdr.crnt && !rplcerrdr.org) {
				return
			}
			if rplcerrdr.mxlrplcL > 0 && size > 0 && (rplcerrdr.org) {
				mtcdids := map[int]bool{}
				for rpi, rplk := range rplcerrdr.rplcekeys {
					if !mtcdids[rpi] {
						if rplk[0:1] == string(r) {
							mtcdids[rpi] = true
						}
					}
				}
				if len(mtcdids) == 0 {
					return
				}
				txtrns := make([]rune, rplcerrdr.mxlrplcL)
				for rplci := range rplcerrdr.mxlrplcL {
					txtrns[rplci] = r
					for rpi, rplk := range rplcerrdr.rplcekeys {
						if !mtcdids[rpi] {
							if len(rplk) > rplci && rplk[0:rplci+1] == string(txtrns[0:rplci+1]) {
								mtcdids[rpi] = true
							}
						} else if mtcdids[rpi] && len(rplk) >= (rplci+1) && rplk[0:rplci+1] != string(txtrns[0:rplci+1]) {
							delete(mtcdids, rpi)
						}
					}
					if len(mtcdids) == 0 {
						rplcerrdr.crntrdr = strings.NewReader(string(txtrns[:rplci+1]))
						clear(txtrns)
						r, size, err = underlineReadRune(rplcerrdr)
						return
					} else {
						if rplci < (rplcerrdr.mxlrplcL - 1) {
							if r, size, err = rplcerrdr.orgrdr.ReadRune(); size > 0 && (err == io.EOF || err == nil) {
								if err == io.EOF {
									err = nil
								}
							} else {
								break
							}
						} else {
							if len(mtcdids) == 1 {
								for fndid, _ := range mtcdids {
									rplcerrdr.crntrdr = replacedWithReader(rplcerrdr.rplcewith, rplcerrdr.rplcekeys[fndid], true)
									clear(txtrns)
									r, size, err = underlineReadRune(rplcerrdr)
									return
								}
							} else {
								rplcerrdr.crntrdr = strings.NewReader(string(txtrns[:rplci+1]))
								clear(txtrns)
								r, size, err = underlineReadRune(rplcerrdr)
								return
							}
						}
					}
				}
				continue
			}
			break
		}
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
