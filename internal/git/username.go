package git

import (
	"fmt"
	"strings"
)

// GetUserName returns the git user name from the repository's config.
func GetUserName(runner CommandRunner, repoPath string) (string, error) {
	out, err := runner.Run(repoPath, "config", "user.name")
	if err != nil {
		return "", fmt.Errorf("getting git user.name: %w", err)
	}

	name := strings.TrimSpace(out)
	if name == "" {
		return "", fmt.Errorf("git user.name is empty")
	}

	return name, nil
}
