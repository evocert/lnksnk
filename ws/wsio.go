package ws

import (
	"bufio"
	"context"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/lnksnk/lnksnk/iorw"

	"github.com/lnksnk/lnksnk/websocket"
)

// ReaderWriter - struct
type ReaderWriter struct {
	lcladdr  string
	rmtaddr  string
	ws       *websocket.Conn
	r        io.Reader
	MaxRead  int64
	rbuf     *bufio.Reader
	rerr     error
	w        io.WriteCloser
	wbuf     *bufio.Writer
	werr     error
	isText   bool
	isBinary bool
}

var upgrader = &websocket.Upgrader{
	ReadBufferSize:  256,
	WriteBufferSize: 256,
	WriteBufferPool: &sync.Pool{},
	Error: func(w http.ResponseWriter, r *http.Request, status int, reason error) {

	},
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// NewServerReaderWriter - instance
func NewServerReaderWriter(w http.ResponseWriter, r *http.Request) (wsrw *ReaderWriter, err error) {
	if w != nil && r != nil {
		if websocket.IsWebSocketUpgrade(r) && r.Method == "GET" {
			wsu := upgrader
			if ws, wserr := wsu.Upgrade(w, r, nil); wserr == nil {
				wsrw = &ReaderWriter{ws: ws, lcladdr: ws.LocalAddr().String(), rmtaddr: ws.RemoteAddr().String(), isText: false, isBinary: false, rerr: nil, werr: nil, MaxRead: -1}
			} else {
				err = wserr
			}
			wsu = nil
		}
	}
	return
}

func NewClientReaderWriter(rqstpath string, headers http.Header) (wsrw *ReaderWriter, resp *http.Response, err error) {
	if rqstpath != "" {
		var ws *websocket.Conn = nil
		if ws, resp, err = websocket.DefaultDialer.Dial(rqstpath, headers); err == nil && ws != nil {
			wsrw = &ReaderWriter{ws: ws, lcladdr: ws.LocalAddr().String(), rmtaddr: ws.RemoteAddr().String(), isText: false, isBinary: false, rerr: nil, werr: nil, MaxRead: -1}
		}
	}
	return
}

func (wsrw *ReaderWriter) LocalAddr() string {
	if wsrw != nil {
		return wsrw.lcladdr
	}
	return ""
}

func (wsrw *ReaderWriter) RemoteAddr() string {
	if wsrw != nil {
		return wsrw.rmtaddr
	}
	return ""
}

// SetMaxRead - set max read implementation for Reader interface compliance
func (wsrw *ReaderWriter) SetMaxRead(maxlen int64) (err error) {
	if wsrw != nil {
		if maxlen < 0 {
			maxlen = -1
		}
		wsrw.MaxRead = maxlen
	}
	return
}

// ReadRune - refer to io.RuneReader
func (wsrw *ReaderWriter) ReadRune() (r rune, size int, err error) {
	if wsrw != nil {
		if wsrw.rbuf == nil {
			wsrw.rbuf = bufio.NewReader(wsrw)
		}
		r, size, err = wsrw.rbuf.ReadRune()
	} else {
		r, size, err = 0, 0, io.EOF
	}
	return
}

// WriteRune - refer to bufio.Writer - WriteRune
func (wsrw *ReaderWriter) WriteRune(r rune) (size int, err error) {
	if wsrw != nil {
		if wsrw.wbuf == nil {
			wsrw.wbuf = bufio.NewWriter(wsrw)
		}
		size, err = wsrw.wbuf.WriteRune(r)
	}
	return
}

// CanRead - can Read
func (wsrw *ReaderWriter) CanRead() bool {
	return wsrw != nil && wsrw.rerr == nil
}

// CanWrite - can Write
func (wsrw *ReaderWriter) CanWrite() bool {
	return wsrw != nil && wsrw.werr == nil
}

// Read - refer io.Reader
func (wsrw *ReaderWriter) Read(p []byte) (n int, err error) {
	if wsrw != nil {
		if wsrw.MaxRead == 0 {
			err = io.EOF
		} else {
			if pl := len(p); pl > 0 {
				if wsrw.MaxRead > 0 {
					if int64(pl) >= wsrw.MaxRead {
						pl = int(wsrw.MaxRead)
					}
				}
				if wsrw.r == nil {
					if err = wsrw.Flush(); err == nil {
						if wsrw.CanRead() {
							var messageType int
							var rdr io.Reader = nil

							messageType, rdr, wsrw.rerr = wsrw.ws.NextReader()
							wsrw.isText = messageType == websocket.TextMessage
							wsrw.isBinary = messageType == websocket.BinaryMessage
							if wsrw.rerr != nil {
								if wsrw.rerr != io.EOF {
									return 0, wsrw.rerr
								}
								return 0, io.EOF
							}
							if rdr != nil {
								ctx, ctxcancel := context.WithCancel(context.Background())
								pr, pw := io.Pipe()
								go func() {
									var pwerr error = nil
									defer func() {
										if pwerr != io.EOF {
											pw.CloseWithError(pwerr)
										} else {
											pw.Close()
										}
									}()
									ctxcancel()
									_, pwerr = io.Copy(pw, rdr)
								}()
								<-ctx.Done()
								ctx = nil
								wsrw.r = pr
							}
						}
					} else {
						return 0, io.EOF
					}
				}
				for n = 0; n < len(p[:pl]); {
					var m int
					m, err = wsrw.r.Read(p[n:])
					if m > 0 && wsrw.MaxRead > 0 {
						wsrw.MaxRead -= int64(m)
						if wsrw.MaxRead < 0 {
							wsrw.MaxRead = 0
						}
					}
					n += m
					if err != nil {
						if err == io.EOF {
							wsrw.r = nil
							break
						} else {
							wsrw.rerr = err
						}
					}
					if err != nil {
						break
					}
				}

				if n == 0 && err == nil {
					err = io.EOF
				}
			}
		}
	} else {
		err = io.EOF
	}
	return
}

// Seek - empty implementation refer to iorw.Reader
func (wsrw *ReaderWriter) Seek(offset int64, whence int) (n int64, err error) {
	return
}

// Readln - read single line
func (wsrw *ReaderWriter) Readln() (s string, err error) {
	s = ""
	if wsrw != nil {
		var rns = make([]rune, 1024)
		var rnsi = 0
		for {
			rn, size, rnerr := wsrw.ReadRune()
			if size > 0 {
				if rn == rune(10) {
					if rnsi > 0 {
						s += string(rns[:rnsi])
						rnsi = 0
					}
					break
				} else {
					rns[rnsi] = rn
					rnsi++
					if rnsi == len(rns) {
						s += string(rns[:rnsi])
						rnsi = 0
					}
				}
			}
			if rnerr != nil {
				if rnerr != io.EOF {
					err = rnerr
				}
				break
			}
		}
		if s == "" && rnsi > 0 {
			s += string(rns[:rnsi])
			rnsi = 0
		}
		if s != "" {
			s = strings.TrimSpace(s)
		}
	}
	return
}

// Readlines - return lines []string slice
func (wsrw *ReaderWriter) ReadLines() (lines []string, err error) {
	var line = ""
	if wsrw != nil {
		for {
			if line, err = wsrw.Readln(); line != "" && (err == nil || err == io.EOF) {
				if lines == nil {
					lines = []string{}
				}
				lines = append(lines, line)
			}
			if err != nil {
				if err == io.EOF {
					err = nil
				}
				break
			}
		}
	}
	return
}

// ReadAll - return all read content as string
func (wsrw *ReaderWriter) ReadAll() (s string, err error) {
	s = ""
	if wsrw != nil {
		var rns = make([]rune, 1024)
		var rnsi = 0
		for {
			rn, size, rnerr := wsrw.ReadRune()
			if size > 0 {
				rns[rnsi] = rn
				rnsi++
				if rnsi == len(rns) {
					s += string(rns[:rnsi])
					rnsi = 0
				}
			}
			if rnerr != nil {
				if rnsi > 0 {
					s += string(rns[:rnsi])
					rnsi = 0
				}
				if rnerr != io.EOF {
					err = rnerr
				}
				break
			}
		}
	}
	return
}

func (wsrw *ReaderWriter) socketIOType() int {
	if wsrw != nil {
		if wsrw.isText {
			return websocket.TextMessage
		} else if wsrw.isBinary {
			return websocket.BinaryMessage
		}
	}
	return websocket.TextMessage
}

// Flush - flush invoke done onmessage
func (wsrw *ReaderWriter) Flush() (err error) {
	if wsrw != nil {
		if wsrw.wbuf != nil {
			if err = wsrw.wbuf.Flush(); err != nil {
				return
			}
		}
		if wsrw.w != nil {
			err = wsrw.w.Close()
			wsrw.w = nil
		}
	}
	return
}

// Print - refer to fmt.Fprint
func (wsrw *ReaderWriter) Print(a ...interface{}) (err error) {
	if wsrw != nil {
		if err = iorw.Fprint(wsrw, a...); err == nil {
			err = wsrw.Flush()
		}
	}
	return
}

// Println - refer to fmt.Fprintln
func (wsrw *ReaderWriter) Println(a ...interface{}) (err error) {
	if wsrw != nil {
		if err = iorw.Fprintln(wsrw, a...); err == nil {
			err = wsrw.Flush()
		}
	}
	return
}

// Write - refer io.Writer
func (wsrw *ReaderWriter) Write(p []byte) (n int, err error) {
	if pl := len(p); pl > 0 {
		if wsrw != nil {
			if wsrw.w == nil && wsrw.CanWrite() {
				wsrw.w, wsrw.werr = wsrw.ws.NextWriter(wsrw.socketIOType())
				if wsrw.werr != nil {
					err = wsrw.werr
					return 0, err
				}
			}
			for n = 0; n < len(p); {
				var m int
				m, err = wsrw.w.Write(p[n : n+(len(p)-n)])
				n += m
				if err != nil {
					break
				}
			}
		}
		if n == 0 && err == nil {
			err = io.EOF
		}
	}
	return
}

// Close - refer io.Closer
func (wsrw *ReaderWriter) Close() (err error) {
	if wsrw != nil {
		if wsrw.r != nil {
			wsrw.r = nil
		}
		if wsrw.w != nil {
			wsrw.w.Close()
			wsrw.w = nil
		}
		if wsrw.rbuf != nil {
			wsrw.rbuf = nil
		}
		if wsrw.wbuf != nil {
			wsrw.wbuf = nil
		}
		if wsrw.ws != nil {
			err = wsrw.ws.Close()
			wsrw.ws = nil
		}
		wsrw = nil
	}
	return
}
