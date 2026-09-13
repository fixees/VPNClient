package profiles

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"myinternetvpn/client/internal/defaults"
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
	Name                string      `json:"name"`
	Note                string      `json:"note,omitempty"`
	Proxies             []ProxyNode `json:"proxies"`
	SubscriptionURL     string      `json:"subscriptionURL,omitempty"`
	Quota               Quota       `json:"quota,omitempty"`
	LastSyncAt          string      `json:"lastSyncAt,omitempty"` // RFC3339
	UpdateIntervalHours int         `json:"updateIntervalHours,omitempty"`
	ProviderTitle       string      `json:"providerTitle,omitempty"`
}

// Settings are persisted client preferences.
type Settings struct {
	ActiveProfile           string `json:"activeProfile"`
	MixedPort               int    `json:"mixedPort"`
	ControllerURL           string `json:"controllerURL"`
	Secret                  string `json:"secret"`
	Mode                    string `json:"mode"` // rule | global | direct
	TUN                     bool   `json:"tun"`
	TUNStack                string `json:"tunStack"` // system | gvisor | mixed
	UseSystemProxy          bool   `json:"useSystemProxy"`
	AllowLAN                bool   `json:"allowLan"`
	IPv6                    bool   `json:"ipv6"`
	LogLevel                string `json:"logLevel"`
	KillSwitch              bool   `json:"killSwitch"`
	DNSLeakProtection       bool   `json:"dnsLeakProtection"`
	AutoReconnect           bool   `json:"autoReconnect"`
	HealthIntervalSec       int    `json:"healthIntervalSec"`
	VPNInterface            string `json:"vpnInterface"`
	BypassLAN               bool   `json:"bypassLan"`
	BypassGEOIP             string `json:"bypassGeoip"`     // e.g. CN; empty = off
	DNSEnhancedMode         string `json:"dnsEnhancedMode"` // fake-ip | redir-host
	DNSNameservers          string `json:"dnsNameservers"`  // comma/space separated
	DNSFallbacks            string `json:"dnsFallbacks"`
	DNSFakeIPRange          string `json:"dnsFakeIpRange"`
	Sniffer                 bool   `json:"sniffer"`
	TCPConcurrent           bool   `json:"tcpConcurrent"`
	UnifiedDelay            bool   `json:"unifiedDelay"`
	WARPEnabled             bool   `json:"warpEnabled"`
	WARPPrivateKey          string `json:"warpPrivateKey"`
	WARPLocalAddress        string `json:"warpLocalAddress"`
	WARPEndpoint            string `json:"warpEndpoint"`
	WARPPublicKey           string `json:"warpPublicKey"`
	UpdateOwner             string `json:"updateOwner"`
	UpdateRepo              string `json:"updateRepo"`
	UpdateTag               string `json:"updateTag"`
	RequireCoreHash         bool   `json:"requireCoreHash"`
	AutoUpdateSubscriptions bool   `json:"autoUpdateSubscriptions"`
	SubscriptionIntervalMin int    `json:"subscriptionIntervalMin"`
	Autostart               bool   `json:"autostart"`
	CloseToTray             bool   `json:"closeToTray"`
	ProxyGroup              string `json:"proxyGroup"`
	SelectedNode            string `json:"selectedNode"`
}

func DefaultSettings() *Settings {
	return &Settings{
		MixedPort:               defaults.MixedPort,
		ControllerURL:           defaults.ControllerAddr,
		Secret:                  defaults.NewLocalSecret(),
		Mode:                    defaults.DefaultMode,
		TUN:                     true,
		TUNStack:                defaults.DefaultTUNStack,
		UseSystemProxy:          true,
		AllowLAN:                false,
		IPv6:                    false,
		LogLevel:                defaults.DefaultLogLevel,
		KillSwitch:              false,
		DNSLeakProtection:       false,
		AutoReconnect:           true,
		HealthIntervalSec:       int(defaults.DefaultHealthInterval / time.Second),
		VPNInterface:            defaults.DefaultVPNIface,
		BypassLAN:               true,
		BypassGEOIP:             defaults.DefaultBypassGEOIP,
		DNSEnhancedMode:         defaults.DNSEnhancedMode,
		DNSNameservers:          stringsJoin(defaults.DNSNameservers),
		DNSFallbacks:            stringsJoin(defaults.DNSFallbacks),
		DNSFakeIPRange:          defaults.DNSFakeIPRange,
		Sniffer:                 true,
		TCPConcurrent:           true,
		UnifiedDelay:            true,
		WARPEnabled:             false,
		WARPPrivateKey:          "",
		WARPLocalAddress:        defaults.WARPLocalAddress,
		WARPEndpoint:            defaults.WARPEndpoint,
		WARPPublicKey:           defaults.WARPPublicKey,
		UpdateOwner:             defaults.UpdateOwner,
		UpdateRepo:              defaults.UpdateRepo,
		UpdateTag:               defaults.UpdateTag,
		RequireCoreHash:         false,
		AutoUpdateSubscriptions: true,
		SubscriptionIntervalMin: defaults.DefaultSubIntervalMin,
		Autostart:               false,
		CloseToTray:             true,
		ProxyGroup:              defaults.ProxyGroup,
		SelectedNode:            "",
	}
}

func stringsJoin(items []string) string {
	if len(items) == 0 {
		return ""
	}
	out := items[0]
	for i := 1; i < len(items); i++ {
		out += ", " + items[i]
	}
	return out
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
	// Keep a generated secret only if file omits it.
	generatedSecret := cfg.Secret
	cfg.Secret = ""
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	NormalizeSettings(cfg, generatedSecret)
	return cfg, nil
}

func SaveSettings(path string, cfg *Settings) error {
	NormalizeSettings(cfg, "")
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

// NormalizeSettings fills empty/zero fields from defaults.
func NormalizeSettings(cfg *Settings, fallbackSecret string) {
	if cfg == nil {
		return
	}
	if cfg.MixedPort <= 0 {
		cfg.MixedPort = defaults.MixedPort
	}
	if cfg.ControllerURL == "" {
		cfg.ControllerURL = defaults.ControllerAddr
	}
	if cfg.Secret == "" {
		if fallbackSecret != "" {
			cfg.Secret = fallbackSecret
		} else {
			cfg.Secret = defaults.NewLocalSecret()
		}
	}
	if cfg.Mode == "" {
		cfg.Mode = defaults.DefaultMode
	}
	if cfg.LogLevel == "" {
		cfg.LogLevel = defaults.DefaultLogLevel
	}
	switch strings.ToLower(cfg.TUNStack) {
	case "system", "gvisor", "mixed":
		cfg.TUNStack = strings.ToLower(cfg.TUNStack)
	default:
		cfg.TUNStack = defaults.DefaultTUNStack
	}
	switch strings.ToLower(cfg.DNSEnhancedMode) {
	case "fake-ip", "redir-host":
		cfg.DNSEnhancedMode = strings.ToLower(cfg.DNSEnhancedMode)
	default:
		cfg.DNSEnhancedMode = defaults.DNSEnhancedMode
	}
	if cfg.DNSNameservers == "" {
		cfg.DNSNameservers = stringsJoin(defaults.DNSNameservers)
	}
	if cfg.DNSFallbacks == "" {
		cfg.DNSFallbacks = stringsJoin(defaults.DNSFallbacks)
	}
	if cfg.DNSFakeIPRange == "" {
		cfg.DNSFakeIPRange = defaults.DNSFakeIPRange
	}
	if cfg.HealthIntervalSec <= 0 {
		cfg.HealthIntervalSec = int(defaults.DefaultHealthInterval / time.Second)
	}
	if cfg.VPNInterface == "" {
		cfg.VPNInterface = defaults.DefaultVPNIface
	}
	if cfg.UpdateTag == "" {
		cfg.UpdateTag = defaults.UpdateTag
	}
	if cfg.UpdateOwner == "" {
		cfg.UpdateOwner = defaults.UpdateOwner
	}
	if cfg.UpdateRepo == "" {
		cfg.UpdateRepo = defaults.UpdateRepo
	}
	if cfg.SubscriptionIntervalMin <= 0 {
		cfg.SubscriptionIntervalMin = defaults.DefaultSubIntervalMin
	}
	if cfg.ProxyGroup == "" {
		cfg.ProxyGroup = defaults.ProxyGroup
	}
	if cfg.WARPEndpoint == "" {
		cfg.WARPEndpoint = defaults.WARPEndpoint
	}
	if cfg.WARPPublicKey == "" {
		cfg.WARPPublicKey = defaults.WARPPublicKey
	}
	if cfg.WARPLocalAddress == "" {
		cfg.WARPLocalAddress = defaults.WARPLocalAddress
	}
	cfg.BypassGEOIP = strings.ToUpper(strings.TrimSpace(cfg.BypassGEOIP))
}

// SplitList parses a comma/space/newline separated list.
func SplitList(raw string) []string {
	raw = strings.ReplaceAll(raw, "\n", ",")
	raw = strings.ReplaceAll(raw, ";", ",")
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\t'
	})
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
