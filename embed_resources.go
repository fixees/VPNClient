package main

import (
	"embed"
)

// Bundled mihomo core + geo assets. Extracted to %AppData%/MyInternetVPN/core on startup.
// Build requires these files locally (scripts/download-mihomo.ps1).
//
//go:embed resources/core/mihomo.exe
//go:embed resources/core/mihomo.exe.sha256
//go:embed resources/core/geoip.metadb
//go:embed resources/core/geosite.dat
var bundledCore embed.FS
