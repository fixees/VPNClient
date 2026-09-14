package main

import (
	_ "embed"
	"fmt"

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
	a.setTrayDisconnected()

	mShow := systray.AddMenuItem("Показать", "Открыть окно")
	mToggle := systray.AddMenuItem("Подключить", "Подключить / отключить VPN")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Выйти", "Закрыть "+defaults.ProductName)

	a.trayMu.Lock()
	a.trayToggle = mToggle
	a.trayReady = true
	a.trayMu.Unlock()
	a.refreshTrayStatus()

	go func() {
		for {
			select {
			case <-mShow.ClickedCh:
				a.ShowWindow()
			case <-mToggle.ClickedCh:
				if err := a.ToggleConnect(); err != nil && a.log != nil {
					a.log.Warn("tray toggle: %v", err)
				}
				a.refreshTrayStatus()
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

func (a *App) setTrayDisconnected() {
	systray.SetTooltip(defaults.ProductName + " — Отключено")
}

func (a *App) refreshTrayStatus() {
	a.trayMu.Lock()
	ready := a.trayReady
	toggle := a.trayToggle
	a.trayMu.Unlock()
	if !ready {
		return
	}

	connected := a.manager != nil && a.manager.Running()
	profile := ""
	node := ""
	if a.settings != nil {
		profile = a.settings.ActiveProfile
		node = a.settings.SelectedNode
	}

	if connected {
		tip := defaults.ProductName + " — Подключено"
		if profile != "" {
			tip += " · " + profile
		}
		if node != "" {
			tip += " · " + node
		}
		systray.SetTooltip(tip)
		if toggle != nil {
			toggle.SetTitle("Отключить")
			toggle.SetTooltip(fmt.Sprintf("Отключить VPN (%s)", profileOrDash(profile)))
		}
		return
	}

	systray.SetTooltip(defaults.ProductName + " — Отключено")
	if toggle != nil {
		toggle.SetTitle("Подключить")
		toggle.SetTooltip("Подключить VPN")
	}
}

func profileOrDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}
