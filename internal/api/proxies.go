package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
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

// ProxyNodeRuntime is a leaf proxy entry.
type ProxyNodeRuntime struct {
	Name    string        `json:"name"`
	Type    string        `json:"type"`
	UDP     bool          `json:"udp"`
	History []DelaySample `json:"history,omitempty"`
	Delay   int           `json:"delay"`
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

// TestDelay triggers URL-test delay for a proxy name.
func (c *Client) TestDelay(name, testURL string, timeoutMs int) (int, error) {
	if testURL == "" {
		testURL = "https://www.gstatic.com/generate_204"
	}
	if timeoutMs <= 0 {
		timeoutMs = 5000
	}
	path := fmt.Sprintf("/proxies/%s/delay?url=%s&timeout=%d",
		url.PathEscape(name), url.QueryEscape(testURL), timeoutMs)
	var out struct {
		Delay int `json:"delay"`
	}
	if err := c.getJSON(path, &out); err != nil {
		return 0, err
	}
	return out.Delay, nil
}

// ListSelectableNodes returns PROXY group members with best-known delay.
func (c *Client) ListSelectableNodes(group string) (ProxyGroup, []ProxyNodeRuntime, error) {
	if group == "" {
		group = "PROXY"
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
		if name == "DIRECT" || name == "REJECT" {
			continue
		}
		node := ProxyNodeRuntime{Name: name}
		if rawNode, ok := raw[name]; ok {
			var meta struct {
				Type    string        `json:"type"`
				UDP     bool          `json:"udp"`
				History []DelaySample `json:"history"`
			}
			_ = json.Unmarshal(rawNode, &meta)
			node.Type = meta.Type
			node.UDP = meta.UDP
			node.History = meta.History
			if len(meta.History) > 0 {
				node.Delay = meta.History[len(meta.History)-1].Delay
			}
		}
		// Skip nested groups from leaf list except AUTO which users may select.
		if node.Type == "Selector" || node.Type == "URLTest" || node.Type == "Fallback" || node.Type == "LoadBalance" {
			if name != "AUTO" {
				continue
			}
		}
		out = append(out, node)
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	return g, out, nil
}
