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
	if m.mode == modeSources {
		return renderHelpMenu(width, [][]helpAction{
			{{"1", "subsonic", crushMint}, {"2", "local", crushGold}, {"enter", "open", crushMint}},
			{{"left", "back", muted}, {"q", "quit", muted}},
		})
	}
	if m.mode == modeLocal {
		return renderHelpMenu(width, [][]helpAction{
			{{"left", "sources", muted}, {"q", "quit", muted}},
		})
	}
	return renderHelpMenu(width, [][]helpAction{
		{{"1", "newest", crushGold}, {"2", "playlists", crushPink}, {"3", "random", crushMint}, {"/", "search", crushPurple}},
		{{"enter", "open/play", crushMint}, {"r", "station", crushPink}, {"ctrl+r", "rename", crushPurple}},
		{{"space", "pause", crushMint}, {"s", "stop", crushGold}},
		{{"left", "back", muted}, {"esc", "back", muted}, {"q", "quit", muted}},
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
