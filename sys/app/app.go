package app

import (
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/evocert/lnksnk/fsutils"
	"github.com/evocert/lnksnk/iorw"
	"github.com/evocert/lnksnk/listen"
	"github.com/evocert/lnksnk/resources"
	"github.com/evocert/lnksnk/serve"
)

func appName(args ...string) (appname string) {
	if len(args) == 0 {
		args = os.Args
	}
	appname = strings.Replace(args[0], "\\", "/", -1)
	appname = appname[strings.LastIndex(appname, "/")+1:]
	if strings.Contains(appname, "__debug_bin") {
		if strings.Contains(appname, ".") {
			appname = appname[:strings.LastIndex(appname, "__debug_bin")+len("__debug_bin")] + appname[strings.LastIndex(appname, "."):]
		} else {
			appname = appname[:strings.LastIndex(appname, "__debug_bin")+len("__debug_bin")]
		}
	}
	if lsti := strings.LastIndex(appname, "."); lsti > 0 {
		appname = appname[:lsti]
	}
	return
}

func App(args ...string) {
	fs := fsutils.NewFSUtils()
	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)
	listen.DefaultHandler = http.HandlerFunc(serve.ServeHTTPRequest)
	serve.LISTEN = listen.NewListen(nil)
	var appEnvPath = ""
	ai, al := 0, len(args)
	for ai < al {
		if strings.EqualFold(args[ai], "-env-path") {
			args = append(args[:ai], args[ai+1:]...)
			al--
			if ai < al {
				if args[ai] != "" {
					appEnvPath = strings.TrimFunc(args[ai], iorw.IsSpace)
				}
				args = append(args[:ai], args[ai+1:]...)
				al--
			}
		}
		ai++
	}
	if appEnvPath == "" {
		appEnvPath = "./"
	}
	appName := appName()
	if fs.EXISTS(appEnvPath) {
		resources.GLOBALRSNG().FS().MKDIR("/"+appName+"/env", appEnvPath)
		appEnvPath = "/" + appName + "/env/"
	}
	if err := serve.ProcessRequestPath("/active:"+appEnvPath[1:]+appName+".conf.js", nil); err != nil {
		println(err.Error())
	}
	<-done
	serve.ProcessRequestPath("/active:"+appName+".fin.js", nil, &fs)
	listen.ShutdownAll()
}
