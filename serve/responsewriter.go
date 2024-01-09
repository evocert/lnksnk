package serve

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"

	"strings"

	"github.com/evocert/lnksnk/iorw"
)

type responseWriter struct {
	ctx           context.Context
	chunkedWriter io.WriteCloser
	wroteHeader   bool
	bufw          *bufio.Writer
	statusCode    int
	header        http.Header
	orgwtr        io.Writer
	req           *http.Request
}

func NewResponseWriter(req *http.Request, wt io.Writer) (rspnswtr *responseWriter) {
	var orgwtr io.Writer = wt
	rspnswtr = &responseWriter{
		header: http.Header{}, bufw: bufio.NewWriterSize(orgwtr, 65536), orgwtr: orgwtr, statusCode: 200, req: req, ctx: req.Context()}
	return
}

func (w *responseWriter) Header() http.Header {
	return w.header
}

func (w *responseWriter) Flush() {
	if w != nil && w.bufw != nil {
		w.bufw.Flush()
	}
}

func (w *responseWriter) Hijack() (con net.Conn, bufrw *bufio.ReadWriter, err error) {
	if w != nil {
		if con, _ = w.orgwtr.(net.Conn); con != nil {
			bufrw = bufio.NewReadWriter(bufio.NewReader(con), bufio.NewWriter(con))
		} else {
			err = fmt.Errorf("unable to hijack connection")
		}
	} else {
		err = fmt.Errorf("unable to hijack connection")
	}
	return con, bufrw, err
}

func (w *responseWriter) Close() (err error) {
	if w != nil {
		w.Flush()
		if !w.wroteHeader {
			w.writeHeader()
		} else {
			w.wroteHeader = false
		}
		if w.req != nil {
			w.req = nil
		}
		if w.chunkedWriter != nil {
			w.chunkedWriter.Close()
			w.chunkedWriter = nil
		}
		if w.orgwtr != nil {
			if clswtr, _ := w.orgwtr.(io.Closer); clswtr != nil {
				clswtr.Close()
			}
			w.orgwtr = nil
		}
	}
	return err
}

func (w *responseWriter) writeHeader() {
	if !w.wroteHeader {
		w.wroteHeader = true
		if w.bufw != nil {
			if w.req != nil {
				protoHeaderLine := fmt.Sprintf("%s %d %s\r\n", w.req.Proto, w.statusCode, http.StatusText(w.statusCode))
				fmt.Fprint(w.bufw, protoHeaderLine)
				ischunked := false
				if len(w.header) > 0 {
					for hdr, hdv := range w.header {

						iorw.Fprint(w.bufw, hdr+": "+strings.Join(hdv, ";"), "\r\n")
					}
				}
				iorw.Fprint(w.bufw, "\r\n")
				w.Flush()
				if ischunked {
					w.chunkedWriter = httputil.NewChunkedWriter(w.orgwtr)
					w.bufw.Reset(w.chunkedWriter)
				}
			}
		}
	}
}

func (w *responseWriter) Write(b []byte) (n int, err error) {
	if bl := len(b); bl > 0 {
		if w != nil && w.bufw != nil {
			w.writeHeader()
			if w.ctx != nil {
				select {
				case <-w.ctx.Done():
					if err = w.ctx.Err(); err != nil {
						return
					} else {
						err = fmt.Errorf("Interrupted")
						return
					}
				default:
				}
			}
			n, err = w.bufw.Write(b[:bl])
		}
	}
	return n, err
}

func (w *responseWriter) WriteHeader(statusCode int) {
	if w != nil {
		if !w.wroteHeader {
			w.statusCode = statusCode
		}
	}
}
