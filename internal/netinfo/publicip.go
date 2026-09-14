package netinfo

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"myinternetvpn/client/internal/defaults"
)

// LookupPublicIP returns the apparent public IP.
// If mixedPort > 0, the request goes through the local mixed proxy (VPN exit IP).
func LookupPublicIP(ctx context.Context, mixedPort int) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout: 4 * time.Second,
		}).DialContext,
		DisableKeepAlives: true,
	}
	if mixedPort > 0 {
		proxyURL, err := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", mixedPort))
		if err != nil {
			return "", err
		}
		transport.Proxy = http.ProxyURL(proxyURL)
	}
	client := &http.Client{
		Timeout:   6 * time.Second,
		Transport: transport,
	}

	endpoints := []string{
		"https://api.ipify.org",
		"https://ifconfig.me/ip",
		"https://icanhazip.com",
	}
	var last error
	for _, ep := range endpoints {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, ep, nil)
		if err != nil {
			last = err
			continue
		}
		req.Header.Set("User-Agent", defaults.UserAgent)
		resp, err := client.Do(req)
		if err != nil {
			last = err
			continue
		}
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 128))
		_ = resp.Body.Close()
		if resp.StatusCode >= 300 {
			last = fmt.Errorf("%s: status %d", ep, resp.StatusCode)
			continue
		}
		ip := strings.TrimSpace(string(body))
		if parsed := net.ParseIP(ip); parsed != nil {
			return parsed.String(), nil
		}
		last = fmt.Errorf("%s: invalid ip %q", ep, ip)
	}
	if last == nil {
		last = fmt.Errorf("public ip lookup failed")
	}
	return "", last
}

// MaskIP hides roughly half of an IP for UI display.
// IPv4: 185.***.***.42  IPv6: 2001:db8:****:****:****:****:****:1
func MaskIP(ip string) string {
	ip = strings.TrimSpace(ip)
	if ip == "" {
		return "-"
	}
	if host, _, err := net.SplitHostPort(ip); err == nil {
		ip = host
	}
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return "-"
	}
	if v4 := parsed.To4(); v4 != nil {
		return fmt.Sprintf("%d.***.***.%d", v4[0], v4[3])
	}
	// IPv6: keep first 2 and last 1 hextets.
	full := parsed.String()
	parts := strings.Split(full, ":")
	if len(parts) < 3 {
		return full
	}
	for i := 2; i < len(parts)-1; i++ {
		if parts[i] != "" {
			parts[i] = "****"
		}
	}
	return strings.Join(parts, ":")
}
