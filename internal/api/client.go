package api

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
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

// Client talks to mihomo external-controller HTTP API.
type Client struct {
	baseURL    string
	secret     string
	httpClient *http.Client
}

func NewClient(controllerAddr, secret string) *Client {
	base := controllerAddr
	if !strings.HasPrefix(base, "http://") && !strings.HasPrefix(base, "https://") {
		base = "http://" + base
	}
	base = strings.TrimRight(base, "/")
	return &Client{
		baseURL: base,
		secret:  secret,
		httpClient: &http.Client{
			Timeout: 3 * time.Second,
		},
	}
}

// BaseURL returns the normalized controller base URL.
func (c *Client) BaseURL() string {
	return c.baseURL
}

func (c *Client) Version() (VersionInfo, error) {
	var out VersionInfo
	if err := c.getJSON("/version", &out); err != nil {
		return VersionInfo{}, err
	}
	return out, nil
}

func (c *Client) Healthy() bool {
	_, err := c.Version()
	return err == nil
}

// TrafficOnce reads a single sample from the chunked /traffic stream.
func (c *Client) TrafficOnce(ctx context.Context) (TrafficSnapshot, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/traffic", nil)
	if err != nil {
		return TrafficSnapshot{}, err
	}
	c.applyAuth(req)

	client := &http.Client{Timeout: 0} // stream; rely on ctx
	resp, err := client.Do(req)
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
	req, err := http.NewRequest(method, c.baseURL+path, reader)
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
	raw, err := io.ReadAll(resp.Body)
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
	if c.secret != "" {
		req.Header.Set("Authorization", "Bearer "+c.secret)
	}
}

// WaitHealthy polls until the controller answers or timeout elapses.
func (c *Client) WaitHealthy(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	var last error
	for time.Now().Before(deadline) {
		if c.Healthy() {
			return nil
		}
		last = fmt.Errorf("controller not ready")
		time.Sleep(150 * time.Millisecond)
	}
	if last == nil {
		last = fmt.Errorf("controller not ready")
	}
	return last
}
