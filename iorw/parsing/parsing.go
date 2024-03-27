package parsing

import (
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/evocert/lnksnk/fsutils"
	"github.com/evocert/lnksnk/iorw"
)

func prepPathAndRoot(path, pathroot, defaultext string) (string, string, string) {
	path = strings.Replace(path, "\\", "/", -1)
	pathroot = strings.Replace(pathroot, "\\", "/", -1)
	if defaultext == "" {
		defaultext = ".html"
	}
	if path != "" {
		if strings.HasPrefix(path, "/") {
			pathroot = path[:strings.LastIndex(path, "/")+1]
			path = path[strings.LastIndex(path, "/")+1:]
		} else {
			if !strings.HasSuffix(pathroot, "/") {
				pathroot += "/"
			}
		}
		if pathext := filepath.Ext(path); pathext != "" {
			defaultext = pathext
		}
	} else {
		if pathroot != "" {
			if !strings.HasSuffix(pathroot, "/") {
				pathroot += "/"
			}
		}
	}
	return path, pathroot, defaultext
}

func CanParse(canParse bool, pathModified time.Time, path string, pathroot string, defaultext string, out io.Writer, fs *fsutils.FSUtils, invertActive bool, evalcode func(cdebuf *iorw.BuffReader, cdepgrm func() interface{}, setcdeprgm func(interface{})) error) (canprse bool, canprserr error) {
	var cancache = !strings.Contains(path, ":no-cache/")
	if !cancache {
		if nocachsep := strings.Index(path, ":no-cache/"); nocachsep > 0 {
			path = path[:nocachsep] + path[nocachsep+len(":no-cache/"):]
		}
	}
	path, pathroot, _ = prepPathAndRoot(path, pathroot, defaultext)
	fullpath := pathroot + path

	if cancache {
		if chdscrpt := GLOBALCACHEDSCRIPTING().Script(func() (scrptpath string) {
			if invertActive {
				return "/active:" + fullpath
			}
			return fullpath
		}()); chdscrpt != nil {
			if chdscrpt.IsValidSince(pathModified, fs) {
				if out != nil {
					_, canprserr = chdscrpt.WritePsvTo(out)
				}
				if canprserr == nil && evalcode != nil {
					if canprserr = chdscrpt.EvalAtv(evalcode); canprserr != nil {
						chdscrpt.Dispose()
					}
				}
				return
			}
			chdscrpt.Dispose()
			chdscrpt = nil
			return
		}
	}
	canprse = canprserr == nil
	return
}

func Parse(parseOnly bool, canParse bool, pathModified time.Time, path string, pathroot string, defaultext string, out io.Writer, in io.Reader, fs *fsutils.FSUtils, invertActive bool, evalcode func(cdebuf *iorw.BuffReader, cdepgrm func() interface{}, setcdeprgm func(interface{})) error, a ...interface{}) (prserr error) {
	var cancache = !strings.Contains(path, ":no-cache/")
	if !cancache {
		if nocachsep := strings.Index(path, ":no-cache/"); nocachsep > 0 {
			path = path[:nocachsep] + path[nocachsep+len(":no-cache/"):]
		}
	}
	path, pathroot, defaultext = prepPathAndRoot(path, pathroot, defaultext)
	var rnrdrs []io.RuneReader = nil
	if in == nil {
		if path == "" {
			path = "index" + defaultext
		}
		if in = fs.CAT(pathroot + path); in == nil {
			if len(a) > 0 {
				var buf *iorw.Buffer = nil
				var initn = -1
				var lastn = -1
				for dn, d := range a {
					if rnrdr, _ := d.(io.RuneReader); rnrdr != nil {
						if initn > -1 {
							buf = iorw.NewBuffer()
							buf.Print(a[initn : lastn+1]...)
							if buf.Size() > 0 {
								rnrdrs = append(rnrdrs, buf.Reader(true))
							}
							initn = -1
							lastn = -1
						}
						rnrdrs = append(rnrdrs, rnrdr)
					} else {
						if initn == -1 {
							initn = dn
						}
						if lastn = dn; lastn == len(a)-1 {
							if initn > -1 {
								buf = iorw.NewBuffer()
								buf.Print(a[initn : lastn+1]...)
								if buf.Size() > 0 {
									rnrdrs = append(rnrdrs, buf.Reader(true))
								}
								initn = -1
								lastn = -1
							}
						}
					}
				}
			}
		} else {
			if rnrdr, _ := in.(io.RuneReader); rnrdr != nil {
				rnrdrs = append(rnrdrs, rnrdr)
			} else {
				rnrdrs = append(rnrdrs, iorw.NewEOFCloseSeekReader(in))
			}
		}
	} else {
		if rnrdr, _ := in.(io.RuneReader); rnrdr != nil {
			rnrdrs = append(rnrdrs, rnrdr)
		} else {
			rnrdrs = append(rnrdrs, iorw.NewEOFCloseSeekReader(in))
		}
	}
	prserr = processParsing(parseOnly, canParse, cancache, pathModified, path, pathroot, defaultext, out, fs, invertActive, evalcode, rnrdrs...)
	return
}

func processParsing(
	parseOnly bool,
	canparse bool,
	cancache bool,
	pathModified time.Time,
	path, pathroot, pathext string,
	out io.Writer,
	fs *fsutils.FSUtils,
	invertActive bool,
	evalcode func(*iorw.BuffReader, func() interface{}, func(interface{})) error,
	rnrdrs ...io.RuneReader) error {

	return internalProcessParsing(parseOnly, canparse, cancache,
		pathModified,
		path,
		pathroot,
		pathext,
		out,
		fs,
		invertActive,
		evalcode,
		rnrdrs...)
}

var DefaultParseFS *fsutils.FSUtils = nil

func ParseSourceLoader(path string) (source []byte, err error) {
	passiveContentBuf := iorw.NewBuffer()
	activeCodeBuf := iorw.NewBuffer()
	pathmodified := time.Now()
	if DefaultParseFS != nil {
		if fcat := DefaultParseFS.CAT(path, func(mod time.Time) {
			pathmodified = mod
		}); fcat != nil {
			err = Parse(false, true, pathmodified, path, "", "", passiveContentBuf, fcat, DefaultParseFS, true, func(atvrdr *iorw.BuffReader, atvpgrm func() interface{}, setatvpgrm func(interface{})) (prscerr error) {
				if passiveContentBuf.Size() > 0 {
					activeCodeBuf.Print("print(`", passiveContentBuf, "`);")
					passiveContentBuf.Clear()
				}
				atvrdr.WriteTo(activeCodeBuf)
				return
			})
			source = append(source, []byte(activeCodeBuf.String())...)
		}
	}

	return
}

var DefaultMinifyPsv func(psvext string, psvbuf *iorw.Buffer, psvrdr io.Reader) = nil

var DefaultMinifyCde func(cdeext string, cdebuf *iorw.Buffer, cderdr io.Reader) = nil

func init() {
	/*DefaultParseFS = resources.GLOBALRSNG().FS()
	DefaultParseFS.MKDIR("/parse")
	DefaultParseFS.SET("/parse/index.html", `<html><body><main-heading>THE HEADING</main-heading></body></html>`)
	DefaultParseFS.SET("/parse/main.html", "<@for(var i=0;i<10;i++) {@><main-heading>THE HEADING</main-heading><@}@>")
	//DefaultParseFS.SET("/parse/main.html", "for(var i=0;i<10;i++) {@><main-heading><@@>`${(i+1)}`<@@></main-heading><@}")
	DefaultParseFS.SET("/parse/main-heading.html", "<h1>HEADING <:cntnt:/></h1>")
	ParseSourceLoader("/parse/index.html")*/
}
