//go:build windows

package winutil

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

// PIDTrackerInterface defines the contract for tracking child process PIDs.
// This allows the core manager to work with or without tracking.
type PIDTrackerInterface interface {
	RecordCorePID(corePID int) error
	Clear() error
	LoadOrphans() (corePID, mainPID int, isOrphan bool)
}

// PIDTracker records child process PIDs to a file so we can recognize orphans on restart.
type PIDTracker struct {
	mu       sync.Mutex
	filePath string
	state    pidTrackState
}

type pidTrackState struct {
	MainPID int       `json:"mainPid"`
	CorePID int       `json:"corePid"`
	Started time.Time `json:"started"`
}

// NewPIDTracker creates a tracker backed by filePath.
func NewPIDTracker(filePath string) *PIDTracker {
	return &PIDTracker{filePath: filePath}
}

// RecordCorePID writes the main process PID and mihomo core PID to disk.
func (p *PIDTracker) RecordCorePID(corePID int) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	mainPID := os.Getpid()
	p.state = pidTrackState{
		MainPID: mainPID,
		CorePID: corePID,
		Started: time.Now(),
	}
	raw, err := json.Marshal(p.state)
	if err != nil {
		return err
	}
	dir := filepath.Dir(p.filePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(p.filePath, raw, 0o600)
}

// Clear removes the PID file (called on clean exit).
func (p *PIDTracker) Clear() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.state = pidTrackState{}
	err := os.Remove(p.filePath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// LoadOrphans reads the PID file, validates it's not our current run, and returns the core PID if orphaned.
// Returns (corePID, mainPID, isOrphan).
func (p *PIDTracker) LoadOrphans() (int, int, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	raw, err := os.ReadFile(p.filePath)
	if err != nil {
		return 0, 0, false
	}
	var st pidTrackState
	if err := json.Unmarshal(raw, &st); err != nil {
		return 0, 0, false
	}
	// If the recorded main PID matches our current process, it's us (not an orphan).
	if st.MainPID == os.Getpid() {
		return 0, 0, false
	}
	// Check if the recorded main PID still exists (it might have been force-killed).
	// If it doesn't exist, the core PID is likely orphaned.
	if !processExists(uint32(st.MainPID)) {
		return st.CorePID, st.MainPID, true
	}
	// Main process still running — not our concern.
	return 0, 0, false
}

// processExists checks if a process with the given PID is currently running.
func processExists(pid uint32) bool {
	if pid == 0 {
		return false
	}
	// Open with minimal access (PROCESS_QUERY_LIMITED_INFORMATION) to check existence.
	h, err := openProcessForQuery(pid)
	if err != nil {
		return false
	}
	defer procCloseHandle.Call(uintptr(h))
	// If we can open it, it exists.
	return true
}

func openProcessForQuery(pid uint32) (uintptr, error) {
	const processQueryLimited = 0x1000
	h, _, err := procOpenProcess.Call(processQueryLimited, 0, uintptr(pid))
	if h == 0 {
		return 0, fmt.Errorf("OpenProcess(%d): %w", pid, err)
	}
	return h, nil
}

// FormatPIDFile returns a suggested PID file path for a data directory.
func FormatPIDFile(dataDir string) string {
	return filepath.Join(dataDir, "core.pid")
}

// KillPID terminates a process by PID if it's still running.
func KillPID(pid int) error {
	if pid <= 0 {
		return fmt.Errorf("invalid pid %d", pid)
	}
	if !processExists(uint32(pid)) {
		return nil // Already dead
	}
	return terminatePID(uint32(pid))
}

// IsProcessAlive checks if a process with the given PID is running.
func IsProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	return processExists(uint32(pid))
}

// GetProcessImagePath returns the full executable path for a PID, or empty string on error.
func GetProcessImagePath(pid int) string {
	if pid <= 0 {
		return ""
	}
	return queryProcessPath(uint32(pid))
}

// ParsePIDFromString safely converts a string to an int PID.
func ParsePIDFromString(s string) int {
	pid, err := strconv.Atoi(s)
	if err != nil || pid <= 0 {
		return 0
	}
	return pid
}
