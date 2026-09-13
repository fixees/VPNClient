# Mihomo core + geo

Place here:

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
