package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type helpAction struct {
	key   string
	label string
	color lipgloss.Color
}

func (m Model) helpMenu(width int) string {
	return renderHelpMenu(width, [][]helpAction{
		{{"h", "home", crushGold}, {"1", "newest", crushGold}, {"2", "playlists", crushPink}, {"3", "random", crushMint}, {"4", "queue", crushGold}, {"5", "private", crushPink}, {"y", "sync", crushMint}, {"/", "search", crushPurple}},
		{{"enter", "open/play", crushMint}, {"r", "station", crushPink}, {"ctrl+r", "rename", crushPurple}},
		{{"n", "next", crushMint}, {"p", "prev", crushGold}, {"a", "enqueue", crushPink}, {"w", "save queue", crushGold}, {"space", "pause", crushMint}, {"s", "stop", crushGold}},
		{{"x", "remove", crushPurple}, {"del", "delete", crushPink}, {"c", "clear", crushGold}, {"u/d", "move", crushMint}, {"left", "back", muted}, {"esc", "back", muted}, {"q", "quit", muted}},
	})
}

func renderHelpMenu(width int, groups [][]helpAction) string {
	var parts []string
	for _, group := range groups {
		var actions []string
		for _, action := range group {
			actions = append(actions, renderHelpAction(action))
		}
		parts = append(parts, strings.Join(actions, "  "))
	}
	separator := lipgloss.NewStyle().Foreground(crushPurple).Faint(true).Render("  ╱  ")
	return wrapStyled(strings.Join(parts, separator), width)
}

func renderHelpAction(action helpAction) string {
	key := lipgloss.NewStyle().Foreground(crushGold).Bold(true).Render("[" + action.key + "]")
	label := lipgloss.NewStyle().Foreground(action.color).Render(" " + action.label)
	return key + label
}

func wrapStyled(s string, width int) string {
	if lipgloss.Width(s) <= width {
		return s
	}
	var out strings.Builder
	lineWidth := 0
	for _, part := range strings.Split(s, "  ") {
		partWidth := lipgloss.Width(part)
		sep := 0
		if lineWidth > 0 {
			sep = 2
		}
		if lineWidth > 0 && lineWidth+sep+partWidth > width {
			out.WriteByte('\n')
			lineWidth = 0
			sep = 0
		}
		if sep > 0 {
			out.WriteString("  ")
			lineWidth += sep
		}
		out.WriteString(part)
		lineWidth += partWidth
	}
	return out.String()
}
