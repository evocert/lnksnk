package stdio

import (
	"bufio"
	"io"
	"strings"
	"time"

	"github.com/lnksnk/lnksnk/iorw"
)

type Printer interface {
	Write([]byte) (int, error)
	Print(...interface{}) error
	Println(...interface{}) error
	Flush() error
	io.Closer
}

type Reader interface {
	Read([]byte) (int, error)
	Readln(...bool) (string, error)
	ReadLines() ([]string, error)
	ReadAll(...bool) (string, error)
	io.Closer
}

type ReaderPrinter interface {
	Reader
	Printer
	io.Closer
}

type funcrdwtr struct {
	bufr        *bufio.Reader
	tcr         *time.Ticker
	initialdur  time.Duration
	nextdur     time.Duration
	crntdur     time.Duration
	crntrn      chan int
	crntrerr    chan error
	rd          func([]byte) (int, error)
	bufw        *bufio.Writer
	wt          func([]byte) (int, error)
	closeReader func() error
	closeWriter func() error
	closeAll    func() error
}

func (funcrw *funcrdwtr) resetBufr() (set bool) {
	if funcrw != nil {
		if funcrw.bufr == nil {
			funcrw.bufr = bufio.NewReader(funcrw)
		} else {
			funcrw.bufr.Reset(funcrw)
		}
		set = true
	}
	return
}

func (funcrw *funcrdwtr) readBuffer() (bufr *bufio.Reader) {
	if funcrw != nil {
		if funcrw.bufr == nil {
			funcrw.bufr = bufio.NewReader(funcrw)
		}
		bufr = funcrw.bufr
	}
	return
}

func (funcrw *funcrdwtr) ReadUntil(f func(byte) bool, ignoreeof ...bool) (s string, err error) {
	if funcrw != nil && f != nil {
		bout := []byte{}
		bp := make([]byte, 1)
		bpn := 0
		bperr := error(nil)
		noeof := len(ignoreeof) == 0 || (len(ignoreeof) > 1 && ignoreeof[0])
		if f != nil {
			if bf := funcrw.readBuffer(); bf != nil {
				for {
					if bpn, bperr = bf.Read(bp); bpn > 0 && (bperr == nil || bperr == io.EOF) {
						if f(bp[0]) {
							break
						} else {
							bout = append(bout, bp...)
						}
					} else if bperr != nil {
						if err = bperr; err == io.EOF && noeof {
							err = nil
						}
						break
					}
				}
			}
		}
		if err == nil || err == io.EOF {
			s = string(bout)
			if noeof {
				s = strings.TrimFunc(s, iorw.IsSpace)
			}
		}
	}
	return
}

func (funcrw *funcrdwtr) Readln(ignoreeof ...bool) (ln string, err error) {
	ln, err = funcrw.ReadUntil(func(b byte) bool {
		return b == '\n'
	}, ignoreeof...)
	return
}

func (funcrw *funcrdwtr) ReadAll(ignoreeof ...bool) (all string, err error) {
	bout := []byte{}
	bp := make([]byte, 1)
	bpn := 0
	bperr := error(nil)
	noeof := len(ignoreeof) == 0 || (len(ignoreeof) > 1 && ignoreeof[0])
	if bf := funcrw.readBuffer(); bf != nil {
		for {
			if bpn, bperr = bf.Read(bp); bpn > 0 && (bperr == nil || bperr == io.EOF) {
				bout = append(bout, bp...)
			}
			if bperr != nil {
				if err = bperr; err == io.EOF && noeof {
					err = nil
				}
				break
			}
		}
	}
	if err == nil || err == io.EOF {
		all = string(bout)
		if noeof {
			all = strings.TrimRightFunc(all, iorw.IsSpace)
		}
	}
	return
}

func (funcrw *funcrdwtr) ReadLines() (lines []string, err error) {
	ln := ""
	for err == nil {
		if ln, err = funcrw.Readln(false); err == nil || err == io.EOF {
			if lines == nil {
				lines = append([]string{}, strings.TrimRightFunc(ln, iorw.IsSpace))
			} else {
				lines = append(lines, strings.TrimRightFunc(ln, iorw.IsSpace))
			}
			if err == io.EOF {
				break
			}
		}
	}
	return
}

func (funcrw *funcrdwtr) ReadRune() (r rune, size int, err error) {
	if funcrw != nil {
		if funcrw.bufr == nil {
			if funcrw.resetBufr() {
				r, size, err = funcrw.bufr.ReadRune()
			}
		} else {
			r, size, err = funcrw.bufr.ReadRune()
		}
	}
	return
}

func (funcrw *funcrdwtr) Read(p []byte) (n int, err error) {
	if funcrw != nil && funcrw.rd != nil {
		if doIntrvl := funcrw.initialdur > 0 && funcrw.nextdur > 0 && funcrw.initialdur > funcrw.nextdur; doIntrvl {
			if funcrw.crntdur == -1 {
				funcrw.crntdur = funcrw.initialdur
			}
			if funcrw.tcr == nil {
				funcrw.tcr = time.NewTicker(funcrw.crntdur)
				go func() {
					for funcrw.rd != nil {
						gn, gerr := funcrw.rd(p)
						funcrw.crntrn <- gn
						funcrw.crntrerr <- gerr
						if gerr != nil {
							if gerr != io.EOF {
								break
							}
						}
					}
				}()
			} else {
				funcrw.tcr.Reset(funcrw.crntdur)
			}
			select {
			case <-funcrw.tcr.C:
				funcrw.crntdur = -1
				n = 0
				err = io.EOF
			case n = <-funcrw.crntrn:
				err = <-funcrw.crntrerr
				if err != nil {
					if err == io.EOF {
						funcrw.crntdur = funcrw.initialdur
					}
				} else {
					funcrw.crntdur = funcrw.nextdur
				}
			}
		} else {
			n, err = funcrw.rd(p)
		}
	}
	return
}

func (funcrw *funcrdwtr) Flush() (err error) {
	if funcrw != nil {
		if bufw := funcrw.bufw; bufw != nil {
			err = bufw.Flush()
		}
	}
	return
}

func (funcrw *funcrdwtr) Print(a ...interface{}) (err error) {
	if funcrw != nil {
		if bufw := funcrw.writeBuffer(); bufw != nil {
			if err = iorw.Fprint(bufw, a...); err != nil {
				err = funcrw.Flush()
			}
		}
	}
	return
}

func (funcrw *funcrdwtr) Println(a ...interface{}) (err error) {
	if funcrw != nil {
		if bufw := funcrw.writeBuffer(); bufw != nil {
			if err = iorw.Fprintln(bufw, a...); err == nil {
				err = funcrw.Flush()
			}
		}
	}
	return
}

func (funcrw *funcrdwtr) writeBuffer() (bufw *bufio.Writer) {
	if funcrw != nil {
		if funcrw.bufw == nil {
			funcrw.bufw = bufio.NewWriter(funcrw)
		}
		bufw = funcrw.bufw
	}
	return
}

func (funcrw *funcrdwtr) resetBufw() (set bool) {
	if funcrw != nil {
		if funcrw.bufw == nil {
			funcrw.bufw = bufio.NewWriter(funcrw)
		} else {
			funcrw.bufw.Reset(funcrw)
		}
		set = true
	}
	return
}

func (funcrw *funcrdwtr) Write(p []byte) (n int, err error) {
	if funcrw != nil && funcrw.wt != nil {
		n, err = funcrw.wt(p)
	}
	return
}

func (funcrw *funcrdwtr) Close() (err error) {
	if funcrw != nil {
		if funcrw.rd != nil {
			funcrw.rd = nil
		}
		if funcrw.bufr != nil {
			funcrw.bufr = nil
		}
		if funcrw.closeReader != nil {
			err = funcrw.closeReader()
			funcrw.closeReader = nil
		}
		err = funcrw.Flush()
		if funcrw.wt != nil {
			funcrw.wt = nil
		}
		if funcrw.bufw != nil {
			funcrw.bufw = nil
		}
		if funcrw.closeWriter != nil {
			err = funcrw.closeWriter()
			funcrw.closeWriter = nil
		}
		if funcrw.closeAll != nil {
			err = funcrw.closeAll()
			funcrw.closeAll = nil
		}
	}
	return
}

func NewStdioReaderWriter(r io.ReadCloser, initialdur time.Duration, nextdur time.Duration, w io.WriteCloser, closeReader func() error, closeWriter func() error, closeAll func() error) ReaderPrinter {

	funcrw := &funcrdwtr{wt: func(p []byte) (n int, err error) {
		n, err = w.Write(p)
		return
	}, closeReader: closeReader, closeWriter: closeWriter, closeAll: closeAll, crntdur: -1, initialdur: initialdur, nextdur: nextdur}

	if initialdur > 0 && nextdur > 0 && initialdur > nextdur {
		funcrw.crntrerr = make(chan error, 1)
		funcrw.crntrn = make(chan int, 1)
	}
	funcrw.rd = func(p []byte) (n int, err error) {
		n, err = r.Read(p)
		return
	}
	return funcrw
}
