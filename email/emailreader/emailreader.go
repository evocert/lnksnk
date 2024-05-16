package emailreader

import (
	"context"
	"io"
	"strings"
	"time"

	"github.com/evocert/lnksnk/iorw"

	"github.com/evocert/lnksnk/message"
	_ "github.com/evocert/lnksnk/message/charset"
	"github.com/evocert/lnksnk/message/mail"
)

type EmailReader struct {
	mr         *mail.Reader
	lstmltiprt *MailPart
	OnClose    func(*EmailReader) error
}

func (emailrdr *EmailReader) Close() (err error) {
	if emailrdr != nil {
		if emailrdr.mr != nil {
			err = emailrdr.mr.Close()
			emailrdr.mr = nil
		}
		if emailrdr.OnClose != nil {
			emailrdr.OnClose(emailrdr)
			emailrdr.OnClose = nil
		}
		if emailrdr.lstmltiprt != nil {
			emailrdr.lstmltiprt = nil
		}
		emailrdr = nil
	}
	return
}

func (emailrdr *EmailReader) Subject() (subject string, err error) {
	if emailrdr != nil && emailrdr.mr != nil {
		subject, err = emailrdr.mr.Header.Subject()
	}
	return
}

func (emailrdr *EmailReader) Date() (date time.Time, err error) {
	if emailrdr != nil && emailrdr.mr != nil {
		date, err = emailrdr.mr.Header.Date()
	}
	return
}

func (emailrdr *EmailReader) From() (from *mail.Address, err error) {
	if emailrdr != nil && emailrdr.mr != nil {
		if fromadr, fromerr := emailrdr.mr.Header.AddressList("From"); fromerr == nil && len(fromadr) == 1 {
			from = fromadr[0]
		} else if fromerr != nil {
			err = fromerr
		}
	}
	return
}

func (emailrdr *EmailReader) To() (to []*mail.Address, err error) {
	if emailrdr != nil && emailrdr.mr != nil {
		to, err = emailrdr.mr.Header.AddressList("To")
	}
	return
}

func (emailrdr *EmailReader) Cc() (Cc []*mail.Address, err error) {
	if emailrdr != nil && emailrdr.mr != nil {
		Cc, err = emailrdr.mr.Header.AddressList("Cc")
	}
	return
}

func (emailrdr *EmailReader) Bcc() (bcc []*mail.Address, err error) {
	if emailrdr != nil && emailrdr.mr != nil {
		bcc, err = emailrdr.mr.Header.AddressList("Bcc")
	}
	return
}

func ReadMail(a ...interface{}) (emailrdr *EmailReader, err error) {
	if len(a) > 0 {
		var ctx, ctxcnl = context.WithCancel(context.Background())
		var pi, pw = io.Pipe()
		go func() {
			var prerr error = nil
			defer func() {
				if prerr != nil && prerr != io.EOF {
					pw.CloseWithError(prerr)
				} else {
					pw.Close()
				}
			}()
			var crntrunes = make([]rune, 4096)
			var crntrnsi = 0
			var crntstrng = ""
			var prvr = rune(0)
			var argr = iorw.NewMultiArgsReader(a...)
			ctxcnl()
			var crnthdr = ""
			var crnthdrval = ""
			var fndcntnt = false

			var flushcrntrunes = func() {
				if crntrnsi > 0 {
					crntstrng += string(crntrunes[:crntrnsi])
					crntrnsi = 0
				}
			}
			var writingheaders = true
			for writingheaders {
				r, size, rerr := argr.ReadRune()
				if size > 0 {
					if string([]rune{prvr, r}) == "\r\n" || string([]rune{prvr, r}) == "\n" {
						if string([]rune{prvr, r}) == "\r\n" && crntrnsi > 0 && crntrunes[crntrnsi-1] == prvr {
							crntrnsi--
						}
						flushcrntrunes()
						if fndcntnt {
							if crntstrng != "" {
								if crnthdrval == "" {
									crnthdrval = crntstrng
								} else {
									if strings.TrimSpace(crntstrng[:1]) == "" {
										crnthdrval += crntstrng
									} else {
										crnthdrval += " " + crntstrng
									}
								}
								crntstrng = ""
							}
							fndcntnt = false
						} else {
							if crnthdr != "" && crnthdrval != "" {
								if _, prerr = pw.Write([]byte(crnthdr + ": ")); prerr != nil {
									break
								}
								if _, prerr = pw.Write([]byte(crnthdrval)); prerr != nil {
									break
								}
								crnthdr = ""
								crnthdrval = ""
							}
							if _, prerr = pw.Write([]byte("\r\n")); prerr != nil {
								break
							}
							if _, prerr = pw.Write([]byte("\r\n")); prerr != nil {
								break
							}
							writingheaders = false
						}
						prvr = 0
					} else if string([]rune{prvr, r}) == ": " {
						if crntrnsi > 0 && crntrunes[crntrnsi-1] == prvr {
							crntrnsi--
						}
						flushcrntrunes()
						if fndcntnt {
							if crnthdr != "" && crnthdrval != "" {
								if _, prerr = pw.Write([]byte(crnthdr + ": ")); prerr != nil {
									break
								}
								if _, prerr = pw.Write([]byte(crnthdrval)); prerr != nil {
									break
								}
								if _, prerr = pw.Write([]byte("\r\n")); prerr != nil {
									break
								}
								crnthdr = ""
								crnthdrval = ""
							}
							if crntstrng != "" && crnthdr == "" {
								crnthdr = crntstrng
								crntstrng = ""
							}
							fndcntnt = false
						}
						prvr = 0
					} else {
						if !fndcntnt && r != '\r' {
							fndcntnt = true
						}
						prvr = r
						crntrunes[crntrnsi] = prvr
						crntrnsi++
						if crntrnsi == len(crntrunes) {
							flushcrntrunes()
						}
					}
				}
				if rerr != nil {
					prerr = rerr
					break
				}
			}
			if prerr == nil {
				_, prerr = io.Copy(pw, argr)
			}
		}()
		<-ctx.Done()

		if mr, err := mail.CreateReader(pi); err == nil {
			emailrdr = &EmailReader{mr: mr}
			/*for {
				if fcsd, _ := emailrdr.FocusNextPart(); fcsd {
					p := emailrdr.Part()

					contenttype, _ := p.ContentType()
					log.Printf("Content-Type: %v\n", contenttype)

					b, _ := p.Body.ReadAll()
					log.Printf("Got text: %v\n", b)
				} else {
					break
				}
			}*/
		}
	}
	return
}

func (emailrdr *EmailReader) Part() (mailpart *MailPart) {
	if emailrdr != nil {
		mailpart = emailrdr.lstmltiprt
	}
	return
}

func (emailrdr *EmailReader) FocusNextPart() (focused bool, focusederr error) {
	if emailrdr != nil {
		crntprt := emailrdr.Part()
		if nxtPart, nxtparterr := emailrdr.NextPart(); (nxtparterr == nil || nxtparterr == io.EOF) && nxtPart != nil {
			if crntprt == nil {
				if crntprt = nxtPart; crntprt != nil {
					focused = true
				}
			} else if crntprt != nxtPart {
				focused = true
			}
		} else if nxtparterr != nil && nxtparterr != io.EOF {
			focusederr = nxtparterr
		}
	}
	return
}

func (emailrdr *EmailReader) NextPart() (mailpart *MailPart, err error) {
	if emailrdr != nil && emailrdr.mr != nil {
		if prt, prterr := emailrdr.mr.NextPart(); prterr == nil || prterr != nil {
			if prterr == nil && prt != nil {
				if emailrdr.lstmltiprt != nil {
					emailrdr.lstmltiprt = nil
				}

				var header *message.Header = nil
				if hnlne, _ := prt.Header.(*mail.InlineHeader); hnlne != nil {
					header = &hnlne.Header
				}

				mailpart = &MailPart{prt: prt, Header: prt.Header, header: header, Body: iorw.NewEOFCloseSeekReader(prt.Body, false)}
				emailrdr.lstmltiprt = mailpart
			}
			if prterr != nil {
				err = prterr
			}
		}
		if mailpart == nil {
			err = io.EOF
		}
	}
	return
}

type MailPart struct {
	prt    *mail.Part
	Header mail.PartHeader
	header *message.Header
	Body   *iorw.EOFCloseSeekReader
}

func (mltiprt *MailPart) IsContentType(contenttype string) (istype bool, err error) {
	if mltiprt != nil && mltiprt.header != nil {
		if contentType, _, cntyperr := mltiprt.header.ContentType(); contentType != "" && cntyperr == nil {
			istype = strings.Contains(contentType, contenttype)
		} else if cntyperr != nil {
			err = cntyperr
		}
	}
	return
}

func (mltiprt *MailPart) ContentType() (contentType string, err error) {
	if mltiprt != nil && mltiprt.header != nil {
		contentType, _, err = mltiprt.header.ContentType()
	}
	return
}

// Filename parses the attachment's filename.
func (mltiprt *MailPart) Filename() (filename string, err error) {
	if mltiprt != nil && mltiprt.Header != nil {

		if hnlne, _ := mltiprt.Header.(*mail.InlineHeader); hnlne != nil {
			if _, params, prmserr := hnlne.ContentDisposition(); prmserr == nil {

				fname, ok := params["filename"]
				if !ok {
					// Using "name" in Content-Type is discouraged
					if _, params, err = hnlne.ContentType(); prmserr == nil {
						if params["filename"] != "" {
							fname = params["name"]
						}
					}
				}
				if prmserr != nil {
					err = prmserr
				} else {
					filename = fname
				}
			}
		} else if atthdr, _ := mltiprt.Header.(*mail.AttachmentHeader); atthdr != nil {
			if _, params, prmserr := atthdr.ContentDisposition(); prmserr == nil {

				fname, ok := params["filename"]
				if !ok {
					// Using "name" in Content-Type is discouraged
					if _, params, err = atthdr.ContentType(); prmserr == nil {
						if params["filename"] != "" {
							fname = params["name"]
						}
					}
				}
				if prmserr != nil {
					err = prmserr
				} else {
					filename = fname
				}
			}
		}
	}
	return filename, err
}
