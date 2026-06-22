package tui

import (
	"fmt"
	"net/url"
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
		{"h", "Home", modeHome},
		{"1", "Newest albums", modeNewest},
		{"2", "Playlists", modePlaylists},
		{"3", "Random albums", modeRandomAlbums},
		{"4", "Queue", modeQueue},
		{"5", "Private playlists", modePrivatePlaylists},
		{"y", "Sync cache", modeHome},
		{"g", "Generate queue", modeQueue},
		{"G", "LLM curate", modeQueue},
		{"/", "Song search", modeSearch},
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
	b.WriteString("\n\n")
	b.WriteString(m.railHeader("CACHE"))
	b.WriteString("\n")
	b.WriteString(m.styles.help.Render(ansi.Truncate(m.cacheLabel(), max(8, width-4), "...")))
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
	case modeQueue:
		return m.mode == modeQueue
	case modePrivatePlaylists:
		return m.mode == modePrivatePlaylists
	default:
		return m.mode == entry
	}
}

func (m Model) serverLabel() string {
	u, err := url.Parse(m.cfg.Server)
	if err != nil || u.Host == "" {
		return "not connected"
	}
	return u.Host
}

func (m Model) cacheLabel() string {
	if m.cacheStatus.TrackCount <= 0 {
		return "not synced"
	}
	return fmt.Sprintf("%d tracks", m.cacheStatus.TrackCount)
}
