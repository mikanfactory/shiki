package tmux

import (
	"fmt"
	"testing"
)

func TestFakeRunner_ReturnsOutput(t *testing.T) {
	runner := FakeRunner{
		Outputs: map[string]string{
			"[list-windows -F #{window_name}]": "main\nfeature\n",
		},
	}

	out, err := runner.Run("list-windows", "-F", "#{window_name}")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "main\nfeature\n" {
		t.Errorf("unexpected output: %q", out)
	}
}

func TestFakeRunner_ReturnsError(t *testing.T) {
	runner := FakeRunner{
		Errors: map[string]error{
			"[select-window -t 1]": fmt.Errorf("tmux failed"),
		},
	}

	_, err := runner.Run("select-window", "-t", "1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestFakeRunner_NoOutput(t *testing.T) {
	runner := FakeRunner{
		Outputs: map[string]string{},
	}

	_, err := runner.Run("unknown")
	if err == nil {
		t.Fatal("expected error for missing key, got nil")
	}
}

func TestOSRunner_InvalidCommand(t *testing.T) {
	runner := OSRunner{}
	_, err := runner.Run("invalid-subcommand-that-does-not-exist")
	if err == nil {
		t.Fatal("expected error for invalid tmux subcommand")
	}
}

func TestFakeRunner_RecordsCalls(t *testing.T) {
	runner := &FakeRunner{
		Outputs: map[string]string{
			"[list-windows]": "",
		},
	}

	runner.Run("list-windows")

	if len(runner.Calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(runner.Calls))
	}
	if runner.Calls[0][0] != "list-windows" {
		t.Errorf("unexpected call args: %v", runner.Calls[0])
	}
}
