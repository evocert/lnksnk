package rcpnet

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/evocert/lnksnk/iorw"
	"github.com/evocert/lnksnk/parameters"
	"github.com/evocert/lnksnk/reflection/reflectrpc"
)

func RpcHttpServe(rpcstck *reflectrpc.RpcStack, w http.ResponseWriter, r *http.Request) {
	if rpcstck != nil {
		var rpcpath = r.URL.Path
		var hcjck = strings.Contains(rpcpath, "/hjck/")
		if hcjck {
			rpcpath = strings.Replace(rpcpath, "/hjck/", "/", -1)
		}
		var proto = r.Proto
		if strings.Contains(proto, "/") {
			proto = proto[:strings.Index(proto, "/")]
		}
		var host = r.Host
		var protoversion = fmt.Sprintf("%d%s%d", r.ProtoMajor, "/", r.ProtoMinor)
		var genJSApi = strings.HasSuffix(rpcpath, ".js")
		if genJSApi && proto == "HTTP" {
			rpcpath = rpcpath[:len(rpcpath)-len("/api.js")]
			if rpcitm := rpcstck.RpcItem(rpcpath); rpcitm != nil {
				w.Header().Set("Content-Type", "application/javascript; charset=utf-8")

				iorw.Fprintln(w, `function Api(){`)
				iorw.Fprintln(w, `	this.host="`+host+`";`)
				iorw.Fprintln(w, `	this.protocol="`+proto+`";`)
				iorw.Fprintln(w, `	this.protocolversion="`+protoversion+`";`)
				iorw.Fprintln(w, `	this.rpcaddr="`+rpcitm.ObjAddr+`";`)
				/*
					function User(name, birthday) {
						this.name = name;
						this.birthday = birthday;

						// age is calculated from the current date and birthday
						Object.defineProperty(this, "age", {
						  get() {
							let todayYear = new Date().getFullYear();
							return todayYear - this.birthday.getFullYear();
						  }
						});
					  }
				*/
				if _, fields := rpcitm.Fields(); len(fields) > 0 {
					for _, field := range fields {
						if field != "" {
							iorw.Fprintln(w,
								strings.Replace(`
	this._<field>="";
	// <field>
	Object.defineProperty(this, "<field>", {
		get() {
		return this._<field>;
		},
		set(val) {
		this._<field>=val;
		}
	});`, `<field>`, field, -1))
						}
					}
				}
				if _, methods := rpcitm.Methods(); len(methods) > 0 {
					for _, method := range methods {
						if method != "" {
							iorw.Fprintln(w,
								strings.Replace(`
	function <method>(){

	}
	`, `<method>`, method, -1))
						}
					}
				}
				iorw.Fprintln(w, `}`)
			}
		} else {
			if rpcpath != "" {

				var params parameters.ParametersAPI = parameters.NewParameters()
				defer params.CleanupParameters()
				parameters.LoadParametersFromRawURL(params, r.URL.RawQuery)
				var rcpcall = ""
				if callsep := strings.LastIndex(rpcpath, "-"); callsep > -1 {
					rcpcall = rpcpath[callsep+1:]
					rpcpath = rpcpath[:callsep]
				}
				if strings.HasPrefix(rpcpath, "/") {
					rpcpath = rpcpath[1:]
				}
				if rpcpath != "" {
					if rcpcall != "" {
						if rpcitm := rpcstck.RpcItem(rpcpath); rpcitm != nil {
							w.Header().Set("Content-Type", "application/json; charset=utf-8")
							var args []interface{} = nil
							if stdkeys := params.StandardKeys(); len(stdkeys) > 0 {
								for _, argk := range stdkeys {
									if strings.EqualFold(argk, "args") {
										argsv := params.Parameter(argk)
										for _, arv := range argsv {
											args = append(args, arv)
										}
									}
								}
							}

							if called, result := rpcitm.Invoke(rcpcall, args...); called {
								enc := json.NewEncoder(w)
								enc.Encode(&result)
							} else if called, result := rpcitm.Field(rcpcall, args...); called {
								enc := json.NewEncoder(w)
								if len(result) == 1 {
									enc.Encode(&result[0])
								} else {
									enc.Encode(nil)
								}
							}
						}
					}
				}
				/*if call, args := params.StringParameter("call", ""), params.Parameter("args"); call != "" {

				}*/
			}
		}
	}
}
