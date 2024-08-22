package web

import (
	"context"
	"io"
	"net/http"
	"strings"

	"github.com/lnksnk/lnksnk/iorw"
	"github.com/lnksnk/lnksnk/iorw/active"

	"github.com/lnksnk/lnksnk/ws"

	"github.com/lnksnk/lnksnk/websocket"
)

type ClientHandle struct {
	*Client
	SendReceive       func(rqstpath string, a ...interface{}) (rw ReaderWriter, err error)
	SendRespondString func(rqstpath string, a ...interface{}) (rspstr string, err error)
	Send              func(rqstpath string, a ...interface{}) (rspr iorw.Reader, err error)
	Close             func()
}

// Client - struct
type Client struct {
	httpclient *http.Client
}

// NewClient - instance
func NewClient() (clnt *Client) {
	clnt = &Client{httpclient: &http.Client{}}
	return
}

// Close *Client
func (clnt *Client) Close() {
	if clnt != nil {
		if clnt.httpclient != nil {
			clnt.httpclient.CloseIdleConnections()
			clnt = nil
		}
		clnt = nil
	}
}

// ReaderWriter interface
type ReaderWriter interface {
	iorw.PrinterReader
	io.ReadWriteCloser
	Flush() error
}

// SendReceive return ReaderWriter that implement io.Reader,io.Writer
func (clnt *Client) SendReceive(rqstpath string, a ...interface{}) (rw ReaderWriter, err error) {
	if strings.HasPrefix(rqstpath, "ws:") || strings.HasPrefix(rqstpath, "wss://") {
		var aok = false
		var ai = 0
		var rntme active.Runtime = nil
		var onsucess interface{} = nil
		var onerror interface{} = nil
		var headers http.Header
		for ai < len(a) {
			d := a[ai]
			if rntme, aok = d.(active.Runtime); aok {
				if ai < len(a)-1 {
					a = append(a[:ai], a[ai+1:]...)
					continue
				} else {
					a = append(a[:ai], a[ai+1:]...)
					break
				}
			} else if mp, mpok := d.(map[string]interface{}); mpok {
				if len(mp) > 0 {
					for k, v := range mp {
						if k == "success" {
							if onsucess == nil {
								onsucess = v
							}
						} else if k == "error" {
							if onerror == nil {
								onerror = v
							}
						} else if k == "headers" {
							if v != nil {
								if hdrs, hdrsok := v.(map[string]interface{}); hdrsok {
									for hdrk, hdrv := range hdrs {
										if hdrv != nil {
											if s, sok := hdrv.(string); sok && s != "" {
												headers.Set(hdrk, s)
											}
										}
									}
								}
							}
						}
					}
				}
				if ai < len(a)-1 {
					a = append(a[:ai], a[ai+1:]...)
					continue
				} else {
					a = append(a[:ai], a[ai+1:]...)
					break
				}
			}
			ai++
		}
		func() {

			var c *websocket.Conn = nil
			var resp *http.Response = nil

			defer func() {
				if err != nil {
					if c != nil {
						c.Close()
						c = nil
					}
				}
				if resp != nil {
					resp = nil
				}
				if c != nil {
					c = nil
				}
			}()

			if rw, resp, err = ws.NewClientReaderWriter(rqstpath, headers); err == nil {
				if rntme != nil && onsucess != nil {
					rntme.InvokeFunction(onsucess, resp)
				}
			} else {
				rw = nil
				if rntme != nil && onerror != nil {
					rntme.InvokeFunction(onerror, err, onerror)
				}
			}
		}()
	}
	return
}

// SendRespondString - Client Send but return response as string
func (clnt *Client) SendRespondString(rqstpath string, a ...interface{}) (rspstr string, err error) {
	var rspr iorw.Reader = nil
	rspstr = ""
	if rspr, err = clnt.Send(rqstpath, a...); err == nil {
		if rspr != nil {
			rspstr, err = rspr.ReadAll()
		}
	}
	return
}

type Response struct {
	Status     string // e.g. "200 OK"
	StatusCode int    // e.g. 200
	Proto      string // e.g. "HTTP/1.0"
	resp       *http.Response
	rdr        *iorw.EOFCloseSeekReader
}

func newresponse(resp *http.Response) (rspns *Response) {
	rspns = &Response{resp: resp,
		Status:     resp.Status,
		StatusCode: resp.StatusCode,
		Proto:      resp.Proto,
	}
	return
}

func (rspns *Response) Cookies() (cookies []*http.Cookie) {
	if rspns != nil && rspns.resp != nil {
		cookies = rspns.resp.Cookies()
	}
	return
}

func (rspns *Response) dispose() {
	if rspns != nil {
		if rspns.resp != nil {
			if rspns.resp.Body != nil {

			}
			rspns.resp = nil
		}
		if rspns.rdr != nil {
			func() {
				defer rspns.rdr.Close()
			}()
			rspns.rdr = nil
		}
	}
}

func (rspns *Response) Reader() (rdr *iorw.EOFCloseSeekReader) {
	if rspns != nil {
		if rspns.rdr == nil {
			if rspns.resp != nil {
				rspns.rdr = iorw.NewEOFCloseSeekReader(rspns.resp.Body)
			} else {
				rspns.rdr = iorw.NewEOFCloseSeekReader(rspns.resp.Body)
			}
		}
		rdr = rspns.rdr
	}
	return
}

func (rspns *Response) Headers() (headers []string) {
	if rspns != nil && rspns.resp != nil {
		if hdrsl := len(rspns.resp.Header); hdrsl > 0 {
			headers = make([]string, hdrsl)
			hdrn := 0
			for hdr := range rspns.resp.Header {
				headers[hdrn] = hdr
				hdrn++
			}
		}
	}
	return
}

func (rspns *Response) Header(hdr string) (val string) {
	if hdr != "" && rspns != nil && rspns.resp != nil && len(rspns.resp.Header) > 0 {
		val = rspns.resp.Header.Get(hdr)
	}
	return
}

// Send - Client send
func (clnt *Client) Send(rqstpath string, a ...interface{}) (rspr iorw.Reader, err error) {
	if strings.HasPrefix(rqstpath, "http:") || strings.HasPrefix(rqstpath, "https://") {
		var method string = ""
		var r io.Reader = nil
		var w io.Writer = nil
		var aok bool = false
		var ai = 0
		var rntme active.Runtime = nil
		var onsucess interface{} = nil
		var onerror interface{} = nil

		var rqstheaders http.Header
		var rspnselts []interface{} = nil
		for err == nil && ai < len(a) {
			d := a[ai]
			if r == nil {
				if rs, rsok := d.(string); rsok {
					if rs != "" {
						if rspnselts == nil {
							rspnselts = []interface{}{}
						}
						rspnselts = append(rspnselts, rs)
					}
					if ai < len(a)-1 {
						a = append(a[:ai], a[ai+1:]...)
						continue
					} else {
						break
					}
				} else if rr, raok := d.(io.Reader); raok {
					if rspnselts == nil {
						rspnselts = []interface{}{}
					}
					rspnselts = append(rspnselts, rr)
					if ai < len(a)-1 {
						a = append(a[:ai], a[ai+1:]...)
						continue
					} else {
						break
					}
				} else if trntme, taok := d.(active.Runtime); taok {
					if rntme == nil {
						rntme = trntme
					}
					if ai < len(a)-1 {
						a = append(a[:ai], a[ai+1:]...)
						continue
					} else {
						a = append(a[:ai], a[ai+1:]...)
						break
					}
				} else if mp, mpok := d.(map[string]interface{}); mpok {
					if len(mp) > 0 {
						for k, v := range mp {
							if k == "body" {
								if v != nil {
									if rspnselts == nil {
										rspnselts = []interface{}{}
									}
									rspnselts = append(rspnselts, v)
								}
							} else if k == "success" {
								if onsucess == nil {
									onsucess = v
								}
							} else if k == "error" {
								if onerror == nil {
									onerror = v
								}
							} else if k == "headers" {
								if v != nil {
									if hdrs, hdrsok := v.(map[string]interface{}); hdrsok {
										for hdrk, hdrv := range hdrs {
											if hdrv != nil {
												if s, sok := hdrv.(string); sok {
													if rqstheaders == nil {
														rqstheaders = http.Header{}
													}
													rqstheaders.Set(hdrk, s)
												}
											}
										}
									}
								}
							} else {
								if s, sok := v.(string); sok && s != "" {
									if k == "method" && method == "" {
										method = strings.ToUpper(s)
									}
								}
							}
						}
					}
					if ai < len(a)-1 {
						a = append(a[:ai], a[ai+1:]...)
						continue
					} else {
						a = append(a[:ai], a[ai+1:]...)
						break
					}
				}
			}
			if w == nil {
				if w, aok = d.(io.Writer); aok {
					if ai < len(a)-1 {
						a = append(a[:ai], a[ai+1:]...)
						continue
					} else {
						break
					}
				}
			}
			ai++
		}

		if len(rspnselts) > 0 {
			pr, pw := io.Pipe()
			ctx, ctxcancel := context.WithCancel(context.Background())
			go func() {
				defer pw.Close()
				ctxcancel()
				iorw.Fprint(pw, rspnselts...)
			}()
			<-ctx.Done()
			r = pr
		} else {
			r = nil
		}

		if r != nil {
			if method == "" || method == "GET" {
				method = "POST"
			}
		} else if method == "" {
			method = "GET"
		}
		var rqst, rqsterr = http.NewRequest(method, rqstpath, r)
		if rqsterr == nil {
			if len(rqstheaders) > 0 {
				for hdk, hdv := range rqstheaders {
					for _, hv := range hdv {
						rqst.Header.Add(hdk, hv)
					}
				}
			}
			func() {
				var rspns *Response = nil
				defer func() {
					if rspns != nil {
						rspns.dispose()
					}
				}()
				var resp, resperr = clnt.Do(rqst)
				if resperr == nil {
					if rntme != nil && onsucess != nil {
						rspns = newresponse(resp)
						rntme.InvokeFunction(onsucess, method, rspns)
					} else {
						if scde := resp.StatusCode; scde >= 200 && scde < 300 {
							if respbdy := resp.Body; respbdy != nil {
								if w != nil {
									ctx, ctxcancel := context.WithCancel(context.Background())
									pi, pw := io.Pipe()
									go func() {
										defer func() {
											pw.Close()
										}()
										ctxcancel()
										if w != nil {
											io.Copy(pw, respbdy)
										}
									}()
									<-ctx.Done()
									ctx = nil
									io.Copy(w, pi)
								} else if rspr == nil {
									rspr = iorw.NewEOFCloseSeekReader(respbdy)
								}
							}
						}
					}
				} else {
					err = resperr
					if rntme != nil && onerror != nil {
						rspns = newresponse(resp)
						rntme.InvokeFunction(onerror, err, method, rspns)
					}
				}
			}()
		} else {
			if rntme != nil && onerror != nil {
				rntme.InvokeFunction(onerror, err)
			}
		}
	}
	return
}

// Do - refer tp http.Client Do interface
func (clnt *Client) Do(rqst *http.Request) (rspnse *http.Response, err error) {
	rspnse, err = clnt.httpclient.Do(rqst)
	return
}

// DefaultClient  - default global web Client
var DefaultClient *Client

func init() {
	DefaultClient = NewClient()
}
