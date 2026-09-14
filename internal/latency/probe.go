package latency

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"myinternetvpn/client/internal/api"
	"myinternetvpn/client/internal/defaults"
	"myinternetvpn/client/internal/profiles"
	"myinternetvpn/client/internal/winutil"
)

// Options for a single node probe.
type Options struct {
	Method    string
	TestURL   string
	Timeout   time.Duration
	MixedPort int
	API       *api.Client
	Proxy     profiles.ProxyNode // leaf node definition (server/port)
}

// Probe measures latency for one node according to method.
func Probe(ctx context.Context, opts Options) (int, error) {
	if opts.Timeout <= 0 {
		opts.Timeout = time.Duration(defaults.URLTestTimeoutMS) * time.Millisecond
	}
	if opts.TestURL == "" {
		opts.TestURL = defaults.URLTestURL
	}
	method := strings.ToLower(strings.TrimSpace(opts.Method))
	if method == "" {
		method = defaults.DefaultPingMethod
	}

	switch method {
	case defaults.PingProxyHTTPGet:
		return probeProxyHTTPGet(opts)
	case defaults.PingProxyHTTPHead:
		return probeViaLocalProxy(ctx, opts, http.MethodHead)
	case defaults.PingHTTPGet:
		return probeDirectHTTP(ctx, opts, http.MethodGet)
	case defaults.PingTCP:
		return probeTCP(ctx, opts)
	case defaults.PingICMP:
		return probeICMP(ctx, opts)
	default:
		return 0, fmt.Errorf("unknown ping method %q", method)
	}
}

func probeProxyHTTPGet(opts Options) (int, error) {
	if opts.API == nil {
		return 0, fmt.Errorf("api client required")
	}
	name, _ := opts.Proxy["name"].(string)
	if name == "" {
		return 0, fmt.Errorf("proxy name required")
	}
	return opts.API.TestDelay(name, opts.TestURL, int(opts.Timeout/time.Millisecond))
}

func probeViaLocalProxy(ctx context.Context, opts Options, method string) (int, error) {
	if opts.MixedPort <= 0 {
		opts.MixedPort = defaults.MixedPort
	}
	name, _ := opts.Proxy["name"].(string)
	if name == "" {
		return 0, fmt.Errorf("proxy name required")
	}
	if opts.API == nil {
		return 0, fmt.Errorf("api client required")
	}
	// Select node then issue request through mixed port.
	if err := opts.API.SelectProxy(defaults.ProxyGroup, name); err != nil {
		return 0, err
	}
	proxyURL, err := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", opts.MixedPort))
	if err != nil {
		return 0, err
	}
	client := &http.Client{
		Timeout: opts.Timeout,
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	req, err := http.NewRequestWithContext(ctx, method, opts.TestURL, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("User-Agent", defaults.UserAgent)
	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64<<10))
	_ = resp.Body.Close()
	ms := int(time.Since(start) / time.Millisecond)
	if ms <= 0 {
		ms = 1
	}
	return ms, nil
}

func probeDirectHTTP(ctx context.Context, opts Options, method string) (int, error) {
	client := &http.Client{
		Timeout: opts.Timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	req, err := http.NewRequestWithContext(ctx, method, opts.TestURL, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("User-Agent", defaults.UserAgent)
	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64<<10))
	_ = resp.Body.Close()
	ms := int(time.Since(start) / time.Millisecond)
	if ms <= 0 {
		ms = 1
	}
	return ms, nil
}

func probeTCP(ctx context.Context, opts Options) (int, error) {
	host, port, err := ProxyEndpoint(opts.Proxy)
	if err != nil {
		return 0, err
	}
	d := net.Dialer{Timeout: opts.Timeout}
	start := time.Now()
	conn, err := d.DialContext(ctx, "tcp", net.JoinHostPort(host, strconv.Itoa(port)))
	if err != nil {
		return 0, err
	}
	_ = conn.Close()
	ms := int(time.Since(start) / time.Millisecond)
	if ms <= 0 {
		ms = 1
	}
	return ms, nil
}

var pingTimeRe = regexp.MustCompile(`(?i)(?:time[=<]|время[=<])\s*([\d.]+)\s*ms`)

func probeICMP(ctx context.Context, opts Options) (int, error) {
	host, _, err := ProxyEndpoint(opts.Proxy)
	if err != nil {
		return 0, err
	}
	// Prefer OS ping — works without raw sockets / admin on Windows.
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "ping", "-n", "1", "-w", strconv.Itoa(int(opts.Timeout/time.Millisecond)), host)
	} else {
		sec := int(opts.Timeout / time.Second)
		if sec < 1 {
			sec = 1
		}
		cmd = exec.CommandContext(ctx, "ping", "-c", "1", "-W", strconv.Itoa(sec), host)
	}
	winutil.HideConsole(cmd)
	start := time.Now()
	out, err := cmd.CombinedOutput()
	elapsed := int(time.Since(start) / time.Millisecond)
	text := string(out)
	if m := pingTimeRe.FindStringSubmatch(text); len(m) == 2 {
		if f, perr := strconv.ParseFloat(m[1], 64); perr == nil {
			ms := int(f)
			if ms <= 0 {
				ms = 1
			}
			return ms, nil
		}
	}
	if err != nil {
		return 0, fmt.Errorf("icmp: %v (%s)", err, strings.TrimSpace(text))
	}
	if elapsed <= 0 {
		elapsed = 1
	}
	return elapsed, nil
}

// ProxyEndpoint extracts server host/port from a Clash proxy map.
func ProxyEndpoint(p profiles.ProxyNode) (host string, port int, err error) {
	if p == nil {
		return "", 0, fmt.Errorf("empty proxy")
	}
	host, _ = p["server"].(string)
	if host == "" {
		return "", 0, fmt.Errorf("proxy missing server")
	}
	switch v := p["port"].(type) {
	case int:
		port = v
	case int64:
		port = int(v)
	case float64:
		port = int(v)
	case string:
		port, _ = strconv.Atoi(v)
	}
	if port <= 0 {
		return "", 0, fmt.Errorf("proxy missing port")
	}
	return host, port, nil
}

// FindProxy returns a named proxy from a profile (exact name match).
func FindProxy(list []profiles.ProxyNode, name string) (profiles.ProxyNode, bool) {
	for _, p := range list {
		if n, _ := p["name"].(string); n == name {
			return p, true
		}
	}
	return nil, false
}
