package builds

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/evocert/lnksnk/iorw"
)

func BuildGoAppDistribution(ctx context.Context, sourcepath, destinationpath, destappname string) {
	var distdefs = []interface{}{}
	if err := json.Unmarshal([]byte(distjson), &distdefs); err == nil {
		go func() {
			for _, distdef := range distdefs {
				if argcpump, _ := distdef.(map[string]interface{}); len(argcpump) > 0 {
					var goos, _ = argcpump["GOOS"].(string)
					var goarch, _ = argcpump["GOARCH"].(string)
					var cgoSupported, _ = argcpump["CgoSupported"].(bool)
					var firstClass, _ = argcpump["FirstClass"].(bool)
					bfrslt := iorw.NewBuffer()
					bferrrslt := iorw.NewBuffer()
					BuildGoApp(goos, goarch, cgoSupported, firstClass,
						"-s -w", sourcepath, destinationpath, func() (appname string) {
							appname = destappname
							appname = appname + "_" + goos + "_" + goarch
							if goos == "windows" {
								appname = appname + ".exe"
							} else if goos == "js" {
								appname = appname + ".wasm"
							}
							return
						}(), bfrslt, bferrrslt)
					if bfrslt.Size() > 0 {
						fmt.Println("[info ", goos, "/", goarch, "] ", bfrslt)
						bfrslt.Close()
					}
					if bferrrslt.Size() > 0 {
						fmt.Println("[error ", goos, "/", goarch, "] ", bferrrslt)
						bferrrslt.Close()
					}
				}
			}
		}()
	}
}

func init() {
	//buildGoAppDistribution(nil, "C:/projects/lnksnknext/app", "C:/projects/lnksnknext/builds/dist/", "lnksnk")
	/*var distdefs = []interface{}{}
	if err := json.Unmarshal([]byte(distjson), &distdefs); err == nil {
		go func() {
			for _, distdef := range distdefs {
				if argcpump, _ := distdef.(map[string]interface{}); len(argcpump) > 0 {
					var goos, _ = argcpump["GOOS"].(string)
					var goarch, _ = argcpump["GOARCH"].(string)
					var cgoSupported, _ = argcpump["CgoSupported"].(bool)
					var firstClass, _ = argcpump["FirstClass"].(bool)
					bfrslt := iorw.NewBuffer()
					bferrrslt := iorw.NewBuffer()
					BuildGoApp(goos, goarch, cgoSupported, firstClass,
						"-s -w", "C:/projects/lnksnknext/app", "C:/projects/lnksnknext/builds/dist/", func() (appname string) {
							appname = "lnksnk"
							appname = appname + "_" + goos + goarch
							if goos == "windows" {
								appname = appname + ".exe"
							} else if goos == "js" {
								appname = appname + ".wasm"
							}
							return
						}(), bfrslt, bferrrslt)
					if bfrslt.Size() > 0 {
						fmt.Println("[info ", goos, "/", goarch, "] ", bfrslt)
						bfrslt.Close()
					}
					if bferrrslt.Size() > 0 {
						fmt.Println("[error ", goos, "/", goarch, "] ", bferrrslt)
						bferrrslt.Close()
					}
				}
			}
		}()
	}*/
}
