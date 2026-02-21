package claude

import (
	"errors"
	"testing"
)

func TestParseHistory_ValidJSONL(t *testing.T) {
	data := []byte(`{"display":"fix the login bug","project":"/home/user/repo","sessionId":"sess-1","timestamp":1000}
{"display":"add tests","project":"/home/user/repo","sessionId":"sess-2","timestamp":2000}
`)
	entries, err := ParseHistory(data)
	if err != nil {
		t.Fatalf("ParseHistory failed: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("len(entries) = %d, want 2", len(entries))
	}
	if entries[0].Display != "fix the login bug" {
		t.Errorf("entries[0].Display = %q, want %q", entries[0].Display, "fix the login bug")
	}
	if entries[1].SessionID != "sess-2" {
		t.Errorf("entries[1].SessionID = %q, want %q", entries[1].SessionID, "sess-2")
	}
}

func TestParseHistory_MalformedLines(t *testing.T) {
	data := []byte(`{"display":"valid","project":"/repo","sessionId":"s1","timestamp":100}
not-json
{"display":"also valid","project":"/repo","sessionId":"s2","timestamp":200}
`)
	entries, err := ParseHistory(data)
	if err != nil {
		t.Fatalf("ParseHistory failed: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("len(entries) = %d, want 2 (malformed line should be skipped)", len(entries))
	}
}

func TestParseHistory_EmptyInput(t *testing.T) {
	entries, err := ParseHistory([]byte{})
	if err != nil {
		t.Fatalf("ParseHistory failed: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("len(entries) = %d, want 0", len(entries))
	}
}

func TestFindFirstPrompt_MatchByProject(t *testing.T) {
	entries := []HistoryEntry{
		{Display: "fix auth in other repo", Project: "/other/repo", SessionID: "s1", Timestamp: 100},
		{Display: "add user settings page to the dashboard", Project: "/my/repo", SessionID: "s2", Timestamp: 200},
	}

	prompt, sessionID, found := FindFirstPrompt(entries, "/my/repo", 0)
	if !found {
		t.Fatal("expected to find prompt, got not found")
	}
	if prompt != "add user settings page to the dashboard" {
		t.Errorf("prompt = %q, want %q", prompt, "add user settings page to the dashboard")
	}
	if sessionID != "s2" {
		t.Errorf("sessionID = %q, want %q", sessionID, "s2")
	}
}

func TestFindFirstPrompt_MatchByTimestamp(t *testing.T) {
	entries := []HistoryEntry{
		{Display: "old prompt that should be ignored here", Project: "/my/repo", SessionID: "s1", Timestamp: 100},
		{Display: "new prompt after worktree creation", Project: "/my/repo", SessionID: "s2", Timestamp: 300},
	}

	prompt, _, found := FindFirstPrompt(entries, "/my/repo", 200)
	if !found {
		t.Fatal("expected to find prompt")
	}
	if prompt != "new prompt after worktree creation" {
		t.Errorf("prompt = %q, want %q", prompt, "new prompt after worktree creation")
	}
}

func TestFindFirstPrompt_SkipsCommands(t *testing.T) {
	entries := []HistoryEntry{
		{Display: "/commit -m fix", Project: "/my/repo", SessionID: "s1", Timestamp: 100},
		{Display: "exit", Project: "/my/repo", SessionID: "s1", Timestamp: 200},
		{Display: "go build ./...", Project: "/my/repo", SessionID: "s1", Timestamp: 300},
		{Display: "implement dark mode for the user profile page", Project: "/my/repo", SessionID: "s2", Timestamp: 400},
	}

	prompt, _, found := FindFirstPrompt(entries, "/my/repo", 0)
	if !found {
		t.Fatal("expected to find prompt")
	}
	if prompt != "implement dark mode for the user profile page" {
		t.Errorf("prompt = %q, want %q", prompt, "implement dark mode for the user profile page")
	}
}

func TestFindFirstPrompt_MinLength(t *testing.T) {
	entries := []HistoryEntry{
		{Display: "short", Project: "/my/repo", SessionID: "s1", Timestamp: 100},
		{Display: "a", Project: "/my/repo", SessionID: "s1", Timestamp: 200},
		{Display: "refactor the authentication module to use JWT", Project: "/my/repo", SessionID: "s2", Timestamp: 300},
	}

	prompt, _, found := FindFirstPrompt(entries, "/my/repo", 0)
	if !found {
		t.Fatal("expected to find prompt")
	}
	if prompt != "refactor the authentication module to use JWT" {
		t.Errorf("prompt = %q, want %q", prompt, "refactor the authentication module to use JWT")
	}
}

func TestFindFirstPrompt_NotFound(t *testing.T) {
	entries := []HistoryEntry{
		{Display: "fix the bug in payment processing", Project: "/other/repo", SessionID: "s1", Timestamp: 100},
	}

	_, _, found := FindFirstPrompt(entries, "/my/repo", 0)
	if found {
		t.Error("expected not found, got found")
	}
}

func TestFindFirstPrompt_EmptyEntries(t *testing.T) {
	_, _, found := FindFirstPrompt(nil, "/my/repo", 0)
	if found {
		t.Error("expected not found, got found")
	}
}

func TestOSReader_ReadHistoryFile(t *testing.T) {
	reader := OSReader{HistoryPath: "/nonexistent/path/history.jsonl"}
	_, err := reader.ReadHistoryFile()
	if err == nil {
		t.Error("expected error for nonexistent file, got nil")
	}
}

func TestFakeReader_ReadHistoryFile(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		reader := FakeReader{Data: []byte("test"), Err: nil}
		data, err := reader.ReadHistoryFile()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(data) != "test" {
			t.Errorf("data = %q, want %q", string(data), "test")
		}
	})

	t.Run("error", func(t *testing.T) {
		reader := FakeReader{Err: errors.New("read error")}
		_, err := reader.ReadHistoryFile()
		if err == nil {
			t.Error("expected error, got nil")
		}
	})
}

func TestIsSkippable(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"short", true},
		{"", true},
		{"a", true},
		{"/commit", true},
		{"/review-pr 123", true},
		{"exit", true},
		{"quit", true},
		{"go build ./...", true},
		{"yes", true},
		{"no", true},
		{"y", true},
		{"n", true},
		{"implement the dark mode feature", false},
		{"fix the login redirect bug", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := isSkippable(tt.input)
			if got != tt.want {
				t.Errorf("isSkippable(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
