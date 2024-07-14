package parsing

import (
	"io"

	"github.com/evocert/lnksnk/iorw"
)

type MatchPhraseHandler struct {
	runerdr          io.RuneReader
	intrnl           bool
	rplcrdr          *iorw.ReplaceRuneReader
	prefixes         map[string]interface{}
	prefixbufs       map[string]*iorw.Buffer
	mtchthis         map[string]interface{}
	FoundPhraseEvent func(prefix, postfix, phrase string, result interface{}) (nxtrslt interface{})
}

func NewMatchPhraseHandler(rdr io.RuneReader, prepostfixes ...string) (mtchphrshndl *MatchPhraseHandler) {
	mtchphrshndl = &MatchPhraseHandler{runerdr: rdr}
	if mtchphrshndl.rplcrdr, _ = rdr.(*iorw.ReplaceRuneReader); mtchphrshndl.rplcrdr == nil {
		mtchphrshndl.intrnl = true
		mtchphrshndl.rplcrdr = iorw.NewReplaceRuneReader(rdr)
	}
	mtchphrshndl.SetPrePostFixes(prepostfixes...)
	return
}

func (mtchphrshndl *MatchPhraseHandler) Close() (err error) {
	if mtchphrshndl == nil {
		return
	}
	prefixes, prefixbufs, mtchthis, rplcrdr := mtchphrshndl.prefixes, mtchphrshndl.prefixbufs, mtchphrshndl.mtchthis, mtchphrshndl.rplcrdr
	mtchphrshndl.rplcrdr = nil
	mtchphrshndl.runerdr = nil
	mtchphrshndl.prefixes = nil
	mtchphrshndl.prefixbufs = nil
	mtchphrshndl.mtchthis = nil
	mtchphrshndl.FoundPhraseEvent = nil
	if mtchphrshndl.intrnl {
		mtchphrshndl.intrnl = false
		if rplcrdr != nil {
			rplcrdr.Close()
		}
	}
	if prefixes != nil {
		clear(prefixes)
	}
	if mtchthis != nil {
		clear(mtchthis)
	}
	if len(prefixbufs) > 0 {
		for pk, pbuf := range prefixbufs {
			if !pbuf.Empty() {
				pbuf.Close()
			}
			delete(prefixbufs, pk)
		}
	}
	return
}

func (mtchphrshndl *MatchPhraseHandler) Match(phrase string, result interface{}) {
	if mtchphrshndl == nil || phrase == "" {
		return
	}
	mtchthis := mtchphrshndl.mtchthis
	if mtchthis == nil {
		mtchthis = map[string]interface{}{}
		mtchphrshndl.mtchthis = mtchthis
	}
	mtchthis[phrase] = result
}

func (mtchphrshndl *MatchPhraseHandler) SetPrePostFixes(prepostfixes ...string) {
	if mtchphrshndl == nil {
		return
	}
	prefixes := mtchphrshndl.prefixes
	phxl := len(prepostfixes)
	for phxl > 1 {
		if prepostfixes[0] == "" {
			break
		}
		if prepostfixes[1] == "" {
			break
		}
		if prefixes == nil {
			prefixes = make(map[string]interface{})
			mtchphrshndl.prefixes = prefixes
		}
		prefixes[prepostfixes[0]] = prepostfixes[1]
		prepostfixes = prepostfixes[2:]
		phxl -= 2
	}
	if len(prefixes) > 0 {
		if rplcrdr := mtchphrshndl.rplcrdr; rplcrdr != nil {
			for prfxk := range prefixes {
				rplcrdr.ReplaceWith(prfxk, mtchphrshndl.MatchPhraseEvent)
			}
		}
	}
}

func (mtchphrshndl *MatchPhraseHandler) MatchPhraseEvent(matchedphrase string, rplcerdr *iorw.ReplaceRuneReader) (nxtrdr interface{}) {
	if mtchphrshndl == nil {
		return
	}
	if prefixes, prefixbufs, mtchthis, fndphrsevt := mtchphrshndl.prefixes, mtchphrshndl.prefixbufs, mtchphrshndl.mtchthis, mtchphrshndl.FoundPhraseEvent; fndphrsevt != nil && len(prefixes) > 0 && len(mtchthis) > 0 {
		if postfx, postfxok := prefixes[matchedphrase]; postfxok {
			prefix := matchedphrase
			var phrsbuf *iorw.Buffer = func() (bf *iorw.Buffer) {
				if prefixbufs == nil {
					prefixbufs = map[string]*iorw.Buffer{}
					mtchphrshndl.prefixbufs = prefixbufs
				}
				if bf = prefixbufs[matchedphrase]; bf == nil {
					bf = iorw.NewBuffer()
					prefixbufs[matchedphrase] = bf
					return
				}
				return
			}()
			if postfxs, _ := postfx.(string); postfxs != "" {
				mtchphrshndl.ReadUntil(phrsbuf, rplcerdr, postfxs, func(postfix string) {
					if mtchphrshndl.MatchThis(phrsbuf, prefix, postfxs, mtchthis, func(mtchprefix, mtchpostfix, mtchkey string, value interface{}) {
						nxtrdr = fndphrsevt(mtchprefix, mtchpostfix, mtchkey, value)
					}) {
						return
					}
				})
				return
			}
			if postfxevent, _ := postfx.(func(phrsbuf *iorw.Buffer, rplcrdr *iorw.ReplaceRuneReader, prefix string) interface{}); postfxevent != nil {
				return postfxevent(phrsbuf, rplcerdr, prefix)
			}
		}
	}
	return
}

func (mtchphrshndl *MatchPhraseHandler) ReadUntil(phrsbuf *iorw.Buffer, rplcrdr *iorw.ReplaceRuneReader, postfix string, FoundEof func(fndpostfix string)) (found bool) {
	if mtchphrshndl != nil && postfix != "" && FoundEof != nil && phrsbuf != nil {
		if rplcrdr == nil {
			if rplcrdr = mtchphrshndl.rplcrdr; rplcrdr == nil {
				return
			}
		}
		rnsuntil := rplcrdr.ReadRunesUntil(postfix)
		phrsbuf.Clear()
		phrsbuf.ReadRunesFrom(rnsuntil)
		if rplcrdr.FoundEOF() {
			FoundEof(postfix)
			found = true
		}
	}
	return
}

func (mtchphrshndl *MatchPhraseHandler) MatchThis(phrsbuf *iorw.Buffer, prefix, postfix string, mtchthis map[string]interface{}, triggrmatch func(mtchprefix, mtchpostfix, mtchkey string, value interface{})) (trigged bool) {
	if !phrsbuf.Empty() && len(mtchthis) > 0 && prefix != "" && postfix != "" && triggrmatch != nil {
		for mtchk, mtchv := range mtchthis {
			if trigged, _ = phrsbuf.Equals(mtchk); trigged {
				triggrmatch(prefix, postfix, mtchk, mtchv)
				return
			}
		}
	}
	return
}

func (mtchphrshndl *MatchPhraseHandler) ReadRune() (r rune, size int, err error) {
	if mtchphrshndl == nil {
		return
	}
	if rdr := mtchphrshndl.runerdr; rdr != nil {
		r, size, err = rdr.ReadRune()
	}
	return
}
