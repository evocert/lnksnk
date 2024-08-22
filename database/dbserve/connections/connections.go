package connections

import (
	"encoding/json"

	"github.com/lnksnk/lnksnk/database"
	"github.com/lnksnk/lnksnk/database/dbserve"
	"github.com/lnksnk/lnksnk/fsutils"
	"github.com/lnksnk/lnksnk/serve/serveio"
)

var cmdconnections dbserve.CommandFunc = func(path, ext string, dbhnl *database.DBMSHandler, w serveio.Writer, r serveio.Reader, fs *fsutils.FSUtils) (err error) {
	encd := json.NewEncoder(w)
	cnctns := dbhnl.Connections()
	if len(cnctns) == 0 {
		err = w.Print("[]")
		return
	}
	err = encd.Encode(cnctns)
	return
}

var cmdconnection dbserve.CommandFunc = func(path, ext string, dbhnl *database.DBMSHandler, w serveio.Writer, r serveio.Reader, fs *fsutils.FSUtils) (err error) {
	return
}

func init() {
	dbserve.HandleCommand("connections", cmdconnections)
	dbserve.HandleCommand("connection", cmdconnection)
}
