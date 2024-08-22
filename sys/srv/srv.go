package srv

import (
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/lnksnk/lnksnk/fsutils"
	"github.com/lnksnk/lnksnk/iorw"
	"github.com/lnksnk/lnksnk/listen"
	"github.com/lnksnk/lnksnk/logging"
	"github.com/lnksnk/lnksnk/resources"
	"github.com/lnksnk/lnksnk/serve"
	"github.com/lnksnk/lnksnk/service"
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
	Config     *service.Config
	appPath    string
	logger     logging.Logger
	appEnvPath string
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
	fs := fsutils.NewFSUtils()
	if p.appEnvPath == "" {
		p.appEnvPath = "./"
	}
	if fs.EXISTS(p.appEnvPath) {
		resources.GLOBALRSNG().FS().MKDIR("/"+p.Config.Name+"/env", p.appEnvPath)
		p.appEnvPath = "/" + p.Config.Name + "/env/"
	}
	listen.DefaultHandler = http.HandlerFunc(serve.ServeHTTPRequest)
	serve.LISTEN = listen.NewListen(nil)

	var conflabel = "conf"
	var configjs = p.Config.Name
	p.logger.Info("init run")
	serve.ProcessRequestPath("/active:"+p.appEnvPath[1:]+configjs+".init.js", nil)

	p.logger.Info("load ", p.appEnvPath+configjs+".init.js")
	p.logger.Info("start run")
	serve.ProcessRequestPath("/active:"+p.appEnvPath[1:]+configjs+"."+conflabel+".js", nil)
	p.logger.Info("load ", p.appEnvPath+configjs+"."+conflabel+".js")

}

func (p *program) Stop(s service.Service) error {
	resources.GLOBALRSNG().FS().MKDIR("/", p.appPath)
	var conflabel = "fin"
	p.logger.Info("stop run")
	var configjs = p.Config.Name
	serve.ProcessRequestPath("/active:"+p.appEnvPath[1:]+configjs+"."+conflabel+".js", nil)
	p.logger.Info("load ", p.appEnvPath+configjs+"."+conflabel+".js")
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
	ai, al := 0, len(args)
	cntrlcmd := ""
	for ai < al {
		if strings.Contains(",install,uninstall,start,stop,", ","+args[ai]+",") {
			cntrlcmd = args[ai]
			args = append(args[:ai], args[ai+1:]...)
			al--
			continue
		}
		if strings.EqualFold(args[ai], "-env-path") {
			args = append(args[:ai], args[ai+1:]...)
			al--
			if ai < al {
				if args[ai] != "" {
					prg.appEnvPath = strings.TrimFunc(args[ai], iorw.IsSpace)
				}
				args = append(args[:ai], args[ai+1:]...)
				al--
			}
		}
		ai++
	}
	if 0 < al && prg.appEnvPath == "" {
		prg.appEnvPath = strings.Replace(args[0], "\\", "/", -1)
		if si := strings.LastIndex(prg.appEnvPath, "/"); si > -1 {
			prg.appEnvPath = prg.appEnvPath[:si]
		}
		args = args[1:]
		al--
	}
	if cntrlcmd != "" {
		if err := service.Control(s, cntrlcmd); err != nil {
			prg.logger.Error(err.Error())
			return
		}
	}
	err = s.Run()
	if err != nil {
		prg.logger.Error(err.Error())
	}
}
