<div align="center">

<img src="winres/icon128.png" alt="Мой VPN" width="128" height="128" />

# Мой VPN Client

Desktop VPN client for Windows 10/11

[![CI](https://github.com/fixees/VPNClient/actions/workflows/ci.yml/badge.svg)](https://github.com/fixees/VPNClient/actions/workflows/ci.yml)
[![Platform](https://img.shields.io/badge/platform-Windows%2010%2F11-0078D4?logo=windows&logoColor=white)](https://myinternetvpn.com/)
[![Go](https://img.shields.io/badge/Go-1.22%2B-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![Wails](https://img.shields.io/badge/UI-Wails-red)](https://wails.io/)
[![Core](https://img.shields.io/badge/core-mihomo-2f8cff)](https://github.com/MetaCubeX/mihomo)
[![Release](https://img.shields.io/github/v/release/fixees/VPNClient?include_prereleases&label=release)](https://github.com/fixees/VPNClient/releases/tag/latest-main)

[Website](https://myinternetvpn.com/) · [Releases](https://github.com/fixees/VPNClient/releases) · [Issues](https://github.com/fixees/VPNClient/issues)

</div>

---

The application provides a native Windows UI, subscription and profile management, per-app routing, and automatic updates. The networking core is [mihomo](https://github.com/MetaCubeX/mihomo) (Clash Meta), started as a local process and controlled through its HTTP API.

## Requirements

| Component | Notes |
|-----------|--------|
| OS | Windows 10 / 11 (x64) |
| Privileges | Administrator (TUN, firewall, system routes) |
| Runtime | Microsoft WebView2 (included with current Windows) |

## Features

- Single executable distribution (core and geo databases embedded)
- Subscription import (HTTP/HTTPS), share links, and Clash-compatible configs
- QR code sharing for subscription URLs (transfer to other devices)
- Copy subscription URLs to clipboard for easy sharing
- Rule / global / direct modes
- TUN mode and optional system proxy
- Per-application split routing
- Optional WARP protection layer
- Kill switch and DNS leak protection
- Auto-reconnect, tray integration, single-instance launch
- Signed in-app updates from GitHub Releases

## Repository layout

```text
app.go, main.go, tray.go   Application entry and UI bindings
internal/                  Backend packages
frontend/                  Vite frontend
resources/core/            Bundled mihomo and geo assets
scripts/                   Build and packaging helpers
tests/                     Automated tests
.github/workflows/         CI
```

## Branches

| Branch | Role |
|--------|------|
| `main` | Stable line. Tests, build, and Windows release package on every push. |
| `dev` | Development line. Tests and compile checks on push and pull requests. |

Recommended flow: implement on `dev`, merge to `main` for release packaging.

## Build

Prerequisites: Go 1.22+, Node.js 20+.

```powershell
.\scripts\download-mihomo.ps1
.\scripts\build.ps1
.\scripts\package.ps1
```

Production builds must use Wails tags:

```text
go build -tags "desktop,production" -ldflags "-w -s -H windowsgui"
```

Do not use a plain `go build` for the shipping binary.

## Configuration and data

User data is stored under:

```text
%APPDATA%\MyInternetVPN\
```

Typical contents: `profiles.json`, `settings.json`, `core\`, `updates\`, `client.log`.

Default auto-update source:

```json
{
  "updateOwner": "fixees",
  "updateRepo": "VPNClient",
  "updateTag": "latest-main"
}
```

## Release signing

Packages published from `main` include:

- `*.zip`
- `*.zip.sha256`
- `*.zip.sig` (Ed25519 signature of the zip payload)

Public key: `resources/update/ed25519_public.key` (embedded in the client).  
Private key: CI secret `UPDATE_SIGNING_KEY`, or local `secrets/update_ed25519_private.key` (gitignored).

```powershell
go run ./scripts/gen-update-keys
.\scripts\package.ps1
```

## Tests

```powershell
go test ./tests/... -count=1
```

## Continuous integration

Workflow: [`.github/workflows/ci.yml`](.github/workflows/ci.yml)

- Push or pull request to `main` / `dev`: run tests, build frontend, compile the application
- Push to `main`: produce a rolling GitHub release `latest-main` with stable asset names:
  - `MyInternetVPN-windows-amd64.zip`
  - `MyInternetVPN-windows-amd64.zip.sha256`
  - `MyInternetVPN-windows-amd64.zip.sig`
  - `MyInternetVPN-windows-amd64.zip.version`

Each push to `main` **replaces** those assets (older hash-named builds are pruned). The commit SHA is recorded in the release notes and embedded in the binary.

## License and product

Client branding: **Мой VPN** / **Мой VPN Client**.  
Repository: [fixees/VPNClient](https://github.com/fixees/VPNClient).
