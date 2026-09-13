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
