package iorw

import (
	"bufio"
	"io"
)

type UntilReader interface {
	RemainingBytes() []byte
	ReadRune() (rune, int, error)
	ReadLine() (string, error)
	ReadLines() ([]string, error)
	ReadAll() (string, error)
	Reset(eof ...interface{})
	Read([]byte) (int, error)
}

type readeruntil struct {
	eofbytes []byte
	eofdone  bool
	eofl     int
	eofi     int
	prveofb  byte
	rdr      io.Reader
	bufr     *bufio.Reader
	intrmbuf []byte
	tmpbuf   []byte
	intrml   int
	tmpintrl int
	intrmi   int
	rmngbts  []byte
}

func ReaderUntil(r io.Reader, eof ...interface{}) (rdr UntilReader) {
	rdr = &readeruntil{rdr: r}
	rdr.Reset(eof...)
	return
}

func (rdruntil *readeruntil) RemainingBytes() []byte {
	if rdruntil != nil {
		return rdruntil.rmngbts
	}
	return []byte{}
}

func (rdruntil *readeruntil) ReadRune() (r rune, size int, err error) {
	if rdruntil != nil {
		if rdruntil.bufr == nil {
			rdruntil.bufr = bufio.NewReaderSize(rdruntil, 2)
		}
		r, size, err = rdruntil.bufr.ReadRune()
	}
	return
}

func (rdruntil *readeruntil) ReadLine() (ln string, err error) {
	if rdruntil != nil {
		ln, err = ReadLine(rdruntil)
	}
	return
}

func (rdruntil *readeruntil) ReadLines() (ln []string, err error) {
	if rdruntil != nil {
		ln, err = ReadLines(rdruntil)
	}
	return
}

func (rdruntil *readeruntil) ReadAll() (all string, err error) {
	if rdruntil != nil {
		all, err = ReaderToString(rdruntil)
	}
	return
}

func (rdruntil *readeruntil) Reset(eof ...interface{}) {
	if rdruntil != nil {
		if rdruntil.bufr != nil {
			rdruntil.bufr.Reset(rdruntil)
		}
		var eofbytes []byte = nil
		if len(eof) == 1 {
			if s, sok := eof[0].(string); sok && s != "" {
				eofbytes = []byte(s)
			} else {
				eofbytes, _ = eof[0].([]byte)
			}
		}
		eofl := len(eofbytes)
		if eofl == 0 {
			if eofl = len(rdruntil.eofbytes); eofl > 0 {
				eofbytes = append([]byte{}, rdruntil.eofbytes...)
			}
		}
		if rdruntil.eofdone = !(eofl > 0); !rdruntil.eofdone {
			if rdruntil.eofbytes != nil {
				rdruntil.eofbytes = nil
			}
			rdruntil.eofbytes = append([]byte{}, eofbytes...)
			if eofl > 0 {
				if rdruntil.intrmbuf == nil {
					rdruntil.intrmbuf = make([]byte, 8192)
				}
			}
			rdruntil.intrml = len(rdruntil.intrmbuf)
			rdruntil.intrmi = 0
			rdruntil.tmpintrl = 0
			rdruntil.eofl = eofl
			rdruntil.eofi = 0
			rdruntil.prveofb = 0
			rdruntil.tmpbuf = []byte{}
		}
	}
}

func (rdruntil *readeruntil) Read(p []byte) (n int, err error) {
	if rdruntil != nil && !rdruntil.eofdone {
		if pl := len(p); pl > 0 && rdruntil.intrml > 0 {
			for tn, tb := range rdruntil.tmpbuf {
				p[n] = tb
				n++
				if n == pl {
					rdruntil.tmpbuf = rdruntil.tmpbuf[tn+1:]
					return
				}
			}
			if rdruntil.tmpintrl == 0 {
				rdruntil.intrmi = 0
				if rdruntil.tmpintrl, err = rdruntil.rdr.Read(rdruntil.intrmbuf); err == nil {
					return rdruntil.Read(p)
				}
			} else {
				tmpintrmbuf := rdruntil.intrmbuf[rdruntil.intrmi : rdruntil.intrmi+(rdruntil.tmpintrl-rdruntil.intrmi)]
				tmpintrmbufl := len(tmpintrmbuf)
				for bn, bb := range tmpintrmbuf {
					if rdruntil.eofi > 0 && rdruntil.eofbytes[rdruntil.eofi-1] == rdruntil.prveofb && rdruntil.eofbytes[rdruntil.eofi] != bb {
						tmpbuf := rdruntil.eofbytes[:rdruntil.eofi]
						rdruntil.eofi = 0
						for tn, tb := range tmpbuf {
							p[n] = tb
							n++
							if n == pl {
								if tn < len(tmpbuf)-1 {
									rdruntil.tmpbuf = append(rdruntil.tmpbuf, tmpbuf[tn+1:]...)
								}
								rdruntil.intrmi += bn + 1
								if rdruntil.intrmi == rdruntil.tmpintrl {
									rdruntil.tmpintrl = 0
								}
								return
							}
						}
					}
					if rdruntil.eofbytes[rdruntil.eofi] == bb {
						rdruntil.eofi++
						if rdruntil.eofi == rdruntil.eofl {
							rdruntil.eofdone = true
							rdruntil.rmngbts = append([]byte{}, tmpintrmbuf[bn+1:]...)
							return
						} else {
							rdruntil.prveofb = bb
						}
					} else {
						if rdruntil.eofi > 0 {
							tmpbuf := rdruntil.eofbytes[:rdruntil.eofi]
							rdruntil.eofi = 0
							for tn, tb := range tmpbuf {
								p[n] = tb
								n++
								if n == pl {
									if tn < len(tmpbuf)-1 {
										rdruntil.tmpbuf = append(rdruntil.tmpbuf, tmpbuf[tn+1:]...)
									}
									rdruntil.intrmi += bn + 1
									if rdruntil.intrmi == rdruntil.tmpintrl {
										rdruntil.tmpintrl = 0
									}
									return
								}
							}
						}
						rdruntil.prveofb = bb
						p[n] = bb
						n++
						if n == pl {
							rdruntil.intrmi += bn + 1
							if rdruntil.intrmi == rdruntil.tmpintrl {
								rdruntil.tmpintrl = 0
							}
							return
						}
					}
					if bn == tmpintrmbufl-1 {
						rdruntil.intrmi += tmpintrmbufl
						if rdruntil.intrmi == rdruntil.tmpintrl {
							rdruntil.tmpintrl = 0
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
