package main

import (
	"net"
	"net/http"
	"strings"

	"github.com/lnksnk/lnksnk/reflection/reflectrpc"
	"github.com/lnksnk/lnksnk/reflection/reflectrpc/rcpnet"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

func rpcserve(w http.ResponseWriter, req *http.Request) {
	rcpnet.RpcHttpServe(gblstack, w, req)
}

var gblstack = reflectrpc.NewRpcStack()

type TestObject struct {
	Title string
}

func (tstobj *TestObject) Hello(a ...string) string {
	return "hello from object" + strings.Join(a, ",")
}

func main() {
	tst := &TestObject{}
	tst.Title = "The Title"
	gblstack.Register("testobj", tst)
	gblstack.RpcItem("testobj").Field("Title")

	//http.HandleFunc("/", rpcserve)
	httpsrv := &http.Server{Handler: h2c.NewHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rpcserve(w, r)
	}), &http2.Server{})}
	if ln, lnerr := net.Listen("tcp", ":8090"); ln != nil && lnerr == nil {
		httpsrv.Serve(ln)
	}

}
