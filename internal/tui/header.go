package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

func (m Model) appHeader(width int) string {
	left := m.styles.header.Render("Subweazl")
	section := lipgloss.NewStyle().Foreground(crushMint).Render(strings.ToUpper(m.list.Title))
	right := m.styles.help.Render(ansi.Truncate(m.serverLabel(), max(8, width/3), "..."))
	gap := width - lipgloss.Width(left) - lipgloss.Width(section) - lipgloss.Width(right) - 4
	if gap < 1 {
		gap = 1
	}
	line := left + "  " + section + strings.Repeat(" ", gap) + right
	if m.input.Focused() {
		line = m.input.View()
	}
	style := m.styles.panel
	if m.input.Focused() {
		style = m.styles.active
	}
	return style.Width(width).Height(1).Render(line)
}
