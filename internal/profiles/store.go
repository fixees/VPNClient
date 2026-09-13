package profiles

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

// ProxyNode is a Clash/mihomo proxy definition (passthrough map).
type ProxyNode map[string]any

// Quota holds subscription traffic/expiry from provider headers.
type Quota struct {
	Upload     int64 `json:"upload"`
	Download   int64 `json:"download"`
	Total      int64 `json:"total"`
	ExpireUnix int64 `json:"expireUnix"`
}

// Used returns upload+download bytes.
func (q Quota) Used() int64 { return q.Upload + q.Download }

// Remaining returns leftover quota bytes, or -1 if unlimited/unknown.
func (q Quota) Remaining() int64 {
	if q.Total <= 0 {
		return -1
	}
	left := q.Total - q.Used()
	if left < 0 {
		return 0
	}
	return left
}

// Expired reports whether expire timestamp is in the past.
func (q Quota) Expired() bool {
	if q.ExpireUnix <= 0 {
		return false
	}
	return time.Now().Unix() >= q.ExpireUnix
}

// Profile stores a named set of proxies for a user subscription/key.
type Profile struct {
	Name                 string      `json:"name"`
	Note                 string      `json:"note,omitempty"`
	Proxies              []ProxyNode `json:"proxies"`
	SubscriptionURL      string      `json:"subscriptionURL,omitempty"`
	Quota                Quota       `json:"quota,omitempty"`
	LastSyncAt           string      `json:"lastSyncAt,omitempty"` // RFC3339
	UpdateIntervalHours  int         `json:"updateIntervalHours,omitempty"`
	ProviderTitle        string      `json:"providerTitle,omitempty"`
}

// Settings are persisted client preferences.
type Settings struct {
	ActiveProfile             string `json:"activeProfile"`
	MixedPort                 int    `json:"mixedPort"`
	ControllerURL             string `json:"controllerURL"`
	Secret                    string `json:"secret"`
	Mode                      string `json:"mode"` // rule | global | direct
	TUN                       bool   `json:"tun"`
	UseSystemProxy            bool   `json:"useSystemProxy"` // when TUN is off
	LogLevel                  string `json:"logLevel"`
	KillSwitch                bool   `json:"killSwitch"`
	DNSLeakProtection         bool   `json:"dnsLeakProtection"`
	AutoReconnect             bool   `json:"autoReconnect"`
	HealthIntervalSec         int    `json:"healthIntervalSec"`
	VPNInterface              string `json:"vpnInterface"`
	UpdateOwner               string `json:"updateOwner"`
	UpdateRepo                string `json:"updateRepo"`
	UpdateTag                 string `json:"updateTag"`
	RequireCoreHash           bool   `json:"requireCoreHash"`
	AutoUpdateSubscriptions   bool   `json:"autoUpdateSubscriptions"`
	SubscriptionIntervalMin   int    `json:"subscriptionIntervalMin"`
	Autostart                 bool   `json:"autostart"`
	CloseToTray               bool   `json:"closeToTray"`
	ProxyGroup                string `json:"proxyGroup"`
	SelectedNode              string `json:"selectedNode"`
}

func DefaultSettings() *Settings {
	return &Settings{
		MixedPort:               7890,
		ControllerURL:           "127.0.0.1:9090",
		Secret:                  "myinternetvpn-local",
		Mode:                    "rule",
		TUN:                     true,
		UseSystemProxy:          true,
		LogLevel:                "info",
		KillSwitch:              false,
		DNSLeakProtection:       false,
		AutoReconnect:           true,
		HealthIntervalSec:       30,
		VPNInterface:            "Meta",
		UpdateOwner:             "",
		UpdateRepo:              "",
		UpdateTag:               "latest-main",
		RequireCoreHash:         false,
		AutoUpdateSubscriptions: true,
		SubscriptionIntervalMin: 360,
		Autostart:               false,
		CloseToTray:             true,
		ProxyGroup:              "PROXY",
		SelectedNode:            "",
	}
}

// Store is a thread-safe JSON profile repository.
type Store struct {
	mu       sync.RWMutex
	path     string
	profiles []Profile
}

func NewStore(path string) *Store {
	return &Store{path: path, profiles: []Profile{}}
}

func (s *Store) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			s.profiles = []Profile{}
			return nil
		}
		return err
	}
	var list []Profile
	if err := json.Unmarshal(data, &list); err != nil {
		return err
	}
	s.profiles = list
	return nil
}

func (s *Store) Save() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := json.MarshalIndent(s.profiles, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0o600)
}

func (s *Store) List() ([]Profile, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Profile, len(s.profiles))
	copy(out, s.profiles)
	return out, nil
}

func (s *Store) Get(name string) (Profile, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, p := range s.profiles {
		if p.Name == name {
			return p, nil
		}
	}
	return Profile{}, fmt.Errorf("profile %q not found", name)
}

func (s *Store) Upsert(p Profile) error {
	if p.Name == "" {
		return fmt.Errorf("profile name is required")
	}
	if len(p.Proxies) == 0 {
		return fmt.Errorf("profile %q has no proxies", p.Name)
	}
	for i, proxy := range p.Proxies {
		if name, _ := proxy["name"].(string); name == "" {
			return fmt.Errorf("proxy at index %d is missing name", i)
		}
		if typ, _ := proxy["type"].(string); typ == "" {
			return fmt.Errorf("proxy %q is missing type", proxy["name"])
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	for i, existing := range s.profiles {
		if existing.Name == p.Name {
			s.profiles[i] = p
			return nil
		}
	}
	s.profiles = append(s.profiles, p)
	return nil
}

func (s *Store) Remove(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, p := range s.profiles {
		if p.Name == name {
			s.profiles = append(s.profiles[:i], s.profiles[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("profile %q not found", name)
}

func LoadSettings(path string) (*Settings, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	cfg := DefaultSettings()
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func SaveSettings(path string, cfg *Settings) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}
