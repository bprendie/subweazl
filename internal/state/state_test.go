package state

import (
	"os"
	"testing"

	"github.com/bprendie/subweazl/internal/subsonic"
)

func TestSaveLoadLastPlayed(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	track := subsonic.Track{
		ID:       "song-1",
		Title:    "Low Light",
		Artist:   "The Testers",
		Album:    "Fixture Radio",
		AlbumID:  "album-1",
		CoverID:  "cover-1",
		Duration: 180,
	}
	if err := Save(State{LastPlayed: ptr(FromTrack(track))}); err != nil {
		t.Fatalf("save state: %v", err)
	}
	st, err := Load()
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	if st.LastPlayed == nil {
		t.Fatal("last played was not loaded")
	}
	got := st.LastPlayed.Track()
	if got != track {
		t.Fatalf("loaded track mismatch: %#v", got)
	}
}

func TestLoadMissingState(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	st, err := Load()
	if err != nil {
		t.Fatalf("load missing state: %v", err)
	}
	if st.LastPlayed != nil {
		t.Fatalf("unexpected last played: %#v", st.LastPlayed)
	}
}

func TestStateFileMode(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := Save(State{}); err != nil {
		t.Fatalf("save state: %v", err)
	}
	path, err := Path()
	if err != nil {
		t.Fatalf("state path: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat state: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %o, want 600", info.Mode().Perm())
	}
}

func ptr[T any](v T) *T { return &v }
