package imapcmd

import (
	"github.com/evocert/lnksnk/emailservice/emailserve"
	"github.com/evocert/lnksnk/fsutils"
	"github.com/evocert/lnksnk/imapservice"
	"github.com/evocert/lnksnk/serve/serveio"
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
