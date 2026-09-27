//go:build !windows

package winutil

// PIDTrackerInterface stub for non-Windows platforms.
type PIDTrackerInterface interface {
	RecordCorePID(corePID int) error
	Clear() error
	LoadOrphans() (corePID, mainPID int, isOrphan bool)
}

type PIDTracker struct{}

func NewPIDTracker(filePath string) *PIDTracker                     { return &PIDTracker{} }
func (p *PIDTracker) RecordCorePID(corePID int) error               { return nil }
func (p *PIDTracker) Clear() error                                  { return nil }
func (p *PIDTracker) LoadOrphans() (corePID, mainPID int, ok bool) { return 0, 0, false }

func FormatPIDFile(dataDir string) string { return "" }
func KillPID(pid int) error                { return ErrNotWindows }
func IsProcessAlive(pid int) bool          { return false }
func GetProcessImagePath(pid int) string   { return "" }
func ParsePIDFromString(s string) int      { return 0 }
