package material

import (
	"context"
	"embed"
	"io"
	"strings"

	"github.com/evocert/lnksnk/resources"
)

//go:embed index.html
var indexhtml string

//go:embed fonts/*
var assetFonts embed.FS

//go:embed css/*
var assetCss embed.FS

//go:embed material.css
var metarialcss string

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
	gblrs.FS().MKDIR("/raw:fonts/material", "")
	gblrs.FS().SET("/fonts/material/index.html", indexhtml)
	gblrs.FS().MKDIR("/raw:fonts/material/css", "")

	var dirsfonts, _ = assetFonts.ReadDir("fonts")
	for _, dirfont := range dirsfonts {
		gblrs.FS().SET("/fonts/material/fonts/"+dirfont.Name(), readFile(assetFonts, "fonts/"+dirfont.Name()))
	}

	var dirscss, _ = assetCss.ReadDir("css")
	var cssfiles = []string{}
	for _, dircss := range dirscss {
		if strings.HasSuffix(dircss.Name(), ".map") {
			continue
		}
		cssfiles = append(cssfiles, "/fonts/material/css/"+dircss.Name())
		gblrs.FS().SET(cssfiles[len(cssfiles)-1], readFile(assetCss, "css/"+dircss.Name()))
	}

	gblrs.FS().SET("/fonts/material/material.css", metarialcss)
	gblrs.FS().SET("/fonts/material/head.html", `<link rel="stylesheet" type="text/css" href="`+cssfiles[1]+`">`)
}
