package parsing

import (
	"io"

	"github.com/lnksnk/lnksnk/iorw"
)

type ParseEventReader struct {
	*iorw.RuneReaderSlice
	prelbl              []rune
	prevr               rune
	preL                int
	postlbl             []rune
	PostTxtr            rune
	postL               int
	lbli                []int
	cachebuff           *iorw.Buffer
	cacherdr            *iorw.BuffReader
	CanPreParse         func() bool
	CanPostParse        func() bool
	PreRunesEvent       func(reset bool, rnsl int, rns ...rune) (rnserr error)
	PostRunesEvent      func(reset bool, rnsl int, rns ...rune) (rnserr error)
	PreBufferEvent      func(prebuff *iorw.Buffer)
	PostBufferEvent     func(postbuf *iorw.Buffer)
	PostCanSetTextPar   func(prevr, r rune) bool
	PostCanResetTextPar func(prevr, r rune) bool
	PreResetEvent       func(prel, postl int, prelbl []rune, postlbl []rune, lbli []int) (reseterr error)
	PostResetEvent      func(prel, postl int, prelbl []rune, postlbl []rune, lbli []int) (reseterr error)
}

func newParseEventReader(prelabel, postlabel string, rnrdrs ...io.RuneReader) (prsevtrdr *ParseEventReader) {
	prsevtrdr = &ParseEventReader{
		RuneReaderSlice: iorw.NewRuneReaderSlice(rnrdrs...),
		prelbl:          []rune(prelabel),
		postlbl:         []rune(postlabel),
	}
	prsevtrdr.preL = len(prsevtrdr.prelbl)
	prsevtrdr.postL = len(prsevtrdr.postlbl)
	prsevtrdr.lbli = []int{0, 0}
	return
}

func (prsevtrdr *ParseEventReader) InternalReadRune() (rune, int, error) {
	if prsevtrdr == nil {
		return 0, 0, io.EOF
	}
	if slcrdr := prsevtrdr.RuneReaderSlice; slcrdr != nil {
		return slcrdr.ReadRune()
	}
	return 0, 0, io.EOF
}

func (prsevtrdr *ParseEventReader) ReadRune() (rune, int, error) {
	return internalParseEvtReadRune(prsevtrdr)
}

func (prsevtrdr *ParseEventReader) parsePrebuffer() {
	if prsevtrdr != nil {
		if prebuffEvent, cachebuff := prsevtrdr.PreBufferEvent, prsevtrdr.cachebuff; prebuffEvent != nil {
			prebuffEvent(cachebuff)
			if cachebuff != nil {
				cachebuff.Clear()
			}
		}
	}
}

func (prsevtrdr *ParseEventReader) parsePostbuffer() {
	if prsevtrdr != nil {
		if postbuffEvent, cachebuff := prsevtrdr.PostBufferEvent, prsevtrdr.cachebuff; postbuffEvent != nil {
			postbuffEvent(cachebuff)
			if cachebuff != nil {
				cachebuff.Clear()
			}
		}
	}
}

func (prsevtrdr *ParseEventReader) ClearCacheBuffer() {
	if prsevtrdr != nil {
		if cachedbuf := prsevtrdr.cachebuff; cachedbuf != nil {
			cachedbuf.Clear()
		}
	}
}

func (prsevtrdr *ParseEventReader) parsePreRunes(resetlbl bool, rns ...rune) (err error) {
	if prsevtrdr == nil {
		return
	}
	if rnsl := len(rns); rnsl > 0 {
		if parseRnsEvt := prsevtrdr.PreRunesEvent; parseRnsEvt != nil {
			err = parseRnsEvt(resetlbl, rnsl, rns...)
			return
		}
		prsevtrdr.cachebuffer().WriteRunes(rns...)
	}
	return
}

func (prsevtrdr *ParseEventReader) parsePostRunes(resetlbl bool, rns ...rune) (reset bool, err error) {
	if prsevtrdr == nil {
		return
	}
	if rnsl := len(rns); rnsl > 0 {
		if parseRnsEvt := prsevtrdr.PostRunesEvent; parseRnsEvt != nil {
			err = parseRnsEvt(resetlbl, rnsl, rns...)
			return
		}
		prsevtrdr.cachebuffer().WriteRunes(rns...)
	}
	return
}

type parseStage int

const (
	UnknownStage parseStage = iota
	PreStage
	PostStage
)

func (rsgstg parseStage) String() string {
	if rsgstg == PostStage {
		return "post"
	}
	if rsgstg == PreStage {
		return "pre"
	}
	return "unknown"
}

func (prsevtrdr *ParseEventReader) ParseStage() parseStage {
	if prsevtrdr == nil {
		return UnknownStage
	}
	if lbli := prsevtrdr.lbli; len(lbli) == 2 {
		if lbli[0] == len(prsevtrdr.prelbl) {
			return PostStage
		}
		if lbli[1] == 0 {
			return PreStage
		}
	}
	return UnknownStage
}

func (prsevtrdr *ParseEventReader) SwapParseState() {
	if prsevtrdr == nil {
		return
	}
	if lbli := prsevtrdr.lbli; len(lbli) == 2 {
		if lbli[0] == len(prsevtrdr.prelbl) {
			lbli[0] = 0
			lbli[1] = 0
			prsevtrdr.prevr = 0
			return
		}
		if lbli[1] == 0 {
			lbli[0] = len(prsevtrdr.prelbl)
			lbli[1] = 0
			prsevtrdr.prevr = 0
			return
		}
	}
}

func (prsevtrdr *ParseEventReader) cachebuffer() (cachedbuf *iorw.Buffer) {
	if prsevtrdr != nil {
		if cachedbuf = prsevtrdr.cachebuff; cachedbuf == nil {
			cachedbuf = iorw.NewBuffer()
			prsevtrdr.cachebuff = cachedbuf
		}
	}
	return
}

func (prsevtrdr *ParseEventReader) resetPost(resetonly bool) (rseterr error) {
	if prsevtrdr != nil {
		if !resetonly {
			if resetPostEvt := prsevtrdr.PostResetEvent; resetPostEvt != nil {
				rseterr = resetPostEvt(prsevtrdr.preL, prsevtrdr.postL, prsevtrdr.prelbl, prsevtrdr.postlbl, prsevtrdr.lbli)
			}
		}
		prsevtrdr.lbli[0] = 0
		prsevtrdr.lbli[1] = 0
		prsevtrdr.prevr = 0
	}
	return
}
func (prsevtrdr *ParseEventReader) setPrevR(r rune) {
	if prsevtrdr != nil {
		prsevtrdr.prevr = r
	}
}

func (prsevtrdr *ParseEventReader) parseRunes(rns ...rune) (err error) {
	if prsevtrdr != nil {
		for _, r := range rns {
			err = internalParseEvent(prsevtrdr, prsevtrdr.CanPreParse, prsevtrdr.CanPostParse, prsevtrdr.prevr, prsevtrdr.setPrevR, r, prsevtrdr.preL, prsevtrdr.postL, prsevtrdr.prelbl, prsevtrdr.postlbl, prsevtrdr.lbli)
		}
	}
	return
}

func (prsevtrdr *ParseEventReader) parseRune(r rune) (err error) {
	if prsevtrdr != nil {
		err = internalParseEvent(prsevtrdr, prsevtrdr.CanPreParse, prsevtrdr.CanPostParse, prsevtrdr.prevr, prsevtrdr.setPrevR, r, prsevtrdr.preL, prsevtrdr.postL, prsevtrdr.prelbl, prsevtrdr.postlbl, prsevtrdr.lbli)
	}
	return
}

func (prsevtrdr *ParseEventReader) resetPre(resetonly bool) (rseterr error) {
	if prsevtrdr != nil {
		if !resetonly {
			if resetPreEvt := prsevtrdr.PreResetEvent; resetPreEvt != nil {
				rseterr = resetPreEvt(prsevtrdr.preL, prsevtrdr.postL, prsevtrdr.prelbl, prsevtrdr.postlbl, prsevtrdr.lbli)
			}
		}
		prsevtrdr.prevr = 0
	}
	return
}

func (prsevtrdr *ParseEventReader) DummyEOFRead() (err error) {
	if prsevtrdr == nil {
		return
	}
	for {
		if _, _, err = prsevtrdr.ReadRune(); err != nil {
			if err == io.EOF {
				err = nil
			}
			break
		}
	}
	return
}

func internalParseEvent(prsevtrdr *ParseEventReader, canpreparse func() bool, canpostparse func() bool, prevr rune, setprevr func(rune), r rune, preLen, postLen int, prelbl, postlbl []rune, lbli []int) (prserr error) {
	if prsevtrdr == nil {
		return
	}
	reset := false
	if prsevtrdr.PostTxtr > 0 {
		resetposttxtparevt := prsevtrdr.PostCanResetTextPar
		if resetposttxtparevt != nil && resetposttxtparevt(prevr, r) {
			reset, prserr = prsevtrdr.parsePostRunes(false, r)
			prsevtrdr.PostTxtr = 0
			prsevtrdr.prevr = 0
			return
		}
		if resetposttxtparevt != nil {
			reset, prserr = prsevtrdr.parsePostRunes(false, r)
			prsevtrdr.prevr = r
		}
		return
	}

	if (canpreparse == nil || canpreparse()) && lbli[1] == 0 && lbli[0] < preLen {
		if lbli[0] > 0 && prelbl[lbli[0]-1] == prevr && prelbl[lbli[0]] != r {
			li := lbli[0]
			lbli[0] = 0
			setprevr(0)
			if prserr = prsevtrdr.parsePreRunes(true, prelbl[:li]...); prserr != nil {
				return
			}
		}
		if prelbl[lbli[0]] == r {
			lbli[0]++
			if lbli[0] == preLen {
				prsevtrdr.resetPre(false)
				return
			}
			setprevr(r)
			return
		}
		if lbli[0] > 1 {
			li := lbli[0]
			lbli[0] = 0
			setprevr(0)
			prerns := make([]rune, li+1)
			copy(prerns, prelbl[:li])
			prerns[li] = r
			if prserr = prsevtrdr.parsePreRunes(true, append(prerns, r)...); prserr != nil {
				return
			}
			return
		}
		if prserr = prsevtrdr.parsePreRunes(false, r); prserr != nil {
			return
		}
		setprevr(r)
		return
	}
	if (canpostparse == nil || canpostparse()) && lbli[0] == preLen && lbli[1] < postLen {
		if setposttxtparevt := prsevtrdr.PostCanSetTextPar; setposttxtparevt != nil && prsevtrdr.PostTxtr == 0 && setposttxtparevt(prevr, r) {
			prsevtrdr.PostTxtr = r
			prsevtrdr.prevr = 0
			reset, prserr = prsevtrdr.parsePostRunes(false, r)
			return
		}
		if postlbl[lbli[1]] == r {
			lbli[1]++
			if lbli[1] == postLen {

				prsevtrdr.resetPost(false)
				return
			}
			setprevr(r)
			return
		}
		if lbli[1] > 0 {
			li := lbli[1]
			lbli[1] = 0
			setprevr(0)
			pstrns := make([]rune, li+1)
			copy(pstrns, postlbl[:li])
			pstrns[li] = r
			if reset, prserr = prsevtrdr.parsePostRunes(true, pstrns...); prserr != nil {
				return
			}
			if reset {
				prsevtrdr.resetPre(reset)
				prsevtrdr.resetPost(reset)
			}
			return
		}
		if reset, prserr = prsevtrdr.parsePostRunes(false, r); prserr != nil {
			return
		}
		if reset {
			prsevtrdr.resetPre(reset)
			prsevtrdr.resetPost(reset)
			return
		}
		setprevr(r)
		return
	}
	return
}

func internalParseEvtReadRune(prsevtrdr *ParseEventReader) (r rune, size int, err error) {
	if prsevtrdr != nil {
		canpreparse, canpostparse, cacherdr, cachedbuf := prsevtrdr.CanPreParse, prsevtrdr.CanPostParse, prsevtrdr.cacherdr, prsevtrdr.cachebuff
		if !cachedbuf.Empty() && cacherdr == nil {
			cacherdr = cachedbuf.Clone(true).Reader(true)
		}
		if cacherdr != nil {
			r, size, err = cacherdr.ReadRune()
			if err != nil {
				cacherdr.Close()
				prsevtrdr.cacherdr = nil
			}
			if err == io.EOF {
				err = nil
			}
			if size > 0 {
				return
			}
		}
		for err == nil {
		NXTRD:
			pr, ps, perr := prsevtrdr.InternalReadRune()
			if ps > 0 {
				if perr == nil || perr == io.EOF {
					if err = internalParseEvent(prsevtrdr, canpreparse, canpostparse, prsevtrdr.prevr, prsevtrdr.setPrevR, pr, prsevtrdr.preL, prsevtrdr.postL, prsevtrdr.prelbl, prsevtrdr.postlbl, prsevtrdr.lbli); err != nil {
						break
					}
				}
				if perr == nil {
					goto NXTRD
				}
			}
			if perr != nil {
				if cachedbuf = prsevtrdr.cachebuff; !cachedbuf.Empty() {
					r, size, err = internalParseEvtReadRune(prsevtrdr)
				}
				break
			}
		}

		/*if r, size, err = rnslcrdr.ReadRune(); err != nil {
			prsevtrdr.RuneReaderSlice = nil
			rnslcrdr.Close()
		}*/
	}
	if size == 0 && err == nil {
		err = io.EOF
	}
	return
}
