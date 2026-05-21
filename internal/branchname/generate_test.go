package branchname

import (
	"testing"
)

func TestSanitizeBranchName_KebabCase(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"fix-login-redirect", "fix-login-redirect"},
		{"Fix Login Redirect", "fix-login-redirect"},
		{"ADD_USER_SETTINGS", "addusersettings"},
		{"simple", "simple"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := SanitizeBranchName(tt.input)
			if got != tt.want {
				t.Errorf("SanitizeBranchName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSanitizeBranchName_TruncatesAt30Chars(t *testing.T) {
	long := "this-is-a-very-long-branch-name-that-exceeds-thirty-characters"
	got := SanitizeBranchName(long)
	if len(got) > maxBranchNameLength {
		t.Errorf("len(SanitizeBranchName(%q)) = %d, want <= %d", long, len(got), maxBranchNameLength)
	}
}

func TestSanitizeBranchName_RemovesSpecialChars(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"fix: login bug!", "fix-login-bug"},
		{"feature/add-auth", "featureadd-auth"},
		{"hello@world#test", "helloworldtest"},
		{"  spaces  ", "spaces"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := SanitizeBranchName(tt.input)
			if got != tt.want {
				t.Errorf("SanitizeBranchName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSanitizeBranchName_NoTrailingHyphen(t *testing.T) {
	input := "a-very-long-branch-name-ending-"
	got := SanitizeBranchName(input)
	if got[len(got)-1] == '-' {
		t.Errorf("SanitizeBranchName(%q) = %q, should not end with hyphen", input, got)
	}
}

func TestSanitizeBranchName_EmptyInput(t *testing.T) {
	got := SanitizeBranchName("")
	if got != "" {
		t.Errorf("SanitizeBranchName(%q) = %q, want empty string", "", got)
	}
}
