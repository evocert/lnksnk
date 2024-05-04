package resources

import (
	"bufio"
	"io"
	"os"
	"strings"
	"time"

	"github.com/evocert/lnksnk/fsutils"
	"github.com/evocert/lnksnk/iorw"
)

// ResourcingManager - struct
type ResourcingManager struct {
	fsutils      *fsutils.FSUtils
	rsngpaths    map[string]*ResourcingEndpoint
	rsngendpaths map[*ResourcingEndpoint]string
}

// FS return fsutils.FSUtils implementation for *ResourcingManager
func (rscngmngr *ResourcingManager) FS() *fsutils.FSUtils {
	if rscngmngr.fsutils == nil {
		rscngmngr.fsutils = &fsutils.FSUtils{
			EXISTS: func(path string) bool {
				return rscngmngr.fsexists(path)
			},
			ABS: func(path string) (abspath string) {
				abspath, _ = rscngmngr.fsabs(path)
				return
			},
			FIND: func(path ...interface{}) (finfos []fsutils.FileInfo) {
				finfos, _ = rscngmngr.fsfind(path...)
				return
			}, FINDROOT: func(path ...interface{}) (root string) {
				root, _ = rscngmngr.fsfindroot(path...)
				return
			}, FINDROOTS: func(path ...interface{}) (roots []string) {
				roots, _ = rscngmngr.fsfindroots(path...)
				return
			}, LS: func(path ...interface{}) (finfos []fsutils.FileInfo) {
				finfos = rscngmngr.fsls(path...)
				return
			}, MKDIR: func(path ...interface{}) bool {
				return rscngmngr.fsmkdir(path...)
			}, MKDIRALL: func(path ...interface{}) bool {
				return rscngmngr.fsmkdirall(path...)
			}, RM: func(path string) bool {
				return rscngmngr.fsrm(path)
			}, MV: func(path string, destpath string) bool {
				return rscngmngr.fsmv(path, destpath)
			}, TOUCH: func(path string) bool {
				return rscngmngr.fstouch(path)
			}, PIPE: func(path string, a ...interface{}) io.Reader {
				return rscngmngr.fspipe(path, a...)
			},
			PIPES: func(path string, a ...interface{}) string {
				return rscngmngr.fspipes(path, a...)
			}, CAT: func(path string, a ...interface{}) io.Reader {
				return rscngmngr.fscat(path, a...)
			},
			CATS: func(path string, a ...interface{}) string {
				return rscngmngr.fscats(path, a...)
			}, SET: func(path string, a ...interface{}) bool {
				return rscngmngr.fsset(path, a...)
			}, APPEND: func(path string, a ...interface{}) bool {
				return rscngmngr.fsappend(path, a...)
			}, MULTICAT: func(path ...string) (r io.Reader) {
				return rscngmngr.fsmulticat(path...)
			}, MULTICATS: func(path ...string) string {
				return rscngmngr.fsmulticats(path...)
			},
		}
	}
	return rscngmngr.fsutils
}

func (rscngmngr *ResourcingManager) findrsendpnt(path string) (epnt *ResourcingEndpoint, rpath, rroot string) {
	if path != "" && !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if len(path) > 0 {
		pths := strings.Split(path, "/")
		rpath = ""
		tpth := ""
		tpthl := 0
		for pn := range pths {
			if tpth == "" {
				tpth = "/" + pths[pn]
			} else {
				tpth += pths[pn]
			}
			if epntfnd, epntfndok := rscngmngr.rsngpaths[tpth]; epntfndok && tpthl < len(tpth) {
				rroot = tpth
				rpath = strings.Join(pths[pn+1:], "/")
				tpthl = len(tpth)
				epnt = epntfnd //rscngmngr.rsngrootpaths[epntfnd]
			}
			if tpth != "/" {
				tpth += "/"
			}
		}
	}
	return
}

func (rscngmngr *ResourcingManager) findrsendpnts(path ...string) (epnts []*ResourcingEndpoint, epnttphs, epnttroots []string) {
	if pl := len(path); pl > 0 {
		epnts = make([]*ResourcingEndpoint, pl)
		epnttphs = make([]string, pl)
		epnttroots = make([]string, pl)
		for pn, pth := range path {
			epnts[pn], epnttphs[pn], epnttroots[pn] = rscngmngr.findrsendpnt(pth)
		}
	}
	return
}

func (rscngmngr *ResourcingManager) findrsendpntpaths(path ...string) (epnts []*ResourcingEndpoint, epntpaths, paths []string, roots []string) {
	if pl := len(path); pl > 0 {
		if epntssrchd, epntssrchdphs, epntssrchdproots := rscngmngr.findrsendpnts(path...); len(epntssrchd) > 0 {
			for pn := range epntssrchd {
				if ept := epntssrchd[pn]; ept != nil {
					if epnts == nil {
						epnts = []*ResourcingEndpoint{}
					}
					epnts = append(epnts, ept)
					if epntpaths == nil {
						epntpaths = []string{}
					}
					epntpaths = append(epntpaths, epntssrchdphs[pn])
					if paths == nil {
						paths = []string{}
					}
					paths = append(paths, path[pn])
					if roots == nil {
						roots = []string{}
					}
					roots = append(roots, epntssrchdproots[pn])
				}
			}
		}
	}
	return
}

func (rscngmngr *ResourcingManager) fsappend(path string, a ...interface{}) (fnd bool) {
	if path != "" && !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if epnts, paths, _, _ := rscngmngr.findrsendpntpaths(path); epnts != nil && paths != nil {
		if len(epnts) == 1 && len(paths) == 1 {
			fnd = epnts[0].fsappend(paths[0], a...)
		}
		epnts = nil
		paths = nil
	}
	return fnd
}

func (rscngmngr *ResourcingManager) fsset(path string, a ...interface{}) (set bool) {
	if path != "" && !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if epnts, paths, _, _ := rscngmngr.findrsendpntpaths(path); epnts != nil && paths != nil {
		if len(epnts) == 1 && len(paths) == 1 {
			set = epnts[0].fsset(paths[0], a...)
		}
		epnts = nil
		paths = nil
	}
	return
}

func (rscngmngr *ResourcingManager) fscat(path string, a ...interface{}) (r io.Reader) {
	if path != "" && !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if epnts, paths, _, _ := rscngmngr.findrsendpntpaths(path); epnts != nil && paths != nil {
		if len(epnts) == 1 && len(paths) == 1 {
			r = epnts[0].fscat(paths[0], a...)
		}
		epnts = nil
		paths = nil
	}
	return
}

func (rscngmngr *ResourcingManager) fsmulticats(path ...string) (cntnt string) {
	if len(path) > 0 {
		for _, pth := range path {
			if pth != "" {
				cntnt += rscngmngr.fscats(pth)
			}
		}
	}
	return
}

func (rscngmngr *ResourcingManager) fsmulticat(path ...string) (r io.Reader) {
	var rdrs []io.Reader = nil
	if pthl := len(path); pthl > 0 {
		for _, pth := range path {

			if nxtpth := pth; nxtpth != "" {
				if nxtpth != "" && !strings.HasPrefix(nxtpth, "/") {
					nxtpth = "/" + nxtpth
				}
				if epnts, paths, _, _ := rscngmngr.findrsendpntpaths(nxtpth); epnts != nil && paths != nil {
					if len(epnts) == 1 && len(pth) == 1 {
						rdrs = append(rdrs, epnts[0].fscat(paths[0]))
					}
					epnts = nil
					paths = nil
				}
			}
		}
	}
	r = iorw.NewMultiEOFCloseSeekReader(rdrs...)
	return
}

func (rscngmngr *ResourcingManager) fscats(path string, a ...interface{}) (s string) {
	if path != "" && !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if epnts, paths, _, _ := rscngmngr.findrsendpntpaths(path); epnts != nil && paths != nil {
		if len(epnts) == 1 && len(paths) == 1 {
			s = epnts[0].fscats(paths[0], a...)
		}
		epnts = nil
		paths = nil
	}
	return s
}

func (rscngmngr *ResourcingManager) fspipe(path string, a ...interface{}) (r io.Reader) {
	if path != "" && !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if epnts, paths, _, _ := rscngmngr.findrsendpntpaths(path); epnts != nil && paths != nil {
		if len(epnts) == 1 && len(paths) == 1 {
			r = epnts[0].fspipe(paths[0], a...)
		}
		epnts = nil
		paths = nil
	}
	return
}

func (rscngmngr *ResourcingManager) fspipes(path string, a ...interface{}) (s string) {
	if path != "" && !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if epnts, paths, _, _ := rscngmngr.findrsendpntpaths(path); epnts != nil && paths != nil {
		if len(epnts) == 1 && len(paths) == 1 {
			s = epnts[0].fspipes(paths[0], a...)
		}
		epnts = nil
		paths = nil
	}
	return s
}

func (rscngmngr *ResourcingManager) fstouch(path string) (tchd bool) {
	if path != "" && !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if epnts, paths, _, _ := rscngmngr.findrsendpntpaths(path); epnts != nil && paths != nil {
		if len(epnts) == 1 && len(paths) == 1 {
			tchd = epnts[0].fstouch(paths[0])
		}
		epnts = nil
		paths = nil
	}
	return
}

func (rscngmngr *ResourcingManager) fsmv(path string, destpath string) (mvd bool) {
	if path != "" && !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if destpath != "" && !strings.HasPrefix(destpath, "/") {
		destpath = "/" + destpath
	}
	if epnts, paths, _, _ := rscngmngr.findrsendpntpaths(path); epnts != nil && paths != nil {
		if destepnts, destpaths, _, _ := rscngmngr.findrsendpntpaths(destpath); destepnts != nil && destpaths != nil {
			if len(epnts) == 1 && len(paths) == 1 && len(destepnts) == 1 && len(destpaths) == 1 && epnts[0] == destepnts[0] {
				mvd = epnts[0].fsmv(paths[0], destpaths[0])
			} else if len(epnts) == 1 && len(paths) == 1 && len(destepnts) == 1 && len(destpaths) == 1 && epnts[0] != destepnts[0] {
				if mverr := fsutils.MV(epnts[0].path+paths[0], destepnts[0].path+destpaths[0]); mverr == nil {
					mvd = true
				}
			}
			destepnts = nil
			destpaths = nil
		}
		epnts = nil
		paths = nil
	}
	return
}

func (rscngmngr *ResourcingManager) fsrm(path string) (rmd bool) {
	if path != "" && !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if epnts, epntpaths, paths, _ := rscngmngr.findrsendpntpaths(path); epnts != nil && paths != nil {
		if endpntsl := len(epnts); endpntsl > 0 && endpntsl == len(paths) {
			endpntsi := 0
			pthstoUnregister := []string{}
			for endpntsi < endpntsl {
				rmd = epnts[endpntsi].fsrm(epntpaths[endpntsi])
				if epntpaths[endpntsi] == "" {
					pthstoUnregister = append(pthstoUnregister, paths[endpntsi])
				}
				endpntsi++
			}
			rscngmngr.UnregisterPaths(pthstoUnregister...)
		}
		epnts = nil
		paths = nil
	}
	return
}

func (rscngmngr *ResourcingManager) fsmkdirall(path ...interface{}) (mkdall bool) {
	if pthl := len(path); pthl > 0 {
		if pthl == 1 {
			path = append(path, "")
			pthl++
		}
		var pth1, _ = path[0].(string)
		pth1 = strings.TrimSpace(pth1)
		var pth2 = ""
		if pthl > 1 {
			pth2, _ = path[1].(string)
			pth2 = strings.TrimSpace(pth2)
			path[1] = pth2
		}
		if pth1 != "" && !strings.HasPrefix(pth1, "/") {
			pth1 = "/" + pth1
		}
		path[0] = pth1
		if pthl == 1 {
			if epnts, paths, _, _ := rscngmngr.findrsendpntpaths(pth1); epnts != nil && paths != nil {
				if len(epnts) == 1 && len(paths) == 1 {
					mkdall = epnts[0].fsmkdirall(paths[0])
				}
				epnts = nil
				paths = nil
			} else if pthl == 1 && pth1 != "" {
				rscngmngr.RegisterEndpoint(pth1, "")
				mkdall = true
			}
		} else if pthl >= 2 {
			rscngmngr.RegisterEndpoint(pth1, pth2, path[2:]...)
			mkdall = true
		}
	}
	return
}

func (rscngmngr *ResourcingManager) fsmkdir(path ...interface{}) (mkd bool) {
	var fs FS = nil
	var pthl = len(path)

	if pthl > 0 {
		if pthl == 1 {
			path = append(path, "")
			pthl++
		}
		var pthi = 0
		for pthi < pthl {
			if d := path[pthi]; d != nil {
				if fsd, _ := d.(FS); fsd != nil {
					if fs == nil {
						fs = fsd
						path = append(path[:pthi], path[pthi+1:]...)
						pthl--
						continue
					}
				}
			}
			pthi++
		}
	}
	if pthl > 0 {
		var pth1, _ = path[0].(string)
		pth1 = strings.TrimSpace(pth1)
		var pth2 = ""
		if pthl > 1 {
			if pth2, _ = path[1].(string); pth2 != "" {
				if pth2 = strings.TrimSpace(pth2); pth2 == "./" {
					pth2 = strings.Replace(os.Args[0], "\\", "/", -1)
					if lsti := strings.LastIndex(pth2, "/"); lsti > 0 {
						pth2 = pth2[:lsti]
					} else if lsti == -1 {
						pth2 = "."
					}
				}
			}
			path[1] = pth2
		}
		if pth1 != "" && !strings.HasPrefix(pth1, "/") {
			pth1 = "/" + pth1
		}
		path[0] = pth1
		if pthl == 1 {
			if fs == nil {
				if epnts, paths, _, _ := rscngmngr.findrsendpntpaths(pth1); epnts != nil && paths != nil {
					if len(epnts) == 1 && len(paths) == 1 {
						mkd = epnts[0].fsmkdir(paths[0])
					}
					epnts = nil
					paths = nil
				} else if pthl == 1 && pth1 != "" {
					rscngmngr.RegisterEndpoint(pth1, "")
					mkd = true
				}
			} else if pthl == 1 && pth1 != "" {
				rscngmngr.RegisterEndpoint(pth1, "", fs)
				mkd = true
			}
		} else if pthl >= 2 {
			rscngmngr.RegisterEndpoint(pth1, pth2, path[2:]...)
			mkd = true
			//} else if pthl > 2 {
			//	rscngmngr.RegisterEndpoint(pth1, pth2, path[2:]...)
			//	mkd = true
		}
	}
	return
}

func (rscngmngr *ResourcingManager) fsexists(path string) (pathexists bool) {
	if path != "" {
		path = strings.Replace(path, "\\", "/", -1)
	}
	if lsinfo := rscngmngr.fsls(path); len(lsinfo) == 1 {
		pathexists = true
	}
	return
}

func (rscngmngr *ResourcingManager) fsabs(path string) (abspath string, err error) {
	if path != "" {
		path = strings.Replace(path, "\\", "/", -1)
	}
	if epnts, epntpaths, paths, _ := rscngmngr.findrsendpntpaths(path); epnts != nil && paths != nil {
		if len(epnts) > 0 && len(paths) == len(epnts) {

			abspath, err = epnts[0].fsabs(epntpaths[0], paths[0])

		}
	}
	return
}

func (rscngmngr *ResourcingManager) fsls(paths ...interface{}) (finfos []fsutils.FileInfo) {
	path := []string{}
	a := []interface{}{}
	for _, d := range paths {
		if ds, dsk := d.(string); dsk {
			path = append(path, ds)
		} else {
			a = append(a, d)
		}
	}
	if epnts, epntpaths, paths, _ := rscngmngr.findrsendpntpaths(path...); epnts != nil && paths != nil {
		if len(epnts) > 0 && len(paths) == len(epnts) {
			if finfos == nil {
				finfos = []fsutils.FileInfo{}
			}
			for nepnt := range epnts {
				nwa := []interface{}{}
				//for _, extpntpth := range epntpaths[nepnt] {
				nwa = append(nwa, epntpaths[nepnt])
				//}
				//nwa = append(nwa, paths[nepnt])
				if fis := epnts[0].fsls(append(nwa, a...)...); fis != nil {
					finfos = append(finfos, fis...)
				}
			}
		}
		epnts = nil
		paths = nil
	}
	return
}

func (rscngmngr *ResourcingManager) fsfindroot(path ...interface{}) (root string, err error) {
	if roots, rootserr := rscngmngr.fsfindroots(path...); rootserr == nil {
		if len(roots) > 0 {
			root = roots[0]
		}
	} else {
		err = rootserr
	}
	return
}

func (rscngmngr *ResourcingManager) fsfindroots(paths ...interface{}) (roots []string, err error) {
	path := []string{}
	for _, d := range paths {
		if ds, dsk := d.(string); dsk {
			path = append(path, ds)
		}
	}
	if epnts, epntpaths, paths, pathroots := rscngmngr.findrsendpntpaths(path...); epnts != nil && paths != nil {
		if len(epnts) > 0 && len(paths) == len(epnts) {
			var tmproots []string = nil
			var maxlen = 0
			for nepnt := range epnts {
				if fis, _ := epnts[0].fsfindroots(epntpaths[nepnt], paths[nepnt]); fis != nil {
					for _, fsrt := range fis {
						if len(fsrt) > maxlen {
							maxlen = len(fsrt)
							tmproots = append(tmproots, fsrt)
						}
					}
				}
			}
			for _, pthrt := range pathroots {
				if len(pthrt) > maxlen {
					maxlen = len(pthrt)
					tmproots = append(tmproots, pthrt)
				}
			}
			if maxlen > 0 {
				for _, tmprt := range tmproots {
					if len(tmprt) == maxlen {
						roots = append(roots, tmprt)
					}
				}
				tmproots = nil
			}
		}
		epnts = nil
		paths = nil
	}
	return
}

func (rscngmngr *ResourcingManager) fsfind(paths ...interface{}) (finfos []fsutils.FileInfo, err error) {
	path := []string{}
	a := []interface{}{}
	for _, d := range paths {
		if ds, dsk := d.(string); dsk {
			path = append(path, ds)
		} else {
			a = append(a, d)
		}
	}
	if epnts, epntpaths, paths, _ := rscngmngr.findrsendpntpaths(path...); epnts != nil && paths != nil {
		if len(epnts) > 0 && len(paths) == len(epnts) {
			if finfos == nil {
				finfos = []fsutils.FileInfo{}
			}
			for nepnt := range epnts {
				if fis, _ := epnts[nepnt].fsfind(epntpaths[nepnt], paths[nepnt]); fis != nil {
					finfos = append(finfos, fis...)
				}
			}
		}
		epnts = nil
		paths = nil
	}
	return
}

// RemovePathResource - Remove Endpoint Resource via path
func (rscngmngr *ResourcingManager) RemovePathResource(path string) (rmvd bool) {
	if path != "" {
		path = strings.Replace(path, "\\", "/", -1)
		if rune(path[0]) != '/' {
			path = "/" + path
		}
		if path == "/" {
			return
		}
		var rspthFound = ""

		for rsgnpath := range rscngmngr.rsngpaths {
			if len(rsgnpath) > len(rspthFound) && strings.HasPrefix(path, rsgnpath) {
				if len(rsgnpath) > len(rspthFound) {
					rspthFound = rsgnpath
				}
			}
		}
		if len(rspthFound) > 0 {
			if epntToRemove := rscngmngr.rsngpaths[rspthFound]; epntToRemove != nil {
				epntToRemove.RemoveResource(path[len(rspthFound):])
			}
			//rmvd = rscngmngr.rsngrootpaths[rscngmngr.rsngpaths[rspthFound]].RemoveResource(path[len(rspthFound):])
		}
	}
	return
}

/*// EndpointViaRootPath return ResourcingEndpoint via root path
func (rscngmngr *ResourcingManager) EndpointViaRootPath(rootpath string) (rsngendpt *ResourcingEndpoint) {
	if rootpath != "" {
		rsngendpt = rscngmngr.rsngrootpaths[rootpath]
	}
	return
}*/

// EndpointViaPath return ResourcingEndpoint via path
func (rscngmngr *ResourcingManager) EndpointViaPath(path string) (rsngendpt *ResourcingEndpoint) {
	if path != "" {
		if endpntpth, endpntpthok := rscngmngr.rsngpaths[path]; endpntpthok {
			rsngendpt = endpntpth //rscngmngr.rsngrootpaths[endpntpth]
		}
	}
	return
}

// EndpointResource - Endpoint embedded resource via path
func (rscngmngr *ResourcingManager) EndpointResource(path string) (epntrs interface{}) {
	if path != "" {
		path = strings.Replace(path, "\\", "/", -1)
		if rune(path[0]) != '/' {
			path = "/" + path
		}
		if path == "/" {
			return
		}
		var rspthFound = ""

		for rsgnpath := range rscngmngr.rsngpaths {
			if len(rsgnpath) > len(rspthFound) && strings.HasPrefix(path, rsgnpath) {
				if len(rsgnpath) > len(rspthFound) {
					rspthFound = rsgnpath
				}
			}
		}
		if len(rspthFound) > 0 {
			//epntrs = rscngmngr.rsngrootpaths[rscngmngr.rsngpaths[rspthFound]].Resource(path[len(rspthFound):])
			if endpt := rscngmngr.rsngpaths[rspthFound]; endpt != nil {
				epntrs = endpt.Resource(path[len(rspthFound):])
			}
			//epntrs = rscngmngr.rsngrootpaths[rscngmngr.rsngpaths[rspthFound]].Resource(path[len(rspthFound):])
		}
	}
	return
}

// UnregisterPaths unregister multiple paths
func (rscngmngr *ResourcingManager) UnregisterPaths(path ...string) {
	if len(path) > 0 {
		for _, pth := range path {
			if pth != "" {
				if pndpth, pthok := rscngmngr.rsngpaths[pth]; pthok {
					/*delete(rscngmngr.rsngpaths, pth)
					fndEndPtsh := false
					for _, ptepth := range rscngmngr.rsngpaths {
						if ptepth == pndpth {
							fndEndPtsh = true
							break
						}
					}
					if !fndEndPtsh {
						if rspnt := rscngmngr.rsngrootpaths[pndpth]; rspnt != nil {
							rspnt.dispose()
							rspnt = nil
						}
					}*/
					pndpth.dispose()
				}
			}
		}
	}
}

var emptypaths []string = make([]string, 0)

/*// RegisteredRootPaths return registered rootpaths
func (rscngmngr *ResourcingManager) RegisteredRootPaths() (paths []string) {
	if rscngmngr != nil {
		if ln := len(rscngmngr.rsngrootpaths); ln > 0 {
			paths = make([]string, ln)
			pi := 0
			for pth := range rscngmngr.rsngrootpaths {
				paths[pi] = pth
				pi++
			}
			return paths
		}
	}
	return emptypaths
}*/

// IsRegisteredPath return true indicating if a path is registered
func (rscngmngr *ResourcingManager) IsRegisteredPath(path string) (exists bool) {
	if path != "" {
		_, exists = rscngmngr.rsngpaths[path]
	}
	return
}

/*// IsRegisteredRootPath return true indicating if a rootpath is registered
func (rscngmngr *ResourcingManager) IsRegisteredRootPath(rootpath string) (exists bool) {
	if rootpath != "" {
		_, exists = rscngmngr.rsngrootpaths[rootpath]
	}
	return
}*/

// RegisteredPaths return registered paths
func (rscngmngr *ResourcingManager) RegisteredPaths() (paths []string) {
	if rscngmngr != nil {
		if ln := len(rscngmngr.rsngpaths); ln > 0 {
			paths = make([]string, ln)
			pi := 0
			for pth := range rscngmngr.rsngpaths {
				paths[pi] = pth
				pi++
			}
			return paths
		}
	}
	return emptypaths
}

// UnregisterPath - register path string
func (rscngmngr *ResourcingManager) UnregisterPath(path string) (rmvd bool) {
	if path != "" {
		if pndpth, pthok := rscngmngr.rsngpaths[path]; pthok {
			delete(rscngmngr.rsngpaths, path)
			pndpth.dispose()
			/*fndEndPtsh := false
			for _, ptepth := range rscngmngr.rsngpaths {
				if ptepth == pndpth {
					fndEndPtsh = true
					break
				}
			}
			if !fndEndPtsh {
				if rspnt := rscngmngr.rsngrootpaths[pndpth]; rspnt != nil {
					rspnt.dispose()
					rspnt = nil
				}
			}*/
		}
	}
	return
}

/*// UnregisterRootPaths unregister multiple RootPaths and their ResourcingEndPoints
func (rscngmngr *ResourcingManager) UnregisterRootPaths(epntpath ...string) {
	if len(epntpath) > 0 {
		for _, epth := range epntpath {
			if epth != "" {
				if rsndpt := rscngmngr.rsngrootpaths[epth]; rsndpt != nil {
					rsndpt.dispose()
				}
			}
		}
	}
}*/

/*// UnregisterRootPath unregister RootPath and dispose the ResourcingEndPoint
func (rscngmngr *ResourcingManager) UnregisterRootPath(epntpath string) (rmvd bool) {
	if epntpath != "" {
		if rsndpt := rscngmngr.rsngrootpaths[epntpath]; rsndpt != nil {
			rsndpt.dispose()
		}
	}
	return
}*/

// RegisterEndpoints register multiple Endpoints
func (rscngmngr *ResourcingManager) RegisterEndpoints(args ...interface{}) {
	var a []interface{} = nil
	var epntpath string = ""
	var path string = ""
	var argok = false
	for {
		if argsl := len(args); argsl >= 2 {
			if epntpath, argok = args[0].(string); argok {
				if path, argok = args[1].(string); argok {
					if argsl >= 3 {
						if a, argok = args[2].([]interface{}); argok {
							rscngmngr.RegisterEndpoint(epntpath, path, a...)
							args = args[3:]
						} else if argsl > 3 {
							rscngmngr.RegisterEndpoint(epntpath, path)
							args = args[2:]
						} else {
							break
						}
					} else {
						rscngmngr.RegisterEndpoint(epntpath, path)
						args = args[2:]
					}
				} else {
					break
				}
			} else {
				break
			}
		} else {
			break
		}
	}
}

// RegisterEndpoint - register ResourcingEndPoint
func (rscngmngr *ResourcingManager) RegisterEndpoint(path string, rootpath string, prms ...interface{}) {
	if path != "" {
		var israw = false
		var isactive = false
		if strings.Contains(path, "/raw:") {
			israw = true
			path = strings.Replace(path, "/raw:", "/", 1)
		}

		if strings.Contains(path, "/active:") {
			isactive = !israw
			path = strings.Replace(path, "/active:", "/", 1)
		}

		if rootpath == "./" {
			execpath := strings.Replace(os.Args[0], "\\", "/", -1)
			execpath = execpath[:strings.LastIndex(execpath, "/")+1]
			if fis, _ := fsutils.FIND(execpath); len(fis) > 0 && !fis[0].IsDir() {
				rootpath = strings.Replace(fis[0].AbsolutePath(), "\\", "/", -1)
				rootpath = rootpath[:strings.LastIndex(rootpath, "/")+1]
			} else {
				rootpath = strings.Replace(execpath, "\\", "/", -1)
			}
			if lsti := strings.LastIndex(rootpath, "/"); lsti > -1 {
				rootpath = rootpath[:strings.LastIndex(rootpath, "/")+1]
			} else {
				rootpath = "./"
			}
			if rootpath == "" || rootpath == "./" {
				if len(prms) == 0 {
					if fs := os.DirFS("./"); fs != nil {
						prms = append([]interface{}{}, fs)
					}
				}
			}
		}

		if _, rsngepntok := rscngmngr.rsngpaths[path]; !rsngepntok {
			if newrsngepnt, _ := nextResourcingEndpoint(rscngmngr, rootpath, israw, isactive, prms...); newrsngepnt != nil {
				//if newrsngepntpath == "" {
				//	newrsngepntpath = rsngepntpath
				//}
				rscngmngr.rsngpaths[path] = newrsngepnt
				rscngmngr.rsngendpaths[newrsngepnt] = path
				//rscngmngr.rsngrootpaths[newrsngepntpath] = newrsngepnt
			}
		}

		/*if rsngepnt, rsngepntok := rscngmngr.rsngpaths[path]; !rsngepntok {
			//if rsngepntpath == "" {
			//	if rsngepntpath = rootpath; rsngepntpath == "" {
			//		rsngepntpath = path
			//	}
			//}
			//if rscngmngr.rsngrootpaths[rsngepntpath] != nil {
			//	rscngmngr.rsngpaths[path] = rsngepntpath
			//} else {
			if newrsngepnt, newrsngepntpath := nextResourcingEndpoint(rscngmngr, rootpath, israw, isactive, prms...); newrsngepnt != nil {
				//if newrsngepntpath == "" {
				//	newrsngepntpath = rsngepntpath
				//}
				rscngmngr.rsngpaths[path] = newrsngepnt
				//rscngmngr.rsngrootpaths[newrsngepntpath] = newrsngepnt
			}
			//}
		} else if rsngepntok {
			if rscngmngr.rsngrootpaths[rsngepntpath] == nil {
				if newrsngepnt, newrsngepntpath := nextResourcingEndpoint(rscngmngr, rootpath, israw, isactive, prms...); newrsngepnt != nil {
					rscngmngr.rsngrootpaths[newrsngepntpath] = newrsngepnt
				}
			}
		} else {
			if rscngmngr.rsngpaths[path] != rootpath {
				if newrsngepnt, newrsngepntpath := nextResourcingEndpoint(rscngmngr, rootpath, israw, isactive, prms...); newrsngepnt != nil {
					rsngepnt, rsngepntok := rscngmngr.rsngrootpaths[newrsngepntpath]
					if rsngepntok {
						if rsngepnt != newrsngepnt {
							rsngepnt.dispose()
							rscngmngr.rsngrootpaths[newrsngepntpath] = newrsngepnt
							rscngmngr.rsngpaths[path] = newrsngepntpath
						}
					} else {
						rscngmngr.rsngrootpaths[newrsngepntpath] = newrsngepnt
						rscngmngr.rsngpaths[path] = newrsngepntpath
					}
				}
			}
		}*/
	}
}

// FindRSString - find Resource
func (rscngmngr *ResourcingManager) FindRSString(path string) (s string, modified time.Time, err error) {
	if rs, mdfd, rserr := rscngmngr.FindRS(path); rs != nil /*&& rs.isText*/ {
		modified = mdfd
		func() {
			defer rs.Close()
			p := make([]rune, 1024)
			pi := 0
			buf := bufio.NewReader(rs)
			for {
				r, size, rerr := buf.ReadRune()
				if size > 0 {
					p[pi] = r
					pi++
					if pi == len(p) {
						pi = 0
						s += string(p[:])
					}
				}
				if rerr != nil {
					if rerr == io.EOF {
						rerr = nil
					} else {
						err = rerr
					}
					break
				}
			}
			if pi > 0 {
				s += string(p[:pi])
			}
		}()
	} else if rserr != nil {
		err = rserr
	}
	return
}

// FindRS - find Resource
func (rscngmngr *ResourcingManager) FindRS(path string) (rs io.ReadCloser, modified time.Time, err error) {
	if path != "" {
		path = strings.Replace(path, "\\", "/", -1)
		if rune(path[0]) != '/' {
			path = "/" + path
		}
		if path == "/" {
			return
		}
		var rspthFound = ""

		for rsgnpath := range rscngmngr.rsngpaths {
			if len(rsgnpath) > len(rspthFound) && strings.HasPrefix(path, rsgnpath) {
				if len(rsgnpath) > len(rspthFound) {
					rspthFound = rsgnpath
				}
			}
		}
		if len(rspthFound) > 0 {
			//rs, modified, err = rscngmngr.rsngrootpaths[rscngmngr.rsngpaths[rspthFound]].findRS(path[len(rspthFound):])
			rs, modified, err = rscngmngr.rsngpaths[rspthFound].findRS(path[len(rspthFound):])
		} else {
			modified = time.Now()
		}
	}
	return
}

// Close *ResouringManager
func (rscngmngr *ResourcingManager) Close() (err error) {
	if rscngmngr != nil {
		if rscngmngr.fsutils != nil {
			rscngmngr.fsutils = nil
		}
		if rscngmngr.rsngpaths != nil {
			for pth := range rscngmngr.rsngpaths {
				rscngmngr.RemovePathResource(pth)
			}
			rscngmngr.rsngpaths = nil
		}
		/*if rscngmngr.rsngrootpaths != nil {
			rscngmngr.rsngrootpaths = nil
		}*/
		rscngmngr = nil
	}
	return
}

// NewResourcingManager - instance
func NewResourcingManager() (rscngmngr *ResourcingManager) {
	rscngmngr = &ResourcingManager{rsngendpaths: map[*ResourcingEndpoint]string{}, rsngpaths: map[string]*ResourcingEndpoint{}}
	return
}

var glbrscngmngr *ResourcingManager

// GLOBALRSNG - GLOBAL Resourcing for app
func GLOBALRSNG() *ResourcingManager {
	return glbrscngmngr
}

func init() {
	if glbrscngmngr == nil {
		glbrscngmngr = NewResourcingManager()
		glbrscngmngr.RegisterEndpoint("/", "./")
	}
}
