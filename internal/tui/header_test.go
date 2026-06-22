package tui

import (
	"testing"

	"github.com/bprendie/subweazl/internal/config"
)

func TestMainHeaderUsesCompactHeaderAtNormalHeights(t *testing.T) {
	m := headerTestModel(t, 100, 40)
	got := m.appHeader(90)
	if got != "" && got[0] == '\n' {
		t.Fatal("main header should stay compact at normal terminal heights")
	}
}

func TestFullLogoHeaderStartsWithSafeBlankLineWhenTall(t *testing.T) {
	m := headerTestModel(t, 100, 46)
	got := m.appHeader(90)
	if got == "" || got[0] != '\n' {
		t.Fatal("full logo header should start with a safe blank line")
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
