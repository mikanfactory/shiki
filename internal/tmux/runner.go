package tmux

import (
	"fmt"
	"os/exec"
)

// Runner abstracts tmux command execution for testability.
type Runner interface {
	Run(args ...string) (string, error)
}

// OSRunner executes real tmux commands via os/exec.
type OSRunner struct{}

func (r OSRunner) Run(args ...string) (string, error) {
	cmd := exec.Command("tmux", args...)
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
