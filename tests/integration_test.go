package tests

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"myinternetvpn/client/internal/config"
	"myinternetvpn/client/internal/core"
	"myinternetvpn/client/internal/paths"
	"myinternetvpn/client/internal/profiles"
)

// TestConnectPipeline exercises profile → config YAML → manager start/stop.
func TestConnectPipeline(t *testing.T) {
	tmp := t.TempDir()
	resolver := paths.NewWithRoots(filepath.Join(tmp, "data"), filepath.Join(tmp, "resources"))
	if err := resolver.Ensure(); err != nil {
		t.Fatalf("Ensure: %v", err)
	}

	store := profiles.NewStore(resolver.ProfilesFile())
	profile := profiles.Profile{
		Name: "Work VPN",
		Note: "integration",
		Proxies: []profiles.ProxyNode{
			{"name": "node-1", "type": "ss", "server": "example.com", "port": 443, "cipher": "aes-128-gcm", "password": "x"},
		},
	}
	if err := store.Upsert(profile); err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	if err := store.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	settings := profiles.DefaultSettings()
	settings.ActiveProfile = profile.Name
	settings.TUN = false
	settings.MixedPort = 17890
	if err := profiles.SaveSettings(resolver.SettingsFile(), settings); err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}

	loaded, err := store.Get(settings.ActiveProfile)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	yamlDoc, err := config.Build(config.BuildInput{
		Profile:       loaded,
		MixedPort:     settings.MixedPort,
		ControllerURL: settings.ControllerURL,
		Secret:        settings.Secret,
		Mode:          settings.Mode,
		TUN:           settings.TUN,
		LogLevel:      settings.LogLevel,
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	bin := filepath.Join(tmp, "data", "core", "mihomo.exe")
	if err := os.MkdirAll(filepath.Dir(bin), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bin, []byte("fake"), 0o755); err != nil {
		t.Fatal(err)
	}

	runner := &fakeRunner{}
	mgr := core.NewManager(core.Options{
		CorePath:     bin,
		WorkDir:      resolver.CoreWorkDir(),
		ConfigPath:   resolver.RuntimeConfig(),
		API:          &fakeAPI{healthy: true},
		Runner:       runner,
		ReadyTimeout: time.Second,
	})
	if err := mgr.Start(yamlDoc); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if !mgr.Running() {
		t.Fatal("expected running")
	}
	if _, err := os.Stat(resolver.RuntimeConfig()); err != nil {
		t.Fatalf("runtime config missing: %v", err)
	}
	if err := mgr.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}
}
