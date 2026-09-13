package tests

import (
	"path/filepath"
	"testing"

	"myinternetvpn/client/internal/profiles"
)

func TestProfilesStoreUpsertGetRemove(t *testing.T) {
	path := filepath.Join(t.TempDir(), "profiles.json")
	store := profiles.NewStore(path)

	err := store.Upsert(profiles.Profile{
		Name: "Tokyo",
		Proxies: []profiles.ProxyNode{
			{"name": "tokyo-1", "type": "ss", "server": "example.com", "port": 443},
		},
	})
	if err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	if err := store.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	store2 := profiles.NewStore(path)
	if err := store2.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}
	p, err := store2.Get("Tokyo")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(p.Proxies) != 1 || p.Proxies[0]["name"] != "tokyo-1" {
		t.Fatalf("unexpected profile: %+v", p)
	}

	if err := store2.Remove("Tokyo"); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if _, err := store2.Get("Tokyo"); err == nil {
		t.Fatal("expected missing profile error")
	}
}

func TestProfilesUpsertValidation(t *testing.T) {
	store := profiles.NewStore(filepath.Join(t.TempDir(), "p.json"))
	if err := store.Upsert(profiles.Profile{Name: ""}); err == nil {
		t.Fatal("expected empty name error")
	}
	if err := store.Upsert(profiles.Profile{Name: "x", Proxies: nil}); err == nil {
		t.Fatal("expected empty proxies error")
	}
	if err := store.Upsert(profiles.Profile{
		Name:    "x",
		Proxies: []profiles.ProxyNode{{"type": "ss"}},
	}); err == nil {
		t.Fatal("expected missing proxy name error")
	}
}

func TestProfilesSettingsRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	cfg := profiles.DefaultSettings()
	cfg.ActiveProfile = "Work"
	cfg.TUN = false
	if err := profiles.SaveSettings(path, cfg); err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}
	loaded, err := profiles.LoadSettings(path)
	if err != nil {
		t.Fatalf("LoadSettings: %v", err)
	}
	if loaded.ActiveProfile != "Work" || loaded.TUN {
		t.Fatalf("unexpected settings: %+v", loaded)
	}
}
