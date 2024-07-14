package iorw

import (
	"bufio"
	"io"
	"unicode/utf8"
)

type UntilRunesReader interface {
	RemainingRunes() []rune
	ReadRune() (rune, int, error)
	ReadLine() (string, error)
	ReadLines() ([]string, error)
	ReadAll() (string, error)
	Reset(eof ...interface{})
	Read([]rune) (int, error)
	FoundEOF() bool
}

type runesreaderuntil struct {
	eofrunes []rune
	eofdone  bool
	eofl     int
	eofi     int
	prveofr  rune
	p        []rune
	pl       int
	orgrdr   io.RuneReader
	intrmbuf []rune
	tmpbuf   []rune
	intrml   int
	tmpintrl int
	intrmi   int
	rmngrns  []rune
}

type ReadRuneFunc func() (rune, int, error)

func (rdrnefunc ReadRuneFunc) ReadRune() (rune, int, error) {
	return rdrnefunc()
}

func RunesReaderUntil(r interface{}, eof ...interface{}) (rdr UntilRunesReader) {
	var rd io.RuneReader = nil
	if rd, _ = r.(io.RuneReader); rd == nil {
		if rr, _ := r.(io.Reader); rr != nil {
			rd = bufio.NewReader(rr)
		}
	}
	rdr = &runesreaderuntil{orgrdr: rd}
	rdr.Reset(eof...)
	return
}

func (rdrrnsuntil *runesreaderuntil) FoundEOF() bool {
	if rdrrnsuntil != nil {
		return rdrrnsuntil.eofdone
	}
	return false
}

func (rdrrnsuntil *runesreaderuntil) RemainingRunes() []rune {
	if rdrrnsuntil != nil {
		return rdrrnsuntil.rmngrns
	}
	return []rune{}
}

func (rdrrnsuntil *runesreaderuntil) ReadRune() (r rune, size int, err error) {
	if rdrrnsuntil != nil {
		if rdrrnsuntil.pl == 0 {
			if len(rdrrnsuntil.p) == rdrrnsuntil.pl {
				rdrrnsuntil.pl = 8192
				rdrrnsuntil.p = make([]rune, rdrrnsuntil.pl)
			}
			if rdrrnsuntil.pl, err = rdrrnsuntil.Read(rdrrnsuntil.p[:rdrrnsuntil.pl]); err != nil {
				if rdrrnsuntil.pl == 0 && err == io.EOF {
					return
				}
				if rdrrnsuntil.pl > 0 && err == io.EOF {
					err = nil
				}
				if err != io.EOF {
					return
				}
			}
			rdrrnsuntil.p = rdrrnsuntil.p[:rdrrnsuntil.pl]
		}
		if rdrrnsuntil.pl > 0 {
			rdrrnsuntil.pl--
			r = rdrrnsuntil.p[0]
			rdrrnsuntil.p = rdrrnsuntil.p[1:]
			size = 1
			if r >= utf8.RuneSelf {
				size = utf8.RuneLen(r)
			}
		}
	}
	return
}

func (rdrrnsuntil *runesreaderuntil) ReadLine() (ln string, err error) {
	if rdrrnsuntil != nil {
		ln, err = ReadLine(rdrrnsuntil)
	}
	return
}

func (rdrrnsuntil *runesreaderuntil) ReadLines() (ln []string, err error) {
	if rdrrnsuntil != nil {
		ln, err = ReadLines(rdrrnsuntil)
	}
	return
}

func (rdrrnsuntil *runesreaderuntil) ReadAll() (all string, err error) {
	if rdrrnsuntil != nil {
		all, err = ReaderToString(rdrrnsuntil)
	}
	return
}

func (rdrrnsuntil *runesreaderuntil) Reset(eof ...interface{}) {
	if rdrrnsuntil != nil {

		var eofrunes []rune = nil
		if len(eof) == 1 {
			if s, sok := eof[0].(string); sok && s != "" {
				eofrunes = []rune(s)
			} else {
				eofrunes, _ = eof[0].([]rune)
			}
		}
		eofl := len(eofrunes)
		if eofl == 0 {
			if eofl = len(rdrrnsuntil.eofrunes); eofl > 0 {
				eofrunes = append([]rune{}, rdrrnsuntil.eofrunes...)
			}
		}
		if rdrrnsuntil.eofdone = !(eofl > 0); !rdrrnsuntil.eofdone {
			if rdrrnsuntil.eofrunes != nil {
				rdrrnsuntil.eofrunes = nil
			}
			rdrrnsuntil.eofrunes = append([]rune{}, eofrunes...)
			if eofl > 0 {
				if rdrrnsuntil.intrmbuf == nil {
					rdrrnsuntil.intrmbuf = make([]rune, 8192)
				}
			}
			rdrrnsuntil.intrml = len(rdrrnsuntil.intrmbuf)
			rdrrnsuntil.intrmi = 0
			rdrrnsuntil.tmpintrl = 0
			rdrrnsuntil.eofl = eofl
			rdrrnsuntil.eofi = 0
			rdrrnsuntil.prveofr = 0
			rdrrnsuntil.tmpbuf = []rune{}
		}
	}
}

func (rdrrnsuntil *runesreaderuntil) Read(p []rune) (n int, err error) {
	if rdrrnsuntil != nil && !rdrrnsuntil.eofdone {
		if pl := len(p); pl > 0 && rdrrnsuntil.intrml > 0 {
			for tn, tb := range rdrrnsuntil.tmpbuf {
				p[n] = tb
				n++
				if n == pl {
					rdrrnsuntil.tmpbuf = rdrrnsuntil.tmpbuf[tn+1:]
					return
				}
			}
			if rdrrnsuntil.tmpintrl == 0 {
				rdrrnsuntil.intrmi = 0
				if rdrrnsuntil.tmpintrl, err = ReadRunes(rdrrnsuntil.intrmbuf, rdrrnsuntil.orgrdr); err == nil {
					return rdrrnsuntil.Read(p)
				}
			} else {
				tmpintrmbuf := rdrrnsuntil.intrmbuf[rdrrnsuntil.intrmi : rdrrnsuntil.intrmi+(rdrrnsuntil.tmpintrl-rdrrnsuntil.intrmi)]
				tmpintrmbufl := len(tmpintrmbuf)
				for bn, bb := range tmpintrmbuf {
					if rdrrnsuntil.eofi > 0 && rdrrnsuntil.eofrunes[rdrrnsuntil.eofi-1] == rdrrnsuntil.prveofr && rdrrnsuntil.eofrunes[rdrrnsuntil.eofi] != bb {
						tmpbuf := rdrrnsuntil.eofrunes[:rdrrnsuntil.eofi]
						rdrrnsuntil.eofi = 0
						for tn, tb := range tmpbuf {
							p[n] = tb
							n++
							if n == pl {
								if tn < len(tmpbuf)-1 {
									rdrrnsuntil.tmpbuf = append(rdrrnsuntil.tmpbuf, tmpbuf[tn+1:]...)
								}
								rdrrnsuntil.intrmi += bn + 1
								if rdrrnsuntil.intrmi == rdrrnsuntil.tmpintrl {
									rdrrnsuntil.tmpintrl = 0
								}
								return
							}
						}
					}
					if rdrrnsuntil.eofrunes[rdrrnsuntil.eofi] == bb {
						rdrrnsuntil.eofi++
						if rdrrnsuntil.eofi == rdrrnsuntil.eofl {
							rdrrnsuntil.eofdone = true
							rdrrnsuntil.rmngrns = append([]rune{}, tmpintrmbuf[bn+1:]...)
							return
						} else {
							rdrrnsuntil.prveofr = bb
						}
					} else {
						if rdrrnsuntil.eofi > 0 {
							tmpbuf := rdrrnsuntil.eofrunes[:rdrrnsuntil.eofi]
							rdrrnsuntil.eofi = 0
							for tn, tb := range tmpbuf {
								p[n] = tb
								n++
								if n == pl {
									if tn < len(tmpbuf)-1 {
										rdrrnsuntil.tmpbuf = append(rdrrnsuntil.tmpbuf, tmpbuf[tn+1:]...)
									}
									rdrrnsuntil.intrmi += bn + 1
									if rdrrnsuntil.intrmi == rdrrnsuntil.tmpintrl {
										rdrrnsuntil.tmpintrl = 0
									}
									return
								}
							}
						}
						rdrrnsuntil.prveofr = bb
						p[n] = bb
						n++
						if n == pl {
							rdrrnsuntil.intrmi += bn + 1
							if rdrrnsuntil.intrmi == rdrrnsuntil.tmpintrl {
								rdrrnsuntil.tmpintrl = 0
							}
							return
						}
					}
					if bn == tmpintrmbufl-1 {
						rdrrnsuntil.intrmi += tmpintrmbufl
						if rdrrnsuntil.intrmi == rdrrnsuntil.tmpintrl {
							rdrrnsuntil.tmpintrl = 0
						}
					}
				}
			}
		}
	}

	if n == 0 && err == nil {
		err = io.EOF
	}
	return
}
