package mimes

import (
	"bufio"
	"context"
	_ "embed"
	"io"
	"path/filepath"
	"strings"
	"sync"

	"github.com/evocert/lnksnk/iorw"
)

//go:embed mimetypes.txt
var mimetypescsv string

// MimeTypesCSV - return Mime Types CSV reader
var mimebuf = iorw.NewBuffer()
var mimebuflck = &sync.RWMutex{}

func MimeTypesCSV() io.Reader {
	if mimebuf.Size() == 0 {
		func() {
			mimebuflck.Lock()
			defer mimebuflck.Unlock()
			if mimebuf.Size() == 0 {
				mimebuf.Print(mimetypescsv)
			}
		}()
	}
	return mimebuf.Reader().DisposeEOFReader() //strings.NewReader(mimetypescsv)
}

func ExtMimeType(ext string, defaultext string, defaulttype ...string) (mimetype string) {
	var defaulttpe = ""
	if len(defaulttype) > 0 {
		defaulttpe = defaulttype[0]
	}
	if ext = filepath.Ext(ext); ext == "" {
		ext = filepath.Ext(defaultext)
	}
	mimetype, _, _ = FindMimeType(ext, defaulttpe)
	return
}

// FindMimeType - ext or defaulttype
func FindMimeType(ext string, defaulttype string) (mimetype string, texttype bool, mediatype bool) {
	if defaulttype == "" {
		defaulttype = "text/plain"
	}
	texttype = false
	if ext = filepath.Ext(ext); ext != "" {
		func() {
			mtypesfoundlck.RLock()
			if _, mimetypeok := mtypesfound[ext]; mimetypeok {
				defer mtypesfoundlck.RUnlock()
				mimetype = mtypesfound[ext]
				if _, textextok := mtextexts[ext]; textextok {
					texttype = mtextexts[ext]
				}
			} else {
				mtypesfoundlck.RUnlock()
				mtypesfoundlck.Lock()
				defer mtypesfoundlck.Unlock()
				//var bufr = bufio.NewReader(MimeTypesCSV())
				ctx, ctxcancel := context.WithCancel(context.Background())
				go func() {
					defer ctxcancel()
					var bufr = bufio.NewReader(MimeTypesCSV())
					for {
						lineb, lineberr := iorw.ReadLine(bufr)
						if len(lineb) > 0 {
							var lines = strings.Split(string(lineb), "\t")
							if len(lines) == 4 && lines[2] == ext {
								mimetype = lines[1]
								mtypesfound[ext] = mimetype
								if _, textextok := mtextexts[ext]; textextok {
									texttype = mtextexts[ext]
								}
								break
							}
						}
						if lineberr != nil {
							break
						}
					}
					bufr = nil
				}()
				<-ctx.Done()
				if mimetype == "" {
					if mimetype = defaulttype; mimetype == "" {
						mimetype = "text/plain"
					}
				}
			}
		}()
	} else {
		mimetype = defaulttype
	}
	mediatype = strings.Contains(mimetype, "video/") || strings.Contains(mimetype, "audio/")
	return
}

var mtypesfound map[string]string
var mtypesfoundlck = &sync.RWMutex{}

var mtextexts map[string]bool

func init() {
	mtypesfound = map[string]string{}
	mtextexts = map[string]bool{}
	mtextexts[".js"] = true
	mtextexts[".json"] = true
	mtextexts[".html"] = true
	mtextexts[".xhtml"] = true
	mtextexts[".htm"] = true
	mtextexts[".js"] = true
}
