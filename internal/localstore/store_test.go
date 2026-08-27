package localstore

import (
	"os"
	"path/filepath"
	"testing"
	"time"
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

func TestOpenWaitsForCompetingWriter(t *testing.T) {
	path := filepath.Join(t.TempDir(), "library.sqlite3")
	first, err := Open(path)
	if err != nil {
		t.Fatalf("Open first: %v", err)
	}
	defer first.Close()
	if err := first.Migrate(); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	second, err := Open(path)
	if err != nil {
		t.Fatalf("Open second: %v", err)
	}
	defer second.Close()

	tx, err := first.db.Begin()
	if err != nil {
		t.Fatalf("Begin: %v", err)
	}
	if _, err := tx.Exec(`insert into folders (id, payload) values ('held', '{}')`); err != nil {
		t.Fatalf("hold write lock: %v", err)
	}
	done := make(chan error, 1)
	go func() {
		_, err := second.db.Exec(`insert into folders (id, payload) values ('waited', '{}')`)
		done <- err
	}()
	time.Sleep(50 * time.Millisecond)
	if err := tx.Commit(); err != nil {
		t.Fatalf("Commit: %v", err)
	}
	if err := <-done; err != nil {
		t.Fatalf("competing writer did not wait: %v", err)
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
