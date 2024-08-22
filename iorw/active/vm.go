package active

import (
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/lnksnk/lnksnk/iorw"

	"github.com/lnksnk/lnksnk/iorw/active/require"

	"github.com/lnksnk/lnksnk/ja"
	"github.com/lnksnk/lnksnk/ja/parser"
)

type VM struct {
	vm            *ja.Runtime
	vmreq         *require.RequireModule
	objmap        map[string]interface{}
	DisposeObject func(string, interface{})
	W             io.Writer
	R             io.Reader
	buffs         map[*iorw.Buffer]*iorw.Buffer
	ErrPrint      func(...interface{}) error
}

func NewVM(a ...interface{}) (vm *VM) {
	var w io.Writer = nil
	var r io.Reader = nil
	var stngs map[string]interface{} = nil
	for _, d := range a {
		if d != nil {
			if wd, _ := d.(io.Writer); wd != nil {
				if w == nil {
					w = wd
				}
			} else if rd, _ := d.(io.Reader); rd != nil {
				if r == nil {
					r = rd
				}
			} else if stngsd, _ := d.(map[string]interface{}); stngsd != nil {
				if stngs == nil {
					stngs = map[string]interface{}{}
					for stngk, stngv := range stngsd {
						stngs[stngk] = stngv
					}
				}
			}
		}
	}
	vm = &VM{vm: ja.New(), W: w, R: r, objmap: map[string]interface{}{}}
	vm.Set("console", map[string]interface{}{
		"log": func(a ...interface{}) {
			iorw.Fprintln(os.Stdout, a...)
		},
		"error": func(a ...interface{}) {
			iorw.Fprintln(os.Stdout, a...)
		},
		"warn": func(a ...interface{}) {
			iorw.Fprintln(os.Stdout, a...)
		},
	})
	vm.vmreq = gojaregistry.Enable(vm.vm, vm.Print, vm.Println)
	//vm.vm.RunProgram(typescript.TypeScriptProgram)
	vm.vm.RunProgram(adhocPrgm)

	var fldmppr = &fieldmapper{fldmppr: ja.UncapFieldNameMapper()}
	vm.vm.SetFieldNameMapper(fldmppr)
	for stngk, stngv := range stngs {
		if stngv != nil {
			if strings.EqualFold(stngk, "ERRPRINT") {
				if errprint, _ := stngv.(func(a ...interface{}) error); errprint != nil {
					if vm.ErrPrint == nil {
						vm.ErrPrint = errprint
					}
				}
			} else if strings.EqualFold(stngk, "$") {
				vm.vmreq.SetObj("$", stngv)
				vm.Set("$", stngv)
			}
		}
		delete(stngs, stngk)
	}
	vm.Set("include", func(modname string) bool {
		IncludeModule(vm.vm, modname)
		return true
	})
	vm.Set("setPrinter", vm.SetPrinter)
	vm.Set("print", vm.Print)
	vm.Set("println", vm.Println)
	vm.Set("binwrite", vm.Write)

	vm.Set("setReader", vm.SetReader)
	vm.Set("binread", vm.Read)
	vm.Set("readln", vm.Readln)
	vm.Set("readlines", vm.ReadLines)
	vm.Set("readAll", vm.ReadAll)
	vm.Set("sleep", func(d int64, a ...interface{}) interface{} {
		time.Sleep(time.Duration(d) * time.Millisecond)
		if al := len(a); al > 0 {
			if al == 1 {
				return vm.InvokeFunction(a[0])
			} else if al > 1 {
				return vm.InvokeFunction(a[0], a[1:]...)
			}
		}
		return nil
	})
	vm.Set("sleepnano", func(d int64, a ...interface{}) interface{} {
		time.Sleep(time.Duration(d) * time.Nanosecond)
		if al := len(a); al > 0 {
			if al == 1 {
				return vm.InvokeFunction(a[0])
			} else if al > 1 {
				return vm.InvokeFunction(a[0], a[1:]...)
			}
		}
		return nil
	})
	vm.Set("sleepsec", func(d int64, a ...interface{}) interface{} {
		time.Sleep(time.Duration(d) * time.Second)
		if al := len(a); al > 0 {
			if al == 1 {
				return vm.InvokeFunction(a[0])
			} else if al > 1 {
				return vm.InvokeFunction(a[0], a[1:]...)
			}
		}
		return nil
	})
	vm.Set("sleepmin", func(d int64, a ...interface{}) interface{} {
		time.Sleep(time.Duration(d) * time.Minute)
		if al := len(a); al > 0 {
			if al == 1 {
				return vm.InvokeFunction(a[0])
			} else if al > 1 {
				return vm.InvokeFunction(a[0], a[1:]...)
			}
		}
		return nil
	})
	vm.Set("buffer", func() (buf *iorw.Buffer) {
		buf = iorw.NewBuffer()
		if vm.buffs == nil {
			vm.buffs = map[*iorw.Buffer]*iorw.Buffer{}
		}
		buf.OnClose = func(b *iorw.Buffer) {
			delete(vm.buffs, b)
		}
		return
	})
	return
}

type fieldmapper struct {
	fldmppr ja.FieldNameMapper
}

// FieldName returns a JavaScript name for the given struct field in the given type.
// If this method returns "" the field becomes hidden.
func (fldmppr *fieldmapper) FieldName(t reflect.Type, f reflect.StructField) (fldnme string) {
	if f.Tag != "" {
		fldnme = f.Tag.Get("json")
	} else {
		fldnme = uncapitalize(f.Name) // fldmppr.fldmppr.FieldName(t, f)
	}
	return
}

// MethodName returns a JavaScript name for the given method in the given type.
// If this method returns "" the method becomes hidden.
func (fldmppr *fieldmapper) MethodName(t reflect.Type, m reflect.Method) (mthdnme string) {
	mthdnme = uncapitalize(m.Name)
	return
}

func uncapitalize(s string) (nme string) {
	if sl := len(s); sl > 0 {
		var nrxtsr = rune(0)
		for sn := range s {
			sr := s[sn]
			if 'A' <= sr && sr <= 'Z' {
				sr += 'a' - 'A'
				nme += string(sr)
			} else {
				nme += string(sr)
			}
			if sn <= (sl-1)-1 {
				nrxtsr = rune(s[sn+1])
			} else {
				nrxtsr = rune(0)
			}
			if 'a' <= nrxtsr && nrxtsr <= 'z' {
				nme += s[sn+1:]
				break
			}
		}
	}
	return nme
}

func (vm *VM) Get(objname string) (obj interface{}) {
	if vm != nil {
		if objmap := vm.objmap; objmap != nil {
			obj = objmap[objname]
		}
	}
	return
}

func (vm *VM) Set(objname string, obj interface{}) {
	if vm != nil && vm.objmap != nil && objname != "" {
		objm, objok := vm.objmap[objname]
		if objok && &objm != &obj {
			if objm != nil {
				vm.Remove(objname)
				if vm.vm != nil {
					vm.objmap[objname] = obj
					vm.vm.Set(objname, obj)
				}
			}
		} else {
			if vm.vm != nil {
				vm.objmap[objname] = obj
				vm.vm.Set(objname, obj)
			}
		}
	}
}

func (vm *VM) InvokeFunction(functocall interface{}, args ...interface{}) (result interface{}) {
	if functocall != nil {
		if vm != nil && vm.vm != nil {
			var fnccallargs []ja.Value = nil
			var argsn = 0

			for argsn < len(args) {
				if fnccallargs == nil {
					fnccallargs = make([]ja.Value, len(args))
				}
				fnccallargs[argsn] = vm.vm.ToValue(args[argsn])
				argsn++
			}
			if atvfunc, atvfuncok := functocall.(func(ja.FunctionCall) ja.Value); atvfuncok {
				if len(fnccallargs) == 0 || fnccallargs == nil {
					fnccallargs = []ja.Value{}
				}
				var funccll = ja.FunctionCall{This: ja.Undefined(), Arguments: fnccallargs}
				if rsltval := atvfunc(funccll); rsltval != nil {
					result = rsltval.Export()
				}
			}
		}
	}
	return
}

func (vm *VM) Remove(objname string) {
	if vm != nil && objname != "" {
		if vm.objmap != nil {
			if _, objok := vm.objmap[objname]; objok {
				if vm.vm != nil {
					if glblobj := vm.vm.GlobalObject(); glblobj != nil {
						glblobj.Delete(objname)
					}
				}
				vm.objmap[objname] = nil
				delete(vm.objmap, objname)
			}
		}
	}
}

func (vm *VM) SetReaderPrinter(r io.Reader, w io.Writer) {
	vm.SetReader(r)
	vm.SetPrinter(w)
}

func (vm *VM) SetReader(r io.Reader) {
	if vm != nil && vm.R != r {
		vm.R = r
	}
}

func (vm *VM) Read(p ...byte) (n int, err error) {
	if vm != nil && vm.R != nil {
		n, err = vm.R.Read(p)
	}
	return
}

func (vm *VM) Readln() (ln string, err error) {
	if vm != nil && vm.R != nil {
		ln, err = iorw.ReadLine(vm.R)
	}
	return
}

func (vm *VM) ReadLines() (lines []string, err error) {
	if vm != nil && vm.R != nil {
		lines, err = iorw.ReadLines(vm.R)
	}
	return
}

func (vm *VM) ReadAll() (all string, err error) {
	if vm != nil && vm.R != nil {
		all, err = iorw.ReaderToString(vm.R)
	}
	return
}

func (vm *VM) SetPrinter(w io.Writer) {
	if vm != nil && vm.W != w {
		vm.W = w
	}
}

func (vm *VM) Print(a ...interface{}) (err error) {
	if vm != nil && vm.W != nil {
		err = iorw.Fprint(vm.W, a...)
	}
	return
}

func (vm *VM) Println(a ...interface{}) (err error) {
	if vm != nil && vm.W != nil {
		err = iorw.Fprintln(vm.W, a...)
	}
	return
}

func (vm *VM) Write(p ...byte) (n int, err error) {
	if vm != nil && vm.W != nil {
		n, err = vm.W.Write(p)
	}
	return
}

var DefaultTransformCode func(code string) (transformedcode string, errors []string, warnings []string)

func (vm *VM) Eval(a ...interface{}) (val interface{}, err error) {
	if vm != nil && vm.vm != nil {
		var cdes = ""
		var chdprgm *ja.Program = nil
		var setchdprgm func(interface{}, error, error)
		var ai, ail = 0, len(a)

		var errfound func(...interface{}) error = nil
		for ai < ail {
			if chdpgrmd, chdpgrmdok := a[ai].(*ja.Program); chdpgrmdok {
				if chdprgm == nil && chdpgrmd != nil {
					chdprgm = chdpgrmd
				}
				ail--
				a = append(a[:ai], a[ai+1:]...)
			} else if setchdpgrmd, setchdpgrmdok := a[ai].(func(interface{}, error, error)); setchdpgrmdok {
				if setchdprgm == nil && setchdpgrmd != nil {
					setchdprgm = setchdpgrmd
				}
				ail--
				a = append(a[:ai], a[ai+1:]...)
			} else if errfoundd, errfounddok := a[ai].(func(...interface{}) error); errfounddok {
				if errfound == nil && errfoundd != nil {
					errfound = errfoundd
				}
				ail--
				a = append(a[:ai], a[ai+1:]...)
			} else {
				ai++
			}
		}
		if func() {
			var psrdprgm = func() (p *ja.Program, perr error) {
				if chdprgm != nil {
					p = chdprgm
					return
				}
				if p == nil {
					var cde = iorw.NewMultiArgsReader(a...)
					defer cde.Close()
					cdes, _ = cde.ReadAll()
					if prsd, prsderr := parser.ParseFile(nil, "", cdes, 0, parser.WithDisableSourceMaps); prsderr == nil {
						if p, perr = ja.CompileAST(prsd, false); perr == nil && p != nil {
							if setchdprgm != nil {
								setchdprgm(p, nil, nil)
							}
						}
						if p == nil && perr != nil && setchdprgm == nil {
							setchdprgm(nil, nil, perr)
						}
					} else {
						if setchdprgm != nil {
							setchdprgm(nil, prsderr, nil)
						}
						p, perr = nil, prsderr
					}
				}
				return
			}
			p, perr := psrdprgm()
			if perr == nil && p != nil {
				gojaval, gojaerr := vm.vm.RunProgram(p)
				if gojaerr == nil {
					if gojaval != nil {
						val = gojaval.Export()
						return
					}
					return
				}
				err = gojaerr
				return
			}
			err = perr
		}(); err != nil {
			errfns := []func(...interface{}) error{}
			if vm.ErrPrint != nil {
				errfns = append(errfns, vm.ErrPrint)
			}
			if errfound != nil {
				errfns = append(errfns, errfound)
			}
			for _, ErrPrint := range errfns {
				func() {
					var linecnt = 1
					var errcdebuf = iorw.NewBuffer()
					errcdebuf.Print(fmt.Sprintf("%d: ", linecnt))
					defer errcdebuf.Close()
					var prvr = rune(0)
					for _, r := range cdes {
						if r == '\n' {
							linecnt++
							if prvr == '\r' {
								errcdebuf.WriteRune(prvr)
							}
							errcdebuf.WriteRune(r)
							errcdebuf.Print(fmt.Sprintf("%d: ", linecnt))
							prvr = 0
						} else {
							if r != '\r' {
								errcdebuf.WriteRune(r)
							}
						}
						prvr = r
					}
					ErrPrint("err:"+err.Error(), "\r\n", "err-code:"+errcdebuf.String())
				}()
			}
		}
	}
	return
}

func (vm *VM) Close() {
	if vm != nil {
		if vm.objmap != nil {
			if DisposeObject, objmap := vm.DisposeObject, vm.objmap; objmap != nil || DisposeObject != nil {
				if DisposeObject != nil {
					vm.DisposeObject = nil
					if objmap != nil {
						for objname, objval := range vm.objmap {
							vm.Remove(objname)
							DisposeObject(objname, objval)
						}
						vm.objmap = nil
					}
				}
				if objmap != nil {
					for objname := range vm.objmap {
						vm.Remove(objname)
					}
					vm.objmap = nil
				}
			}
			vm.objmap = nil
		}
		if vm.buffs != nil {
			for buf := range vm.buffs {
				buf.Close()
			}
			vm.buffs = nil
		}
		if vm.vmreq != nil {
			vm.vmreq = nil
		}
		if gojavm := vm.vm; gojavm != nil {
			vm.vm = nil
		}
	}
}

var gobalMods *sync.Map

var adhocPrgm *ja.Program = nil

var gojaregistry *require.Registry

var VMSourceLoader require.SourceLoader

func LoadGlobalModule(modname string, a ...interface{}) {
	if _, ok := gobalMods.Load(modname); ok {

	} else {
		func() {
			var cdebuf = iorw.NewBuffer()
			defer cdebuf.Close()
			if prgmast, _ := ja.Parse(modname, cdebuf.String()); prgmast != nil {
				if prgm, _ := ja.CompileAST(prgmast, false); prgm != nil {
					gobalMods.Store(modname, prgm)
				}
			}
		}()
	}
}

func IncludeModule(vm *ja.Runtime, modname string) {
	if prgv, ok := gobalMods.Load(modname); ok {
		if prg, _ := prgv.(*ja.Program); prg != nil {
			vm.RunProgram(prg)
		}
	}
}

func init() {

	gobalMods = &sync.Map{}
	gojaregistry = require.NewRegistryWithLoader(func(path string) (src []byte, err error) {
		if VMSourceLoader != nil {
			src, err = VMSourceLoader(path)
		}
		return
	})

	if adhocast, _ := ja.Parse(``, `_methods = (obj) => {
		let properties = new Set()
		let currentObj = obj
		Object.entries(currentObj).forEach((key)=>{
			key=(key=(key+"")).indexOf(",")>0?key.substring(0,key.indexOf(',')):key;
			if (typeof currentObj[key] === 'function') {
				var item=key;
				properties.add(item);
			}
		});
		if (properties.size===0) {
			do {
				Object.getOwnPropertyNames(currentObj).map(item => properties.add(item))
			} while ((currentObj = Object.getPrototypeOf(currentObj)))
		}
		return [...properties.keys()].filter(item => typeof obj[item] === 'function')
	}
	
	_fields = (obj) => {
		let properties = new Set()
		let currentObj = obj
		Object.entries(currentObj).forEach((key)=>{
			key=(key=(key+"")).indexOf(",")>0?key.substring(0,key.indexOf(',')):key;
			if (typeof currentObj[key] !== 'function') {
				var item=key;
				properties.add(item);
			}
		});
		if (properties.size===0) {
			do {
				Object.getOwnPropertyNames(currentObj).map(item => properties.add(item))
			} while ((currentObj = Object.getPrototypeOf(currentObj)))
		}
		return [...properties.keys()].filter(item => item!=='__proto__' && typeof obj[item] !== 'function')
	}`); adhocast != nil {
		adhocPrgm, _ = ja.CompileAST(adhocast, false)
	}

}
