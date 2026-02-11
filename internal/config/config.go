package config

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"worktree-ui/internal/model"
)

const DefaultSidebarWidth = 30

// LoadFromFile reads and parses a YAML config file.
func LoadFromFile(path string) (model.Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return model.Config{}, fmt.Errorf("reading config file: %w", err)
	}

	var cfg model.Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return model.Config{}, fmt.Errorf("parsing config file: %w", err)
	}

	if cfg.SidebarWidth == 0 {
		cfg.SidebarWidth = DefaultSidebarWidth
	}

	if len(cfg.Repositories) == 0 {
		return model.Config{}, fmt.Errorf("config must have at least one repository")
	}

	return cfg, nil
}

// ResolveConfigPath determines the config file path from flag or default location.
func ResolveConfigPath(flagPath string) (string, error) {
	if flagPath != "" {
		if _, err := os.Stat(flagPath); err != nil {
			return "", fmt.Errorf("config file not found: %s", flagPath)
		}
		return flagPath, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("getting home directory: %w", err)
	}

	defaultPath := filepath.Join(home, ".config", "shiki", "config.yaml")
	if _, err := os.Stat(defaultPath); err != nil {
		return "", fmt.Errorf("default config not found at %s: create it or use --config flag", defaultPath)
	}

	return defaultPath, nil
}

// detectGitRoot returns the repo name and root path of the current git repository.
func detectGitRoot() (string, string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err != nil {
		return "", "", fmt.Errorf("not inside a git repository")
	}
	root := strings.TrimSpace(string(out))
	name := filepath.Base(root)
	return name, root, nil
}

// detectGitRootFn is a testable function variable for detectGitRoot.
var detectGitRootFn = detectGitRoot

// EnsureDefaultConfig creates the default config file if it doesn't exist.
// Returns the config path, whether a file was created, and any error.
func EnsureDefaultConfig() (string, bool, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", false, fmt.Errorf("getting home directory: %w", err)
	}

	configDir := filepath.Join(home, ".config", "shiki")
	configPath := filepath.Join(configDir, "config.yaml")

	if _, err := os.Stat(configPath); err == nil {
		return configPath, false, nil
	}

	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return "", false, fmt.Errorf("creating config directory %s: %w", configDir, err)
	}

	name, root, gitErr := detectGitRootFn()

	var content string
	if gitErr == nil {
		content = fmt.Sprintf("sidebar_width: 30\n\nrepositories:\n  - name: %s\n    path: %s\n", name, root)
		fmt.Fprintf(os.Stderr, "Created default config at %s with repository %q (%s)\n", configPath, name, root)
	} else {
		content = "# sidebar_width: 30\n#\n# repositories:\n#   - name: my-repo\n#     path: /path/to/my-repo\n"
		fmt.Fprintf(os.Stderr, "Created config template at %s -- edit it to add your repositories\n", configPath)
	}

	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		return "", false, fmt.Errorf("writing default config: %w", err)
	}

	return configPath, true, nil
}

// Load resolves the config path and loads the config.
func Load(flagPath string) (model.Config, error) {
	if flagPath == "" {
		createdPath, created, err := EnsureDefaultConfig()
		if err != nil {
			return model.Config{}, fmt.Errorf("ensuring default config: %w", err)
		}
		if created {
			cfg, loadErr := LoadFromFile(createdPath)
			if loadErr != nil {
				return model.Config{}, fmt.Errorf(
					"edit the config at %s to add your repositories, then run again",
					createdPath,
				)
			}
			return cfg, nil
		}
	}

	path, err := ResolveConfigPath(flagPath)
	if err != nil {
		return model.Config{}, err
	}
	return LoadFromFile(path)
}
