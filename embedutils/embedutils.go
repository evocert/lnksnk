package embedutils

import (
	"embed"
	"path/filepath"
	"strings"

	"github.com/evocert/lnksnk/fsutils"
)

func ImportResource(fs *fsutils.FSUtils, emdfs embed.FS, incldsubdirs bool, pathroot string, paths ...string) {
	internalImportResource(fs, nil, emdfs, incldsubdirs, pathroot, paths...)
}

func internalImportResource(fs *fsutils.FSUtils, mpdextns map[string][]string, emdfs embed.FS, incldsubdirs bool, pathroot string, paths ...string) {
	isfirst := mpdextns == nil
	if isfirst {
		mpdextns = map[string][]string{}
	}
	rawmkdir := strings.Contains(pathroot, "/raw:")
	if fs != nil {
		fs.MKDIR(pathroot)
		if rawmkdir {
			pathroot = strings.Replace(pathroot, "/raw:", "/", 1)
		}
	}

	for _, path := range paths {
		emddirs, _ := emdfs.ReadDir(path)
		subroot := pathroot
		if incldsubdirs {
			subroot += "/" + path
			if rawmkdir {
				if strings.HasPrefix(subroot, "/") {
					subroot = "/raw:" + subroot[1:]
				} else {
					subroot = "/raw:" + subroot
				}
			}
			fs.MKDIR(subroot)
			if rawmkdir {
				subroot = strings.Replace(subroot, "/raw:", "/", 1)
			}
		}
		for _, emddir := range emddirs {
			if emddir.IsDir() && incldsubdirs {
				esfroot := path
				for esfroot != "" && esfroot[0] == '.' {
					esfroot = esfroot[1:]
				}
				if esfroot != "" && esfroot[len(esfroot)-1] != '/' {
					esfroot += "/"
				}
				internalImportResource(fs, mpdextns, emdfs, incldsubdirs, subroot+"/"+emddir.Name(), esfroot+emddir.Name())
			} else {
				if ext := filepath.Ext(emddir.Name()); ext != "" {
					emdfname := emddir.Name()
					mpdextns[ext] = append(mpdextns[ext], subroot+"/"+emdfname)
					if fs != nil {
						esfroot := path
						for esfroot != "" && esfroot[0] == '.' {
							esfroot = esfroot[1:]
						}
						if esfroot != "" && esfroot[len(esfroot)-1] != '/' {
							esfroot += "/"
						}
						if f, _ := emdfs.Open(esfroot + emdfname); f != nil {
							fs.SET(subroot+"/"+emdfname, f)
						}
					}
				}
			}
		}
	}

	if isfirst {
		htmlhead := ""
		for ext, extpaths := range mpdextns {
			for extptshn, eextpath := range extpaths {
				ismin := strings.Contains(eextpath, ".min.")
				if !ismin {
					if extptshn < len(extpaths)-1 {
						if strings.Contains(extpaths[extptshn+1], ".min.") && strings.EqualFold(eextpath, strings.Replace(extpaths[extptshn+1], ".min.", ".", 1)) {
							continue
						}
					}
				}
				if strings.EqualFold(".css", ext) {
					htmlhead += `<link rel="stylesheet" type="text/css" href="` + eextpath + `">`
				} else if strings.EqualFold(".js", ext) {
					htmlhead += `<script type="text/javascript" src="` + eextpath + `"></script>`
				}
				if extptshn < len(extpaths)-1 {
					htmlhead += "\r\n"
				}
			}
		}
		if htmlhead != "" {
			if fs != nil {
				fs.SET(pathroot+"/head.html", htmlhead)
			}
		}
	}
}
