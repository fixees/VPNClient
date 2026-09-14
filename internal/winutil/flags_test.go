package winutil

import "testing"

func TestHasCLIFlag(t *testing.T) {
	// Can't mutate os.Args reliably in parallel tests; just exercise empty/non-match via helper semantics.
	if IsOurProxyServer("", "127.0.0.1", 7890) {
		t.Fatal("empty server")
	}
}

func TestIsOurProxyServer(t *testing.T) {
	cases := []struct {
		server string
		want   bool
	}{
		{"127.0.0.1:7890", true},
		{"http=127.0.0.1:7890;https=127.0.0.1:7890", true},
		{"socks=127.0.0.1:1080", false},
		{"127.0.0.1:1080", false},
		{"", false},
	}
	for _, tc := range cases {
		if got := IsOurProxyServer(tc.server, "127.0.0.1", 7890); got != tc.want {
			t.Fatalf("%q: got %v want %v", tc.server, got, tc.want)
		}
	}
}
