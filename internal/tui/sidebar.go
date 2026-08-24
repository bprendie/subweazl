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

var sidebarNavEntries = []navEntry{
	{"h", "Home", modeHome}, {"1", "Newest albums", modeNewest},
	{"2", "Playlists", modePlaylists}, {"3", "Random albums", modeRandomAlbums},
	{"4", "Queue", modeQueue}, {"5", "Private playlists", modePrivatePlaylists},
	{"y", "Sync cache", modeHome}, {"g", "Generate queue", modeQueue},
	{"G", "Generate AI Mix", modeQueue}, {"M", "Generate Mood", modePlaylists},
	{"ctrl+l", "LLM setup", modeLLMProvider},
	{"/", "Song search", modeSearch},
}

func (m Model) sidebar(width, height int) string {
	var b strings.Builder
	b.WriteString(m.railHeader("SUBSONIC"))
	b.WriteString("\n\n")
	for i, entry := range sidebarNavEntries {
		b.WriteString(m.navLine(entry, i, width-4))
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
	style := m.styles.panel
	if m.focus == focusSidebar {
		style = m.styles.active
	}
	return style.Width(width).Height(height).Render(b.String())
}

func (m Model) railHeader(label string) string {
	return lipgloss.NewStyle().
		Foreground(warning).
		Bold(true).
		Render(label)
}

func (m Model) navLine(entry navEntry, index, width int) string {
	key := lipgloss.NewStyle().Foreground(secondary).Bold(true).Render(entry.key)
	label := ansi.Truncate(entry.label, max(8, width-4), "...")
	style := m.styles.item
	prefix := "  "
	if m.focus == focusSidebar && m.sidebarIndex == index {
		style = m.styles.selected
		prefix = "> "
	} else if m.navEntryActive(entry.mode) {
		prefix = "• "
	}
	return prefix + key + " " + style.Render(label)
}

func (m *Model) focusSidebar() {
	m.focus = focusSidebar
	for i, entry := range sidebarNavEntries {
		if m.navEntryActive(entry.mode) {
			m.sidebarIndex = i
			return
		}
	}
	m.sidebarIndex = 0
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
	case modeLLMProvider:
		return m.isLLMConfigMode()
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
