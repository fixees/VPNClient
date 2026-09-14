package subscription

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"myinternetvpn/client/internal/defaults"
	"myinternetvpn/client/internal/parse"
	"myinternetvpn/client/internal/profiles"
)

// Result is a fetched subscription payload plus quota metadata.
type Result struct {
	Proxies               []profiles.ProxyNode
	Quota                 profiles.Quota
	UpdateIntervalHours   int
	ProviderTitle         string
	SubscriptionUserInfo  string
}

// Fetcher downloads remote subscription documents.
type Fetcher struct {
	HTTPClient *http.Client
	UserAgent  string
}

func NewFetcher() *Fetcher {
	return &Fetcher{
		HTTPClient: &http.Client{Timeout: 45 * time.Second},
		UserAgent:  defaults.UserAgent,
	}
}

// Fetch downloads and parses an http(s) subscription URL.
func (f *Fetcher) Fetch(rawURL string) (Result, error) {
	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return Result{}, err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return Result{}, fmt.Errorf("subscription URL must be http(s), got %q", u.Scheme)
	}

	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return Result{}, err
	}
	req.Header.Set("User-Agent", f.UserAgent)
	req.Header.Set("Accept", "*/*")

	resp, err := f.HTTPClient.Do(req)
	if err != nil {
		return Result{}, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return Result{}, err
	}
	if resp.StatusCode >= 300 {
		return Result{}, fmt.Errorf("subscription fetch status %d", resp.StatusCode)
	}

	nodes, err := parse.ParseSubscriptionBodyWithFetch(body, f.fetchProviderURL)
	if err != nil {
		return Result{}, err
	}

	out := Result{Proxies: nodes}
	out.Quota, out.SubscriptionUserInfo = ParseUserInfoHeader(headerGet(resp.Header, "Subscription-Userinfo"))
	if out.SubscriptionUserInfo == "" {
		// Some panels use lowercase / alternate spellings.
		out.Quota, out.SubscriptionUserInfo = ParseUserInfoHeader(headerGet(resp.Header, "subscription-userinfo"))
	}
	if v := headerGet(resp.Header, "Profile-Update-Interval"); v != "" {
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
			out.UpdateIntervalHours = n
		}
	}
	if v := headerGet(resp.Header, "Content-Disposition"); v != "" {
		out.ProviderTitle = filenameFromDisposition(v)
	}
	if v := headerGet(resp.Header, "Profile-Title"); v != "" {
		out.ProviderTitle = strings.Trim(v, `"' `)
	}
	return out, nil
}

func (f *Fetcher) fetchProviderURL(rawURL string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	ua := f.UserAgent
	if ua == "" {
		ua = defaults.UserAgent
	}
	req.Header.Set("User-Agent", ua)
	req.Header.Set("Accept", "*/*")
	client := f.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("provider fetch status %d", resp.StatusCode)
	}
	return body, nil
}

// ParseUserInfoHeader parses Clash-style subscription-userinfo values.
// Example: upload=1; download=2; total=3; expire=1710000000
func ParseUserInfoHeader(raw string) (profiles.Quota, string) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return profiles.Quota{}, ""
	}
	q := profiles.Quota{}
	for _, part := range strings.Split(raw, ";") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		key, val, ok := strings.Cut(part, "=")
		if !ok {
			continue
		}
		key = strings.ToLower(strings.TrimSpace(key))
		val = strings.TrimSpace(val)
		n, _ := strconv.ParseInt(val, 10, 64)
		switch key {
		case "upload":
			q.Upload = n
		case "download":
			q.Download = n
		case "total":
			q.Total = n
		case "expire":
			q.ExpireUnix = NormalizeExpireUnix(n)
		}
	}
	return q, raw
}

// NormalizeExpireUnix converts provider expire values to Unix seconds.
// Some panels send milliseconds (or rarely microseconds); Hiddify-class #2330.
func NormalizeExpireUnix(n int64) int64 {
	if n <= 0 {
		return 0
	}
	// Seconds for year ~2001..2286 fit under 1e10; ms for 2001+ are >= 1e12.
	const msThreshold = int64(1_000_000_000_000)     // 1e12
	const usThreshold = int64(1_000_000_000_000_000) // 1e15
	switch {
	case n >= usThreshold:
		return n / 1_000_000
	case n >= msThreshold:
		return n / 1_000
	default:
		return n
	}
}

func headerGet(h http.Header, key string) string {
	if v := h.Get(key); v != "" {
		return v
	}
	for k, vals := range h {
		if strings.EqualFold(k, key) && len(vals) > 0 {
			return vals[0]
		}
	}
	return ""
}

func filenameFromDisposition(v string) string {
	// attachment; filename="My Plan"
	lower := strings.ToLower(v)
	idx := strings.Index(lower, "filename=")
	if idx < 0 {
		return ""
	}
	name := strings.TrimSpace(v[idx+len("filename="):])
	return strings.Trim(name, `"'`)
}
