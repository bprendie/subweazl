package tui

import (
	"testing"

	"github.com/bprendie/subweazl/internal/config"
)

func TestFullLogoHeaderStartsWithSafeBlankLine(t *testing.T) {
	t.Setenv("SUBWEAZL_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	m := New(config.Config{Server: "https://example.test", Username: "user", Password: "pass"})
	m.width = 100
	m.height = 40
	got := m.appHeader(90)
	if got == "" || got[0] != '\n' {
		t.Fatalf("full logo header should start with a safe blank line")
	}
}
