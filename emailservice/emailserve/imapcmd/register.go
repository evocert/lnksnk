package imapcmd

import (
	"github.com/lnksnk/lnksnk/emailservice/emailserve"
	"github.com/lnksnk/lnksnk/fsutils"
	"github.com/lnksnk/lnksnk/imapservice"
	"github.com/lnksnk/lnksnk/serve/serveio"
)

var cmdregister emailserve.ImapCommandFunc = func(alias, path, ext string, imapsvchndl *imapservice.IMAPSvrHandler, w serveio.Writer, r serveio.Reader, fs *fsutils.FSUtils) (err error) {

	return
}

var cmdunregister emailserve.ImapCommandFunc = func(alias, path, ext string, imapsvchndl *imapservice.IMAPSvrHandler, w serveio.Writer, r serveio.Reader, fs *fsutils.FSUtils) (err error) {

	return
}

func init() {
	emailserve.HandleCommand("register", cmdregister, "unregister", cmdunregister)
}
