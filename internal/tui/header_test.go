package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"

	"github.com/bprendie/subweazl/internal/config"
)

func TestMainHeaderUsesFullLogoWithoutTopPadding(t *testing.T) {
	m := headerTestModel(t, 100, 40)
	got := m.appHeader(90)
	if got == "" || got[0] == '\n' {
		t.Fatal("main logo header should not start with a blank line")
	}
	if lines := strings.Count(got, "\n") + 1; lines < 7 {
		t.Fatalf("main header should include the full logo, got %d lines", lines)
	}
}

func TestMainHeaderFallsBackToCompactWhenNarrow(t *testing.T) {
	m := headerTestModel(t, 60, 40)
	got := m.appHeader(40)
	if got == "" || got[0] == '\n' {
		t.Fatal("compact header should not start with a blank line")
	}
	if lines := strings.Count(got, "\n") + 1; lines != 3 {
		t.Fatalf("compact header should be one bordered line, got %d rows", lines)
	}
}

func TestMainViewFitsTerminalWithFullLogo(t *testing.T) {
	for _, size := range []struct {
		width  int
		height int
	}{
		{100, 40},
		{100, 34},
		{82, 34},
	} {
		m := headerTestModel(t, size.width, size.height)
		m.resize(size.width, size.height)
		rendered := m.View()
		if got := lipgloss.Height(rendered); got > size.height {
			t.Fatalf("view height = %d, terminal height = %d", got, size.height)
		}
	}
}

func headerTestModel(t *testing.T, width, height int) Model {
	t.Helper()
	t.Setenv("SUBWEAZL_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	m := New(config.Config{Server: "https://example.test", Username: "user", Password: "pass"})
	m.width = width
	m.height = height
	return m
}
