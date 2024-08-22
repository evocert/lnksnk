package drivers

import (
	"encoding/json"

	"github.com/lnksnk/lnksnk/database"
	"github.com/lnksnk/lnksnk/database/dbserve"
	"github.com/lnksnk/lnksnk/fsutils"
	"github.com/lnksnk/lnksnk/serve/serveio"
)

var cmddrivers dbserve.CommandFunc = func(path, ext string, dbhnl *database.DBMSHandler, w serveio.Writer, r serveio.Reader, fs *fsutils.FSUtils) (err error) {
	encd := json.NewEncoder(w)
	drvrs := dbhnl.Drivers()
	if len(drvrs) == 0 {
		err = w.Print("[]")
		return
	}
	err = encd.Encode(drvrs)
	return
}

var cmddriver dbserve.CommandFunc = func(path, ext string, dbhnl *database.DBMSHandler, w serveio.Writer, r serveio.Reader, fs *fsutils.FSUtils) (err error) {
	return
}

func init() {
	dbserve.HandleCommand("drivers", cmddrivers)
	dbserve.HandleCommand("driver", cmddriver)
}
