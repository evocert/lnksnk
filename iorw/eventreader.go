package iorw

import "io"

type EventReader struct {
	prefixrplcerdr    *ReplaceRuneReader
	PrefixPhraseEvent ReplaceRunesEvent
	suffixrplcerdr    *ReplaceRuneReader
	SuffixPhraseEvent ReplaceRunesEvent
}

func NewEventReader(prefixrdr interface{}, suffixrdr interface{}) (evntrdr *EventReader) {
	evntrdr = &EventReader{prefixrplcerdr: NewReplaceRuneReader(prefixrdr), suffixrplcerdr: NewReplaceRuneReader(suffixrdr)}
	return
}

func (evntrdr *EventReader) PrefixReplace(phrase string) {
	if evntrdr != nil && phrase != "" {
		if prefixrplcrdr := evntrdr.prefixrplcerdr; prefixrplcrdr != nil {
			prefixrplcrdr.ReplaceWith(phrase, evntrdr.prefixEvent)
		}
	}
}

func (evntrdr *EventReader) SuffixReplace(phrase string) {
	if evntrdr != nil && phrase != "" {
		if suffixrplcrdr := evntrdr.suffixrplcerdr; suffixrplcrdr != nil {
			suffixrplcrdr.ReplaceWith(phrase, evntrdr.suffixEvent)
		}
	}
}

func (evntrdr *EventReader) ReadPrefixRune() (r rune, size int, err error) {
	if evntrdr != nil {
		if prefixrdr := evntrdr.prefixrplcerdr; prefixrdr != nil {
			return prefixrdr.ReadRune()
		}
		return 0, 0, io.EOF
	}
	return 0, 0, io.EOF
}

func (evntrdr *EventReader) ReadSuffixRune() (r rune, size int, err error) {
	if evntrdr != nil {
		if suffixrdr := evntrdr.suffixrplcerdr; suffixrdr != nil {
			return suffixrdr.ReadRune()
		}
		return 0, 0, io.EOF
	}
	return 0, 0, io.EOF
}

func (evntrdr *EventReader) prefixEvent(matchphrase string, rplcerrdr *ReplaceRuneReader) interface{} {
	if evntrdr != nil {
		if prefixevent := evntrdr.PrefixPhraseEvent; prefixevent != nil {
			return prefixevent(matchphrase, rplcerrdr)
		}
	}
	return nil
}

func (evntrdr *EventReader) suffixEvent(matchphrase string, rplcerrdr *ReplaceRuneReader) interface{} {
	if evntrdr != nil {
		if suffixevent := evntrdr.SuffixPhraseEvent; suffixevent != nil {
			return suffixevent(matchphrase, rplcerrdr)
		}
	}
	return nil
}

func (evntrdr *EventReader) ReadRune() (r rune, size int, err error) {
	if evntrdr != nil {
		if r, size, err = evntrdr.ReadPrefixRune(); err == io.EOF && size == 0 {
			r, size, err = evntrdr.ReadSuffixRune()
		}
	}
	return
}
