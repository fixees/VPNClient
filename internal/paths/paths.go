package paths

import (
	"os"
	"path/filepath"
	"runtime"
)

// Resolver locates app data and bundled resources.
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
	dataDir     string
	resourceDir string
}

// New returns a Windows-oriented path resolver.
func New() Resolver {
	appData, err := os.UserConfigDir()
	if err != nil || appData == "" {
		appData = "."
	}
	dataDir := filepath.Join(appData, "MyInternetVPN")

	exe, err := os.Executable()
	if err != nil {
		exe = "."
	}
	resourceDir := filepath.Join(filepath.Dir(exe), "resources")
	// Dev fallback: resources next to module root.
	if _, err := os.Stat(resourceDir); err != nil {
		if wd, err := os.Getwd(); err == nil {
			candidate := filepath.Join(wd, "resources")
			if _, err := os.Stat(candidate); err == nil {
				resourceDir = candidate
			}
		}
	}

	return &defaultResolver{
		dataDir:     dataDir,
		resourceDir: resourceDir,
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
	return filepath.Join(r.resourceDir, "core", name)
}

func (r *defaultResolver) Ensure() error {
	for _, dir := range []string{r.dataDir, r.CoreWorkDir()} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return nil
}

// NewWithRoots is used by tests.
func NewWithRoots(dataDir, resourceDir string) Resolver {
	return &defaultResolver{dataDir: dataDir, resourceDir: resourceDir}
}
