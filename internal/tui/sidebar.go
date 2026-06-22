package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

type navEntry struct {
	key   string
	label string
	mode  mode
}

func (m Model) sidebar(width, height int) string {
	entries := []navEntry{
		{"1", "Newest albums", modeNewest},
		{"2", "Playlists", modePlaylists},
		{"3", "Random albums", modeRandomAlbums},
		{"/", "Song search", modeSearch},
		{"l", "Local folders", modeLocal},
	}
	var b strings.Builder
	b.WriteString(m.railHeader("SUBSONIC"))
	b.WriteString("\n\n")
	for _, entry := range entries {
		b.WriteString(m.navLine(entry, width-4))
		b.WriteByte('\n')
	}
	b.WriteString("\n")
	b.WriteString(m.railHeader("SERVER"))
	b.WriteString("\n")
	b.WriteString(m.styles.help.Render(ansi.Truncate(m.serverLabel(), max(8, width-4), "...")))
	if height > 15 {
		b.WriteString("\n\n")
		b.WriteString(m.railHeader("LOCAL"))
		b.WriteString("\n")
		b.WriteString(m.styles.help.Render(ansi.Truncate(m.localLabel(), max(8, width-4), "...")))
	}
	return m.styles.panel.Width(width).Height(height).Render(b.String())
}

func (m Model) railHeader(label string) string {
	return lipgloss.NewStyle().
		Foreground(crushGold).
		Bold(true).
		Render(label)
}

func (m Model) navLine(entry navEntry, width int) string {
	key := lipgloss.NewStyle().Foreground(crushPurple).Bold(true).Render(entry.key)
	label := ansi.Truncate(entry.label, max(8, width-4), "...")
	style := m.styles.item
	prefix := "  "
	if m.navEntryActive(entry.mode) {
		style = m.styles.selected
		prefix = "> "
	}
	return prefix + key + " " + style.Render(label)
}

func (m Model) navEntryActive(entry mode) bool {
	switch entry {
	case modeNewest:
		return m.mode == modeNewest || m.mode == modeLastPlayed
	case modeRandomAlbums:
		return m.mode == modeRandomAlbums
	case modePlaylists:
		return m.mode == modePlaylists || m.mode == modeStation
	case modeSearch:
		return m.mode == modeSearch
	case modeLocal:
		return m.mode == modeLocal
	default:
		return m.mode == entry
	}
}
