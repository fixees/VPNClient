# Tests

```powershell
go test ./tests/... -count=1
```

| File | Covers |
|------|--------|
| `api_test.go` | External Controller client basics |
| `subscription_test.go` | subscription fetch, quota headers, schedule due |
| `config_test.go` | Clash Meta YAML builder |
| `core_test.go` | process manager start/stop/rollback |
| `paths_test.go` | data/resource path resolver |
| `profiles_test.go` | profile store + settings |
| `integration_test.go` | profile → config → manager pipeline |
