package localstore

import (
	"errors"
	"strings"
	"testing"

	"github.com/bprendie/subweazl/internal/playqueue"
	"github.com/bprendie/subweazl/internal/subsonic"
)

func TestQueueSnapshotRequiresUnlock(t *testing.T) {
	store := newMigratedStore(t)
	err := store.SaveQueueSnapshot(playqueue.Snapshot{Tracks: []subsonic.Track{{ID: "song_1"}}})
	if !errors.Is(err, errLocked) {
		t.Fatalf("SaveQueueSnapshot locked error = %v, want errLocked", err)
	}
}

func TestQueueSnapshotEncryptsPayload(t *testing.T) {
	store := newMigratedStore(t)
	if err := store.CreateVault("pw"); err != nil {
		t.Fatalf("CreateVault: %v", err)
	}
	snapshot := playqueue.Snapshot{
		Current: 1,
		Tracks: []subsonic.Track{
			{ID: "song_1", Title: "Private Intro", Artist: "The Weazls"},
			{ID: "song_2", Title: "Private Signal", Artist: "The Weazls"},
		},
	}
	if err := store.SaveQueueSnapshot(snapshot); err != nil {
		t.Fatalf("SaveQueueSnapshot: %v", err)
	}
	var blob string
	if err := store.db.QueryRow(`select payload from queue_snapshot where id = 1`).Scan(&blob); err != nil {
		t.Fatalf("query queue snapshot: %v", err)
	}
	if strings.Contains(blob, "Private Signal") || strings.Contains(blob, "The Weazls") {
		t.Fatalf("queue snapshot exposes plaintext: %q", blob)
	}
	got, ok, err := store.QueueSnapshot()
	if err != nil {
		t.Fatalf("QueueSnapshot: %v", err)
	}
	if !ok || got.Current != 1 || len(got.Tracks) != 2 || got.Tracks[1].Title != "Private Signal" {
		t.Fatalf("snapshot = %#v ok=%v", got, ok)
	}
}
