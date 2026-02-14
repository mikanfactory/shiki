package git

import (
	"fmt"
	"testing"
)

func TestGetUserName(t *testing.T) {
	runner := FakeCommandRunner{
		Outputs: map[string]string{
			"/repo:[config user.name]": "mikanfactory\n",
		},
	}

	name, err := GetUserName(runner, "/repo")
	if err != nil {
		t.Fatalf("GetUserName failed: %v", err)
	}
	if name != "mikanfactory" {
		t.Errorf("GetUserName() = %q, want %q", name, "mikanfactory")
	}
}

func TestGetUserName_Trimmed(t *testing.T) {
	runner := FakeCommandRunner{
		Outputs: map[string]string{
			"/repo:[config user.name]": "  some-user  \n",
		},
	}

	name, err := GetUserName(runner, "/repo")
	if err != nil {
		t.Fatalf("GetUserName failed: %v", err)
	}
	if name != "some-user" {
		t.Errorf("GetUserName() = %q, want %q", name, "some-user")
	}
}

func TestGetUserName_Empty(t *testing.T) {
	runner := FakeCommandRunner{
		Outputs: map[string]string{
			"/repo:[config user.name]": "\n",
		},
	}

	_, err := GetUserName(runner, "/repo")
	if err == nil {
		t.Error("expected error for empty user.name")
	}
}

func TestGetUserName_CommandError(t *testing.T) {
	runner := FakeCommandRunner{
		Errors: map[string]error{
			"/repo:[config user.name]": fmt.Errorf("config not found"),
		},
	}

	_, err := GetUserName(runner, "/repo")
	if err == nil {
		t.Error("expected error when git command fails")
	}
}
