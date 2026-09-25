package main

import (
	"context"
	_ "embed"
	"fmt"
	"time"

	"myinternetvpn/client/internal/defaults"
	"myinternetvpn/client/internal/format"

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

		// Append live rates if connected and fresh data available.
		if a.api != nil {
			if up, down, age, ok := a.api.CachedTraffic(); ok && age < 3*time.Second {
				tip += fmt.Sprintf(" · ↓%s ↑%s", format.FormatRate(down), format.FormatRate(up))
			}
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

// startTrayRefresh launches a background ticker to update the tray tooltip with live rates.
// Stops automatically when the app context is canceled or when VPN disconnects.
func (a *App) startTrayRefresh() {
	a.trayMu.Lock()
	// Already running? Don't spawn another ticker.
	if a.trayRefreshCancel != nil {
		a.trayMu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(a.ctx)
	a.trayRefreshCancel = cancel
	a.trayMu.Unlock()

	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// Only refresh if VPN is connected.
				if a.manager != nil && a.manager.Running() {
					a.refreshTrayStatus()
				} else {
					// Disconnected; stop the ticker.
					a.stopTrayRefresh()
					return
				}
			}
		}
	}()
}

// stopTrayRefresh stops the background tray tooltip refresher.
func (a *App) stopTrayRefresh() {
	a.trayMu.Lock()
	if a.trayRefreshCancel != nil {
		a.trayRefreshCancel()
		a.trayRefreshCancel = nil
	}
	a.trayMu.Unlock()
}
