package main

import (
	"embed"
	"log"
	"os"

	"myinternetvpn/client/internal/winutil"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

// Set via: go build -tags "desktop,production" -ldflags "-w -s -H windowsgui -X main.version=<sha>"
var version = "dev"

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	ok, err := winutil.AcquireSingleInstance()
	if err != nil {
		log.Printf("single-instance warning: %v", err)
	}
	if !ok {
		log.Println("MyInternetVPN is already running")
		os.Exit(0)
	}
	defer winutil.ReleaseSingleInstance()

	app := NewApp()
	hideOnClose := true

	err = wails.Run(&options.App{
		Title:     "MyInternetVPN",
		Width:     1100,
		Height:    720,
		MinWidth:  900,
		MinHeight: 600,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour:  &options.RGBA{R: 7, G: 20, B: 39, A: 255},
		OnStartup:         app.startup,
		OnShutdown:        app.shutdown,
		HideWindowOnClose: hideOnClose,
		Bind: []interface{}{
			app,
		},
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			DisableWindowIcon:   false,
		},
	})
	if err != nil {
		log.Fatal(err)
	}
}
