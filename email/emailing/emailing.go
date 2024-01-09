package emailing

import (
	"strconv"
	"strings"
	"sync"

	"github.com/evocert/lnksnk/email/emailreader"
	"github.com/evocert/lnksnk/email/emailwriter"
	"github.com/evocert/lnksnk/email/imaplib"
	"github.com/evocert/lnksnk/email/pop3"
	"github.com/evocert/lnksnk/email/pop3lib"
	"github.com/evocert/lnksnk/email/smtplib"
	"github.com/evocert/lnksnk/fsutils"
	"github.com/evocert/lnksnk/iorw/active"
	"github.com/evocert/lnksnk/parameters"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-message"
)

type EMail struct {
	smptusername    string
	smptpassword    string
	smtphost        string
	smtpmode        string
	smtpmodeoptions map[string]interface{}
	pop3clnt        *pop3lib.POP3Client
	imapclnt        *imaplib.IMapClient
}

func (email *EMail) SendMail(a ...interface{}) (err error) {
	if email != nil && email.smptusername != "" && email.smptpassword != "" && email.smtphost != "" && email.smtpmode != "" {
		smtplib.SendMail(email.smptusername, email.smptpassword, email.smtphost, email.smtpmode, email.smtpmodeoptions, a...)
	}
	return
}

func (email *EMail) Pop3() (pop3clnt *pop3lib.POP3Client) {
	return
}

type EMailManager struct {
	emails        map[string]*EMail
	OnReadMessage func(protocol string, box string, emailrdr *emailreader.EmailReader) (rderr error)
	lckemails     *sync.RWMutex
}

func NewEmailManager() (emailmngr *EMailManager) {
	emailmngr = &EMailManager{lckemails: &sync.RWMutex{}, emails: map[string]*EMail{}}
	return
}

func (emailmngr *EMailManager) ReadMessages(alias string, box string, a ...interface{}) (err error) {
	if alias = strings.TrimSpace(alias); alias != "" && emailmngr != nil && len(emailmngr.emails) > 0 {
		var settings map[string]interface{} = nil
		var onreadMessage func(string, string, string, *emailreader.EmailReader) error = nil
		var onsetCriterial func(string, string, string, *imap.SearchCriteria) error = nil
		var max uint32 = 0

		for _, d := range a {
			if stngs, _ := d.(map[string]interface{}); len(stngs) > 0 {
				if settings == nil {
					for stngk, stngv := range stngs {
						if strings.EqualFold(stngk, "max") {
							if smax, _ := stngv.(string); smax != "" {
								if imax, _ := strconv.ParseInt(smax, 10, 64); imax > 0 {
									if max == 0 {
										max = uint32(imax)
									}
								}
							} else if imax, _ := stngv.(int64); imax > 0 {
								if max == 0 {
									max = uint32(imax)
								}
							}
						}
					}
				}
			} else if dreadmsg, _ := d.(func(string, string, string, *emailreader.EmailReader) error); dreadmsg != nil {
				if onreadMessage == nil {
					onreadMessage = dreadmsg
				}
			} else if dsetcriteria, _ := d.(func(string, string, string, *imap.SearchCriteria) error); dsetcriteria != nil {
				if onsetCriterial == nil {
					onsetCriterial = dsetcriteria
				}
			}
		}
		var imapclnt *imaplib.IMapClient = nil
		func() {
			emailmngr.lckemails.RLock()
			defer emailmngr.lckemails.RUnlock()
			if email := emailmngr.emails[alias]; email != nil {
				imapclnt = email.imapclnt
			}
		}()
		var onReadMessage = func(protocol, box string, emailrdr *emailreader.EmailReader) (rderr error) {
			rderr = onreadMessage(alias, protocol, box, emailrdr)
			return
		}
		var onSetCriterial func(string, string, *imap.SearchCriteria) error = nil
		if onsetCriterial != nil {
			onSetCriterial = func(protocol, box string, sc *imap.SearchCriteria) error {
				return onsetCriterial(alias, protocol, box, sc)
			}
		}
		if imapclnt != nil {
			err = imapclnt.ReadMessages(box, true, onSetCriterial, onReadMessage, max)
		} else {
			var pop3clnt *pop3lib.POP3Client = nil
			func() {
				emailmngr.lckemails.RLock()
				defer emailmngr.lckemails.RUnlock()
				if email := emailmngr.emails[alias]; email != nil {
					pop3clnt = email.pop3clnt
				}
			}()
			if pop3clnt != nil {
				err = pop3clnt.ReadMessages(onReadMessage, true)
			}
		}
	}
	return
}

func invokeReadMessage(script active.Runtime, onreadmessage interface{}, alias string, protocol string, box string, emailReader *emailreader.EmailReader) {
	if onreadmessage != nil {
		if fncreadmsg, fncsuccessok := onreadmessage.(func(alias string, protocol string, box string, emailReader *emailreader.EmailReader)); fncsuccessok {
			fncreadmsg(alias, protocol, box, emailReader)
		} else if script != nil {
			script.InvokeFunction(onreadmessage, alias, protocol, box, emailReader)
		}
	}
}

func invokeSetReadCriteria(script active.Runtime, onsetreadcriteria interface{}, alias string, protocol string, box string, sc *imap.SearchCriteria) {
	if onsetreadcriteria != nil {
		if fncreadmsg, fncsuccessok := onsetreadcriteria.(func(alias string, protocol string, box string, sc *imap.SearchCriteria)); fncsuccessok {
			fncreadmsg(alias, protocol, box, sc)
		} else if script != nil {
			script.InvokeFunction(onsetreadcriteria, alias, protocol, box, sc)
		}
	}
}

func (emailmngr *EMailManager) Register(alias string, a ...interface{}) {
	if emailmngr != nil {
		if alias = strings.TrimSpace(alias); alias != "" {
			var smtpsettings = map[string]string{}
			var pop3settings = map[string]string{}
			var imapsettings = map[string]string{}
			var email, _ = emailmngr.emails[alias]
			for _, d := range a {
				if d != nil {
					if dmp, _ := d.(map[string]interface{}); len(dmp) > 0 {
						for dmpk, dmpv := range dmp {
							if dmpk = strings.TrimSpace(dmpk); dmpk != "" {
								if dmpvmp, _ := dmpv.(map[string]interface{}); len(dmpvmp) > 0 {
									for dmpvk, dmpvv := range dmpvmp {
										if strings.Contains(",username,password,host,mode,mode-settings,mode-options", ","+strings.TrimSpace(strings.ToLower(dmpvk))+",") {
											if dmpvs, _ := dmpvv.(string); strings.TrimSpace(dmpvs) != "" {
												if strings.EqualFold(dmpk, "smtp") {
													smtpsettings[dmpvk] = dmpvs
												} else if strings.EqualFold(dmpk, "pop3") {
													pop3settings[dmpvk] = dmpvs
												} else if strings.EqualFold(dmpk, "imap") {
													imapsettings[dmpvk] = dmpvs
												}
											} else if dmpvmap, _ := dmpv.(map[string]interface{}); len(dmpvmap) > 0 && (strings.EqualFold(dmpvk, "mode-settings") || strings.EqualFold(dmpvk, "mode-options")) {
												if strings.EqualFold(dmpk, "smtp") {
													smtpsettings[dmpvk] = dmpvs
												} else if strings.EqualFold(dmpk, "imap") {
													imapsettings[dmpvk] = dmpvs
												}
											}
										}
									}
								}
							}
						}
					}
				}
			}

			if len(smtpsettings) > 0 || len(pop3settings) > 0 || len(imapsettings) > 0 {
				if email == nil {
					func() {
						email = &EMail{smptusername: smtpsettings["username"], smptpassword: smtpsettings["password"], smtphost: smtpsettings["host"], smtpmode: smtpsettings["mode"]}
						if len(imapsettings) > 0 {
							email.imapclnt = &imaplib.IMapClient{Username: imapsettings["username"], Password: imapsettings["password"], Mode: imapsettings["mode"], Host: imapsettings["host"]}
						} else if len(pop3settings) > 0 {
							email.pop3clnt = &pop3lib.POP3Client{Username: pop3settings["username"], Password: pop3settings["password"], Host: pop3settings["host"]}
						}
						emailmngr.lckemails.Lock()
						defer emailmngr.lckemails.Unlock()
						emailmngr.emails[alias] = email
					}()
				} else {
					func() {
						emailmngr.lckemails.Lock()
						defer emailmngr.lckemails.RUnlock()
						email.smptusername = smtpsettings["username"]
						email.smptpassword = smtpsettings["password"]
						email.smtphost = smtpsettings["host"]
						email.imapclnt = &imaplib.IMapClient{Username: imapsettings["username"], Password: imapsettings["password"], Host: imapsettings["host"]}
						email.pop3clnt = &pop3lib.POP3Client{Username: pop3settings["username"], Password: pop3settings["password"], Host: pop3settings["host"]}
					}()
				}
			}
		}
	}
}

func (emailmngr *EMailManager) SendMail(alias string, a ...interface{}) (err error) {
	if emailmngr != nil {
		if alias = strings.TrimSpace(alias); alias != "" {
			var email *EMail = nil
			func() {
				emailmngr.lckemails.RLock()
				defer emailmngr.lckemails.RUnlock()
				email = emailmngr.emails[alias]
			}()
			if email != nil {
				err = email.SendMail(a...)
			}
		}
	}
	return
}

func (emailmngr *EMailManager) Info(alias string, settings ...string) (stngsfound map[string]interface{}) {
	if emailmngr != nil {
		if alias = strings.TrimSpace(alias); alias != "" {
			func() {
				emailmngr.lckemails.RLock()
				defer emailmngr.lckemails.RUnlock()
				if email := emailmngr.emails[alias]; email != nil {
					if len(settings) == 0 {
						settings = append(settings, "smtp", "pop3", "imap")
					}
					stngsfound = map[string]interface{}{}
					var foundsmtp, foundpop3, foundimap = false, false, false
					for _, stng := range settings {
						if !foundsmtp && strings.EqualFold(stng, "smtp") && email.smptpassword != "" && email.smptusername != "" && email.smtphost != "" {
							foundsmtp = true
							var smtpsettings = map[string]string{}
							smtpsettings["username"] = email.smptusername
							smtpsettings["password"] = email.smptpassword
							smtpsettings["host"] = email.smtphost
							smtpsettings["mode"] = email.smtpmode
							stngsfound["smtp"] = smtpsettings
						} else if !foundpop3 && strings.EqualFold(stng, "pop3") && email.pop3clnt != nil {
							foundpop3 = true
							var pop3settings = map[string]string{}
							pop3settings["username"] = email.pop3clnt.Username
							pop3settings["password"] = email.pop3clnt.Password
							pop3settings["host"] = email.pop3clnt.Host
							stngsfound["pop3"] = pop3settings
						} else if !foundimap && strings.EqualFold(stng, "imap") && email.imapclnt != nil {
							foundimap = true
							var imapsettings = map[string]string{}
							imapsettings["username"] = email.imapclnt.Username
							imapsettings["password"] = email.imapclnt.Password
							imapsettings["host"] = email.imapclnt.Host
							imapsettings["mode"] = email.imapclnt.Mode
							stngsfound["imap"] = imapsettings
						}
					}
				}
			}()
		}
	}
	return
}

func (emailmngr *EMailManager) Pop3Noop(alias string) (count int, err error) {
	if emailmngr != nil {
		var pop3clnt *pop3lib.POP3Client = nil
		func() {
			emailmngr.lckemails.RLock()
			defer emailmngr.lckemails.RUnlock()
			if email := emailmngr.emails[alias]; email != nil {
				pop3clnt = email.pop3clnt
			}
		}()
		if pop3clnt != nil {
			if clnt, _ := pop3clnt.Client(); clnt != nil {
				err = clnt.Noop()
			}
		}
	}
	return
}

func (emailmngr *EMailManager) Pop3Rset(alias string) (err error) {
	if emailmngr != nil {
		var pop3clnt *pop3lib.POP3Client = nil
		func() {
			emailmngr.lckemails.RLock()
			defer emailmngr.lckemails.RUnlock()
			if email := emailmngr.emails[alias]; email != nil {
				pop3clnt = email.pop3clnt
			}
		}()
		if pop3clnt != nil {
			if clnt, _ := pop3clnt.Client(); clnt != nil {
				err = clnt.Rset()
			}
		}
	}
	return
}

func (emailmngr *EMailManager) Pop3Stat(alias string) (count int, err error) {
	if emailmngr != nil {
		var pop3clnt *pop3lib.POP3Client = nil
		func() {
			emailmngr.lckemails.RLock()
			defer emailmngr.lckemails.RUnlock()
			if email := emailmngr.emails[alias]; email != nil {
				pop3clnt = email.pop3clnt
			}
		}()
		if pop3clnt != nil {
			if clnt, _ := pop3clnt.Client(); clnt != nil {
				count, _, err = clnt.Stat()
			}
		}
	}
	return
}

func (emailmngr *EMailManager) Pop3List(alias string) (msglist []pop3.MessageList, err error) {
	if emailmngr != nil {
		var pop3clnt *pop3lib.POP3Client = nil
		func() {
			emailmngr.lckemails.RLock()
			defer emailmngr.lckemails.RUnlock()
			if email := emailmngr.emails[alias]; email != nil {
				pop3clnt = email.pop3clnt
			}
		}()
		if pop3clnt != nil {
			if clnt, _ := pop3clnt.Client(); clnt != nil {
				msglist, err = clnt.ListAll()
			}
		}
	}
	return
}

func (emailmngr *EMailManager) Pop3Dele(alias string, msg ...int) (msgs []int, err error) {
	if emailmngr != nil {
		var pop3clnt *pop3lib.POP3Client = nil
		func() {
			emailmngr.lckemails.RLock()
			defer emailmngr.lckemails.RUnlock()
			if email := emailmngr.emails[alias]; email != nil {
				pop3clnt = email.pop3clnt
			}
		}()
		if pop3clnt != nil {
			if clnt, _ := pop3clnt.Client(); clnt != nil {
				for _, mi := range msg {
					if err = clnt.Dele(mi); err == nil {
						msgs = append(msgs, mi)
					}
				}
			}
		}
	}
	return
}

func (emailmngr *EMailManager) Pop3ReadMessages(alias string, anddel bool, readMsgEntity func(*message.Entity) error, msgid ...int) (err error) {
	if emailmngr != nil {
		var pop3clnt *pop3lib.POP3Client = nil
		func() {
			emailmngr.lckemails.RLock()
			defer emailmngr.lckemails.RUnlock()
			if email := emailmngr.emails[alias]; email != nil {
				pop3clnt = email.pop3clnt
			}
		}()
		if pop3clnt != nil {
			if clnt, _ := pop3clnt.Client(); clnt != nil {
				var msgenty *message.Entity = nil
				if readMsgEntity != nil {
					for _, msi := range msgid {
						if msgenty, err = clnt.Retr(msi, anddel); msgenty != nil && err == nil {
							if err = readMsgEntity(msgenty); err != nil {
								break
							}
						}
					}
				}
			}
		}
	}
	return
}

func (emailmngr *EMailManager) Pop3Quit(alias string) (err error) {
	if emailmngr != nil {
		var pop3clnt *pop3lib.POP3Client = nil
		func() {
			emailmngr.lckemails.RLock()
			defer emailmngr.lckemails.RUnlock()
			if email := emailmngr.emails[alias]; email != nil {
				pop3clnt = email.pop3clnt
			}
		}()
		if pop3clnt != nil {
			err = pop3clnt.Close()
		}
	}
	return
}

type ActiveEmailManager struct {
	atvrntme            active.Runtime
	prmsfnc             func() parameters.ParametersAPI
	emailmngr           *EMailManager
	emailrdrs           map[*emailreader.EmailReader]*emailreader.EmailReader
	emailwtrs           map[*emailwriter.EmailWriter]*emailwriter.EmailWriter
	crntreadmessage     interface{}
	crntsetreadcriteria interface{}
	fs                  *fsutils.FSUtils
}

func (atvemailmng *ActiveEmailManager) onReadMessage(alias string, protocol string, box string, emailrdr *emailreader.EmailReader) (err error) {
	if atvemailmng != nil && atvemailmng.crntreadmessage != nil {
		invokeReadMessage(atvemailmng, atvemailmng.crntreadmessage, alias, protocol, box, emailrdr)
	}
	return
}

func (atvemailmng *ActiveEmailManager) onSetReadCriteria(alias string, protocol string, box string, sc *imap.SearchCriteria) (err error) {
	if atvemailmng != nil && atvemailmng.crntsetreadcriteria != nil {
		invokeSetReadCriteria(atvemailmng, atvemailmng.crntsetreadcriteria, alias, protocol, box, sc)
	}
	return
}

func (atvemailmngr *ActiveEmailManager) InvokeFunction(a interface{}, args ...interface{}) interface{} {
	if atvemailmngr != nil && atvemailmngr.atvrntme != nil {
		return atvemailmngr.atvrntme.InvokeFunction(a, args...)
	}
	return nil
}

func (atvemailmngr *ActiveEmailManager) ReadMessages(alias string, box string, a ...interface{}) (err error) {
	if atvemailmngr != nil && atvemailmngr.emailmngr != nil {
		if alias != "" && box != "" {
			a = append(a, atvemailmngr.onReadMessage, atvemailmngr.onSetReadCriteria)
			if len(a) > 0 {
				for an, d := range a {
					if d != nil {
						if dstngs, _ := d.(map[string]interface{}); len(dstngs) > 0 {
							for stngk, stngv := range dstngs {
								if strings.EqualFold(stngk, "readmessage") || strings.EqualFold(stngk, "onreadmessage") {
									if _, okrdmsg := d.(func(string, string, string, *emailreader.EmailReader) error); !okrdmsg {
										if stngv != nil && stngv != atvemailmngr.crntreadmessage {
											atvemailmngr.crntreadmessage = stngv
										}
										delete(dstngs, stngk)
										a[an] = dstngs
									}
								} else if strings.EqualFold(stngk, "setreadcriteria") || strings.EqualFold(stngk, "onsetreadcriteria") {
									if _, okrdmsg := d.(func(string, string, string, *imap.SearchCriteria) error); !okrdmsg {
										if stngv != nil && stngv != atvemailmngr.crntsetreadcriteria {
											atvemailmngr.crntsetreadcriteria = stngv
										}
										delete(dstngs, stngk)
										a[an] = dstngs
									}
								}
							}
						}
					}
				}
				return atvemailmngr.emailmngr.ReadMessages(alias, box, a...)
			}
		}
	}
	return
}

func (atvemailmngr *ActiveEmailManager) Pop3Quit(alias string) (err error) {
	if atvemailmngr != nil && atvemailmngr.emailmngr != nil {
		err = atvemailmngr.emailmngr.Pop3Quit(alias)
	}
	return
}

func (atvemailmngr *ActiveEmailManager) Pop3Dele(alias string, msg ...int) (msgs []int, err error) {
	if atvemailmngr != nil && atvemailmngr.emailmngr != nil {
		msgs, err = atvemailmngr.emailmngr.Pop3Dele(alias)
	}
	return
}

func (atvemailmngr *ActiveEmailManager) Pop3List(alias string) (msglist []pop3.MessageList, err error) {
	if atvemailmngr != nil && atvemailmngr.emailmngr != nil {
		msglist, err = atvemailmngr.emailmngr.Pop3List(alias)
	}
	return
}

func (atvemailmngr *ActiveEmailManager) Pop3Stat(alias string) (count int, err error) {
	if atvemailmngr != nil && atvemailmngr.emailmngr != nil {
		count, err = atvemailmngr.emailmngr.Pop3Stat(alias)
	}
	return
}

func (atvemailmngr *ActiveEmailManager) Pop3Rset(alias string) (err error) {
	if atvemailmngr != nil && atvemailmngr.emailmngr != nil {
		err = atvemailmngr.emailmngr.Pop3Rset(alias)
	}
	return
}

func (atvemailmngr *ActiveEmailManager) Pop3Noop(alias string) (count int, err error) {
	if atvemailmngr != nil && atvemailmngr.emailmngr != nil {
		count, err = atvemailmngr.emailmngr.Pop3Noop(alias)
	}
	return
}

func (atvemailmngr *ActiveEmailManager) EmailInfo(alias string, settings ...string) (stngsfound map[string]interface{}) {
	if atvemailmngr != nil && atvemailmngr.emailmngr != nil {
		stngsfound = atvemailmngr.emailmngr.Info(alias, settings...)
	}
	return
}

func (atvemailmngr *ActiveEmailManager) SendMail(alias string, a ...interface{}) (err error) {
	if atvemailmngr != nil && atvemailmngr.emailmngr != nil {
		err = atvemailmngr.emailmngr.SendMail(alias, a...)
	}
	return
}

func (atvemailmngr *ActiveEmailManager) Register(alias string, a ...interface{}) {
	if atvemailmngr != nil && atvemailmngr.emailmngr != nil {
		atvemailmngr.emailmngr.Register(alias, a...)
	}
}

func (atvemailmngr *ActiveEmailManager) Reader(a ...interface{}) (emailrdr *emailreader.EmailReader, err error) {
	if atvemailmngr != nil {
		emailrdr, err = emailreader.ReadMail(a...)
		if err == nil && emailrdr != nil {
			if atvemailmngr.emailrdrs == nil {
				atvemailmngr.emailrdrs = map[*emailreader.EmailReader]*emailreader.EmailReader{}
			}
			emailrdr.OnClose = func(er *emailreader.EmailReader) (clserr error) {
				delete(atvemailmngr.emailrdrs, er)
				return
			}
		}
	}
	return
}

func (atvemailmngr *ActiveEmailManager) Writer() (emailwtr *emailwriter.EmailWriter, err error) {
	if atvemailmngr != nil {
		emailwtr = &emailwriter.EmailWriter{}
		if err == nil && emailwtr != nil {
			if atvemailmngr.emailwtrs == nil {
				atvemailmngr.emailwtrs = map[*emailwriter.EmailWriter]*emailwriter.EmailWriter{}
			}
			emailwtr.OnClose = func(ew *emailwriter.EmailWriter) (clserr error) {
				delete(atvemailmngr.emailwtrs, ew)
				return
			}
		}
	}
	return
}

func (atvemailmngr *ActiveEmailManager) Close() (err error) {
	if atvemailmngr != nil {
		if atvemailmngr.atvrntme != nil {
			atvemailmngr.atvrntme = nil
		}
		if atvemailmngr.prmsfnc != nil {
			atvemailmngr.prmsfnc = nil
		}
		if atvemailmngr.emailrdrs != nil {
			for emailr := range atvemailmngr.emailrdrs {
				emailr.Close()
			}
			atvemailmngr.emailrdrs = nil
		}
		if atvemailmngr.emailwtrs != nil {
			for emailw := range atvemailmngr.emailwtrs {
				emailw.Close()
			}
			atvemailmngr.emailwtrs = nil
		}
		if atvemailmngr.fs != nil {
			atvemailmngr.fs = nil
		}
	}
	return
}

// ActiveEmailManager return registered connections
func (emailmngr *EMailManager) ActiveEmailManager(rntme active.Runtime, prmsfnc func() parameters.ParametersAPI, fs *fsutils.FSUtils) (atvemailmngr *ActiveEmailManager) {
	return newActiveDBMS(emailmngr, rntme, prmsfnc, fs)
}

func newActiveDBMS(emailmngr *EMailManager, rntme active.Runtime, prmsfnc func() parameters.ParametersAPI, fs *fsutils.FSUtils) (atvemailmngr *ActiveEmailManager) {
	if emailmngr != nil && rntme != nil {
		atvemailmngr = &ActiveEmailManager{emailmngr: emailmngr, atvrntme: rntme, prmsfnc: prmsfnc, fs: fs}
	}
	return
}

var glblemailmngr = NewEmailManager()

func GLOBALEMAILMNGR() *EMailManager {
	return glblemailmngr
}
