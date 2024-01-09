package imaplib

import (
	"context"
	"io"
	"strings"
	"sync"

	"github.com/evocert/lnksnk/email/emailreader"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/emersion/go-sasl"
	"golang.org/x/oauth2"
)

type IMapClient struct {
	Username    string
	Password    string
	Host        string
	Mode        string
	ModeOptions map[string]interface{}
	client      *imapclient.Client
	oathbr      *OAuthBearer
}

func New() (imapclnt *IMapClient) {
	imapclnt = &IMapClient{ModeOptions: map[string]interface{}{}}
	return
}

func (imapclnt *IMapClient) Connect() (err error) {
	if imapclnt != nil {
		if imapclnt.client != nil {
			if err = imapclnt.client.Noop().Wait(); err != nil {
				imapclnt.client.Close()
				imapclnt.client = nil
			}
		}
		if imapclnt.client == nil {
			if imapclnt.client, err = imapclient.DialTLS(imapclnt.Host, nil); err != nil {
				//imapclnt.client, err = imapclient.Dial(imapclnt.Host)
			}
			if imapclnt.client != nil && err == nil {
				if strings.EqualFold(imapclnt.Mode, "plain") {
					err = imapclnt.client.Authenticate(sasl.NewPlainClient("", imapclnt.Username, imapclnt.Password))
				} else if strings.EqualFold(imapclnt.Mode, "login") {
					err = imapclnt.client.Login(imapclnt.Username, imapclnt.Password).Wait() // Authenticate(sasl.NewLoginClient(imapclnt.Username, imapclnt.Password))
				} else if strings.EqualFold(imapclnt.Mode, "oauth2") {
					if imapclnt.oathbr == nil {
						imapclnt.oathbr = &OAuthBearer{}
					}
					err = imapclnt.oathbr.Authenticate(imapclnt.Username, imapclnt.Password, imapclnt.client)
				}
			}
		}
	}
	return
}

func (imapclnt *IMapClient) MailBoxes(filter string) (mailboxes []*imap.ListData, err error) {
	if imapclnt != nil {
		if err = imapclnt.Connect(); err == nil {
			if imapclnt.client != nil {
				mailboxeschn := make(chan *imap.ListData, 10)
				done := make(chan error, 1)
				go func() {
					listCmd := imapclnt.client.List("", filter, &imap.ListOptions{
						ReturnStatus: &imap.StatusOptions{
							NumMessages: true,
							NumUnseen:   true,
						},
					})

					for {
						mbox := listCmd.Next()
						if mbox == nil {
							break
						}
						mailboxeschn <- mbox
					}
					done <- listCmd.Close()
				}()

				//log.Println("Mailboxes:")
				for m := range mailboxeschn {
					mailboxes = append(mailboxes, m)
					//log.Println("* " + m.Name)
				}

				if dnerr := <-done; dnerr != nil {
					err = dnerr
				}
			}
		}
	}
	return
}

func (imapclnt *IMapClient) ReadMessages(box string, recent bool, onSetCriterial func(string, string, *imap.SearchCriteria) error, onReadMessage func(string, string, *emailreader.EmailReader) (err error), max ...uint32) (err error) {
	if imapclnt != nil && onReadMessage != nil {
		if err = imapclnt.Connect(); err == nil {
			if imapclnt.client != nil {
				if mbox, mboxerr := imapclnt.client.Select(box, nil).Wait(); mboxerr != nil {
					err = mboxerr
				} else if mbox != nil && recent && mbox.NumMessages > 0 {

					criteria := &imap.SearchCriteria{}
					criteria.Flag = []imap.Flag{"\\Seen"}
					if onSetCriterial != nil {
						if err = onSetCriterial("imap", box, criteria); err != nil {
							return
						}
					}
					seqset := new(imap.SeqSet)
					//seqset.AddRange(from, to)
					var uids []uint32
					if schrdata, srcherr := imapclnt.client.UIDSearch(criteria, nil).Wait(); srcherr == nil && schrdata.Count > 0 {

						from := uint32(1)
						to := schrdata.Count

						if len(max) > 0 && max[0] < to {
							from = to - max[0]
						}
						if from <= to {
							uids = uids[from-1 : to]
						}
						if len(uids) > 0 {
							seqset.AddNum(uids...)
						}
					}
					if seqnums, seqok := seqset.Nums(); seqok && len(seqnums) > 0 {
						fetchOptions := &imap.FetchOptions{
							UID:         true,
							BodySection: []*imap.FetchItemBodySection{{}},
						}
						var countemailreads = int64(len(uids))
						//var messages = make(chan *imapclient.FetchMessageBuffer, countemailreads)
						wg := &sync.WaitGroup{}
						wg.Add(1)
						go func() {
							defer wg.Done()
							if ftchcmd := imapclnt.client.Fetch(*seqset, fetchOptions); ftchcmd != nil {
								for {
									if msg := ftchcmd.Next(); msg == nil {
										break
									} else {
										if ftchmsgbuf, fcthmsgbuferr := msg.Collect(); fcthmsgbuferr == nil {
											//ftchmsgbuf.Envelope
											mlpi, mlpw := io.Pipe()
											ctx, ctxcnl := context.WithCancel(context.Background())
											go func() {
												pwerr := error(nil)
												defer func() {
													if pwerr == nil {
														mlpw.Close()
													} else {
														mlpw.CloseWithError(pwerr)
													}
													ctxcnl()
													for _, buf := range ftchmsgbuf.BodySection {
														mlpw.Write(buf)
													}
												}()
											}()
											<-ctx.Done()
											var emailrdr, emailrdrerr = emailreader.ReadMail(mlpi)
											countemailreads--
											if emailrdrerr != nil {
												err = emailrdrerr
											} else if emailrdr != nil {
												func() {
													defer emailrdr.Close()
													err = onReadMessage("imap", box, emailrdr) //readMsg(box, emailrdr, msg)
												}()
												if err == nil {
													//ucidtomark = append(ucidtomark, msg.Uid)
												}
											}
										}
									}
								}
								err = ftchcmd.Close()
							}
						}()
						wg.Wait()
					}
				}
			}
		}
	}
	return
}

type OAuthBearer struct {
	OAuth2  *oauth2.Config
	Enabled bool
}

func (c *OAuthBearer) ExchangeRefreshToken(refreshToken string) (*oauth2.Token, error) {
	token := new(oauth2.Token)
	token.RefreshToken = refreshToken
	token.TokenType = "Bearer"
	return c.OAuth2.TokenSource(context.TODO(), token).Token()
}

func (c *OAuthBearer) Authenticate(username string, password string, client *imapclient.Client) error {
	if c.OAuth2.Endpoint.TokenURL != "" {
		token, err := c.ExchangeRefreshToken(password)
		if err != nil {
			return err
		}
		password = token.AccessToken
	}

	saslClient := sasl.NewOAuthBearerClient(&sasl.OAuthBearerOptions{
		Username: username,
		Token:    password,
	})

	return client.Authenticate(saslClient)
}
