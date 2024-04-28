package parsing

import (
	"io"

	"sync"
	"time"

	"github.com/evocert/lnksnk/fsutils"
	"github.com/evocert/lnksnk/iorw"

	"github.com/evocert/lnksnk/concurrent"
)

type CachedScript struct {
	chdscrptng *CachedScripting
	path       string
	modified   time.Time
	psvlck     sync.RWMutex
	psvbuf     *iorw.Buffer
	atvlck     sync.RWMutex
	atvbuf     *iorw.Buffer
	chdsublems *concurrent.Map
	scrptprgm  interface{}
}

func (chdscrpt *CachedScript) IsValidSince(testmod time.Time, fs *fsutils.FSUtils) (isvalid bool) {
	if chdscrpt != nil {
		if isvalid = chdscrpt.modified == testmod; isvalid {
			if chdsublems := chdscrpt.chdsublems; fs != nil && chdsublems != nil {
				lstmods := map[string]time.Time{}
				lspaths := []string{}
				chdsublems.Range(func(key, value any) bool {
					return func(lstpath string, lstmod time.Time) bool {
						lspaths = append(lspaths, lstpath)
						lstmods[lstpath] = lstmod
						return true
					}(key.(string), value.(time.Time))
				})

				a := []interface{}{}
				for _, d := range lspaths {
					a = append(a, d)
				}
				fsinfos := fs.FIND(a...)

				for fsinfon, fsinfo := range fsinfos {
					if fsinfo.Path() == lspaths[fsinfon] {
						if fsinfo.ModTime() != lstmods[lspaths[fsinfon]] {
							isvalid = false
							break
						}
					}
				}
			}
		}
	}
	return
}

func (chdscrpt *CachedScript) SetScriptProgram(scrptpgrm interface{}) {
	if chdscrpt != nil && scrptpgrm != nil {
		chdscrpt.scrptprgm = scrptpgrm
	}
}

func (chdscrpt *CachedScript) ScriptProgram() (scrptpgrm interface{}) {
	if chdscrpt != nil {
		return chdscrpt.scrptprgm
	}
	return
}

func (chdscrpt *CachedScript) Dispose() {
	if chdscrpt != nil {
		if chdscrpt.chdscrptng != nil {
			chdscrpt.chdscrptng.chdscrpts.Del(chdscrpt.path)
			chdscrpt.chdscrptng = nil
		}
		if chdscrpt.psvbuf != nil {
			chdscrpt.psvbuf.Close()
			chdscrpt.psvbuf = nil
		}
		if chdscrpt.atvbuf != nil {
			chdscrpt.atvbuf.Close()
			chdscrpt.atvbuf = nil
		}
		if chdscrpt.chdsublems != nil {
			chdscrpt.chdsublems.Dispose()
			chdscrpt.chdsublems = nil
		}
	}
}

func (chdscrpt *CachedScript) WritePsvTo(w io.Writer, path ...string) (n int64, err error) {
	if chdscrpt != nil {
		psvbuf := chdscrpt.psvbuf
		if psvbuf.Empty() {
			return
		}
		chdscrpt.psvlck.RLock()
		defer chdscrpt.psvlck.RUnlock()
		n, err = psvbuf.WriteTo(w)
	}
	return
}

func (chdscrpt *CachedScript) WriteAtvTo(w io.Writer, path ...string) (n int64, err error) {
	if chdscrpt != nil {
		atvbuf := chdscrpt.atvbuf
		if atvbuf.Empty() {
			return
		}
		chdscrpt.atvlck.RLock()
		defer chdscrpt.atvlck.RUnlock()
		n, err = atvbuf.WriteTo(w)
	}
	return
}

func (chdscrpt *CachedScript) EvalAtv(evalatv func(a ...interface{}) (interface{}, error)) (result interface{}, err error) {
	if chdscrpt != nil && evalatv != nil {
		if chdprg := chdscrpt.ScriptProgram(); chdprg != nil {
			result, err = evalatv(chdprg)
			return
		}
	}
	return
}

func newCachedScript(chdscrptng *CachedScripting, path string, modified time.Time, psvbuf *iorw.Buffer, atvbuf *iorw.Buffer, validElems map[string]time.Time) (chdscrpt *CachedScript) {
	chdscrpt = &CachedScript{chdscrptng: chdscrptng, path: path, modified: modified}
	if len(validElems) > 0 {
		if chdscrpt.chdsublems == nil {
			chdscrpt.chdsublems = concurrent.NewMap()
		}
		for velmfullpath, velmmod := range validElems {
			chdscrpt.chdsublems.Set(velmfullpath, velmmod)
		}
	}
	if psvbuf != nil {
		chdscrpt.psvbuf = psvbuf.Clone()
	}
	if atvbuf != nil {
		chdscrpt.atvbuf = atvbuf.Clone()
	}
	return
}

type CachedScripting struct {
	chdscrpts *concurrent.Map
}

func (chdscrptng *CachedScripting) Load(modified time.Time, psvbuf *iorw.Buffer, atvbuf *iorw.Buffer, validElems map[string]time.Time, path string) (chdscrpt *CachedScript) {
	if chdscrptng != nil {
		if path != "" {
			chdscrptok := false
			chdscrptany := interface{}(nil)
			if chdscrptany, chdscrptok = chdscrptng.chdscrpts.Get(path); !chdscrptok {
				chdscrpt = newCachedScript(chdscrptng /*nil,*/, path, modified, psvbuf, atvbuf, validElems)
				chdscrptng.chdscrpts.Set(path, chdscrpt)
			} else if chdscrptok {
				if chdscrpt, _ = chdscrptany.(*CachedScript); chdscrpt != nil {
					chdscrpt.modified = modified
					if psvbuf == nil {
						func() {
							chdscrpt.psvlck.Lock()
							defer chdscrpt.psvlck.Unlock()
							if chdscrpt.psvbuf != nil {
								chdscrpt.psvbuf.Close()
								chdscrpt.psvbuf = nil
							}
						}()
					} else {
						func() {
							chdscrpt.psvlck.Lock()
							defer chdscrpt.psvlck.Unlock()
							if chdscrpt.psvbuf != nil {
								chdscrpt.psvbuf.Clear()
								psvbuf.WriteTo(chdscrpt.psvbuf)
							} else {
								chdscrpt.psvbuf = psvbuf.Clone()
							}
						}()
					}
					if atvbuf == nil {
						func() {
							chdscrpt.atvlck.Lock()
							defer chdscrpt.atvlck.Unlock()
							if chdscrpt.atvbuf != nil {
								chdscrpt.atvbuf.Close()
								chdscrpt.atvbuf = nil
							}
						}()
					} else {
						func() {
							chdscrpt.atvlck.Lock()
							defer chdscrpt.atvlck.Unlock()
							if chdscrpt.atvbuf != nil {
								chdscrpt.atvbuf.Clear()
								atvbuf.WriteTo(chdscrpt.atvbuf)
							} else {
								chdscrpt.atvbuf = atvbuf.Clone()
							}
						}()
					}
				}
			}
		}
	}
	return
}

func (chdscrptng *CachedScripting) Script(path string) (chdscrpt *CachedScript) {
	if chdscrptng != nil {
		if path != "" {
			chdscrptok := false
			chdscrptany := interface{}(nil)
			if chdscrptany, chdscrptok = chdscrptng.chdscrpts.Get(path); chdscrptok {
				chdscrpt, _ = chdscrptany.(*CachedScript)
			}
		}
	}
	return
}

var gblchdscrptng *CachedScripting = nil

func GLOBALCACHEDSCRIPTING() *CachedScripting {
	return gblchdscrptng
}

func init() {
	gblchdscrptng = &CachedScripting{chdscrpts: concurrent.NewMap()}
}
