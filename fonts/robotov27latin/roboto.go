package robotov27latin

import (
	"bytes"
	_ "embed"

	"github.com/evocert/lnksnk/resources"
)

//go:embed index.css
var indexcss string

//go:embed roboto-v27-latin-regular.woff
var roboto_v27_latin_regular_woff []byte

//go:embed roboto-v27-latin-regular.woff2
var roboto_v27_latin_regular_woff2 []byte

func init() {
	gblrs := resources.GLOBALRSNG()
	gblrs.FS().MKDIR("/raw:fonts/roboto/css", "")
	gblrs.FS().MKDIR("/raw:fonts/roboto/fonts", "")
	gblrs.FS().SET("/fonts/roboto/css/index.css", indexcss)
	gblrs.FS().SET("/fonts/roboto/fonts/roboto-v27-latin-regular.woff", bytes.NewReader(roboto_v27_latin_regular_woff))
	gblrs.FS().SET("/fonts/roboto/fonts/roboto-v27-latin-regular.woff2", bytes.NewReader(roboto_v27_latin_regular_woff2))
	gblrs.FS().MKDIR("/raw:fonts/roboto", "")
	gblrs.FS().SET("/fonts/roboto/head.html", `<link rel="stylesheet" type="text/css" href="/fonts/roboto/css/index.css">`)
}
