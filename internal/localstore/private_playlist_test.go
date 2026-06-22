package localstore

import (
	"errors"
	"strings"
	"testing"

	"github.com/bprendie/subweazl/internal/playqueue"
	"github.com/bprendie/subweazl/internal/subsonic"
)

func TestPrivatePlaylistRequiresUnlock(t *testing.T) {
	store := newMigratedStore(t)
	_, err := store.SavePrivatePlaylist("Locked", playqueue.Snapshot{Tracks: []subsonic.Track{{ID: "song_1"}}})
	if !errors.Is(err, errLocked) {
		t.Fatalf("SavePrivatePlaylist locked error = %v, want errLocked", err)
	}
}

func TestPrivatePlaylistEncryptsAndRoundTrips(t *testing.T) {
	store := newUnlockedPrivatePlaylistStore(t)
	playlist, err := store.SavePrivatePlaylist("Night Drive", playqueue.Snapshot{
		Current: 1,
		Tracks: []subsonic.Track{
			{ID: "song_1", Title: "Private Intro", Artist: "The Weazls"},
			{ID: "song_2", Title: "Private Signal", Artist: "The Weazls"},
		},
	})
	if err != nil {
		t.Fatalf("SavePrivatePlaylist: %v", err)
	}
	var blob string
	if err := store.db.QueryRow(`select payload from local_playlists where id = ?`, playlist.ID).Scan(&blob); err != nil {
		t.Fatalf("query private playlist: %v", err)
	}
	if strings.Contains(blob, "Night Drive") || strings.Contains(blob, "Private Signal") {
		t.Fatalf("private playlist exposes plaintext: %q", blob)
	}
	playlists, err := store.PrivatePlaylists()
	if err != nil {
		t.Fatalf("PrivatePlaylists: %v", err)
	}
	if len(playlists) != 1 || playlists[0].Name != "Night Drive" || playlists[0].Tracks[1].Title != "Private Signal" {
		t.Fatalf("playlists = %#v", playlists)
	}
	loaded, ok, err := store.PrivatePlaylist(playlist.ID)
	if err != nil || !ok {
		t.Fatalf("PrivatePlaylist err=%v ok=%v", err, ok)
	}
	if loaded.Snapshot().Current != 1 || len(loaded.Snapshot().Tracks) != 2 {
		t.Fatalf("snapshot = %#v", loaded.Snapshot())
	}
}

func TestPrivatePlaylistRenameAndDelete(t *testing.T) {
	store := newUnlockedPrivatePlaylistStore(t)
	playlist, err := store.SavePrivatePlaylist("First", playqueue.Snapshot{Tracks: []subsonic.Track{{ID: "song_1", Title: "One"}}})
	if err != nil {
		t.Fatalf("SavePrivatePlaylist: %v", err)
	}
	if err := store.RenamePrivatePlaylist(playlist.ID, "Second"); err != nil {
		t.Fatalf("RenamePrivatePlaylist: %v", err)
	}
	loaded, ok, err := store.PrivatePlaylist(playlist.ID)
	if err != nil || !ok || loaded.Name != "Second" {
		t.Fatalf("loaded = %#v ok=%v err=%v", loaded, ok, err)
	}
	if err := store.DeletePrivatePlaylist(playlist.ID); err != nil {
		t.Fatalf("DeletePrivatePlaylist: %v", err)
	}
	_, ok, err = store.PrivatePlaylist(playlist.ID)
	if err != nil || ok {
		t.Fatalf("after delete ok=%v err=%v", ok, err)
	}
}

func newUnlockedPrivatePlaylistStore(t *testing.T) *Store {
	t.Helper()
	store := newMigratedStore(t)
	if err := store.CreateVault("pw"); err != nil {
		t.Fatalf("CreateVault: %v", err)
	}
	return store
}
