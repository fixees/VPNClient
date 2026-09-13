# MyInternetVPN Client

Windows 10/11 desktop VPN client (Go + Wails + mihomo).

**Stack:** Go + [Wails](https://wails.io) · **Core:** [mihomo](https://github.com/MetaCubeX/mihomo) (Clash Meta)

> [Hiddify](https://github.com/hiddify/hiddify-app) uses sing-box. This client uses **mihomo**
> (Clash Verge model: spawn core → External Controller HTTP API).

## Branches

| Branch | Purpose |
|--------|---------|
| `main` | Stable releases. CI runs tests and **auto-builds** a Windows zip package on every push. |
| `dev` | Active development. CI runs tests (and compile check) on push/PR. No release package. |

Flow: develop on `dev` → PR into `main` → package published as artifact + rolling release `latest-main`.

## Layout

```text
app.go / main.go     Wails app + bindings
internal/            backend packages
tests/               black-box + integration tests
frontend/            Vite UI
resources/core/      mihomo + geo (embedded into exe at build)
scripts/             build / package helpers
.github/workflows/   CI for main & dev
```

## Prerequisites

- Go 1.22+
- Node.js 20+
- Wails CLI (optional): `go install github.com/wailsapp/wails/v2/cmd/wails@latest`
- WebView2 (included with Windows 10/11)

## Setup

```powershell
.\scripts\download-mihomo.ps1
.\scripts\build.ps1

# release zip (same as main CI):
.\scripts\package.ps1
```

> Important: do **not** run plain `go build`. Wails needs:
> `go build -tags "desktop,production" -ldflags "-w -s -H windowsgui"`
> (or just `.\scripts\build.ps1`). Release zip contains a **single** `.exe` — mihomo/geo are embedded.

Configure auto-update (defaults already point to [fixees/VPNClient](https://github.com/fixees/VPNClient)):

```json
{
  "updateOwner": "fixees",
  "updateRepo": "VPNClient",
  "updateTag": "latest-main"
}
```

Data: `%APPDATA%\MyInternetVPN\` (`profiles.json`, `settings.json`, `core/`, `updates/`).

## Backend capabilities

- Always runs elevated (manifest `requireAdministrator` + startup check)
- Single-instance guard (second launch activates existing window)
- System proxy when TUN is off
- Kill switch + DNS leak guard (Windows firewall rules)
- Auto-reconnect health monitor
- Live speed/totals from mihomo `/traffic` + `/connections`
- Core SHA-256 integrity sidecar
- GeoIP/GeoSite assets beside core
- Import `vless://` `vmess://` `ss://` / Clash YAML / JSON / **http(s) subscription URL**
- Subscription quota/expiry sync (`subscription-userinfo`) + scheduled node refresh
- Modes: rule / global / direct
- Node list + switch (mihomo PROXY group) with delay
- Single-instance lock, autostart, close-to-hide, Quit/Show
- File logger with secret redaction (`%APPDATA%\MyInternetVPN\client.log`)
- Apply downloaded update only after **SHA-256 + Ed25519** verification
- System tray icon (Show / Connect-Disconnect / Quit)
- Bulk URL-test for all nodes
- Graceful core stop (interrupt, then kill)
- Auto-update check/download from GitHub `latest-main`

### Signed releases

Packages on `main` must include:

- `*.zip`
- `*.zip.sha256`
- `*.zip.sig` (Ed25519 over zip bytes)

Public key: `resources/update/ed25519_public.key` (embedded in the app).  
Private key: CI secret `UPDATE_SIGNING_KEY` or local `secrets/update_ed25519_private.key` (gitignored).

```powershell
go run ./scripts/gen-update-keys   # once
.\scripts\package.ps1              # builds + checksum + signature
```


## Tests

All automated tests live under `tests/`:

```powershell
go test ./tests/... -count=1
```

## CI

Workflow: [`.github/workflows/ci.yml`](.github/workflows/ci.yml)

- **push/PR → `main` or `dev`:** run `go test ./tests/...`, build frontend, compile app
- **push → `main` only:** create `MyInternetVPN-windows-amd64-*.zip`, upload Actions artifact, update GitHub release tag `latest-main`

## Roadmap

- Auth / account integration and Telegram bot
- System tray, autostart, kill switch
- Import Clash / Clash Meta subscription URLs
