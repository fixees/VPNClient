package tests

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"myinternetvpn/client/internal/api"
	"myinternetvpn/client/internal/health"
	"myinternetvpn/client/internal/integrity"
	"myinternetvpn/client/internal/parse"
	"myinternetvpn/client/internal/update"
)

func TestIntegrityVerifyBeside(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "mihomo.exe")
	content := []byte("core-bytes")
	if err := os.WriteFile(bin, content, 0o755); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(content)
	hexSum := hex.EncodeToString(sum[:])
	if err := os.WriteFile(bin+".sha256", []byte(hexSum+"  mihomo.exe\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	checked, err := integrity.VerifyBeside(bin)
	if err != nil || !checked {
		t.Fatalf("checked=%v err=%v", checked, err)
	}
	_ = os.WriteFile(bin+".sha256", []byte("deadbeef"), 0o644)
	if _, err := integrity.VerifyBeside(bin); err == nil {
		t.Fatal("expected mismatch")
	}
}

func TestParseShareLinksAndClash(t *testing.T) {
	ss := "ss://Y2hhY2hhMjAtaWV0Zi1wb2x5MTMwNTp0ZXN0cGFzcw@example.com:8388#Tokyo"
	node, err := parse.ParseShareLink(ss)
	if err != nil {
		t.Fatalf("ss: %v", err)
	}
	if node["type"] != "ss" || node["server"] != "example.com" {
		t.Fatalf("ss node: %#v", node)
	}

	vmessJSON := `{"v":"2","ps":"VNode","add":"v.example","port":"443","id":"11111111-1111-1111-1111-111111111111","aid":"0","net":"ws","type":"none","host":"v.example","path":"/","tls":"tls"}`
	vmess := "vmess://" + base64.StdEncoding.EncodeToString([]byte(vmessJSON))
	vnode, err := parse.ParseShareLink(vmess)
	if err != nil {
		t.Fatalf("vmess: %v", err)
	}
	if vnode["type"] != "vmess" || vnode["server"] != "v.example" {
		t.Fatalf("vmess node: %#v", vnode)
	}

	vless := "vless://11111111-1111-1111-1111-111111111111@vl.example:443?encryption=none&security=tls&type=ws&host=vl.example&path=%2Fws#VL"
	vl, err := parse.ParseShareLink(vless)
	if err != nil {
		t.Fatalf("vless: %v", err)
	}
	if vl["type"] != "vless" || vl["server"] != "vl.example" {
		t.Fatalf("vless node: %#v", vl)
	}

	yamlDoc := `
proxies:
  - name: yaml-1
    type: ss
    server: y.example
    port: 443
    cipher: aes-128-gcm
    password: x
`
	nodes, err := parse.ParseClashYAML([]byte(yamlDoc))
	if err != nil || len(nodes) != 1 || nodes[0]["name"] != "yaml-1" {
		t.Fatalf("clash: %v %#v", err, nodes)
	}
}

func TestParseVLESSECH(t *testing.T) {
	raw := "vless://11111111-1111-1111-1111-111111111111@ech.example:443?security=tls&type=tcp&ech=1&ech-config=YmFzZTY0#ECH"
	node, err := parse.ParseShareLink(raw)
	if err != nil {
		t.Fatal(err)
	}
	opts, ok := node["ech-opts"].(map[string]any)
	if !ok || opts["enable"] != true || opts["config"] != "YmFzZTY0" {
		t.Fatalf("ech-opts=%#v", node["ech-opts"])
	}
}

func TestParseClashProxyProviders(t *testing.T) {
	providerBody := []byte(`
proxies:
  - name: from-provider
    type: ss
    server: p.example
    port: 443
    cipher: aes-128-gcm
    password: x
`)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(providerBody)
	}))
	t.Cleanup(srv.Close)

	doc := []byte(`
proxy-providers:
  remote:
    type: http
    url: "` + srv.URL + `"
    interval: 3600
`)
	nodes, err := parse.ParseClashYAMLWithFetch(doc, func(u string) ([]byte, error) {
		resp, err := http.Get(u)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		return io.ReadAll(resp.Body)
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes) != 1 || nodes[0]["name"] != "from-provider" {
		t.Fatalf("%#v", nodes)
	}

	_, err = parse.ParseClashYAML(doc)
	if err == nil {
		t.Fatal("expected error without fetch for providers-only yaml")
	}
}

func TestAPITrafficModeConnections(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/traffic", func(w http.ResponseWriter, r *http.Request) {
		flusher, _ := w.(http.Flusher)
		_, _ = w.Write([]byte(`{"up":10,"down":20}` + "\n"))
		if flusher != nil {
			flusher.Flush()
		}
	})
	mux.HandleFunc("/connections", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(api.ConnectionsInfo{UploadTotal: 100, DownloadTotal: 200})
	})
	mux.HandleFunc("/configs", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPatch {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method == http.MethodPut {
			if r.URL.RawQuery != "force=true" {
				t.Fatalf("expected force=true query, got %q", r.URL.RawQuery)
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"mode": "rule"})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	c := api.NewClient(srv.URL, "")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	snap, err := c.TrafficOnce(ctx)
	if err != nil || snap.Up != 10 || snap.Down != 20 {
		t.Fatalf("traffic: %#v err=%v", snap, err)
	}
	conn, err := c.Connections()
	if err != nil || conn.DownloadTotal != 200 {
		t.Fatalf("connections: %#v err=%v", conn, err)
	}
	if err := c.SetMode("global"); err != nil {
		t.Fatalf("SetMode: %v", err)
	}
	if err := c.ReloadConfig(`C:\tmp\config.yaml`); err != nil {
		t.Fatalf("ReloadConfig: %v", err)
	}
	mode, err := c.Mode()
	if err != nil || mode != "rule" {
		t.Fatalf("Mode: %s err=%v", mode, err)
	}
}

func TestHealthReconnect(t *testing.T) {
	healthy := false
	reconnects := 0
	c := health.New(time.Hour, func() bool { return healthy }, func() error {
		reconnects++
		healthy = true
		return nil
	})
	c.MaxFails = 2
	c.TickOnce()
	c.TickOnce()
	if reconnects != 1 {
		t.Fatalf("reconnects=%d", reconnects)
	}
}

func TestUpdateFindWindowsZip(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(update.Release{
			TagName: "latest-main",
			Assets: []update.ReleaseAsset{
				{Name: "MyInternetVPN-windows-amd64-abc.zip", BrowserDownloadURL: "http://example/x.zip", Size: 12},
			},
		})
	}))
	t.Cleanup(srv.Close)

	c := update.NewGitHub("o", "r", "latest-main")
	c.HTTPClient = srv.Client()
	// Point FetchRelease URL by temporarily using custom transport via replacing Fetch - call FindWindowsZip directly.
	rel := update.Release{
		TagName: "latest-main",
		Assets: []update.ReleaseAsset{
			{Name: "notes.txt"},
			{Name: "MyInternetVPN-windows-amd64-abc.zip", BrowserDownloadURL: "http://example/x.zip"},
		},
	}
	asset, err := c.FindWindowsZip(rel)
	if err != nil || asset.Name == "" {
		t.Fatalf("asset err=%v %#v", err, asset)
	}
}
