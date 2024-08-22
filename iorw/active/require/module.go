package require

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"sync"
	"syscall"
	"text/template"
	"time"

	"github.com/lnksnk/lnksnk/fsutils"

	js "github.com/lnksnk/lnksnk/ja"
	"github.com/lnksnk/lnksnk/ja/parser"
)

type ModuleLoader func(*js.Runtime, *js.Object)

// SourceLoader represents a function that returns a file data at a given path.
// The function should return ModuleFileDoesNotExistError if the file either doesn't exist or is a directory.
// This error will be ignored by the resolver and the search will continue. Any other errors will be propagated.
type SourceLoader func(path string) ([]byte, error)

var (
	InvalidModuleError     = errors.New("Invalid module")
	IllegalModuleNameError = errors.New("Illegal module name")

	ModuleFileDoesNotExistError = errors.New("module file does not exist")
)

var native map[string]ModuleLoader

// Registry contains a cache of compiled modules which can be used by multiple Runtimes
type Registry struct {
	sync.Mutex
	native        map[string]ModuleLoader
	compiled      map[string]*js.Program
	compiledmod   map[string]time.Time
	srcLoader     SourceLoader
	globalFolders []string
}

type RequireModule struct {
	r           *Registry
	runtime     *js.Runtime
	modules     map[string]*js.Object
	nodeModules map[string]*js.Object
	Print       func(a ...interface{}) (err error)
	Println     func(a ...interface{}) (err error)
	adhocobjs   map[string]interface{}
}

func NewRegistry(opts ...Option) *Registry {
	r := &Registry{}

	for _, opt := range opts {
		opt(r)
	}

	return r
}

func NewRegistryWithLoader(srcLoader SourceLoader) *Registry {
	return NewRegistry(WithLoader(srcLoader))
}

type Option func(*Registry)

// WithLoader sets a function which will be called by the require() function in order to get a source code for a
// module at the given path. The same function will be used to get external source maps.
// Note, this only affects the modules loaded by the require() function. If you need to use it as a source map
// loader for code parsed in a different way (such as runtime.RunString() or eval()), use (*Runtime).SetParserOptions()
func WithLoader(srcLoader SourceLoader) Option {
	return func(r *Registry) {
		r.srcLoader = srcLoader
	}
}

// WithGlobalFolders appends the given paths to the registry's list of
// global folders to search if the requested module is not found
// elsewhere.  By default, a registry's global folders list is empty.
// In the reference Node.js implementation, the default global folders
// list is $NODE_PATH, $HOME/.node_modules, $HOME/.node_libraries and
// $PREFIX/lib/node, see
// https://nodejs.org/api/modules.html#modules_loading_from_the_global_folders.
func WithGlobalFolders(globalFolders ...string) Option {
	return func(r *Registry) {
		r.globalFolders = globalFolders
	}
}

// Enable adds the require() function to the specified runtime.
func (r *Registry) Enable(runtime *js.Runtime, a ...interface{}) *RequireModule {
	rrt := &RequireModule{
		r:           r,
		runtime:     runtime,
		modules:     make(map[string]*js.Object),
		nodeModules: make(map[string]*js.Object),
	}

	for _, d := range a {
		if prntd, _ := d.(func(...interface{}) error); prntd != nil {
			if rrt.Print == nil {
				rrt.Print = prntd
			} else if rrt.Println == nil {
				rrt.Println = prntd
			}
		}
	}

	runtime.Set("require", rrt.require)
	return rrt
}

func (r *Registry) RegisterNativeModule(name string, loader ModuleLoader) {
	r.Lock()
	defer r.Unlock()

	if r.native == nil {
		r.native = make(map[string]ModuleLoader)
	}
	name = filepathClean(name)
	r.native[name] = loader
}

var DefaultSourceFS *fsutils.FSUtils

// DefaultSourceLoader is used if none was set (see WithLoader()). It simply loads files from the host's filesystem.
func DefaultSourceLoader(filename string) ([]byte, error) {
	if DefaultSourceFS != nil {
		if rcat := DefaultSourceFS.CAT(filename); rcat != nil {
			return io.ReadAll(rcat)
		}
		return nil, ModuleFileDoesNotExistError
	}
	fp := filepath.FromSlash(filename)
	f, err := os.Open(fp)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			err = ModuleFileDoesNotExistError
		} else if runtime.GOOS == "windows" {
			if errors.Is(err, syscall.Errno(0x7b)) { // ERROR_INVALID_NAME, The filename, directory name, or volume label syntax is incorrect.
				err = ModuleFileDoesNotExistError
			}
		}
		return nil, err
	}

	defer f.Close()
	// On some systems (e.g. plan9 and FreeBSD) it is possible to use the standard read() call on directories
	// which means we cannot rely on read() returning an error, we have to do stat() instead.
	if fi, err := f.Stat(); err == nil {
		if fi.IsDir() {
			return nil, ModuleFileDoesNotExistError
		}
	} else {
		return nil, err
	}
	return io.ReadAll(f)
}

func (r *Registry) getSource(p string) ([]byte, error) {
	srcLoader := r.srcLoader
	if srcLoader == nil {
		srcLoader = DefaultSourceLoader
	}
	return srcLoader(p)
}

func (r *Registry) getCompiledSource(p string) (*js.Program, error) {
	r.Lock()
	defer r.Unlock()

	prg := r.compiled[p]
	if DefaultSourceFS != nil {
		if prg != nil {
			if r.compiledmod == nil {
				r.compiledmod = map[string]time.Time{}
				delete(r.compiled, p)
				prg = nil
			} else {
				prgmod := r.compiledmod[p]
				if fsinfos := DefaultSourceFS.FIND(p); len(fsinfos) == 1 {
					if fsinfos[0].ModTime() != prgmod {
						delete(r.compiled, p)
						prg = nil
					} else {
						return prg, nil
					}
				}
			}
		}
	}
	if prg == nil {
		buf, err := r.getSource(p)
		if err != nil {
			return nil, err
		}
		s := string(buf)

		if path.Ext(p) == ".json" {
			s = "module.exports = JSON.parse('" + template.JSEscapeString(s) + "')"
		}

		source := "(function(exports, require, module) {" + s + "\n})"
		parsed, err := js.Parse(p, source, parser.WithSourceMapLoader(r.srcLoader))
		if err != nil {
			return nil, err
		}
		prg, err = js.CompileAST(parsed, false)
		if err == nil {
			if r.compiled == nil {
				r.compiled = make(map[string]*js.Program)
			}
			r.compiled[p] = prg
			if r.compiledmod == nil {
				r.compiledmod = map[string]time.Time{}
			}
			if DefaultSourceFS != nil {
				if fsinfos := DefaultSourceFS.FIND(p); len(fsinfos) == 1 {
					r.compiledmod[p] = fsinfos[0].ModTime()
				} else {
					r.compiledmod[p] = time.Now()
				}
			} else {
				r.compiledmod[p] = time.Now()
			}
		}
		return prg, err
	}
	return prg, nil
}

func (r *RequireModule) require(call js.FunctionCall) js.Value {
	ret, err := r.Require(call.Argument(0).String())
	if err != nil {
		if _, ok := err.(*js.Exception); !ok {
			panic(r.runtime.NewGoError(err))
		}
		panic(err)
	}
	return ret
}

func filepathClean(p string) string {
	return path.Clean(p)
}

// Require can be used to import modules from Go source (similar to JS require() function).
func (r *RequireModule) Require(p string) (ret js.Value, err error) {
	module, err := r.resolve(p)
	if err != nil {
		return
	}
	ret = module.Get("exports")
	return
}

func Require(runtime *js.Runtime, name string) js.Value {
	if r, ok := js.AssertFunction(runtime.Get("require")); ok {
		mod, err := r(js.Undefined(), runtime.ToValue(name))
		if err != nil {
			panic(err)
		}
		return mod
	}
	panic(runtime.NewTypeError("Please enable require for this runtime using new(require.Registry).Enable(runtime)"))
}

func RegisterNativeModule(name string, loader ModuleLoader) {
	if native == nil {
		native = make(map[string]ModuleLoader)
	}
	name = filepathClean(name)
	native[name] = loader
}
