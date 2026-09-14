package warp

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"golang.org/x/crypto/curve25519"

	"myinternetvpn/client/internal/defaults"
)

// Account holds WireGuard credentials for Cloudflare WARP.
type Account struct {
	PrivateKey   string
	PublicKey    string
	LocalAddress string // e.g. 172.16.0.2/32
	IPv6Address  string
	ClientID     string
	AccountType  string
}

type regRequest struct {
	Key          string `json:"key"`
	InstallID    string `json:"install_id"`
	FCMToken     string `json:"fcm_token"`
	TOS          string `json:"tos"`
	Model        string `json:"model"`
	SerialNumber string `json:"serial_number"`
	Locale       string `json:"locale"`
}

type regResponse struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Token   string `json:"token"`
	Account struct {
		AccountType string `json:"account_type"`
	} `json:"account"`
	Config struct {
		ClientID  string `json:"client_id"`
		Interface struct {
			Addresses struct {
				V4 string `json:"v4"`
				V6 string `json:"v6"`
			} `json:"addresses"`
		} `json:"interface"`
		Peers []struct {
			PublicKey string `json:"public_key"`
			Endpoint  struct {
				Host string `json:"host"`
				V4   string `json:"v4"`
			} `json:"endpoint"`
		} `json:"peers"`
	} `json:"config"`
}

// GenerateKeyPair returns base64 WireGuard private/public keys.
func GenerateKeyPair() (privateKey, publicKey string, err error) {
	var priv [32]byte
	if _, err = rand.Read(priv[:]); err != nil {
		return "", "", err
	}
	// Curve25519 clamping
	priv[0] &= 248
	priv[31] &= 127
	priv[31] |= 64
	var pub [32]byte
	curve25519.ScalarBaseMult(&pub, &priv)
	return base64.StdEncoding.EncodeToString(priv[:]), base64.StdEncoding.EncodeToString(pub[:]), nil
}

// Register creates a free WARP account via Cloudflare API.
func Register(licenseKey string) (Account, error) {
	priv, pub, err := GenerateKeyPair()
	if err != nil {
		return Account{}, err
	}
	body, _ := json.Marshal(regRequest{
		Key:          pub,
		InstallID:    "",
		FCMToken:     "",
		TOS:          time.Now().UTC().Format(time.RFC3339),
		Model:        defaults.ProductName,
		SerialNumber: "",
		Locale:       "en_US",
	})
	req, err := http.NewRequest(http.MethodPost, "https://api.cloudflareclient.com/v0a2158/reg", bytes.NewReader(body))
	if err != nil {
		return Account{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", defaults.UserAgent)

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return Account{}, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode >= 300 {
		return Account{}, fmt.Errorf("cloudflare reg: status %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	var out regResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return Account{}, err
	}
	v4 := strings.TrimSpace(out.Config.Interface.Addresses.V4)
	if v4 == "" {
		v4 = defaults.WARPLocalAddress
	}
	if !strings.Contains(v4, "/") {
		v4 += "/32"
	}
	acc := Account{
		PrivateKey:   priv,
		PublicKey:    pub,
		LocalAddress: v4,
		IPv6Address:  strings.TrimSpace(out.Config.Interface.Addresses.V6),
		ClientID:     out.Config.ClientID,
		AccountType:  out.Account.AccountType,
	}
	if license := strings.TrimSpace(licenseKey); license != "" && out.ID != "" && out.Token != "" {
		if err := applyLicense(client, out.ID, out.Token, license); err != nil {
			// Keys still work for free WARP; surface soft failure via AccountType note.
			acc.AccountType = strings.TrimSpace(acc.AccountType + " (license: " + err.Error() + ")")
		} else {
			acc.AccountType = "license"
		}
	}
	return acc, nil
}

func applyLicense(client *http.Client, id, token, license string) error {
	payload, _ := json.Marshal(map[string]string{"license": license})
	req, err := http.NewRequest(http.MethodPut, "https://api.cloudflareclient.com/v0a2158/reg/"+id+"/account", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("User-Agent", defaults.UserAgent)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("status %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	return nil
}
