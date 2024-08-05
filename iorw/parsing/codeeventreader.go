package parsing

import (
	"io"
	"strings"

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
	cmntevtrdrs        map[string]*commentevtreader
	cmntrdrfound       *commentevtreader
	ValidElemEvent     func(elemname string) bool
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
		if cmntrdrfound := cdeevtrdr.cmntrdrfound; cmntrdrfound == cmntevtrdr {
			cdeevtrdr.cmntrdrfound = nil
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
	cdeevtrdr = &codeeventreader{ParseEventReader: newParseEventReader(prelabel, postlabel, rnrdrs...), cmntevtrdrs: map[string]*commentevtreader{}}

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

	var crntevtrdrs [][]rune = nil
	var crntkrns []rune = nil
	var crntktsti = 1
	var cdetxt = rune(0)
	var lstcdetxt = rune(0)
	var prvcder = rune(0)
	var cdepostrunsevt func(rnsl int, rns ...rune) (rnserr error) = nil
	var cmntrdrfound *commentevtreader = nil
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
		cmntrdrfound = cdeevtrdr.cmntrdrfound
		if cdepostrunsevt == nil {
			cdepostrunsevt = cdeevtrdr.CodePostRunesEvent
		}
		cmntevtrdrs := cdeevtrdr.cmntevtrdrs
		if len(cmntevtrdrs) > 0 {
		redo:
			for rn, r := range rns {
				if cdetxt > 0 {
					if cdetxt == r && prvcder != '\\' {
						cdetxt = 0
						lstcdetxt = 0
					}
				} else if cdetxt == 0 {
					if prvcder != '\\' && iorw.IsTxtPar(r) {
						cdetxt = r
						lstcdetxt = r
					}
				}
				prvcder = r
				if cmntrdrfound != nil {
					lststge := cmntrdrfound.ParseStage()
					rnserr = cmntrdrfound.parseRune(r)
					if lststge != cmntrdrfound.ParseStage() {
						if cmntrdrfound.lbli[1] > 0 {

						}
						cmntrdrfound = cdeevtrdr.cmntrdrfound
						rns = append(rns[:rn], rns[rn+1:]...)
						if rnsl = len(rns); rnsl == 0 {
							return
						}
						goto redo
					}
					continue
				}
				if cdetxt == 0 && r != lstcdetxt {
					if len(crntevtrdrs) == 0 {
						for cmntk := range cmntevtrdrs {
							if cmtktst := []rune(cmntk); cmtktst[0] == r {
								crntevtrdrs = append(crntevtrdrs, cmtktst)
								if len(crntkrns) < crntktsti {
									crntkrns = append(crntkrns, r)
								}
							}
						}
						if len(crntevtrdrs) > 0 {
							rns = rns[rn+1:]
							if rnsl = len(rns); rnsl == 0 {
								return
							}
							goto redo
						}
						continue
					}
				recheck:
					fndval := false
					for cmkn, cmntrnsk := range crntevtrdrs {
						if cmntkl := len(cmntrnsk); cmntkl >= len(crntkrns) {
							if cmntrnsk[crntktsti] == r {
								if !fndval {
									fndval = true
								}
								if len(crntkrns) <= crntktsti {
									crntkrns = append(crntkrns, r)
								}
								if len(crntevtrdrs) == 1 {
									crntks, crntktst := string(crntevtrdrs[cmkn]), string(crntkrns)
									if strings.HasPrefix(crntks, crntktst) {
										if len(crntks) == len(crntktst) {
											if crntks == crntktst {
												if cdeevtrdr.cmntrdrfound == nil {
													crntevtrdrs = append(crntevtrdrs[:cmkn], crntevtrdrs[cmkn+1:]...)
													cmntrdrfound = cdeevtrdr.cmntevtrdrs[crntks]
													cmntrdrfound.SwapParseState()
													cdeevtrdr.cmntrdrfound = cmntrdrfound
													crntktsti = 1
													crntkrns = nil
													rns = append(rns[:rn], rns[rn+1:]...)
													if rnsl = len(rns); rnsl == 0 {
														return
													}
													goto redo
												}
											}
											crntkrns = crntkrns[:len(crntkrns)-1]
											crntevtrdrs = append(crntevtrdrs[:cmkn], crntevtrdrs[cmkn+1:]...)
											goto nomore
										}
										crntktsti++
										fndval = false
										rns = append(rns[:rn], rns[rn+1:]...)
										if rnsl = len(rns); rnsl == 0 {
											return
										}
										goto redo
									}
									crntkrns = crntkrns[:len(crntkrns)-1]
									crntevtrdrs = append(crntevtrdrs[:cmkn], crntevtrdrs[cmkn+1:]...)
									goto nomore
								}
								continue
							}
						}
						crntevtrdrs = append(crntevtrdrs[:cmkn], crntevtrdrs[cmkn+1:]...)
						goto recheck
					}
					if fndval {
						crntktsti++
					}
				nomore:
					crntevtrdrsl := len(crntevtrdrs)
					if crntevtrdrsl == 0 {
						crntktsti = 1
						fndval = false
						rns = append(rns[:rn], rns[rn+1:]...)
						if len(crntkrns) > 0 {
							crntkrns = append(crntkrns, r)
							if cdepostrunsevt != nil {
								if rnserr = cdepostrunsevt(len(crntkrns), crntkrns...); rnserr != nil {
									return
								}
							}
							crntkrns = nil
						}
						if rnsl = len(rns); rnsl == 0 {
							return
						}
					}
				}
				if lstcdetxt > 0 {
					lstcdetxt = 0
				}
			}
		}
		if rnsl > 0 {
			if cmntrdrfound == nil && cdepostrunsevt != nil {
				rnserr = cdepostrunsevt(rnsl, rns...)
			}
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
		canpostcontent := false
		var cmmntpostevent PostCommentEventFunc = nil
		cmntevtrdrs := cdeevtrdr.cmntevtrdrs
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
				if canpostcontent {
					cdecmntevtr.postcmnt = canpostcontent
					canpostcontent = false
				}
				if cmmntpostevent != nil {
					cdecmntevtr.postcmntevt = cmmntpostevent
					cmmntpostevent = nil
				}
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
					if cmntevtrdrs == nil {
						cmntevtrdrs = map[string]*commentevtreader{}
						cdeevtrdr.cmntevtrdrs = cmntevtrdrs
					}
					cmntevtrdrs[prelbl] = newCmntEvtReader(cdeevtrdr, prelbl, postlbl, cmmntpostevent, canpostcontent)
					if canpostcontent {
						canpostcontent = false
					}
					if cmmntpostevent != nil {
						cmmntpostevent = nil
					}
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
