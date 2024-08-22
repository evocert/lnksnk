package smtplib

import (
	"io"
	"strconv"
	"strings"

	"github.com/lnksnk/lnksnk/email/emailwriter"

	"github.com/lnksnk/lnksnk/sasl"
	smtpe "github.com/lnksnk/lnksnk/smtp"
)

type SMTPClient struct {
	username string
	password string
	host     string
	mode     string
}

func SendMail(username string, password string, host string, mode string, modeoptions map[string]interface{}, a ...interface{}) {
	if al := len(a); al > 0 {
		var emailwtrs = []*emailwriter.EmailWriter{}
		if al := len(a); al > 0 {
			var ai = 0
			var ampToWriter = func(msgmap map[string]interface{}) (emlwtr *emailwriter.EmailWriter) {
				if len(msgmap) > 0 {
					var messagetxt = ""
					var subject = ""
					var foundSubject = false
					var foundMessageTxt = false
					var messagehtml = ""
					var foundMessageHtml = false
					var from = []string{}
					var foundFrom = false
					var to = []string{}
					var foundTo = false
					var cc = []string{}
					var bcc = []string{}
					var attatchments = []map[string]interface{}{}
					var foundAttachements = false
					for msgk, msgv := range msgmap {
						if strings.EqualFold(msgk, "subject") {
							if msgs, _ := msgv.(string); !foundSubject {
								foundSubject = true
								subject = msgs
							}
						} else if strings.EqualFold(msgk, "message") {
							if msgs, _ := msgv.(string); !foundMessageTxt {
								foundMessageTxt = true
								messagetxt = msgs
							}
						} else if strings.EqualFold(msgk, "html-message") {
							if msgs, _ := msgv.(string); !foundMessageHtml {
								foundMessageHtml = true
								messagehtml = msgs
							}
						} else if strings.EqualFold(msgk, "from") {
							if !foundFrom {
								if froms, _ := msgv.(string); froms != "" {
									from = append(from, froms)
									foundFrom = true
								} else if fromarr, _ := msgv.([]string); len(fromarr) > 0 {
									for _, froms = range fromarr {
										if froms != "" && !foundFrom {
											from = append(from, froms)
											foundFrom = true
											break
										}
									}
								} else if fromarr, _ := msgv.([]interface{}); len(fromarr) > 0 {
									for _, froma := range fromarr {
										if froms, _ := froma.(string); froms != "" && !foundFrom {
											from = append(from, froms)
											foundFrom = true
											break
										}
									}
								}
							}
						} else if strings.EqualFold(msgk, "to") {
							if tos, _ := msgv.(string); tos != "" {
								to = append(to, tos)
							} else if toarr, _ := msgv.([]string); len(toarr) > 0 {
								for _, tos = range toarr {
									if tos != "" {
										to = append(to, tos)
									}
								}
							} else if toarr, _ := msgv.([]interface{}); len(toarr) > 0 {
								for _, toa := range toarr {
									if tos, _ = toa.(string); tos != "" {
										to = append(to, tos)
									}
								}
							}
						} else if strings.EqualFold(msgk, "cc") {
							if ccs, _ := msgv.(string); ccs != "" {
								cc = append(cc, ccs)
							} else if ccarr, _ := msgv.([]string); len(ccarr) > 0 {
								for _, ccs = range ccarr {
									if ccs != "" {
										cc = append(cc, ccs)
									}
								}
							} else if ccarr, _ := msgv.([]interface{}); len(ccarr) > 0 {
								for _, cca := range ccarr {
									if ccs, _ = cca.(string); ccs != "" {
										cc = append(cc, ccs)
									}
								}
							}
						} else if strings.EqualFold(msgk, "bcc") {
							if bccs, _ := msgv.(string); bccs != "" {
								bcc = append(bcc, bccs)
							} else if bccarr, _ := msgv.([]string); len(bccarr) > 0 {
								for _, bccs = range bccarr {
									if bccs != "" {
										bcc = append(bcc, bccs)
									}
								}
							} else if bccarr, _ := msgv.([]interface{}); len(bccarr) > 0 {
								for _, bcca := range bccarr {
									if bccs, _ = bcca.(string); bccs != "" {
										bcc = append(bcc, bccs)
									}
								}
							}
						} else if strings.EqualFold(msgk, "attachedments") {
							if attarr, _ := msgv.([]interface{}); len(attarr) > 0 {
								for _, attr := range attarr {
									if attmp, _ := attr.(map[string]interface{}); len(attmp) > 0 {
										//attatchments = append(attatchments, attr)
									}
								}
							}
						}
					}
					if !foundFrom {
						foundFrom = true
						from = []string{username}
					}
					foundTo = len(to) > 0
					foundAttachements = len(attatchments) > 0
					if foundFrom && foundTo && foundSubject {
						emlwtr = &emailwriter.EmailWriter{}
						emlwtr.From = from[0]
						emlwtr.To = to
						emlwtr.Subject = subject
						emlwtr.Bcc = bcc
						emlwtr.Cc = cc
						if foundAttachements {

						}
						if messagetxt != "" {
							emlwtr.Message().Print(messagetxt)
						}
						if messagehtml != "" {
							emlwtr.Message().Print(messagehtml)
						}
						emlwtr.ConstructMail()
					}
				}
				return
			}
			for ai < al {
				if d := a[ai]; d != nil {
					if msgmap, _ := d.(map[string]interface{}); msgmap != nil {
						if emlwtr := ampToWriter(msgmap); emlwtr != nil {
							emailwtrs = append(emailwtrs, emlwtr)
						}
					} else if emlwrtr, _ := d.(*emailwriter.EmailWriter); emlwrtr != nil {
						emlwrtr.ConstructMail()
						emailwtrs = append(emailwtrs, emlwrtr)
					}
				}
				ai++
			}
		}
		if len(emailwtrs) > 0 {
			var smtpclnt = &SMTPClient{username: username, password: password, host: host, mode: mode}
			defer func() {
				smtpclnt = nil
			}()
			func() {
				smtpclnt.SendMail(emailwtrs...)
			}()
		}
	}
}

func (smptclnt *SMTPClient) SendMail(emailwtrs ...*emailwriter.EmailWriter) (err error) {
	c, err := smtpe.Dial(smptclnt.host)
	if err != nil {
		if c, err = smtpe.DialStartTLS(smptclnt.host, nil); err != nil {
			return err
		}
	}
	defer c.Close()
	//if err = c.Hello("localhost"); err != nil {
	//	return err
	//}
	//var startedtls = false
	if ok, _ := c.Extension("STARTTLS"); ok {
		c.Close()
		if c, err = smtpe.DialStartTLS(smptclnt.host, nil); err != nil {
			return err
		} else {
			//startedtls = true
			defer c.Close()
		}
	}

	var a sasl.Client = nil

	if strings.EqualFold(smptclnt.mode, "plain") {
		a = sasl.NewPlainClient("", smptclnt.username, smptclnt.password)
	} else if strings.EqualFold(smptclnt.mode, "login") {
		a = sasl.NewLoginClient(smptclnt.username, smptclnt.password)
	} else if strings.EqualFold(smptclnt.mode, "oauth2") {
		var smtphost = smptclnt.host
		var smtpport = int64(0)
		if strings.Contains(smtphost, ":") {
			smtpport, _ = strconv.ParseInt(smtphost[strings.Index(smtphost, ":")+1:], 10, 64)
			smtphost = smtphost[:strings.Index(smtphost, ":")]
		}
		a = sasl.NewOAuthBearerClient((&sasl.OAuthBearerOptions{
			Username: smptclnt.username,
			Token:    smptclnt.password,
			Host:     smtphost,
			Port:     int(smtpport),
		}))
	}
	if a != nil {
		if err = c.Auth(a); err != nil {
			return err
		}
	}
	if a != nil {
		for _, emailwtr := range emailwtrs {
			err = func() error {
				defer emailwtr.Close()
				var from, _ = emailwtr.AllRecipients("From", "Sender")
				var to, _ = emailwtr.AllRecipients("To", "Reply-To", "Bcc", "Cc")
				if len(from) == 1 && len(to) > 0 {
					//var smpteoptns = &smtpe.MailOptions{}
					if err = c.Mail(from[0], nil); err != nil {
						return err
					}
					for _, addr := range to {
						if err = c.Rcpt(addr, nil); err != nil {
							return err
						}
					}
					w, err := c.Data()
					if err != nil {
						return err
					}
					var written int64 = 0
					written, err = io.Copy(w, emailwtr.Reader())
					if written > 0 && err == io.EOF {
						err = nil
					}
					if err != nil {
						return err
					}
					err = w.Close()
					if err != nil {
						return err
					}
				}
				return nil
			}()
		}
	}
	err = c.Quit()
	return
}
