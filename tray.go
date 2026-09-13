package main

import (
	_ "embed"

	"myinternetvpn/client/internal/defaults"

	"github.com/getlantern/systray"
)

//go:embed resources/images/app.ico
var trayIcon []byte

//go:embed resources/update/ed25519_public.key
var updatePublicKey []byte

func (a *App) startTray() {
	go func() {
		systray.Run(a.onTrayReady, func() {})
	}()
}

func (a *App) onTrayReady() {
	systray.SetIcon(trayIcon)
	systray.SetTitle(defaults.ProductName)
	systray.SetTooltip(defaults.ProductName)

	mShow := systray.AddMenuItem("Show", "Show main window")
	mToggle := systray.AddMenuItem("Connect / Disconnect", "Toggle VPN connection")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Quit", "Exit "+defaults.ProductName)

	go func() {
		for {
			select {
			case <-mShow.ClickedCh:
				a.ShowWindow()
			case <-mToggle.ClickedCh:
				if err := a.ToggleConnect(); err != nil && a.log != nil {
					a.log.Warn("tray toggle: %v", err)
				}
			case <-mQuit.ClickedCh:
				systray.Quit()
				a.QuitApp()
				return
			case <-a.ctx.Done():
				systray.Quit()
				return
			}
		}
	}()
}
