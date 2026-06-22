package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

func (m Model) setupView(width int) string {
	var b strings.Builder
	title := lipgloss.NewStyle().Foreground(crushGold).Bold(true).Render("connections")
	b.WriteString(title)
	b.WriteString("\n")
	b.WriteString(m.styles.help.Render("[tab] next  [enter] test/save  [ctrl+s] save  [q] quit"))
	b.WriteString("\n\n")
	b.WriteString(m.styles.panel.Width(width).Render(m.setupPanel(width - 4)))
	b.WriteString("\n")
	b.WriteString(m.statusLine())
	if m.err != "" {
		b.WriteString("\n" + m.styles.error.Render(ansi.Wordwrap(m.err, max(20, width-2), " /_-")))
	}
	return b.String()
}

func (m Model) setupPanel(width int) string {
	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Foreground(crushMint).Render("Subsonic / Navidrome"))
	b.WriteString("\n")
	for i := setupServer; i <= setupPassword; i++ {
		b.WriteString(m.setup[i].View())
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().Foreground(crushPink).Render("Local music folders"))
	b.WriteString("\n")
	b.WriteString(m.setup[setupFolders].View())
	b.WriteString("\n")
	b.WriteString(m.styles.help.Render(ansi.Truncate("comma-separated paths are stored for local library support", max(24, width), "...")))
	return b.String()
}
