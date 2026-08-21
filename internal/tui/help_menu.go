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
		{{"tab", "pane", crushPurple}, {"enter", "open/play", crushMint}, {"/", "search", crushPurple}, {"space", "play/pause", crushMint}, {"p/n", "prev/next", crushGold}, {"m", "mode", crushPink}, {"g", "queue", crushGold}, {"G", "AI Mix", crushPink}, {"M", "Mood", crushMint}, {"v", "copy playlist", crushGold}, {"?", "help", muted}, {"q", "quit", muted}},
	})
}

func (m Model) fullHelpPopup(width, height int) string {
	panelWidth := clampInt(width-4, 44, 92)
	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Foreground(crushGold).Bold(true).Render("help"))
	b.WriteString("\n")
	b.WriteString(m.styles.help.Render("press ? or esc to close"))
	b.WriteString("\n\n")
	b.WriteString(renderHelpSection("Navigation", []helpAction{
		{"tab", "switch sidebar/list", crushPurple},
		{"up/down", "select sidebar item", crushMint},
		{"h", "home", crushGold},
		{"1", "newest albums", crushGold},
		{"2", "playlists", crushPink},
		{"3", "random albums", crushMint},
		{"4", "queue", crushGold},
		{"5", "private playlists", crushPink},
		{"/", "search", crushPurple},
		{"left/esc", "back", muted},
	}, panelWidth-4))
	b.WriteString("\n\n")
	b.WriteString(renderHelpSection("Playback", []helpAction{
		{"enter", "open or play", crushMint},
		{"space", "play/pause", crushMint},
		{"n", "next track", crushMint},
		{"p", "previous track", crushGold},
		{"m", "playback mode", crushPink},
		{"s", "stop", crushGold},
		{"a", "enqueue selected", crushPink},
	}, panelWidth-4))
	b.WriteString("\n\n")
	b.WriteString(renderHelpSection("Queue & Curation", []helpAction{
		{"g", "generate deterministic queue", crushGold},
		{"G", "generate AI Mix", crushPink},
		{"M", "overwrite server Mood from playing/queued track", crushMint},
		{"ctrl+l", "llm setup", crushPurple},
		{"y", "sync encrypted cache", crushMint},
		{"w", "save queue", crushGold},
		{"x", "remove queue row", crushPurple},
		{"c", "clear queue", crushGold},
		{"u/d", "move queue row", crushMint},
	}, panelWidth-4))
	b.WriteString("\n\n")
	b.WriteString(renderHelpSection("Playlists", []helpAction{
		{"v", "copy server ↔ vault", crushMint},
		{"r", "create station", crushPink},
		{"ctrl+r", "rename", crushPurple},
		{"del", "delete playlist", crushPink},
		{"q", "quit", muted},
	}, panelWidth-4))
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, m.styles.active.Width(panelWidth).Render(b.String()))
}

func renderHelpSection(title string, actions []helpAction, width int) string {
	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Foreground(crushMint).Bold(true).Render(title))
	b.WriteString("\n")
	b.WriteString(wrapStyled(renderHelpActions(actions), width))
	return b.String()
}

func renderHelpActions(actions []helpAction) string {
	parts := make([]string, 0, len(actions))
	for _, action := range actions {
		parts = append(parts, renderHelpAction(action))
	}
	return strings.Join(parts, "  ")
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
