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
	DownloadTotal int64              `json:"downloadTotal"`
	UploadTotal   int64              `json:"uploadTotal"`
	Connections   []ConnectionDetail `json:"connections"`
}

// ConnectionDetail is a single active connection from mihomo.
type ConnectionDetail struct {
	ID       string                 `json:"id"`
	Metadata map[string]interface{} `json:"metadata"`
	Upload   int64                  `json:"upload"`
	Download int64                  `json:"download"`
	Start    string                 `json:"start"`
	Chains   []string               `json:"chains"`
	Rule     string                 `json:"rule"`
	RulePayload string              `json:"rulePayload"`
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

	mu          sync.Mutex
	cachedVer   string
	cachedVerAt time.Time

	// Live rates from the persistent /traffic stream (and totals delta fallback).
	trafficUp      int64
	trafficDown    int64
	trafficAt      time.Time
	trafficCancel  context.CancelFunc
	trafficRunning bool

	cachedMode   string
	cachedModeAt time.Time

	prevTotalUp   int64
	prevTotalDown int64
	prevTotalsAt  time.Time
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
	c.StopTrafficStream()
	c.mu.Lock()
	c.baseURL = strings.TrimRight(base, "/")
	c.secret = secret
	c.cachedVer = ""
	c.cachedVerAt = time.Time{}
	c.cachedMode = ""
	c.cachedModeAt = time.Time{}
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
	if _, _, age, ok := c.CachedTraffic(); ok && age < defaults.TrafficFreshness*2 {
		return true
	}
	_, err := c.Version()
	return err == nil
}

func (c *Client) trafficURL() string {
	return fmt.Sprintf("%s/traffic?interval=%d", c.BaseURL(), defaults.TrafficStreamInterval)
}

func (c *Client) setTrafficRates(up, down int64) {
	c.mu.Lock()
	c.trafficUp = up
	c.trafficDown = down
	c.trafficAt = time.Now()
	c.mu.Unlock()
}

// CachedTraffic returns the latest rates from the background stream (or zero).
func (c *Client) CachedTraffic() (up, down int64, age time.Duration, ok bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.trafficAt.IsZero() {
		return 0, 0, 0, false
	}
	return c.trafficUp, c.trafficDown, time.Since(c.trafficAt), true
}

// StartTrafficStream keeps a long-lived /traffic reader so UI polls stay fast.
func (c *Client) StartTrafficStream() {
	c.mu.Lock()
	if c.trafficRunning {
		c.mu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	c.trafficCancel = cancel
	c.trafficRunning = true
	c.mu.Unlock()
	go c.runTrafficStream(ctx)
}

// StopTrafficStream cancels the background reader and clears rates.
func (c *Client) StopTrafficStream() {
	c.mu.Lock()
	cancel := c.trafficCancel
	c.trafficCancel = nil
	c.trafficRunning = false
	c.trafficUp = 0
	c.trafficDown = 0
	c.trafficAt = time.Time{}
	c.prevTotalUp = 0
	c.prevTotalDown = 0
	c.prevTotalsAt = time.Time{}
	c.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (c *Client) runTrafficStream(ctx context.Context) {
	defer func() {
		c.mu.Lock()
		c.trafficRunning = false
		c.mu.Unlock()
	}()
	backoff := 300 * time.Millisecond
	for {
		if ctx.Err() != nil {
			return
		}
		err := c.consumeTrafficStream(ctx)
		if ctx.Err() != nil {
			return
		}
		if err != nil {
			time.Sleep(backoff)
			if backoff < 2*time.Second {
				backoff *= 2
			}
			continue
		}
		backoff = 300 * time.Millisecond
	}
}

func (c *Client) consumeTrafficStream(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.trafficURL(), nil)
	if err != nil {
		return err
	}
	c.applyAuth(req)
	resp, err := c.streamClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("api /traffic: status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	reader := bufio.NewReader(resp.Body)
	for {
		line, err := reader.ReadBytes('\n')
		if len(line) > 0 {
			var snap TrafficSnapshot
			if json.Unmarshal(bytes.TrimSpace(line), &snap) == nil {
				c.setTrafficRates(snap.Up, snap.Down)
			}
		}
		if err != nil {
			if err == io.EOF || ctx.Err() != nil {
				return nil
			}
			return err
		}
	}
}

// TrafficOnce reads a single sample from the chunked /traffic stream.
func (c *Client) TrafficOnce(ctx context.Context) (TrafficSnapshot, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.trafficURL(), nil)
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
	c.setTrafficRates(snap.Up, snap.Down)
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

// noteRatesFromTotals derives B/s from cumulative counters between polls.
func (c *Client) noteRatesFromTotals(totalUp, totalDown int64) (up, down int64, ok bool) {
	now := time.Now()
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.prevTotalsAt.IsZero() {
		dt := now.Sub(c.prevTotalsAt).Seconds()
		if dt >= 0.2 && dt <= 30 {
			up = int64(float64(totalUp-c.prevTotalUp) / dt)
			down = int64(float64(totalDown-c.prevTotalDown) / dt)
			if up < 0 {
				up = 0
			}
			if down < 0 {
				down = 0
			}
			ok = true
		}
	}
	c.prevTotalUp = totalUp
	c.prevTotalDown = totalDown
	c.prevTotalsAt = now
	return up, down, ok
}

func (c *Client) cachedModeValue() (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cachedMode == "" {
		return "", false
	}
	if time.Since(c.cachedModeAt) > defaults.ModeCacheTTL {
		return "", false
	}
	return c.cachedMode, true
}

func (c *Client) rememberMode(mode string) {
	mode = strings.TrimSpace(mode)
	if mode == "" {
		return
	}
	c.mu.Lock()
	c.cachedMode = mode
	c.cachedModeAt = time.Now()
	c.mu.Unlock()
}

// Live gathers version/mode/traffic for UI polls without blocking on /connections.
func (c *Client) Live(ctx context.Context) LiveSnapshot {
	c.StartTrafficStream()

	var (
		ver, mode string
		up, down  int64
		wg        sync.WaitGroup
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

	if cached, ok := c.cachedModeValue(); ok {
		mode = cached
	} else {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if m, err := c.Mode(); err == nil {
				mode = m
				c.rememberMode(m)
			}
		}()
	}

	wg.Wait()

	if cachedUp, cachedDown, age, ok := c.CachedTraffic(); ok && age < defaults.TrafficFreshness {
		up, down = cachedUp, cachedDown
	} else {
		// Cold start only: one sample. Soft polls must not block ~2s every tick.
		select {
		case <-ctx.Done():
		default:
			tctx, cancel := context.WithTimeout(ctx, defaults.TrafficSampleTimeout)
			if snap, err := c.TrafficOnce(tctx); err == nil {
				up, down = snap.Up, snap.Down
			}
			cancel()
		}
	}

	return LiveSnapshot{
		Version: ver,
		Mode:    mode,
		Up:      up,
		Down:    down,
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
	if err := c.doJSON(http.MethodPatch, "/configs", payload, nil); err != nil {
		return err
	}
	c.rememberMode(mode)
	return nil
}

// ReloadConfig asks mihomo to reload rules/DNS from an on-disk config path.
func (c *Client) ReloadConfig(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return fmt.Errorf("config path is empty")
	}
	payload, err := json.Marshal(map[string]string{"path": path})
	if err != nil {
		return err
	}
	return c.doJSON(http.MethodPut, "/configs?force=true", payload, nil)
}

// Mode reads current mode from /configs.
func (c *Client) Mode() (string, error) {
	var cfg struct {
		Mode string `json:"mode"`
	}
	if err := c.getJSON("/configs", &cfg); err != nil {
		return "", err
	}
	return cfg.Mode, nil
}

func (c *Client) getJSON(path string, dest any) error {
	return c.doJSON(http.MethodGet, path, nil, dest)
}

func (c *Client) getJSONWithTimeout(path string, timeout time.Duration, dest any) error {
	if timeout <= 0 {
		timeout = defaults.APITimeout
	}
	req, err := http.NewRequest(http.MethodGet, c.BaseURL()+path, nil)
	if err != nil {
		return err
	}
	c.applyAuth(req)
	client := &http.Client{
		Timeout:   timeout,
		Transport: c.httpClient.Transport,
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode >= 300 {
		return fmt.Errorf("api GET %s: status %d: %s", path, resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	if dest == nil || len(raw) == 0 {
		return nil
	}
	return json.Unmarshal(raw, dest)
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
