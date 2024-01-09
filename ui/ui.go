package ui

import (
	"embed"

	"github.com/evocert/lnksnk/embedutils"
	"github.com/evocert/lnksnk/resources"
)

//go:embed js/*.*
var uijsfs embed.FS

func init() {
	gblrsngfs := resources.GLOBALRSNG().FS()
	embedutils.ImportResource(gblrsngfs, uijsfs, true, "/ui/js", "js")
}
