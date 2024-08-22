package webapp

import (
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/lnksnk/lnksnk/fsutils"
	"github.com/lnksnk/lnksnk/iorw"
	"github.com/lnksnk/lnksnk/listen"
	"github.com/lnksnk/lnksnk/resources"
	"github.com/lnksnk/lnksnk/screen"
	"github.com/lnksnk/lnksnk/serve"

	webview "github.com/webview/webview_go"
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
	if w := webview.New(true); w != nil {
		defer w.Destroy()
		width, height := screen.Size()
		w.SetSize(width, height, webview.HintMax)
		w.SetTitle("LNKSNK - WEBAPP")
		var wepappmap map[string]interface{} = nil
		wepappmap = map[string]interface{}{
			"webapp": map[string]interface{}{
				"navigate": func(nav string) {
					if prtci := strings.Index(nav, "://"); prtci > -1 {
						proto := nav[:prtci]
						nav = nav[prtci+len("://"):]
						if proto == "http" || proto == "https" {
							w.Navigate(proto + "://" + nav)
						}
						return
					}
					if nav != "" {
						bufout := iorw.NewBuffer()
						defer bufout.Close()
						serve.ProcessIORequest(nav, bufout, nil, wepappmap)
						bufr := bufout.Reader(true)
						w.SetHtml(bufr.SubString(bufout.IndexOf("\r\n\r\n") + int64(len("\r\n\r\n")+1)))
						return
					}
				},
				"host":     "",
				"port":     "",
				"setTitle": w.SetTitle,
				"maximize": func() {
					w.SetSize(width, height, webview.HintNone)

				},
				"fullScreen": func() {

					w.Eval(`try{ openFullscreen(document.documentElement);} catch(e){
						var openFullscreen=(elem)=>{
  if (elem.requestFullscreen) {
    elem.requestFullscreen();
  } else if (elem.webkitRequestFullscreen) { /* Safari */
    elem.webkitRequestFullscreen();
  } else if (elem.msRequestFullscreen) { /* IE11 */
    elem.msRequestFullscreen();
  }
}
  openFullscreen(document.documentElement);
					}`)
				},
				"restoreScreen": func() {
					w.Eval(`try{ closeFullscreen();} catch(e){
						var closeFullscreen=()=>{
  if (document.exitFullscreen) {
    document.exitFullscreen();
  } else if (document.webkitExitFullscreen) { /* Safari */
    document.webkitExitFullscreen();
  } else if (document.msExitFullscreen) { /* IE11 */
    document.msExitFullscreen();
  }
}
  openFullscreen(document.documentElement);
					}`)
				},
			},
		}
		if err := serve.ProcessRequestPath("/active:"+appEnvPath[1:]+appName+".conf.js", wepappmap); err != nil {
			println(err.Error())
		}
		w.Run()
		done <- os.Interrupt
	} else {
		done <- os.Interrupt
	}

	<-done
	serve.ProcessRequestPath("/active:"+appName+".fin.js", nil, &fs)

	listen.ShutdownAll()
}
