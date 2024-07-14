package imapservice

import (
	"context"
	"crypto/tls"
	"strings"
	"sync"

	"github.com/evocert/lnksnk/database"
	"github.com/evocert/lnksnk/fsutils"
	"github.com/evocert/lnksnk/imap"
	"github.com/evocert/lnksnk/imap/imapserver"
	"github.com/evocert/lnksnk/iorw"
	"github.com/evocert/lnksnk/iorw/active"
	"github.com/evocert/lnksnk/listen"
	"github.com/evocert/lnksnk/parameters"
	"github.com/evocert/lnksnk/resources"
)

type ImapService struct {
	dbms      *database.DBMS
	fs        *fsutils.FSUtils
	lsnrs     *sync.Map
	imapsrvrs *sync.Map
}

func NewImapService(a ...interface{}) (imapsvc *ImapService) {
	var dbms *database.DBMS = nil
	var fs *fsutils.FSUtils = nil
	ai, al := 0, len(a)
	for ai < al {
		if dbmsd, _ := a[ai].(*database.DBMS); dbmsd != nil {
			if dbms == nil {
				dbms = dbmsd
			}
			a = append(a[:ai], a[ai+1:]...)
			continue
		}
		if fsd, _ := a[ai].(*fsutils.FSUtils); fsd != nil {
			if fs == nil {
				fs = fsd
			}
			a = append(a[:ai], a[ai+1:]...)
			continue
		}
		ai++
	}
	if fs == nil {
		fs = resources.GLOBALRSNG().FS()
	}
	if dbms == nil {
		dbms = database.GLOBALDBMS()
	}
	imapsvc = &ImapService{dbms: dbms, fs: fs, imapsrvrs: &sync.Map{}, lsnrs: &sync.Map{}}

	return
}

func (imapsvc *ImapService) IMAPSvcHandler(ctx context.Context, runtime active.Runtime, prms parameters.ParametersAPI) (imapsvchndl *IMAPSvrHandler) {
	if imapsvc != nil {
		imapsvchndl = &IMAPSvrHandler{
			imapsvc: imapsvc, ctx: ctx, prms: prms, runtime: runtime,
		}
	}
	return
}

func (imapsvc *ImapService) Register(alias string, netaddr string, tlsConfig ...*tls.Config) (done bool, err error) {
	if alias = strings.TrimFunc(alias, iorw.IsSpace); alias == "" {
		return
	}
	if imapsrvrs := imapsvc.imapsrvrs; imapsrvrs != nil {
		var srvr *imapserver.Server = nil
		var tlscnf *tls.Config = nil

		impsvrv, impsvrvok := imapsrvrs.Load(alias)
		if impsvrvok {
			srvr, _ = impsvrv.(*imapserver.Server)
		} else {
			ln, lnerr := listen.Listen("tcp", netaddr)
			if lnerr != nil {
				err = lnerr
				println(err.Error())
				return
			}
			if len(tlsConfig) > 0 {
				tlscnf = tlsConfig[0].Clone()
			} else {
				/*host, _ := listen.AddrHosts("tcp", netaddr)
				if tlscnf, err = listen.GenerateTlsConfig(host, ""); err != nil {
					return
				}*/
			}
			srvr = imapserver.New(&imapserver.Options{
				NewSession: func(conn *imapserver.Conn) (ssn imapserver.Session, grtng *imapserver.GreetingData, err error) {
					return imapsvc.NewSession(alias)
				},
				Caps: imap.CapSet{
					imap.CapIMAP4rev1: {},
					imap.CapIMAP4rev2: {},

					/*imap.CapNamespace:    {},
					imap.CapUnselect:     {},
					imap.CapUIDPlus:      {},
					imap.CapESearch:      {},
					imap.CapSearchRes:    {},
					imap.CapEnable:       {},
					imap.CapIdle:         {},
					imap.CapSASLIR:       {},
					imap.CapListExtended: {},
					imap.CapListStatus:   {},
					imap.CapMove:         {},
					imap.CapLiteralMinus: {},
					imap.CapStatusSize:   {},*/
				},
				TLSConfig:    tlscnf,
				InsecureAuth: tlscnf == nil,
				DebugWriter:  nil,
			})
			go func() {
				if err = srvr.Serve(ln); err != nil {
					imapsrvrs.Delete(alias)
					srvr.Close()
					return
				}
			}()
			imapsrvrs.Store(alias, srvr)
		}
	}
	return
}

func (imapsvc *ImapService) NewSession(alias string) (ssn imapserver.Session, grtng *imapserver.GreetingData, err error) {
	if alias != "" {
		if dbms, imapsrvrs := imapsvc.dbms, imapsvc.imapsrvrs; imapsrvrs != nil && dbms != nil {
			if dbms.Exists(alias + "imap") {
				if impsrvrv, _ := imapsrvrs.Load(alias); impsrvrv != nil {
					if svr, _ := impsrvrv.(*imapserver.Server); svr != nil {
						imapssn := &imapsession{alias: alias, imapsvc: imapsvc, svr: svr, dbalias: alias + "imap"}
						imapssn.dbhndl = dbms.DBMSHandler(nil, imapssn, nil, nil, imapsvc.fs, nil)
						ssn = imapssn
					}
				}
			}
		}
	}
	return
}
