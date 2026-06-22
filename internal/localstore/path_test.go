package localstore

import (
	"path/filepath"
	"testing"
)

func TestPathUsesXDGDataHome(t *testing.T) {
	dataHome := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataHome)
	t.Setenv("SUBWEAZL_DATA_HOME", "")
	got, err := Path()
	if err != nil {
		t.Fatalf("Path: %v", err)
	}
	want := filepath.Join(dataHome, "subweazl", "library.sqlite3")
	if got != want {
		t.Fatalf("Path() = %q, want %q", got, want)
	}
}

func TestPathUsesOverride(t *testing.T) {
	dataHome := t.TempDir()
	t.Setenv("SUBWEAZL_DATA_HOME", dataHome)
	got, err := Path()
	if err != nil {
		t.Fatalf("Path: %v", err)
	}
	want := filepath.Join(dataHome, "subweazl", "library.sqlite3")
	if got != want {
		t.Fatalf("Path() = %q, want %q", got, want)
	}
}
