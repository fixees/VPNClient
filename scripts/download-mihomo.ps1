# Download latest mihomo + geo databases into resources/core/
$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $PSScriptRoot
$outDir = Join-Path $root 'resources\core'
New-Item -ItemType Directory -Force -Path $outDir | Out-Null

$headers = @{ 'User-Agent' = 'MyInternetVPN' }

$release = Invoke-RestMethod -Uri 'https://api.github.com/repos/MetaCubeX/mihomo/releases/latest' -Headers $headers
$asset = $release.assets | Where-Object { $_.name -match '^mihomo-windows-amd64-compatible-.*\.zip$' } | Select-Object -First 1
if (-not $asset) {
  throw 'Compatible Windows amd64 asset not found in latest release'
}

$zip = Join-Path $env:TEMP $asset.name
Write-Host "Downloading $($asset.name)..."
Invoke-WebRequest -Uri $asset.browser_download_url -OutFile $zip -UseBasicParsing -Headers $headers

$extract = Join-Path $env:TEMP ("mihomo-extract-" + [guid]::NewGuid().ToString())
New-Item -ItemType Directory -Force -Path $extract | Out-Null
Expand-Archive -Path $zip -DestinationPath $extract -Force

$exe = Get-ChildItem -Path $extract -Recurse -Filter '*.exe' | Select-Object -First 1
if (-not $exe) { throw 'No exe found in archive' }

$exePath = Join-Path $outDir 'mihomo.exe'
Copy-Item $exe.FullName $exePath -Force

# Sidecar integrity hash
$hash = (Get-FileHash -Algorithm SHA256 -Path $exePath).Hash.ToLowerInvariant()
Set-Content -Path ($exePath + '.sha256') -Value "$hash  mihomo.exe" -NoNewline
Write-Host "Installed: $exePath"
Write-Host "SHA256: $hash"
Write-Host "Version tag: $($release.tag_name)"

# Geo databases used by mihomo rules
$geoFiles = @(
  @{ Name = 'geoip.metadb'; Url = 'https://github.com/MetaCubeX/meta-rules-dat/releases/download/latest/geoip.metadb' },
  @{ Name = 'geosite.dat'; Url = 'https://github.com/MetaCubeX/meta-rules-dat/releases/download/latest/geosite.dat' }
)
foreach ($g in $geoFiles) {
  $dest = Join-Path $outDir $g.Name
  Write-Host "Downloading $($g.Name)..."
  Invoke-WebRequest -Uri $g.Url -OutFile $dest -UseBasicParsing -Headers $headers
}
