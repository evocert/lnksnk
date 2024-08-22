package parsing

import (
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/evocert/lnksnk/fsutils"
	"github.com/evocert/lnksnk/iorw"
)

func internalProcessParsingOld(
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
	tmpmatchthis["rsroot"] = func() (rsroot string) {
		if root != "" {
			if root == "/" {
				rsroot = root
				return
			}
			if root[0] == '/' {
				rsroot = "/"
				if rti := strings.Index(root[1:], "/"); rti > -1 {
					rsroot += root[1:][:rti+1]
					return
				}
				return
			}
			rsroot = "/"
			if rti := strings.Index(root, "/"); rti > -1 {
				rsroot += root[:rti+1]
				return
			}
			return
		}
		return
	}()
	rsroot := tmpmatchthis["rsroot"].(string)
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
	tmpmatchthis["elembase"] = func() (elembase string) {
		elmbases := strings.Split(elempath, ":")
		enajst := 0
		for en, elmb := range elmbases {
			if elmb == "" {
				if en == 0 {
					elembase = ":" + elembase
				}
				enajst++
				continue
			}
			if (en + enajst) < len(elmbases)-1 {
				elembase += elmb + ":"
			}
		}
		return
	}()
	elemroot := tmpmatchthis["elemroot"].(string)
	tmpmatchthis["elemrsbase"] = func() (elemrsbase string) {

		if elemroot != "" {
			if elemroot == ":" {
				elemrsbase = elemroot
				return
			}
			if elemroot[0] == ':' {
				elemrsbase = ":"
				if rti := strings.Index(elemroot[1:], ":"); rti > -1 {
					elemrsbase += elemroot[1:][:rti+1]
					return
				}
				return
			}
			elemrsbase = ":"
			if rti := strings.Index(elemroot, ":"); rti > -1 {
				elemrsbase += elemroot[:rti+1]
				return
			}
			return
		}
		return
	}()
	//elemrsbase := tmpmatchthis["elemrsbase"].(string)
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
		validelempaths[fi.Path()] = fi.ModTime()
		elmnext.level = len(elemlevels)
		if elmnext.level > 0 {
			elmnext.prvctntelem = elemlevels[elmnext.level-1]
		}
		elemlevels = append([]*contentelem{elmnext}, elemlevels...)
		return
	}
	var nextfullname = func(elemname string, elmlvl ctntelemlevel) (fullname string) {
		if strings.Contains(elemname, "..:") {
			elemname = strings.Replace(elemname, "..:", ":", -1)
		}
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
	}

	ctntEventReader.ValidElemEvent = func(elmlvl ctntelemlevel, elemname string, elmbuf *iorw.Buffer, args *contentargsreader) (evtvalid bool, vlerr error) {
		if fs != nil {
			var fi fsutils.FileInfo = nil

			fullelemname := nextfullname(elemname, elmlvl)
			if invalidelempaths[fullelemname] {
				if !elmbuf.Empty() && crntnextelm != nil {
					prepInvalidElemBuf(elmbuf, crntnextelm)
				}
				return
			}
			if elmlvl == ctntElemStart || elmlvl == ctntElemSingle {
				testpath := strings.Replace(fullelemname, ":", "/", -1)
				testext := filepath.Ext(testpath)
				if testext != "" {
					testpath = testpath[:len(testpath)-len(testext)]
				}

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
								if testpath[0] == '/' && rsroot != "/" && !strings.HasPrefix(testpath, rsroot) {
									if fios := fs.LS(rsroot + testpath[1:] + "index" + nextpth); len(fios) == 1 {
										return fios[0]
									}
								}
								if testpath[0] == '/' && !strings.HasPrefix(testpath, pathroot) {
									if fios := fs.LS(pathroot + testpath[1:] + "index" + nextpth); len(fios) == 1 {
										return fios[0]
									}
								}
							}
						}
					}
					if fios := fs.LS(testpath + testext); len(fios) == 1 {
						return fios[0]
					}
					if testpath[0] == '/' && rsroot != "/" && !strings.HasPrefix(testpath, rsroot) {
						if fios := fs.LS(rsroot + testpath[1:] + testext); len(fios) == 1 {
							return fios[0]
						}
					}
					if testpath[0] == '/' && !strings.HasPrefix(testpath, pathroot) {
						if fios := fs.LS(pathroot + testpath[1:] + testext); len(fios) == 1 {
							return fios[0]
						}
					}
					return nil
				}(); fi == nil {
					invalidelempaths[fullelemname] = true
					if !elmbuf.Empty() && crntnextelm != nil {
						prepInvalidElemBuf(elmbuf, crntnextelm)
					}
					return
				}
				evtvalid = true
				crntnextelm = addelemlevel(fi, fullelemname, fi.PathExt())
				if !args.Done() {
					crntnextelm.prebuf = elmbuf
				} else {
					prevargs := func() (prvsttngs map[string]interface{}) {
						if len(elemlevels) > 0 {
							prvsttngs = elemlevels[len(elemlevels)-1].attrs
						}
						return
					}()
					for argk, argv := range args.args {
						if crntnextelm.attrs == nil {
							crntnextelm.attrs = map[string]interface{}{}
						}
						if len(prevargs) > 0 {
							arvbf := iorw.NewBuffer()
							arvbf.Print(argv)
							arvrplcrdr := iorw.NewReplaceRuneReader(arvbf.Clone(true).Reader(true))
							for prvk, prvv := range prevargs {
								arvrplcrdr.ReplaceWith(prvk, prvv)
							}
							arvbf.ReadRunesFrom(arvrplcrdr)
							argv = arvbf.Clone(true)
						}
						crntnextelm.attrs[argk] = argv
						delete(args.args, argk)
					}
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
					if !args.Done() {
						crntnextelm.prebuf = elmbuf
					} else {
						for argk, argv := range args.args {
							if crntnextelm.attrs == nil {
								crntnextelm.attrs = map[string]interface{}{}
							}
							crntnextelm.attrs[argk] = argv
						}
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
		fmt.Println("code:\r\n" + cdebuf.String())
		if DefaultMinifyCde != nil {
			prsngerr = DefaultMinifyCde(".js", cdebuf, nil)
		}
		if evalcode != nil && prsngerr == nil {
			var evalresult interface{} = nil
			evalresult, prsngerr = evalcode(cdebuf.Reader(), func(prgm interface{}, prsccdeerr error, cmpleerr error) {
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
			})
			if prsngerr == nil {
				if pathext == ".json" {
					if out != nil {
						json.NewEncoder(out).Encode(&evalresult)
					}
					return
				}
				iorw.Fbprint(out, evalresult)
			}
		}
		if prsngerr != nil {
			println(prsngerr.Error())
			println()
			if cderr, _ := prsngerr.(CodeError); cderr != nil {
				println(cderr.Code())

			} else {
				println(cdebuf.String())
			}
		}
	}
	return
}
