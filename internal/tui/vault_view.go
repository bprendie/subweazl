package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

func (m Model) vaultView(width int) string {
	var b strings.Builder
	title := lipgloss.NewStyle().Foreground(crushGold).Bold(true).Render("private vault")
	b.WriteString(title)
	b.WriteString("\n")
	b.WriteString(m.styles.help.Render("[enter] continue  [esc] cancel confirm  [q] quit"))
	b.WriteString("\n\n")
	b.WriteString(m.styles.panel.Width(width).Render(m.vaultPanel(width - 4)))
	b.WriteString("\n")
	b.WriteString(m.statusLine())
	if m.err != "" {
		b.WriteString("\n" + m.styles.error.Render(ansi.Wordwrap(m.err, max(20, width-2), " /_-")))
	}
	return b.String()
}

func (m Model) vaultPanel(width int) string {
	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Foreground(crushMint).Render("PRIVATE // PERSONAL // VAULTED"))
	b.WriteString("\n")
	b.WriteString(m.styles.help.Render(ansi.Wordwrap(vaultPurpose(), max(24, width), " /_-")))
	b.WriteString("\n\n")
	b.WriteString(m.vaultInput.View())
	return b.String()
}

func vaultPurpose() string {
	return "Protects play history, queues, private playlists, synced server cache, and recommendation context."
}
