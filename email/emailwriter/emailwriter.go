package emailwriter

import (
	"io"
	"log"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/evocert/lnksnk/iorw"
	"github.com/evocert/lnksnk/mimes"

	_ "github.com/emersion/go-message/charset"
	"github.com/emersion/go-message/mail"
)

type EmailWriter struct {
	buffer      *iorw.Buffer
	From        string
	To          []string
	Cc          []string
	Bcc         []string
	Subject     string
	message     *iorw.Buffer
	htmlmessage *iorw.Buffer
	mailHeader  *mail.Header
	mw          *mail.Writer
	OnClose     func(*EmailWriter) error
	attchmntlck sync.RWMutex
	attachments map[string]*attachement
	attchments  map[*attachement]*attachement
}

func (emailwtr *EmailWriter) Message() (message *iorw.Buffer) {
	if emailwtr != nil {
		if emailwtr.message == nil {
			emailwtr.message = iorw.NewBuffer()
		}
		message = emailwtr.message
	}
	return
}

func (emailwtr *EmailWriter) HtmlMessage() (htmlmessage *iorw.Buffer) {
	if emailwtr != nil {
		if emailwtr.htmlmessage == nil {
			emailwtr.htmlmessage = iorw.NewBuffer()
		}
		htmlmessage = emailwtr.htmlmessage
	}
	return
}

func (emailwtr *EmailWriter) AllRecipients(filteraddresses ...string) (recipients []string, err error) {
	if emailwtr != nil && emailwtr.mailHeader != nil {
		var AddrFieldsToUse = strings.Split("From,To,Sender,Reply-To,Bcc,Cc", ",")
		var AddrFldsToUseL = len(AddrFieldsToUse)
		if fltrAddrsL := len(filteraddresses); fltrAddrsL > 0 {
			var fltrAddrI = 0
			var AddrFldsI = 0
			var foundAddr = false
			for fltrAddrI < fltrAddrsL {
				foundAddr = false
				AddrFldsI = 0
				for AddrFldsI < AddrFldsToUseL && !foundAddr {
					if strings.EqualFold(AddrFieldsToUse[AddrFldsI], filteraddresses[fltrAddrI]) {
						filteraddresses[fltrAddrI] = AddrFieldsToUse[AddrFldsI]
						//fltrAddrI++
						foundAddr = true
						break
					}
					AddrFldsI++
				}
				if !foundAddr {
					filteraddresses = append(filteraddresses[:fltrAddrI], filteraddresses[fltrAddrI+1:]...)
					fltrAddrsL--
					continue
				}
				fltrAddrI++
			}
			AddrFldsToUseL = fltrAddrsL
			if fltrAddrsL > 0 {
				AddrFieldsToUse = filteraddresses[:fltrAddrsL]
			}
		}
		if AddrFldsToUseL > 0 {
			for _, addrfields := range AddrFieldsToUse {
				if addrsses, addrsseserr := emailwtr.mailHeader.AddressList(addrfields); addrsseserr == nil && len(addrsses) > 0 {
					for _, addr := range addrsses {
						recipients = append(recipients, addr.Address)
					}
				}
			}
		}
	}
	return
}

func stringAddressesToMailAddresses(address ...string) (mailaddress []*mail.Address) {
	if len(address) > 0 {
		for _, addr := range address {
			if nextaddr, _ := mail.ParseAddress(addr); nextaddr != nil {
				mailaddress = append(mailaddress, nextaddr)
			}
		}
	}
	return
}

func (emailwtr *EmailWriter) prepHeader() {
	if emailwtr != nil {
		if emailwtr.mailHeader == nil {
			if emailwtr.From, emailwtr.Subject = strings.TrimSpace(emailwtr.From), strings.TrimSpace(emailwtr.Subject); emailwtr.From != "" && emailwtr.Subject != "" {

				var from, to, cc, bcc = stringAddressesToMailAddresses(emailwtr.From), stringAddressesToMailAddresses(emailwtr.To...), stringAddressesToMailAddresses(emailwtr.Cc...), stringAddressesToMailAddresses(emailwtr.Bcc...)

				if len(from) > 0 && len(to) > 0 {
					emailwtr.mailHeader = &mail.Header{}

					emailwtr.mailHeader.SetAddressList("From", from)
					emailwtr.mailHeader.SetAddressList("To", to)
					if len(cc) > 0 {
						emailwtr.mailHeader.SetAddressList("Cc", cc)
					}
					if len(bcc) > 0 {
						emailwtr.mailHeader.SetAddressList("Bcc", bcc)
					}
					emailwtr.mailHeader.SetDate(time.Now())
					emailwtr.mailHeader.SetSubject(emailwtr.Subject)
				}
			}
		}
	}
}

func (emailwtr *EmailWriter) mailwtr() (mw *mail.Writer, err error) {
	if emailwtr != nil {
		if emailwtr.mw == nil {
			emailwtr.prepHeader()
			if emailwtr.mailHeader != nil {
				if emailwtr.buffer == nil {
					emailwtr.buffer = iorw.NewBuffer()
				}
				if emailwtr.mw, err = mail.CreateWriter(emailwtr.buffer, *emailwtr.mailHeader); emailwtr.mw != nil {

					if emailwtr.htmlmessage != nil && emailwtr.htmlmessage.Size() > 0 {
						var h1 mail.InlineHeader
						h1.SetContentType("text/html", nil)
						if t1, err := emailwtr.mw.CreateInline(); err == nil {
							w1, err := t1.CreatePart(h1)
							if err != nil {
								log.Fatal(err)
							}
							emailwtr.htmlmessage.WriteTo(w1)
							w1.Close()
							t1.Close()
						}
					}
					if emailwtr.message != nil && emailwtr.message.Size() > 0 {
						var h2 mail.InlineHeader
						h2.SetContentType("text/plain", nil)
						if t2, err := emailwtr.mw.CreateInline(); err == nil {
							w2, err := t2.CreatePart(h2)
							if err != nil {
								log.Fatal(err)
							}
							emailwtr.message.WriteTo(w2)
							w2.Close()
							t2.Close()
						}
					}
					for _, attchmnt := range emailwtr.attachments {
						func() {
							defer func() {
								attchmnt.close()
							}()

							if attchmnt.buffer != nil && attchmnt.buffer.Size() > 0 {
								if attw, attwerr := emailwtr.mw.CreateAttachment(*attchmnt.header); attwerr == nil {
									defer attw.Close()
									attrdr := attchmnt.buffer.Reader()
									defer attrdr.Close()
									io.Copy(attw, attrdr)
								}
							}
						}()
					}
				}
			}
			mw = emailwtr.mw
		}
	}
	return
}

func (emailwtr *EmailWriter) Reader() (rdr *iorw.BuffReader) {
	if emailwtr != nil && emailwtr.buffer != nil {
		if emailwtr.buffer.Size() == 0 {
			emailwtr.mailwtr()
		}
		rdr = emailwtr.buffer.Reader()
	}
	return
}

func (emailwtr *EmailWriter) AddAttachment(filename string, a ...interface{}) (attchmnt *attachement, err error) {
	if emailwtr != nil && filename != "" {
		emailwtr.attchmntlck.Lock()
		defer emailwtr.attchmntlck.Unlock()
		if emailwtr.attachments == nil {
			emailwtr.attachments = map[string]*attachement{}
		}
		if emailwtr.attchments == nil {
			emailwtr.attchments = map[*attachement]*attachement{}
		}
		if attchmnt, _ = emailwtr.attachments[filename]; attchmnt == nil {
			attchmnt = newAttachement(filename, a...)
			attchmnt.onclose = func(attchmntref *attachement) {
				emailwtr.attachments[attchmntref.Filename] = nil
				delete(emailwtr.attachments, attchmntref.Filename)
				emailwtr.attchments[attchmntref] = nil
				delete(emailwtr.attchments, attchmntref)
			}
			emailwtr.attachments[filename] = attchmnt
			emailwtr.attchments[attchmnt] = attchmnt

		}
	}
	return
}

func (emailwtr *EmailWriter) AttachementNames() (attmntnames []string) {
	if emailwtr != nil && len(emailwtr.attachments) > 0 {
		func() {
			emailwtr.attchmntlck.RLock()
			defer emailwtr.attchmntlck.RUnlock()
			for attchmntnme := range emailwtr.attachments {
				attmntnames = append(attmntnames, attchmntnme)
			}
		}()
	}
	return
}

func (emailwtr *EmailWriter) Attachements(fnames ...string) (attmnts []*attachement) {
	if emailwtr != nil && len(emailwtr.attachments) > 0 {
		func() {
			emailwtr.attchmntlck.RLock()
			defer emailwtr.attchmntlck.RUnlock()
			if len(emailwtr.attachments) > 0 {
				var attchmntnmes []string = nil
				var attchmnts []*attachement = nil
				var fnamesl = len(fnames)
				for attchmntnme, attchmnt := range emailwtr.attachments {
					if fnamesl > 0 {
						attchmntnmes = append(attchmntnmes, attchmntnme)
					}
					attchmnts = append(attmnts, attchmnt)
				}
				if fnamesl == 0 {
					attmnts = attchmnts[:]
				} else {
					var fnamei = 0
					var fnamefnd = false
					for fnamei < fnamesl {
						for attchmntnmen, attchmntnme := range attchmntnmes {
							if strings.EqualFold(fnames[fnamei], attchmntnme) {
								fnamei++
								fnamefnd = true
								attmnts = append(attmnts, attmnts[attchmntnmen])
								break
							}
						}
						if !fnamefnd {
							fnames = append(fnames[:fnamei], fnames[fnamei+1:]...)
							fnamesl--
						} else {
							fnamefnd = false
						}
					}
				}
			}
		}()
	}
	return
}

func (emailwtr *EmailWriter) Attachement(filename string) (attchmnt *attachement) {
	if emailwtr != nil && filename != "" {
		emailwtr.attchmntlck.RLock()
		defer emailwtr.attchmntlck.RUnlock()
		if len(emailwtr.attachments) > 0 {
			for fname, attmnt := range emailwtr.attachments {
				if strings.EqualFold(fname, filename) {
					attchmnt = attmnt
					break
				}
			}
		}
	}
	return
}

type attachement struct {
	Filename string
	header   *mail.AttachmentHeader
	buffer   *iorw.Buffer
	onclose  func(*attachement)
}

func (attchmnt *attachement) close() {
	if attchmnt != nil {
		if attchmnt.onclose != nil {
			attchmnt.onclose(attchmnt)
			attchmnt.onclose = nil
		}
		if attchmnt.header != nil {
			attchmnt.header = nil
		}
		if attchmnt.buffer != nil {
			attchmnt.buffer.Close()
			attchmnt.buffer = nil
		}

	}
}

func (attchmnt *attachement) setHeader(header string, value string) {
	if attchmnt != nil {
		if attchmnt.header == nil {
			attchmnt.header = &mail.AttachmentHeader{}
		}
	}
}

func (attchmnt *attachement) Print(a ...interface{}) (err error) {
	if attchmnt != nil && len(a) > 0 {
		if attchmnt.buffer == nil {
			attchmnt.buffer = iorw.NewBuffer()
		}
		err = attchmnt.buffer.Print(a...)
	}
	return
}

func (attchmnt *attachement) Println(a ...interface{}) (err error) {
	if attchmnt != nil && len(a) > 0 {
		if attchmnt.buffer == nil {
			attchmnt.buffer = iorw.NewBuffer()
		}
		err = attchmnt.buffer.Println(a...)
	}
	return
}

func newAttachement(filename string, a ...interface{}) (attchmt *attachement) {
	if filename = strings.TrimSpace(filename); filename != "" {
		if fileext := filepath.Ext(filename); fileext != "" {
			if filecntnttpe := mimes.ExtMimeType(fileext, ""); filecntnttpe != "" {
				attchmt = &attachement{header: &mail.AttachmentHeader{}, Filename: filename}
				attchmt.header.Set("ContentType", filecntnttpe)
				attchmt.header.SetFilename(filename)
				attchmt.Print(a...)
			}
		}
	}
	return
}

func (emailwtr *EmailWriter) SetTo(to ...string) {
	if emailwtr != nil && len(to) > 0 {
		var toChecked = map[string]bool{}
		var toi, tol = 0, len(to)
		var hasTo = len(emailwtr.To) > 0
		for toi < tol {
			to[toi] = strings.TrimSpace(to[toi])
			if to[toi] == "" {
				tol--
				to = append(to[:toi], to[toi+1:]...)
				continue
			} else if hasTo {
				for _, chkto := range emailwtr.To {
					if !toChecked[chkto] {
						if tol > 0 && strings.EqualFold(to[toi], chkto) {
							toChecked[chkto] = true
							tol--
							to = append(to[:toi], to[toi+1:]...)
						}
					}
					if tol == 0 {
						break
					}
				}
			} else {
				emailwtr.To = append(emailwtr.To, to[toi])
				toi++
			}
		}
		if tol > 0 {
			emailwtr.To = append(emailwtr.To, to...)
		}
	}
}

func (emailwtr *EmailWriter) SetCC(cc ...string) {
	if emailwtr != nil && len(cc) > 0 {
		var ccChecked = map[string]bool{}
		var cci, ccl = 0, len(cc)
		var hasCc = len(emailwtr.Cc) > 0
		for cci < ccl {
			cc[cci] = strings.TrimSpace(cc[cci])
			if cc[cci] == "" {
				ccl--
				cc = append(cc[:cci], cc[cci+1:]...)
				continue
			} else if hasCc {
				for _, chkcc := range emailwtr.Cc {
					if !ccChecked[chkcc] {
						if ccl > 0 && strings.EqualFold(cc[cci], chkcc) {
							ccChecked[chkcc] = true
							ccl--
							cc = append(cc[:cci], cc[cci+1:]...)
						}
					}
					if ccl == 0 {
						break
					}
				}
			} else {
				emailwtr.Cc = append(emailwtr.Cc, cc[cci])
				cci++
			}
		}
		if ccl > 0 {
			emailwtr.Cc = append(emailwtr.Cc, cc...)
		}
	}
}

func (emailwtr *EmailWriter) SetBcc(bcc ...string) {
	if emailwtr != nil && len(bcc) > 0 {
		var bccChecked = map[string]bool{}
		var bcci, bccl = 0, len(bcc)
		var hasBcc = len(emailwtr.Bcc) > 0
		for bcci < bccl {
			bcc[bcci] = strings.TrimSpace(bcc[bcci])
			if bcc[bcci] == "" {
				bccl--
				bcc = append(bcc[:bcci], bcc[bcci+1:]...)
				continue
			} else if hasBcc {
				for _, chkbcc := range emailwtr.Bcc {
					if !bccChecked[chkbcc] {
						if bccl > 0 && strings.EqualFold(bcc[bcci], chkbcc) {
							bccChecked[chkbcc] = true
							bccl--
							bcc = append(bcc[:bcci], bcc[bcci+1:]...)
						}
					}
					if bccl == 0 {
						break
					}
				}
			} else {
				emailwtr.Bcc = append(emailwtr.Bcc, bcc[bcci])
				bcci++
			}
		}
		if bccl > 0 {
			emailwtr.Bcc = append(emailwtr.Bcc, bcc...)
		}
	}
}

func (emailwtr *EmailWriter) ConstructMail() (err error) {
	if emailwtr != nil {
		emailwtr.mailwtr()
	}
	return
}

func (emailwtr *EmailWriter) Close() (err error) {
	if emailwtr != nil {
		if emailwtr.mw != nil {
			err = emailwtr.mw.Close()
			emailwtr.mw = nil
		}
		if emailwtr.OnClose != nil {
			emailwtr.OnClose(emailwtr)
			emailwtr.OnClose = nil
		}
		if emailwtr.mailHeader != nil {
			emailwtr.mailHeader = nil
		}
		if emailwtr.buffer != nil {
			emailwtr.buffer.Close()
			emailwtr.buffer = nil
		}
		if emailwtr.attachments != nil {
			for _, atchmntnme := range emailwtr.AttachementNames() {
				attchmnt := emailwtr.attachments[atchmntnme]
				attchmnt.close()
			}
		}
		emailwtr = nil
	}
	return
}
