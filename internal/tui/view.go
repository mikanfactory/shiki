package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"

	"worktree-ui/internal/model"
)

func (m Model) View() string {
	if m.quitting {
		return ""
	}

	if m.selected != "" {
		return m.selected
	}

	if m.addingRepo {
		return renderAddRepoView(m)
	}

	if m.loading {
		return titleStyle.Render("Workspaces") + "\n\n  Loading..."
	}

	if m.err != nil {
		return titleStyle.Render("Workspaces") + "\n\n  Error: " + m.err.Error()
	}

	var b strings.Builder

	b.WriteString(titleStyle.Render("Workspaces"))
	b.WriteString("\n")

	for i, item := range m.items {
		isSelected := i == m.cursor
		line := renderItem(item, isSelected, m.sidebarWidth)
		if item.Selectable {
			line = zone.Mark(ZoneID(i), line)
		}
		b.WriteString(line)
		b.WriteString("\n")
	}

	b.WriteString(helpStyle.Render("q: quit  ↑↓/jk: move  enter/click: select"))

	return zone.Scan(b.String())
}

func renderItem(item model.NavigableItem, selected bool, width int) string {
	switch item.Kind {
	case model.ItemKindGroupHeader:
		return groupHeaderStyle.Render(item.Label)

	case model.ItemKindWorktree:
		return renderWorktree(item, selected, width)

	case model.ItemKindAddWorktree, model.ItemKindAddRepo, model.ItemKindSettings:
		return renderAction(item, selected)

	default:
		return item.Label
	}
}

func renderWorktree(item model.NavigableItem, selected bool, width int) string {
	agentIcon := AgentIcon(item.AgentStatus)
	branchName := item.Label

	// Use inline styles to avoid PaddingLeft double-application when
	// inserting agent icon between indent and branch name.
	selectedBranchStyle := lipgloss.NewStyle().Foreground(colorAccent).Bold(true)
	normalBranchStyle := lipgloss.NewStyle().Foreground(colorFg)

	if selected {
		prefix := " > " + agentIcon // 1-space indent + cursor + icon
		maxBranchLen := width - lipgloss.Width(prefix)
		if maxBranchLen > 0 && lipgloss.Width(branchName) > maxBranchLen {
			branchName = truncate(branchName, maxBranchLen)
		}
		return selectedBranchStyle.Render(" > ") + agentIcon + selectedBranchStyle.Render(branchName)
	}

	prefix := "   " + agentIcon // 3-space indent + icon
	maxBranchLen := width - lipgloss.Width(prefix)
	if maxBranchLen > 0 && lipgloss.Width(branchName) > maxBranchLen {
		branchName = truncate(branchName, maxBranchLen)
	}

	return "   " + agentIcon + normalBranchStyle.Render(branchName)
}

func renderAction(item model.NavigableItem, selected bool) string {
	if selected {
		return actionSelectedStyle.Render(fmt.Sprintf("> %s", item.Label))
	}
	return actionStyle.Render(fmt.Sprintf("  %s", item.Label))
}

func renderAddRepoView(m Model) string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Add Repository"))
	b.WriteString("\n\n")

	if m.loading {
		b.WriteString("  Validating...")
		return b.String()
	}

	b.WriteString("  Enter the path to a git repository:\n\n")
	b.WriteString("  ")
	b.WriteString(m.textInput.View())
	b.WriteString("\n")

	if m.err != nil {
		b.WriteString("\n")
		b.WriteString(errorStyle.Render(fmt.Sprintf("  Error: %s", m.err.Error())))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("enter: confirm  esc: cancel"))

	return b.String()
}

func truncate(s string, maxLen int) string {
	if maxLen <= 3 {
		return s[:maxLen]
	}
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen-1]) + "…"
}
