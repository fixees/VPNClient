package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"myinternetvpn/client/internal/api"
	"myinternetvpn/client/internal/applog"
	"myinternetvpn/client/internal/config"
	"myinternetvpn/client/internal/core"
	"myinternetvpn/client/internal/geo"
	"myinternetvpn/client/internal/health"
	"myinternetvpn/client/internal/integrity"
	"myinternetvpn/client/internal/parse"
	"myinternetvpn/client/internal/paths"
	"myinternetvpn/client/internal/profiles"
	"myinternetvpn/client/internal/subscription"
	"myinternetvpn/client/internal/update"
	"myinternetvpn/client/internal/winutil"

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

	mu sync.Mutex
}

func NewApp() *App {
	return &App{
		killSwitch: winutil.NewKillSwitch(),
		dnsGuard:   winutil.NewDNSLeakGuard(),
		subs:       subscription.NewFetcher(),
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.paths = paths.New()
	_ = a.paths.Ensure()

	if lg, err := applog.Open(filepath.Join(a.paths.DataDir(), "client.log")); err == nil {
		a.log = lg
		a.log.Info("starting MyInternetVPN %s", version)
	}

	a.store = profiles.NewStore(a.paths.ProfilesFile())
	_ = a.store.Load()

	a.settings = profiles.DefaultSettings()
	if loaded, err := profiles.LoadSettings(a.paths.SettingsFile()); err == nil {
		a.settings = loaded
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
	state := "disconnected"
	coreVersion := ""
	mode := a.settings.Mode
	up, down := int64(0), int64(0)
	totalUp, totalDown := int64(0), int64(0)

	if a.manager != nil && a.manager.Running() {
		state = "connected"
		if v, err := a.api.Version(); err == nil {
			coreVersion = v.Version
		}
		if m, err := a.api.Mode(); err == nil && m != "" {
			mode = m
		}
		ctx, cancel := context.WithTimeout(context.Background(), 800*time.Millisecond)
		if snap, err := a.api.TrafficOnce(ctx); err == nil {
			up, down = snap.Up, snap.Down
		}
		cancel()
		if conn, err := a.api.Connections(); err == nil {
			totalUp, totalDown = conn.UploadTotal, conn.DownloadTotal
		}
	}

	active := ""
	list, _ := a.store.List()
	var quota map[string]any
	subURL := ""
	lastSync := ""
	expireUnix := int64(0)
	if a.settings != nil {
		active = a.settings.ActiveProfile
		if active != "" {
			if p, err := a.store.Get(active); err == nil {
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

	return map[string]any{
		"state":                   state,
		"coreVersion":             coreVersion,
		"activeProfile":           active,
		"profileCount":            len(list),
		"mixedPort":               a.settings.MixedPort,
		"mode":                    mode,
		"tun":                     a.settings.TUN,
		"useSystemProxy":          a.settings.UseSystemProxy,
		"killSwitch":              a.settings.KillSwitch,
		"dnsLeakProtection":       a.settings.DNSLeakProtection,
		"autoReconnect":           a.settings.AutoReconnect,
		"autoUpdateSubscriptions": a.settings.AutoUpdateSubscriptions,
		"isAdmin":                 winutil.IsAdmin(),
		"speedUp":                 up,
		"speedDown":               down,
		"totalUp":                 totalUp,
		"totalDown":               totalDown,
		"subscriptionURL":         subURL,
		"subscriptionQuota":       quota,
		"subscriptionLastSync":    lastSync,
		"subscriptionExpireUnix":  expireUnix,
		"selectedNode":            a.settings.SelectedNode,
		"proxyGroup":              a.settings.ProxyGroup,
		"autostart":               a.settings.Autostart,
		"closeToTray":             a.settings.CloseToTray,
		"product":                 "MyInternetVPN",
		"site":                    "https://myinternetvpn.com",
		"appVersion":              version,
	}
}

func (a *App) GetSettings() *profiles.Settings {
	return a.settings
}

func (a *App) SaveSettings(s profiles.Settings) error {
	if s.Mode == "" {
		s.Mode = "rule"
	}
	switch strings.ToLower(s.Mode) {
	case "rule", "global", "direct":
	default:
		return fmt.Errorf("invalid mode %q", s.Mode)
	}
	if s.SubscriptionIntervalMin <= 0 {
		s.SubscriptionIntervalMin = 360
	}
	a.settings = &s
	if a.manager != nil && a.manager.Running() {
		_ = a.api.SetMode(s.Mode)
	}
	if a.scheduler != nil {
		a.scheduler.SetInterval(time.Duration(s.SubscriptionIntervalMin) * time.Minute)
		if s.AutoUpdateSubscriptions {
			a.scheduler.Start()
		} else {
			a.scheduler.Stop()
		}
	}
	if err := a.applyAutostartSetting(); err != nil {
		return err
	}
	return profiles.SaveSettings(a.paths.SettingsFile(), a.settings)
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
	return a.UpsertProfile(profiles.Profile{Name: name, Note: note, Proxies: nodes})
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

func (a *App) Connect() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.settings.ActiveProfile == "" {
		return fmt.Errorf("no active profile selected")
	}
	if err := winutil.RequireAdminForTUN(a.settings.TUN); err != nil {
		return err
	}

	checked, err := integrity.VerifyBeside(a.paths.CoreBinary())
	if err != nil {
		return err
	}
	if a.settings.RequireCoreHash && !checked {
		return fmt.Errorf("mihomo.exe.sha256 is required but missing")
	}

	resourceRoot := filepath.Dir(filepath.Dir(a.paths.CoreBinary()))
	if err := geo.Ensure(a.paths.CoreWorkDir(), resourceRoot); err != nil {
		// Non-fatal if offline and assets already absent; still try to connect.
		_ = err
	}

	profile, err := a.store.Get(a.settings.ActiveProfile)
	if err != nil {
		return err
	}
	doc, err := config.Build(config.BuildInput{
		Profile:       profile,
		MixedPort:     a.settings.MixedPort,
		ControllerURL: a.settings.ControllerURL,
		Secret:        a.settings.Secret,
		Mode:          a.settings.Mode,
		TUN:           a.settings.TUN,
		LogLevel:      a.settings.LogLevel,
	})
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
	}

	if a.settings.KillSwitch {
		if err := a.killSwitch.Enable(a.settings.VPNInterface); err != nil {
			_ = a.cleanupNetwork()
			_ = a.manager.Stop()
			return fmt.Errorf("kill switch: %w", err)
		}
	}
	if a.settings.DNSLeakProtection {
		if err := a.dnsGuard.Enable(); err != nil {
			_ = a.cleanupNetwork()
			_ = a.manager.Stop()
			return fmt.Errorf("dns leak protection: %w", err)
		}
	}

	if a.settings.AutoReconnect && a.health != nil {
		a.health.Start()
	}
	if a.settings.SelectedNode != "" {
		group := a.settings.ProxyGroup
		if group == "" {
			group = "PROXY"
		}
		if err := a.api.SelectProxy(group, a.settings.SelectedNode); err != nil {
			a.log.Warn("restore selected node: %v", err)
		} else {
			_ = a.api.CloseConnections()
		}
	}
	a.log.Info("connected profile=%s tun=%v mode=%s", a.settings.ActiveProfile, a.settings.TUN, a.settings.Mode)
	return nil
}

func (a *App) Disconnect() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.health != nil {
		a.health.Stop()
	}
	_ = a.cleanupNetwork()
	if a.manager == nil {
		return nil
	}
	return a.manager.Stop()
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
	return nil
}

func (a *App) ToggleConnect() error {
	if a.manager != nil && a.manager.Running() {
		return a.Disconnect()
	}
	return a.Connect()
}

func (a *App) GetTraffic() map[string]any {
	out := map[string]any{"up": 0, "down": 0, "totalUp": 0, "totalDown": 0}
	if a.manager == nil || !a.manager.Running() {
		return out
	}
	ctx, cancel := context.WithTimeout(context.Background(), 800*time.Millisecond)
	defer cancel()
	if snap, err := a.api.TrafficOnce(ctx); err == nil {
		out["up"] = snap.Up
		out["down"] = snap.Down
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
	helper, err := update.ApplyZip(zipPath, target)
	if err != nil {
		return err
	}
	a.log.Info("update helper started: %s", helper)
	_ = a.Disconnect()
	runtime.Quit(a.ctx)
	return nil
}

// ListNodes returns selectable proxies from the PROXY group.
func (a *App) ListNodes() ([]api.ProxyNodeRuntime, error) {
	if a.manager == nil || !a.manager.Running() {
		return []api.ProxyNodeRuntime{}, nil
	}
	group := a.settings.ProxyGroup
	if group == "" {
		group = "PROXY"
	}
	_, nodes, err := a.api.ListSelectableNodes(group)
	return nodes, err
}

// CurrentNode returns the active selector choice.
func (a *App) CurrentNode() (string, error) {
	if a.manager == nil || !a.manager.Running() {
		return a.settings.SelectedNode, nil
	}
	group := a.settings.ProxyGroup
	if group == "" {
		group = "PROXY"
	}
	g, err := a.api.Group(group)
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
	group := a.settings.ProxyGroup
	if group == "" {
		group = "PROXY"
	}
	if err := a.api.SelectProxy(group, name); err != nil {
		return err
	}
	_ = a.api.CloseConnections()
	a.settings.SelectedNode = name
	_ = profiles.SaveSettings(a.paths.SettingsFile(), a.settings)
	a.log.Info("selected node=%s group=%s", name, group)
	return nil
}

// TestNodeDelay measures latency for a node.
func (a *App) TestNodeDelay(name string) (int, error) {
	if a.manager == nil || !a.manager.Running() {
		return 0, fmt.Errorf("not connected")
	}
	return a.api.TestDelay(name, "", 5000)
}

// NodeDelayResult is one URL-test outcome.
type NodeDelayResult struct {
	Name  string `json:"name"`
	Delay int    `json:"delay"`
	Error string `json:"error,omitempty"`
}

// TestAllNodes runs URL-test against all selectable nodes (limited concurrency).
func (a *App) TestAllNodes() ([]NodeDelayResult, error) {
	nodes, err := a.ListNodes()
	if err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return []NodeDelayResult{}, nil
	}

	type job struct {
		name string
	}
	jobs := make(chan job)
	results := make(chan NodeDelayResult, len(nodes))

	workers := 5
	if len(nodes) < workers {
		workers = len(nodes)
	}
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				d, err := a.api.TestDelay(j.name, "", 5000)
				res := NodeDelayResult{Name: j.name, Delay: d}
				if err != nil {
					res.Error = err.Error()
					res.Delay = 0
				}
				results <- res
			}
		}()
	}
	go func() {
		for _, n := range nodes {
			if n.Name == "AUTO" {
				continue
			}
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
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Delay == 0 && out[j].Delay > 0 {
			return false
		}
		if out[j].Delay == 0 && out[i].Delay > 0 {
			return true
		}
		if out[i].Delay == out[j].Delay {
			return out[i].Name < out[j].Name
		}
		return out[i].Delay < out[j].Delay
	})
	return out, nil
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

// GetLogsTail returns the last N lines of client.log (best-effort).
func (a *App) GetLogsTail(maxLines int) (string, error) {
	if maxLines <= 0 {
		maxLines = 100
	}
	path := filepath.Join(a.paths.DataDir(), "client.log")
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	lines := strings.Split(string(data), "\n")
	if len(lines) > maxLines {
		lines = lines[len(lines)-maxLines:]
	}
	return strings.Join(lines, "\n"), nil
}
