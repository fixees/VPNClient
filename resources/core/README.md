# Mihomo core + geo (build-time embed)

These files are **embedded into MyInternetVPN.exe** at compile time and extracted
to `%APPDATA%\MyInternetVPN\core\` on first launch. Users only need the single `.exe`.

Place here before `go build` / `.\scripts\build.ps1` / `.\scripts\package.ps1`:

```text
resources/core/mihomo.exe
resources/core/mihomo.exe.sha256
resources/core/geoip.metadb
resources/core/geosite.dat
```

Download everything with:

```powershell
.\scripts\download-mihomo.ps1
```

Integrity: on connect the app verifies `mihomo.exe` against `mihomo.exe.sha256` when present.
Set `requireCoreHash: true` in settings to make the sidecar mandatory.
