package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"myinternetvpn/client/internal/api"
	"myinternetvpn/client/internal/applog"
	"myinternetvpn/client/internal/bundle"
	"myinternetvpn/client/internal/config"
	"myinternetvpn/client/internal/core"
	"myinternetvpn/client/internal/deeplink"
	"myinternetvpn/client/internal/defaults"
	"myinternetvpn/client/internal/geo"
	"myinternetvpn/client/internal/health"
	"myinternetvpn/client/internal/integrity"
	"myinternetvpn/client/internal/latency"
	"myinternetvpn/client/internal/netinfo"
	"myinternetvpn/client/internal/parse"
	"myinternetvpn/client/internal/paths"
	"myinternetvpn/client/internal/profiles"
	"myinternetvpn/client/internal/subscription"
	"myinternetvpn/client/internal/update"
	"myinternetvpn/client/internal/warp"
	"myinternetvpn/client/internal/winutil"

	"github.com/getlantern/systray"
	qrcode "github.com/skip2/go-qrcode"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App is the Wails-bound application facade.
type App struct {
	ctx      context.Context
	paths    paths.Resolver
	store    *profiles.Store
	manager  *core.Manager
	api      *api.Client
	settings *profiles.Settings
	log      *applog.Logger

	killSwitch *winutil.KillSwitch
	dnsGuard   *winutil.DNSLeakGuard
	health     *health.Checker
	subs       *subscription.Fetcher
	scheduler  *subscription.Scheduler
	proxySnap  winutil.SystemProxySnapshot
	proxyOn    bool

	ipMu      sync.Mutex
	cachedIP  string
	cachedAt  time.Time
	cachedVia bool // fetched through mixed proxy

	statusMu      sync.Mutex
	statusCache   map[string]any
	statusCacheAt time.Time
	isAdmin       bool

	logTailMu   sync.Mutex
	logTailText string
	logTailAt   time.Time
	logClientSt os.FileInfo
	logCoreSt   os.FileInfo

	trayMu     sync.Mutex
	trayReady  bool
	trayToggle *systray.MenuItem

	startupDeepLink string
	deeplinkMu      sync.Mutex

	mu sync.Mutex
}

func NewApp() *App {
	return &App{
		killSwitch: winutil.NewKillSwitch(),
		dnsGuard:   winutil.NewDNSLeakGuard(),
		subs:       subscription.NewFetcher(),
	}
}

// SetStartupDeepLink stores a myvpn:// URL received on process start.
func (a *App) SetStartupDeepLink(link string) {
	a.deeplinkMu.Lock()
	a.startupDeepLink = strings.TrimSpace(link)
	a.deeplinkMu.Unlock()
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.paths = paths.New()
	_ = a.paths.Ensure()

	if lg, err := applog.Open(filepath.Join(a.paths.DataDir(), "client.log")); err == nil {
		a.log = lg
		a.log.Info("starting %s %s", defaults.ProductName, version)
	}

	// Unpack embedded mihomo + geo into AppData (single-exe distribution).
	if err := a.ensureBundledCore(); err != nil {
		if a.log != nil {
			a.log.Error("extract bundled core: %v", err)
		}
		winutil.MessageBox(defaults.WindowTitle, "Не удалось подготовить ядро VPN.\nПереустановите приложение.\n\n"+err.Error(), true)
	} else if a.log != nil {
		a.log.Info("core ready: %s", a.paths.CoreBinary())
	}

	a.store = profiles.NewStore(a.paths.ProfilesFile())
	_ = a.store.Load()

	a.settings = profiles.DefaultSettings()
	settingsPath := a.paths.SettingsFile()
	if loaded, err := profiles.LoadSettings(settingsPath); err == nil {
		a.settings = loaded
	} else {
		// Persist generated local secret on first run.
		_ = profiles.SaveSettings(settingsPath, a.settings)
	}
	_ = a.applyAutostartSetting()

	a.api = api.NewClient(a.settings.ControllerURL, a.settings.Secret)
	a.manager = core.NewManager(core.Options{
		CorePath:      a.paths.CoreBinary(),
		WorkDir:       a.paths.CoreWorkDir(),
		ConfigPath:    a.paths.RuntimeConfig(),
		ControllerURL: a.settings.ControllerURL,
		Secret:        a.settings.Secret,
		MixedPort:     a.settings.MixedPort,
		API:           a.api,
	})

	interval := time.Duration(a.settings.HealthIntervalSec) * time.Second
	a.health = health.New(interval, func() bool {
		return a.manager != nil && a.manager.Running() && a.api.Healthy()
	}, func() error {
		if !a.settings.AutoReconnect {
			return nil
		}
		a.log.Warn("health check failed, reconnecting")
		_ = a.Disconnect()
		return a.Connect()
	})

	subEvery := time.Duration(a.settings.SubscriptionIntervalMin) * time.Minute
	a.scheduler = subscription.NewScheduler(subEvery, func(ctx context.Context) error {
		if !a.settings.AutoUpdateSubscriptions {
			return nil
		}
		n, err := a.syncSubscriptions(false)
		if a.log != nil {
			a.log.Info("subscription scheduler updated=%d err=%v", n, err)
		}
		return err
	})
	if a.settings.AutoUpdateSubscriptions {
		a.scheduler.Start()
	}
	a.startTray()

	// Recover from crash / hard power-off: sticky firewall rules + orphan WinINET proxy.
	a.recoverOrphanNetworkState()

	if err := winutil.RegisterURLProtocols(); err != nil {
		if a.log != nil {
			a.log.Warn("register URL protocols: %v", err)
		}
	} else if a.log != nil {
		a.log.Info("URL schemes registered: %s:// %s://", defaults.URLScheme, defaults.URLSchemeAlt)
	}

	// Windows Run key launches us with --autostart → connect if a profile is selected.
	if winutil.HasCLIFlag("--autostart") {
		go a.autostartConnect()
	}

	go a.consumeDeepLinks()
}

func (a *App) shutdown(ctx context.Context) {
	if a.scheduler != nil {
		a.scheduler.Stop()
	}
	if a.health != nil {
		a.health.Stop()
	}
	_ = a.Disconnect()
	if a.log != nil {
		a.log.Info("shutdown")
		_ = a.log.Close()
	}
}

// GetStatus returns connection and profile summary for the UI.
func (a *App) GetStatus() map[string]any {
	a.mu.Lock()
	settings := a.settings
	manager := a.manager
	apiClient := a.api
	store := a.store
	a.mu.Unlock()

	state := "disconnected"
	coreVersion := ""
	mode := ""
	up, down := int64(0), int64(0)
	totalUp, totalDown := int64(0), int64(0)
	if settings != nil {
		mode = settings.Mode
	}

	if manager != nil && apiClient != nil && manager.Running() {
		state = "connected"
		ctx, cancel := context.WithTimeout(context.Background(), defaults.APITimeout)
		live := apiClient.Live(ctx)
		cancel()
		coreVersion = live.Version
		if live.Mode != "" {
			mode = live.Mode
		}
		up, down = live.Up, live.Down
		totalUp, totalDown = live.TotalUp, live.TotalDown
	}

	active := ""
	list := []profiles.Profile{}
	if store != nil {
		list, _ = store.List()
	}
	var quota map[string]any
	subURL := ""
	lastSync := ""
	expireUnix := int64(0)
	if settings != nil && store != nil {
		active = settings.ActiveProfile
		if active != "" {
			if p, err := store.Get(active); err == nil {
				subURL = p.SubscriptionURL
				lastSync = p.LastSyncAt
				expireUnix = p.Quota.ExpireUnix
				quota = map[string]any{
					"upload":     p.Quota.Upload,
					"download":   p.Quota.Download,
					"total":      p.Quota.Total,
					"used":       p.Quota.Used(),
					"remaining":  p.Quota.Remaining(),
					"expireUnix": p.Quota.ExpireUnix,
					"expired":    p.Quota.Expired(),
				}
			}
		}
	}

	out := map[string]any{
		"state":                  state,
		"coreVersion":            coreVersion,
		"activeProfile":          active,
		"profileCount":           len(list),
		"speedUp":                up,
		"speedDown":              down,
		"totalUp":                totalUp,
		"totalDown":              totalDown,
		"subscriptionURL":        subURL,
		"subscriptionQuota":      quota,
		"subscriptionLastSync":   lastSync,
		"subscriptionExpireUnix": expireUnix,
		"isAdmin":                winutil.IsAdmin(),
		"product":                defaults.ProductName,
		"appVersion":             version,
		"publicIP":               a.cachedPublicIPMasked(state == "connected"),
		"publicIPReady":          a.hasFreshPublicIP(state == "connected"),
		"warpEnabled":            settings != nil && settings.WARPEnabled,
		"warpMode":               "",
	}
	if settings != nil {
		out["warpMode"] = settings.WARPMode
		if settings.WARPMode == "" {
			out["warpMode"] = defaults.WARPModeViaProxy
		}
	}
	if settings != nil {
		out["mixedPort"] = settings.MixedPort
		out["mode"] = mode
		out["tun"] = settings.TUN
		out["useSystemProxy"] = settings.UseSystemProxy
		out["killSwitch"] = settings.KillSwitch
		out["dnsLeakProtection"] = settings.DNSLeakProtection
		out["autoReconnect"] = settings.AutoReconnect
		out["autoUpdateSubscriptions"] = settings.AutoUpdateSubscriptions
		out["selectedNode"] = settings.SelectedNode
		out["proxyGroup"] = a.proxyGroup()
		out["autostart"] = settings.Autostart
		out["closeToTray"] = settings.CloseToTray
	}
	return out
}

func (a *App) clearPublicIPCache() {
	a.ipMu.Lock()
	a.cachedIP = ""
	a.cachedAt = time.Time{}
	a.cachedVia = false
	a.ipMu.Unlock()
}

func (a *App) hasFreshPublicIP(viaProxy bool) bool {
	a.ipMu.Lock()
	defer a.ipMu.Unlock()
	return a.cachedIP != "" && a.cachedVia == viaProxy && time.Since(a.cachedAt) < 45*time.Second
}

func (a *App) cachedPublicIPMasked(viaProxy bool) string {
	a.ipMu.Lock()
	defer a.ipMu.Unlock()
	if a.cachedIP == "" || a.cachedVia != viaProxy {
		return ""
	}
	if time.Since(a.cachedAt) > 2*time.Minute {
		return ""
	}
	return netinfo.MaskIP(a.cachedIP)
}

// GetPublicIP returns the current public IP (masked for UI) and raw value for diagnostics.
// When connected, the lookup goes through the local mixed proxy (exit IP).
func (a *App) GetPublicIP() map[string]any {
	viaProxy := a.manager != nil && a.manager.Running()
	mixed := 0
	if viaProxy && a.settings != nil && a.settings.MixedPort > 0 {
		mixed = a.settings.MixedPort
	} else if viaProxy {
		mixed = defaults.MixedPort
	}

	a.ipMu.Lock()
	if a.cachedIP != "" && a.cachedVia == viaProxy && time.Since(a.cachedAt) < 45*time.Second {
		ip := a.cachedIP
		a.ipMu.Unlock()
		return map[string]any{
			"ip":      ip,
			"masked":  netinfo.MaskIP(ip),
			"viaProxy": viaProxy,
		}
	}
	a.ipMu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	ip, err := netinfo.LookupPublicIP(ctx, mixed)
	if err != nil || ip == "" {
		return map[string]any{
			"ip":      "",
			"masked":  "-",
			"viaProxy": viaProxy,
			"error":   fmt.Sprint(err),
		}
	}
	a.ipMu.Lock()
	a.cachedIP = ip
	a.cachedAt = time.Now()
	a.cachedVia = viaProxy
	a.ipMu.Unlock()
	return map[string]any{
		"ip":       ip,
		"masked":   netinfo.MaskIP(ip),
		"viaProxy": viaProxy,
	}
}

func (a *App) proxyGroup() string {
	if a.settings != nil && strings.TrimSpace(a.settings.ProxyGroup) != "" {
		return a.settings.ProxyGroup
	}
	return defaults.ProxyGroup
}

func (a *App) GetSettings() *profiles.Settings {
	if a.settings == nil {
		return profiles.DefaultSettings()
	}
	return a.settings
}

// GenerateWARPConfig registers a free Cloudflare WARP account and stores keys in settings.
func (a *App) GenerateWARPConfig() (map[string]any, error) {
	a.mu.Lock()
	license := ""
	if a.settings != nil {
		license = a.settings.WARPLicenseKey
	}
	a.mu.Unlock()

	acc, err := warp.Register(license)
	if err != nil {
		return nil, err
	}

	a.mu.Lock()
	defer a.mu.Unlock()
	if a.settings == nil {
		a.settings = profiles.DefaultSettings()
	}
	a.settings.WARPPrivateKey = acc.PrivateKey
	a.settings.WARPLocalAddress = acc.LocalAddress
	if a.settings.WARPPublicKey == "" {
		a.settings.WARPPublicKey = defaults.WARPPublicKey
	}
	if a.settings.WARPEndpoint == "" {
		a.settings.WARPEndpoint = defaults.WARPEndpoint
	}
	if a.settings.WARPMode == "" {
		a.settings.WARPMode = defaults.WARPModeViaProxy
	}
	_ = profiles.SaveSettings(a.paths.SettingsFile(), a.settings)
	if a.log != nil {
		a.log.Info("WARP config generated type=%s", acc.AccountType)
	}
	return map[string]any{
		"privateKey":   acc.PrivateKey,
		"localAddress": acc.LocalAddress,
		"accountType":  acc.AccountType,
		"clientId":     acc.ClientID,
	}, nil
}

func (a *App) SaveSettings(s profiles.Settings) error {
	profiles.NormalizeSettings(&s, "")
	if s.Mode == "" {
		s.Mode = defaults.DefaultMode
	}
	switch strings.ToLower(s.Mode) {
	case "rule", "global", "direct":
	default:
		return fmt.Errorf("invalid mode %q", s.Mode)
	}
	if s.MixedPort < 1024 || s.MixedPort > 65535 {
		return fmt.Errorf("mixed port must be 1024–65535")
	}
	if s.WARPEnabled && strings.TrimSpace(s.WARPPrivateKey) == "" {
		return fmt.Errorf("для WARP нужен конфиг — нажмите «Сгенерировать»")
	}
	if s.SelectedNode == defaults.WARPProxyName {
		s.SelectedNode = defaults.AutoGroup
	}

	a.mu.Lock()
	prev := a.settings
	a.settings = &s
	if a.api != nil && (prev == nil || prev.ControllerURL != s.ControllerURL || prev.Secret != s.Secret) {
		a.api.UpdateEndpoint(s.ControllerURL, s.Secret)
	}
	apiClient := a.api
	manager := a.manager
	scheduler := a.scheduler
	healthMon := a.health
	wasRunning := manager != nil && manager.Running()
	reconnect := wasRunning && coreSettingsChanged(prev, &s)
	a.mu.Unlock()

	if healthMon != nil {
		healthMon.SetInterval(time.Duration(s.HealthIntervalSec) * time.Second)
	}
	if manager != nil && wasRunning && apiClient != nil && !reconnect {
		_ = apiClient.SetMode(s.Mode)
	}
	if scheduler != nil {
		scheduler.SetInterval(time.Duration(s.SubscriptionIntervalMin) * time.Minute)
		if s.AutoUpdateSubscriptions {
			scheduler.Start()
		} else {
			scheduler.Stop()
		}
	}
	if err := a.applyAutostartSetting(); err != nil {
		return err
	}
	if err := profiles.SaveSettings(a.paths.SettingsFile(), a.settings); err != nil {
		return err
	}
	if reconnect {
		if coreNeedsProcessRestart(prev, &s) {
			_ = a.Disconnect()
			if err := a.Connect(); err != nil {
				return fmt.Errorf("настройки сохранены, но переподключение не удалось: %w", err)
			}
			return nil
		}
		if err := a.reloadRunningConfig(); err != nil {
			a.log.Warn("hot reload failed, full reconnect: %v", err)
			_ = a.Disconnect()
			if err2 := a.Connect(); err2 != nil {
				return fmt.Errorf("настройки сохранены, но переподключение не удалось: %w", err2)
			}
		}
	}
	return nil
}

func coreNeedsProcessRestart(prev, next *profiles.Settings) bool {
	if prev == nil || next == nil {
		return true
	}
	return prev.TUN != next.TUN ||
		prev.TUNStack != next.TUNStack ||
		prev.MixedPort != next.MixedPort ||
		prev.AllowLAN != next.AllowLAN ||
		prev.UseSystemProxy != next.UseSystemProxy ||
		prev.ControllerURL != next.ControllerURL ||
		prev.Secret != next.Secret ||
		prev.KillSwitch != next.KillSwitch ||
		prev.DNSLeakProtection != next.DNSLeakProtection ||
		prev.WARPEnabled != next.WARPEnabled ||
		prev.WARPMode != next.WARPMode ||
		prev.WARPPrivateKey != next.WARPPrivateKey ||
		prev.WARPLocalAddress != next.WARPLocalAddress ||
		prev.WARPEndpoint != next.WARPEndpoint ||
		prev.WARPPublicKey != next.WARPPublicKey ||
		prev.WARPCleanIP != next.WARPCleanIP ||
		prev.WARPPort != next.WARPPort ||
		prev.WARPNoiseCount != next.WARPNoiseCount ||
		prev.WARPNoiseMode != next.WARPNoiseMode ||
		prev.WARPNoiseSize != next.WARPNoiseSize ||
		prev.WARPNoiseDelay != next.WARPNoiseDelay ||
		prev.VPNInterface != next.VPNInterface
}

// reloadRunningConfig rebuilds YAML and asks mihomo to reload without tearing down TUN.
func (a *App) reloadRunningConfig() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.settings == nil || a.manager == nil || !a.manager.Running() || a.api == nil {
		return fmt.Errorf("vpn is not running")
	}
	if a.settings.ActiveProfile == "" {
		return fmt.Errorf("no active profile selected")
	}
	profile, err := a.store.Get(a.settings.ActiveProfile)
	if err != nil {
		return err
	}
	in := config.FromSettings(a.settings)
	in.Profile = profile
	in.ProxyGroup = a.proxyGroup()
	doc, err := config.Build(in)
	if err != nil {
		return err
	}
	cfgPath := a.paths.RuntimeConfig()
	if abs, err := filepath.Abs(cfgPath); err == nil {
		cfgPath = abs
	}
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(cfgPath, doc, 0o600); err != nil {
		return err
	}
	if err := a.api.ReloadConfig(cfgPath); err != nil {
		return err
	}
	_ = a.api.CloseConnections()
	if a.settings.SelectedNode != "" {
		if err := a.restoreSelectedNode("reload"); err != nil && a.log != nil {
			a.log.Warn("restore selected node after reload: %v", err)
		}
	}
	a.clearPublicIPCache()
	a.log.Info("reloaded routing config profile=%s", a.settings.ActiveProfile)
	return nil
}

func coreSettingsChanged(prev, next *profiles.Settings) bool {
	if prev == nil || next == nil {
		return false
	}
	return prev.TUN != next.TUN ||
		prev.TUNStack != next.TUNStack ||
		prev.MixedPort != next.MixedPort ||
		prev.AllowLAN != next.AllowLAN ||
		prev.IPv6 != next.IPv6 ||
		prev.LogLevel != next.LogLevel ||
		prev.BypassLAN != next.BypassLAN ||
		prev.BypassGEOIP != next.BypassGEOIP ||
		prev.RouteDirect != next.RouteDirect ||
		prev.RouteBlock != next.RouteBlock ||
		prev.RouteProxy != next.RouteProxy ||
		prev.RouteWhitelist != next.RouteWhitelist ||
		prev.AppRouteMode != next.AppRouteMode ||
		prev.AppRouteList != next.AppRouteList ||
		prev.DNSEnhancedMode != next.DNSEnhancedMode ||
		prev.DNSNameservers != next.DNSNameservers ||
		prev.DNSFallbacks != next.DNSFallbacks ||
		prev.DNSFakeIPRange != next.DNSFakeIPRange ||
		prev.Sniffer != next.Sniffer ||
		prev.TCPConcurrent != next.TCPConcurrent ||
		prev.UnifiedDelay != next.UnifiedDelay ||
		prev.WARPEnabled != next.WARPEnabled ||
		prev.WARPMode != next.WARPMode ||
		prev.WARPPrivateKey != next.WARPPrivateKey ||
		prev.WARPLocalAddress != next.WARPLocalAddress ||
		prev.WARPEndpoint != next.WARPEndpoint ||
		prev.WARPPublicKey != next.WARPPublicKey ||
		prev.WARPLicenseKey != next.WARPLicenseKey ||
		prev.WARPCleanIP != next.WARPCleanIP ||
		prev.WARPPort != next.WARPPort ||
		prev.WARPNoiseCount != next.WARPNoiseCount ||
		prev.WARPNoiseMode != next.WARPNoiseMode ||
		prev.WARPNoiseSize != next.WARPNoiseSize ||
		prev.WARPNoiseDelay != next.WARPNoiseDelay ||
		prev.URLTestPreset != next.URLTestPreset ||
		prev.URLTestURL != next.URLTestURL ||
		prev.URLTestIntervalSec != next.URLTestIntervalSec ||
		prev.KillSwitch != next.KillSwitch ||
		prev.DNSLeakProtection != next.DNSLeakProtection ||
		prev.UseSystemProxy != next.UseSystemProxy ||
		prev.ControllerURL != next.ControllerURL ||
		prev.Secret != next.Secret
}

func (a *App) applyAutostartSetting() error {
	if a.settings == nil {
		return nil
	}
	if a.settings.Autostart {
		return winutil.EnableAutostart()
	}
	return winutil.DisableAutostart()
}

func (a *App) SetMode(mode string) error {
	mode = strings.ToLower(strings.TrimSpace(mode))
	switch mode {
	case "rule", "global", "direct":
	default:
		return fmt.Errorf("invalid mode %q", mode)
	}
	a.settings.Mode = mode
	_ = profiles.SaveSettings(a.paths.SettingsFile(), a.settings)
	if a.manager != nil && a.manager.Running() {
		return a.api.SetMode(mode)
	}
	return nil
}

func (a *App) ListRunningApps() ([]winutil.RunningApp, error) {
	return winutil.ListRunningApps()
}

func (a *App) ListProfiles() ([]profiles.Profile, error) {
	return a.store.List()
}

func (a *App) UpsertProfile(p profiles.Profile) error {
	if err := a.store.Upsert(p); err != nil {
		return err
	}
	return a.store.Save()
}

// ImportProfileText accepts JSON proxy, share links, or Clash YAML.
func (a *App) ImportProfileText(name, note, raw string) error {
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(strings.ToLower(raw), "http://") || strings.HasPrefix(strings.ToLower(raw), "https://") {
		return a.ImportSubscription(name, note, raw)
	}
	nodes, err := parse.ParseInput(raw)
	if err != nil {
		return err
	}
	if name == "" {
		name = "Imported"
	}
	if err := a.UpsertProfile(profiles.Profile{Name: name, Note: note, Proxies: nodes}); err != nil {
		return err
	}
	return a.SetActiveProfile(name)
}

// ImportSubscription downloads an http(s) subscription URL into a profile.
func (a *App) ImportSubscription(name, note, rawURL string) error {
	rawURL = strings.TrimSpace(rawURL)
	if name == "" {
		name = "Subscription"
	}
	res, err := a.subs.Fetch(rawURL)
	if err != nil {
		return err
	}
	p := profiles.Profile{
		Name:                name,
		Note:                note,
		Proxies:             res.Proxies,
		SubscriptionURL:     rawURL,
		Quota:               res.Quota,
		LastSyncAt:          time.Now().UTC().Format(time.RFC3339),
		UpdateIntervalHours: res.UpdateIntervalHours,
		ProviderTitle:       res.ProviderTitle,
	}
	if p.Note == "" && res.ProviderTitle != "" {
		p.Note = res.ProviderTitle
	}
	if err := a.UpsertProfile(p); err != nil {
		return err
	}
	return a.SetActiveProfile(name)
}

// SyncSubscription refreshes one profile from its subscription URL.
func (a *App) SyncSubscription(name string) error {
	p, err := a.store.Get(name)
	if err != nil {
		return err
	}
	if strings.TrimSpace(p.SubscriptionURL) == "" {
		return fmt.Errorf("profile %q has no subscription URL", name)
	}
	res, err := a.subs.Fetch(p.SubscriptionURL)
	if err != nil {
		return err
	}
	p.Proxies = res.Proxies
	p.Quota = res.Quota
	p.LastSyncAt = time.Now().UTC().Format(time.RFC3339)
	if res.UpdateIntervalHours > 0 {
		p.UpdateIntervalHours = res.UpdateIntervalHours
	}
	if res.ProviderTitle != "" {
		p.ProviderTitle = res.ProviderTitle
	}
	if err := a.UpsertProfile(p); err != nil {
		return err
	}
	// If this is the active connected profile, reconnect to apply new nodes.
	if a.settings.ActiveProfile == name && a.manager != nil && a.manager.Running() {
		_ = a.Disconnect()
		return a.Connect()
	}
	return nil
}

// SyncAllSubscriptions refreshes due remote profiles (or all when force=true).
func (a *App) SyncAllSubscriptions() (int, error) {
	return a.syncSubscriptions(true)
}

func (a *App) syncSubscriptions(force bool) (int, error) {
	list, err := a.store.List()
	if err != nil {
		return 0, err
	}
	updated := 0
	var firstErr error
	now := time.Now()
	for _, p := range list {
		if strings.TrimSpace(p.SubscriptionURL) == "" {
			continue
		}
		if !force && !subscription.Due(p, a.settings.SubscriptionIntervalMin, now) {
			continue
		}
		if err := a.SyncSubscription(p.Name); err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		updated++
	}
	return updated, firstErr
}

// GetSubscriptionInfo returns quota/expiry for a profile.
func (a *App) GetSubscriptionInfo(name string) (map[string]any, error) {
	p, err := a.store.Get(name)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"name":                p.Name,
		"subscriptionURL":     p.SubscriptionURL,
		"providerTitle":       p.ProviderTitle,
		"lastSyncAt":          p.LastSyncAt,
		"updateIntervalHours": p.UpdateIntervalHours,
		"nodeCount":           len(p.Proxies),
		"upload":              p.Quota.Upload,
		"download":            p.Quota.Download,
		"total":               p.Quota.Total,
		"used":                p.Quota.Used(),
		"remaining":           p.Quota.Remaining(),
		"expireUnix":          p.Quota.ExpireUnix,
		"expired":             p.Quota.Expired(),
	}, nil
}

// GenerateSubscriptionQR generates a QR code for a profile's subscription URL as a base64-encoded PNG.
// Returns empty string if the profile has no subscription URL.
func (a *App) GenerateSubscriptionQR(name string) (string, error) {
	p, err := a.store.Get(name)
	if err != nil {
		return "", err
	}
	subURL := strings.TrimSpace(p.SubscriptionURL)
	if subURL == "" {
		return "", fmt.Errorf("profile %q has no subscription URL", name)
	}
	
	// Generate QR code as PNG (256x256 is a good size for screen display).
	png, err := qrcode.Encode(subURL, qrcode.Medium, 256)
	if err != nil {
		return "", fmt.Errorf("generate QR: %w", err)
	}
	
	// Return as data URI (base64-encoded PNG).
	encoded := "data:image/png;base64," + base64.StdEncoding.EncodeToString(png)
	if a.log != nil {
		a.log.Info("generated QR for profile=%s url_len=%d", name, len(subURL))
	}
	return encoded, nil
}

func (a *App) RemoveProfile(name string) error {
	if err := a.store.Remove(name); err != nil {
		return err
	}
	if a.settings.ActiveProfile == name {
		a.settings.ActiveProfile = ""
		_ = profiles.SaveSettings(a.paths.SettingsFile(), a.settings)
	}
	return a.store.Save()
}

func (a *App) SetActiveProfile(name string) error {
	if _, err := a.store.Get(name); err != nil {
		return err
	}
	a.settings.ActiveProfile = name
	return profiles.SaveSettings(a.paths.SettingsFile(), a.settings)
}

func (a *App) IsAdmin() bool { return winutil.IsAdmin() }

func (a *App) RelaunchAsAdmin() error {
	return winutil.RelaunchAsAdmin()
}

// ensureBundledCore extracts embedded mihomo/geo into AppData when missing or outdated.
func (a *App) ensureBundledCore() error {
	if err := bundle.ExtractFS(bundledCore, "resources/core", a.paths.CoreWorkDir()); err != nil {
		return err
	}
	if _, err := os.Stat(a.paths.CoreBinary()); err != nil {
		return fmt.Errorf("core binary missing after extract: %w", err)
	}
	return nil
}

func (a *App) Connect() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.settings.ActiveProfile == "" {
		return fmt.Errorf("no active profile selected")
	}
	if err := winutil.RequireAdminForTUN(a.settings.TUN); err != nil {
		return err
	}

	if err := a.ensureBundledCore(); err != nil {
		return fmt.Errorf("prepare core: %w", err)
	}
	// Manager may have been constructed before extract; keep path in sync.
	if a.manager != nil {
		a.manager = core.NewManager(core.Options{
			CorePath:      a.paths.CoreBinary(),
			WorkDir:       a.paths.CoreWorkDir(),
			ConfigPath:    a.paths.RuntimeConfig(),
			ControllerURL: a.settings.ControllerURL,
			Secret:        a.settings.Secret,
			MixedPort:     a.settings.MixedPort,
			API:           a.api,
		})
	}

	checked, err := integrity.VerifyBeside(a.paths.CoreBinary())
	if err != nil {
		return err
	}
	if a.settings.RequireCoreHash && !checked {
		return fmt.Errorf("mihomo.exe.sha256 is required but missing")
	}

	if err := geo.Ensure(a.paths.CoreWorkDir()); err != nil {
		// Non-fatal if offline and assets already absent; still try to connect.
		_ = err
	}

	profile, err := a.store.Get(a.settings.ActiveProfile)
	if err != nil {
		return err
	}
	in := config.FromSettings(a.settings)
	in.Profile = profile
	in.ProxyGroup = a.proxyGroup()
	doc, err := config.Build(in)
	if err != nil {
		return err
	}

	if err := a.manager.Start(doc); err != nil {
		return err
	}

	if !a.settings.TUN && a.settings.UseSystemProxy {
		snap, err := winutil.EnableSystemProxy("127.0.0.1", a.settings.MixedPort)
		if err != nil {
			_ = a.manager.Stop()
			return winutil.FormatProxyError(err)
		}
		a.proxySnap = snap
		a.proxyOn = true
		_ = a.saveProxySnapshot(snap, a.settings.MixedPort)
	}

	appWhitelist := config.NormalizeAppRouteMode(a.settings.AppRouteMode) == config.AppRouteWhitelist &&
		len(config.ParseAppList(a.settings.AppRouteList)) > 0

	if a.settings.KillSwitch {
		// Whitelist sends non-listed apps DIRECT via the physical NIC; a firewall
		// kill-switch that only allows the TUN iface would block Discord etc.
		if appWhitelist {
			a.log.Warn("kill switch skipped: app whitelist uses DIRECT for non-listed apps")
		} else {
			iface := strings.TrimSpace(a.settings.VPNInterface)
			if iface == "" {
				iface = defaults.DefaultVPNIface
				a.settings.VPNInterface = iface
			}
			if err := a.killSwitch.Enable(iface); err != nil {
				_ = a.cleanupNetwork()
				_ = a.manager.Stop()
				return fmt.Errorf("kill switch: %w", err)
			}
		}
	}
	if a.settings.DNSLeakProtection {
		// Same conflict: DIRECT apps may need real DNS paths when hijack gaps exist.
		if appWhitelist {
			a.log.Warn("dns leak protection skipped: incompatible with app whitelist DIRECT path")
		} else if err := a.dnsGuard.Enable(); err != nil {
			_ = a.cleanupNetwork()
			_ = a.manager.Stop()
			return fmt.Errorf("dns leak protection: %w", err)
		}
	}

	if a.settings.AutoReconnect && a.health != nil {
		a.health.Start()
	}
	if a.settings.SelectedNode != "" {
		if err := a.restoreSelectedNode("connect"); err != nil && a.log != nil {
			a.log.Warn("restore selected node: %v", err)
		}
	}
	if a.api != nil {
		a.api.StartTrafficStream()
	}
	a.clearPublicIPCache()
	a.log.Info("connected profile=%s tun=%v mode=%s", a.settings.ActiveProfile, a.settings.TUN, a.settings.Mode)
	a.refreshTrayStatus()
	return nil
}

func (a *App) Disconnect() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.health != nil {
		a.health.Stop()
	}
	_ = a.cleanupNetwork()
	if a.api != nil {
		a.api.StopTrafficStream()
	}
	a.clearPublicIPCache()
	var err error
	if a.manager != nil {
		err = a.manager.Stop()
	}
	a.refreshTrayStatus()
	return err
}

func (a *App) cleanupNetwork() error {
	if a.dnsGuard != nil {
		_ = a.dnsGuard.Disable()
	}
	if a.killSwitch != nil {
		_ = a.killSwitch.Disable()
	}
	if a.proxyOn {
		_ = winutil.RestoreSystemProxy(a.proxySnap)
		a.proxyOn = false
	}
	_ = a.clearProxySnapshotFile()
	return nil
}

type persistedProxySnapshot struct {
	Port int                         `json:"port"`
	Snap winutil.SystemProxySnapshot `json:"snap"`
}

func (a *App) proxySnapshotPath() string {
	if a.paths == nil {
		return ""
	}
	return filepath.Join(a.paths.DataDir(), "system-proxy.json")
}

func (a *App) saveProxySnapshot(snap winutil.SystemProxySnapshot, port int) error {
	path := a.proxySnapshotPath()
	if path == "" {
		return nil
	}
	raw, err := json.Marshal(persistedProxySnapshot{Port: port, Snap: snap})
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o600)
}

func (a *App) loadProxySnapshot() (persistedProxySnapshot, bool) {
	path := a.proxySnapshotPath()
	if path == "" {
		return persistedProxySnapshot{}, false
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return persistedProxySnapshot{}, false
	}
	var p persistedProxySnapshot
	if err := json.Unmarshal(raw, &p); err != nil {
		return persistedProxySnapshot{}, false
	}
	return p, true
}

func (a *App) clearProxySnapshotFile() error {
	path := a.proxySnapshotPath()
	if path == "" {
		return nil
	}
	err := os.Remove(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// recoverOrphanNetworkState clears sticky firewall rules and leftover system proxy
// after a crash / hard power-off (Hiddify-class #2306 / #2323).
func (a *App) recoverOrphanNetworkState() {
	if a.killSwitch != nil {
		if err := a.killSwitch.Disable(); err != nil && a.log != nil {
			a.log.Warn("startup kill-switch cleanup: %v", err)
		}
	}
	if a.dnsGuard != nil {
		if err := a.dnsGuard.Disable(); err != nil && a.log != nil {
			a.log.Warn("startup dns-leak cleanup: %v", err)
		}
	}

	if snap, ok := a.loadProxySnapshot(); ok {
		if err := winutil.RestoreSystemProxy(snap.Snap); err != nil {
			if a.log != nil {
				a.log.Warn("startup proxy restore: %v", err)
			}
		} else if a.log != nil {
			a.log.Info("restored system proxy from crash snapshot (port=%d)", snap.Port)
		}
		_ = a.clearProxySnapshotFile()
	}

	port := defaults.MixedPort
	if a.settings != nil && a.settings.MixedPort > 0 {
		port = a.settings.MixedPort
	}
	if cleared, err := winutil.ClearOurSystemProxy("127.0.0.1", port); err != nil {
		if a.log != nil {
			a.log.Warn("startup orphan proxy clear: %v", err)
		}
	} else if cleared && a.log != nil {
		a.log.Info("cleared orphan system proxy 127.0.0.1:%d", port)
	}

	// Reap mihomo left behind after Task Manager kill of the UI.
	if a.paths != nil {
		coreBin := a.paths.CoreBinary()
		if n, err := winutil.KillProcessesWithImagePath(coreBin); err != nil && a.log != nil {
			a.log.Warn("startup orphan core cleanup: %v", err)
		} else if n > 0 && a.log != nil {
			a.log.Info("killed %d orphan core process(es) at %s", n, coreBin)
		}
	}
}

// autostartConnect runs after Windows logon launch (--autostart).
func (a *App) autostartConnect() {
	// Let Wails finish binding / tray before Connect (TUN + elevation already done).
	time.Sleep(800 * time.Millisecond)
	if a.ctx != nil && a.settings != nil && a.settings.CloseToTray {
		runtime.WindowHide(a.ctx)
	}
	a.mu.Lock()
	hasProfile := a.settings != nil && strings.TrimSpace(a.settings.ActiveProfile) != ""
	profile := ""
	if a.settings != nil {
		profile = a.settings.ActiveProfile
	}
	a.mu.Unlock()
	if !hasProfile {
		if a.log != nil {
			a.log.Info("autostart: no active profile, skip connect")
		}
		return
	}
	if a.manager != nil && a.manager.Running() {
		return
	}
	if err := a.Connect(); err != nil {
		if a.log != nil {
			a.log.Warn("autostart connect failed: %v", err)
		}
		return
	}
	a.maybeInitialNodesProbe()
	if a.log != nil {
		a.log.Info("autostart connected profile=%s", profile)
	}
}

func (a *App) consumeDeepLinks() {
	// Wait for Wails bindings / tray.
	time.Sleep(1200 * time.Millisecond)

	a.deeplinkMu.Lock()
	startup := a.startupDeepLink
	a.startupDeepLink = ""
	a.deeplinkMu.Unlock()
	if startup != "" {
		a.applyDeepLink(startup)
	}

	ticker := time.NewTicker(800 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-a.ctx.Done():
			return
		case <-ticker.C:
			if link, ok := winutil.TakePendingDeepLink(); ok {
				a.applyDeepLink(link)
			}
		}
	}
}

func (a *App) applyDeepLink(raw string) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return
	}
	if a.log != nil {
		a.log.Info("deep link: %s", applog.Redact(raw))
	}
	act, err := deeplink.Parse(raw)
	if err != nil {
		a.emitDeepLink("error", friendlyDeepLinkErr(err), "")
		return
	}
	a.ShowWindow()

	switch act.Kind {
	case deeplink.KindOpen:
		a.emitDeepLink("open", "Окно открыто", "")
	case deeplink.KindConnect:
		if err := a.Connect(); err != nil {
			a.emitDeepLink("error", err.Error(), "")
			return
		}
		a.emitDeepLink("connect", "Подключено", "")
	case deeplink.KindDisconnect:
		if err := a.Disconnect(); err != nil {
			a.emitDeepLink("error", err.Error(), "")
			return
		}
		a.emitDeepLink("disconnect", "Отключено", "")
	case deeplink.KindToggle:
		if err := a.ToggleConnect(); err != nil {
			a.emitDeepLink("error", err.Error(), "")
			return
		}
		a.emitDeepLink("toggle", "Состояние VPN переключено", "")
	case deeplink.KindImport, deeplink.KindAdd:
		name := strings.TrimSpace(act.Name)
		if name == "" {
			name = "Deep link"
		}
		if err := a.ImportProfileText(name, "", act.Data); err != nil {
			a.emitDeepLink("error", err.Error(), "")
			return
		}
		a.emitDeepLink("import", "Профиль импортирован: "+name, name)
	default:
		a.emitDeepLink("error", "Неизвестная команда deep link", "")
	}
}

func (a *App) emitDeepLink(kind, message, profile string) {
	if a.ctx == nil {
		return
	}
	runtime.EventsEmit(a.ctx, "deeplink", map[string]any{
		"kind":    kind,
		"message": message,
		"profile": profile,
		"ok":      kind != "error",
	})
}

func friendlyDeepLinkErr(err error) string {
	if err == nil {
		return "ошибка deep link"
	}
	return err.Error()
}

func (a *App) ToggleConnect() error {
	if a.manager != nil && a.manager.Running() {
		return a.Disconnect()
	}
	if err := a.Connect(); err != nil {
		return err
	}
	a.maybeInitialNodesProbe()
	return nil
}

func (a *App) GetTraffic() map[string]any {
	out := map[string]any{"up": 0, "down": 0, "totalUp": 0, "totalDown": 0}
	if a.manager == nil || !a.manager.Running() || a.api == nil {
		return out
	}
	a.api.StartTrafficStream()
	if up, down, age, ok := a.api.CachedTraffic(); ok && age < 3*time.Second {
		out["up"] = up
		out["down"] = down
	} else {
		ctx, cancel := context.WithTimeout(context.Background(), defaults.TrafficSampleTimeout)
		defer cancel()
		if snap, err := a.api.TrafficOnce(ctx); err == nil {
			out["up"] = snap.Up
			out["down"] = snap.Down
		}
	}
	if conn, err := a.api.Connections(); err == nil {
		out["totalUp"] = conn.UploadTotal
		out["totalDown"] = conn.DownloadTotal
	}
	return out
}

// CheckForUpdate queries GitHub release tag (default latest-main).
func (a *App) CheckForUpdate() (map[string]any, error) {
	owner := a.settings.UpdateOwner
	repo := a.settings.UpdateRepo
	if owner == "" || repo == "" {
		return map[string]any{"available": false, "reason": "updateOwner/updateRepo not configured"}, nil
	}
	checker := update.NewGitHub(owner, repo, a.settings.UpdateTag)
	upd, ok, err := checker.Check(version)
	if err != nil {
		return nil, err
	}
	if !ok {
		return map[string]any{"available": false}, nil
	}
	return map[string]any{
		"available": true,
		"tag":       upd.Tag,
		"name":      upd.Asset.Name,
		"url":       upd.Download,
		"size":      upd.Asset.Size,
	}, nil
}

// DownloadUpdate fetches the latest-main package into the updates folder.
// Requires release assets: zip + .sha256 + .sig (Ed25519).
func (a *App) DownloadUpdate() (string, error) {
	owner := a.settings.UpdateOwner
	repo := a.settings.UpdateRepo
	if owner == "" || repo == "" {
		return "", fmt.Errorf("updateOwner/updateRepo not configured")
	}
	checker := update.NewGitHub(owner, repo, a.settings.UpdateTag)
	rel, err := checker.FetchRelease()
	if err != nil {
		return "", err
	}
	asset, err := checker.FindWindowsZip(rel)
	if err != nil {
		return "", err
	}
	dest := filepath.Join(a.paths.DataDir(), "updates")
	path, err := checker.DownloadVerified(rel, asset, dest, string(updatePublicKey))
	if err != nil {
		return "", err
	}
	a.log.Info("verified update downloaded: %s", path)
	return path, nil
}

// ApplyUpdate verifies local sidecars again, then installs and restarts.
func (a *App) ApplyUpdate(zipPath string) error {
	if zipPath == "" {
		return fmt.Errorf("zip path is required")
	}
	pub, err := update.ParsePublicKey(string(updatePublicKey))
	if err != nil {
		return err
	}
	if err := update.VerifyPackage(zipPath, zipPath+".sha256", zipPath+".sig", pub); err != nil {
		return err
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	target := filepath.Dir(exe)
	helper, err := update.ApplyZip(zipPath, target, filepath.Base(exe))
	if err != nil {
		return err
	}
	a.log.Info("update helper started: %s", helper)
	_ = a.Disconnect()
	runtime.Quit(a.ctx)
	return nil
}

// ListNodes returns selectable proxies from the configured proxy group.
func (a *App) ListNodes() ([]api.ProxyNodeRuntime, error) {
	if a.manager == nil || !a.manager.Running() {
		return []api.ProxyNodeRuntime{}, nil
	}
	_, nodes, err := a.api.ListSelectableNodes(a.proxyGroup())
	return nodes, err
}

// CurrentNode returns the active selector choice.
func (a *App) CurrentNode() (string, error) {
	if a.manager == nil || !a.manager.Running() {
		return a.settings.SelectedNode, nil
	}
	g, err := a.api.Group(a.proxyGroup())
	if err != nil {
		return a.settings.SelectedNode, err
	}
	return g.Now, nil
}

// SelectNode switches the active proxy and closes existing connections.
func (a *App) SelectNode(name string) error {
	if a.manager == nil || !a.manager.Running() {
		return fmt.Errorf("not connected")
	}
	group := a.proxyGroup()
	if err := a.api.SelectProxy(group, name); err != nil {
		return err
	}
	_ = a.api.CloseConnections()
	a.settings.SelectedNode = name
	_ = profiles.SaveSettings(a.paths.SettingsFile(), a.settings)
	a.clearPublicIPCache()
	a.log.Info("selected node=%s group=%s", name, group)
	a.refreshTrayStatus()

	// Wake url-test so AUTO starts measuring and picks a live node ASAP.
	if name == defaults.AutoGroup {
		go func() {
			timeout := defaults.AutoGroupDelayFloorMS
			// Prefer /group/.../delay for URLTest; /proxies/AUTO/delay is flaky on some mihomo builds.
			if _, err := a.api.TestGroupDelay(defaults.AutoGroup, a.probeURL(), timeout); err != nil {
				if d, err2 := a.api.TestDelay(defaults.AutoGroup, a.probeURL(), timeout); err2 != nil {
					a.log.Warn("AUTO url-test: %v", err)
				} else {
					a.log.Info("AUTO url-test delay=%dms", d)
				}
			} else {
				a.log.Info("AUTO group delay refreshed")
			}
		}()
	}
	return nil
}

// TestNodeDelay measures latency for a node using configured ping method.
func (a *App) TestNodeDelay(name string) (int, error) {
	if a.manager == nil || !a.manager.Running() {
		return 0, fmt.Errorf("not connected")
	}
	return a.probeNode(name)
}

// NodeDelayResult is one URL-test outcome.
type NodeDelayResult struct {
	Name  string `json:"name"`
	Delay int    `json:"delay"`
	Error string `json:"error,omitempty"`
}

func (a *App) probeURL() string {
	if a.settings == nil {
		return defaults.URLTestURL
	}
	return defaults.ResolveURLTestURL(a.settings.URLTestPreset, a.settings.URLTestURL)
}

func (a *App) probeMethod() string {
	if a.settings == nil || a.settings.PingMethod == "" {
		return defaults.DefaultPingMethod
	}
	return a.settings.PingMethod
}

func (a *App) probeNode(name string) (int, error) {
	// AUTO / nested groups have no leaf server — always use controller delay API.
	if name == defaults.AutoGroup {
		if a.api == nil {
			return 0, fmt.Errorf("api client required")
		}
		timeout := defaults.AutoGroupDelayFloorMS
		// Group endpoint is the correct API for url-test strategies.
		if m, err := a.api.TestGroupDelay(name, a.probeURL(), timeout); err == nil {
			best := 0
			for _, v := range m {
				if v > 0 && (best == 0 || v < best) {
					best = v
				}
			}
			return best, nil
		}
		return a.api.TestDelay(name, a.probeURL(), timeout)
	}

	proxy := profiles.ProxyNode{"name": name}
	if a.settings != nil && a.settings.ActiveProfile != "" && a.store != nil {
		if p, err := a.store.Get(a.settings.ActiveProfile); err == nil {
			if found, ok := latency.FindProxy(p.Proxies, name); ok {
				proxy = profiles.ProxyNode{}
				for k, v := range found {
					proxy[k] = v
				}
				proxy["name"] = name
			}
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(defaults.URLTestTimeoutMS)*time.Millisecond+time.Second)
	defer cancel()
	mixed := defaults.MixedPort
	if a.settings != nil && a.settings.MixedPort > 0 {
		mixed = a.settings.MixedPort
	}
	return latency.Probe(ctx, latency.Options{
		Method:    a.probeMethod(),
		TestURL:   a.probeURL(),
		Timeout:   time.Duration(defaults.URLTestTimeoutMS) * time.Millisecond,
		MixedPort: mixed,
		API:       a.api,
		Proxy:     proxy,
	})
}

func (a *App) emitNodePing(name, phase string, delay int, errMsg string) {
	if a.ctx == nil {
		return
	}
	payload := map[string]any{
		"name":  name,
		"phase": phase,
		"delay": delay,
	}
	if errMsg != "" {
		payload["error"] = errMsg
	}
	runtime.EventsEmit(a.ctx, "node:ping", payload)
}

// ConnectAndProbe connects (if needed) and runs latency tests for all nodes.
// Used after importing a profile — not on every VPN toggle.
func (a *App) ConnectAndProbe() ([]NodeDelayResult, error) {
	if a.manager == nil || !a.manager.Running() {
		if err := a.Connect(); err != nil {
			return nil, err
		}
	}
	results, err := a.TestAllNodes()
	if err == nil {
		a.markInitialNodesProbed()
	}
	return results, err
}

// maybeInitialNodesProbe pings all nodes once after the very first successful connect.
func (a *App) maybeInitialNodesProbe() {
	a.mu.Lock()
	done := a.settings != nil && a.settings.InitialNodesProbed
	a.mu.Unlock()
	if done {
		return
	}
	go func() {
		if a.manager == nil || !a.manager.Running() {
			return
		}
		if a.ctx != nil {
			runtime.EventsEmit(a.ctx, "nodes:probe", map[string]any{"phase": "start"})
		}
		_, err := a.TestAllNodes()
		if a.ctx != nil {
			payload := map[string]any{"phase": "done"}
			if err != nil {
				payload["error"] = err.Error()
			}
			runtime.EventsEmit(a.ctx, "nodes:probe", payload)
		}
		if err != nil {
			if a.log != nil {
				a.log.Warn("initial nodes probe: %v", err)
			}
			return
		}
		a.markInitialNodesProbed()
		if a.log != nil {
			a.log.Info("initial nodes probe completed")
		}
	}()
}

func (a *App) markInitialNodesProbed() {
	a.mu.Lock()
	if a.settings == nil || a.settings.InitialNodesProbed {
		a.mu.Unlock()
		return
	}
	a.settings.InitialNodesProbed = true
	path := ""
	if a.paths != nil {
		path = a.paths.SettingsFile()
	}
	cfg := a.settings
	a.mu.Unlock()
	if path != "" {
		_ = profiles.SaveSettings(path, cfg)
	}
}

// TestAllNodes runs latency tests against all selectable nodes (limited concurrency).
func (a *App) TestAllNodes() ([]NodeDelayResult, error) {
	nodes, err := a.ListNodes()
	if err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return []NodeDelayResult{}, nil
	}

	prev := ""
	if a.settings != nil {
		prev = a.settings.SelectedNode
	}
	if cur, err := a.CurrentNode(); err == nil && cur != "" {
		prev = cur
	}

	type job struct {
		name string
	}
	jobs := make(chan job)
	results := make(chan NodeDelayResult, len(nodes))

	workers := 5
	method := a.probeMethod()
	// HEAD/proxy-select must be serialized to avoid thrashing the active node.
	if method == defaults.PingProxyHTTPHead {
		workers = 1
	}
	// AUTO group test is heavy; keep a slot but serialize AUTO separately first.
	if len(nodes) < workers {
		workers = len(nodes)
	}
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				a.emitNodePing(j.name, "start", 0, "")
				d, err := a.probeNode(j.name)
				res := NodeDelayResult{Name: j.name, Delay: d}
				if err != nil {
					res.Error = err.Error()
					res.Delay = 0
				}
				a.emitNodePing(j.name, "done", res.Delay, res.Error)
				results <- res
			}
		}()
	}
	go func() {
		for _, n := range nodes {
			jobs <- job{name: n.Name}
		}
		close(jobs)
		wg.Wait()
		close(results)
	}()

	out := make([]NodeDelayResult, 0, len(nodes))
	for r := range results {
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Delay == 0 {
			return false
		}
		if out[j].Delay == 0 {
			return true
		}
		return out[i].Delay < out[j].Delay
	})

	// Restore previous selection (mass ping with HEAD switches nodes).
	if prev != "" && a.settings != nil {
		a.settings.SelectedNode = prev
	}
	if prev != "" && a.api != nil && a.manager != nil && a.manager.Running() {
		if err := a.restoreSelectedNode("ping"); err != nil {
			// Core may have been stopped mid-ping (settings reload / disconnect) — not actionable.
			if a.log != nil && !isControllerUnavailable(err) {
				a.log.Warn("restore node after ping: %v", err)
			}
		}
	}
	return out, nil
}

func isControllerUnavailable(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "connection refused") ||
		strings.Contains(s, "connectex") ||
		strings.Contains(s, "no connection could be made") ||
		strings.Contains(s, "actively refused") ||
		strings.Contains(s, "wsarecv") ||
		strings.Contains(s, "eof")
}

// restoreSelectedNode selects settings.SelectedNode if it still exists in the group.
// Stale names (after subscription refresh) are cleared and we fall back to AUTO.
func (a *App) restoreSelectedNode(reason string) error {
	if a.api == nil || a.settings == nil {
		return nil
	}
	want := strings.TrimSpace(a.settings.SelectedNode)
	if want == "" {
		return nil
	}
	group := a.proxyGroup()
	g, err := a.api.Group(group)
	if err != nil {
		return err
	}
	exists := want == defaults.AutoGroup
	if !exists {
		for _, n := range g.All {
			if n == want {
				exists = true
				break
			}
		}
	}
	if !exists {
		if a.log != nil {
			a.log.Info("selected node %q gone after %s — falling back to %s", want, reason, defaults.AutoGroup)
		}
		a.settings.SelectedNode = defaults.AutoGroup
		want = defaults.AutoGroup
		_ = profiles.SaveSettings(a.paths.SettingsFile(), a.settings)
	}
	if g.Now == want {
		return nil
	}
	if err := a.api.SelectProxy(group, want); err != nil {
		return err
	}
	_ = a.api.CloseConnections()
	return nil
}

// ShowWindow brings the main window back (close-to-tray flow).
func (a *App) ShowWindow() {
	if a.ctx != nil {
		runtime.WindowShow(a.ctx)
		runtime.WindowUnminimise(a.ctx)
	}
}

// HideWindow hides the main window.
func (a *App) HideWindow() {
	if a.ctx != nil {
		runtime.WindowHide(a.ctx)
	}
}

// QuitApp disconnects and exits.
func (a *App) QuitApp() {
	_ = a.Disconnect()
	if a.ctx != nil {
		runtime.Quit(a.ctx)
	}
}

// ShouldCloseToTray reports whether the window close button should hide instead of quit.
func (a *App) ShouldCloseToTray() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.settings == nil || a.settings.CloseToTray
}

// GetLogsTail returns recent lines from client.log and mihomo.log.
func (a *App) GetLogsTail(maxLines int) (string, error) {
	if maxLines <= 0 {
		maxLines = 200
	}
	clientPath := filepath.Join(a.paths.DataDir(), "client.log")
	clientTail, err1 := applog.Tail(clientPath, maxLines)
	coreTail, err2 := applog.Tail(filepath.Join(a.paths.CoreWorkDir(), "mihomo.log"), maxLines/2)

	var b strings.Builder
	b.WriteString("—— client.log ——\n")
	if err1 != nil {
		b.WriteString("(нет записей)\n")
	} else if strings.TrimSpace(clientTail) == "" {
		b.WriteString("(пусто)\n")
	} else {
		b.WriteString(clientTail)
		if !strings.HasSuffix(clientTail, "\n") {
			b.WriteByte('\n')
		}
	}
	b.WriteString("\n—— mihomo.log ——\n")
	if err2 != nil {
		b.WriteString("(ядро ещё не писало лог)\n")
	} else if strings.TrimSpace(coreTail) == "" {
		b.WriteString("(пусто)\n")
	} else {
		b.WriteString(coreTail)
	}
	return b.String(), nil
}

// OpenLogsFolder opens the data directory that holds client.log (and core/ beside it).
func (a *App) OpenLogsFolder() error {
	if a.paths == nil {
		return fmt.Errorf("paths not ready")
	}
	return winutil.OpenFolder(a.paths.DataDir())
}

// ExportLogs writes content (usually the UI-filtered view) via a save dialog.
func (a *App) ExportLogs(content string) error {
	if a.ctx == nil {
		return fmt.Errorf("app not ready")
	}
	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Сохранить лог",
		DefaultFilename: "moy-vpn-logs.txt",
		Filters: []runtime.FileFilter{
			{DisplayName: "Текст (*.txt)", Pattern: "*.txt"},
		},
	})
	if err != nil {
		return err
	}
	if strings.TrimSpace(path) == "" {
		return nil // cancelled
	}
	if !strings.HasSuffix(strings.ToLower(path), ".txt") {
		path += ".txt"
	}
	return os.WriteFile(path, []byte(content), 0o600)
}
