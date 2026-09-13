# Download mihomo + geo databases into resources/core/
# Prefer GITHUB_TOKEN/GH_TOKEN (CI) to avoid API rate limits.
# Fallback: resolve latest tag via HTML redirect, then direct asset URL.
$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $PSScriptRoot
$outDir = Join-Path $root 'resources\core'
New-Item -ItemType Directory -Force -Path $outDir | Out-Null

$ua = 'MyInternetVPN-CI'
$apiHeaders = @{
  'User-Agent' = $ua
  'Accept'     = 'application/vnd.github+json'
  'X-GitHub-Api-Version' = '2022-11-28'
}
$token = $env:GITHUB_TOKEN
if (-not $token) { $token = $env:GH_TOKEN }
if ($token) {
  $apiHeaders['Authorization'] = "Bearer $token"
  Write-Host "Using GitHub token for API requests"
}

function Resolve-LatestTagFromRedirect {
  $url = 'https://github.com/MetaCubeX/mihomo/releases/latest'
  # Ask for the first hop Location without following the full redirect chain.
  try {
    $null = Invoke-WebRequest -Uri $url -Method Head -MaximumRedirection 0 -Headers @{ 'User-Agent' = $ua } -UseBasicParsing
  } catch {
    $resp = $_.Exception.Response
    if ($resp -and $resp.Headers['Location']) {
      $loc = [string]$resp.Headers['Location']
      if ($loc -match '/tag/([^/?#]+)') { return $Matches[1] }
    }
  }

  # PowerShell 7+ / curl fallback
  $curl = Get-Command curl.exe -ErrorAction SilentlyContinue
  if ($curl) {
    $effective = & curl.exe -sI -o NUL -w '%{url_effective}' -L $url
    if ($effective -match '/tag/([^/?#]+)') { return $Matches[1] }
  }

  $page = Invoke-WebRequest -Uri $url -UseBasicParsing -Headers @{ 'User-Agent' = $ua } -MaximumRedirection 5
  $href = $page.BaseResponse.ResponseUri.AbsoluteUri
  if ($href -match '/tag/([^/?#]+)') { return $Matches[1] }
  if ($page.Content -match '/MetaCubeX/mihomo/releases/tag/([A-Za-z0-9._-]+)') { return $Matches[1] }
  throw "Could not resolve latest mihomo tag from $url"
}

function Get-LatestTag {
  if ($env:MIHOMO_TAG) {
    Write-Host "Using pinned MIHOMO_TAG=$($env:MIHOMO_TAG)"
    return $env:MIHOMO_TAG
  }

  try {
    $release = Invoke-RestMethod -Uri 'https://api.github.com/repos/MetaCubeX/mihomo/releases/latest' -Headers $apiHeaders
    if ($release.tag_name) { return [string]$release.tag_name }
  } catch {
    Write-Warning "GitHub API unavailable: $($_.Exception.Message)"
  }

  Write-Host "Resolving latest tag via releases/latest redirect…"
  return Resolve-LatestTagFromRedirect
}

$tag = Get-LatestTag
$zipName = "mihomo-windows-amd64-compatible-$tag.zip"
$zipUrl = "https://github.com/MetaCubeX/mihomo/releases/download/$tag/$zipName"
$zip = Join-Path $env:TEMP $zipName

Write-Host "Downloading $zipName ..."
Invoke-WebRequest -Uri $zipUrl -OutFile $zip -UseBasicParsing -Headers @{ 'User-Agent' = $ua }

$extract = Join-Path $env:TEMP ("mihomo-extract-" + [guid]::NewGuid().ToString())
New-Item -ItemType Directory -Force -Path $extract | Out-Null
Expand-Archive -Path $zip -DestinationPath $extract -Force

$exe = Get-ChildItem -Path $extract -Recurse -Filter '*.exe' | Select-Object -First 1
if (-not $exe) { throw 'No exe found in archive' }

$exePath = Join-Path $outDir 'mihomo.exe'
Copy-Item $exe.FullName $exePath -Force

$hash = (Get-FileHash -Algorithm SHA256 -Path $exePath).Hash.ToLowerInvariant()
Set-Content -Path ($exePath + '.sha256') -Value "$hash  mihomo.exe" -NoNewline
Write-Host "Installed: $exePath"
Write-Host "SHA256: $hash"
Write-Host "Version tag: $tag"

$geoHeaders = @{ 'User-Agent' = $ua }
$geoFiles = @(
  @{ Name = 'geoip.metadb'; Url = 'https://github.com/MetaCubeX/meta-rules-dat/releases/download/latest/geoip.metadb' },
  @{ Name = 'geosite.dat'; Url = 'https://github.com/MetaCubeX/meta-rules-dat/releases/download/latest/geosite.dat' }
)
foreach ($g in $geoFiles) {
  $dest = Join-Path $outDir $g.Name
  Write-Host "Downloading $($g.Name)..."
  Invoke-WebRequest -Uri $g.Url -OutFile $dest -UseBasicParsing -Headers $geoHeaders
}
