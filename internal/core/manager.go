package core

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"myinternetvpn/client/internal/api"
	"myinternetvpn/client/internal/defaults"
	"myinternetvpn/client/internal/winutil"
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
		opts.ReadyTimeout = defaults.ReadyTimeout
	}
	if opts.StopGrace == 0 {
		opts.StopGrace = defaults.StopGrace
	}
	if opts.API == nil {
		opts.API = api.NewClient(opts.ControllerURL, opts.Secret)
	}
	return &Manager{opts: opts}
}

func (m *Manager) Running() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.running && !m.opts.Runner.Running() {
		m.running = false
	}
	return m.running
}

func (m *Manager) Start(configYAML []byte) error {
	m.mu.Lock()
	if m.opts.CorePath == "" {
		m.mu.Unlock()
		return fmt.Errorf("mihomo binary path is empty")
	}
	if _, err := os.Stat(m.opts.CorePath); err != nil {
		m.mu.Unlock()
		return fmt.Errorf("mihomo binary not found at %s: %w", m.opts.CorePath, err)
	}
	if err := os.MkdirAll(m.opts.WorkDir, 0o755); err != nil {
		m.mu.Unlock()
		return err
	}
	if err := os.WriteFile(m.opts.ConfigPath, configYAML, 0o600); err != nil {
		m.mu.Unlock()
		return err
	}

	if m.running {
		_ = m.opts.Runner.Stop(m.opts.StopGrace)
		m.running = false
	}

	args := []string{"-d", m.opts.WorkDir, "-f", m.opts.ConfigPath}
	apiClient := m.opts.API
	readyTimeout := m.opts.ReadyTimeout
	stopGrace := m.opts.StopGrace
	corePath := m.opts.CorePath
	workDir := m.opts.WorkDir
	runner := m.opts.Runner

	if err := runner.Start(corePath, args, workDir); err != nil {
		m.mu.Unlock()
		return fmt.Errorf("start mihomo: %w", err)
	}
	m.running = true
	m.mu.Unlock()

	// Wait for controller outside the lock so Stop/Running stay responsive.
	if err := apiClient.WaitHealthy(readyTimeout); err != nil {
		m.mu.Lock()
		_ = runner.Stop(stopGrace)
		m.running = false
		m.mu.Unlock()
		if detail := lastMihomoFatal(workDir); detail != "" {
			return fmt.Errorf("mihomo API not ready: %w (%s)", err, detail)
		}
		return fmt.Errorf("mihomo API not ready: %w", err)
	}
	return nil
}

func lastMihomoFatal(workDir string) string {
	raw, err := os.ReadFile(filepath.Join(workDir, "mihomo.log"))
	if err != nil || len(raw) == 0 {
		return ""
	}
	const maxTail = 8 << 10
	if len(raw) > maxTail {
		raw = raw[len(raw)-maxTail:]
	}
	lines := strings.Split(string(raw), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		if strings.Contains(line, "level=fatal") || strings.Contains(line, "Parse config error") {
			if idx := strings.Index(line, "msg="); idx >= 0 {
				msg := strings.Trim(line[idx+4:], `"`)
				if msg != "" {
					return msg
				}
			}
			return line
		}
	}
	return ""
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
	mu      sync.Mutex
	cmd     *exec.Cmd
	logFile *os.File
	done    chan error
	alive   bool
}

func (r *execRunner) Start(bin string, args []string, workDir string) error {
	cmd := exec.Command(bin, args...)
	cmd.Dir = workDir

	// Keep mihomo output in a file — never attach to a visible console.
	logPath := filepath.Join(workDir, "mihomo.log")
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	winutil.HideConsole(cmd)

	if err := cmd.Start(); err != nil {
		_ = logFile.Close()
		return err
	}
	// Kill mihomo when the UI process dies (Task Manager / crash).
	_ = winutil.AssignProcessToChildKillJob(cmd.Process.Pid)

	r.mu.Lock()
	r.cmd = cmd
	r.logFile = logFile
	r.done = make(chan error, 1)
	r.alive = true
	done := r.done
	r.mu.Unlock()

	go func() {
		waitErr := cmd.Wait()
		r.mu.Lock()
		r.alive = false
		if r.logFile != nil {
			_ = r.logFile.Close()
			r.logFile = nil
		}
		r.mu.Unlock()
		done <- waitErr
	}()
	return nil
}

func (r *execRunner) Stop(grace time.Duration) error {
	r.mu.Lock()
	cmd := r.cmd
	done := r.done
	alive := r.alive
	r.mu.Unlock()
	if cmd == nil || cmd.Process == nil {
		return nil
	}
	proc := cmd.Process
	if !alive {
		r.mu.Lock()
		r.cmd = nil
		r.mu.Unlock()
		return nil
	}
	if runtime.GOOS == "windows" {
		// Interrupt is unreliable for Win32 console-less children; kill promptly.
		err := proc.Kill()
		if done != nil {
			<-done
		}
		r.mu.Lock()
		r.cmd = nil
		r.alive = false
		r.mu.Unlock()
		return err
	}
	_ = proc.Signal(os.Interrupt)

	timer := time.NewTimer(grace)
	defer timer.Stop()
	select {
	case <-done:
		r.mu.Lock()
		r.cmd = nil
		r.alive = false
		r.mu.Unlock()
		return nil
	case <-timer.C:
		err := proc.Kill()
		if done != nil {
			<-done
		}
		r.mu.Lock()
		r.cmd = nil
		r.alive = false
		r.mu.Unlock()
		return err
	}
}

func (r *execRunner) Running() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.alive && r.cmd != nil && r.cmd.Process != nil
}
