package srv

import (
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/evocert/lnksnk/listen"
	"github.com/evocert/lnksnk/logging"
	"github.com/evocert/lnksnk/resources"
	"github.com/evocert/lnksnk/serve"
	"github.com/evocert/lnksnk/service"
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

type program struct {
	Config  *service.Config
	appPath string
	logger  logging.Logger
}

func (p *program) Start(s service.Service) error {
	// Start should not block. Do the actual work async.
	go p.run()
	return nil
}

func (p *program) prepConfig(args ...string) {
	if p != nil {
		if len(args) == 0 {
			args = os.Args
		}
		if p.appPath == "" {
			p.appPath = strings.Replace(args[0], "\\", "/", -1)
			if strings.LastIndex(p.appPath, "/") > 0 {
				p.appPath = p.appPath[:strings.LastIndex(p.appPath, "/")]
			}
		}
		if p.Config == nil {
			p.Config = &service.Config{}
		}
		if p.Config.Name == "" {
			p.Config.Name = appName(args...)
			if p.Config.DisplayName == "" {
				p.Config.DisplayName = p.Config.Name
			}
		}
		if p.logger == nil {
			p.logger, _ = logging.GLOBALLOGGING().Register(p.Config.Name, "/logs", p.appPath)
		}
	}
}

func (p *program) run() {
	listen.DefaultHandler = http.HandlerFunc(serve.ServeHTTPRequest)
	serve.LISTEN = listen.NewListen(nil)

	var conflabel = "conf"
	var configjs = p.Config.Name
	p.logger.Info("init run")
	serve.ProcessRequestPath("/active:"+configjs+".init.js", nil)

	p.logger.Info("load ", configjs+".init.js")
	p.logger.Info("start run")
	serve.ProcessRequestPath("/active:"+configjs+"."+conflabel+".js", nil)
	p.logger.Info("load ", configjs+"."+conflabel+".js")

}

func (p *program) Stop(s service.Service) error {
	resources.GLOBALRSNG().FS().MKDIR("/", p.appPath)
	var conflabel = "fin"
	p.logger.Info("stop run")
	var configjs = p.Config.Name
	serve.ProcessRequestPath("/active:"+configjs+"."+conflabel+".js", nil)
	p.logger.Info("load ", configjs+"."+conflabel+".js")
	return nil
}

func Serve(args ...string) {
	prg := &program{Config: &service.Config{
		Name:        "",
		DisplayName: "",
		Description: "",
	}}
	prg.prepConfig()
	s, err := service.New(prg, prg.Config)
	if err != nil {
		log.Fatal(err)
	}
	if err != nil {
		log.Fatal(err)
	}
	prg.logger.Info("process args")
	if len(args) > 1 {
		for _, arg := range args {
			if strings.Contains(",install,uninstall,start,stop,", ","+arg+",") {
				if err := service.Control(s, arg); err != nil {
					prg.logger.Error(err.Error())
					return
				} else {
					return
				}
			}
		}
	}

	err = s.Run()
	if err != nil {
		prg.logger.Error(err.Error())
	}
}
