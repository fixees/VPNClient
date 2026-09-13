package paths

import (
	"os"
	"path/filepath"
	"runtime"

	"myinternetvpn/client/internal/defaults"
)

// Resolver locates app data and runtime core files.
type Resolver interface {
	DataDir() string
	ProfilesFile() string
	SettingsFile() string
	CoreWorkDir() string
	RuntimeConfig() string
	CoreBinary() string
	Ensure() error
}

type defaultResolver struct {
	dataDir string
}

// New returns a Windows-oriented path resolver.
// Core binary lives under DataDir/core after embedded resources are extracted.
func New() Resolver {
	appData, err := os.UserConfigDir()
	if err != nil || appData == "" {
		appData = "."
	}
	return &defaultResolver{
		dataDir: filepath.Join(appData, defaults.DataDirName),
	}
}

func (r *defaultResolver) DataDir() string       { return r.dataDir }
func (r *defaultResolver) ProfilesFile() string  { return filepath.Join(r.dataDir, "profiles.json") }
func (r *defaultResolver) SettingsFile() string  { return filepath.Join(r.dataDir, "settings.json") }
func (r *defaultResolver) CoreWorkDir() string   { return filepath.Join(r.dataDir, "core") }
func (r *defaultResolver) RuntimeConfig() string { return filepath.Join(r.dataDir, "core", "config.yaml") }

func (r *defaultResolver) CoreBinary() string {
	name := "mihomo"
	if runtime.GOOS == "windows" {
		name = "mihomo.exe"
	}
	return filepath.Join(r.CoreWorkDir(), name)
}

func (r *defaultResolver) Ensure() error {
	for _, dir := range []string{r.dataDir, r.CoreWorkDir()} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return nil
}

// NewWithRoots is used by tests (dataDir only; core is always under dataDir/core).
func NewWithRoots(dataDir, _ string) Resolver {
	return &defaultResolver{dataDir: dataDir}
}
