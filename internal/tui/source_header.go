package tui

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

func (m Model) sourceHeader(width int) string {
	if m.mode == modeLocal {
		local := m.sourceBadge("local", m.localLabel(), crushGold, true)
		section := m.sectionBadge(width)
		return wrapStyled(strings.Join([]string{local, section, m.switchHint("b", "subsonic")}, "  "), width)
	}
	if m.mode == modeSources {
		return wrapStyled(strings.Join([]string{
			m.sourceBadge("subsonic", m.serverLabel(), crushMint, true),
			m.sourceBadge("local", m.localLabel(), crushGold, false),
			m.sectionBadge(width),
		}, "  "), width)
	}
	streaming := m.sourceBadge("subsonic", m.serverLabel(), crushMint, true)
	section := m.sectionBadge(width)
	line := strings.Join([]string{streaming, section, m.switchHint("l", "local")}, "  ")
	return wrapStyled(line, width)
}

func (m Model) sourceBadge(label, value string, color lipgloss.Color, active bool) string {
	style := lipgloss.NewStyle().
		Foreground(color).
		Border(lipgloss.NormalBorder(), false, true, false, true).
		BorderForeground(crushPurple).
		Padding(0, 1)
	if !active {
		style = style.Foreground(muted).BorderForeground(border)
	}
	return style.Render(strings.ToUpper(label) + " " + value)
}

func (m Model) activeSource() string {
	switch m.mode {
	case modeLocal:
		return "local"
	case modeSources:
		return "sources"
	default:
		return "subsonic"
	}
}

func (m Model) switchHint(key, label string) string {
	return lipgloss.NewStyle().
		Foreground(muted).
		Render("[" + key + "] " + label)
}

func (m Model) sectionBadge(width int) string {
	title := m.list.Title
	if title == "" {
		title = "library"
	}
	value := ansi.Truncate(title, max(12, width/3), "...")
	return lipgloss.NewStyle().
		Foreground(crushPink).
		Bold(true).
		Render("section: " + value)
}

func (m Model) serverLabel() string {
	u, err := url.Parse(m.cfg.Server)
	if err != nil || u.Host == "" {
		return "not connected"
	}
	return u.Host
}

func (m Model) localLabel() string {
	count := len(m.cfg.LocalMusicFolders)
	if count == 0 {
		return "not configured"
	}
	if count == 1 {
		return "1 folder"
	}
	return fmt.Sprintf("%d folders", count)
}
