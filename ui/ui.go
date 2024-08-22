package ui

import (
	"embed"

	"github.com/lnksnk/lnksnk/embedutils"
	"github.com/lnksnk/lnksnk/resources"
)

//go:embed js/*.*
var uijsfs embed.FS

func init() {
	gblrsngfs := resources.GLOBALRSNG().FS()
	embedutils.ImportResource(gblrsngfs, uijsfs, true, "/ui/js", "js")
}
