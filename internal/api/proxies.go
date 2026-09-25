package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"myinternetvpn/client/internal/defaults"
)

// ProxyGroup is a selector/url-test/etc group from GET /proxies.
type ProxyGroup struct {
	Name    string   `json:"name"`
	Type    string   `json:"type"`
	Now     string   `json:"now"`
	All     []string `json:"all"`
	History []DelaySample `json:"history,omitempty"`
}

// DelaySample is a latency history point.
type DelaySample struct {
	Time  string `json:"time"`
	Delay int    `json:"delay"`
}

// ProxyNodeRuntime is a leaf proxy entry (or selectable nested group like AUTO).
type ProxyNodeRuntime struct {
	Name    string        `json:"name"`
	Type    string        `json:"type"`
	UDP     bool          `json:"udp"`
	History []DelaySample `json:"history,omitempty"`
	Delay   int           `json:"delay"`
	Now     string        `json:"now,omitempty"` // for AUTO: currently chosen leaf
}

type proxiesResponse struct {
	Proxies map[string]json.RawMessage `json:"proxies"`
}

// Proxies returns raw proxy map from the controller.
func (c *Client) Proxies() (map[string]json.RawMessage, error) {
	var resp proxiesResponse
	if err := c.getJSON("/proxies", &resp); err != nil {
		return nil, err
	}
	return resp.Proxies, nil
}

// Group returns a named proxy group (e.g. PROXY).
func (c *Client) Group(name string) (ProxyGroup, error) {
	var g ProxyGroup
	path := "/proxies/" + url.PathEscape(name)
	if err := c.getJSON(path, &g); err != nil {
		return ProxyGroup{}, err
	}
	g.Name = name
	return g, nil
}

// SelectProxy switches the current node inside a group.
func (c *Client) SelectProxy(group, name string) error {
	payload, _ := json.Marshal(map[string]string{"name": name})
	path := "/proxies/" + url.PathEscape(group)
	return c.doJSON(http.MethodPut, path, payload, nil)
}

// CloseConnections drops active connections (useful after node switch).
func (c *Client) CloseConnections() error {
	return c.doJSON(http.MethodDelete, "/connections", nil, nil)
}

// CloseConnection drops a single active connection by ID.
func (c *Client) CloseConnection(id string) error {
	if id == "" {
		return fmt.Errorf("connection ID is required")
	}
	path := "/connections/" + url.PathEscape(id)
	return c.doJSON(http.MethodDelete, path, nil, nil)
}

// TestDelay triggers URL-test delay for a proxy name.
// timeoutMs is clamped to DelayAPITimeoutMaxMS — mihomo parses ?timeout= as int16
// and returns HTTP 400 {"message":"Body invalid"} when the value overflows.
func (c *Client) TestDelay(name, testURL string, timeoutMs int) (int, error) {
	if testURL == "" {
		testURL = defaults.URLTestURL
	}
	timeoutMs = clampDelayTimeout(timeoutMs)
	q := url.Values{}
	q.Set("url", testURL)
	q.Set("timeout", strconv.Itoa(timeoutMs))
	if expected := delayExpectedStatus(testURL); expected != "" {
		q.Set("expected", expected)
	}
	path := "/proxies/" + url.PathEscape(name) + "/delay?" + q.Encode()
	var out struct {
		Delay int `json:"delay"`
	}
	if err := c.getJSONWithTimeout(path, time.Duration(timeoutMs)*time.Millisecond+3*time.Second, &out); err != nil {
		return 0, err
	}
	return out.Delay, nil
}

// TestGroupDelay runs delay tests for every member of a strategy group (AUTO/url-test).
func (c *Client) TestGroupDelay(name, testURL string, timeoutMs int) (map[string]int, error) {
	if testURL == "" {
		testURL = defaults.URLTestURL
	}
	timeoutMs = clampDelayTimeout(timeoutMs)
	q := url.Values{}
	q.Set("url", testURL)
	q.Set("timeout", strconv.Itoa(timeoutMs))
	if expected := delayExpectedStatus(testURL); expected != "" {
		q.Set("expected", expected)
	}
	path := "/group/" + url.PathEscape(name) + "/delay?" + q.Encode()
	var out map[string]int
	if err := c.getJSONWithTimeout(path, time.Duration(timeoutMs)*time.Millisecond+5*time.Second, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func clampDelayTimeout(timeoutMs int) int {
	if timeoutMs <= 0 {
		timeoutMs = defaults.URLTestTimeoutMS
	}
	if timeoutMs > defaults.DelayAPITimeoutMaxMS {
		timeoutMs = defaults.DelayAPITimeoutMaxMS
	}
	return timeoutMs
}

func delayExpectedStatus(testURL string) string {
	u := strings.ToLower(testURL)
	if strings.Contains(u, "generate_204") || strings.Contains(u, "hotspot-detect") {
		return "204"
	}
	if strings.Contains(u, "cdn-cgi/trace") {
		return "200"
	}
	return ""
}

// ListSelectableNodes returns PROXY group members with best-known delay.
func (c *Client) ListSelectableNodes(group string) (ProxyGroup, []ProxyNodeRuntime, error) {
	if group == "" {
		group = defaults.ProxyGroup
	}
	g, err := c.Group(group)
	if err != nil {
		return ProxyGroup{}, nil, err
	}
	raw, err := c.Proxies()
	if err != nil {
		return g, nil, err
	}
	out := make([]ProxyNodeRuntime, 0, len(g.All))
	for _, name := range g.All {
		if name == "DIRECT" || name == "REJECT" || name == defaults.WARPProxyName {
			continue
		}
		node := ProxyNodeRuntime{Name: name}
		if rawNode, ok := raw[name]; ok {
			var meta struct {
				Type    string        `json:"type"`
				UDP     bool          `json:"udp"`
				Now     string        `json:"now"`
				History []DelaySample `json:"history"`
			}
			_ = json.Unmarshal(rawNode, &meta)
			node.Type = meta.Type
			node.UDP = meta.UDP
			node.Now = meta.Now
			node.History = meta.History
			if len(meta.History) > 0 {
				node.Delay = meta.History[len(meta.History)-1].Delay
			}
		}
		// Skip nested groups from leaf list except AUTO which users may select.
		if node.Type == "Selector" || node.Type == "URLTest" || node.Type == "Fallback" || node.Type == "LoadBalance" {
			if name != defaults.AutoGroup {
				continue
			}
		}
		out = append(out, node)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Name == defaults.AutoGroup {
			return true
		}
		if out[j].Name == defaults.AutoGroup {
			return false
		}
		return out[i].Name < out[j].Name
	})
	return g, out, nil
}
