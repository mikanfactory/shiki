package tmux

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

// Runner abstracts tmux command execution for testability.
type Runner interface {
	Run(args ...string) (string, error)
}

// OSRunner executes real tmux commands via os/exec.
type OSRunner struct{}

var (
	resolvedTmuxPath string
	resolveTmuxOnce  sync.Once
)

// tmuxBinary returns the path to the tmux binary that matches the running server.
// When $TMUX is set (i.e. running inside tmux), the server's PID tells us which
// binary started the server. We resolve /proc/<pid>/exe (macOS: lsof) to find
// the exact binary. This avoids version-mismatch errors when multiple tmux
// versions are installed (e.g. homebrew 3.4 vs mise 3.6a).
func tmuxBinary() string {
	resolveTmuxOnce.Do(func() {
		resolvedTmuxPath = resolveTmuxFromServer()
	})
	return resolvedTmuxPath
}

func resolveTmuxFromServer() string {
	tmuxEnv := os.Getenv("TMUX")
	if tmuxEnv == "" {
		return "tmux"
	}

	// $TMUX format: socket_path,server_pid,session_index
	parts := strings.SplitN(tmuxEnv, ",", 3)
	if len(parts) < 2 || parts[1] == "" {
		return "tmux"
	}
	pid := parts[1]

	// On macOS, use lsof to find the binary for the server PID.
	out, err := exec.Command("lsof", "-p", pid, "-Fn").Output()
	if err != nil {
		return "tmux"
	}

	// lsof -Fn output: lines starting with "n" are file names.
	// The first "n" line after a "f" line with "ftxt" is the executable path.
	lines := strings.Split(string(out), "\n")
	foundTxt := false
	for _, line := range lines {
		if line == "ftxt" {
			foundTxt = true
			continue
		}
		if foundTxt && strings.HasPrefix(line, "n") {
			candidate := line[1:] // strip "n" prefix
			if filepath.Base(candidate) == "tmux" {
				return candidate
			}
		}
		if strings.HasPrefix(line, "f") && line != "ftxt" {
			foundTxt = false
		}
	}

	return "tmux"
}

func (r OSRunner) Run(args ...string) (string, error) {
	cmd := exec.Command(tmuxBinary(), args...)
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("tmux %v failed: %s", args, string(exitErr.Stderr))
		}
		return "", fmt.Errorf("tmux %v failed: %w", args, err)
	}
	return string(out), nil
}

// FakeRunner is a test double that returns preset output and records calls.
type FakeRunner struct {
	Outputs map[string]string
	Errors  map[string]error
	Calls   [][]string
}

func (r *FakeRunner) key(args ...string) string {
	return fmt.Sprintf("%v", args)
}

func (r *FakeRunner) Run(args ...string) (string, error) {
	r.Calls = append(r.Calls, args)
	key := r.key(args...)
	if r.Errors != nil {
		if err, ok := r.Errors[key]; ok {
			return "", err
		}
	}
	if r.Outputs != nil {
		if out, ok := r.Outputs[key]; ok {
			return out, nil
		}
	}
	return "", fmt.Errorf("FakeRunner: no output for key %q", key)
}
