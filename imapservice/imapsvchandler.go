package imapservice

import (
	"context"

	"github.com/lnksnk/lnksnk/fsutils"
	"github.com/lnksnk/lnksnk/iorw/active"
	"github.com/lnksnk/lnksnk/parameters"
)

type IMAPSvrHandler struct {
	imapsvc *ImapService
	ctx     context.Context
	fs      *fsutils.FSUtils
	runtime active.Runtime
	prms    parameters.ParametersAPI
}

func (imapsvchndl *IMAPSvrHandler) InvokeFunction(funcref interface{}, args ...interface{}) (result interface{}) {
	if imapsvchndl != nil {
		if runtime := imapsvchndl.runtime; runtime != nil {
			result = runtime.InvokeFunction(funcref, args...)
		}
	}
	return result
}

func (imapsvchndl *IMAPSvrHandler) Register(alias string, netaddr string) (done bool, err error) {
	if imapsvchndl != nil {
		if imapsvc := imapsvchndl.imapsvc; imapsvc != nil {
			done, err = imapsvc.Register(alias, netaddr)
		}
	}
	return
}

func (imapsvchndl *IMAPSvrHandler) Dispose() {
	if imapsvchndl != nil {
		if imapsvchndl.ctx != nil {
			imapsvchndl.ctx = nil
		}
		if imapsvchndl.fs != nil {
			imapsvchndl.fs = nil
		}
		if imapsvchndl.imapsvc != nil {
			imapsvchndl.imapsvc = nil
		}
		if imapsvchndl.prms != nil {
			imapsvchndl.prms = nil
		}
		if imapsvchndl.runtime != nil {
			imapsvchndl.runtime = nil
		}
	}
}
