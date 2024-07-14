package robotov27latin

import (
	"context"
	"embed"
	_ "embed"
	"io"

	"github.com/evocert/lnksnk/resources"
)

//go:embed index.css
var indexcss string

//go:embed *.woff*
var ebfnts embed.FS

//go:embed roboto-v27-latin-regular.woff
var roboto_v27_latin_regular_woff []byte

//go:embed roboto-v27-latin-regular.woff2
var roboto_v27_latin_regular_woff2 []byte

func init() {
	var readFile = func(fs embed.FS, filepath string) (rdc io.ReadCloser) {
		pi, pw := io.Pipe()
		ctx, ctxcancel := context.WithCancel(context.Background())
		go func() {
			defer pw.Close()
			ctxcancel()
			if f, ferr := fs.Open(filepath); ferr == nil && f != nil {
				func() {
					defer f.Close()
					io.Copy(pw, f)
				}()
			}
		}()
		<-ctx.Done()
		rdc = pi
		return
	}
	gblrs := resources.GLOBALRSNG()
	gblrs.FS().MKDIR("/raw:fonts/roboto/css", "")
	gblrs.FS().MKDIR("/raw:fonts/roboto/fonts", "")
	gblrs.FS().SET("/fonts/roboto/css/index.css", indexcss)
	gblrs.FS().SET("/fonts/roboto/fonts/roboto-v27-latin-regular.woff", readFile(ebfnts, "roboto-v27-latin-regular.woff"))   // io.MultiReader(bytes.NewReader(roboto_v27_latin_regular_woff)))
	gblrs.FS().SET("/fonts/roboto/fonts/roboto-v27-latin-regular.woff2", readFile(ebfnts, "roboto-v27-latin-regular.woff2")) // io.MultiReader(bytes.NewReader(roboto_v27_latin_regular_woff2)))
	gblrs.FS().MKDIR("/raw:fonts/roboto", "")
	gblrs.FS().SET("/fonts/roboto/head.html", `<link rel="stylesheet" type="text/css" href="/fonts/roboto/css/index.css">`)
}
