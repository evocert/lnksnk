package command

import (
	"bufio"
	"context"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/lnksnk/lnksnk/iorw"
	"github.com/lnksnk/lnksnk/stdio"
)

func removeLeadingDuplicates(env []string) (ret []string) {
	for i, e := range env {
		found := false
		if eq := strings.IndexByte(e, '='); eq != -1 {
			keq := e[:eq+1] // "key="
			for _, e2 := range env[i+1:] {
				if strings.HasPrefix(e2, keq) {
					found = true
					break
				}
			}
		}
		if !found {
			ret = append(ret, e)
		}
	}
	return
}

type Command struct {
	execmd    *exec.Cmd
	ctxcncl   context.CancelFunc
	Pid       int
	OnClose   func(int)
	stdin     io.WriteCloser
	bfin      *iorw.Buffer
	bfout     *iorw.Buffer
	bfrdr     *iorw.BuffReader
	rdrout    *tmpread
	bferrrdr  *iorw.BuffReader
	bferrout  *iorw.Buffer
	rdrerrout *tmpread
	wtrin     *tmpwrite
}

type tmpwrite struct {
	fnwrite func([]byte) (int, error)
	bfw     *iorw.Buffer
}

func (tmpw *tmpwrite) buffer() (buf *iorw.Buffer) {
	if tmpw != nil {
		if buf = tmpw.bfw; buf == nil {
			buf = iorw.NewBuffer()
			tmpw.bfw = buf
		}
	}
	return
}

func (tmpw *tmpwrite) Write(p []byte) (n int, err error) {
	if tmpw != nil {
		n, err = tmpw.buffer().Write(p)
	}
	return
}

func (tmpw *tmpwrite) Print(a ...interface{}) (err error) {
	if tmpw != nil {
		if err = tmpw.buffer().Print(a...); err == nil {
			err = tmpw.Flush()
		}
	}
	return
}

func (tmpw *tmpwrite) Println(a ...interface{}) (err error) {
	if tmpw != nil {
		if err = tmpw.buffer().Println(a...); err == nil {
			err = tmpw.Flush()
		}
	}
	return
}

func (tmpw *tmpwrite) Flush() (err error) {
	if tmpw != nil && tmpw.bfw != nil && tmpw.bfw.Size() > 0 {
		if tmpw.fnwrite != nil {
			_, err = iorw.WriteToFunc(tmpw.bfw.Clone(true).Reader(true), tmpw.fnwrite)
		}
	}
	return
}

func (tmpw *tmpwrite) Close() (err error) {
	if tmpw != nil {
		if tmpw.bfw != nil {
			tmpw.bfw.Close()
			tmpw.bfw = nil
		}
		if tmpw.fnwrite != nil {
			tmpw.fnwrite = nil
		}
	}
	return
}

type tmpread struct {
	fnread func([]byte) (int, error)
	bfr    *bufio.Reader
}

func (tmpr *tmpread) Read(p []byte) (n int, err error) {
	if tmpr != nil && tmpr.fnread != nil {
		n, err = tmpr.fnread(p)
	}
	return
}

func (tmpr *tmpread) ReadRune() (r rune, size int, err error) {
	if bfr := tmpr.bfr; bfr == nil {
		bfr = bufio.NewReaderSize(tmpr, 1)
		tmpr.bfr = bfr
		r, size, err = bfr.ReadRune()
	} else {
		r, size, err = bfr.ReadRune()
	}
	return
}

func (tmpr *tmpread) Readln(noef ...bool) (ln string, err error) {
	if tmpr != nil {
		if ln, err = iorw.ReadLine(tmpr); len(noef) > 0 && noef[0] && err == io.EOF {
			err = nil
		}
	}
	return
}

func (tmpr *tmpread) ReadLines() (lns []string, err error) {
	if tmpr != nil {
		if lns, err = iorw.ReadLines(tmpr); err == io.EOF {
			err = nil
		}
	}
	return
}

func (tmpr *tmpread) ReadAll(noef ...bool) (all string, err error) {
	if tmpr != nil {
		if all, err = iorw.ReaderToString(tmpr); len(noef) > 0 && noef[0] && err == io.EOF {
			err = nil
		}
	}
	return
}

func (tmpr *tmpread) Close() (err error) {
	if tmpr != nil {
		if tmpr.fnread != nil {
			tmpr.fnread = nil
		}
		if tmpr.bfr != nil {
			tmpr.bfr = nil
		}
	}
	return
}

func (cmd *Command) In() (wtr stdio.Printer) {
	if cmd != nil {
		if cmd.wtrin == nil {
			cmd.wtrin = &tmpwrite{fnwrite: func(b []byte) (int, error) {
				return cmdwrite(cmd, b)
			}}
			wtr = cmd.wtrin
		} else {
			wtr = cmd.wtrin
		}
	}
	return
}

func (cmd *Command) Out() (rdr stdio.Reader) {
	if cmd != nil {
		if cmd.rdrout == nil {
			cmd.rdrout = &tmpread{fnread: func(b []byte) (int, error) {
				return cmdread(cmd, b)
			}}
			rdr = cmd.rdrout
		} else {
			rdr = cmd.rdrout
		}
	}
	return
}

func (cmd *Command) Err() (rdr stdio.Reader) {
	if cmd != nil {
		if cmd.rdrerrout == nil {
			cmd.rdrerrout = &tmpread{fnread: func(b []byte) (int, error) {
				return cmderrread(cmd, b)
			}}
			rdr = cmd.rdrerrout
		} else {
			rdr = cmd.rdrerrout
		}
	}
	return
}

func cmdwrite(cmd *Command, p []byte) (n int, err error) {
	if cmd != nil {
		if cmd.stdin != nil {
			n, err = cmd.stdin.Write(p)
		}
	}
	return
}

func cmdread(cmd *Command, p []byte) (n int, err error) {
	if cmd != nil {
		if cmd.bfrdr == nil {
			if cmd.bfout != nil && cmd.bfout.Size() > 0 {
				cmd.bfrdr = cmd.bfout.Clone(true).Reader(true)
				n, err = cmdread(cmd, p)
				return
			}
		} else {
			if n, err = cmd.bfrdr.Read(p); err == io.EOF {
				cmd.bfrdr = nil
				n, err = cmdread(cmd, p)
				return
			}
		}
		if n == 0 && err == nil {
			err = io.EOF
		}
	}
	return
}

func cmderrread(cmd *Command, p []byte) (n int, err error) {
	if cmd != nil {
		if cmd.bferrrdr == nil {
			if cmd.bferrout != nil && cmd.bferrout.Size() > 0 {
				cmd.bferrrdr = cmd.bferrout.Clone(true).Reader(true)
				n, err = cmderrread(cmd, p)
				return
			}
		} else {
			if n, err = cmd.bferrrdr.Read(p); err == io.EOF {
				cmd.bferrrdr = nil
				n, err = cmderrread(cmd, p)
				return
			}
		}
		if n == 0 && err == nil {
			err = io.EOF
		}
	}
	return
}

func (cmd *Command) CommandExecuted() (cmds string) {
	if cmd != nil && cmd.execmd != nil {
		cmds = cmd.execmd.String()
	}
	return
}

func (cmd *Command) Wait() (err error) {
	if cmd != nil && cmd.execmd != nil {
		if err = cmd.execmd.Wait(); err == io.EOF {
			cmd.ctxcncl = nil
			return nil
		} else if cmd.ctxcncl != nil {
			cmd.ctxcncl()
			cmd.ctxcncl = nil
		}
	}
	return
}

var testHookStartProcess func(*os.Process)

func NewCommand(path string, env []string, args ...string) (cmd *Command, err error) {
	env = append(os.Environ(), env...)
	ctx, ctxcncl := context.WithCancel(context.Background())
	execmd := exec.CommandContext(ctx, path, args...)
	execmd.Env = removeLeadingDuplicates(env)
	bfout := iorw.NewBuffer()
	bferrout := iorw.NewBuffer()
	execmd.Stdout = bfout
	execmd.Stderr = bferrout

	if stdin, stdinerr := execmd.StdinPipe(); stdinerr == nil {
		if err = execmd.Start(); err == nil {
			if hook := testHookStartProcess; hook != nil {
				hook(execmd.Process)
			}
			cmd = &Command{execmd: execmd, ctxcncl: ctxcncl, Pid: execmd.Process.Pid, bfout: bfout, bferrout: bferrout, stdin: stdin}
		}
	} else {
		err = stdinerr
	}
	ctxcncl = nil
	return
}

func (cmd *Command) Close() (err error) {
	if cmd != nil {
		if cmd.bfout != nil {
			cmd.bfout.Close()
			cmd.bfout = nil
		}
		if cmd.bfrdr != nil {
			cmd.bfrdr.Close()
			cmd.bfrdr = nil
		}
		if cmd.rdrout != nil {
			cmd.rdrout.Close()
			cmd.rdrout = nil
		}

		if cmd.bferrout != nil {
			cmd.bferrout.Close()
			cmd.bferrout = nil
		}
		if cmd.bferrrdr != nil {
			cmd.bferrrdr.Close()
			cmd.bferrrdr = nil
		}
		if cmd.rdrerrout != nil {
			cmd.rdrerrout.Close()
			cmd.rdrerrout = nil
		}
		if cmd.wtrin != nil {
			cmd.wtrin.Close()
			cmd.wtrin = nil
		}
		if cmd.stdin != nil {
			cmd.stdin.Close()
			cmd.stdin = nil
		}

		if cmd.execmd != nil {
			if cmd.execmd.ProcessState == nil {
				if waiterr := cmd.execmd.Wait(); waiterr != nil {
					if cmd.execmd.Process != nil {
						if waiterr = cmd.execmd.Process.Release(); waiterr != nil {
							cmd.execmd.Process.Kill()
							if cmd.ctxcncl != nil {
								cmd.ctxcncl = nil
							}
						} else {
							if cmd.ctxcncl != nil {
								cmd.ctxcncl = nil
							}
						}
					}
				} else {
					if cmd.ctxcncl != nil {
						cmd.ctxcncl = nil
					}
				}
			} else {
				if waiterr := cmd.execmd.Process.Release(); waiterr != nil {
					cmd.execmd.Process.Kill()
					if cmd.ctxcncl != nil {
						cmd.ctxcncl = nil
					}
				}
			}
			if cmd.ctxcncl != nil {
				cmd.ctxcncl()
				cmd.ctxcncl = nil
			}
			cmd.execmd = nil
		}
		if cmd.OnClose != nil {
			cmd.OnClose(cmd.Pid)
			cmd.OnClose = nil
		}
	}
	return
}
