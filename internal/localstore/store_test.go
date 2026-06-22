package localstore

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOpenCreatesDatabaseDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "library.sqlite3")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer store.Close()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("database file was not created: %v", err)
	}
}

func TestMigrateCreatesLocalLibrarySchema(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "library.sqlite3"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer store.Close()
	if err := store.Migrate(); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	for _, table := range []string{
		"vault",
		"folders",
		"tracks",
		"albums",
		"artists",
		"local_playlists",
		"local_playlist_tracks",
		"queue_snapshot",
		"play_history",
		"ratings",
		"station_recipes",
		"recommendation_runs",
	} {
		if !schemaObjectExists(t, store, "table", table) {
			t.Fatalf("table %q was not created", table)
		}
	}
	for _, index := range []string{
		"idx_tracks_folder",
		"idx_tracks_file_hash",
		"idx_tracks_missing",
		"idx_local_playlist_tracks_playlist",
		"idx_local_playlist_tracks_track",
		"idx_play_history_recent",
		"idx_play_history_track",
		"idx_local_playlists_updated",
		"idx_recommendation_runs_created",
	} {
		if !schemaObjectExists(t, store, "index", index) {
			t.Fatalf("index %q was not created", index)
		}
	}
}

func schemaObjectExists(t *testing.T, store *Store, kind, name string) bool {
	t.Helper()
	var count int
	err := store.db.QueryRow(`select count(*) from sqlite_master where type = ? and name = ?`, kind, name).Scan(&count)
	if err != nil {
		t.Fatalf("query schema object %s %s: %v", kind, name, err)
	}
	return count == 1
}
