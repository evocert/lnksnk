package parsing

import (
	"io"
	"unicode/utf8"

	"github.com/evocert/lnksnk/iorw"
)

type untilrunereader struct {
	rmngrns    []rune
	rmngrnsi   int
	eofrns     []rune
	eofrnsi    int
	eofprvr    rune
	FoundUntil bool
	Done       bool
	rdr        io.RuneReader
}

func NewUntilRuneReader(rdr io.RuneReader, eof ...interface{}) (eofrdr *untilrunereader) {
	eofrdr = &untilrunereader{rdr: rdr}
	eofrdr.NextUntil(eof...)
	return
}

func (eofrdr *untilrunereader) WriteTo(w io.Writer) (n int64, err error) {
	if eofrdr != nil && w != nil {
		if bfw, _ := w.(*iorw.Buffer); bfw != nil {
			for {
				r, s, rerr := eofrdr.ReadRune()
				if s > 0 {
					n++
					bfw.WriteRune(r)
				}
				if rerr != nil {
					err = rerr
					break
				}
			}
		} else {
			for {
				r, s, rerr := eofrdr.ReadRune()
				if s > 0 {
					n++
					w.Write(iorw.RunesToUTF8(r))
				}
				if rerr != nil {
					err = rerr
					break
				}
			}
		}

	}
	return
}

func (eofrdr *untilrunereader) NextUntil(a ...interface{}) {
	if eofrdr != nil {
		b := iorw.NewBuffer()
		b.Print(a...)
		eofrdr.eofrns = []rune(b.String())
		b.Close()
		eofrdr.eofprvr = rune(0)
		eofrdr.eofrnsi = 0
		eofrdr.rmngrns = nil
		eofrdr.rmngrnsi = 0
		eofrdr.FoundUntil = false
		eofrdr.Done = false
	}
}

func (eofrdr *untilrunereader) ReadRune() (r rune, size int, err error) {
	if eofrdr != nil {
		if rmngl := len(eofrdr.rmngrns); eofrdr.rmngrnsi < rmngl {
			r = eofrdr.rmngrns[eofrdr.rmngrnsi]
			size = utf8.RuneLen(r)
			eofrdr.rmngrnsi++
			if eofrdr.rmngrnsi == rmngl {
				eofrdr.rmngrns = nil
				eofrdr.rmngrnsi = 0
			}
			return
		}
		if !eofrdr.Done {
			if eofrnsl := len(eofrdr.eofrns); eofrnsl > 0 && !eofrdr.FoundUntil {
				for !eofrdr.FoundUntil && len(eofrdr.rmngrns) == 0 {
					rr, rsize, rerr := eofrdr.rdr.ReadRune()
					if rsize > 0 {
						if eofrdr.eofrnsi > 0 && eofrdr.eofrns[eofrdr.eofrnsi-1] == eofrdr.eofprvr && eofrdr.eofrns[eofrdr.eofrnsi] != rr {
							eofi := eofrdr.eofrnsi
							eofrdr.eofrnsi = 0
							eofrdr.eofprvr = 0
							eofrdr.rmngrns = append(eofrdr.rmngrns, eofrdr.eofrns[:eofi]...)
						}
						if eofrdr.eofrns[eofrdr.eofrnsi] == rr {
							eofrdr.eofrnsi++
							if eofrdr.FoundUntil = eofrdr.eofrnsi == eofrnsl; !eofrdr.FoundUntil {
								eofrdr.eofprvr = rr
							} else if eofrdr.FoundUntil {
								eofrdr.Done = eofrdr.FoundUntil
							}
						} else {
							if eofrdr.eofrnsi > 0 {
								eofi := eofrdr.eofrnsi
								eofrdr.eofrnsi = 0
								eofrdr.eofprvr = rr

								eofrdr.rmngrns = append(append(eofrdr.rmngrns, eofrdr.eofrns[:eofi]...), rr)
								break
							}
							eofrdr.eofprvr = rr
							if len(eofrdr.rmngrns) > 0 {
								eofrdr.rmngrns = append(eofrdr.rmngrns, rr)
								break
							}
							return rr, utf8.RuneLen(rr), nil
						}
					}
					if rerr != nil {
						eofrdr.Done = true
						if rerr != io.EOF {
							err = rerr
							return
						}
						break
					}
				}
				if len(eofrdr.rmngrns) > 0 {
					r, size, err = eofrdr.ReadRune()
				}
			} else {
				if len(eofrdr.rmngrns) > 0 {
					r, size, err = eofrdr.ReadRune()
					return
				}
				err = io.EOF
			}
		} else {
			err = io.EOF
		}
	}
	return
}
