package tui

import (
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

func (m *Model) refreshTheme(now time.Time) {
	if now.Sub(m.themeChecked) < time.Second {
		return
	}
	m.themeChecked = now
	palette, signature := loadOmarchyPaletteSnapshot()
	if signature == m.themeSignature {
		return
	}
	m.themeSignature = signature
	applyPalette(palette)
	m.styles = newStyles()
	m.applyThemeStyles()
}

func (m *Model) applyThemeStyles() {
	m.list.SetDelegate(delegate{styles: m.styles})
	m.list.Styles.Title = lipgloss.NewStyle().Foreground(foreground).Background(accent).Padding(0, 1)
	m.list.Styles.Spinner = lipgloss.NewStyle().Foreground(accent)
	m.list.Styles.FilterPrompt = lipgloss.NewStyle().Foreground(success)
	m.list.Styles.FilterCursor = lipgloss.NewStyle().Foreground(secondary)
	m.list.Styles.DefaultFilterCharacterMatch = lipgloss.NewStyle().Foreground(warning).Underline(true)
	m.list.Styles.NoItems = lipgloss.NewStyle().Foreground(muted)
	m.list.Styles.ActivePaginationDot = lipgloss.NewStyle().Foreground(accent)
	m.list.Styles.InactivePaginationDot = lipgloss.NewStyle().Foreground(muted)
	styleTextInput(&m.input)
	styleTextInput(&m.vaultInput)
	for i := range m.setup {
		styleTextInput(&m.setup[i])
	}
	m.spinner.Style = lipgloss.NewStyle().Foreground(accent)
}

func styleTextInput(input *textinput.Model) {
	input.PromptStyle = lipgloss.NewStyle().Foreground(accent).Bold(true)
	input.TextStyle = lipgloss.NewStyle().Foreground(foreground)
	input.PlaceholderStyle = lipgloss.NewStyle().Foreground(muted)
	input.CompletionStyle = lipgloss.NewStyle().Foreground(muted)
	input.Cursor.Style = lipgloss.NewStyle().Foreground(secondary)
}
