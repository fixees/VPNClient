param(
  [string]$Script = "src/main.py",
  [string]$Name = "iSecureClient",
  [string]$PyVer = "3.13"
)

$ErrorActionPreference = "Stop"

Write-Host "== iSecure Client build ==" -ForegroundColor Cyan
$PyArgs = @("-$PyVer")
Write-Host "Installing deps..." -ForegroundColor Cyan
& py @PyArgs -m pip install --upgrade pip
& py @PyArgs -m pip install -r requirements.txt

Write-Host "Cleaning old dist/build..." -ForegroundColor Cyan
if (Test-Path ".\dist") { Remove-Item -Recurse -Force ".\dist" }
if (Test-Path ".\build") { Remove-Item -Recurse -Force ".\build" }
if (Test-Path ".\*.spec") { Remove-Item -Force ".\*.spec" }

Write-Host "Preparing icon (.ico)..." -ForegroundColor Cyan
& py @PyArgs .\tools\make_icon.py ".\resources\images\nekobox.png" ".\resources\images\iSecureVPN.ico"

Write-Host "Running PyInstaller (onefile)..." -ForegroundColor Cyan
& py @PyArgs -m PyInstaller `
  --noconsole `
  --onefile `
  --name $Name `
  --icon ".\resources\images\iSecureVPN.ico" `
  --hidden-import "psutil" `
  --hidden-import "pystray" `
  --add-binary ".\resources\core\nekobox_core.exe;resources/core" `
  --add-data ".\resources\core\nekobox_core.sha256;resources/core" `
  --add-data ".\resources\data\geoip.db;resources/data" `
  --add-data ".\resources\data\geosite.db;resources/data" `
  --add-data ".\resources\images\nekobox.png;resources/images" `
  --add-data ".\resources\sounds\success_start.mp3;resources/sounds" `
  --add-data ".\resources\sounds\failed_start.mp3;resources/sounds" `
  $Script

Write-Host ""
Write-Host "Done. Output:" -ForegroundColor Green
Write-Host "  .\dist\$Name.exe" -ForegroundColor Green

