package tests

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"myinternetvpn/client/internal/core"
)

type fakeAPI struct {
	mu       sync.Mutex
	healthy  bool
	failOnce int
}

func (f *fakeAPI) WaitHealthy(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		f.mu.Lock()
		ok := f.healthy
		if f.failOnce > 0 {
			f.failOnce--
			ok = false
		}
		f.mu.Unlock()
		if ok {
			return nil
		}
		time.Sleep(10 * time.Millisecond)
	}
	return errors.New("not healthy")
}

func (f *fakeAPI) Healthy() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.healthy
}

type fakeRunner struct {
	mu      sync.Mutex
	started bool
	bin     string
	args    []string
	stops   int
}

func (r *fakeRunner) Start(bin string, args []string, workDir string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.started = true
	r.bin = bin
	r.args = append([]string{}, args...)
	return nil
}

func (r *fakeRunner) Stop(grace time.Duration) error {
	_ = grace
	r.mu.Lock()
	defer r.mu.Unlock()
	r.started = false
	r.stops++
	return nil
}

func (r *fakeRunner) Running() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.started
}

func TestCoreManagerStartStop(t *testing.T) {
	tmp := t.TempDir()
	bin := filepath.Join(tmp, "mihomo.exe")
	if err := os.WriteFile(bin, []byte("fake"), 0o755); err != nil {
		t.Fatal(err)
	}
	work := filepath.Join(tmp, "work")
	cfg := filepath.Join(work, "config.yaml")

	apiStub := &fakeAPI{healthy: true}
	runner := &fakeRunner{}
	mgr := core.NewManager(core.Options{
		CorePath:     bin,
		WorkDir:      work,
		ConfigPath:   cfg,
		API:          apiStub,
		Runner:       runner,
		ReadyTimeout: time.Second,
	})

	if err := mgr.Start([]byte("mixed-port: 1\n")); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if !mgr.Running() {
		t.Fatal("expected running")
	}
	raw, err := os.ReadFile(cfg)
	if err != nil || string(raw) != "mixed-port: 1\n" {
		t.Fatalf("config not written: %v %q", err, raw)
	}
	if runner.bin != bin || len(runner.args) < 4 {
		t.Fatalf("unexpected start args: %s %v", runner.bin, runner.args)
	}

	if err := mgr.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if mgr.Running() {
		t.Fatal("expected stopped")
	}
	if runner.stops != 1 {
		t.Fatalf("stops = %d", runner.stops)
	}
}

func TestCoreManagerMissingBinary(t *testing.T) {
	mgr := core.NewManager(core.Options{
		CorePath: filepath.Join(t.TempDir(), "missing.exe"),
		WorkDir:  t.TempDir(),
		API:      &fakeAPI{healthy: true},
		Runner:   &fakeRunner{},
	})
	if err := mgr.Start([]byte("x: 1\n")); err == nil {
		t.Fatal("expected missing binary error")
	}
}

func TestCoreManagerAPINotReadyRollsBack(t *testing.T) {
	tmp := t.TempDir()
	bin := filepath.Join(tmp, "mihomo.exe")
	_ = os.WriteFile(bin, []byte("fake"), 0o755)

	runner := &fakeRunner{}
	mgr := core.NewManager(core.Options{
		CorePath:     bin,
		WorkDir:      filepath.Join(tmp, "work"),
		ConfigPath:   filepath.Join(tmp, "work", "config.yaml"),
		API:          &fakeAPI{healthy: false},
		Runner:       runner,
		ReadyTimeout: 50 * time.Millisecond,
	})
	if err := mgr.Start([]byte("x: 1\n")); err == nil {
		t.Fatal("expected API not ready error")
	}
	if mgr.Running() {
		t.Fatal("should not stay running after failed ready check")
	}
	if runner.stops == 0 {
		t.Fatal("expected rollback stop")
	}
}
