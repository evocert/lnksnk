package pop3lib

import (
	"context"
	"io"

	"github.com/evocert/lnksnk/email/emailreader"
	"github.com/evocert/lnksnk/email/pop3"

	"github.com/emersion/go-message"
)

type POP3Client struct {
	Username string
	Password string
	Host     string
	client   *pop3.Client
	OnClose  func(*POP3Client)
}

func New() (pop3clnt *POP3Client) {
	pop3clnt = &POP3Client{}
	return
}

func (pop3clnt *POP3Client) Connect() (err error) {
	if pop3clnt != nil {
		if pop3clnt.client != nil {
			if err = pop3clnt.client.Noop(); err != nil {
				pop3clnt.client.Quit()
				pop3clnt.client = nil
			}
		}
		if pop3clnt.client == nil {
			if pop3clnt.client, err = pop3.TryDial(pop3clnt.Host); err == nil {
				err = pop3clnt.client.Authorization(pop3clnt.Username, pop3clnt.Password)
			}
		}
	}
	return
}

func (pop3clnt *POP3Client) Client() (client *pop3.Client, err error) {
	if pop3clnt != nil {
		if err = pop3clnt.Connect(); err == nil && pop3clnt.client != nil {
			client = pop3clnt.client
		}
	}
	return
}

func (pop3clnt *POP3Client) ReadMessages(onReadMessage func(string, string, *emailreader.EmailReader) error, anddel ...bool) (err error) {
	if pop3clnt != nil && onReadMessage != nil {
		if clnt, _ := pop3clnt.Client(); clnt != nil {
			var msgenty *message.Entity = nil
			var readPop3MsgEntity func(*message.Entity) error = func(enty *message.Entity) (rderr error) {
				var pi, pw = io.Pipe()
				var cntx, cntxcnl = context.WithCancel(context.Background())
				go func() {
					var pwerr error = nil
					defer func() {
						if pwerr == nil {
							pw.Close()
						} else {
							pw.CloseWithError(pwerr)
						}
					}()
					cntxcnl()
					pwerr = enty.WriteTo(pw)
				}()
				<-cntx.Done()
				var emailReader, emailrdrerr = emailreader.ReadMail(pi)
				if emailrdrerr != nil {
					rderr = emailrdrerr
				} else if emailReader != nil {
					func() {
						defer emailReader.Close()
						rderr = onReadMessage("pop3", "inbox", emailReader)
					}()
				}
				return
			}

			if list, listerr := clnt.ListAll(); len(list) > 0 && listerr == nil {
				for _, msi := range list {
					if msgenty, err = clnt.Retr(msi.ID, anddel...); msgenty != nil && err == nil {
						if err = readPop3MsgEntity(msgenty); err != nil {
							break
						}
					}
				}
			}
		}
	}
	return
}

func (pop3clnt *POP3Client) Close() (err error) {
	if pop3clnt != nil {
		if pop3clnt.client != nil {
			pop3clnt.client.Quit()
			pop3clnt.client = nil
		}
		if pop3clnt.OnClose != nil {
			pop3clnt.OnClose(pop3clnt)
			pop3clnt.OnClose = nil
		}
	}
	return
}
