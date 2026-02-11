package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"worktree-ui/internal/model"
)

func TestLoadFromFile(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	content := `sidebar_width: 35
repositories:
  - name: myrepo
    path: /home/user/myrepo
  - name: other
    path: /home/user/other
`
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadFromFile(cfgPath)
	if err != nil {
		t.Fatalf("LoadFromFile failed: %v", err)
	}

	if cfg.SidebarWidth != 35 {
		t.Errorf("SidebarWidth = %d, want 35", cfg.SidebarWidth)
	}
	if len(cfg.Repositories) != 2 {
		t.Fatalf("len(Repositories) = %d, want 2", len(cfg.Repositories))
	}
	if cfg.Repositories[0].Name != "myrepo" {
		t.Errorf("Repositories[0].Name = %q, want %q", cfg.Repositories[0].Name, "myrepo")
	}
	if cfg.Repositories[1].Path != "/home/user/other" {
		t.Errorf("Repositories[1].Path = %q, want %q", cfg.Repositories[1].Path, "/home/user/other")
	}
}

func TestLoadFromFile_DefaultSidebarWidth(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	content := `repositories:
  - name: repo1
    path: /tmp/repo1
`
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadFromFile(cfgPath)
	if err != nil {
		t.Fatalf("LoadFromFile failed: %v", err)
	}

	if cfg.SidebarWidth != DefaultSidebarWidth {
		t.Errorf("SidebarWidth = %d, want default %d", cfg.SidebarWidth, DefaultSidebarWidth)
	}
}

func TestLoadFromFile_NotFound(t *testing.T) {
	_, err := LoadFromFile("/nonexistent/config.yaml")
	if err == nil {
		t.Error("expected error for nonexistent file, got nil")
	}
}

func TestLoadFromFile_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	if err := os.WriteFile(cfgPath, []byte("{{invalid yaml"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadFromFile(cfgPath)
	if err == nil {
		t.Error("expected error for invalid YAML, got nil")
	}
}

func TestLoadFromFile_EmptyRepositories(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	content := `sidebar_width: 30
repositories: []
`
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadFromFile(cfgPath)
	if err == nil {
		t.Error("expected error for empty repositories, got nil")
	}
}

func TestDetectGitRoot_InRepo(t *testing.T) {
	name, root, err := detectGitRoot()
	if err != nil {
		t.Fatalf("detectGitRoot failed in git repo: %v", err)
	}
	if name == "" {
		t.Error("expected non-empty repo name")
	}
	if root == "" {
		t.Error("expected non-empty repo root")
	}
	if filepath.Base(root) != name {
		t.Errorf("name = %q, want %q (basename of root %q)", name, filepath.Base(root), root)
	}
}

func TestDetectGitRoot_NotInRepo(t *testing.T) {
	original := detectGitRootFn
	detectGitRootFn = func() (string, string, error) {
		return "", "", fmt.Errorf("not inside a git repository")
	}
	t.Cleanup(func() { detectGitRootFn = original })

	_, _, err := detectGitRootFn()
	if err == nil {
		t.Error("expected error outside git repo, got nil")
	}
}

func TestEnsureDefaultConfig_CreatesFile(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	original := detectGitRootFn
	detectGitRootFn = func() (string, string, error) {
		return "my-repo", "/home/user/my-repo", nil
	}
	t.Cleanup(func() { detectGitRootFn = original })

	path, created, err := EnsureDefaultConfig()
	if err != nil {
		t.Fatalf("EnsureDefaultConfig failed: %v", err)
	}
	if !created {
		t.Error("expected created=true")
	}

	wantPath := filepath.Join(tmpHome, ".config", "shiki", "config.yaml")
	if path != wantPath {
		t.Errorf("path = %q, want %q", path, wantPath)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading created config: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "my-repo") {
		t.Errorf("config should contain repo name, got:\n%s", content)
	}
	if !strings.Contains(content, "/home/user/my-repo") {
		t.Errorf("config should contain repo path, got:\n%s", content)
	}
}

func TestEnsureDefaultConfig_AlreadyExists(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	configDir := filepath.Join(tmpHome, ".config", "shiki")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(configDir, "config.yaml")
	existingContent := "existing content"
	if err := os.WriteFile(configPath, []byte(existingContent), 0o644); err != nil {
		t.Fatal(err)
	}

	path, created, err := EnsureDefaultConfig()
	if err != nil {
		t.Fatalf("EnsureDefaultConfig failed: %v", err)
	}
	if created {
		t.Error("expected created=false for existing file")
	}
	if path != configPath {
		t.Errorf("path = %q, want %q", path, configPath)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != existingContent {
		t.Errorf("existing file was modified: got %q, want %q", string(data), existingContent)
	}
}

func TestEnsureDefaultConfig_NotInGitRepo(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	original := detectGitRootFn
	detectGitRootFn = func() (string, string, error) {
		return "", "", fmt.Errorf("not inside a git repository")
	}
	t.Cleanup(func() { detectGitRootFn = original })

	path, created, err := EnsureDefaultConfig()
	if err != nil {
		t.Fatalf("EnsureDefaultConfig failed: %v", err)
	}
	if !created {
		t.Error("expected created=true")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading created config: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "#") {
		t.Errorf("config for non-git-repo should be commented out, got:\n%s", content)
	}
}

func TestLoad_AutoCreatesConfig(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	original := detectGitRootFn
	detectGitRootFn = func() (string, string, error) {
		return "my-repo", "/tmp/my-repo", nil
	}
	t.Cleanup(func() { detectGitRootFn = original })

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load auto-create failed: %v", err)
	}
	if len(cfg.Repositories) != 1 {
		t.Fatalf("expected 1 repository, got %d", len(cfg.Repositories))
	}
	if cfg.Repositories[0].Name != "my-repo" {
		t.Errorf("repo name = %q, want %q", cfg.Repositories[0].Name, "my-repo")
	}
	if cfg.Repositories[0].Path != "/tmp/my-repo" {
		t.Errorf("repo path = %q, want %q", cfg.Repositories[0].Path, "/tmp/my-repo")
	}
}

func TestLoad_AutoCreatesConfig_NoGitRepo(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	original := detectGitRootFn
	detectGitRootFn = func() (string, string, error) {
		return "", "", fmt.Errorf("not inside a git repository")
	}
	t.Cleanup(func() { detectGitRootFn = original })

	_, err := Load("")
	if err == nil {
		t.Fatal("expected error when no git repo and no config")
	}
	if !strings.Contains(err.Error(), "edit the config") {
		t.Errorf("error should guide user to edit config, got: %v", err)
	}
}

func TestResolveConfigPath(t *testing.T) {
	t.Run("explicit path exists", func(t *testing.T) {
		dir := t.TempDir()
		flagPath := filepath.Join(dir, "config.yaml")
		if err := os.WriteFile(flagPath, []byte("repositories:\n  - name: x\n    path: /x"), 0o644); err != nil {
			t.Fatal(err)
		}

		result, err := ResolveConfigPath(flagPath)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if result != flagPath {
			t.Errorf("result = %q, want %q", result, flagPath)
		}
	})

	t.Run("explicit path not exists", func(t *testing.T) {
		_, err := ResolveConfigPath("/nonexistent/path.yaml")
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("default path exists", func(t *testing.T) {
		tmpHome := t.TempDir()
		t.Setenv("HOME", tmpHome)

		configDir := filepath.Join(tmpHome, ".config", "shiki")
		if err := os.MkdirAll(configDir, 0o755); err != nil {
			t.Fatal(err)
		}
		configPath := filepath.Join(configDir, "config.yaml")
		if err := os.WriteFile(configPath, []byte("repositories:\n  - name: x\n    path: /x"), 0o644); err != nil {
			t.Fatal(err)
		}

		result, err := ResolveConfigPath("")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if result != configPath {
			t.Errorf("result = %q, want %q", result, configPath)
		}
	})

	t.Run("default path not exists", func(t *testing.T) {
		tmpHome := t.TempDir()
		t.Setenv("HOME", tmpHome)

		_, err := ResolveConfigPath("")
		if err == nil {
			t.Error("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "default config not found") {
			t.Errorf("unexpected error message: %v", err)
		}
	})
}

func TestLoad(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	content := `sidebar_width: 28
repositories:
  - name: testrepo
    path: /tmp/testrepo
`
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	want := model.Config{
		SidebarWidth: 28,
		Repositories: []model.RepositoryDef{
			{Name: "testrepo", Path: "/tmp/testrepo"},
		},
	}

	if cfg.SidebarWidth != want.SidebarWidth {
		t.Errorf("SidebarWidth = %d, want %d", cfg.SidebarWidth, want.SidebarWidth)
	}
	if len(cfg.Repositories) != len(want.Repositories) {
		t.Fatalf("len(Repositories) = %d, want %d", len(cfg.Repositories), len(want.Repositories))
	}
	if cfg.Repositories[0] != want.Repositories[0] {
		t.Errorf("Repositories[0] = %+v, want %+v", cfg.Repositories[0], want.Repositories[0])
	}
}
