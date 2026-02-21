package claude

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os"
	"strings"
)

// HistoryEntry represents a single line from ~/.claude/history.jsonl.
type HistoryEntry struct {
	Display   string `json:"display"`
	Project   string `json:"project"`
	SessionID string `json:"sessionId"`
	Timestamp int64  `json:"timestamp"`
}

// Reader abstracts file system access for testability.
type Reader interface {
	ReadHistoryFile() ([]byte, error)
}

// OSReader reads the real file from ~/.claude/history.jsonl.
type OSReader struct {
	HistoryPath string
}

func (r OSReader) ReadHistoryFile() ([]byte, error) {
	return os.ReadFile(r.HistoryPath)
}

// FakeReader is a test double.
type FakeReader struct {
	Data []byte
	Err  error
}

func (r FakeReader) ReadHistoryFile() ([]byte, error) {
	return r.Data, r.Err
}

// minPromptLength is the minimum character count for a prompt to be considered
// meaningful enough for branch naming.
const minPromptLength = 10

// skipPrefixes lists command-like inputs that should be ignored.
var skipPrefixes = []string{"/", "exit", "quit", "q", "go", "yes", "no", "y", "n"}

// ParseHistory parses JSONL content into HistoryEntry slices.
// Malformed lines are silently skipped.
func ParseHistory(data []byte) ([]HistoryEntry, error) {
	var entries []HistoryEntry
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		var entry HistoryEntry
		if err := json.Unmarshal(line, &entry); err != nil {
			continue // skip malformed lines
		}
		entries = append(entries, entry)
	}
	return entries, scanner.Err()
}

// FindFirstPrompt searches history entries for the first meaningful user prompt
// in a session that started within the given worktree path after the given timestamp.
// Returns the prompt text, session ID, and whether a match was found.
func FindFirstPrompt(entries []HistoryEntry, worktreePath string, afterTimestamp int64) (prompt string, sessionID string, found bool) {
	for _, e := range entries {
		if e.Project != worktreePath {
			continue
		}
		if e.Timestamp < afterTimestamp {
			continue
		}
		if isSkippable(e.Display) {
			continue
		}
		return e.Display, e.SessionID, true
	}
	return "", "", false
}

// isSkippable returns true if the prompt is too short or looks like a command.
func isSkippable(display string) bool {
	trimmed := strings.TrimSpace(display)
	if len(trimmed) < minPromptLength {
		return true
	}
	// Slash commands (e.g., /commit, /review-pr 123)
	if strings.HasPrefix(trimmed, "/") {
		return true
	}
	lower := strings.ToLower(trimmed)
	for _, prefix := range skipPrefixes {
		if prefix == "/" {
			continue
		}
		if lower == prefix || strings.HasPrefix(lower, prefix+" ") {
			return true
		}
	}
	return false
}
