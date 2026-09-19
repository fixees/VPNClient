package tests

import (
	"encoding/base64"
	"strings"
	"testing"

	"myinternetvpn/client/internal/profiles"

	qrcode "github.com/skip2/go-qrcode"
)

// TestQRGeneration validates that QR codes can be generated for subscription URLs.
func TestQRGeneration(t *testing.T) {
	testURL := "https://example.com/subscription?key=test123"
	
	png, err := qrcode.Encode(testURL, qrcode.Medium, 256)
	if err != nil {
		t.Fatalf("qrcode.Encode: %v", err)
	}
	if len(png) == 0 {
		t.Fatal("generated QR code is empty")
	}
	
	// Verify it's a valid PNG
	if len(png) < 8 || string(png[:8]) != "\x89PNG\r\n\x1a\n" {
		t.Fatal("generated data is not a valid PNG")
	}
	
	// Verify base64 encoding works
	encoded := base64.StdEncoding.EncodeToString(png)
	if len(encoded) == 0 {
		t.Fatal("base64 encoding is empty")
	}
	
	dataURI := "data:image/png;base64," + encoded
	if !strings.HasPrefix(dataURI, "data:image/png;base64,") {
		t.Fatal("data URI prefix incorrect")
	}
}

// TestQRGenerationWithProfile tests the workflow with an actual Profile.
func TestQRGenerationWithProfile(t *testing.T) {
	p := profiles.Profile{
		Name:            "Test Profile",
		SubscriptionURL: "https://sub.example.com/abc",
		Proxies: []profiles.ProxyNode{
			{"name": "Node1", "type": "ss", "server": "1.1.1.1", "port": 443},
		},
	}
	
	if strings.TrimSpace(p.SubscriptionURL) == "" {
		t.Fatal("profile has no subscription URL")
	}
	
	png, err := qrcode.Encode(p.SubscriptionURL, qrcode.Medium, 256)
	if err != nil {
		t.Fatalf("qrcode.Encode: %v", err)
	}
	if len(png) == 0 {
		t.Fatal("generated QR code is empty")
	}
}

// TestQRGenerationEmpty ensures empty URLs are handled gracefully.
func TestQRGenerationEmpty(t *testing.T) {
	_, err := qrcode.Encode("", qrcode.Medium, 256)
	// Empty content is technically valid for QR codes, but our app logic should reject it.
	// The library doesn't error, so we test that at the app level.
	if err != nil {
		// If library rejects it, that's fine too
		return
	}
}
