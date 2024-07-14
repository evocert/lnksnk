package parsing

import (
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/evocert/lnksnk/fsutils"
	"github.com/evocert/lnksnk/iorw"
)

func validLastCdeRune(cr rune) bool {
	return cr == '=' || cr == '(' || cr == '[' || cr == ',' || cr == '+' || cr == '/'
}

type parsefunc func(r rune, preLen, postLen int, prelbl, postlbl []rune, lbli []int) (prserr error)

type contentelem struct {
	modified     time.Time
	fi           fsutils.FileInfo
	elemname     string
	elemroot     string
	elemext      string
	ctntbuf      *iorw.Buffer
	prebuf       *iorw.Buffer
	postbuf      *iorw.Buffer
	runerdr      io.RuneReader
	rawBuf       *iorw.Buffer
	eofevent     func(*contentelem, error)
	tmplts       map[string]*iorw.Buffer
	mtchphrshndl *MatchPhraseHandler
}

func (ctntelm *contentelem) writeRune(r rune) {
	if ctntelm != nil {
		if ctntelm.rawBuf != nil {
			ctntelm.rawBuf.WriteRune(r)
			return
		}
		ctntelm.content().WriteRune(r)
	}
}

func (ctntelm *contentelem) content() (ctntbuf *iorw.Buffer) {
	if ctntelm != nil {
		if ctntbuf = ctntelm.ctntbuf; ctntbuf == nil {
			ctntbuf = iorw.NewBuffer()
			ctntelm.ctntbuf = ctntbuf
		}
	}
	return
}

// ReadRune implements io.RuneReader.
func (ctntelm *contentelem) ReadRune() (r rune, size int, err error) {
	if ctntelm != nil {
		if ctntelm.runerdr != nil {
			if r, size, err = ctntelm.runerdr.ReadRune(); err != nil {
				if eofevent := ctntelm.eofevent; eofevent != nil {
					ctntelm.eofevent = nil
					if err == io.EOF {
						eofevent(ctntelm, nil)
						return
					}
					eofevent(ctntelm, err)
				}
			}
			return
		}
		prepContentElemReader(ctntelm)
		if ctntelm.runerdr != nil {
			if ctntelm.rawBuf == nil {
				ctntelm.rawBuf = iorw.NewBuffer()
			}
			if r, size, err = ctntelm.runerdr.ReadRune(); err != nil {
				if eofevent := ctntelm.eofevent; eofevent != nil {
					ctntelm.eofevent = nil
					if err == io.EOF {
						eofevent(ctntelm, nil)
						return
					}
					eofevent(ctntelm, err)
				}
			}
		}
	}
	if size == 0 && err == nil {
		err = io.EOF
	}
	return
}

// Close implements io.Closer
func (ctntelm *contentelem) Close() (err error) {
	if ctntelm != nil {
		if ctntelm.postbuf != nil {
			ctntelm.postbuf.Close()
			ctntelm.postbuf = nil
		}

		if ctntelm.prebuf != nil {
			ctntelm.prebuf.Close()
			ctntelm.prebuf = nil
		}

		if ctntelm.ctntbuf != nil {
			ctntelm.ctntbuf.Close()
			ctntelm.ctntbuf = nil
		}

		if ctntelm.runerdr != nil {
			ctntelm.runerdr = nil
		}

		if ctntelm.fi != nil {
			ctntelm.fi = nil
		}

		if ctntelm.rawBuf != nil {
			ctntelm.rawBuf.Close()
			ctntelm.rawBuf = nil
		}

		if ctntelm.eofevent != nil {
			ctntelm.eofevent = nil
		}
		if ctntelm.mtchphrshndl != nil {
			ctntelm.mtchphrshndl.Close()
			ctntelm.mtchphrshndl = nil
		}
	}
	return
}

func prepContentElemReader(ctntelm *contentelem) {
	if ctntelm != nil && ctntelm.runerdr == nil && ctntelm.fi != nil {
		ctntbuf := ctntelm.ctntbuf
		ctntelm.ctntbuf = nil
		var rdr io.RuneReader = nil
		if r, rerr := ctntelm.fi.Open(); rerr == nil {
			if rdr, _ = (r).(io.RuneReader); rdr == nil {
				rdr = iorw.NewEOFCloseSeekReader(r)
			}
		}

		path := ctntelm.fi.Path()
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
		root := pathroot
		if root[0:1] == "/" && root[len(root)-1:] == "/" && root != "/" {
			root = root[:strings.LastIndex(root[:len(root)-1], "/")+1]
		}
		if strings.HasSuffix(ctntelm.elemname, ":") {
			path = ""
		}
		mtchphrshndl := ctntelm.mtchphrshndl
		if mtchphrshndl == nil {
			mtchphrshndl = NewMatchPhraseHandler(iorw.NewReplaceRuneReader(rdr), "<:_:", ":/>", "<:", ":/>", "[:", ":/]")
			ctntelm.mtchphrshndl = mtchphrshndl
		}
		prebuf := ctntelm.prebuf
		if !prebuf.Empty() {
			mtchphrshndl.Match("pre", prebuf.String())
		}
		postbuf := ctntelm.postbuf
		if !postbuf.Empty() {
			mtchphrshndl.Match("post", postbuf.String())
		}
		if !ctntbuf.Empty() {
			if ctntbuf.Contains("[#") {
				rdr := ctntbuf.Clone(true).Reader(true)
				tmpltcntrdr := NewUntilRuneReader(rdr, "#")
				tmpbuf := iorw.NewBuffer()
				for !tmpltcntrdr.Done {
					tmpltcntrdr.WriteTo(tmpbuf)
					if tmpltcntrdr.FoundUntil {
						if !tmpbuf.Empty() {
							tmpbuf.WriteTo(ctntbuf)
							tmpbuf.Clear()
						}
						tmpltcntrdr.NextUntil("#")
						tmpltcntrdr.WriteTo(tmpbuf)
						if tmpltcntrdr.FoundUntil {
							if !tmpbuf.Empty() {
								tmpltnme := tmpbuf.String()
								tmpbuf.Clear()
								tmpltcntrdr.NextUntil("#" + tmpltnme + "#]")
								tmpltcntrdr.WriteTo(tmpbuf)
								if tmpltcntrdr.FoundUntil {
									mtchphrshndl.Match(tmpltnme, tmpbuf.String())
									//tmplts = append(tmplts, "<:_:"+tmpltnme+":/>", tmpbuf.String())
									tmpbuf.Clear()
									tmpltcntrdr.NextUntil("[#")
								} else {
									tmpbuf.Clear()
								}
							}
						}
					}
				}
				if !tmpbuf.Empty() {
					tmpbuf.WriteTo(ctntbuf)
					tmpbuf.Clear()
				}
			}
		}
		mtchphrshndl.Match("pathroot", pathroot)
		mtchphrshndl.Match("root", root)
		mtchphrshndl.Match("elemroot", func() (elmroot string) {
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
		mtchphrshndl.Match("cntnt", ctntbuf)
		mtchphrshndl.FoundPhraseEvent = func(prefix, postfix, phrase string, result interface{}) (nxtrslt interface{}) {
			if fnds, _ := result.(string); fnds != "" {
				nxtrslt = fnds
				return
			}
			if fndbuf, _ := result.(*iorw.Buffer); !fndbuf.Empty() {
				nxtrslt = fndbuf.Clone(true).Reader(true)
				return
			}
			return
		}
		ctntelm.runerdr = mtchphrshndl
		/*bfr := iorw.NewBuffer()
		fmt.Println()
		fmt.Println(ctntelm.elemname)
		bfr.Print(ctntelm.runerdr)
		fmt.Println(bfr.String())
		fmt.Println()
		ctntelm.runerdr = bfr.Reader(true)*/
	}
}

type ctntelemlevel int

func (ctntelmlvl ctntelemlevel) String() string {
	if ctntelmlvl == ctntElemStart {
		return "start"
	}
	if ctntelmlvl == ctntElemSingle {
		return "single"
	}
	if ctntelmlvl == ctntElemEnd {
		return "end"
	}
	return "unknown"
}

func internalProcessParsing(
	capturecache func(fullpath string, pathModified time.Time, cachedpaths map[string]time.Time, prsdpsv, prsdatv *iorw.Buffer, preppedatv interface{}) (cshderr error),
	pathModified time.Time,
	path, pathroot, pathext string,
	out io.Writer,
	fs *fsutils.FSUtils,
	invertActive bool,
	evalcode func(...interface{}) (interface{}, error),
	rnrdrs ...io.RuneReader) (prsngerr error) {
	fullpath := pathroot + path
	validelempaths := map[string]time.Time{}
	invalidelempaths := map[string]bool{}
	var codeEventReader *codeeventreader = nil
	tmpphrasebuf := iorw.NewBuffer()
	root := pathroot
	if root[0:1] == "/" && root[len(root)-1:] == "/" && root != "/" {
		root = root[:strings.LastIndex(root[:len(root)-1], "/")+1]
	}
	tmpmatchthis := map[string]interface{}{}
	tmpmatchthis["pathroot"] = pathroot
	tmpmatchthis["root"] = root

	var elempath = func() (elmroot string) {
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
	}()
	tmpmatchthis["elemroot"] = elempath
	ctntEventReader := newContentEventReader("<", ">", iorw.NewReplaceRuneReader(iorw.NewRuneReaderSlice(rnrdrs...), "<:_:", func(matchedphrase string, rplcrdr *iorw.ReplaceRuneReader) (nxtrdr interface{}) {
		if matchedphrase == "<:_:" {
			rnsuntil := rplcrdr.ReadRunesUntil(":/>")
			tmpphrasebuf.Clear()
			tmpphrasebuf.ReadRunesFrom(rnsuntil)
			if !tmpphrasebuf.Empty() {
				defer tmpphrasebuf.Clear()
				if rplcrdr.FoundEOF() {
					for fndk, fndv := range tmpmatchthis {
						if equals, _ := tmpphrasebuf.Equals(fndk); equals {
							if fnds, _ := fndv.(string); fnds != "" {
								nxtrdr = fnds
							}
							return
						}
					}
				}
			}
			return
		}
		return
	}))
	var crntnextelm *contentelem = nil
	var elemlevels = []*contentelem{}

	ctntEventReader.PreRunesEvent = func(resetlbl bool, rnsl int, rns ...rune) (rnserr error) {
		if crntnextelm != nil {
			if crntnextelm.rawBuf == nil {
				crntnextelm.content().WriteRunes(rns...)
				return
			}
			crntnextelm.rawBuf.WriteRunes(rns...)
			return
		}
		rnserr = codeEventReader.parseRunes(rns...)
		return
	}

	var addelemlevel = func(fi fsutils.FileInfo, elemname string, elemext string) (elmnext *contentelem) {
		elmnext = &contentelem{
			modified: fi.ModTime(),
			fi:       fi,
			elemname: elemname,
			elemroot: elemname[:strings.LastIndex(elemname, ":")+1],
			elemext:  elemext,
		}
		//println(fi.Path())
		validelempaths[fi.Path()] = fi.ModTime()
		elemlevels = append([]*contentelem{elmnext}, elemlevels...)
		return
	}
	ctntEventReader.ValidElemEvent = func(elmlvl ctntelemlevel, elemname string, elmbuf *iorw.Buffer) (evtvalid bool, vlerr error) {
		if fs != nil {
			var fi fsutils.FileInfo = nil

			fullelemname := func() string {
				if elemname[0:1] == ":" {
					return elemname
				}

				if crntnextelm != nil {
					if elmlvl == ctntElemEnd {
						if al := len(elemlevels); al > 0 && elemlevels[0] == crntnextelm && strings.HasSuffix(crntnextelm.elemname, elemname) {
							return crntnextelm.elemname
						}
					}
					return crntnextelm.elemroot + elemname
				}
				return elempath + elemname
			}()
			if invalidelempaths[fullelemname] {
				return
			}
			if elmlvl == ctntElemStart || elmlvl == ctntElemSingle {
				testpath := strings.Replace(fullelemname, ":", "/", -1)
				testext := filepath.Ext(testpath)

				if fi = func() fsutils.FileInfo {
					if fs == nil {
						return nil
					}
					if testext == "" {
						testext = pathext
					}
					if fullelemname[len(fullelemname)-1] == ':' {
						for _, nextpth := range []string{testext, ".js"} {
							if nextpth != "" && nextpth[0:1] == "." {
								if fios := fs.LS(testpath + "index" + nextpth); len(fios) == 1 {
									return fios[0]
								}
							}
						}
					}
					if fios := fs.LS(testpath + testext); len(fios) == 1 {
						return fios[0]
					}
					return nil
				}(); fi == nil {
					invalidelempaths[fullelemname] = true
					return
				}
				evtvalid = true
				crntnextelm = addelemlevel(fi, fullelemname, fi.PathExt())
				if !elmbuf.Empty() {
					crntnextelm.prebuf = elmbuf.Clone(true)
				}
				if elmlvl == ctntElemSingle {
					crntnextelm.eofevent = func(crntelm *contentelem, elmerr error) {
						if elmerr == nil {
							if !crntelm.rawBuf.Empty() {
								ctntEventReader.PreAppend(crntelm.rawBuf.Clone(true).Reader(true))
							}
							crntelm.Close()
							crntnextelm = nil
							if elemlvlL := len(elemlevels); elemlvlL > 0 {
								elemlevels = elemlevels[1:]
								if elemlvlL > 1 {
									crntnextelm = elemlevels[0]
									return
								}
							}
							return
						}
						vlerr = elmerr
					}
					ctntEventReader.PreAppend(crntnextelm)
					return
				}
				return
			}
			if elmlvl == ctntElemEnd {
				if crntnextelm != nil && crntnextelm.elemname == fullelemname {
					evtvalid = true
					if !elmbuf.Empty() {
						crntnextelm.postbuf = elmbuf.Clone(true)
					}
					crntnextelm.eofevent = func(crntelm *contentelem, elmerr error) {
						if elmerr == nil {
							if !crntelm.rawBuf.Empty() {
								ctntEventReader.PreAppend(crntnextelm.rawBuf.Clone(true).Reader(true))
							}
							crntelm.Close()
							crntnextelm = nil
							if elemlvlL := len(elemlevels); elemlvlL > 0 {
								elemlevels = elemlevels[1:]
								if elemlvlL > 1 {
									crntnextelm = elemlevels[0]
									return
								}
							}
							return
						}
						vlerr = elmerr
					}
					ctntEventReader.PreAppend(crntnextelm)
					return
				}
			}
		}
		return

	}

	codeEventReader = newCodeEventReader("<@", "@>", ctntEventReader)
	cdebuf := iorw.NewBuffer()
	defer cdebuf.Close()
	codeEventReader.AddCommentsEventReader(true, "//", "\n", true, "/*", "*/", func(imprtbuf *iorw.Buffer, prelbl, postlbl string) (poserr error) {
		if !imprtbuf.Empty() {
			imports := imprtbuf.String()
			importsl := len(imports)
			if strings.Index(imports, "\"") == 0 && strings.LastIndex(imports, "\"") == importsl-1 {
				cdebuf.Println("require(", imports, ");")
				return
			}
			if imprts := strings.Split(imports, "from"); len(imprts) == 2 {
				if modname := strings.TrimFunc(imprts[1], iorw.IsSpace); modname != "" {
					if imprtlne := strings.TrimFunc(imprts[0], iorw.IsSpace); imprtlne != "" {
						lstdfltimprt := "_defltrqr"
						cdebuf.Println("var _defltrqr=require(", modname, ");")
						defltimprts := []string{}
						nmedimprts := map[string][]string{}
						for {
							if imprtlne := strings.TrimFunc(imprtlne, iorw.IsSpace); imprtlne != "" {
								brsi, cmai := strings.Index(imprtlne, "{"), strings.Index(imprtlne, ",")
								if brsi > -1 && (cmai == -1 || cmai > brsi) {
									if brsei := strings.Index(imprtlne, "}"); brsei > -1 {
										if brsei > brsi {
											if cmai == -1 || brsi < cmai {
												if nmprts := strings.Split(strings.TrimFunc(imprtlne[brsi+1:brsei-brsi], iorw.IsSpace), ","); len(nmprts) > 0 {
													for _, nmprt := range nmprts {
														if nmprt = strings.TrimFunc(nmprt, iorw.IsSpace); nmprt != "" {
															if len(nmedimprts[lstdfltimprt]) == 0 {
																nmedimprts[lstdfltimprt] = []string{nmprt}
																continue
															}
															nmedimprts[lstdfltimprt] = append(nmedimprts[lstdfltimprt], nmprt)
														}
													}
												}
											}
										}
										if imprtlne = imprtlne[brsei+1:]; imprtlne == "" {
											break
										}
										continue
									}
								}
								if cmai > -1 {
									if cmaimprt := strings.TrimFunc(imprtlne[:cmai+1], iorw.IsSpace); cmaimprt != "" {
										defltimprts = append(defltimprts, cmaimprt)
										lstdfltimprt = cmaimprt
									}
									imprtlne = imprtlne[cmai+1:]
									continue
								}
								defltimprts = append(defltimprts, imprtlne)
								imprtlne = ""
								break
							}
							break
						}
						lstdfltimprt = "_defltrqr"
						if len(nmedimprts[lstdfltimprt]) > 0 {
							defltimprts = append([]string{lstdfltimprt}, defltimprts...)
						}
						for dfltn, dfltport := range defltimprts {
							if dltprts := strings.Split(dfltport, "as "); len(dltprts) == 2 {
								dltprts[0], dltprts[1] = strings.TrimFunc(dltprts[0], iorw.IsSpace), strings.TrimFunc(dltprts[1], iorw.IsSpace)
								if dltprts[1] == "" {
									continue
								}
								if dltprts[0] == "*" {
									cdebuf.Println("var ", dltprts[1], "=", lstdfltimprt, ";")
								} else {
									cdebuf.Println("var ", dltprts[0], "=", lstdfltimprt, ";")
									cdebuf.Println("var ", dltprts[1], "=", dltprts[0], ";")
								}
								dfltport = dltprts[1]
							} else if dfltport != "_defltrqr" {
								cdebuf.Println("var ", dfltport, "=", lstdfltimprt, ";")
							}
							for _, nmdprt := range nmedimprts[defltimprts[dfltn]] {
								if strings.Contains(nmdprt, "as ") {
									if nmdprts := strings.Split(nmdprt, "as "); len(nmdprts) == 2 {
										if nmdprts[0], nmdprts[1] = strings.TrimFunc(nmdprts[0], iorw.IsSpace), strings.TrimFunc(nmdprts[1], iorw.IsSpace); nmdprts[1] != "" && nmdprts[0] != "" {
											if nmdprtsl := len(nmdprts[0]); nmdprts[0][0] == '"' && nmdprts[0][nmdprtsl-1] == '"' {
												if nmdprtsl > 2 {
													cdebuf.Println("var ", nmdprts[1], "=", dfltport, "[", nmdprts[0], "];")
												}
												continue
											}
											cdebuf.Println("var ", nmdprts[1], "=", dfltport, ".", nmdprts[0], ";")
										}
									}
									continue
								}
								cdebuf.Println("var ", nmdprt, "=", dfltport, ".", nmdprt, ";")
							}
						}
					}
				}
			}
		}
		return
	}, "import", ";")
	//ctntbuf := iorw.NewBuffer()
	ctntbuf := iorw.NewBuffer()
	defer ctntbuf.Close()
	chdctntbuf := iorw.NewBuffer()
	cdelstr := rune(0)

	ctntflush := func() (flsherr error) {
		if cdepsvs := ctntbuf.Size(); cdepsvs > 0 {
			defer ctntbuf.Clear()
			hstmpltfx := ctntbuf.HasPrefix("`") && ctntbuf.HasSuffix("`") && cdepsvs >= 2
			cntsinlinebraseortmpl := !hstmpltfx && ctntbuf.Contains("${") || ctntbuf.Contains("`")
			var psvrdr io.RuneReader = func() io.RuneReader {
				if hstmpltfx {
					return ctntbuf.Clone(true).Reader(true)
				}
				if cntsinlinebraseortmpl {
					return iorw.NewReplaceRuneReader(ctntbuf.Clone(true).Reader(true), "`", "\\`", "${", "\\${")
				}
				return iorw.NewReplaceRuneReader(ctntbuf.Clone(true).Reader(true), `"\`, `"\\`)
			}()

			if cdelstr > 0 {
				cdelstr = 0
				if hstmpltfx {
					cdebuf.Print(psvrdr)
					return
				}
				if cntsinlinebraseortmpl {
					cdebuf.Print("`", psvrdr, "`")
					return
				}
				cdebuf.Print("`", psvrdr, "`")
				return
			}
			if hstmpltfx {
				cdebuf.Print("print(", psvrdr, ");")
				return
			}
			if cntsinlinebraseortmpl {
				cdebuf.Print("print(`", psvrdr, "`);")
				return
			}
			cdebuf.Print("print(`", psvrdr, "`);")
		}
		return
	}
	codeEventReader.CodePreRunesEvent = func(foundcode bool, rnsl int, rns ...rune) (prerr error) {
		if foundcode {
			ctntbuf.WriteRunes(rns...)
			return
		}
		chdctntbuf.WriteRunes(rns...)
		return
	}

	codeEventReader.CodeFoundEvent = func(foundcode bool) (fnderr error) {
		if foundcode {
			fnderr = ctntflush()
			return
		}
		return
	}

	codeEventReader.CodePreResetEvent = func(foundcode bool, prel, postl int, prelbl, postlbl []rune, lbli []int) (rseterr error) {
		if foundcode {
			ctntflush()
		}
		return
	}
	codeEventReader.CodePostRunesEvent = func(rnsl int, rns ...rune) (preerr error) {
		if codeEventReader.PostTxtr == 0 {
			cdelstr = 0
			for rn := range rns {
				lsrn := rnsl - (rn + 1)
				if !iorw.IsSpace(rns[lsrn]) {
					cdelstr = func() rune {
						if validLastCdeRune(rns[lsrn]) {
							return rns[lsrn]
						}
						return 0
					}()
					break
				}
			}
			cdebuf.WriteRunes(rns...)
			return
		}
		if cdelstr > 0 {
			cdelstr = 0
		}
		cdebuf.WriteRunes(rns...)
		return
	}
	if invertActive {
		codeEventReader.SwapParseState()
	}
	if prsngerr = codeEventReader.DummyEOFRead(); prsngerr != nil {
		return
	}
	ctntflush()
	var chdpgrm interface{} = nil
	if !chdctntbuf.Empty() && cdebuf.Empty() {
		DefaultMinifyPsv(pathext, chdctntbuf, nil)
		if capturecache != nil {
			prsngerr = capturecache(fullpath, pathModified, validelempaths, chdctntbuf, cdebuf, chdpgrm)
		}
	}
	if !chdctntbuf.Empty() {
		if out != nil {
			if _, prsngerr = chdctntbuf.WriteTo(out); prsngerr != nil {
				return
			}
		}
	}

	if !cdebuf.Empty() {
		if DefaultMinifyCde != nil {
			DefaultMinifyCde(".js", cdebuf, nil)
		}
		if evalcode != nil {
			if _, prsngerr = evalcode(cdebuf.Reader(), func(prgm interface{}, prsccdeerr error, cmpleerr error) {
				if cmpleerr == nil && prsccdeerr == nil {
					chdpgrm = prgm
				}
				if prsccdeerr != nil {
					prsngerr = prsccdeerr
				}
				if cmpleerr != nil {
					prsngerr = cmpleerr
				}
				if prsngerr == nil {
					if capturecache != nil {
						prsngerr = capturecache(fullpath, pathModified, validelempaths, chdctntbuf, cdebuf, chdpgrm)
					}
				}
			}); prsngerr != nil {
				println(prsngerr.Error())
				println()
				println(cdebuf.String())
			}
		}
	}
	return
}

func internalProcessParsingv0(
	capturecache func(fullpath string, pathModified time.Time, cachedpaths map[string]time.Time, prsdpsv, prsdatv *iorw.Buffer, preppedatv interface{}) (cshderr error),
	pathModified time.Time,
	path, pathroot, pathext string,
	out io.Writer,
	fs *fsutils.FSUtils,
	invertActive bool,
	evalcode func(...interface{}) (interface{}, error),
	rnrdrs ...io.RuneReader) (prsngerr error) {
	fullpath := pathroot + path
	invalidelempaths := map[string]bool{}
	validelempaths := map[string]time.Time{}
	var chdpsvbuf *iorw.Buffer = nil

	cdelblrns := [][]rune{[]rune("<@"), []rune("@>")}
	cdepreL := len(cdelblrns[0])
	cdepostL := len(cdelblrns[1])
	cdelbli := []int{0, 0}
	cdetxtr := rune(0)
	cdeprvr := rune(0)
	cdelstr := rune(0)
	if invertActive {
		cdelbli[0] = cdepreL
	}
	cdebuf := iorw.NewBuffer()
	defer cdebuf.Close()

	cdeatvbuf := iorw.NewBuffer()
	defer cdeatvbuf.Close()

	cmdslcted := ""
	var cdecmdlbl []rune = nil
	cdecmdi := 0
	var cdecrntbuf *iorw.Buffer = cdeatvbuf

	cdeimprtbuf := iorw.NewBuffer()
	defer cdeimprtbuf.Close()

	cdeflushatv := func() {
		if !cdeatvbuf.Empty() {
			cdeatvbuf.WriteTo(cdebuf)
			cdeatvbuf.Clear()
		}
	}

	var crntnextelm *contentelem = nil

	cdeatvparse := func(notxt bool, atvr rune) {
		if len(cdecmdlbl) == 0 {
			if notxt {
				cdetxtr = 0
				cdelstr = 0
				cdecrntbuf.WriteRune(atvr)
				cdeprvr = atvr
				return
			}
			if cdeprvr != '\\' && iorw.IsTxtPar(atvr) {
				cdetxtr = atvr
				cdelstr = 0
				cdecrntbuf.WriteRune(atvr)
				cdeprvr = atvr
				return
			}
			if !iorw.IsSpace(atvr) {
				cdelstr = func() rune {
					if validLastCdeRune(atvr) {
						return atvr
					}
					return 0
				}()
			}

			if iorw.IsSpace(atvr) {
				if cmdslcted != "" {
					cmdslcted = ""
				}
				cdecrntbuf.WriteRune(atvr)
				cdeprvr = atvr
				return
			}
			cmdslcted += string(atvr)
			if cmdslcted == "//" {
				cdecmdlbl = []rune("\n")
				cdecrntbuf.WriteRune(atvr)
				cdeprvr = atvr
				cmdslcted = ""
				return
			}
			if cmdslcted == "/*" {
				cdecmdlbl = []rune("\n")
				cdecrntbuf.WriteRune(atvr)
				cdeprvr = atvr
				cmdslcted = ""
				return
			}
			if cmdslcted == "import"[:len(cmdslcted)] {
				cdecrntbuf = cdeimprtbuf
				if cmdslcted == "import" {
					cdecmdlbl = []rune(";")
					cmdslcted = ""
					cdeprvr = atvr
					cdeimprtbuf.Clear()
					return
				}
				cdeprvr = atvr
				cdecrntbuf.WriteRune(atvr)
				return
			}
			if !cdeimprtbuf.Empty() {
				cmdslcted = ""
				cdecrntbuf = cdeatvbuf
				cdeimprtbuf.WriteTo(cdecrntbuf)
				cdeimprtbuf.Clear()
				cdecrntbuf.WriteRune(atvr)
				cdeprvr = atvr
				return
			}
			if len(cmdslcted) > 2 {
				cmdslcted = ""
			}
			cdecrntbuf.WriteRune(atvr)
			cdeprvr = atvr
			return
		}
		if cdecmdlbl[cdecmdi] == atvr {
			cdecmdi++
			if cdecmdi == len(cdecmdlbl) {
				cmdslcted = ""
				cdecmdi = 0
				if string(cdecmdlbl) == ";" {
					cdecmdlbl = nil
					if !cdeimprtbuf.Empty() {
						func() {
							cdecrntbuf = cdeatvbuf
							defer cdeimprtbuf.Clear()
							if tmpimports := cdeimprtbuf.String(); tmpimports != "" {
								cdeimprtbuf.Clear()
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
								cdecrntbuf.Print(imprtcde)
							}
						}()
						cdeprvr = atvr
						return
					}
				}
				cdecmdlbl = nil
				cdecrntbuf.WriteRune(atvr)
				cdeprvr = atvr
				return
			}
			cdecrntbuf.WriteRune(atvr)
			cdeprvr = atvr
		}
		if cdecmdi > 0 {
			for _, cdavr := range cdecmdlbl[:cdecmdi] {
				cdecrntbuf.WriteRune(cdavr)
				cdeprvr = cdavr
			}
			cdecmdi = 0
		}
		cdecrntbuf.WriteRune(atvr)
		cdeprvr = atvr
	}

	cdepsvbuf := iorw.NewBuffer()
	defer cdepsvbuf.Close()

	cdeflushpsv := func() (flsherr error) {
		if cdepsvs := cdepsvbuf.Size(); cdepsvs > 0 {
			hstmpltfx := cdepsvbuf.HasPrefix("`") && cdepsvbuf.HasSuffix("`") && cdepsvs >= 2
			if !hstmpltfx && cdebuf.Size() == 0 {
				if out != nil {
					_, flsherr = cdepsvbuf.WriteTo(out)
				}
				if capturecache != nil && flsherr == nil {
					if chdpsvbuf == nil {
						chdpsvbuf = iorw.NewBuffer()
					}
					cdepsvbuf.WriteTo(chdpsvbuf)
				}
				cdepsvbuf.Clear()
				return
			}
			cntsinlinebraseortmpl := !hstmpltfx && cdepsvbuf.Contains("${") || cdepsvbuf.Contains("`")
			var psvrdr io.RuneReader = func() io.RuneReader {
				if hstmpltfx {
					return cdepsvbuf.Clone(true).Reader(true)
				}
				if cntsinlinebraseortmpl {
					return iorw.NewReplaceRuneReader(cdepsvbuf.Clone(true).Reader(true), "`", "\\`", "${", "\\${")
				}
				return iorw.NewReplaceRuneReader(cdepsvbuf.Clone(true).Reader(true), `"\`, `"\\`)
			}()

			if cdelstr > 0 {
				cdelstr = 0
				if hstmpltfx {
					cdebuf.Print(psvrdr)
					return
				}
				if cntsinlinebraseortmpl {
					cdebuf.Print("`", psvrdr, "`")
					return
				}
				cdebuf.Print("`", psvrdr, "`")
				return
			}
			if hstmpltfx {
				cdebuf.Print("print(", psvrdr, ");")
				return
			}
			if cntsinlinebraseortmpl {
				cdebuf.Print("print(`", psvrdr, "`);")
				return
			}
			cdebuf.Print("print(`", psvrdr, "`);")
		}
		return
	}

	cdepsvparse := func(ptvr rune) {
		cdeflushatv()
		cdepsvbuf.WriteRune(ptvr)
	}

	var cdeparser parsefunc = nil
	cdeparser = func(r rune, preLen, postLen int, prelbl, postlbl []rune, lbli []int) (prserr error) {
		if cdetxtr == 0 {
			if lbli[1] == 0 && lbli[0] < preLen {
				if lbli[0] > 0 && cdeprvr == prelbl[lbli[0]-1] && r != prelbl[lbli[0]] {
					for _, cder := range prelbl[:lbli[0]] {
						cdepsvparse(cder)
						cdeprvr = cder
					}
					lbli[0] = 0
					cdeprvr = 0
					return cdeparser(r, preLen, postLen, prelbl, postlbl, lbli)
				}
				if prelbl[lbli[0]] == r {
					lbli[0]++
					if lbli[0] == preLen {

						cdeprvr = 0
						cdeflushpsv()
						return
					}
					cdeprvr = r
					return
				}
				if lbli[0] > 0 {
					for _, cder := range prelbl[:lbli[0]] {
						cdepsvparse(cder)
						cdeprvr = cder
					}
					lbli[0] = 0
					cdeprvr = 0
				}
				cdepsvparse(r)
				cdeprvr = r
				return
			}
			if lbli[0] == preLen && lbli[1] < postLen {
				if postlbl[lbli[1]] == r {
					lbli[1]++
					if lbli[1] == postLen {

						lbli[0] = 0
						lbli[1] = 0
						cdeprvr = 0
						cdeflushatv()
						return
					}
					return
				}
				if lbli[1] > 0 {
					for _, atvr := range prelbl[:lbli[1]] {
						cdeatvparse(false, atvr)
					}
					lbli[1] = 0
				}
				cdeatvparse(false, r)
				return
			}
		}
		if len(cdecmdlbl) == 0 {
			if cdeprvr != '\\' && cdetxtr == r {
				cdeatvparse(cdetxtr == r, r)
				return
			}
			cdeatvparse(false, r)
			return
		}
		if lbli[0] == preLen {
			cdeatvparse(false, r)
			return
		}
		if lbli[1] == postLen {
			cdepsvparse(r)
		}
		return
	}

	root := pathroot
	if root[0:1] == "/" && root[len(root)-1:] == "/" && root != "/" {
		root = root[:strings.LastIndex(root[:len(root)-1], "/")+1]
	}
	tmpphrasebuf := iorw.NewBuffer()
	tmpmatchthis := map[string]interface{}{}
	tmpmatchthis["pathroot"] = pathroot
	tmpmatchthis["root"] = root
	tmpmatchthis["elemroot"] = func() (elmroot string) {
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
	}()
	var mainrunerdr io.RuneReader = iorw.NewReplaceRuneReader(iorw.NewRuneReaderSlice(rnrdrs...), "<:_:", func(matchedphrase string, rplcrdr *iorw.ReplaceRuneReader) (nxtrdr interface{}) {
		if matchedphrase == "<:_:" {
			rnsuntil := iorw.RunesReaderUntil(rplcrdr, ":/>")
			tmpphrasebuf.Clear()
			tmpphrasebuf.ReadRunesFrom(rnsuntil)
			rmngrns := rnsuntil.RemainingRunes()
			if !tmpphrasebuf.Empty() {
				defer tmpphrasebuf.Clear()
				if rnsuntil.FoundEOF() {
					for fndk, fndv := range tmpmatchthis {
						if equals, _ := tmpphrasebuf.Equals(fndk); equals {
							if fnds, _ := fndv.(string); fnds != "" {
								rmngrns = append([]rune(fnds), rmngrns...)
							}
							break
						}
					}
				}
			}
			if len(rmngrns) > 0 {
				return rmngrns
			}
			return
		}
		return
	})

	ctntrdr := iorw.NewRuneReaderSlice(mainrunerdr)
	defer ctntrdr.Close()
	dneprsng := false
	ctntlblrns := [][]rune{[]rune("<"), []rune(">")}
	ctntpreL := len(ctntlblrns[0])
	ctntpostL := len(ctntlblrns[1])
	ctntlbli := []int{0, 0}
	ctntprvr := rune(0)
	ctnttxtr := rune(0)
	ctntelmname := []rune{}
	ctntelmlvl := ctntElemUnknown
	ctntfndname := false

	ctntrmngbuf := iorw.NewBuffer()
	defer ctntrmngbuf.Close()

	flushctntrmng := func() (prserr error) {
		if !ctntrmngbuf.Empty() {
			if crntnextelm != nil {
				if crntnextelm.rawBuf == nil {
					ctntrmngbuf.WriteTo(crntnextelm.content())
					ctntrmngbuf.Clear()
					return
				}
				ctntrmngbuf.WriteTo(crntnextelm.rawBuf)
				ctntrmngbuf.Clear()
				return
			}

			for _, cr := range func() (rns []rune) {
				rns = []rune(ctntrmngbuf.String())
				ctntrmngbuf.Clear()
				return
			}() {
				if prserr = cdeparser(cr, cdepreL, cdepostL, cdelblrns[0], cdelblrns[1], cdelbli); prsngerr != nil {
					return
				}
			}
		}
		return
	}

	var ctntprspsv func(rune) error = nil

	defer func() {
		if crntnextelm != nil {
			crntnextelm.Close()
			crntnextelm = nil
		}
	}()

	ctntprspsv = func(r rune) (prserr error) {
		if crntnextelm != nil {
			crntnextelm.writeRune(r)
			return
		}
		prserr = cdeparser(r, cdepreL, cdepostL, cdelblrns[0], cdelblrns[1], cdelbli)
		return
	}

	ctntprsvatvbuf := iorw.NewBuffer()

	ctntflushInvalid := func(canprse bool, prelbl, postlbl []rune, lbli []int, r ...rune) (flsrmng []rune) {
		if ctntelmlvl == ctntElemStart {
			ctntelmlvl = ctntElemUnknown
		}
		if canprse {
			if ctntfndname {
				ctntfndname = false
				if lbli[0] > 0 {
					flsrmng = append(flsrmng, prelbl[:lbli[0]]...)
					lbli[0] = 0
				}
				if ctntelmlvl == ctntElemEnd {
					ctntelmlvl = ctntElemUnknown
					flsrmng = append(flsrmng, '/')
				}
				if len(ctntelmname) > 0 {
					flsrmng = append(flsrmng, ctntelmname...)
					ctntelmname = nil
				}
				if !ctntprsvatvbuf.Empty() {
					rdr := ctntprsvatvbuf.Clone(true).Reader(true)
					for {
						if rr, rrs, rrerr := rdr.ReadRune(); rrs > 0 && (rrerr == nil || rrerr == io.EOF) {
							flsrmng = append(flsrmng, rr)
							if rrerr == nil {
								continue
							}
						}
						break
					}
				}
				if ctntelmlvl == ctntElemSingle {
					ctntelmlvl = ctntElemUnknown
					flsrmng = append(flsrmng, '/')
				}
				if lbli[1] > 0 {
					flsrmng = append(flsrmng, postlbl[:lbli[1]]...)
					lbli[1] = 0
				}
				return
			}
			if lbli[0] > 0 {
				flsrmng = append(flsrmng, prelbl[:lbli[0]]...)
				lbli[0] = 0
			}
			if ctntelmlvl == ctntElemEnd {
				ctntelmlvl = ctntElemUnknown
				flsrmng = append(flsrmng, '/')
			}
			if len(ctntelmname) > 0 {
				flsrmng = append(flsrmng, ctntelmname...)
				ctntelmname = nil
			}
			flsrmng = append(flsrmng, r...)
			if lbli[1] > 0 {
				flsrmng = append(flsrmng, postlbl[:lbli[1]]...)
				lbli[1] = 0
			}
			return
		}
		lbli[0] = 0
		lbli[1] = 0
		ctntprvr = 0
		ctntfndname = false
		ctntelmlvl = ctntElemUnknown
		ctntelmname = nil
		ctntprsvatvbuf.Clear()
		return
	}
	var ctntprsatv func(prvr, r rune, prelbl, postlbl []rune, lbli []int) (invld bool) = nil
	ctntprsatv = func(prvr, r rune, prelbl, postlbl []rune, lbli []int) (invld bool) {
		if ctntfndname {
			if ctnttxtr == 0 {
				if r == '/' {
					if ctntelmlvl != ctntElemSingle {
						ctntelmlvl = ctntElemSingle
						return
					}
					invld = true
					ctntrmngbuf.WriteRunes(ctntflushInvalid(true, prelbl, postlbl, lbli, r)...)
					return
				}
				if iorw.IsSpace(r) {
					if ctntelmlvl == ctntElemSingle {
						invld = true
						ctntrmngbuf.WriteRunes(ctntflushInvalid(true, prelbl, postlbl, lbli, r)...)
						return
					}
				}
				if prvr != '\\' && iorw.IsTxtPar(r) {
					ctnttxtr = r
				}
			}
			ctntprsvatvbuf.WriteRune(r)
			return
		}
		if r == '/' {
			if ctntelmlvl != ctntElemEnd {
				if len(ctntelmname) == 0 {
					ctntelmlvl = ctntElemEnd
					ctntprvr = 0
					return
				}
				ctntfndname = true
				invld = ctntprsatv(prvr, r, prelbl, postlbl, lbli)
				return
			}
			invld = true
			ctntrmngbuf.WriteRunes(ctntflushInvalid(true, prelbl, postlbl, lbli, r)...)
			return
		}
		if iorw.IsSpace(r) {
			if invld = len(ctntelmname) == 0; invld {
				ctntrmngbuf.WriteRunes(ctntflushInvalid(true, prelbl, postlbl, lbli, r)...)
				return
			}
			if ctntelmlvl != ctntElemEnd {
				ctntfndname = true
				ctntprsvatvbuf.WriteRune(r)
				return
			}
			invld = true
			ctntrmngbuf.WriteRunes(ctntflushInvalid(true, prelbl, postlbl, lbli, r)...)
			return
		}

		if invld = !validElemChar(func() rune {
			if ctntelmlvl == ctntElemEnd && prvr == '/' {
				return 0
			}
			return prvr
		}(), r); !invld {
			ctntelmname = append(ctntelmname, r)
			return
		}
		ctntrmngbuf.WriteRunes(ctntflushInvalid(true, prelbl, postlbl, lbli, r)...)
		return
	}

	var elempath = strings.Replace(pathroot, "/", ":", -1)
	var elemlevels = []*contentelem{}
	defer func() {
		for len(elemlevels) > 0 {
			elemlevels[0].Close()
			elemlevels = elemlevels[1:]
		}
	}()

	var addelemlevel = func(fi fsutils.FileInfo, elemname string, elemext string) (elmnext *contentelem) {
		elmnext = &contentelem{
			modified: fi.ModTime(),
			fi:       fi,
			elemname: elemname,
			elemroot: elemname[:strings.LastIndex(elemname, ":")+1],
			elemext:  elemext,
		}
		//println(fi.Path())
		validelempaths[fi.Path()] = fi.ModTime()
		elemlevels = append([]*contentelem{elmnext}, elemlevels...)
		return
	}

	var ctntparser parsefunc = nil

	ctntparser = func(r rune, preLen, postLen int, prelbl, postlbl []rune, lbli []int) (prserr error) {
		if ctnttxtr == 0 {
			if lbli[1] == 0 && lbli[0] < preLen {
				if lbli[0] > 1 && ctntprvr == prelbl[lbli[0]-1] && r != prelbl[lbli[0]] {
					for _, cder := range prelbl[:lbli[0]] {
						if prserr = ctntprspsv(cder); prserr != nil {
							return
						}

					}
					lbli[0] = 0
					ctntprvr = 0
					return ctntparser(r, preLen, postLen, prelbl, postlbl, lbli)
				}
				if prelbl[lbli[0]] == r {
					lbli[0]++
					if lbli[0] == preLen {
						ctntprvr = 0
						return
					}
					ctntprvr = r
					return
				}
				if lbli[0] > 0 {
					for _, cder := range prelbl[:lbli[0]] {
						if prserr = ctntprspsv(cder); prserr != nil {
							return
						}
					}
					lbli[0] = 0
				}
				ctntprvr = r
				prserr = ctntprspsv(ctntprvr)
				return
			}
			if lbli[0] == preLen && lbli[1] < postLen {
				if postlbl[lbli[1]] == r {
					lbli[1]++
					if lbli[1] == postLen {
						if ctntfndname, elemname, ctntelmlvl := ctntfndname || len(ctntelmname) > 0, string(ctntelmname), func() ctntelemlevel {
							if ctntelmlvl == ctntElemUnknown {
								return ctntElemStart
							}
							return ctntelmlvl
						}(); ctntfndname && !strings.HasPrefix(elemname, ":_:") {
							if fs != nil {
								var fi fsutils.FileInfo = nil

								fullelemname := func() string {
									if elemname[0:1] == ":" {
										return elemname
									}

									if crntnextelm != nil {
										if ctntelmlvl == ctntElemEnd {
											if al := len(elemlevels); al > 0 && elemlevels[0] == crntnextelm && strings.HasSuffix(crntnextelm.elemname, elemname) {
												return crntnextelm.elemname
											}
										}
										return crntnextelm.elemroot + elemname
									}
									return elempath + elemname
								}()
								if invalidelempaths[fullelemname] {
									ctntrmngbuf.WriteRunes(ctntflushInvalid(true, prelbl, postlbl, lbli)...)
									ctntprvr = 0
									prserr = flushctntrmng()
									return
								}
								if ctntelmlvl == ctntElemStart || ctntelmlvl == ctntElemSingle {
									testpath := strings.Replace(fullelemname, ":", "/", -1)
									testext := filepath.Ext(testpath)

									if fi = func() fsutils.FileInfo {
										if fs == nil {
											return nil
										}
										if testext == "" {
											testext = pathext
										}
										if fullelemname[len(fullelemname)-1] == ':' {
											for _, nextpth := range []string{testext, ".js"} {
												if nextpth != "" && nextpth[0:1] == "." {
													if fios := fs.LS(testpath + "index" + nextpth); len(fios) == 1 {
														return fios[0]
													}
												}
											}
										}
										if fios := fs.LS(testpath + testext); len(fios) == 1 {
											return fios[0]
										}
										return nil
									}(); fi == nil {
										invalidelempaths[fullelemname] = true
										ctntrmngbuf.WriteRunes(ctntflushInvalid(true, prelbl, postlbl, lbli)...)
										ctntprvr = 0
										prserr = flushctntrmng()
										return
									}
									crntnextelm = addelemlevel(fi, fullelemname, fi.PathExt())
									if !ctntprsvatvbuf.Empty() {
										crntnextelm.prebuf = ctntprsvatvbuf.Clone(true)
									}
									ctntflushInvalid(false, prelbl, postlbl, lbli)
									if ctntelmlvl.String() == "single" {
										crntnextelm.eofevent = func(crntelm *contentelem, elmerr error) {
											if elmerr == nil {
												if !crntelm.rawBuf.Empty() {
													ctntrdr.PreAppend(crntelm.rawBuf.Clone(true).Reader(true))
												}
												crntelm.Close()
												crntnextelm = nil
												if elemlvlL := len(elemlevels); elemlvlL > 0 {
													elemlevels = elemlevels[1:]
													if elemlvlL > 1 {
														crntnextelm = elemlevels[0]
														return
													}
												}
												return
											}
											prserr = elmerr
										}
										ctntrdr.PreAppend(crntnextelm)
										return
									}
									return
								}
								if ctntelmlvl.String() == "end" {
									if crntnextelm != nil && crntnextelm.elemname == fullelemname {
										if !ctntprsvatvbuf.Empty() {
											crntnextelm.postbuf = ctntprsvatvbuf.Clone(true)
										}
										ctntflushInvalid(false, prelbl, postlbl, lbli)
										crntnextelm.eofevent = func(crntelm *contentelem, elmerr error) {
											if elmerr == nil {
												if !crntelm.rawBuf.Empty() {
													ctntrdr.PreAppend(crntnextelm.rawBuf.Clone(true).Reader(true))
												}
												crntelm.Close()
												crntnextelm = nil
												if elemlvlL := len(elemlevels); elemlvlL > 0 {
													elemlevels = elemlevels[1:]
													if elemlvlL > 1 {
														crntnextelm = elemlevels[0]
														return
													}
												}
												return
											}
											prserr = elmerr
										}
										ctntrdr.PreAppend(crntnextelm)
										return
									}
								}
								ctntrmngbuf.WriteRunes(ctntflushInvalid(true, prelbl, postlbl, lbli)...)
								ctntprvr = 0
								prserr = flushctntrmng()
								return
							}
							ctntrmngbuf.WriteRunes(ctntflushInvalid(true, prelbl, postlbl, lbli)...)
							ctntprvr = 0
							prserr = flushctntrmng()
							return
						}
						ctntrmngbuf.WriteRunes(ctntflushInvalid(true, prelbl, postlbl, lbli)...)
						ctntprvr = 0
						prserr = flushctntrmng()
						return
					}
					return
				}
				ctntprsinvld := false
				if lbli[1] > 0 {
					for _, atvr := range prelbl[:lbli[1]] {
						if ctntprsinvld {
							ctntrmngbuf.WriteRune(atvr)
							continue
						}
						ctntprsinvld = ctntprsatv(ctntprvr, atvr, prelbl, postlbl, lbli)
						ctntprvr = atvr
					}
					lbli[1] = 0
					if ctntprsinvld {
						ctntrmngbuf.WriteRune(r)
						ctntprvr = 0
						prserr = flushctntrmng()
						return
					}
				}
				if ctntprsinvld = ctntprsatv(ctntprvr, r, prelbl, postlbl, lbli); ctntprsinvld {
					ctntprvr = 0
					prserr = flushctntrmng()
					return
				}
				ctntprvr = r
				return
			}
		}
		if ctnttxtr > 0 {
			if ctntprvr != '\\' && ctnttxtr == r {
				ctntprsvatvbuf.WriteRune(r)
				ctnttxtr = 0
				ctntprvr = r
				return
			}
			ctntprsvatvbuf.WriteRune(r)
			ctntprvr = r
		}
		ctntprvr = r
		return
	}

	var prsc rune
	var prscs int
	var prscerr error
	for prsngerr == nil && !dneprsng {
		if prsc, prscs, prscerr = ctntrdr.ReadRune(); prscs > 0 && (prscerr == nil || prscerr == io.EOF) {
			if prsngerr == nil && cdelbli[0] == cdepreL {
				prsngerr = cdeparser(prsc, cdepreL, cdepostL, cdelblrns[0], cdelblrns[1], cdelbli)
				continue
			}
			prsngerr = ctntparser(prsc, ctntpreL, ctntpostL, ctntlblrns[0], ctntlblrns[1], ctntlbli)
		}
		if dneprsng = prscerr == io.EOF; !dneprsng && prscerr != nil {
			prsngerr = prscerr
		}
	}

	if prsngerr == nil && dneprsng {
		if prsngerr = cdeflushpsv(); prsngerr == nil {
			cdeflushatv()
			var chdpgrm interface{} = nil
			if !cdebuf.Empty() {
				if DefaultMinifyCde != nil {
					DefaultMinifyCde(".js", cdebuf, nil)
				}
				if evalcode != nil {
					_, prsngerr = evalcode(cdebuf.Reader(), func(prgm interface{}) {
						chdpgrm = prgm
					})
				}
			}
			if prsngerr == nil {
				if capturecache != nil {
					if cdebuf.Empty() && !chdpsvbuf.Empty() {
						DefaultMinifyPsv(pathext, chdpsvbuf, nil)
					}
					prsngerr = capturecache(fullpath, pathModified, validelempaths, chdpsvbuf, cdebuf, chdpgrm)
				}
			}
		}
	}

	return
}

func validElmchar(cr rune) bool {
	return ('a' <= cr && cr <= 'z') || ('A' <= cr && cr <= 'Z') || cr == ':' || cr == '.' || cr == '-' || cr == '_' || ('0' <= cr && cr <= '9')
}

func validElemChar(prvr, r rune) (valid bool) {
	if prvr > 0 {
		valid = validElmchar(prvr) && validElmchar(r)
		return
	}
	valid = validElmchar(r)
	return
}
