package deeplink

import "testing"

func TestParseControl(t *testing.T) {
	cases := map[string]string{
		"myvpn://connect":       KindConnect,
		"myvpn://disconnect":    KindDisconnect,
		"myvpn://close":         KindDisconnect,
		"myvpn://toggle":        KindToggle,
		"myvpn://open":          KindOpen,
		"myvpn://status":        KindOpen,
		"myinternetvpn://open":  KindOpen,
	}
	for raw, want := range cases {
		act, err := Parse(raw)
		if err != nil {
			t.Fatalf("%s: %v", raw, err)
		}
		if act.Kind != want {
			t.Fatalf("%s: kind=%s want=%s", raw, act.Kind, want)
		}
	}
}

func TestParseImportAdd(t *testing.T) {
	act, err := Parse("myvpn://import/https://example.com/sub")
	if err != nil {
		t.Fatal(err)
	}
	if act.Kind != KindImport || act.Data != "https://example.com/sub" {
		t.Fatalf("%+v", act)
	}

	act, err = Parse("myvpn://add/vless://11111111-1111-1111-1111-111111111111@host:443?security=tls#DE")
	if err != nil {
		t.Fatal(err)
	}
	if act.Kind != KindAdd {
		t.Fatalf("kind=%s", act.Kind)
	}
	if !stringsHasPrefix(act.Data, "vless://") {
		t.Fatalf("data=%q", act.Data)
	}
}

func stringsHasPrefix(s, p string) bool {
	return len(s) >= len(p) && s[:len(p)] == p
}
