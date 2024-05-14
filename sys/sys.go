package main

import (
	"os"
	"strings"

	_ "github.com/evocert/lnksnk/database/dbserve/connections"
	_ "github.com/evocert/lnksnk/database/dbserve/drivers"
	_ "github.com/evocert/lnksnk/database/dbserve/exec"
	_ "github.com/evocert/lnksnk/database/dbserve/query"
	_ "github.com/evocert/lnksnk/database/dbserve/register"
	_ "github.com/evocert/lnksnk/database/dbserve/status"
	_ "github.com/evocert/lnksnk/database/mssql"
	_ "github.com/evocert/lnksnk/database/mysql"
	_ "github.com/evocert/lnksnk/database/ora"
	_ "github.com/evocert/lnksnk/database/postgres"
	_ "github.com/evocert/lnksnk/fonts"
	"github.com/evocert/lnksnk/sys/app"
	"github.com/evocert/lnksnk/sys/srv"
	_ "github.com/evocert/lnksnk/ui"
)

func main() {

	args := os.Args
	if len(args) > 1 {
		if strings.EqualFold(args[1], "app") {
			args = append(args[:1], args[1:]...)
			app.App(args...)
			return
		}
	}
	srv.Serve(os.Args...)
}
