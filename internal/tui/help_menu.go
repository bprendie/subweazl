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
		{{"tab", "pane", secondary}, {"enter", "open/play", success}, {"/", "search", secondary}, {"space", "play/pause", success}, {"p/n", "prev/next", warning}, {"m", "mode", accent}, {"g", "queue", warning}, {"G", "AI Mix", accent}, {"M", "Mood", success}, {"v", "copy playlist", warning}, {"?", "help", muted}, {"q", "quit", muted}},
	})
}

func (m Model) fullHelpPopup(width, height int) string {
	panelWidth := clampInt(width-4, 44, 92)
	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Foreground(warning).Bold(true).Render("help"))
	b.WriteString("\n")
	b.WriteString(m.styles.help.Render("press ? or esc to close"))
	b.WriteString("\n\n")
	b.WriteString(renderHelpSection("Navigation", []helpAction{
		{"tab", "switch sidebar/list", secondary},
		{"up/down", "select sidebar item", success},
		{"h", "home", warning},
		{"1", "newest albums", warning},
		{"2", "playlists", accent},
		{"3", "random albums", success},
		{"4", "queue", warning},
		{"5", "private playlists", accent},
		{"/", "search", secondary},
		{"left/esc", "back", muted},
	}, panelWidth-4))
	b.WriteString("\n\n")
	b.WriteString(renderHelpSection("Playback", []helpAction{
		{"enter", "open or play", success},
		{"space", "play/pause", success},
		{"n", "next track", success},
		{"p", "previous track", warning},
		{"m", "playback mode", accent},
		{"s", "stop", warning},
		{"a", "enqueue selected", accent},
	}, panelWidth-4))
	b.WriteString("\n\n")
	b.WriteString(renderHelpSection("Queue & Curation", []helpAction{
		{"g", "generate deterministic queue", warning},
		{"G", "generate AI Mix", accent},
		{"M", "overwrite server Mood from playing/queued track", success},
		{"ctrl+l", "llm setup", secondary},
		{"y", "sync encrypted cache", success},
		{"w", "save queue", warning},
		{"x", "remove queue row", secondary},
		{"c", "clear queue", warning},
		{"u/d", "move queue row", success},
	}, panelWidth-4))
	b.WriteString("\n\n")
	b.WriteString(renderHelpSection("Playlists", []helpAction{
		{"v", "copy server ↔ vault", success},
		{"r", "create station", accent},
		{"ctrl+r", "rename", secondary},
		{"del", "delete playlist", accent},
		{"q", "quit", muted},
	}, panelWidth-4))
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, m.styles.active.Width(panelWidth).Render(b.String()))
}

func renderHelpSection(title string, actions []helpAction, width int) string {
	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Foreground(success).Bold(true).Render(title))
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
	separator := lipgloss.NewStyle().Foreground(secondary).Faint(true).Render("  ╱  ")
	return wrapStyled(strings.Join(parts, separator), width)
}

func renderHelpAction(action helpAction) string {
	key := lipgloss.NewStyle().Foreground(warning).Bold(true).Render("[" + action.key + "]")
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
