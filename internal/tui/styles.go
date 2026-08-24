package tui

import "github.com/charmbracelet/lipgloss"

var (
	activePalette, activeThemeSignature = loadOmarchyPaletteSnapshot()
	accent                              = activePalette.accent
	secondary                           = activePalette.secondary
	success                             = activePalette.success
	warning                             = activePalette.warning
	foreground                          = activePalette.foreground
	muted                               = activePalette.muted
	panelColor                          = activePalette.surface
	border                              = activePalette.border
	canvas                              = activePalette.canvas
)

func applyPalette(p themePalette) {
	activePalette = p
	accent, secondary, success = p.accent, p.secondary, p.success
	warning, foreground, muted = p.warning, p.foreground, p.muted
	panelColor, border, canvas = p.surface, p.border, p.canvas
	setGradientStops(p)
}

type styles struct {
	frame    lipgloss.Style
	header   lipgloss.Style
	panel    lipgloss.Style
	active   lipgloss.Style
	track    lipgloss.Style
	status   lipgloss.Style
	help     lipgloss.Style
	selected lipgloss.Style
	item     lipgloss.Style
	error    lipgloss.Style
}

func newStyles() styles {
	return styles{
		frame: lipgloss.NewStyle().
			Foreground(foreground).
			Background(canvas).
			Padding(1, 2),
		header: lipgloss.NewStyle().
			Foreground(accent).
			Bold(true),
		panel: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(border).
			Background(panelColor).
			Padding(0, 1),
		active: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(secondary).
			Background(panelColor).
			Padding(0, 1),
		track: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(secondary).
			Foreground(warning).
			Background(panelColor).
			Padding(0, 1),
		status:   lipgloss.NewStyle().Foreground(success).Bold(true),
		help:     lipgloss.NewStyle().Foreground(muted),
		selected: lipgloss.NewStyle().Foreground(accent).Bold(true),
		item:     lipgloss.NewStyle().Foreground(foreground),
		error:    lipgloss.NewStyle().Foreground(warning).Bold(true),
	}
}
