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

func TestProfilesCopySubscriptionURL(t *testing.T) {
	store := profiles.NewStore(filepath.Join(t.TempDir(), "profiles.json"))

	// Test profile with subscription URL
	err := store.Upsert(profiles.Profile{
		Name:            "TestSub",
		SubscriptionURL: "https://example.com/sub?token=abc123",
		Proxies: []profiles.ProxyNode{
			{"name": "proxy-1", "type": "ss", "server": "example.com", "port": 443},
		},
	})
	if err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	// Test profile without subscription URL
	err = store.Upsert(profiles.Profile{
		Name: "TestLocal",
		Proxies: []profiles.ProxyNode{
			{"name": "proxy-2", "type": "ss", "server": "example.com", "port": 443},
		},
	})
	if err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	// Test that profile with subscription URL can be retrieved
	p, err := store.Get("TestSub")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if p.SubscriptionURL != "https://example.com/sub?token=abc123" {
		t.Fatalf("unexpected subscription URL: %s", p.SubscriptionURL)
	}

	// Test that profile without subscription URL returns empty
	p2, err := store.Get("TestLocal")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if p2.SubscriptionURL != "" {
		t.Fatalf("expected empty subscription URL, got: %s", p2.SubscriptionURL)
	}

	// Test that missing profile returns error
	_, err = store.Get("NonExistent")
	if err == nil {
		t.Fatal("expected error for non-existent profile")
	}
}
