package localstore

import (
	"strings"
	"testing"

	"github.com/bprendie/subweazl/internal/subsonic"
)

func TestSubsonicCacheEncryptsAndSearches(t *testing.T) {
	store := newMigratedStore(t)
	if err := store.CreateVault("pw"); err != nil {
		t.Fatalf("CreateVault: %v", err)
	}
	if err := store.BeginSubsonicCacheSync(); err != nil {
		t.Fatalf("BeginSubsonicCacheSync: %v", err)
	}
	track := subsonic.Track{ID: "song_1", Title: "Private Signal", Artist: "The Weazls", Album: "Vault", Genre: "Synth"}
	if err := store.UpsertSubsonicTrack(track, true); err != nil {
		t.Fatalf("UpsertSubsonicTrack: %v", err)
	}
	if err := store.CompleteSubsonicCacheSync([]string{"song_1"}); err != nil {
		t.Fatalf("CompleteSubsonicCacheSync: %v", err)
	}
	var blob string
	if err := store.db.QueryRow(`select payload from tracks where id = ?`, "song_1").Scan(&blob); err != nil {
		t.Fatalf("query track payload: %v", err)
	}
	if strings.Contains(blob, "Private Signal") || strings.Contains(blob, "The Weazls") {
		t.Fatalf("cache payload exposes plaintext: %q", blob)
	}
	matches, err := store.CachedSubsonicSearch("weazls signal", 10)
	if err != nil {
		t.Fatalf("CachedSubsonicSearch: %v", err)
	}
	if len(matches) != 1 || matches[0].ID != "song_1" {
		t.Fatalf("matches = %#v", matches)
	}
	status, err := store.SubsonicCacheStatus()
	if err != nil {
		t.Fatalf("SubsonicCacheStatus: %v", err)
	}
	if status.TrackCount != 1 || status.LastScanCompletedAt == "" {
		t.Fatalf("status = %#v", status)
	}
}

func TestSubsonicCacheMarksMissing(t *testing.T) {
	store := newMigratedStore(t)
	if err := store.CreateVault("pw"); err != nil {
		t.Fatalf("CreateVault: %v", err)
	}
	if err := store.BeginSubsonicCacheSync(); err != nil {
		t.Fatalf("BeginSubsonicCacheSync: %v", err)
	}
	for _, track := range []subsonic.Track{{ID: "keep", Title: "Keep"}, {ID: "drop", Title: "Drop"}} {
		if err := store.UpsertSubsonicTrack(track, false); err != nil {
			t.Fatalf("UpsertSubsonicTrack: %v", err)
		}
	}
	if err := store.CompleteSubsonicCacheSync([]string{"keep"}); err != nil {
		t.Fatalf("CompleteSubsonicCacheSync: %v", err)
	}
	matches, err := store.CachedSubsonicSearch("drop", 10)
	if err != nil {
		t.Fatalf("CachedSubsonicSearch: %v", err)
	}
	if len(matches) != 0 {
		t.Fatalf("missing track was returned: %#v", matches)
	}
}
