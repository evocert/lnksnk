package iorw

import (
	"io"
	"unicode/utf8"
)

type RunesReader struct {
	rns  []rune
	rnsl int
}

func (rnsrdr *RunesReader) Close() (err error) {
	if rnsrdr == nil {
		return
	}
	if rns := rnsrdr.rns; len(rns) > 0 {
		rnsrdr.rnsl = 0
		rnsrdr.rns = nil
	}
	return
}

func (rnsrdr *RunesReader) ReadRune() (r rune, size int, err error) {
	if rnsrdr == nil {
		return 0, 0, io.EOF
	}
	if rnsrdr.rnsl > 0 {
		r = rnsrdr.rns[0]
		rnsrdr.rns = rnsrdr.rns[1:]
		rnsrdr.rnsl--
		size = utf8.RuneLen(r)
		return
	}
	rnsrdr.Close()
	return 0, 0, io.EOF
}

func NewRunesReader(rns ...rune) (rnsrdr *RunesReader) {
	if len(rns) > 0 {
		rnsrdr = &RunesReader{rns: append([]rune{}, rns...), rnsl: len(rns)}
	}
	return
}
