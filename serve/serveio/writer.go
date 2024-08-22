package serveio

import (
	"bufio"
	"io"
	"net/http"

	"github.com/lnksnk/lnksnk/iorw"
)

type Writer interface {
	io.WriteCloser
	WriteHeader(int)
	Flush() error
	Header() http.Header
	Print(...interface{}) error
	BPrint(...interface{}) error
	Println(...interface{}) error
	ReadFrom(r io.Reader) (n int64, err error)
	MaxWriteSize(int64) bool
}

type writer struct {
	httpw   http.ResponseWriter
	buff    *bufio.Writer
	Status  int
	MaxSize int64
}

func NewWriter(httpw http.ResponseWriter) (rqw *writer) {
	rqw = &writer{httpw: httpw, Status: 200, MaxSize: -1}
	return
}

func (rqw *writer) MaxWriteSize(maxsize int64) bool {
	if rqw == nil {
		return false
	}
	if rqw.MaxSize == -1 {
		rqw.MaxSize = maxsize
		return true
	}
	return false
}

func (rqw *writer) ReadFrom(r io.Reader) (n int64, err error) {
	if rqw != nil {
		if rqw.httpw != nil {
			if rqw.buff != nil {
				if err = rqw.Flush(); err != nil {
					return
				}
			}
			n, err = iorw.ReadToFunc(rqw.httpw, r.Read)
		}
	}
	return
}

func (rqw *writer) Header() http.Header {
	if rqw != nil {
		if httpw := rqw.httpw; httpw != nil {
			return httpw.Header()
		}
	}
	return nil
}

func (rqw *writer) WriteHeader(status int) {
	if rqw != nil {
		if rqw.httpw != nil {
			if status == 0 {
				status = rqw.Status
			}
			rqw.httpw.WriteHeader(status)
		}
	}
}

func (rqw *writer) Flush() (err error) {
	if rqw != nil {
		if buff, httpw := rqw.buff, rqw.httpw; buff != nil && httpw != nil {
			if err = buff.Flush(); err == nil {
				if fuslhr, _ := httpw.(http.Flusher); fuslhr != nil {
					fuslhr.Flush()
				}
			}
		}
	}
	return
}

func (rqw *writer) buffer() *bufio.Writer {
	if rqw != nil {
		if buff := rqw.buff; buff == nil {
			if rqw.httpw != nil {
				bfsize := 32768 * 2
				buff = bufio.NewWriterSize(rqw.httpw, bfsize)
				rqw.buff = buff
			}
			return buff
		} else {
			return buff
		}
	}
	return nil
}

func (rqw *writer) Close() (err error) {
	if rqw != nil {
		if buff, httpw := rqw.buff, rqw.httpw; buff != nil || httpw != nil {
			rqw.Flush()
			rqw.buff = nil
			rqw.httpw = nil
			if fuslhr, _ := httpw.(http.Flusher); fuslhr != nil {
				fuslhr.Flush()
			}
		}
	}
	return
}

func (rqw *writer) Write(p []byte) (n int, err error) {
	if pl := len(p); rqw != nil && pl > 0 {
		if buf := rqw.buffer(); buf != nil {
			maxsize := rqw.MaxSize
			if maxsize > 0 {
				if int64(pl) >= maxsize {
					pl = int(maxsize)
				}
				if n, err = rqw.buffer().Write(p[:pl]); n > 0 {
					rqw.MaxSize -= int64(n)
				}
				return
			}
			if maxsize == -1 {
				n, err = rqw.buffer().Write(p)
				return
			}
		}
	}
	return 0, io.EOF
}

func (rqw *writer) Print(a ...interface{}) (err error) {
	if rqw != nil {
		if err = iorw.Fprint(rqw, a...); err == nil {
			err = rqw.Flush()
		}
	}
	return
}

func (rqw *writer) BPrint(a ...interface{}) (err error) {
	if rqw != nil {
		if err = iorw.Fbprint(rqw, a...); err == nil {
			err = rqw.Flush()
		}
	}
	return
}

func (rqw *writer) Println(a ...interface{}) (err error) {
	if rqw != nil {
		if err = iorw.Fprintln(rqw, a...); err != nil {
			err = rqw.Flush()
		}
	}
	return
}
