package localstore

import (
	"errors"
	"strings"
	"testing"
)

func TestPlayHistoryRequiresUnlock(t *testing.T) {
	store := newMigratedStore(t)
	err := store.AddPlayHistory(PlayHistoryRecord{
		Source:  SourceLocal,
		TrackID: "track_1",
		Payload: map[string]string{"title": "Locked"},
	})
	if !errors.Is(err, errLocked) {
		t.Fatalf("AddPlayHistory locked error = %v, want errLocked", err)
	}
}

func TestPlayHistoryEncryptsPayload(t *testing.T) {
	store := newMigratedStore(t)
	if err := store.CreateVault("pw"); err != nil {
		t.Fatalf("CreateVault: %v", err)
	}
	err := store.AddPlayHistory(PlayHistoryRecord{
		Source:  SourceSubsonic,
		TrackID: "song_1",
		Payload: map[string]any{
			"title":  "Private Signal",
			"artist": "The Weazls",
			"album":  "Vault",
		},
	})
	if err != nil {
		t.Fatalf("AddPlayHistory: %v", err)
	}
	var blob string
	if err := store.db.QueryRow(`select payload from play_history where track_id = ?`, "song_1").Scan(&blob); err != nil {
		t.Fatalf("query play history payload: %v", err)
	}
	if strings.Contains(blob, "Private Signal") || strings.Contains(blob, "The Weazls") {
		t.Fatalf("play history payload exposes plaintext: %q", blob)
	}
	entries, err := store.PlayHistory(10)
	if err != nil {
		t.Fatalf("PlayHistory: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("history entries = %d, want 1", len(entries))
	}
	if entries[0].Source != SourceSubsonic || entries[0].Payload["title"] != "Private Signal" {
		t.Fatalf("history entry = %#v", entries[0])
	}
}
