package emailservice

import (
	"context"

	"github.com/lnksnk/lnksnk/imapservice"
	"github.com/lnksnk/lnksnk/iorw/active"
	"github.com/lnksnk/lnksnk/parameters"
)

type EmailService struct {
	imapsvc *imapservice.ImapService
}

func NewEmailService(a ...interface{}) (emailsvc *EmailService) {
	emailsvc = &EmailService{imapsvc: imapservice.NewImapService(a...)}
	return
}

func (emailsvc *EmailService) IMAP() (imapsvc *imapservice.ImapService) {
	if emailsvc != nil {
		imapsvc = emailsvc.imapsvc
	}
	return
}

func (emailsvc *EmailService) EMAILSvcHandler(ctx context.Context, runtime active.Runtime, prms parameters.ParametersAPI) (emailsvchndl *EMAILSvcHandler) {
	if emailsvc != nil {
		emailsvchndl = &EMAILSvcHandler{emailsvc: emailsvc, imapsvchndl: emailsvc.IMAP().IMAPSvcHandler(ctx, runtime, prms), ctx: ctx, runtime: runtime, prms: prms}
	}
	return
}

var gblemailsvc *EmailService = nil

func GLOABLEMAILSVC() *EmailService {
	return gblemailsvc
}
func init() {
	gblemailsvc = NewEmailService()
}
