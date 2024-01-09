package serve

import (
	"bufio"
	"io"
	"net/http"

	"github.com/evocert/lnksnk/iorw"
)

type writer struct {
	httpw  http.ResponseWriter
	buff   *bufio.Writer
	Status int
}

func newWriter(httpw http.ResponseWriter) (rqw *writer) {
	rqw = &writer{httpw: httpw, Status: 200}
	return
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
				buff = bufio.NewWriterSize(rqw.httpw, 32768*2)
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
	if rqw != nil {
		n, err = rqw.buffer().Write(p)
	}
	return
}

func (rqw *writer) Print(a ...interface{}) (err error) {
	if rqw != nil {
		if err = iorw.Fprint(rqw.buffer(), a...); err == nil {
			err = rqw.Flush()
		}
	}
	return
}

func (rqw *writer) Println(a ...interface{}) (err error) {
	if rqw != nil {
		if err = iorw.Fprintln(rqw.buffer(), a...); err != nil {
			err = rqw.Flush()
		}
	}
	return
}
