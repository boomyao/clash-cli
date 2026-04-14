package core

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
)

// State represents the mihomo process state.
type State int

const (
	StateStopped State = iota
	StateStarting
	StateRunning
	StateStopping
)

func (s State) String() string {
	switch s {
	case StateStopped:
		return "stopped"
	case StateStarting:
		return "starting"
	case StateRunning:
		return "running"
	case StateStopping:
		return "stopping"
	default:
		return "unknown"
	}
}

// Manager manages the mihomo subprocess lifecycle.
type Manager struct {
	binaryPath string
	configDir  string
	extCtl     string // external controller address
	logPath    string // where to write mihomo's stdout/stderr (empty = discard)

	mu       sync.Mutex
	cmd      *exec.Cmd
	state    State
	logFile  *os.File

	// exitCh receives a signal when the process exits unexpectedly
	exitCh chan struct{}
}

// NewManager creates a core process manager.
// logPath is where mihomo's stdout/stderr will be written; if empty, output is discarded.
func NewManager(binaryPath, configDir, extCtl, logPath string) *Manager {
	return &Manager{
		binaryPath: binaryPath,
		configDir:  configDir,
		extCtl:     extCtl,
		logPath:    logPath,
		state:      StateStopped,
		exitCh:     make(chan struct{}, 1),
	}
}

// LogPath returns the path mihomo writes its logs to (may be empty).
func (m *Manager) LogPath() string {
	return m.logPath
}

// State returns the current process state.
func (m *Manager) State() State {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.state
}

// ExitCh returns a channel that is signaled when the process exits.
func (m *Manager) ExitCh() <-chan struct{} {
	return m.exitCh
}

// Start launches the mihomo process.
func (m *Manager) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.state == StateRunning || m.state == StateStarting {
		return fmt.Errorf("core is already %s", m.state)
	}

	m.state = StateStarting

	args := []string{}
	if m.configDir != "" {
		args = append(args, "-d", m.configDir)
	}
	if m.extCtl != "" {
		args = append(args, "-ext-ctl", m.extCtl)
	}

	m.cmd = exec.Command(m.binaryPath, args...)

	// Critical: never write mihomo's logs to our TTY — it would corrupt the TUI.
	var output io.Writer = io.Discard
	if m.logPath != "" {
		f, err := os.OpenFile(m.logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err == nil {
			m.logFile = f
			output = f
		}
	}
	m.cmd.Stdout = output
	m.cmd.Stderr = output

	if err := m.cmd.Start(); err != nil {
		m.state = StateStopped
		if m.logFile != nil {
			m.logFile.Close()
			m.logFile = nil
		}
		return fmt.Errorf("failed to start mihomo: %w", err)
	}

	m.state = StateRunning

	// Monitor the process in background
	go func() {
		_ = m.cmd.Wait()
		m.mu.Lock()
		m.state = StateStopped
		m.cmd = nil
		if m.logFile != nil {
			m.logFile.Close()
			m.logFile = nil
		}
		m.mu.Unlock()

		// Signal that the process exited
		select {
		case m.exitCh <- struct{}{}:
		default:
		}
	}()

	return nil
}

// Stop terminates the mihomo process.
func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.state != StateRunning {
		return fmt.Errorf("core is not running (state: %s)", m.state)
	}

	m.state = StateStopping

	if m.cmd != nil && m.cmd.Process != nil {
		if err := m.cmd.Process.Signal(os.Interrupt); err != nil {
			// If interrupt fails, force kill
			_ = m.cmd.Process.Kill()
		}
	}

	return nil
}

// Restart stops and restarts the mihomo process.
func (m *Manager) Restart() error {
	if m.State() == StateRunning {
		if err := m.Stop(); err != nil {
			return err
		}
		// Wait briefly for the process to exit
		<-m.exitCh
	}
	return m.Start()
}

// IsRunning returns whether the core process is running.
func (m *Manager) IsRunning() bool {
	return m.State() == StateRunning
}
