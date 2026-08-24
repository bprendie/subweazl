package tui

import (
	"os/exec"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type themePalette struct {
	accent     lipgloss.Color
	secondary  lipgloss.Color
	success    lipgloss.Color
	warning    lipgloss.Color
	foreground lipgloss.Color
	muted      lipgloss.Color
	surface    lipgloss.Color
	border     lipgloss.Color
	canvas     lipgloss.Color
	logoA      lipgloss.Color
	logoB      lipgloss.Color
	logoC      lipgloss.Color
	slashA     lipgloss.Color
	slashB     lipgloss.Color
	slashC     lipgloss.Color
	trackA     lipgloss.Color
	trackB     lipgloss.Color
	trackC     lipgloss.Color
}

func loadOmarchyPaletteSnapshot() (themePalette, string) {
	fallback := fallbackPalette()
	output, err := exec.Command("omarchy-theme-color", "--all").Output()
	if err != nil {
		return fallback, "fallback"
	}
	text := string(output)
	return paletteFromColors(parseThemeColors(text), fallback), text
}

func parseThemeColors(output string) map[string]string {
	colors := make(map[string]string)
	for line := range strings.Lines(output) {
		fields := strings.Fields(line)
		if len(fields) == 2 && validHexColor(fields[1]) {
			colors[fields[0]] = fields[1]
		}
	}
	return colors
}

func validHexColor(value string) bool {
	if len(value) != 7 || value[0] != '#' {
		return false
	}
	for _, r := range value[1:] {
		if !strings.ContainsRune("0123456789abcdefABCDEF", r) {
			return false
		}
	}
	return true
}

func paletteFromColors(colors map[string]string, fallback themePalette) themePalette {
	return themePalette{
		accent:     themeColor(colors, "accent", fallback.accent),
		secondary:  themeColor(colors, "magenta", fallback.secondary),
		success:    themeColor(colors, "green", fallback.success),
		warning:    themeColor(colors, "yellow", fallback.warning),
		foreground: themeColor(colors, "foreground", fallback.foreground),
		muted:      themeColor(colors, "muted", fallback.muted),
		surface:    themeColor(colors, "lighter_background", fallback.surface),
		border:     themeColor(colors, "blue", fallback.border),
		canvas:     themeColor(colors, "background", fallback.canvas),
		logoA:      themeColor(colors, "blue", fallback.logoA),
		logoB:      themeColor(colors, "magenta", fallback.logoB),
		logoC:      themeColor(colors, "yellow", fallback.logoC),
		slashA:     themeColor(colors, "yellow", fallback.slashA),
		slashB:     themeColor(colors, "magenta", fallback.slashB),
		slashC:     themeColor(colors, "blue", fallback.slashC),
		trackA:     themeColor(colors, "yellow", fallback.trackA),
		trackB:     themeColor(colors, "magenta", fallback.trackB),
		trackC:     themeColor(colors, "blue", fallback.trackC),
	}
}

func themeColor(colors map[string]string, key string, fallback lipgloss.Color) lipgloss.Color {
	if value := strings.TrimSpace(colors[key]); validHexColor(value) {
		return lipgloss.Color(value)
	}
	return fallback
}

func fallbackPalette() themePalette {
	return themePalette{
		accent:     lipgloss.Color("#F25D94"),
		secondary:  lipgloss.Color("#7D56F4"),
		success:    lipgloss.Color("#04B575"),
		warning:    lipgloss.Color("#F7D774"),
		foreground: lipgloss.Color("#FAFAFA"),
		muted:      lipgloss.Color("#8E8E93"),
		surface:    lipgloss.Color("#181820"),
		border:     lipgloss.Color("#3D315B"),
		canvas:     lipgloss.Color("#0D0D12"),
		logoA:      lipgloss.Color("#F7D774"),
		logoB:      lipgloss.Color("#FFB84D"),
		logoC:      lipgloss.Color("#FF7A1A"),
		slashA:     lipgloss.Color("#7D56F4"),
		slashB:     lipgloss.Color("#B245FF"),
		slashC:     lipgloss.Color("#F25D94"),
		trackA:     lipgloss.Color("#F7D774"),
		trackB:     lipgloss.Color("#FFB84D"),
		trackC:     lipgloss.Color("#F25D94"),
	}
}
