package tests

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"myinternetvpn/client/internal/parse"
	"myinternetvpn/client/internal/profiles"
	"myinternetvpn/client/internal/subscription"
)

func TestParseUserInfoHeader(t *testing.T) {
	q, raw := subscription.ParseUserInfoHeader("upload=10; download=20; total=100; expire=2000000000")
	if raw == "" || q.Upload != 10 || q.Download != 20 || q.Total != 100 || q.ExpireUnix != 2000000000 {
		t.Fatalf("quota=%+v raw=%q", q, raw)
	}
	if q.Used() != 30 || q.Remaining() != 70 || q.Expired() {
		t.Fatalf("derived used/remaining/expired wrong: %+v", q)
	}
	past := profiles.Quota{ExpireUnix: 1}
	if !past.Expired() {
		t.Fatal("expected expired")
	}
}

func TestParseSubscriptionBodyBase64Links(t *testing.T) {
	ss := "ss://Y2hhY2hhMjAtaWV0Zi1wb2x5MTMwNTp0ZXN0cGFzcw@example.com:8388#A"
	vless := "vless://11111111-1111-1111-1111-111111111111@vl.example:443?security=tls&type=tcp#B"
	body := base64.StdEncoding.EncodeToString([]byte(ss + "\n" + vless))
	nodes, err := parse.ParseSubscriptionBody([]byte(body))
	if err != nil {
		t.Fatalf("ParseSubscriptionBody: %v", err)
	}
	if len(nodes) != 2 {
		t.Fatalf("nodes=%d", len(nodes))
	}
}

func TestParseSubscriptionBodyClashYAML(t *testing.T) {
	raw := []byte(`
proxies:
  - name: n1
    type: ss
    server: a.example
    port: 443
    cipher: aes-128-gcm
    password: x
`)
	nodes, err := parse.ParseSubscriptionBody(raw)
	if err != nil || len(nodes) != 1 {
		t.Fatalf("err=%v nodes=%v", err, nodes)
	}
}

func TestSubscriptionFetchAndDue(t *testing.T) {
	ss := "ss://Y2hhY2hhMjAtaWV0Zi1wb2x5MTMwNTp0ZXN0cGFzcw@example.com:8388#Node"
	payload := base64.StdEncoding.EncodeToString([]byte(ss))
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Subscription-Userinfo", "upload=1; download=2; total=10; expire=2500000000")
		w.Header().Set("Profile-Update-Interval", "12")
		w.Header().Set("Profile-Title", "My Plan")
		_, _ = w.Write([]byte(payload))
	}))
	t.Cleanup(srv.Close)

	f := subscription.NewFetcher()
	f.HTTPClient = srv.Client()
	res, err := f.Fetch(srv.URL)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if len(res.Proxies) != 1 || res.Quota.Total != 10 || res.UpdateIntervalHours != 12 || res.ProviderTitle != "My Plan" {
		b, _ := json.Marshal(res)
		t.Fatalf("unexpected result: %s", b)
	}

	p := profiles.Profile{
		Name:            "S",
		SubscriptionURL: srv.URL,
		LastSyncAt:      time.Now().UTC().Add(-2 * time.Hour).Format(time.RFC3339),
	}
	if !subscription.Due(p, 60, time.Now()) { // global 60 minutes, last sync 2h ago
		t.Fatal("expected due")
	}
	p.LastSyncAt = time.Now().UTC().Format(time.RFC3339)
	if subscription.Due(p, 360, time.Now()) {
		t.Fatal("expected not due")
	}
	p.UpdateIntervalHours = 1
	p.LastSyncAt = time.Now().UTC().Add(-2 * time.Hour).Format(time.RFC3339)
	if !subscription.Due(p, 360, time.Now()) {
		t.Fatal("expected due by profile interval")
	}
}
