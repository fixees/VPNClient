package tests

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"myinternetvpn/client/internal/winutil"
)

func TestPIDTrackerRecordAndLoad(t *testing.T) {
	if os.Getenv("GOOS") != "" && os.Getenv("GOOS") != "windows" {
		t.Skip("Windows-only test")
	}

	tmpDir := t.TempDir()
	pidFile := filepath.Join(tmpDir, "test.pid")
	tracker := winutil.NewPIDTracker(pidFile)

	// Record a fake core PID.
	fakePID := 99999
	if err := tracker.RecordCorePID(fakePID); err != nil {
		t.Fatalf("RecordCorePID: %v", err)
	}

	// Verify file was written.
	raw, err := os.ReadFile(pidFile)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	type pidTrackState struct {
		MainPID int       `json:"mainPid"`
		CorePID int       `json:"corePid"`
		Started time.Time `json:"started"`
	}
	var state pidTrackState
	if err := json.Unmarshal(raw, &state); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if state.CorePID != fakePID {
		t.Errorf("CorePID: got %d, want %d", state.CorePID, fakePID)
	}
	if state.MainPID != os.Getpid() {
		t.Errorf("MainPID: got %d, want %d", state.MainPID, os.Getpid())
	}

	// LoadOrphans should return false because MainPID matches our current process.
	corePID, mainPID, isOrphan := tracker.LoadOrphans()
	if isOrphan {
		t.Errorf("LoadOrphans: expected not orphan (same process), got orphan=true")
	}
	if corePID != 0 || mainPID != 0 {
		t.Errorf("LoadOrphans: expected zeros when not orphan, got core=%d main=%d", corePID, mainPID)
	}

	// Clear should remove the file.
	if err := tracker.Clear(); err != nil {
		t.Fatalf("Clear: %v", err)
	}
	if _, err := os.Stat(pidFile); !os.IsNotExist(err) {
		t.Errorf("Clear: file still exists")
	}
}

func TestPIDTrackerOrphanDetection(t *testing.T) {
	if os.Getenv("GOOS") != "" && os.Getenv("GOOS") != "windows" {
		t.Skip("Windows-only test")
	}

	tmpDir := t.TempDir()
	pidFile := filepath.Join(tmpDir, "orphan.pid")

	// Manually write a PID file with a fake (non-existent) main PID.
	fakeMainPID := 1 // PID 1 doesn't exist on Windows
	fakeCorePID := 2
	type pidTrackState struct {
		MainPID int       `json:"mainPid"`
		CorePID int       `json:"corePid"`
		Started time.Time `json:"started"`
	}
	state := pidTrackState{
		MainPID: fakeMainPID,
		CorePID: fakeCorePID,
		Started: time.Now(),
	}
	raw, _ := json.Marshal(state)
	if err := os.WriteFile(pidFile, raw, 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	tracker := winutil.NewPIDTracker(pidFile)
	corePID, mainPID, isOrphan := tracker.LoadOrphans()
	if !isOrphan {
		t.Errorf("LoadOrphans: expected orphan=true, got false")
	}
	if corePID != fakeCorePID {
		t.Errorf("CorePID: got %d, want %d", corePID, fakeCorePID)
	}
	if mainPID != fakeMainPID {
		t.Errorf("MainPID: got %d, want %d", mainPID, fakeMainPID)
	}
}

func TestIsProcessAlive(t *testing.T) {
	if os.Getenv("GOOS") != "" && os.Getenv("GOOS") != "windows" {
		t.Skip("Windows-only test")
	}

	// Our own PID should be alive.
	if !winutil.IsProcessAlive(os.Getpid()) {
		t.Errorf("IsProcessAlive(self): got false, want true")
	}

	// PID 1 doesn't exist on Windows.
	if winutil.IsProcessAlive(1) {
		t.Errorf("IsProcessAlive(1): got true, want false")
	}

	// Invalid PIDs.
	if winutil.IsProcessAlive(0) {
		t.Errorf("IsProcessAlive(0): got true, want false")
	}
	if winutil.IsProcessAlive(-1) {
		t.Errorf("IsProcessAlive(-1): got true, want false")
	}
}
