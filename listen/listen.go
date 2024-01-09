package listen

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"strings"

	"github.com/evocert/lnksnk/concurrent"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

var listerens = concurrent.NewMap()

func ShutdownAll() {
	listerens.Range(func(key, value any) bool {
		if lstnr, _ := value.(*http.Server); lstnr != nil {
			lstnr.Shutdown(context.Background())
			fmt.Println("Shutdown - ", key)
		}
		return true
	})
}

func Shutdown(keys ...interface{}) {
	if len(keys) > 0 {
		keys = append(keys, func(delkeys []interface{}, delvalues []interface{}) {
			for kn, k := range delkeys {
				if lstnr, _ := delvalues[kn].(*http.Server); lstnr != nil {
					lstnr.Shutdown(context.Background())
					fmt.Println("Shutdown - ", k)
				}
			}
		})
		listerens.Del(keys...)
	}
}

type listen struct {
	handler http.Handler
}

func (lsnt *listen) Serve(network string, addr string, tlsconf ...*tls.Config) {
	if lsnt != nil {
		/*if hndlr := lsnt.handler; hndlr != nil {
			Serve(network, addr, hndlr, tlsconf...)
		} else if DefaultHandler != nil {
			Serve(network, addr, DefaultHandler, tlsconf...)
		}*/
		Serve(network, addr, lsnt.handler, tlsconf...)
	}
}

func (lsnt *listen) Shutdown(keys ...interface{}) {
	if lsnt != nil {
		Shutdown(keys...)
	}
}

func NewListen(handerfunc http.HandlerFunc) *listen {
	if handerfunc == nil && DefaultHandler != nil {
		handerfunc = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			DefaultHandler.ServeHTTP(w, r)
		})
	}
	return &listen{handler: handerfunc}
}

func Serve(network string, addr string, handler http.Handler, tlsconf ...*tls.Config) {
	if strings.Contains(network, "tcp") {
		if handler == nil && DefaultHandler != nil {
			handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				DefaultHandler.ServeHTTP(w, r)
			})
		}
		if handler != nil {
			if ln, err := Listen(network, addr); err == nil { //net.Listen(network, addr); err == nil {

				if tlsconfL := len(tlsconf); tlsconfL > 0 && tlsconf[0] != nil {
					ln = tls.NewListener(ln, tlsconf[0].Clone())
				}

				httpsrv := &http.Server{Handler: h2c.NewHandler(handler, &http2.Server{})}
				listerens.Set(ln.Addr().String(), httpsrv)
				go httpsrv.Serve(ln)
				return
			}
		}
	}
}

var DefaultHandler http.Handler = nil

var httpsrv = &http.Server{Handler: h2c.NewHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	if DefaultHandler != nil {
		DefaultHandler.ServeHTTP(w, r)
	}
}), &http2.Server{})}

func init() {
	go func() {
		//httpsrv.Serve(lstnr)
	}()
}
