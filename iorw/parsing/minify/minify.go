package minify

import (
	"io"

	"github.com/lnksnk/lnksnk/iorw"
	"github.com/lnksnk/lnksnk/iorw/parsing"
	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/css"
	"github.com/tdewolff/minify/v2/html"
	"github.com/tdewolff/minify/v2/js"
	"github.com/tdewolff/minify/v2/svg"
)

func MinifyHTML(out io.Writer, rdr io.Reader) (err error) {
	return minifyhtml.Minify("text/html", out, rdr)
}

func MinifyJS(out io.Writer, rdr io.Reader) (err error) {
	return minifyjs.Minify("application/javascript", out, rdr)
}

var minifyhtml *minify.M = nil
var minifyjs *minify.M = nil

type codeerr struct {
	err string
	*iorw.Buffer
}

func (cderr *codeerr) Error() string {
	return cderr.err
}

func (cderr *codeerr) Code() string {
	return cderr.String()
}

func init() {
	minifyjs = minify.New()
	minifyjs.AddFunc("application/javascript", js.Minify)

	minifyhtml = minify.New()
	minifyhtml.AddFunc("text/css", css.Minify)
	minifyhtml.AddFunc("text/html", html.Minify)
	minifyhtml.AddFunc("image/svg+xml", svg.Minify)
	parsing.DefaultMinifyPsv = func(psvext string, psvbuf *iorw.Buffer, psvrdr io.Reader) error {
		if psvext == ".html" {
			if psvrdr == nil {
				psvrdr = psvbuf.Clone(true).Reader(true)
			}
			return MinifyHTML(psvbuf, psvrdr)
		}
		return nil
	}

	parsing.DefaultMinifyCde = func(cdeext string, cdebuf *iorw.Buffer, cderdr io.Reader) (err error) {
		if cdeext == ".js" {
			cderr := &codeerr{Buffer: iorw.NewBuffer()}

			if cderdr == nil {
				cderr.ReadFrom(cdebuf.Clone(true).Reader(true))
				cderdr = cderr.Reader()
			} else {
				cderr.ReadFrom(cderdr)
				cderdr = cderr.Reader(true)
			}
			if err = MinifyJS(cdebuf, cderdr); err != nil {
				cderr.err = err.Error()
				err = cderr
			}
		}
		return
	}
}
