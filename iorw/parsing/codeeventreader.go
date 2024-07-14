package parsing

import (
	"io"

	"github.com/evocert/lnksnk/iorw"
)

type codeeventreader struct {
	*ParseEventReader
	hsecde             bool
	FoundCode          bool
	CodePreResetEvent  func(foundcode bool, prel, postl int, prelbl, postlbl []rune, lbli []int) (rseterr error)
	CodeFoundEvent     func(bool) error
	CodePreRunesEvent  func(foundcode bool, rnsl int, rns ...rune) (rnserr error)
	CodePostRunesEvent func(rnsl int, rns ...rune) (rnserr error)
	cmmntevtrdrs       []*commentevtreader
	cmntrdx            int
	cmntrdrsmap        map[int]*commentevtreader
	cmntrdrfound       *commentevtreader
}

type commentevtreader struct {
	*ParseEventReader
	cdeevtrdr   *codeeventreader
	cmntbuf     *iorw.Buffer
	postcmnt    bool
	postcmntevt PostCommentEventFunc
}

func newCmntEvtReader(cdeevtrdr *codeeventreader, prelbl, postlbl string, cmmntpostevent PostCommentEventFunc, canpost bool) (cmntevtrdr *commentevtreader) {
	cmntevtrdr = &commentevtreader{cdeevtrdr: cdeevtrdr, ParseEventReader: newParseEventReader(prelbl, postlbl), postcmnt: canpost, postcmntevt: cmmntpostevent}
	cmntevtrdr.CanPreParse = cmntevtrdr.canPreParse
	cmntevtrdr.PreResetEvent = cmntevtrdr.preResetEvent
	cmntevtrdr.PostRunesEvent = cmntevtrdr.postRunesEvent
	cmntevtrdr.PostResetEvent = cmntevtrdr.postResetEvent
	return
}

func (cmntevtrdr *commentevtreader) canPreParse() bool {
	if cmntevtrdr.ParseStage() == PostStage {
		return false
	}
	if cdeevtrdr := cmntevtrdr.cdeevtrdr; cdeevtrdr != nil {
		return cdeevtrdr.cmntrdrfound == nil && cdeevtrdr.PostTxtr == 0
	}
	return false
}

func (cmntevtrdr *commentevtreader) preResetEvent(prel, postl int, prelbl, postlbl []rune, lbli []int) (reseterr error) {
	cmntbuf, cdeevtrdr := cmntevtrdr.cmntbuf, cmntevtrdr.cdeevtrdr
	if !cmntbuf.Empty() {
		cmntbuf.Clear()
	}
	if cdeevtrdr != nil {
		if cmntrdrsmap := cdeevtrdr.cmntrdrsmap; cmntrdrsmap != nil {
			for cmix, cmntrdr := range cmntrdrsmap {
				delete(cmntrdrsmap, cmix)
				if cmntrdr == cmntevtrdr {
					cdeevtrdr.cmntrdx = cmix
					continue
				}
				cmntrdr.resetPre(true)
				cmntrdr.resetPost(true)
			}
		}
		cdeevtrdr.cmntrdrfound = cmntevtrdr
	}
	return
}

func (cmntevtrdr *commentevtreader) postResetEvent(prel, postl int, prelbl, postlbl []rune, lbli []int) (reseterr error) {
	cmntbuf, cdeevtrdr := cmntevtrdr.cmntbuf, cmntevtrdr.cdeevtrdr

	if !cmntbuf.Empty() {
		defer cmntbuf.Clear()
		if cdeevtrdr != nil {
			if cmntevtrdr.postcmnt {
				if cdepostrunsevt := cdeevtrdr.CodePostRunesEvent; cdepostrunsevt != nil {
					if reseterr = cdepostrunsevt(prel, prelbl...); reseterr != nil {
						return
					}
					mxint := 4096
					if cnmtbufs := cmntbuf.Size(); cnmtbufs < int64(mxint) {
						mxint = int(cnmtbufs)
					}
					prns := make([]rune, mxint)
					prn := 0
					cmntbfr := cmntbuf.Reader()
					defer cmntbfr.Close()
					for reseterr == nil {
						if prn, reseterr = iorw.ReadRunes(prns, cmntbfr); prn > 0 && (reseterr == nil || reseterr == io.EOF) {
							if reseterr = cdepostrunsevt(prn, prns[:prn]...); reseterr != nil {
								return
							}
							if reseterr == nil {
								continue
							}
							if reseterr == io.EOF {
								reseterr = nil
								break
							}
						}
					}
					if reseterr != nil {
						if reseterr != io.EOF {
							return
						}
						reseterr = nil
					}
					if reseterr = cdepostrunsevt(postl, postlbl...); reseterr != nil {
						return
					}
				}
			}
			if postcmntevt := cmntevtrdr.postcmntevt; postcmntevt != nil {
				reseterr = postcmntevt(cmntbuf, prelbl, postlbl)
			}
		}
	}
	if cdeevtrdr != nil {
		if cmntrdrfound := cdeevtrdr.cmntrdrfound; cmntrdrfound == cmntevtrdr && cdeevtrdr.cmmntevtrdrs[cdeevtrdr.cmntrdx] == cmntevtrdr {
			cdeevtrdr.cmntrdrfound = nil
			cdeevtrdr.cmntrdx = -1
		}
	}
	return
}

func (cmntevtrdr *commentevtreader) postRunesEvent(resetlbl bool, rnsl int, rns ...rune) (rnserr error) {
	cmntbuf := cmntevtrdr.cmntbuf
	if cmntbuf == nil {
		cmntbuf = iorw.NewBuffer()
		cmntevtrdr.cmntbuf = cmntbuf
	}
	cmntbuf.WriteRunes(rns...)
	return
}

func newCodeEventReader(prelabel, postlabel string, rnrdrs ...io.RuneReader) (cdeevtrdr *codeeventreader) {
	cdeevtrdr = &codeeventreader{ParseEventReader: newParseEventReader(prelabel, postlabel, rnrdrs...), cmntrdrsmap: map[int]*commentevtreader{}, cmntrdx: -1}

	cdeevtrdr.PostCanResetTextPar = func(prevr, r rune) bool {
		return (cdeevtrdr.cmntrdrfound == nil || cdeevtrdr.cmntrdrfound.ParseStage() == PreStage) && prevr != '\\' && cdeevtrdr.PostTxtr == r
	}
	cdeevtrdr.PostCanSetTextPar = func(prevr, r rune) (set bool) {
		return (cdeevtrdr.cmntrdrfound == nil || cdeevtrdr.cmntrdrfound.ParseStage() == PreStage) && prevr != '\\' && iorw.IsTxtPar(r)
	}
	cdeevtrdr.PreRunesEvent = func(resetlbl bool, rnsl int, rns ...rune) (prerr error) {
		if cdeevtrdr.hsecde {
			cdeevtrdr.hsecde = false
		}
		if cdePreRunseEvent := cdeevtrdr.CodePreRunesEvent; cdePreRunseEvent != nil {
			prerr = cdePreRunseEvent(cdeevtrdr.FoundCode, rnsl, rns...)
		}
		return
	}

	cdeevtrdr.PostRunesEvent = func(resetlbl bool, rnsl int, rns ...rune) (rnserr error) {
		if !cdeevtrdr.hsecde {
			cdeevtrdr.hsecde = true
			if cdefndcodeevt := cdeevtrdr.CodeFoundEvent; cdefndcodeevt != nil {
				if rnserr = cdefndcodeevt(cdeevtrdr.FoundCode); rnserr != nil {
					return
				}
			}
			if !cdeevtrdr.FoundCode {
				cdeevtrdr.FoundCode = true
			}
		}
		cdepostrunsevt := cdeevtrdr.CodePostRunesEvent
		if cmmntevtrdrs := cdeevtrdr.cmmntevtrdrs; len(cmmntevtrdrs) > 0 {
		fndrdr:
			if cdeevtrdr.cmntrdrfound != nil {
				for rn, r := range rns {
					if rnserr = cdeevtrdr.cmntrdrfound.parseRune(r); rnserr != nil {
						return
					}
					if cdeevtrdr.cmntrdrfound == nil {
						rns = rns[rn+1:]
						if rnsl := len(rns); rnsl == 0 {
							return
						}
						goto unmtchdrdr
					}
					continue
				}
				return
			}
		matchdrdr:
			if len(cdeevtrdr.cmntrdrsmap) > 0 {
				for rn, r := range rns {
					for idx, vcmntevtr := range cdeevtrdr.cmntrdrsmap {
						lbli := vcmntevtr.lbli[0]
						if rnserr = vcmntevtr.parseRune(r); rnserr != nil {
							return
						}
						if cdeevtrdr.cmntrdrfound != nil {
							rns = rns[rn+1:]
							if rnsl := len(rns); rnsl == 0 {
								return
							}
							goto fndrdr
						}
						if vcmntevtr.lbli[0] == 0 {
							delete(cdeevtrdr.cmntrdrsmap, idx)
							vcmntevtr.resetPre(true)
							vcmntevtr.resetPost(true)
							if len(cdeevtrdr.cmntrdrsmap) == 0 {
								if lbli > 0 {
									if cdepostrunsevt != nil {
										if rnserr = cdepostrunsevt(lbli, vcmntevtr.prelbl[:lbli]...); rnserr != nil {
											return
										}
									}
								}
								rns = rns[rn:]
								if rnsl := len(rns); rnsl == 0 {
									return
								}
								goto unmtchdrdr
							}
						}
					}
				}
				if len(cdeevtrdr.cmntrdrsmap) > 0 {
					return
				}
			}
		unmtchdrdr:

			for rn, r := range rns {
				for idx, vcmntevtr := range cmmntevtrdrs {
					if cdeevtrdr.cmntrdrsmap[idx] == nil {
						if rnserr = vcmntevtr.parseRune(r); rnserr != nil {
							return
						}
						if cdeevtrdr.cmntrdrfound != nil {
							rns = rns[rn+1:]
							if rnsl := len(rns); rnsl == 0 {
								return
							}
							goto fndrdr
						}
						if vcmntevtr.lbli[0] > 0 && cdeevtrdr.cmntrdrsmap[idx] == nil {
							cdeevtrdr.cmntrdrsmap[idx] = vcmntevtr
						}
					}
				}
				if len(cdeevtrdr.cmntrdrsmap) > 0 {
					rns = rns[rn+1:]
					if rnsl = len(rns); rnsl == 0 {
						return
					}
					goto matchdrdr
				}
			}
		}
		if cdepostrunsevt != nil {
			rnserr = cdepostrunsevt(rnsl, rns...)
		}
		return
	}

	cdeevtrdr.PreResetEvent = func(prel, postl int, prelbl, postlbl []rune, lbli []int) (reseterr error) {
		if cdepresetevt := cdeevtrdr.CodePreResetEvent; cdepresetevt != nil {
			reseterr = cdepresetevt(cdeevtrdr.FoundCode, prel, postl, prelbl, postlbl, lbli)
		}
		return
	}
	cdeevtrdr.PostResetEvent = func(prel, postl int, prelbl, postlbl []rune, lbli []int) (rseterr error) {
		if cdeevtrdr.hsecde {
			cdeevtrdr.hsecde = false
		}
		return
	}

	return
}

type PostCommentEventFunc func(imprtbuf *iorw.Buffer, prelbl, postlbl []rune) (poserr error)

func (cdeevtrdr *codeeventreader) AddCommentsEventReader(a ...interface{}) {
	if cdeevtrdr != nil {
		al := len(a)
		cmntsl := len(cdeevtrdr.cmmntevtrdrs)
		canpostcontent := false
		var cmmntpostevent PostCommentEventFunc = nil
		for al > 0 {
			if canpostd, canpstdok := a[0].(bool); canpstdok {
				a = a[1:]
				al--
				if canpostd {
					canpostcontent = true
				}
				continue
			}
			if cmmntposteventd, cmmntposteventdok := a[0].(PostCommentEventFunc); cmmntposteventdok {
				a = a[1:]
				al--
				if cmmntposteventd != nil {
					if cmmntpostevent == nil {
						cmmntpostevent = cmmntposteventd
					}
				}
				continue
			}
			if cmmntposteventd, cmmntposteventdok := a[0].(func(*iorw.Buffer, string, string) error); cmmntposteventdok {
				a = a[1:]
				al--
				if cmmntposteventd != nil {
					if cmmntpostevent == nil {
						cmmntpostevent = func(imprtbuf *iorw.Buffer, prelbl, postlbl []rune) (poserr error) {
							return cmmntposteventd(imprtbuf, string(prelbl), string(postlbl))
						}
					}
				}
				continue
			}
			if cmmntposteventd, cmmntposteventdok := a[0].(func(*iorw.Buffer, []rune, []rune) error); cmmntposteventdok {
				a = a[1:]
				al--
				if cmmntposteventd != nil {
					if cmmntpostevent == nil {
						cmmntpostevent = func(imprtbuf *iorw.Buffer, prelbl, postlbl []rune) (poserr error) {
							return cmmntposteventd(imprtbuf, prelbl, postlbl)
						}
					}
				}
				continue
			}
			if cdecmntevtr, _ := a[0].(*commentevtreader); cdecmntevtr != nil {
				cdeevtrdr.cmmntevtrdrs = append(cdeevtrdr.cmmntevtrdrs, cdecmntevtr)
				if canpostcontent {
					cdecmntevtr.postcmnt = canpostcontent
					canpostcontent = false
				}
				if cmmntpostevent != nil {
					cdecmntevtr.postcmntevt = cmmntpostevent
					cmmntpostevent = nil
				}
				cmntsl++
				a = a[1:]
				al--
				continue
			}
			if prelbl, _ := a[0].(string); prelbl != "" {
				a = a[1:]
				al--
				if al == 0 {
					break
				}
				if postlbl := a[0].(string); postlbl != "" {
					a = a[1:]
					al--
					cdeevtrdr.cmmntevtrdrs = append(cdeevtrdr.cmmntevtrdrs, newCmntEvtReader(cdeevtrdr, prelbl, postlbl, cmmntpostevent, canpostcontent))
					if canpostcontent {
						canpostcontent = false
					}
					if cmmntpostevent != nil {
						cmmntpostevent = nil
					}
					cmntsl++
					continue
				}
				continue
			}
			if canpostcontent {
				canpostcontent = false
			}
			if cmmntpostevent != nil {
				cmmntpostevent = nil
			}
			a = a[1:]
			al--
		}
	}
}

/*func prepCodeCommentReader(cdeevtrdr *codeeventreader, cdecmntix int, cdecmntrdr *commentevtreader) *commentevtreader {
	return cdecmntrdr
}*/
