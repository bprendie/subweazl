package localstore

import (
	"path/filepath"
	"testing"
)

func TestFolderAndTrackSummariesRequireUnlock(t *testing.T) {
	path := filepath.Join(t.TempDir(), "library.sqlite3")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := store.Migrate(); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if err := store.CreateVault("pw"); err != nil {
		t.Fatalf("CreateVault: %v", err)
	}
	if err := store.UpsertFolder("folder_1", map[string]string{"path": "/music"}); err != nil {
		t.Fatalf("UpsertFolder: %v", err)
	}
	if err := store.UpsertTrack(TrackRecord{
		ID:       "track_1",
		FolderID: "folder_1",
		Payload: map[string]string{
			"title":  "Signal",
			"artist": "The Weazls",
			"album":  "Vault",
			"path":   "/music/signal.flac",
		},
	}); err != nil {
		t.Fatalf("UpsertTrack: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	locked, err := Open(path)
	if err != nil {
		t.Fatalf("Open locked: %v", err)
	}
	defer locked.Close()
	if _, err := locked.FolderSummaries(); err == nil {
		t.Fatal("FolderSummaries succeeded while locked")
	}
	if err := locked.Unlock("pw"); err != nil {
		t.Fatalf("Unlock: %v", err)
	}
	folders, err := locked.FolderSummaries()
	if err != nil {
		t.Fatalf("FolderSummaries: %v", err)
	}
	tracks, err := locked.TrackSummaries(10)
	if err != nil {
		t.Fatalf("TrackSummaries: %v", err)
	}
	if len(folders) != 1 || folders[0].Path != "/music" {
		t.Fatalf("folders = %#v", folders)
	}
	if len(tracks) != 1 || tracks[0].Title != "Signal" || tracks[0].Artist != "The Weazls" {
		t.Fatalf("tracks = %#v", tracks)
	}
}

func TestTrackSummariesSortByMusicMetadata(t *testing.T) {
	store := newMigratedStore(t)
	if err := store.CreateVault("pw"); err != nil {
		t.Fatalf("CreateVault: %v", err)
	}
	if err := store.UpsertFolder("folder_1", map[string]string{"path": "/music"}); err != nil {
		t.Fatalf("UpsertFolder: %v", err)
	}
	records := []TrackRecord{
		{ID: "b2", FolderID: "folder_1", Payload: map[string]any{
			"title": "Second", "artist": "Artist B", "album": "Album", "track_number": 2, "path": "/music/b2.flac",
		}},
		{ID: "a1", FolderID: "folder_1", Payload: map[string]any{
			"title": "First", "artist": "Artist A", "album": "Album", "track_number": 1, "path": "/music/a1.flac",
		}},
		{ID: "b1", FolderID: "folder_1", Payload: map[string]any{
			"title": "First", "artist": "Artist B", "album": "Album", "track_number": 1, "path": "/music/b1.flac",
		}},
	}
	for _, record := range records {
		if err := store.UpsertTrack(record); err != nil {
			t.Fatalf("UpsertTrack %s: %v", record.ID, err)
		}
	}
	tracks, err := store.TrackSummaries(10)
	if err != nil {
		t.Fatalf("TrackSummaries: %v", err)
	}
	got := []string{tracks[0].ID, tracks[1].ID, tracks[2].ID}
	want := []string{"a1", "b1", "b2"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("sorted ids = %#v, want %#v", got, want)
		}
	}
}
