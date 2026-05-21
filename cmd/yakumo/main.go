package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"

	"github.com/mikanfactory/yakumo/internal/claude"
	"github.com/mikanfactory/yakumo/internal/config"
	"github.com/mikanfactory/yakumo/internal/diffui"
	"github.com/mikanfactory/yakumo/internal/git"
	"github.com/mikanfactory/yakumo/internal/github"
	"github.com/mikanfactory/yakumo/internal/model"
	"github.com/mikanfactory/yakumo/internal/setupspinner"
	"github.com/mikanfactory/yakumo/internal/tmux"
	"github.com/mikanfactory/yakumo/internal/tui"
)

const usage = `Usage: yakumo [command]

Commands:
  (default)         Launch worktree UI
  diff-ui           Launch diff/PR review UI
  swap-center       Swap center pane with background
  swap-right-below  Swap right-below pane with background

Flags (worktree UI only):
  --config <path>   Path to config file
`

func main() {
	if len(os.Args) < 2 {
		runWorktreeUI("")
		return
	}

	switch os.Args[1] {
	case "diff-ui":
		runDiffUI()
	case "swap-center":
		runSwapCenter()
	case "swap-right-below":
		runSwapRightBelow()
	case "--diff":
		fmt.Fprintln(os.Stderr, "Warning: --diff is deprecated, use 'yakumo diff-ui' instead")
		runDiffUI()
	case "--help", "-h", "help":
		fmt.Print(usage)
	default:
		fs := flag.NewFlagSet("yakumo", flag.ExitOnError)
		fs.Usage = func() { fmt.Print(usage) }
		configPath := fs.String("config", "", "path to config file")
		fs.Parse(os.Args[1:])
		runWorktreeUI(*configPath)
	}
}

func runDiffUI() {
	zone.NewGlobal()

	dir, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	gitRunner := git.OSCommandRunner{}
	if _, err := exec.LookPath("gh"); err != nil {
		fmt.Fprintln(os.Stderr, "error: gh CLI is required for diff-ui")
		os.Exit(1)
	}
	ghRunner := github.OSRunner{}

	baseRef := resolveBaseRef()
	p := tea.NewProgram(
		diffui.NewModel(dir, gitRunner, ghRunner, baseRef),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func setupDebugLog() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	logPath := filepath.Join(home, ".config", "yakumo", "debug.log")
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return
	}
	log.SetOutput(f)
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)
}

func runWorktreeUI(configPath string) {
	setupDebugLog()
	zone.NewGlobal()

	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	resolvedConfigPath, err := config.ResolveConfigPath(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	runner := git.OSCommandRunner{}

	var tmuxRunner tmux.Runner
	if tmux.IsInsideTmux() {
		tmuxRunner = tmux.OSRunner{}
		if err := tmux.EnsureMainSession(tmuxRunner); err != nil {
			log.Printf("[main] EnsureMainSession failed (non-fatal): %v", err)
		}
	}

	var ghRunner github.Runner
	if _, err := exec.LookPath("gh"); err == nil {
		ghRunner = github.OSRunner{}
	}

	m := tui.NewModel(cfg, runner, resolvedConfigPath, tmuxRunner, ghRunner)

	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	result, err := p.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	finalModel, ok := result.(tui.Model)
	if !ok || finalModel.Selected() == "" {
		return
	}

	selected := finalModel.Selected()

	if tmux.IsInsideTmux() {
		spinnerModel := setupspinner.New("Setting up workspace...")
		spinnerProg := tea.NewProgram(spinnerModel)

		go runSessionSetup(spinnerProg, cfg, finalModel, selected)

		result, err := spinnerProg.Run()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		if m, ok := result.(setupspinner.Model); ok {
			if err := m.Result(); err != nil {
				fmt.Fprintf(os.Stderr, "tmux error: %v\n", err)
				os.Exit(1)
			}
		}

		return
	}

	fmt.Print(selected)
}

func runSessionSetup(prog *tea.Program, cfg model.Config, finalModel tui.Model, selected string) {
	tmuxRunner := tmux.OSRunner{}
	gitRunner := git.OSCommandRunner{}
	getBranch := tmux.BranchGetter(func(worktreePath string) (string, error) {
		out, err := gitRunner.Run(worktreePath, "symbolic-ref", "--short", "HEAD")
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(out), nil
	})

	prog.Send(setupspinner.StatusMsg("Creating session..."))
	repo := findRepoByPath(cfg, finalModel.SelectedRepoPath())
	layout, err := tmux.SelectWorktreeSession(tmuxRunner, selected, repo.StartupCommand, getBranch)
	if err != nil {
		prog.Send(setupspinner.DoneMsg{Err: fmt.Errorf("tmux error: %w", err)})
		return
	}

	// Run additional commands only for newly created sessions
	if layout.BottomRight1.PaneID != "" {
		// Launch diff-ui in top-right pane
		prog.Send(setupspinner.StatusMsg("Launching diff-ui..."))
		if diffCmd := diffUICommand(); diffCmd != "" {
			if err := tmux.SendKeys(tmuxRunner, layout.TopRight1.PaneID, diffCmd); err != nil {
				log.Printf("[setup] diff-ui launch error: %v", err)
			}
		}

		// Ensure claude trust and launch claude CLI in center pane
		prog.Send(setupspinner.StatusMsg("Launching Claude..."))
		if _, err := exec.LookPath("claude"); err == nil {
			if home, err := os.UserHomeDir(); err == nil {
				configPath := filepath.Join(home, ".claude.json")
				if trustErr := claude.EnsureDirectoryTrusted(configPath, selected); trustErr != nil {
					log.Printf("[setup] claude trust warning: %v", trustErr)
				}
			}
			if err := tmux.SendKeys(tmuxRunner, layout.Center1.PaneID, "claude"); err != nil {
				log.Printf("[setup] claude launch error: %v", err)
			}
		}

		// Focus center pane after all commands are sent
		prog.Send(setupspinner.StatusMsg("Focusing workspace..."))
		if err := tmux.SelectPane(tmuxRunner, layout.Center1.PaneID); err != nil {
			log.Printf("[setup] select pane error: %v", err)
		}
	}

	prog.Send(setupspinner.DoneMsg{})
}

func runSwapCenter() {
	if !tmux.IsInsideTmux() {
		fmt.Fprintln(os.Stderr, "error: swap-center requires running inside tmux")
		os.Exit(1)
	}
	runner := tmux.OSRunner{}
	if err := tmux.SwapCenter(runner); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func runSwapRightBelow() {
	if !tmux.IsInsideTmux() {
		fmt.Fprintln(os.Stderr, "error: swap-right-below requires running inside tmux")
		os.Exit(1)
	}
	runner := tmux.OSRunner{}
	if err := tmux.SwapRightBelow(runner); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func diffUICommand() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	return exe + " diff-ui"
}

func resolveBaseRef() string {
	baseRef := config.DefaultBaseRef
	path, err := config.ResolveConfigPath("")
	if err != nil {
		return baseRef
	}
	cfg, err := config.LoadFromFile(path)
	if err != nil {
		return baseRef
	}
	if cfg.DefaultBaseRef != "" {
		baseRef = cfg.DefaultBaseRef
	}
	return baseRef
}

func findRepoByPath(cfg model.Config, repoPath string) model.RepositoryDef {
	for _, repo := range cfg.Repositories {
		if repo.Path == repoPath {
			return repo
		}
	}
	return model.RepositoryDef{}
}
