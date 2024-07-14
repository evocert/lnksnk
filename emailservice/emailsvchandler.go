package emailservice

import (
	"context"

	"github.com/evocert/lnksnk/fsutils"
	"github.com/evocert/lnksnk/imapservice"
	"github.com/evocert/lnksnk/iorw/active"
	"github.com/evocert/lnksnk/parameters"
)

type EMAILSvcHandler struct {
	emailsvc    *EmailService
	imapsvchndl *imapservice.IMAPSvrHandler
	ctx         context.Context
	fs          *fsutils.FSUtils
	runtime     active.Runtime
	prms        parameters.ParametersAPI
}

func (emailsvshndl *EMAILSvcHandler) IMAP() *imapservice.IMAPSvrHandler {
	if emailsvshndl != nil {
		return emailsvshndl.imapsvchndl
	}
	return nil
}

func (emailsvchndl *EMAILSvcHandler) InvokeFunction(funcref interface{}, args ...interface{}) (result interface{}) {
	if emailsvchndl != nil {
		if runtime := emailsvchndl.runtime; runtime != nil {
			result = runtime.InvokeFunction(funcref, args...)
		}
	}
	return result
}

func (emailsvchndl *EMAILSvcHandler) Dispose() {
	if emailsvchndl != nil {
		if imapsvchndl := emailsvchndl.imapsvchndl; imapsvchndl != nil {
			emailsvchndl.imapsvchndl = nil
			imapsvchndl.Dispose()
		}
		if emailsvchndl.ctx != nil {
			emailsvchndl.ctx = nil
		}
		if emailsvchndl.emailsvc != nil {
			emailsvchndl.emailsvc = nil
		}
		if emailsvchndl.fs != nil {
			emailsvchndl.fs = nil
		}
		if emailsvchndl.prms != nil {
			emailsvchndl.prms = nil
		}
		if emailsvchndl.runtime != nil {
			emailsvchndl.runtime = nil
		}
	}
}
