package parsing

import (
	"encoding/json"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/lnksnk/lnksnk/fsutils"
	"github.com/lnksnk/lnksnk/iorw"
)

type ArgsEventReader struct {
	*iorw.ReplaceRuneReader
	preposts    map[string]string
	prebufs     map[string]*iorw.Buffer
	prepostevts map[string]func(argsevtr *ArgsEventReader, prefix, postfix string, phrsbuf *iorw.Buffer) (fndval bool, val interface{})
}

func NewArgsEventReader(rplcrdr *iorw.ReplaceRuneReader) (argsavtr *ArgsEventReader) {
	if rplcrdr != nil {
		argsavtr = &ArgsEventReader{ReplaceRuneReader: rplcrdr}
	}
	return
}

func (argsevtr *ArgsEventReader) ReadRune() (r rune, size int, err error) {
	if argsevtr != nil && argsevtr.ReplaceRuneReader != nil {
		return argsevtr.ReplaceRuneReader.ReadRune()
	}
	if size == 0 && err == nil {
		err = io.EOF
	}
	return
}

func (argsevtr *ArgsEventReader) ReplaceEvent(prefix string, rplcrdr *iorw.ReplaceRuneReader) (nxtrlst interface{}) {
	if argsevtr != nil {
		if preposts, prebufs := argsevtr.preposts, argsevtr.prebufs; len(preposts) > 0 {
			if postfix := preposts[prefix]; postfix != "" {
				if prebufs == nil {
					prebufs = map[string]*iorw.Buffer{}
					argsevtr.prebufs = prebufs
				}
				prebf := prebufs[prefix]
				if prebf == nil {
					prebf = iorw.NewBuffer()
					prebufs[prefix] = prebf
				}
				prebf.Clear()
				prebf.ReadRunesFrom(rplcrdr.ReadRunesUntil(postfix))
				if fnd, _ := rplcrdr.FoundEOF(); fnd {
					if PostFixFoundEvent := argsevtr.prepostevts[prefix+postfix]; PostFixFoundEvent != nil {
						fnd, val := PostFixFoundEvent(argsevtr, prefix, postfix, prebf)
						if fnd {
							return val
						}
						return ""
					}
				}
				rplcrdr.PreAppend(prebf.Clone(true).Reader(true))
				return
			}
		}
	}
	return
}

func (argsevtr *ArgsEventReader) MatchPhrase(phrsbuf *iorw.Buffer, args map[string]interface{}) (eqls bool, result interface{}) {
	if !phrsbuf.Empty() && len(args) > 0 {
		for argk, arv := range args {
			if eqls, _ = phrsbuf.Equals(argk); eqls {
				result = arv
				return
			}
		}
	}
	return
}

func (argsevtr *ArgsEventReader) SetPrePostFix(a ...interface{}) {
	if al := len(a); argsevtr != nil && al > 2 {
		preposts, prepostevts := argsevtr.preposts, argsevtr.prepostevts
		if preposts == nil {
			preposts = map[string]string{}
			argsevtr.preposts = preposts
		}
		if prepostevts == nil {
			prepostevts = map[string]func(argsevtr *ArgsEventReader, prefix string, postfix string, phrsbuf *iorw.Buffer) (fndval bool, val interface{}){}
			argsevtr.prepostevts = prepostevts
		}
		for al > 2 {
			if prepostevt, _ := a[0].(func(argsevtr *ArgsEventReader, prefix, postfix string, phrsbuf *iorw.Buffer) (fndval bool, val interface{})); prepostevt != nil {
				a = a[1:]
				al--
				if prefix := a[0].(string); prefix != "" {
					a = a[1:]
					al--
					if postfix := a[0].(string); postfix != "" {
						preposts[prefix] = postfix
						a = a[1:]
						al--
						argsevtr.ReplaceWith(prefix, argsevtr)
						prepostevts[prefix+postfix] = prepostevt
						continue
					}
				}
			}
			break
		}
	}
}

func validLastCdeRune(cr rune) bool {
	return cr == '=' || cr == '(' || cr == '[' || cr == ',' || cr == '+' || cr == '/' || cr == ':'
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
	mtchphrshndl *MatchPhraseHandler
	attrs        map[string]interface{}
	level        int
	prvctntelem  *contentelem
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
		prepairContentElem(ctntelm)
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

func prepairContentElem(ctntelm *contentelem) {
	if ctntelm != nil && ctntelm.runerdr == nil && ctntelm.fi != nil {
		cntntbuf := ctntelm.ctntbuf
		ctntelm.ctntbuf = nil
		var rdr io.RuneReader = nil
		if r, rerr := ctntelm.fi.Open(); rerr == nil {
			if rdr, _ = (r).(io.RuneReader); rdr == nil {
				rdr = iorw.NewEOFCloseSeekReader(r)
			}
		}
		ctntstngs := map[string]interface{}{}
		attrs := ctntelm.attrs

		agrsevtrdr := NewArgsEventReader(iorw.NewReplaceRuneReader(rdr))
		agrsevtrdr.SetPrePostFix(func(argsevtr *ArgsEventReader, prefix, postfix string, phrsbuf *iorw.Buffer) (fndval bool, val interface{}) {
			if prefix == "[#" && postfix == "#]" {
				if fndval, val = argsevtr.MatchPhrase(phrsbuf, attrs); fndval {
					return
				}
				return fndval, ""
			}
			return
		}, "[#", "#]")

		if len(attrs) > 0 {
			agrsevtrdr.SetPrePostFix(func(argsevtr *ArgsEventReader, prefix, postfix string, phrsbuf *iorw.Buffer) (fndval bool, val interface{}) {
				if prefix == "#" && postfix == "#" {
					if fndval, val = argsevtr.MatchPhrase(phrsbuf, attrs); fndval {
						return
					}
					return fndval, ""
				}
				return
			}, "#", "#")
		}
		rdr = agrsevtrdr
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
		prebuf := ctntelm.prebuf
		coresttngs := map[string]interface{}{}
		coresttngs["pre"] = prebuf.String()
		postbuf := ctntelm.postbuf
		coresttngs["post"] = postbuf.String()
		coresttngs["pathroot"] = pathroot
		coresttngs["root"] = root
		coresttngs["elemroot"] = func() (elmroot string) {
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
		coresttngs["elembase"] = func() (elembase string) {
			elmbases := strings.Split(coresttngs["elemroot"].(string), ":")
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

		agrsevtrdr.SetPrePostFix(func(argsevtr *ArgsEventReader, prefix, postfix string, phrsbuf *iorw.Buffer) (fndval bool, val interface{}) {
			if prefix == "<:_:" && postfix == ":/>" {
				fndval, val = argsevtr.MatchPhrase(phrsbuf, coresttngs)
			}
			return
		}, "<:_:", ":/>")

		ctntstngs["cntnt"] = cntntbuf
		agrsevtrdr.SetPrePostFix(func(argsevtr *ArgsEventReader, prefix, postfix string, phrsbuf *iorw.Buffer) (fndval bool, val interface{}) {
			if prefix == "<:" && postfix == ":/>" {
				fndval, val = argsevtr.MatchPhrase(phrsbuf, ctntstngs)
			}
			return
		}, "<:", ":/>")
		ctntelm.runerdr = agrsevtrdr
	}
}

// Close implements io.Closer
func (ctntelm *contentelem) Close() (err error) {
	if ctntelm != nil {
		postbuf, prebuf, ctntbuf, rawBuf, mtchphrshndl, attrs := ctntelm.postbuf, ctntelm.prebuf, ctntelm.ctntbuf, ctntelm.rawBuf, ctntelm.mtchphrshndl, ctntelm.attrs
		ctntelm.postbuf = nil
		ctntelm.prebuf = nil
		ctntelm.runerdr = nil
		ctntelm.rawBuf = nil
		ctntelm.fi = nil
		ctntelm.eofevent = nil
		ctntelm.mtchphrshndl = nil
		ctntelm.attrs = nil

		if postbuf != nil {
			postbuf.Close()
		}
		if prebuf != nil {
			prebuf.Close()
		}
		if ctntbuf != nil {
			ctntbuf.Close()
			ctntbuf = nil
		}
		if rawBuf != nil {
			rawBuf.Close()
		}
		if mtchphrshndl != nil {
			mtchphrshndl.Close()
		}
		for _, atv := range attrs {
			if atvbf, _ := atv.(*iorw.Buffer); atvbf != nil {
				atvbf.Close()
			}
		}
	}
	return
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
	ctntEventReader := newContentEventReader("<", ">", iorw.NewReplaceRuneReader(iorw.NewRuneReaderSlice(rnrdrs...), "<:_:", func(matchedphrase string, rplcrdr *iorw.ReplaceRuneReader) (nxtrdr interface{}) {
		if matchedphrase == "<:_:" {
			rnsuntil := rplcrdr.ReadRunesUntil(":/>")
			tmpphrasebuf.Clear()
			tmpphrasebuf.ReadRunesFrom(rnsuntil)
			if !tmpphrasebuf.Empty() {
				defer tmpphrasebuf.Clear()
				if fnd, _ := rplcrdr.FoundEOF(); fnd {
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
							}
						}
					}
					if fios := fs.LS(testpath + testext); len(fios) == 1 {
						return fios[0]
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

func prepInvalidElemBuf(elmbuf *iorw.Buffer, cntntelm *contentelem) {
	if !elmbuf.Empty() && cntntelm != nil && len(cntntelm.attrs) > 0 {
		var phrsbf *iorw.Buffer = nil
		elmrplcrdr := iorw.NewReplaceRuneReader(elmbuf.Clone(true).Reader(true), "#", func(phrase string, rplcrdr *iorw.ReplaceRuneReader) (nxtrdr interface{}) {
			if phrase == "#" {
				if phrsbf == nil {
					phrsbf = iorw.NewBuffer()
				} else {
					phrsbf.Clear()
				}
				prhseofrdr := rplcrdr.ReadRunesUntil("#")
				phrsbf.ReadRunesFrom(prhseofrdr)
				if fnd, _ := rplcrdr.FoundEOF(); fnd {
					for attk, attv := range func() map[string]interface{} {
						return cntntelm.attrs
					}() {
						if eqls, _ := phrsbf.Equals(attk); eqls {
							return attv
						}
					}
					//TEST GITHUB
					return ""
				}
				rplcrdr.PreAppend(phrsbf.Clone(true).Reader(true))
			}
			return
		})
		elmbuf.Print(elmrplcrdr)
	}
}

type CodeError interface {
	error
	Code() string
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
