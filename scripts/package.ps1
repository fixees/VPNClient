# Build a distributable Windows zip for MyInternetVPN (+ checksum + Ed25519 signature).
# Single-file exe: mihomo/geo/icons are embedded and extracted to AppData at runtime.
# Output:
#   dist/package/MyInternetVPN-windows-amd64-<version>.zip   (contains only the .exe)
#   dist/package/MyInternetVPN-windows-amd64-<version>.zip.sha256
#   dist/package/MyInternetVPN-windows-amd64-<version>.zip.sig
$ErrorActionPreference = 'Stop'

$root = Resolve-Path (Join-Path $PSScriptRoot '..')
Set-Location $root

$version = $env:APP_VERSION
if (-not $version) {
  $version = (Get-Date -Format 'yyyyMMdd-HHmmss')
}
if ($version.Length -gt 12 -and $version -match '^[0-9a-f]+$') {
  $version = $version.Substring(0, 12)
}

Write-Host "==> Frontend"
Push-Location frontend
if (Test-Path package-lock.json) { npm ci } else { npm install }
npm run build
Pop-Location

Write-Host "==> Tests"
go test ./tests/... -count=1

Write-Host "==> Download mihomo core (required for embed)"
$coreExe = Join-Path $root 'resources\core\mihomo.exe'
$geoIP = Join-Path $root 'resources\core\geoip.metadb'
$geoSite = Join-Path $root 'resources\core\geosite.dat'
if (-not ((Test-Path $coreExe) -and (Test-Path $geoIP) -and (Test-Path $geoSite))) {
  & "$root\scripts\download-mihomo.ps1"
} else {
  Write-Host "Using existing resources\core"
}

Write-Host "==> Build single-file app"
$outDir = Join-Path $root 'dist\package\staging'
if (Test-Path $outDir) { Remove-Item $outDir -Recurse -Force }
New-Item -ItemType Directory -Force -Path $outDir | Out-Null

$ldflags = "-w -s -H windowsgui -X main.version=$version"
go build -tags "desktop,production" -ldflags $ldflags -o (Join-Path $outDir 'MyInternetVPN.exe') .

$zipName = "MyInternetVPN-windows-amd64-$version.zip"
$pkgDir = Join-Path $root 'dist\package'
$zipPath = Join-Path $pkgDir $zipName
New-Item -ItemType Directory -Force -Path $pkgDir | Out-Null
if (Test-Path $zipPath) { Remove-Item $zipPath -Force }

# Zip contains only the executable — resources are embedded.
Compress-Archive -Path (Join-Path $outDir 'MyInternetVPN.exe') -DestinationPath $zipPath -Force

$hash = (Get-FileHash -Algorithm SHA256 -Path $zipPath).Hash.ToLowerInvariant()
$shaPath = "$zipPath.sha256"
Set-Content -Path $shaPath -Value "$hash  $zipName" -NoNewline
Write-Host "Checksum: $shaPath"

Write-Host "==> Sign package"
$privPath = if ($env:UPDATE_SIGNING_KEY_FILE) { $env:UPDATE_SIGNING_KEY_FILE } else { Join-Path $root 'secrets\update_ed25519_private.key' }
if ($env:UPDATE_SIGNING_KEY) {
  $privPath = Join-Path $env:TEMP "update_ed25519_private.key"
  Set-Content -Path $privPath -Value $env:UPDATE_SIGNING_KEY.Trim() -NoNewline
}
if (-not (Test-Path $privPath)) {
  throw "Signing key not found (set UPDATE_SIGNING_KEY / UPDATE_SIGNING_KEY_FILE or create secrets/update_ed25519_private.key)"
}
$sigPath = "$zipPath.sig"
go run ./scripts/sign-release $zipPath $privPath $sigPath

Write-Host "Package (single exe): $zipPath"
Write-Host "Signature: $sigPath"
