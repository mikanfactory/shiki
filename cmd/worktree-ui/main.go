package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Worktree struct {
	Path   string
	Branch string
}

type Model struct {
	worktrees []Worktree
	cursor    int
	quitting  bool
}

func initialModel() Model {
	return Model{
		worktrees: []Worktree{
			{Path: "/repo/main", Branch: "main"},
			{Path: "/repo/feature-x", Branch: "feature-x"},
		},
		cursor: 0,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch msg.String() {

		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(m.worktrees)-1 {
				m.cursor++
			}
		}
	}

	return m, nil
}

func (m Model) View() string {
	if m.quitting {
		return ""
	}

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205"))

	cursorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("212"))

	var s string
	s += titleStyle.Render(" Worktrees\n\n")

	for i, wt := range m.worktrees {
		cursor := "  "
		if i == m.cursor {
			cursor = cursorStyle.Render("> ")
		}
		s += fmt.Sprintf("%s%s (%s)\n", cursor, wt.Path, wt.Branch)
	}

	s += "\n q: quit  ↑↓/jk: move"

	return s
}

func main() {
	p := tea.NewProgram(initialModel())
	if err := p.Start(); err != nil {
		fmt.Println("error:", err)
		os.Exit(1)
	}
}
