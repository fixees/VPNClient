package compat

import (
	_ "embed"
	"path/filepath"
	"strings"
)

//go:embed conflicts.json
var embeddedCatalog []byte

// InitEmbeddedCatalog installs the built-in signature pack.
func InitEmbeddedCatalog() error {
	return SetDefaultCatalogJSON(embeddedCatalog)
}

// LoadDefault loads embedded signatures, then optionally overrides from
// <dataDir>/compat/conflicts.json so operators can update without rebuilding.
func LoadDefault(dataDir string) {
	_ = InitEmbeddedCatalog()
	dataDir = strings.TrimSpace(dataDir)
	if dataDir == "" {
		return
	}
	override := filepath.Join(dataDir, "compat", "conflicts.json")
	if c, err := LoadCatalog(override); err == nil && len(c.Tools) > 0 {
		SetDefaultCatalog(c)
	}
}
