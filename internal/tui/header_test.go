package tui

import (
	"strings"
	"testing"

	"github.com/bprendie/subweazl/internal/config"
)

func TestMainHeaderUsesCompactHeader(t *testing.T) {
	for _, height := range []int{40, 46, 60} {
		m := headerTestModel(t, 100, height)
		got := m.appHeader(90)
		if got != "" && got[0] == '\n' {
			t.Fatalf("main header should stay compact at height %d", height)
		}
		if strings.Contains(got, ".________") {
			t.Fatalf("main header should not render the full logo at height %d", height)
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
