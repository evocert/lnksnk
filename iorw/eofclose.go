package iorw

import (
	"bufio"
	"io"
	"strings"
)

type MultiEOFCloseSeekReader struct {
	eofrdrs []*EOFCloseSeekReader
	eofrdr  *EOFCloseSeekReader
}

func NewMultiEOFCloseSeekReader(r ...io.Reader) (mltieofclsr *MultiEOFCloseSeekReader) {
	var eofrdrs []*EOFCloseSeekReader = nil
	if rl := len(r); rl > 0 {
		var eofrdrs = make([]*EOFCloseSeekReader, rl)
		for rrn := range r {
			eofrdrs[rrn] = NewEOFCloseSeekReader(r[rrn])
		}
	}
	mltieofclsr = &MultiEOFCloseSeekReader{eofrdrs: eofrdrs}
	return
}

func (mltieofclsr *MultiEOFCloseSeekReader) ReadRune() (r rune, size int, err error) {
	if mltieofclsr.eofrdr == nil && len(mltieofclsr.eofrdrs) > 0 {
		mltieofclsr.eofrdr = mltieofclsr.eofrdrs[0]
		mltieofclsr.eofrdrs = mltieofclsr.eofrdrs[1:]
	} else {
		err = io.EOF
		return
	}
	r, size, err = mltieofclsr.eofrdr.ReadRune()
	if err != nil {
		if err == io.EOF {
			err = mltieofclsr.eofrdr.Close()
			mltieofclsr.eofrdr = nil
			if len(mltieofclsr.eofrdrs) > 0 {
				err = nil
			}
		}
	}
	return
}

func (mltieofclsr *MultiEOFCloseSeekReader) Read(p []byte) (n int, err error) {
	if mltieofclsr != nil {
		if pl := len(p); pl > 0 {
			for n < pl {
				if mltieofclsr.eofrdr != nil {
					eofn, eoferr := mltieofclsr.eofrdr.Read(p[n : n+(pl-n)])
					if eofn > 0 {
						n += eofn
					}
					if eoferr != nil {
						if eoferr == io.EOF {
							eoferr = mltieofclsr.eofrdr.Close()
							mltieofclsr.eofrdr = nil
							if eoferr == nil {
								if len(mltieofclsr.eofrdrs) > 0 {
									eoferr = nil
								} else {
									err = io.EOF
									break
								}
							}
						} else {
							err = eoferr
							break
						}
					}
				} else if mltieofclsr.eofrdr == nil && len(mltieofclsr.eofrdrs) > 0 {
					mltieofclsr.eofrdr = mltieofclsr.eofrdrs[0]
					mltieofclsr.eofrdrs = mltieofclsr.eofrdrs[1:]
				} else {
					err = io.EOF
					break
				}
			}
		}
		if err == io.EOF {
			mltieofclsr.Close()
			mltieofclsr = nil
		}
	}
	return
}

func (mltieofclsr *MultiEOFCloseSeekReader) Close() (err error) {
	if mltieofclsr != nil {
		if mltieofclsr.eofrdr != nil {
			mltieofclsr.eofrdr = nil
		}
		if mltieofclsr.eofrdrs != nil {
			eofrdrsl := len(mltieofclsr.eofrdrs)
			for eofrdrsl > 0 {
				mltieofclsr.eofrdrs[0].Close()
				mltieofclsr.eofrdrs[0] = nil
				mltieofclsr.eofrdrs = mltieofclsr.eofrdrs[1:]
				eofrdrsl--
			}
			mltieofclsr.eofrdrs = nil
		}
		mltieofclsr = nil
	}
	return
}

type EOFCloseSeekReader struct {
	r        io.Reader
	rc       io.Closer
	rs       io.Seeker
	size     int64
	bfr      *bufio.Reader
	maxRead  int64
	CanClose bool
}

func NewEOFCloseSeekReader(r io.Reader, canclose ...bool) (eofclsr *EOFCloseSeekReader) {
	if r != nil {
		eofclsr = &EOFCloseSeekReader{r: r, size: -1, CanClose: len(canclose) == 0 || (len(canclose) > 0 && canclose[0]), maxRead: -1}
		if rc, rck := r.(io.Closer); rck {
			eofclsr.rc = rc
		}
		if rs, rsk := r.(io.Seeker); rsk {
			if size, skerr := rs.Seek(0, io.SeekEnd); skerr == nil && size > 0 {
				eofclsr.size = size
				rs.Seek(0, io.SeekStart)
			}
			eofclsr.rs = rs
		}
	}
	return
}

func (eofclsr *EOFCloseSeekReader) ReadRune() (r rune, size int, err error) {
	if eofclsr == nil {
		err = io.EOF
	} else {
		if eofclsr.bfr == nil && eofclsr.r != nil {
			eofclsr.bfr = bufio.NewReader(eofclsr)
			r, size, err = eofclsr.bfr.ReadRune()
		} else if eofclsr.bfr != nil {
			r, size, err = eofclsr.bfr.ReadRune()
			if err == io.EOF {
				eofclsr.Close()
			}
		} else {
			r, size, err = 0, 0, io.EOF
		}
	}
	return
}

// SetMaxRead - set max read implementation for Reader interface compliance
func (eofclsr *EOFCloseSeekReader) SetMaxRead(maxlen int64) (err error) {
	if eofclsr != nil {
		if maxlen < 0 {
			maxlen = -1
		}
		eofclsr.maxRead = maxlen
	}
	return
}

func (eofclsr *EOFCloseSeekReader) InternalReadln(keeperr bool) (s string, err error) {
	rns := make([]rune, 1024)
	rnsi := 0
	for {
		rn, size, rnerr := eofclsr.ReadRune()
		if size > 0 {
			if rn == '\n' {
				if rnsi > 0 {
					s += string(rns[:rnsi])
					rnsi = 0
				}
				break
			}
			rns[rnsi] = rn
			rnsi++
			if rnsi == len(rns) {
				s += string(rns[:rnsi])
				rnsi = 0
			}
		}
		if rnerr != nil {
			err = rnerr
			if rnsi > 0 {
				s += string(rns[:rnsi])
				rnsi = 0
			}
			break
		}
	}
	s = strings.TrimFunc(s, IsSpace)
	if err == io.EOF {
		if !keeperr {
			err = nil
		}
	}
	return
}

func (eofclsr *EOFCloseSeekReader) Readln() (s string, err error) {
	return eofclsr.InternalReadln(false)
}

func (eofclsr *EOFCloseSeekReader) ReadLines() (lines []string, err error) {
	for {
		ln, lnerr := eofclsr.InternalReadln(true)
		if lines == nil {
			lines = []string{}
		}
		lines = append(lines, ln)
		if lnerr != nil {
			if lnerr != io.EOF {
				err = lnerr
			}
			break
		}
	}
	return
}

func (eofclsr *EOFCloseSeekReader) ReadAll() (string, error) {
	return ReaderToString(eofclsr)
}

func (eofclsr *EOFCloseSeekReader) Size() int64 {
	return eofclsr.size
}

func (eofclsr *EOFCloseSeekReader) Seek(offset int64, whence int) (n int64, err error) {
	if eofclsr != nil && eofclsr.r != nil && eofclsr.rs != nil {
		if n, err = eofclsr.rs.Seek(offset, whence); err == nil && n >= 0 {
			eofclsr.maxRead = 0
			if eofclsr.bfr != nil {
				eofclsr.bfr.Reset(eofclsr)
			}
		}
	} else {
		n = -1
	}
	return
}

func (eofclsr *EOFCloseSeekReader) Read(p []byte) (n int, err error) {
	n, err = eofclsr.internalRead(p)
	return
}

func (eofclsr *EOFCloseSeekReader) internalRead(p []byte) (n int, err error) {
	if eofclsr == nil || eofclsr.maxRead == 0 {
		err = io.EOF
		if eofclsr.bfr == nil {
			eofclsr.Close()
		} else {
			eofclsr.disposeReader()
		}
		return
	} else {
		if r := eofclsr.r; r != nil {
			if pl := len(p); pl > 0 {
				if eofclsr.maxRead > 0 {
					if int64(pl) >= eofclsr.maxRead {
						pl = int(eofclsr.maxRead)
					}
				}
				n, err = r.Read(p[:pl])
				if n > 0 && eofclsr.maxRead > 0 {
					eofclsr.maxRead -= int64(n)
					if eofclsr.maxRead <= 0 {
						eofclsr.maxRead = 0
					}
				}
				if err != nil {
					if eofclsr.bfr == nil {
						eofclsr.Close()
					} else {
						eofclsr.disposeReader()
					}
				}
			}
		}
	}
	return
}

/*func readEofClsr(eofclsr *EOFCloseSeekReader, p []byte) (n int, err error) {
	if eofclsr == nil || eofclsr.maxRead == 0 {
		err = io.EOF
		return
	} else if eofclsr.r != nil {
		if pl := len(p); pl > 0 {
			if eofclsr.maxRead > 0 {
				if int64(pl) >= eofclsr.maxRead {
					pl = int(eofclsr.maxRead)
				}
			}
			for n < pl && err == nil {
				if eofclsr.bufl == 0 || (eofclsr.bufl > 0 && eofclsr.bufi == eofclsr.bufl) {
					if len(eofclsr.buf) != 4096 {
						eofclsr.buf = nil
						eofclsr.buf = make([]byte, 4096)
					}
					bulki := 0
					var eofbtsfnc = func(b []byte) {
						bulki += copy(eofclsr.buf[bulki:bulki+(4096-bulki)], b)
					}
					if bufl, bulkerr := ReadHandle(eofclsr.r, eofbtsfnc, 4096); bulkerr != nil && bulkerr != io.EOF {
						err = bulkerr
						break
					} else if eofclsr.bufl = bufl; eofclsr.bufl == 0 {
						break
					}
					eofclsr.bufi = 0
				}
				cpyl := 0

				cpyl, n, eofclsr.bufi = CopyBytes(p, n, eofclsr.buf[:eofclsr.bufl], eofclsr.bufi)
				if cpyl > 0 && eofclsr.maxRead > 0 {
					eofclsr.maxRead -= int64(cpyl)
					if eofclsr.maxRead < 0 {
						eofclsr.maxRead = 0
					}
				}
			}
			if err != nil {
				if eofclsr.bfr == nil {
					eofclsr.Close()
				} else {
					eofclsr.disposeReader()
				}
				if n > 0 && err == io.EOF {
					err = nil
				}
			}
		}
	}
	if n == 0 && err == nil {
		err = io.EOF
	}
	return
}*/

func (eofclsr *EOFCloseSeekReader) disposeReader() (err error) {
	if eofclsr != nil {
		if eofclsr.CanClose {
			if eofclsr.rc != nil {
				eofclsr.rc.Close()
				eofclsr.rc = nil
			}
		}
		if eofclsr.rs != nil {
			eofclsr.rs = nil
		}
		if eofclsr.r != nil {
			eofclsr.r = nil
		}
	}
	return
}

func (eofclsr *EOFCloseSeekReader) Close() (err error) {
	if eofclsr != nil {
		eofclsr.disposeReader()
		if eofclsr.bfr != nil {
			eofclsr.bfr = nil
		}
		eofclsr = nil
	}
	return
}
