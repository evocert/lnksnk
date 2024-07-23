package parsing

import (
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/evocert/lnksnk/fsutils"
	"github.com/evocert/lnksnk/iorw"
)

func prepPathAndRoot(path, defaultext string) (bool, string, string, string) {
	var cancache = !strings.Contains(path, ":no-cache/")
	if !cancache {
		if nocachsep := strings.Index(path, ":no-cache/"); nocachsep > 0 {
			path = path[:nocachsep] + path[nocachsep+len(":no-cache/"):]
		}
	}
	path = strings.Replace(strings.Replace(path, "\\", "/", -1), "//", "/", -1)
	pathroot := ""
	if defaultext == "" {
		defaultext = ".html"
	}
	if path != "" {
		if path[0:1] == "/" {
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
	}
	return cancache, path, pathroot, defaultext
}

func CanParse(canParse bool, pathModified time.Time, path string, pathroot string, defaultext string, out io.Writer, fs *fsutils.FSUtils, invertActive bool, evalcode func(a ...interface{}) (interface{}, error)) (canprse bool, canprserr error) {
	if cancache, fullpath := func() (chd bool, flpth string) {
		chd, path, pathroot, _ = prepPathAndRoot(path, defaultext)
		flpth = pathroot + path
		return
	}(); cancache {
		if chdscrpt := GLOBALCACHEDSCRIPTING().Script(func() (scrptpath string) {
			if invertActive {
				return "/active:" + fullpath
			}
			return fullpath
		}()); chdscrpt != nil {
			if chdscrpt.IsValidSince(pathModified, fs) {
				if out != nil {
					if _, canprserr = chdscrpt.WritePsvTo(out); canprserr != nil {
						chdscrpt.Dispose()
						chdscrpt = nil
						return
					} else if evalcode != nil {
						if _, canprserr = chdscrpt.EvalAtv(evalcode); canprserr != nil {
							chdscrpt.Dispose()
						}
						return
					}
				} else if evalcode != nil {
					if _, canprserr = chdscrpt.EvalAtv(evalcode); canprserr != nil {
						chdscrpt.Dispose()
					}
					return
				}
			} else {
				chdscrpt.Dispose()
				chdscrpt = nil
			}
		}
	}
	canprse = canprserr == nil
	return
}

func Parse(parseOnly bool, pathModified time.Time, path string, defaultext string, out io.Writer, in interface{}, fs *fsutils.FSUtils, invertActive bool, evalcode func(...interface{}) (interface{}, error), a ...interface{}) (prserr error) {
	pathroot := ""
	cancache, fullpath := func() (chd bool, flpth string) {
		chd, path, pathroot, _ = prepPathAndRoot(path, defaultext)
		flpth = pathroot + path
		return
	}()
	var cachecdefunc func(fullpath string, pathModified time.Time, cachedpaths map[string]time.Time, prsdpsv, prsdatv *iorw.Buffer, preppedatv interface{}) (cshderr error) = nil
	if !parseOnly && cancache {
		if chdscrpt := GLOBALCACHEDSCRIPTING().Script(func() (scrptpath string) {
			if invertActive {
				return "/active:" + fullpath
			}
			return fullpath
		}()); chdscrpt != nil {
			if chdscrpt.IsValidSince(pathModified, fs) {
				if out != nil {
					if _, prserr = chdscrpt.WritePsvTo(out); prserr != nil {
						chdscrpt.Dispose()
						chdscrpt = nil
						return
					}
				}
				if evalcode != nil {
					if _, prserr = chdscrpt.EvalAtv(evalcode); prserr != nil {
						chdscrpt.Dispose()
					}
					return
				}
			} else {
				chdscrpt.Dispose()
				chdscrpt = nil
			}
		}
		cachecdefunc = func(fullpath string, pathModified time.Time, cachedpaths map[string]time.Time, prsdpsv, prsdatv *iorw.Buffer, preppedatv interface{}) (cshderr error) {
			if fullpath != "" {
				if crntscrpt := GLOBALCACHEDSCRIPTING().Load(pathModified, prsdpsv, prsdatv, cachedpaths, func() (scrptpath string) {
					if invertActive {
						if fullpath[0:1] == "/" {
							return "/active:" + fullpath[1:]
						}
						return "/active:" + fullpath
					}
					return fullpath
				}()); crntscrpt != nil && preppedatv != nil {
					crntscrpt.SetScriptProgram(preppedatv)
				}
			}
			return
		}
	}
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
			} else if rdr, _ := in.(io.Reader); rdr != nil {
				rnrdrs = append(rnrdrs, iorw.NewEOFCloseSeekReader(rdr))
			}
		}
	} else {
		if funcrdr, _ := in.(func() (io.Reader, error)); funcrdr != nil {
			rdr, rdrerr := funcrdr()
			if rdrerr == nil {
				if rdr != nil {
					in = rdr
				}
			}
		} else if funcrdr, _ := in.(func() io.Reader); funcrdr != nil {
			if rdr := funcrdr(); rdr != nil {
				in = rdr
			}
		}
		if rnrdr, _ := in.(io.RuneReader); rnrdr != nil {
			rnrdrs = append(rnrdrs, rnrdr)
		} else if rdr, _ := in.(io.Reader); rdr != nil {
			rnrdrs = append(rnrdrs, iorw.NewEOFCloseSeekReader(rdr))
		}
	}

	prserr = internalProcessParsing(cachecdefunc, pathModified, path, pathroot, defaultext, out, fs, invertActive, evalcode, rnrdrs...)
	return
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
			err = Parse(false, pathmodified, path, ".js", passiveContentBuf, fcat, DefaultParseFS, true, func(a ...interface{}) (result interface{}, prscerr error) {
				for _, d := range a {
					if atvrdr, _ := d.(*iorw.BuffReader); atvrdr != nil {
						if passiveContentBuf.Size() > 0 {
							activeCodeBuf.Print("print(`", passiveContentBuf, "`);")
							passiveContentBuf.Clear()
						}
						atvrdr.WriteTo(activeCodeBuf)
						return
					}
				}
				return
			})
			source = append(source, []byte(activeCodeBuf.String())...)
		}
	}

	return
}

var DefaultMinifyPsv func(psvext string, psvbuf *iorw.Buffer, psvrdr io.Reader) error = nil

var DefaultMinifyCde func(cdeext string, cdebuf *iorw.Buffer, cderdr io.Reader) error = nil
