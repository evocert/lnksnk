package builds

import (
	_ "embed"
	"fmt"
	"io"

	"github.com/evocert/lnksnk/iorw"
	"github.com/evocert/lnksnk/resources"
	"github.com/evocert/lnksnk/stdio/command"
)

//go:embed dist.json
var distjson string

func init() {
	gblfs := resources.GLOBALRSNG().FS()
	gblfs.MKDIR("/go-utils")
	gblfs.SET("/go-utils/distros.json", distjson)
}

func BuildGoApp(
	goos string,
	goarch string,
	cgoSupported bool,
	firstClass bool,
	ldflags string,
	codesourcepath string,
	appdestinationpath string,
	appname string, out io.Writer, errout io.Writer) {
	if ldflags == "" {
		ldflags = "-s -w"
	}

	env := []string{"GOOS=" + goos, "GOARCH=" + goarch}
	if cgoSupported {
		env = append(env, "CGO_ENABLED=1")
	} else {
		env = append(env, "CGO_ENABLED=0")
	}
	if cmd, cmderr := command.NewCommand("go", env, "build", "-C", codesourcepath, "-a", "-v", `-ldflags`, ldflags, "-o", appdestinationpath+appname); cmderr == nil {
		defer cmd.Close()
		if cmderr = cmd.Wait(); cmderr == nil {
			if out != nil {
				iorw.Fprint(out, cmd.Out())
			}
			if errout != nil {
				iorw.Fprint(errout, cmd.Err())
			}
		} else {
			if errout != nil {
				iorw.Fprint(errout, cmderr.Error())
			} else if cmderr != nil {
				fmt.Println(cmderr.Error())
			}
		}
	}
}
