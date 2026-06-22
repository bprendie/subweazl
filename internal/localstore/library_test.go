package localstore

import (
	"errors"
	"strings"
	"testing"
)

func TestLibraryWritesRequireUnlock(t *testing.T) {
	store := newMigratedStore(t)
	err := store.UpsertFolder("folder_1", map[string]string{"path": "/music"})
	if !errors.Is(err, errLocked) {
		t.Fatalf("UpsertFolder locked error = %v, want errLocked", err)
	}
	err = store.UpsertTrack(TrackRecord{
		ID:       "track_1",
		FolderID: "folder_1",
		Payload:  map[string]string{"title": "Locked"},
	})
	if !errors.Is(err, errLocked) {
		t.Fatalf("UpsertTrack locked error = %v, want errLocked", err)
	}
}

func TestUpsertFolderAndTrackEncryptPayloads(t *testing.T) {
	store := newMigratedStore(t)
	if err := store.CreateVault("pw"); err != nil {
		t.Fatalf("CreateVault: %v", err)
	}
	if err := store.UpsertFolder("folder_1", map[string]string{"path": "/music/secret"}); err != nil {
		t.Fatalf("UpsertFolder: %v", err)
	}
	if err := store.UpsertTrack(TrackRecord{
		ID:           "track_1",
		FolderID:     "folder_1",
		FileSize:     42,
		ModifiedUnix: 1234,
		Payload: map[string]string{
			"path":  "/music/secret/song.flac",
			"title": "Private Song",
		},
	}); err != nil {
		t.Fatalf("UpsertTrack: %v", err)
	}
	var folderBlob, trackBlob string
	if err := store.db.QueryRow(`select payload from folders where id = ?`, "folder_1").Scan(&folderBlob); err != nil {
		t.Fatalf("query folder payload: %v", err)
	}
	if err := store.db.QueryRow(`select payload from tracks where id = ?`, "track_1").Scan(&trackBlob); err != nil {
		t.Fatalf("query track payload: %v", err)
	}
	for _, blob := range []string{folderBlob, trackBlob} {
		if strings.Contains(blob, "/music") || strings.Contains(blob, "Private Song") {
			t.Fatalf("encrypted payload exposes plaintext: %q", blob)
		}
	}
	plain, err := store.decrypt(trackBlob)
	if err != nil {
		t.Fatalf("decrypt track payload: %v", err)
	}
	if !strings.Contains(plain, "Private Song") {
		t.Fatalf("decrypted payload = %q, want title", plain)
	}
}

func TestMarkMissingTracks(t *testing.T) {
	store := newMigratedStore(t)
	if err := store.CreateVault("pw"); err != nil {
		t.Fatalf("CreateVault: %v", err)
	}
	if err := store.UpsertFolder("folder_1", map[string]string{"path": "/music"}); err != nil {
		t.Fatalf("UpsertFolder: %v", err)
	}
	for _, id := range []string{"track_1", "track_2"} {
		if err := store.UpsertTrack(TrackRecord{ID: id, FolderID: "folder_1", Payload: map[string]string{"id": id}}); err != nil {
			t.Fatalf("UpsertTrack %s: %v", id, err)
		}
	}
	if err := store.MarkMissingTracks("folder_1", []string{"track_2"}); err != nil {
		t.Fatalf("MarkMissingTracks: %v", err)
	}
	got := map[string]int{}
	rows, err := store.db.Query(`select id, missing from tracks order by id`)
	if err != nil {
		t.Fatalf("query tracks: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var missing int
		if err := rows.Scan(&id, &missing); err != nil {
			t.Fatalf("scan track: %v", err)
		}
		got[id] = missing
	}
	if got["track_1"] != 1 || got["track_2"] != 0 {
		t.Fatalf("missing flags = %#v, want track_1 missing and track_2 present", got)
	}
}
