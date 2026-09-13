# Build MyInternetVPN v2 (Go + frontend assets)
$ErrorActionPreference = 'Stop'
$root = Resolve-Path (Join-Path $PSScriptRoot '..')
Set-Location $root

if (-not (Test-Path "resources\core\mihomo.exe")) {
  Write-Host "Downloading mihomo..."
  & "$root\scripts\download-mihomo.ps1"
}

Push-Location frontend
if (Test-Path package-lock.json) { npm ci } else { npm install }
npm run build
Pop-Location

go test ./tests/... -count=1
New-Item -ItemType Directory -Force -Path "build\bin" | Out-Null
# Wails requires desktop,production tags (plain `go build` shows a runtime error dialog).
go build -tags "desktop,production" -ldflags "-w -s -H windowsgui" -o "build\bin\MyInternetVPN.exe" .
Write-Host "Built build\bin\MyInternetVPN.exe"
Write-Host "For a release zip use: .\scripts\package.ps1"
Write-Host "Or use Wails CLI: wails build"
