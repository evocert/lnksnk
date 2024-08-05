package parsing

import "github.com/evocert/lnksnk/iorw"

type argstage int

const (
	argUnknown argstage = iota
	argValue
)

type contentargsreader struct {
	*ParseEventReader
	cntntrdr *contenteventreader
	args     map[string]interface{}
	argnmrns []rune
	argstg   argstage
	argvbf   *iorw.Buffer
	argtxtr  rune
	argprvr  rune
}

func (cargsr *contentargsreader) Done() bool {
	if cargsr == nil {
		return false
	}
	if prgstsg := cargsr.ParseStage(); prgstsg == PostStage {
		return false
	}
	if cargsr.argtxtr > 0 {
		return false
	}
	if cargsr.argstg == argValue {
		return false
	}
	if len(cargsr.argnmrns) > 0 {
		return false
	}
	return len(cargsr.args) > 0
}

func (cargsr *contentargsreader) WriteAtvRunes(rns ...rune) {
	if cargsr != nil {
		if cargsr.argstg == argValue {
			argvbf := cargsr.argvbf
			if argvbf == nil {
				argvbf = iorw.NewBuffer()
				cargsr.argvbf = argvbf
			}
			argvbf.WriteRunes(rns...)
		}
	}
}

func (cargsr *contentargsreader) savearg() {
	if cargsr != nil {
		if cargsr.argstg == argValue {
			if argvbf, argnmrns := cargsr.argvbf, cargsr.argnmrns; len(argnmrns) > 0 {
				args := cargsr.args
				if args == nil {
					args = map[string]interface{}{}
					cargsr.args = args
				}
				if argvbf.Empty() {
					args[string(argnmrns)] = ""
					cargsr.argnmrns = nil
					cargsr.argstg = argUnknown
					return
				}
				args[string(argnmrns)] = argvbf.Clone(true)
				cargsr.argnmrns = nil
				cargsr.argstg = argUnknown
				return
			}
		}
	}
}

func (cargsr *contentargsreader) Close() (err error) {
	if cargsr != nil {
		args, argvbf := cargsr.args, cargsr.argvbf
		cargsr.args = nil
		cargsr.argnmrns = nil
		cargsr.argvbf = nil
		for argk, argv := range args {
			if argbf, _ := argv.(*iorw.Buffer); !argbf.Empty() {
				argbf.Close()
			}
			delete(args, argk)
		}
		if !argvbf.Empty() {
			argvbf.Close()
		}
	}
	return
}

func newContentArgsReader(preflabel, postlabel string, cntntrdr *contenteventreader) (cargsr *contentargsreader) {
	if cntntrdr == nil || preflabel == "" || postlabel == "" {
		return
	}
	cargsr = &contentargsreader{ParseEventReader: newParseEventReader(preflabel, postlabel), cntntrdr: cntntrdr}
	prvgr := rune(0)

	cargsr.PreRunesEvent = func(reset bool, rnsl int, rns ...rune) (rnserr error) {
		for _, r := range rns {
			if cargsr.argstg == argValue {
				if cargsr.argtxtr == 0 {
					if cargsr.argprvr != '\\' && iorw.IsTxtPar(r) {
						cargsr.argtxtr = r
						cargsr.argprvr = 0
						continue
					}
					cargsr.argprvr = r
					continue
				}
				if cargsr.argtxtr == r {
					if cargsr.argprvr != '\\' {
						cargsr.argtxtr = 0
						cargsr.savearg()
						continue
					}
				}
				cargsr.WriteAtvRunes(r)
			} else if cargsr.argstg == argUnknown {
				if iorw.IsSpace(r) {

					continue
				}
				if r == '=' {
					if len(cargsr.argnmrns) > 0 {
						cargsr.argstg = argValue
						continue
					}
				}
				if validElemChar(prvgr, r) {
					cargsr.argnmrns = append(cargsr.argnmrns, r)
					prvgr = r
					continue
				}
			}
		}
		return
	}

	cargsr.PreResetEvent = func(prel, postl int, prelbl, postlbl []rune, lbli []int) (reseterr error) {
		if cargsr.argstg == argUnknown {
			if len(cargsr.argnmrns) > 0 {
				cargsr.argnmrns = nil
			}
			cargsr.argprvr = 0
			cargsr.argtxtr = 0
		}
		return
	}

	cargsr.PostRunesEvent = func(reset bool, rnsl int, rns ...rune) (rnserr error) {
		cargsr.WriteAtvRunes(rns...)
		return
	}

	cargsr.PostResetEvent = func(prel, postl int, prelbl, postlbl []rune, lbli []int) (reseterr error) {
		cargsr.savearg()
		return
	}
	return
}
