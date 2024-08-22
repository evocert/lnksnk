package parsing

import (
	"io"

	"github.com/lnksnk/lnksnk/iorw"
)

type contenteventreader struct {
	*ParseEventReader
	ValidElemEvent func(elmlvl ctntelemlevel, elnname string, elmbuf *iorw.Buffer, elmargs *contentargsreader) (evtvalid bool, vlerr error)
}

const (
	ctntElemUnknown ctntelemlevel = iota
	ctntElemStart
	ctntElemSingle
	ctntElemEnd
)

func (ctntevtrdr *contenteventreader) flushRunes(rns ...rune) (flsherr error) {
	flsherr = ctntevtrdr.parsePreRunes(false, rns...)
	return
}

func newContentEventReader(prelabel, postlabel string, rnrdrs ...io.RuneReader) (ctntevtrdr *contenteventreader) {
	ctntevtrdr = &contenteventreader{ParseEventReader: newParseEventReader(prelabel, postlabel, rnrdrs...)}
	ctntevtrdr.PostCanResetTextPar = func(prevr, r rune) bool {
		return prevr != '\\' && ctntevtrdr.PostTxtr == r
	}
	ctntevtrdr.PostCanSetTextPar = func(prevr, r rune) (set bool) {
		return prevr != '\\' && iorw.IsTxtPar(r)
	}
	var argsr *contentargsreader = nil
	var argsrdr = func() *contentargsreader {
		if argsr == nil {
			argsr = newContentArgsReader("[$", "$]", ctntevtrdr)
		}
		return argsr
	}
	var elmfndname = false
	//var elmtxtr = rune(0)
	var crntelmlvl ctntelemlevel = ctntElemUnknown
	var elmrmngbuf *iorw.Buffer = nil
	var crntelmname []rune = nil

	elmRmngBuffer := func() *iorw.Buffer {
		if elmrmngbuf == nil {
			elmrmngbuf = iorw.NewBuffer()
		}
		return elmrmngbuf
	}

	ctntevtrdr.PreResetEvent = func(prel, postl int, prelbl, postlbl []rune, lbli []int) (rseterr error) {
		return
	}

	elmflushInvalid := func(flsrunsfunc func(...rune) error, canprse bool, prelbl, postlbl []rune, lbli []int, r ...rune) (flsherr error) {
		if crntelmlvl == ctntElemStart {
			crntelmlvl = ctntElemUnknown
		}

		if canprse {
			var flsrmng []rune = nil
			if elmfndname {
				elmfndname = false
				if lbli[0] > 0 {
					flsrmng = append(flsrmng, prelbl[:lbli[0]]...)
					lbli[0] = 0
				}
				if crntelmlvl == ctntElemEnd {
					crntelmlvl = ctntElemUnknown
					flsrmng = append(flsrmng, '/')
				}
				if len(crntelmname) > 0 {
					flsrmng = append(flsrmng, crntelmname...)
					crntelmname = nil
				}
				if !elmrmngbuf.Empty() {
					if len(flsrmng) > 0 {
						if flsherr = flsrunsfunc(flsrmng...); flsherr != nil {
							return
						}
						flsrmng = nil
					}
					flsrmng = append(flsrmng, []rune(elmrmngbuf.String())...)
					elmrmngbuf.Clear()
				}
				if crntelmlvl == ctntElemSingle {
					crntelmlvl = ctntElemUnknown
					flsrmng = append(flsrmng, '/')
				}
				if lbli[1] > 0 {
					flsrmng = append(flsrmng, postlbl[:lbli[1]]...)
					lbli[1] = 0
				}
				if len(flsrmng) > 0 {
					if flsherr = flsrunsfunc(flsrmng...); flsherr != nil {
						return
					}
					flsrmng = nil
				}
				return
			}
			if lbli[0] > 0 {
				flsrmng = append(flsrmng, prelbl[:lbli[0]]...)
				lbli[0] = 0
			}
			if crntelmlvl == ctntElemEnd {
				crntelmlvl = ctntElemUnknown
				flsrmng = append(flsrmng, '/')
			}
			if len(crntelmname) > 0 {
				flsrmng = append(flsrmng, crntelmname...)
				crntelmname = nil
			}
			flsrmng = append(flsrmng, r...)
			if lbli[1] > 0 {
				flsrmng = append(flsrmng, postlbl[:lbli[1]]...)
				lbli[1] = 0
			}
			if len(flsrmng) > 0 {
				if flsherr = flsrunsfunc(flsrmng...); flsherr != nil {
					return
				}
				flsrmng = nil
			}
			return
		}
		ctntevtrdr.resetPre(true)
		ctntevtrdr.resetPost(true)
		elmfndname = false
		crntelmlvl = ctntElemUnknown
		crntelmname = nil
		elmrmngbuf.Clear()
		return
	}

	var postRune func(prvr, r rune, prelbl []rune, postlbl []rune, lbli []int) (invld bool, nvlerr error) = nil
	postRune = func(prvr, r rune, prelbl []rune, postlbl []rune, lbli []int) (invld bool, inlverr error) {
		if elmfndname {
			if ctntevtrdr.PostTxtr == 0 {
				if r == '/' {
					if crntelmlvl != ctntElemSingle {
						crntelmlvl = ctntElemSingle
						return
					}
					invld = true
					inlverr = elmflushInvalid(ctntevtrdr.flushRunes, true, prelbl, postlbl, lbli, r)
					return
				}
				if iorw.IsSpace(r) {
					if crntelmlvl == ctntElemSingle {
						invld = true
						inlverr = elmflushInvalid(ctntevtrdr.flushRunes, true, prelbl, postlbl, lbli, r)
						return
					}
				}
				if prvr != '\\' && iorw.IsTxtPar(r) {
					ctntevtrdr.PostTxtr = r
				}
			}
			elmRmngBuffer().WriteRune(r)
			return
		}
		if r == '/' {
			if crntelmlvl != ctntElemEnd {
				if len(crntelmname) == 0 {
					crntelmlvl = ctntElemEnd
					ctntevtrdr.prevr = 0
					return
				}
				elmfndname = true
				invld, inlverr = postRune(prvr, r, prelbl, postlbl, lbli)
				return
			}
			invld = true
			inlverr = elmflushInvalid(ctntevtrdr.flushRunes, true, prelbl, postlbl, lbli, r)
			return
		}
		if iorw.IsSpace(r) {
			if invld = len(crntelmname) == 0; invld {
				inlverr = elmflushInvalid(ctntevtrdr.flushRunes, true, prelbl, postlbl, lbli, r)
				return
			}
			if crntelmlvl != ctntElemEnd {
				elmfndname = true
				elmRmngBuffer().WriteRune(r)
				return
			}
			invld = true
			inlverr = elmflushInvalid(ctntevtrdr.flushRunes, true, prelbl, postlbl, lbli, r)
			return
		}

		if invld = !validElemChar(func() rune {
			if crntelmlvl == ctntElemEnd && prvr == '/' {
				return 0
			}
			return prvr
		}(), r); !invld {
			crntelmname = append(crntelmname, r)
			return
		}
		inlverr = elmflushInvalid(ctntevtrdr.flushRunes, true, prelbl, postlbl, lbli, r)
		return
	}
	ctntevtrdr.PostResetEvent = func(prel, postl int, prelbl, postlbl []rune, lbli []int) (rseterr error) {
		if elmfnd, elemname, fndelmlvl := elmfndname || len(crntelmname) > 0, string(crntelmname), func() ctntelemlevel {
			if crntelmlvl == ctntElemUnknown {
				return ctntElemStart
			}
			return crntelmlvl
		}(); elmfnd && elemname != "" {
			vldelmevt := ctntevtrdr.ValidElemEvent
			if vldelmevt == nil {
				if rseterr = elmflushInvalid(ctntevtrdr.flushRunes, true, prelbl, postlbl, lbli); rseterr != nil {
					return
				}
				return
			}
			vldelmbuf := elmrmngbuf.Clone(true)
			defer vldelmbuf.Close()
			elmflushInvalid(ctntevtrdr.flushRunes, false, prelbl, postlbl, lbli)
			if !vldelmbuf.Empty() {
				argsrdr().PreAppend(vldelmbuf.Reader())
				argrdrstg := argsrdr().ParseStage()
				if rseterr = argsrdr().DummyEOFRead(); rseterr != nil {
					return
				}
				if argrdrstg != argsrdr().ParseStage() {
					argsr.Close()
					argsr = nil
				}
				if fndelmlvl == ctntElemSingle || fndelmlvl == ctntElemSingle {
					argsrdr().savearg()
				}
			}

			vld, vlderr := vldelmevt(fndelmlvl, elemname, vldelmbuf, argsr)
			if argsr != nil {
				func() {
					argsr.Close()
					argsr = nil
				}()
			}
			if vlderr != nil {
				return
			}
			if vld {
				return
			}
			if !vldelmbuf.Empty() {
				vldelmbuf.WriteTo(elmRmngBuffer())
			}
			lbli[0] = len(postlbl)
			lbli[1] = len(postlbl)
			elmfndname = elmfnd
			crntelmname = []rune(elemname)
			crntelmlvl = fndelmlvl
		}
		rseterr = elmflushInvalid(ctntevtrdr.flushRunes, true, prelbl, postlbl, lbli)
		return
	}

	ctntevtrdr.PostRunesEvent = func(resetlbl bool, rnsl int, rns ...rune) (rnserr error) {
		pstrset := false
		for _, pr := range rns {
			if pstrset, rnserr = postRune(ctntevtrdr.prevr, pr, ctntevtrdr.prelbl, ctntevtrdr.postlbl, ctntevtrdr.lbli); rnserr == nil {
				if pstrset {
					if !elmrmngbuf.Empty() {
						rns = []rune(elmrmngbuf.String())
						rnsl = len(rns)
						elmrmngbuf.Clear()
						return ctntevtrdr.parsePreRunes(false, rns...)
					}
					break
				}
				ctntevtrdr.prevr = pr
			}
		}
		return
	}

	return
}
