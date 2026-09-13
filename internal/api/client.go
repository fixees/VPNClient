package api

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"myinternetvpn/client/internal/defaults"
)

// VersionInfo is returned by GET /version.
type VersionInfo struct {
	Meta    bool   `json:"meta"`
	Version string `json:"version"`
}

// TrafficSnapshot is instantaneous up/down rates from /traffic stream.
type TrafficSnapshot struct {
	Up   int64 `json:"up"`
	Down int64 `json:"down"`
}

// ConnectionsInfo is returned by GET /connections.
type ConnectionsInfo struct {
	DownloadTotal int64 `json:"downloadTotal"`
	UploadTotal   int64 `json:"uploadTotal"`
}

// LiveSnapshot is a single round-trip bundle for UI polling.
type LiveSnapshot struct {
	Version  string
	Mode     string
	Up       int64
	Down     int64
	TotalUp  int64
	TotalDown int64
}

// Client talks to mihomo external-controller HTTP API.
type Client struct {
	baseURL      string
	secret       string
	httpClient   *http.Client
	streamClient *http.Client

	mu           sync.Mutex
	cachedVer    string
	cachedVerAt  time.Time
}

func NewClient(controllerAddr, secret string) *Client {
	base := controllerAddr
	if !strings.HasPrefix(base, "http://") && !strings.HasPrefix(base, "https://") {
		base = "http://" + base
	}
	base = strings.TrimRight(base, "/")
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   2 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:        8,
		IdleConnTimeout:     60 * time.Second,
		DisableCompression:  true,
		ForceAttemptHTTP2:   false,
	}
	return &Client{
		baseURL: base,
		secret:  secret,
		httpClient: &http.Client{
			Timeout:   defaults.APITimeout,
			Transport: transport,
		},
		streamClient: &http.Client{
			Timeout:   0, // streaming; callers bind ctx
			Transport: transport,
		},
	}
}

// UpdateEndpoint switches controller address/secret (after settings change).
func (c *Client) UpdateEndpoint(controllerAddr, secret string) {
	base := controllerAddr
	if !strings.HasPrefix(base, "http://") && !strings.HasPrefix(base, "https://") {
		base = "http://" + base
	}
	c.mu.Lock()
	c.baseURL = strings.TrimRight(base, "/")
	c.secret = secret
	c.cachedVer = ""
	c.cachedVerAt = time.Time{}
	c.mu.Unlock()
}

// BaseURL returns the normalized controller base URL.
func (c *Client) BaseURL() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.baseURL
}

func (c *Client) Version() (VersionInfo, error) {
	var out VersionInfo
	if err := c.getJSON("/version", &out); err != nil {
		return VersionInfo{}, err
	}
	c.mu.Lock()
	c.cachedVer = out.Version
	c.cachedVerAt = time.Now()
	c.mu.Unlock()
	return out, nil
}

func (c *Client) cachedVersion() (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cachedVer == "" {
		return "", false
	}
	if time.Since(c.cachedVerAt) > defaults.VersionCacheTTL {
		return "", false
	}
	return c.cachedVer, true
}

func (c *Client) Healthy() bool {
	_, err := c.Version()
	return err == nil
}

// TrafficOnce reads a single sample from the chunked /traffic stream.
func (c *Client) TrafficOnce(ctx context.Context) (TrafficSnapshot, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL()+"/traffic", nil)
	if err != nil {
		return TrafficSnapshot{}, err
	}
	c.applyAuth(req)

	resp, err := c.streamClient.Do(req)
	if err != nil {
		return TrafficSnapshot{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return TrafficSnapshot{}, fmt.Errorf("api /traffic: status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	reader := bufio.NewReader(resp.Body)
	line, err := reader.ReadBytes('\n')
	if err != nil && len(line) == 0 {
		return TrafficSnapshot{}, err
	}
	var snap TrafficSnapshot
	if err := json.Unmarshal(bytes.TrimSpace(line), &snap); err != nil {
		return TrafficSnapshot{}, err
	}
	return snap, nil
}

// Connections returns cumulative traffic counters.
func (c *Client) Connections() (ConnectionsInfo, error) {
	var out ConnectionsInfo
	if err := c.getJSON("/connections", &out); err != nil {
		return ConnectionsInfo{}, err
	}
	return out, nil
}

// Live gathers version/mode/traffic/totals concurrently for UI polls.
func (c *Client) Live(ctx context.Context) LiveSnapshot {
	var (
		ver, mode string
		up, down, totalUp, totalDown int64
		wg sync.WaitGroup
	)

	if cached, ok := c.cachedVersion(); ok {
		ver = cached
	} else {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if v, err := c.Version(); err == nil {
				ver = v.Version
			}
		}()
	}

	wg.Add(3)
	go func() {
		defer wg.Done()
		if m, err := c.Mode(); err == nil {
			mode = m
		}
	}()
	go func() {
		defer wg.Done()
		tctx, cancel := context.WithTimeout(ctx, defaults.TrafficSampleTimeout)
		defer cancel()
		if snap, err := c.TrafficOnce(tctx); err == nil {
			up, down = snap.Up, snap.Down
		}
	}()
	go func() {
		defer wg.Done()
		if conn, err := c.Connections(); err == nil {
			totalUp, totalDown = conn.UploadTotal, conn.DownloadTotal
		}
	}()
	wg.Wait()
	return LiveSnapshot{
		Version:   ver,
		Mode:      mode,
		Up:        up,
		Down:      down,
		TotalUp:   totalUp,
		TotalDown: totalDown,
	}
}

// SetMode patches runtime mode: rule | global | direct.
func (c *Client) SetMode(mode string) error {
	mode = strings.ToLower(strings.TrimSpace(mode))
	switch mode {
	case "rule", "global", "direct":
	default:
		return fmt.Errorf("unsupported mode %q", mode)
	}
	payload, _ := json.Marshal(map[string]string{"mode": mode})
	return c.doJSON(http.MethodPatch, "/configs", payload, nil)
}

// Mode reads current mode from /configs.
func (c *Client) Mode() (string, error) {
	var cfg map[string]any
	if err := c.getJSON("/configs", &cfg); err != nil {
		return "", err
	}
	mode, _ := cfg["mode"].(string)
	return mode, nil
}

func (c *Client) getJSON(path string, dest any) error {
	return c.doJSON(http.MethodGet, path, nil, dest)
}

func (c *Client) doJSON(method, path string, body []byte, dest any) error {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequest(method, c.BaseURL()+path, reader)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	c.applyAuth(req)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode >= 300 {
		return fmt.Errorf("api %s %s: status %d: %s", method, path, resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	if dest == nil || len(raw) == 0 {
		return nil
	}
	return json.Unmarshal(raw, dest)
}

func (c *Client) applyAuth(req *http.Request) {
	c.mu.Lock()
	secret := c.secret
	c.mu.Unlock()
	if secret != "" {
		req.Header.Set("Authorization", "Bearer "+secret)
	}
}

// WaitHealthy polls until the controller answers or timeout elapses.
func (c *Client) WaitHealthy(timeout time.Duration) error {
	if timeout <= 0 {
		timeout = defaults.ReadyTimeout
	}
	deadline := time.Now().Add(timeout)
	var last error
	for time.Now().Before(deadline) {
		if _, err := c.Version(); err == nil {
			return nil
		} else {
			last = err
		}
		time.Sleep(defaults.HealthPollInterval)
	}
	if last == nil {
		last = fmt.Errorf("controller not ready")
	}
	return last
}
