package tests

import (
	"archive/zip"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"myinternetvpn/client/internal/api"
	"myinternetvpn/client/internal/applog"
	"myinternetvpn/client/internal/update"
)

func TestAPIListAndSelectProxy(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/proxies", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"proxies": map[string]any{
				"PROXY":  map[string]any{"type": "Selector", "now": "node-a", "all": []string{"AUTO", "node-a", "node-b"}},
				"AUTO":   map[string]any{"type": "URLTest", "now": "node-a", "all": []string{"node-a", "node-b"}},
				"node-a": map[string]any{"type": "Shadowsocks", "history": []map[string]any{{"delay": 42}}},
				"node-b": map[string]any{"type": "Vmess", "history": []map[string]any{{"delay": 80}}},
			},
		})
	})
	mux.HandleFunc("/proxies/PROXY", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"type": "Selector", "now": "node-a", "all": []string{"AUTO", "node-a", "node-b"}})
	})
	mux.HandleFunc("/connections", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		_ = json.NewEncoder(w).Encode(api.ConnectionsInfo{})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	c := api.NewClient(srv.URL, "")
	g, nodes, err := c.ListSelectableNodes("PROXY")
	if err != nil {
		t.Fatalf("ListSelectableNodes: %v", err)
	}
	if g.Now != "node-a" || len(nodes) < 2 {
		t.Fatalf("group=%+v nodes=%+v", g, nodes)
	}
	if err := c.SelectProxy("PROXY", "node-b"); err != nil {
		t.Fatalf("SelectProxy: %v", err)
	}
	if err := c.CloseConnections(); err != nil {
		t.Fatalf("CloseConnections: %v", err)
	}
}

func TestLogRedact(t *testing.T) {
	in := `connect password=supersecret uuid=1111 vless://abc@host:443 token=zzz`
	out := applog.Redact(in)
	if strings.Contains(out, "supersecret") || strings.Contains(out, "vless://abc@") || strings.Contains(out, "token=zzz") {
		t.Fatalf("not redacted: %s", out)
	}
	if !strings.Contains(out, "[redacted-link]") {
		t.Fatalf("expected redacted link marker: %s", out)
	}
}

func TestExtractZip(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "pkg.zip")
	zf, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(zf)
	w, err := zw.Create("MyInternetVPN.exe")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = w.Write([]byte("fake-exe"))
	_ = zw.Close()
	_ = zf.Close()

	dest := filepath.Join(dir, "out")
	if err := update.ExtractZip(zipPath, dest); err != nil {
		t.Fatalf("ExtractZip: %v", err)
	}
	raw, err := os.ReadFile(filepath.Join(dest, "MyInternetVPN.exe"))
	if err != nil || string(raw) != "fake-exe" {
		t.Fatalf("extracted file mismatch: %v %q", err, raw)
	}
}
