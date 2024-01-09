package iorw

import (
	"bufio"
	"io"
	"strings"
)

type ReplaceRuneReader struct {
	orgrdr    *RuneReaderSlice
	crntrdr   io.RuneReader
	rplcewith map[string]interface{}
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
			}
			if phrsl := len(sphrase); phrsl > rplcerrdr.mxlrplcL {
				rplcerrdr.mxlrplcL = phrsl
			}
			rplcerrdr.rplcewith[sphrase] = replacewith
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

func (rplcerrdr *ReplaceRuneReader) ReadRune() (r rune, size int, err error) {
	if rplcerrdr != nil {
		if rplcerrdr.crntrdr == nil && rplcerrdr.orgrdr != nil {
			if len(rplcerrdr.rplcewith) > 0 {
				tst := ""
				tstl := 0
				vldkeys := []string{}
				vldksi := 0
				vldksl := 0
				mxkl := 0
				for (rplcerrdr.orgrdr != nil) && rplcerrdr.crntrdr == nil {
					if rplcerrdr.orgrdr != nil {
						r, size, err = rplcerrdr.orgrdr.ReadRune()
					} else {
						r, size, err = 0, 0, io.EOF
					}
					if size > 0 {
						tst += string(r)
						tstl++
						if vldksl == 0 {
							for kv := range rplcerrdr.rplcewith {
								if kvl := len(kv); kv[:tstl] == tst {
									vldkeys = append(vldkeys, kv)
									vldksl++
									if mxkl < kvl {
										mxkl = kvl
									}
								}
							}
							if vldksl == 1 && tstl == mxkl {
								rplcerrdr.crntrdr = replacedWithReader(rplcerrdr.rplcewith, vldkeys[vldksi], true) // strings.NewReader(rplcerrdr.rplcewith[vldkeys[vldksi]])
							} else if vldksi == vldksl && rplcerrdr.crntrdr == nil {
								return r, size, nil
							}
						} else {
							vldksi = 0
							for rplcerrdr.crntrdr == nil && vldksi < vldksl {
								if vldkl := len(vldkeys[vldksi]); vldkeys[vldksi][:tstl] == tst {
									if vldkl == tstl {
										rplcerrdr.crntrdr = replacedWithReader(rplcerrdr.rplcewith, vldkeys[vldksi], true) // strings.NewReader(rplcerrdr.rplcewith[vldkeys[vldksi]])
										break
									} else if vldksl > 1 {
										vldksi++
									} else {
										break
									}
								} else {
									vldkeys = append(vldkeys[:vldksi], vldkeys[vldksi+1:]...)
									vldksl--
									if vldksl == 0 && rplcerrdr.crntrdr == nil {
										if tst[tstl-1:] != "" {
											rplcerrdr.orgrdr.PreAppend(strings.NewReader(tst[tstl-1:]))
										}
										rplcerrdr.crntrdr = strings.NewReader(tst[:tstl-1])
									}
								}
							}
						}
					}
					if err == io.EOF {
						rplcerrdr.orgrdr = nil
						if tst != "" {
							rplcerrdr.crntrdr = strings.NewReader(tst)
						}
					}
				}
			} else {
				rplcerrdr.crntrdr = rplcerrdr.orgrdr
			}
		}
		if rplcerrdr.crntrdr != nil {
			r, size, err = rplcerrdr.crntrdr.ReadRune()
			if size == 0 && err == io.EOF {
				if rplcerrdr.crntrdr == rplcerrdr.orgrdr {
					rplcerrdr.orgrdr = nil
				}
				rplcerrdr.crntrdr = nil
				r, size, err = rplcerrdr.ReadRune()
				return
			} else if err == io.EOF {
				rplcerrdr.crntrdr = nil
				if rplcerrdr.crntrdr == rplcerrdr.orgrdr {
					rplcerrdr.orgrdr = nil
				} else {
					err = nil
				}
				return
			}
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
