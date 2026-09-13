package main

import (
	"context"
	"embed"
	"log"
	"os"

	"myinternetvpn/client/internal/defaults"
	"myinternetvpn/client/internal/winutil"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// Set via: go build -tags "desktop,production" -ldflags "-w -s -H windowsgui -X main.version=<sha>"
var version = "dev"

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Always run elevated (TUN / firewall / routes). Manifest also requests admin;
	// this covers builds without embedded requireAdministrator.
	okAdmin, err := winutil.EnsureAdmin()
	if err != nil {
		winutil.MessageBox(defaults.WindowTitle, "Нужны права администратора.\nЗапустите приложение от имени администратора.", true)
		os.Exit(1)
	}
	if !okAdmin {
		// Elevated child process was started; exit this unelevated copy.
		os.Exit(0)
	}

	ok, err := winutil.AcquireSingleInstance()
	if err != nil {
		log.Printf("single-instance warning: %v", err)
	}
	if !ok {
		winutil.HandleSecondInstance(defaults.WindowTitle)
		os.Exit(0)
	}
	defer winutil.ReleaseSingleInstance()

	app := NewApp()

	err = wails.Run(&options.App{
		Title:     defaults.WindowTitle,
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
		HideWindowOnClose: false,
		OnBeforeClose: func(ctx context.Context) bool {
			if app.ShouldCloseToTray() {
				runtime.WindowHide(ctx)
				return true
			}
			return false
		},
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
