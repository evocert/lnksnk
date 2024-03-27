package parsing

import (
	"io"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"github.com/evocert/lnksnk/fsutils"
	"github.com/evocert/lnksnk/iorw"
)

var validLastCdeRuneMap = map[rune]uint8{'=': 1, '/': 1, '[': 1, '(': 1, ',': 1, '+': 1}

type elemfound struct {
	fs        *fsutils.FSUtils
	finfo     fsutils.FileInfo
	preelmbuf *iorw.Buffer
	elemName  string
	isdone    bool
	ctnt      *iorw.Buffer
	rawCtnt   *iorw.Buffer
	//tmplts     map[string]*iorw.Buffer
	//istmplt    bool
	postelmbuf *iorw.Buffer
	rdrne      func() (rune, int, error)
	//onclose    func(error)
}

func (elmfnd *elemfound) PreElmBuffer() (buf *iorw.Buffer) {
	if elmfnd != nil {
		if buf = elmfnd.preelmbuf; buf == nil {
			buf = iorw.NewBuffer()
			elmfnd.preelmbuf = buf
		}
	}
	return
}

func (elmfnd *elemfound) PostElmBuffer() (buf *iorw.Buffer) {
	if elmfnd != nil {
		if buf = elmfnd.postelmbuf; buf == nil {
			buf = iorw.NewBuffer()
			elmfnd.postelmbuf = buf
		}
	}
	return
}

func (elmfnd *elemfound) Modified() (mod time.Time) {
	if elmfnd != nil && elmfnd.finfo != nil {
		mod = elmfnd.finfo.ModTime()
	}
	return
}

func (elmfnd *elemfound) FullPath() (path string) {
	if elmfnd != nil && elmfnd.finfo != nil {
		path = elmfnd.finfo.Path()
	}
	return
}

func (elmfnd *elemfound) RawContentReader(onclose func(*elemfound, error) error) (rplsrnrdr *iorw.ReplaceRuneReader) {
	if elmfnd != nil && elmfnd.rawCtnt != nil && elmfnd.rawCtnt.Size() > 0 {
		rplsrnrdr = iorw.NewReplaceRuneReader(elmfnd.rawCtnt.Clone(true).Reader(true))
		if onclose != nil {
			rplsrnrdr.OnClose = func(rrr *iorw.ReplaceRuneReader, rrrerr error) (err error) {
				err = onclose(elmfnd, rrrerr)
				return
			}
		}
	}
	return
}

func (elmfnd *elemfound) Close() (err error) {
	if elmfnd != nil {
		if elmfnd.finfo != nil {
			elmfnd.finfo = nil
		}
		if elmfnd.fs != nil {
			elmfnd.fs = nil
		}
		if elmfnd.ctnt != nil {
			elmfnd.ctnt.Close()
			elmfnd.ctnt = nil
		}
		if elmfnd.rawCtnt != nil {
			elmfnd.rawCtnt.Close()
			elmfnd.rawCtnt = nil
		}
		if elmfnd.preelmbuf != nil {
			elmfnd.preelmbuf.Close()
			elmfnd.preelmbuf = nil
		}
		if elmfnd.postelmbuf != nil {
			elmfnd.postelmbuf.Close()
			elmfnd.postelmbuf = nil
		}
	}
	return
}

func (elmfnd *elemfound) Content() (ctnt *iorw.Buffer) {
	if elmfnd != nil {
		if ctnt = elmfnd.ctnt; ctnt == nil {
			ctnt = iorw.NewBuffer()
			elmfnd.ctnt = ctnt
		}
	}
	return
}

func (elmfnd *elemfound) RawContent() (ctnt *iorw.Buffer) {
	if elmfnd != nil {
		if ctnt = elmfnd.rawCtnt; ctnt == nil {
			ctnt = iorw.NewBuffer()
			elmfnd.rawCtnt = ctnt
		}
	}
	return
}

func (elmfnd *elemfound) ReadRune() (r rune, size int, err error) {
	if elmfnd != nil && elmfnd.rdrne != nil {
		r, size, err = elmfnd.rdrne()
	}
	return
}

func (elmfnd *elemfound) NextRuneReader(preAppendElem bool, postAppendElem bool, onclose func(*elemfound, error) error, ctntbuf *iorw.Buffer) (rdr io.RuneReader) {
	if elmfnd != nil && elmfnd.fs != nil && elmfnd.finfo != nil {
		rdr, _ = elmfnd.fs.CAT(elmfnd.finfo.Path()).(io.RuneReader)
		path := elmfnd.finfo.Path()
		pathroot := path
		pthexti, pathpthi := strings.LastIndex(pathroot, "."), strings.LastIndex(pathroot, "/")
		if pathpthi > -1 {
			if pthexti > pathpthi {
				pathroot = pathroot[:pathpthi+1]
			}
			pathroot = pathroot[:pathpthi+1]
		} else {
			pathroot = "/"
		}
		path = path[len(pathroot):]
		if strings.HasSuffix(elmfnd.elemName, ":") {
			path = ""
		}
		if prebuf, postbuf := elmfnd.preelmbuf, elmfnd.postelmbuf; prebuf != nil || postbuf != nil {
			if ctntbuf == nil {
				ctntbuf = iorw.NewBuffer()

				if prebuf != nil && prebuf.Size() > 0 {
					ctntbuf.Print("<:_:pre:>" + prebuf.String() + "</:_:pre:>")
					prebuf.Clear()
				}
				if postbuf != nil && postbuf.Size() > 0 {
					ctntbuf.Print("<:_:post:>" + postbuf.String() + "</:_:post:>")
					prebuf.Clear()
				}
			} else {
				if !ctntbuf.Contains("<:_:pre:>") && prebuf != nil && prebuf.Size() > 0 {
					ctntbuf.Print("<:_:pre:>" + prebuf.String() + "</:_:pre:>")
					prebuf.Clear()
				}
				if !ctntbuf.Contains("<:_:post:>") && postbuf != nil && postbuf.Size() > 0 {
					ctntbuf.Print("<:_:post:>" + postbuf.String() + "</:_:post:>")
					prebuf.Clear()
				}
			}
		}

		if ctntbuf != nil {
			tmplts := []interface{}{}
			if ctntbuf.Contains("<:_:") {
				rdr := ctntbuf.Clone(true).Reader(true)
				tmpltcntrdr := NewUntilRuneReader(rdr, "<:_:")
				tmpbuf := iorw.NewBuffer()
				for !tmpltcntrdr.Done {
					tmpltcntrdr.WriteTo(tmpbuf)
					if tmpltcntrdr.FoundUntil {
						if tmpbuf.Size() > 0 {
							tmpbuf.WriteTo(ctntbuf)
							tmpbuf.Clear()
						}
						tmpltcntrdr.NextUntil(":>")
						tmpltcntrdr.WriteTo(tmpbuf)
						if tmpltcntrdr.FoundUntil {
							if tmpbuf.Size() > 0 {
								tmpltnme := tmpbuf.String()
								tmpbuf.Clear()
								tmpltcntrdr.NextUntil("</:_:" + tmpltnme + ":>")
								tmpltcntrdr.WriteTo(tmpbuf)
								if tmpltcntrdr.FoundUntil {
									tmplts = append(tmplts, "<:_:"+tmpltnme+":/>", tmpbuf.String())
									tmpbuf.Clear()
									tmpltcntrdr.NextUntil("<:_:")
								} else {
									tmpbuf.Clear()
								}
							}
						}
					}
				}
				if tmpbuf.Size() > 0 {
					tmpbuf.WriteTo(ctntbuf)
					tmpbuf.Clear()
				}
			}
			rplcrnrdr := iorw.NewReplaceRuneReader(rdr, append([]interface{}{"<:cntnt:/>", ctntbuf.String(), "<:_:pathroot:/>", pathroot, "<:_:elemroot:/>", func() (elmroot string) {
				if path == "" {
					if strings.HasSuffix(pathroot, "/") {
						if pthi := strings.LastIndex(pathroot[:len(pathroot)-1], "/"); pthi > -1 {
							elmroot = strings.Replace(pathroot[:pthi+1], "/", ":", -1)
						} else {
							elmroot = ""
						}
					} else if pthi := strings.LastIndex(pathroot, "/"); pthi > -1 {
						elmroot = strings.Replace(pathroot[:pthi+1], "/", ":", -1)
					} else {
						elmroot = ""
					}
				} else if pthi := strings.LastIndex(pathroot, "/"); pthi > -1 {
					elmroot = strings.Replace(pathroot[:pthi+1], "/", ":", -1)
				} else {
					elmroot = ""
				}
				return
			}()}, tmplts...)...)
			if preAppendElem {
				rplcrnrdr.PreAppend(strings.NewReader("<" + elmfnd.elemName + ">"))
			}
			if postAppendElem {
				rplcrnrdr.PostAppend(strings.NewReader("</" + elmfnd.elemName + ">"))
			}
			if onclose != nil {
				rplcrnrdr.OnClose = func(rrr *iorw.ReplaceRuneReader, rrrerr error) (err error) {
					err = onclose(elmfnd, rrrerr)
					return
				}
			}
			elmfnd.rdrne = rplcrnrdr.ReadRune
		} else {
			rplcrnrdr := iorw.NewReplaceRuneReader(rdr, "<:cntnt:/>", "", "<:_:pathroot:/>", pathroot, "<:_:elemroot:/>", func() (elmroot string) {
				if path == "" {
					if strings.HasSuffix(pathroot, "/") {
						if pthi := strings.LastIndex(pathroot[:len(pathroot)-1], "/"); pthi > -1 {
							elmroot = strings.Replace(pathroot[:pthi+1], "/", ":", -1)
						} else {
							elmroot = ""
						}
					} else if pthi := strings.LastIndex(pathroot, "/"); pthi > -1 {
						elmroot = strings.Replace(pathroot[:pthi+1], "/", ":", -1)
					} else {
						elmroot = ""
					}
				} else if pthi := strings.LastIndex(pathroot, "/"); pthi > -1 {
					elmroot = strings.Replace(pathroot[:pthi+1], "/", ":", -1)
				} else {
					elmroot = ""
				}
				return
			}())
			if preAppendElem {
				rplcrnrdr.PreAppend(strings.NewReader("<" + elmfnd.elemName + ">"))
			}
			if postAppendElem {
				rplcrnrdr.PostAppend(strings.NewReader("</" + elmfnd.elemName + ">"))
			}
			if onclose != nil {
				rplcrnrdr.OnClose = func(rrr *iorw.ReplaceRuneReader, rrrerr error) (err error) {
					err = onclose(elmfnd, rrrerr)
					return
				}
			}
			elmfnd.rdrne = rplcrnrdr.ReadRune
		}
		return elmfnd
	}
	return
}

func nextElemFound(elemName string, finfo fsutils.FileInfo, fs *fsutils.FSUtils) (nxtelmfnd *elemfound) {
	nxtelmfnd = &elemfound{elemName: elemName, finfo: finfo, fs: fs}
	return
}

func (elmfnd *elemfound) Clone() (nxtelmfnd *elemfound) {
	if elmfnd != nil {
		nxtelmfnd = &elemfound{elemName: elmfnd.elemName, finfo: elmfnd.finfo, fs: elmfnd.fs}
	}
	return
}

func (elmfnd *elemfound) Path() string {
	return elmfnd.finfo.Path()
}

func (elmfnd *elemfound) Ext() (ext string) {
	ext = filepath.Ext(elmfnd.Path())
	return
}

func (elmfnd *elemfound) PathRoot() (pthroot string) {
	if pthroot = elmfnd.Path(); strings.Contains(pthroot, "/") {
		pthroot = pthroot[:strings.LastIndex(pthroot, "/")+1]
	} else {
		pthroot = "/"
	}
	return
}

func internalProcessParsing(
	parseOnly bool,
	canParse bool,
	cancache bool,
	pathModified time.Time,
	path, pathroot, pathext string,
	out io.Writer,
	fs *fsutils.FSUtils,
	invertActive bool,
	evalcode func(*iorw.BuffReader, func() interface{}, func(interface{})) error,
	rnrdrs ...io.RuneReader) (prsngerr error) {

	fullpath := pathroot + path

	/*if cancache {
		if chdscrpt := GLOBALCACHEDSCRIPTING().Script(func() (scrptpath string) {
			if invertActive {
				return "/active:" + fullpath
			}
			return fullpath
		}()); chdscrpt != nil {
			if !chdscrpt.IsValidSince(pathModified, fs) {
				chdscrpt.Dispose()
				chdscrpt = nil
			} else if chdscrpt != nil {
				if out != nil {
					_, prsngerr = chdscrpt.WritePsvTo(out)
				}
				if prsngerr == nil && evalcode != nil {
					if prsngerr = chdscrpt.EvalAtv(evalcode); prsngerr != nil {
						chdscrpt.Dispose()
					}
				}
				return
			}
		}
	}*/

	var chdpsvbuf *iorw.Buffer = nil
	ctntrdr := iorw.NewRuneReaderSlice(iorw.NewReplaceRuneReader(iorw.NewRuneReaderSlice(rnrdrs...), "<:_:pathroot:/>", pathroot, "<:_:elemroot:/>", func() (elmroot string) {
		if path == "" {
			if strings.HasSuffix(pathroot, "/") {
				if pthi := strings.LastIndex(pathroot[:len(pathroot)-1], "/"); pthi > -1 {
					elmroot = strings.Replace(pathroot[:pthi+1], "/", ":", -1)
				} else {
					elmroot = ""
				}
			} else if pthi := strings.LastIndex(pathroot, "/"); pthi > -1 {
				elmroot = strings.Replace(pathroot[:pthi+1], "/", ":", -1)
			} else {
				elmroot = ""
			}
		} else if pthi := strings.LastIndex(pathroot, "/"); pthi > -1 {
			elmroot = strings.Replace(pathroot[:pthi+1], "/", ":", -1)
		} else {
			elmroot = ""
		}
		return
	}()))

	cderdr := iorw.NewRuneReaderSlice()

	prcsection := 0

	cnttxtr := rune(0)
	cntprvr := rune(0)

	cntntpsvlbl := []rune("<")
	cntntatvlbl := []rune(">")
	cntntlbli := []int{0, 0}

	cntntprserr := error(nil)

	var cntelmsfound = map[string]*elemfound{}
	var cntelmsfoundlevel = map[int]*elemfound{}
	var cntelmsfndL = 0
	cntntprevr := func() rune {
		return cntprvr
	}

	cntntsetprevr := func(r rune) {
		cntprvr = r
	}

	cntntcancheckpsv := func() bool {
		return true
	}

	cntntcancheckatv := func() bool {
		return cnttxtr == 0
	}

	cntpsvbuf := iorw.NewBuffer()
	defer cntpsvbuf.Close()
	cntntprcspsvrunes := func(rns ...rune) (prserr error) {
		cntpsvbuf.WriteRunes(rns...)
		return
	}

	cntntflushpsv := func() (prserr error) {
		if cntpsvbuf.Size() > 0 {
			if cntelm := cntelmsfoundlevel[len(cntelmsfoundlevel)-1]; cntelm != nil {
				if cntelm.isdone {
					cntpsvbuf.WriteTo(cntelm.RawContent())
				} else {
					cntpsvbuf.WriteTo(cntelm.Content())
				}
				cntpsvbuf.Clear()
			} else {
				cderdr.PreAppend(cntpsvbuf.Clone(true).Reader(true))
				prcsection = 1
			}
		}
		return
	}

	cntatvbuf := iorw.NewBuffer()
	defer cntatvbuf.Close()
	isEndElem, isSingleElem, isStartElem := false, false, false
	crntElemName := ""

	checkfoundElm := false

	var resetCntntCheckElem = func(incldeend bool, resetonly bool, r ...rune) {
		cntprvr = rune(0)
		cnttxtr = rune(0)
		if !resetonly {
			cntpsvbuf.WriteRunes(cntntpsvlbl...)
		}
		if checkfoundElm {
			checkfoundElm = false
			if crntElemName != "" {
				if !resetonly {
					cntpsvbuf.Print(crntElemName)
				}
				crntElemName = ""
			}
			if !isSingleElem {
				if !resetonly {
					cntatvbuf.WriteTo(cntpsvbuf)
				}
				cntatvbuf.Clear()
			} else {
				if !resetonly {
					cntatvbuf.WriteTo(cntpsvbuf)
					cntatvbuf.Clear()
					if !cntpsvbuf.HasSuffix("/") {
						cntpsvbuf.Print("/")
					}
				}
				isSingleElem = false
			}
			if !resetonly {
				cntpsvbuf.WriteRunes(r...)
			}
		} else {
			if isEndElem {
				if !resetonly {
					cntpsvbuf.WriteRune('/')
				}
				isEndElem = false
			}
			if crntElemName != "" {
				if !resetonly {
					cntpsvbuf.Print(crntElemName)
				}
				crntElemName = ""
			}
			if !resetonly {
				cntpsvbuf.WriteRunes(r...)
			}
		}
		if isStartElem {
			isStartElem = false
		}
		if incldeend {
			if !resetonly {
				cntpsvbuf.WriteRunes(cntntatvlbl...)
			}
		}
	}

	var invalidElems = map[string]bool{}

	var checkifCntntElem = func(r ...rune) (failed bool, chkerr error) {
		var lstcn = -1
		for cn, cr := range r {
			if checkfoundElm {
				if cnttxtr == 0 {
					if iorw.IsTxtPar(cr) {
						cnttxtr = cr
						cntatvbuf.WriteRune(cr)
					} else {
						if !isSingleElem {
							if cr == '/' {
								isSingleElem = true
							} else {
								cntatvbuf.WriteRune(cr)
							}
						} else {
							lstcn = cn
							failed = true
							break
						}
					}
				} else {
					if cnttxtr == cr {
						cnttxtr = 0
					}
					cntatvbuf.WriteRune(cr)
				}
			} else {
				if iorw.IsTxtPar(cr) {
					lstcn = cn
					failed = true
					break
				} else if cr == '@' {
					lstcn = cn
					failed = true
					break
				} else if cr == '/' {
					if len(crntElemName) > 0 && !invalidElems[crntElemName] {
						if isStartElem {
							failed = true
							lstcn = cn
							break
						}
						isSingleElem = true
						checkfoundElm = true
						cntprvr = 0
						return
					} else if crntElemName == "" {
						isEndElem = true
						cntprvr = 0
						return
					} else {
						failed = true
						lstcn = cn
						break
					}
				} else if unicode.IsSpace(cr) {
					if len(crntElemName) > 0 && !invalidElems[crntElemName] {
						cntatvbuf.WriteRune(cr)
						checkfoundElm = true
					} else {
						failed = true
						lstcn = cn
						break
					}
				} else if validElemChar(cntprvr, cr) {
					crntElemName += string(cr)
				} else {
					failed = true
					lstcn = cn
					break
				}
			}
			cntprvr = cr
		}

		if !failed {
			failed = chkerr != nil
		} else if failed {
			if lstcn > -1 {
				resetCntntCheckElem(false, false, r[lstcn:]...)
			} else {
				resetCntntCheckElem(false, false)
			}
		}
		return
	}

	var chkfailed, cherr = false, error(nil)
	cntntprcsatvrunes := func(rns ...rune) (failed bool, prserr error) {
		if chkfailed, cherr = checkifCntntElem(rns...); cherr != nil {
			prserr = cherr
		} else if failed = chkfailed; failed {
			cntntflushpsv()
		}
		return
	}

	var elemName = ""
	var elemPath = ""
	var elemExt = ""

	currentPathRoot := func() string {
		if cntemlfndl := len(cntelmsfoundlevel); cntemlfndl > 0 {
			return cntelmsfoundlevel[cntemlfndl-1].PathRoot()
		} else {
			return pathroot
		}
	}

	currentElemName := func() (elmnme string) {
		if cntemlfndl := len(cntelmsfoundlevel); cntemlfndl > 0 {
			elmnme = cntelmsfoundlevel[cntemlfndl-1].elemName
		} else {
			if lstpthi := strings.LastIndex(path, "."); lstpthi > 0 {
				elmnme = path[:lstpthi]
			} else {
				elmnme = path
			}
		}
		return
	}

	prepairElemName := func(elmNmeToTest string) (elmnmeprpd string) {
		if !strings.HasPrefix(elmNmeToTest, ":") {
			if crntnme := currentElemName(); crntnme != "" {
				if isEndElem && strings.HasSuffix(crntnme, elmNmeToTest) {
					elmnmeprpd = crntnme[:len(crntnme)-len(elmNmeToTest)] + elmNmeToTest
				} else {
					if colonsepi := strings.LastIndex(crntnme, ":"); colonsepi > -1 {
						elmnmeprpd = crntnme[:colonsepi+1] + elmNmeToTest
					} else {
						elmnmeprpd = elmNmeToTest
					}
				}
			} else {
				elmnmeprpd = elmNmeToTest
			}
		} else {
			elmnmeprpd = elmNmeToTest
		}
		return
	}

	var elemToUse *elemfound = nil
	var istmplt = false
	var validCntntElem = func() (valid bool) {
		if isStartElem = (crntElemName != "" && !isStartElem && !isSingleElem && !isEndElem); fs != nil && ((isSingleElem || isStartElem) || isEndElem) && crntElemName != "" {
			if elemExt = filepath.Ext(crntElemName); elemExt != "" {
				if elemExt == pathext {
					crntElemName = crntElemName[:len(crntElemName)-len(elemExt)]
				}
			}
			if istmplt = strings.HasPrefix(crntElemName, ":_:"); istmplt {
				return
			}
			if elemName = prepairElemName(crntElemName); invalidElems[elemName] {
				elemToUse = nil
				return
			} else if elemToUse = cntelmsfound[elemName]; elemToUse != nil {
				valid = true
				return
			} else if isEndElem {
				return
			}

			elemPath = strings.Replace(elemName, ":", "/", -1)
			if elemPath[0] != '/' {
				crntelempath := currentPathRoot()
				if strings.Contains(elemPath, "/") {
					elemPath = func() (pthfnd string) {
						testElemPath := elemPath
						for testElemPath != "" {
							if ctnsi := strings.LastIndex(testElemPath, "/"); ctnsi > 0 {
								if strings.HasSuffix(crntelempath, testElemPath[:ctnsi+1]) {
									testElemPath = testElemPath[:ctnsi+1]
									break
								}
								testElemPath = testElemPath[:ctnsi]
							} else {
								testElemPath = ""
								break
							}
						}
						return crntelempath + elemPath[len(testElemPath):]
					}()
				} else {
					elemPath = crntelempath + elemPath
				}
			}
			if elemExt == "" {
				elemExt = pathext
			}
			if invalidElems[elemName] {
				elemToUse = nil
				return
			} else {
				var elmfnd *elemfound = nil
				if elemExt != "" {
					if pthi := strings.LastIndex(elemPath, "."); pthi > -1 && elemExt == elemPath[pthi:] {
						elemPath = elemPath[:pthi]
					}
				}
				if elemExt != "" {
					if strings.HasSuffix(elemPath, "/") {
						for _, finfo := range fs.LS(elemPath + "index" + elemExt) {
							if !finfo.IsDir() {
								elmfnd = nextElemFound(elemName, finfo, fs)
							}
						}
					} else {
						for _, finfo := range fs.LS(elemPath + elemExt) {
							if !finfo.IsDir() {
								elmfnd = nextElemFound(elemName, finfo, fs)
							}
						}
					}
				}
				if elmfnd == nil && elemExt == "" {
					if strings.HasSuffix(elemPath, "/") {
						for _, nxtext := range []string{".html"} {
							for _, finfo := range fs.LS(elemPath + nxtext) {
								if !finfo.IsDir() {
									elmfnd = nextElemFound(elemName, finfo, fs)
								}
							}
							if elmfnd != nil {
								break
							}
						}
					} else {
						for _, nxtext := range []string{".html"} {
							for _, finfo := range fs.LS(elemPath + nxtext) {
								if !finfo.IsDir() {
									elmfnd = nextElemFound(elemName, finfo, fs)
								}
							}
							if elmfnd != nil {
								break
							}
						}
					}
				}
				if valid = elmfnd != nil; valid {
					cntelmsfound[elemName] = elmfnd
					elemToUse = elmfnd
				} else {
					invalidElems[elemName] = true
					elemToUse = nil
				}
			}
		}
		return
	}

	cntntflushatv := func() (prserr error) {
		validElem, startElem, singleElem, endElem := validCntntElem(), isStartElem, isSingleElem, isEndElem
		if elemToUse != nil {
			cntntflushpsv()
			if validElem && (startElem || singleElem) {
				elemToUse = elemToUse.Clone()
				if cntatvbuf.Size() > 0 {
					elemToUse.PreElmBuffer().Print(strings.TrimFunc(cntatvbuf.String(), iorw.IsSpace))
					cntatvbuf.Clear()
				}
				elemToUse.isdone = singleElem
				resetCntntCheckElem(true, true)
				cntelmsfoundlevel[len(cntelmsfoundlevel)] = elemToUse
				cntelmsfndL++
				if singleElem {
					ctntrdr.PreAppend(elemToUse.NextRuneReader(false, true, func(elmfnd *elemfound, rerr error) (clserr error) {

						return
					}, nil))
				}
				elemToUse = nil
			} else if validElem && endElem {
				if cntntelml, ctntelm := len(cntelmsfoundlevel), cntelmsfoundlevel[len(cntelmsfoundlevel)-1]; ctntelm != nil {
					if ctntelm.elemName == elemToUse.elemName {
						if !ctntelm.isdone {
							if cntatvbuf.Size() > 0 {
								ctntelm.PostElmBuffer().Print(strings.TrimFunc(cntatvbuf.String(), iorw.IsSpace))
								cntatvbuf.Clear()
							}
							resetCntntCheckElem(true, true)
							ctntelm.isdone = true
							if ctntelm.ctnt != nil && ctntelm.ctnt.Size() > 0 {
								ctntrdr.PreAppend(ctntelm.NextRuneReader(false, true, func(elmfnd *elemfound, rerr error) (clserr error) {

									return
								}, ctntelm.ctnt.Clone(true)))
							} else {
								ctntrdr.PreAppend(ctntelm.NextRuneReader(false, true, func(elmfnd *elemfound, rerr error) (clserr error) {

									return
								}, nil))
							}
						} else {
							resetCntntCheckElem(true, true)
							if rplcrwctnt := ctntelm.RawContentReader(func(elmfnd *elemfound, rerr error) (clserr error) {

								return
							}); rplcrwctnt != nil {
								ctntrdr.PreAppend(rplcrwctnt)

							}
							ctntelm.Close()
							delete(cntelmsfoundlevel, cntntelml-1)
							cntelmsfndL--
						}
					}
				}
				elemToUse = nil
				resetCntntCheckElem(true, true)
			} else {
				resetCntntCheckElem(true, false)
				cntntflushpsv()
			}
			elemToUse = nil
		} else {
			if istmplt {
				resetCntntCheckElem(true, istmplt && singleElem)
				istmplt = false
			} else {
				resetCntntCheckElem(true, false)
			}
			cntntflushpsv()
		}
		return
	}

	cdetxtr := rune(0)
	cdemde := ""
	cdeprvr := rune(0)
	cdelstr := rune(0)

	cdepsvlbl := []rune("<@")
	cdeatvlbl := []rune("@>")
	cdelbli := []int{0, 0}

	cdeprserr := error(nil)

	cdepsvbuf := iorw.NewBuffer()
	defer cdepsvbuf.Close()
	cdeatvbuf := iorw.NewBuffer()
	defer cdeatvbuf.Close()

	cdefoundcode := false
	cdeHasCode := false

	if invertActive {
		cdelbli[0] = 2
		cdelbli[1] = 0
		cdefoundcode = true
	}

	codebuf := iorw.NewBuffer()
	defer codebuf.Close()

	cdeprevr := func() rune {
		return cdeprvr
	}

	cdesetprevr := func(r rune) {
		cdeprvr = r
	}

	cdecancheckpsv := func() bool {
		return true
	}

	cdecancheckatv := func() bool {
		return cdetxtr == 0 && cdemde == ""
	}

	cdeprcspsvrunes := func(rns ...rune) (prserr error) {
		if cdeHasCode {
			cdeHasCode = false
		}
		cdepsvbuf.WriteRunes(rns...)
		return
	}

	cdeflushpsv := func() (prserr error) {
		if cdepsvs := cdepsvbuf.Size(); cdepsvs > 0 {
			hstmpltfx := cdepsvbuf.HasPrefix("`") && cdepsvbuf.HasSuffix("`") && cdepsvs >= 2
			if hstmpltfx && !cdefoundcode {
				cdefoundcode = true
			}
			if !parseOnly && cdefoundcode {
				if cdelstr > 0 {
					if hstmpltfx {
						cdeatvbuf.Print(cdepsvbuf.Clone(true).Reader(true))
					} else {
						if cdepsvbuf.Contains("${") || cdepsvbuf.Contains("`") {
							cdeatvbuf.Print("`", iorw.NewReplaceRuneReader(cdepsvbuf.Clone(true).Reader(true), "`", "\\`", "${", "\\${"), "`")
						} else {
							cdeatvbuf.Print("`", iorw.NewReplaceRuneReader(cdepsvbuf.Clone(true).Reader(true), `"\`, `"\\`), "`")
						}
					}
					cdelstr = 0
				} else {
					if hstmpltfx {
						cdeatvbuf.Print("print(", iorw.NewReplaceRuneReader(cdepsvbuf.Clone(true).Reader(true), `"\`, `"\\`), ");")
					} else {
						if cdepsvbuf.Contains("${") || cdepsvbuf.Contains("`") {
							cdeatvbuf.Print("print(`", iorw.NewReplaceRuneReader(cdepsvbuf.Clone(true).Reader(true), `"\`, `"\\`, "`", "\\`", "${", "\\${"), "`);")
						} else {
							cdeatvbuf.Print("print(`", iorw.NewReplaceRuneReader(cdepsvbuf.Clone(true).Reader(true), `"\`, `"\\`), "`);")
						}
					}
				}
			} else {
				if _, prserr = cdepsvbuf.WriteTo(out); prserr == nil {
					if cancache {
						if chdpsvbuf == nil {
							chdpsvbuf = iorw.NewBuffer()
						}
						_, prserr = cdepsvbuf.WriteTo(chdpsvbuf)
					}
					cdepsvbuf.Clear()
				}
			}
		}
		return
	}

	validcdecnmst := map[string]string{"//": "S", "/*": "MS", "*/": "ME", "import": "I"}
	cdetmps := ""
	cdelstmde := ""
	cdeimpoffset := int64(0)

	var codeFlushImports = func(cdeimprtatvbuf *iorw.Buffer) {
		if cdeimprtatvbuf.Size() > 0 {
			defer cdeimprtatvbuf.Clear()
			if tmpimports := cdeimprtatvbuf.String(); tmpimports != "" {
				cdeimprtatvbuf.Clear()
				tmpimports = strings.TrimSpace(tmpimports)
				imprtcde := ""
				if frmindex := strings.Index(tmpimports, "from"); frmindex > 0 {
					tmpimports1 := strings.TrimSpace(tmpimports[:frmindex])
					tmpimports2 := strings.TrimSpace(tmpimports[frmindex+len("from"):])

					if tmpimports1 != "" && tmpimports2 != "" {
						imprtcde += `var _default_mod=require(` + tmpimports2 + `);`
						tstxtr := rune(0)
						tstbrc := 0
						crntelems := []string{}
						crntmod := []string{}
						tmprns := []rune{}
						cdemode := "_default_mod"
						capturetmprns := func() {
							if len(tmprns) > 0 {
								if crnttmp := string(tmprns); crnttmp != "" {
									if tstbrc == 1 {
										crntelems = append(crntelems, crnttmp)
									} else if tstbrc == 0 {
										crntmod = append(crntmod, crnttmp)
									}
								}
							}
							tmprns = nil
						}
						propImportCode := func() {
							if tstbrc == 1 {
								if crntelml := len(crntelems); crntelml > 0 {
									if crntelml == 3 && crntelems[1] == "as" {
										imprtcde += "const " + crntelems[2] + "=" + cdemode + `["` + crntelems[0] + `"];`
									} else if crntelml == 1 {
										imprtcde += "const " + crntelems[0] + "=" + cdemode + `["` + crntelems[0] + `"];`
									}
									crntelems = nil
								}
							} else {
								if crntmdl := len(crntmod); crntmdl > 0 {
									if crntmdl == 3 && crntmod[1] == "as" {
										imprtcde += "const " + crntmod[2] + "=" + cdemode + ";"
									} else if crntmdl == 1 {
										imprtcde += "const " + crntmod[1] + "=" + cdemode + ";"
									}
									crntmod = nil
								}
							}
						}
						for _, tsr := range tmpimports1 {
							if tstxtr == 0 {
								if tsr == '"' {
									tstxtr = tsr
									continue
								}
								if tsr == '{' {
									tstbrc = 1
								} else if tsr == '}' {
									capturetmprns()
									propImportCode()
									tstbrc = 0
								} else if tsr == ',' {
									capturetmprns()
									if tstbrc == 0 {
										propImportCode()
									}
								} else {
									if !iorw.IsSpace(tsr) {
										tmprns = append(tmprns, tsr)
									} else {
										capturetmprns()
									}
								}
							} else {
								if tstxtr == tsr {
									tstxtr = 0
									continue
								}
								tmprns = append(tmprns, tsr)
							}
						}
						capturetmprns()
						propImportCode()
					}
				} else {
					imprtcde += `require(` + tmpimports + `);`
				}
				cdeatvbuf.Print(imprtcde)
			}
		}
	}

	cdeprcsatvrunes := func(rns ...rune) (failed bool, prserr error) {
		if !cdeHasCode {
			cdeHasCode = true
			if cdemde != "" {
				cdemde = ""
			}
			if cdetmps != "" {
				cdetmps = ""
			}
			if !cdefoundcode {
				cdefoundcode = true
			}
		}
		for _, r := range rns {
			if cdetxtr == 0 {
				if cdeprvr != '\\' && iorw.IsTxtPar(r) {
					cdetxtr = r
					cdelstr = 0
					cdetmps = ""
				} else {
					if !iorw.IsSpace(r) {
						if validLastCdeRuneMap[r] > 0 {
							if r == '/' {
								if !strings.ContainsFunc("*/", func(r rune) bool {
									return r == cdeprvr
								}) {
									cdelstr = r
								} else {
									cdelstr = 0
								}
							} else {
								cdelstr = r
							}
						} else {
							cdelstr = 0
						}
						if cdemde == "" {
							if cdemde = validcdecnmst[cdetmps]; cdemde != "" {
								cdetmps = ""
								if cdelstmde = cdemde; cdelstmde == "I" {
									cdeimpoffset = cdeatvbuf.Size()
								} else {
									cdeimpoffset = -1
								}
							} else if len(cdetmps) >= 6 {
								cdetmps = ""
							}
						} else if cdelstmde == "MS" {
							if validcdecnmst[cdetmps] == "ME" {
								cdemde = ""
								cdelstmde = ""
								cdetmps = ""
							} else if len(cdetmps) >= 2 {
								cdetmps = ""
							}
						} else if cdelstmde == "I" {
							if r == ';' {
								cdelstmde = ""
								if cdeimpoffset > -1 {
									cdeimpoffset -= int64(len("import"))
									cdeatvprebuf := iorw.NewBuffer()
									cdeatvprebuf.ReadFrom(cdeatvbuf.Reader(0, cdeimpoffset+1))
									cdeimprtbuf := iorw.NewBuffer()
									cdeimprtbuf.ReadFrom(cdeatvbuf.Reader(cdeimpoffset, cdeatvbuf.Size()))
									cdeimpoffset = -1
									cdeatvbuf.Clear()
									cdeatvprebuf.WriteTo(cdeatvbuf)
									cdeatvprebuf.Clear()
									codeFlushImports(cdeimprtbuf)
									if !cdeatvbuf.HasSuffix(";") {
										cdeatvbuf.WriteRune(r)
									}
								}
							}
							cdetmps = ""
						} else {
							cdetmps = ""
						}
					} else {
						if cdetmps != "" {
							cdetmps = ""
						}
						if r == '\n' {
							if cdemde == "S" {
								cdemde = ""
								cdelstmde = ""
							}
						}
					}
				}
			} else if cdetxtr == r {
				if cdeprvr != '\\' {
					cdetxtr = 0
				}
			}
			cdeprvr = r
		}
		cdeatvbuf.WriteRunes(rns...)
		return
	}

	cdeflushatv := func() (prserr error) {
		if cdeatvbuf.Size() > 0 {
			cdeatvbuf.WriteTo(codebuf)
			cdeatvbuf.Clear()
		}
		return
	}

	var failed = false
	var crntprsrfunc = func(r rune, prvr func() rune, setprevr func(rune), cancheckpsv func() bool, processpsvrunes func(...rune) error, flushpsv func() error, cancheckatv func() bool, processatvrunes func(...rune) (bool, error), flushatv func() error, lbli []int, psvlbl []rune, atvlbl []rune) (prserr error) {

		if psvi, psvl, atvi, atvl := lbli[0], len(psvlbl), lbli[1], len(atvlbl); atvi == 0 && psvi < psvl {
			if psvi > 0 && psvlbl[psvi-1] == prvr() && psvlbl[psvi] != r {
				lbli[0] = 0
				setprevr(0)
				if prserr = processpsvrunes(psvlbl[:psvi]...); prserr != nil {
					return
				}
			}
			if cancheckpsv() && psvlbl[psvi] == r {
				lbli[0]++
				if (psvi + 1) == psvl {
					flushpsv()
					setprevr(0)
				}
			} else {
				if psvi > 0 {
					lbli[0] = 0
					setprevr(0)
					if prserr = processpsvrunes(psvlbl[:psvi]...); prserr != nil {
						return
					}
				}
				setprevr(r)
				prserr = processpsvrunes(prvr())
			}
		} else if psvi == psvl && atvi < atvl {
			if cancheckatv() && atvlbl[atvi] == r {
				lbli[1]++
				if (atvi + 1) == atvl {
					flushatv()
					lbli[0] = 0
					lbli[1] = 0
					setprevr(0)
				}
			} else {
				if atvi > 0 {
					lbli[1] = 0
					if failed, prserr = processatvrunes(psvlbl[:psvi]...); prserr != nil {
						return
					} else if failed {
						lbli[0] = 0
						lbli[1] = 0
						setprevr(0)
					}
				}
				if failed, prserr = processatvrunes(r); prserr != nil {
					return
				} else if failed {
					lbli[0] = 0
					lbli[1] = 0
					setprevr(0)
				}
			}
		}
		return
	}

	for (prcsection == 0 || prcsection == 1) && prsngerr == nil {
		if cntelmsfndL > 0 || (cntelmsfndL == 0 && prcsection == 0) {
			cntr, cnts, cnterr := ctntrdr.ReadRune()
			if (cnterr == nil || cnterr == io.EOF) && cnts > 0 {
				if cntntprserr = crntprsrfunc(cntr, cntntprevr, cntntsetprevr, cntntcancheckpsv, cntntprcspsvrunes, cntntflushpsv, cntntcancheckatv, cntntprcsatvrunes, cntntflushatv, cntntlbli, cntntpsvlbl, cntntatvlbl); cntntprserr != nil {
					prsngerr = cntntprserr
					return
				}
			} else {
				if cntntprserr != nil && cntntprserr != io.EOF {
					return
				}
				if cntpsvbuf.Size() > 0 {
					cntntflushpsv()
				} else if cntatvbuf.Size() > 0 {
					cntntflushatv()
				} else {
					prcsection = -1
				}
			}
		} else if prcsection == 1 {
			cder, cdes, cdeerr := cderdr.ReadRune()
			if cdes > 0 && (cdeerr == nil || cdeerr == io.EOF) {
				if cdeprserr = crntprsrfunc(cder, cdeprevr, cdesetprevr, cdecancheckpsv, cdeprcspsvrunes, cdeflushpsv, cdecancheckatv, cdeprcsatvrunes, cdeflushatv, cdelbli, cdepsvlbl, cdeatvlbl); cdeprserr != nil {
					prsngerr = cdeprserr
					return
				}
			} else {
				if cdeerr != nil && cdeerr != io.EOF {
					return
				}
				prcsection = 0
			}
		}
	}
	if prsngerr == nil {
		cdeflushpsv()
		cdeflushatv()
		if evalcode != nil {
			var chdpgrm interface{} = nil
			if codebuf.Size() > 0 {
				var cderead = iorw.NewReplaceRuneReader(codebuf.Clone(true).Reader(true), "$db.", "$.dbms", "$schdl.", "$.scheduling", "$log.", "$.log")
				cderead.WriteTo(codebuf)
				prsngerr = evalcode(codebuf.Reader(), func() interface{} {
					return chdpgrm
				}, func(prgm interface{}) {
					chdpgrm = prgm
				})
			}
			if prsngerr == nil {
				if cancache && prsngerr == nil {
					if fullpath != "" {
						if crntscrpt := GLOBALCACHEDSCRIPTING().Load(pathModified, chdpsvbuf, codebuf, cntelmsfound, func() (scrptpath string) {
							if invertActive {
								return "/active:" + fullpath
							}
							return fullpath
						}()); crntscrpt != nil && chdpgrm != nil {
							crntscrpt.SetScriptProgram(chdpgrm)
						}
					}
				}
			}
		}
	}
	return

}

var validElmRunes = map[rune]int{'A': 1, 'B': 1, 'C': 1, 'D': 1, 'E': 1, 'F': 1, 'G': 1, 'H': 1, 'I': 1, 'J': 1, 'K': 1, 'L': 1, 'M': 1, 'N': 1, 'O': 1, 'P': 1, 'Q': 1, 'R': 1, 'S': 1, 'T': 1, 'U': 1, 'V': 1, 'W': 1, 'X': 1, 'Y': 1, 'Z': 1,
	'a': 1, 'b': 1, 'c': 1, 'd': 1, 'e': 1, 'f': 1, 'g': 1, 'h': 1, 'i': 1, 'j': 1, 'k': 1, 'l': 1, 'm': 1, 'n': 1, 'o': 1, 'p': 1, 'q': 1, 'r': 1, 's': 1, 't': 1, 'u': 1, 'v': 1, 'w': 1, 'x': 1, 'y': 1, 'z': 1,
	'0': 1, '1': 1, '2': 1, '3': 1, '4': 1, '5': 1, '6': 1, '7': 1, '8': 1, '9': 1,
	':': 1, '-': 1, '_': 1, '.': 1}

func validElemChar(prvr, r rune) (valid bool) {
	valid = true
	if prvr > 0 {
		if valid = validElemChar(0, prvr); valid {
			valid = validElmRunes[r] == 1
		}
	} else if valid {
		valid = validElmRunes[r] == 1
	}
	return
}
