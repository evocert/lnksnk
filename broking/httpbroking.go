package broking

import (
	"bufio"
	"io"
	"net/http"

	"github.com/evocert/lnksnk/serve"
	"github.com/evocert/lnksnk/stdio"
)

func HttpBroker(fin io.ReadCloser, fout io.WriteCloser, exitphrase string, canExit func(string, stdio.ReaderPrinter) (bool, error), doneBroking func() error) (err error) {
	stdrdrprntr := stdio.NewStdioReaderWriter(fin, 0, 0, fout, func() error {
		return nil
	}, func() error {
		return nil
	}, func() error {

		return nil
	})
	if canExit == nil {
		canExit = func(testphrase string, stdrdprnt stdio.ReaderPrinter) (doExit bool, err error) {
			if ln, lnerr := stdrdprnt.Readln(); lnerr != nil {
				err = lnerr
			} else {
				doExit = ln == testphrase
			}
			return
		}
	}
	for {
		if req, reqerr := http.ReadRequest(bufio.NewReader(stdrdrprntr)); reqerr == nil {
			if req != nil {
				rew := serve.NewResponseWriter(req, bufio.NewWriter(stdrdrprntr))
				func() {
					serve.ServeHTTPRequest(rew, req)
				}()
			}
		}
		if exitphrase == "" {
			break
		} else {
			if canExit != nil {
				if doExit, exiterr := canExit(exitphrase, stdrdrprntr); exiterr == nil {
					if doExit {
						break
					}
				} else if exiterr != nil {
					break
				}
			} else {
				break
			}
		}
	}
	if doneBroking != nil {
		err = doneBroking()
	}
	return
}
