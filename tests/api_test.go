package tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"myinternetvpn/client/internal/api"
)

func TestAPIClientVersionAndAuth(t *testing.T) {
	secret := "s3cret"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/version" {
			http.NotFound(w, r)
			return
		}
		if got := r.Header.Get("Authorization"); got != "Bearer "+secret {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		_ = json.NewEncoder(w).Encode(api.VersionInfo{Meta: true, Version: "v1.19.30"})
	}))
	t.Cleanup(srv.Close)

	client := api.NewClient(srv.URL, secret)
	info, err := client.Version()
	if err != nil {
		t.Fatalf("Version: %v", err)
	}
	if !info.Meta || info.Version != "v1.19.30" {
		t.Fatalf("unexpected version: %+v", info)
	}
	if !client.Healthy() {
		t.Fatal("expected healthy")
	}
}

func TestAPIWaitHealthyTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "down", http.StatusServiceUnavailable)
	}))
	t.Cleanup(srv.Close)

	client := api.NewClient(srv.URL, "")
	if err := client.WaitHealthy(300 * time.Millisecond); err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestAPINewClientAddsScheme(t *testing.T) {
	c := api.NewClient("127.0.0.1:9090", "x")
	if c.BaseURL() != "http://127.0.0.1:9090" {
		t.Fatalf("BaseURL = %s", c.BaseURL())
	}
}

func TestAPIConnectionsParsing(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/connections" {
			http.NotFound(w, r)
			return
		}
		sample := map[string]interface{}{
			"downloadTotal": 1024000,
			"uploadTotal":   512000,
			"connections": []map[string]interface{}{
				{
					"id": "conn-123",
					"metadata": map[string]interface{}{
						"host":           "example.com",
						"network":        "tcp",
						"type":           "HTTP",
						"sourceIP":       "192.168.1.100",
						"destinationIP":  "93.184.216.34",
						"destinationPort": "443",
					},
					"upload":      1024,
					"download":    2048,
					"start":       "2024-01-01T12:00:00Z",
					"chains":      []string{"PROXY", "Server1"},
					"rule":        "DOMAIN",
					"rulePayload": "example.com",
				},
			},
		}
		_ = json.NewEncoder(w).Encode(sample)
	}))
	t.Cleanup(srv.Close)

	client := api.NewClient(srv.URL, "")
	conn, err := client.Connections()
	if err != nil {
		t.Fatalf("Connections: %v", err)
	}
	if conn.DownloadTotal != 1024000 {
		t.Errorf("DownloadTotal = %d, want 1024000", conn.DownloadTotal)
	}
	if conn.UploadTotal != 512000 {
		t.Errorf("UploadTotal = %d, want 512000", conn.UploadTotal)
	}
	if len(conn.Connections) != 1 {
		t.Fatalf("len(Connections) = %d, want 1", len(conn.Connections))
	}
	c := conn.Connections[0]
	if c.ID != "conn-123" {
		t.Errorf("ID = %s, want conn-123", c.ID)
	}
	if c.Upload != 1024 {
		t.Errorf("Upload = %d, want 1024", c.Upload)
	}
	if c.Download != 2048 {
		t.Errorf("Download = %d, want 2048", c.Download)
	}
	if c.Rule != "DOMAIN" {
		t.Errorf("Rule = %s, want DOMAIN", c.Rule)
	}
	if len(c.Chains) != 2 || c.Chains[0] != "PROXY" {
		t.Errorf("Chains = %v, want [PROXY Server1]", c.Chains)
	}
	if meta := c.Metadata; meta != nil {
		if host, ok := meta["host"].(string); !ok || host != "example.com" {
			t.Errorf("metadata.host = %v, want example.com", meta["host"])
		}
	}
}

func TestAPICloseConnection(t *testing.T) {
	deleted := ""
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if r.URL.Path == "/connections" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if len(r.URL.Path) > len("/connections/") && r.URL.Path[:len("/connections/")] == "/connections/" {
			deleted = r.URL.Path[len("/connections/"):]
			w.WriteHeader(http.StatusNoContent)
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(srv.Close)

	client := api.NewClient(srv.URL, "")
	if err := client.CloseConnection("test-id-123"); err != nil {
		t.Fatalf("CloseConnection: %v", err)
	}
	if deleted != "test-id-123" {
		t.Errorf("deleted ID = %s, want test-id-123", deleted)
	}

	if err := client.CloseConnections(); err != nil {
		t.Fatalf("CloseConnections: %v", err)
	}
}
