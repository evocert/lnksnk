package fonts

import (
	_ "github.com/evocert/lnksnk/fonts/material"
	_ "github.com/evocert/lnksnk/fonts/robotov27latin"
	"github.com/evocert/lnksnk/resources"
)

func init() {
	gblrsfs := resources.GLOBALRSNG().FS()
	gblrsfs.MKDIR("/fonts")
	gblrsfs.SET("/fonts/head.html", gblrsfs.CAT("/fonts/material/head.html"), "\r\n", gblrsfs.CAT("/fonts/roboto/head.html"))
}
