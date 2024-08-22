package imapservice

import (
	"fmt"
	"net/mail"
	"strings"

	"github.com/lnksnk/lnksnk/database"
	"github.com/lnksnk/lnksnk/imap"
	"github.com/lnksnk/lnksnk/imap/imapserver"
	"github.com/lnksnk/lnksnk/iorw"
	"github.com/lnksnk/lnksnk/iorw/active"
)

var _ imapserver.Session = (*imapsession)(nil)

type imapsession struct {
	svr     *imapserver.Server
	imapsvc *ImapService
	alias   string
	dbalias string
	dbhndl  *database.DBMSHandler
	runtime active.Runtime
}

func (imapssn *imapsession) InvokeFunction(callfunc interface{}, a ...interface{}) (result interface{}) {
	if imapssn != nil {
		if runtime := imapssn.runtime; runtime != nil {
			result = runtime.InvokeFunction(callfunc, a...)
		}
	}
	return
}

func (imapssn *imapsession) Close() (err error) {
	if imapssn != nil {
		if imapsvc := imapssn.imapsvc; imapsvc != nil {
			imapssn.imapsvc = nil
		}
	}
	return
}

func (imapssn *imapsession) Login(username, password string) (err error) {
	if imapssn != nil {
		if username, password = strings.TrimFunc(username, iorw.IsSpace), strings.TrimFunc(password, iorw.IsSpace); username == "" || password == "" {
			err = fmt.Errorf("%s", "Invalid username or password")
			return
		}
		if dbhndl, dbalias, alias := imapssn.dbhndl, strings.TrimFunc(imapssn.dbalias, iorw.IsSpace), strings.TrimFunc(imapssn.alias, iorw.IsSpace); alias != "" && dbalias != "" && dbhndl != nil && dbhndl.Exists(imapssn.dbalias) {
			addr, addrerr := mail.ParseAddress(username)
			if addrerr != nil {
				err = addrerr
				return
			}
			if addr != nil {
				if user := strings.TrimFunc(addr.Address, iorw.IsSpace); user != "" {
					ati := strings.Index(user, "@")
					if ati == -1 {
						ati = len(user)
					}
					domain := strings.TrimFunc(user[ati+1:], iorw.IsSpace)
					if user = strings.TrimFunc(user[:ati], iorw.IsSpace); user != "" {
						if reclogin := dbhndl.Query(dbalias, "/"+alias+"/imap/api/sql/login", map[string]interface{}{"user": user, "password": password, "domain": domain, "address": addr.Address, "name": addr.Name, "imapalias": alias, "error": func(dberr error) {
							err = dberr
						}}); reclogin != nil {
							reclogin.ForEachDataMap(func(data map[string]interface{}, nr int64, first, last bool) bool {
								if first {

								}
								return false
							})
						}
					}
				}
			}
		}
	}
	return
}

func (imapssn *imapsession) Select(mailbox string, options *imap.SelectOptions) (imapslcted *imap.SelectData, err error) {
	if imapssn != nil {

	}
	return
}

func (imapssn *imapsession) Create(mailbox string, options *imap.CreateOptions) (err error) {
	if imapssn != nil {

	}
	return
}

func (imapssn *imapsession) Delete(mailbox string) (err error) {
	if imapssn != nil {

	}
	return
}

func (imapssn *imapsession) Rename(mailbox, newName string) (err error) {
	if imapssn != nil {

	}
	return
}

func (imapssn *imapsession) Subscribe(mailbox string) (err error) {
	if imapssn != nil {

	}
	return
}

func (imapssn *imapsession) Unsubscribe(mailbox string) (err error) {
	if imapssn != nil {

	}
	return
}

func (imapssn *imapsession) List(w *imapserver.ListWriter, ref string, patterns []string, options *imap.ListOptions) (err error) {
	if imapssn != nil {

	}
	return
}

func (imapssn *imapsession) Status(mailbox string, options *imap.StatusOptions) (statusdata *imap.StatusData, err error) {
	if imapssn != nil {

	}
	return
}

func (imapssn *imapsession) Append(mailbox string, r imap.LiteralReader, options *imap.AppendOptions) (appenddata *imap.AppendData, err error) {
	if imapssn != nil {

	}
	return
}

func (imapssn *imapsession) Poll(w *imapserver.UpdateWriter, allowExpunge bool) (err error) {
	if imapssn != nil {

	}
	return
}

func (imapssn *imapsession) Idle(w *imapserver.UpdateWriter, stop <-chan struct{}) (err error) {
	if imapssn != nil {

	}
	return
}

func (imapssn *imapsession) Unselect() (err error) {
	if imapssn != nil {

	}
	return
}

func (imapssn *imapsession) Expunge(w *imapserver.ExpungeWriter, uids *imap.UIDSet) (err error) {
	if imapssn != nil {

	}
	return
}

func (imapssn *imapsession) Search(kind imapserver.NumKind, criteria *imap.SearchCriteria, options *imap.SearchOptions) (searchdata *imap.SearchData, err error) {
	if imapssn != nil {

	}
	return
}

func (imapssn *imapsession) Fetch(w *imapserver.FetchWriter, numSet imap.NumSet, options *imap.FetchOptions) (err error) {
	if imapssn != nil {

	}
	return
}

func (imapssn *imapsession) Store(w *imapserver.FetchWriter, numSet imap.NumSet, flags *imap.StoreFlags, options *imap.StoreOptions) (err error) {
	if imapssn != nil {

	}
	return
}

func (imapssn *imapsession) Copy(numSet imap.NumSet, dest string) (copydata *imap.CopyData, err error) {
	if imapssn != nil {

	}
	return
}

func (imapssn *imapsession) Namespace() (namespcedata *imap.NamespaceData, err error) {
	if imapssn != nil {

	}
	return
}

func (imapssn *imapsession) Move(w *imapserver.MoveWriter, numSet imap.NumSet, dest string) (err error) {
	if imapssn != nil {

	}
	return
}

/*
Close() error

	// Not authenticated state
	Login(username, password string) error

	// Authenticated state
	Select(mailbox string, options *imap.SelectOptions) (*imap.SelectData, error)
	Create(mailbox string, options *imap.CreateOptions) error
	Delete(mailbox string) error
	Rename(mailbox, newName string) error
	Subscribe(mailbox string) error
	Unsubscribe(mailbox string) error
	List(w *ListWriter, ref string, patterns []string, options *imap.ListOptions) error
	Status(mailbox string, options *imap.StatusOptions) (*imap.StatusData, error)
	Append(mailbox string, r imap.LiteralReader, options *imap.AppendOptions) (*imap.AppendData, error)
	Poll(w *UpdateWriter, allowExpunge bool) error
	Idle(w *UpdateWriter, stop <-chan struct{}) error

	// Selected state
	Unselect() error
	Expunge(w *ExpungeWriter, uids *imap.UIDSet) error
	Search(kind NumKind, criteria *imap.SearchCriteria, options *imap.SearchOptions) (*imap.SearchData, error)
	Fetch(w *FetchWriter, numSet imap.NumSet, options *imap.FetchOptions) error
	Store(w *FetchWriter, numSet imap.NumSet, flags *imap.StoreFlags, options *imap.StoreOptions) error
	Copy(numSet imap.NumSet, dest string) (*imap.CopyData, error)
*/

/*
	Namespace() (*imap.NamespaceData, error)
*/

/*
	Move(w *MoveWriter, numSet imap.NumSet, dest string) error
*/
