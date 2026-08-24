package tui

import "testing"

func TestPaletteFromOmarchyColors(t *testing.T) {
	colors := parseThemeColors("accent\t#112233\nforeground #fefefe\ngreen #00aa00\n")
	got := paletteFromColors(colors, fallbackPalette())
	if string(got.accent) != "#112233" || string(got.foreground) != "#fefefe" || string(got.success) != "#00aa00" {
		t.Fatalf("palette = %#v", got)
	}
}

func TestPaletteFallsBackForMissingAndInvalidColors(t *testing.T) {
	fallback := fallbackPalette()
	colors := parseThemeColors("accent nope\nblue #12345z\nforeground #abcdef\n")
	got := paletteFromColors(colors, fallback)
	if got.accent != fallback.accent || got.border != fallback.border || string(got.foreground) != "#abcdef" {
		t.Fatalf("palette = %#v, fallback = %#v", got, fallback)
	}
}

func TestParseThemeColorsIgnoresUnknownShapes(t *testing.T) {
	got := parseThemeColors("accent #112233 extra\n# comment\nmagenta #AABBCC\n")
	if len(got) != 1 || got["magenta"] != "#AABBCC" {
		t.Fatalf("colors = %#v", got)
	}
}
