package core

import (
	"fmt"
	"os"
	"os/exec"
	"sync"
	"time"

	"myinternetvpn/client/internal/api"
)

// API is the subset of mihomo controller used by Manager.
type API interface {
	WaitHealthy(timeout time.Duration) error
	Healthy() bool
}

// ProcessRunner abstracts process start/stop for tests.
type ProcessRunner interface {
	Start(bin string, args []string, workDir string) error
	// Stop attempts a graceful shutdown, then force-kills after grace.
	Stop(grace time.Duration) error
	Running() bool
}

// Options configures the mihomo process manager.
type Options struct {
	CorePath      string
	WorkDir       string
	ConfigPath    string
	ControllerURL string
	Secret        string
	MixedPort     int
	API           API
	Runner        ProcessRunner
	ReadyTimeout  time.Duration
	StopGrace     time.Duration
}

// Manager owns the mihomo child process lifecycle.
type Manager struct {
	mu      sync.Mutex
	opts    Options
	running bool
}

func NewManager(opts Options) *Manager {
	if opts.Runner == nil {
		opts.Runner = &execRunner{}
	}
	if opts.ReadyTimeout == 0 {
		opts.ReadyTimeout = 8 * time.Second
	}
	if opts.StopGrace == 0 {
		opts.StopGrace = 3 * time.Second
	}
	if opts.API == nil {
		opts.API = api.NewClient(opts.ControllerURL, opts.Secret)
	}
	return &Manager{opts: opts}
}

func (m *Manager) Running() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.running && m.opts.Runner.Running()
}

func (m *Manager) Start(configYAML []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.opts.CorePath == "" {
		return fmt.Errorf("mihomo binary path is empty")
	}
	if _, err := os.Stat(m.opts.CorePath); err != nil {
		return fmt.Errorf("mihomo binary not found at %s: %w", m.opts.CorePath, err)
	}
	if err := os.MkdirAll(m.opts.WorkDir, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(m.opts.ConfigPath, configYAML, 0o600); err != nil {
		return err
	}

	if m.running {
		_ = m.opts.Runner.Stop(m.opts.StopGrace)
		m.running = false
	}

	args := []string{"-d", m.opts.WorkDir, "-f", m.opts.ConfigPath}
	if err := m.opts.Runner.Start(m.opts.CorePath, args, m.opts.WorkDir); err != nil {
		return fmt.Errorf("start mihomo: %w", err)
	}
	m.running = true

	if err := m.opts.API.WaitHealthy(m.opts.ReadyTimeout); err != nil {
		_ = m.opts.Runner.Stop(m.opts.StopGrace)
		m.running = false
		return fmt.Errorf("mihomo API not ready: %w", err)
	}
	return nil
}

func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.running {
		return nil
	}
	err := m.opts.Runner.Stop(m.opts.StopGrace)
	m.running = false
	return err
}

type execRunner struct {
	cmd  *exec.Cmd
	done chan error
}

func (r *execRunner) Start(bin string, args []string, workDir string) error {
	cmd := exec.Command(bin, args...)
	cmd.Dir = workDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return err
	}
	r.cmd = cmd
	r.done = make(chan error, 1)
	go func() {
		r.done <- cmd.Wait()
	}()
	return nil
}

func (r *execRunner) Stop(grace time.Duration) error {
	if r.cmd == nil || r.cmd.Process == nil {
		return nil
	}
	proc := r.cmd.Process
	_ = proc.Signal(os.Interrupt)

	timer := time.NewTimer(grace)
	defer timer.Stop()
	select {
	case <-r.done:
		r.cmd = nil
		return nil
	case <-timer.C:
		err := proc.Kill()
		<-r.done
		r.cmd = nil
		return err
	}
}

func (r *execRunner) Running() bool {
	return r.cmd != nil && r.cmd.Process != nil
}
