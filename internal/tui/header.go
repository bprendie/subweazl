package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

func (m Model) appHeader(width int) string {
	if m.canShowLogo(width) && !m.input.Focused() {
		return renderLogo(logo, width)
	}
	left := m.styles.header.Render("Subweazl")
	section := lipgloss.NewStyle().Foreground(crushMint).Render(strings.ToUpper(m.list.Title))
	right := m.serverLabel()
	if m.mode == modeLocal {
		right = "local: " + m.localLabel()
	}
	right = m.styles.help.Render(ansi.Truncate(right, max(8, width/3), "..."))
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

func (m Model) canShowLogo(width int) bool {
	return width >= maxLineWidth(logo) && m.height >= 34
}
