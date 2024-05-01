package minify

import (
	"io"

	"github.com/evocert/lnksnk/iorw"
	"github.com/evocert/lnksnk/iorw/parsing"
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

func init() {
	minifyjs = minify.New()
	minifyjs.AddFunc("application/javascript", js.Minify)

	minifyhtml = minify.New()
	minifyhtml.AddFunc("text/css", css.Minify)
	minifyhtml.AddFunc("text/html", html.Minify)
	minifyhtml.AddFunc("image/svg+xml", svg.Minify)
	parsing.DefaultMinifyPsv = func(psvext string, psvbuf *iorw.Buffer, psvrdr io.Reader) {
		if psvext == ".html" {
			if psvrdr == nil {
				psvrdr = psvbuf.Clone(true).Reader(true)
			}
			MinifyHTML(psvbuf, psvrdr)
		}
	}

	parsing.DefaultMinifyCde = func(cdeext string, cdebuf *iorw.Buffer, cderdr io.Reader) {
		if cdeext == ".js" {
			if cderdr == nil {
				cderdr = cdebuf.Clone(true).Reader(true)
			}
			MinifyJS(cdebuf, cderdr)
		}
	}
}
