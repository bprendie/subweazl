package playqueue

import (
	"testing"

	"github.com/bprendie/subweazl/internal/subsonic"
)

func TestQueueReplaceAndNextPrevious(t *testing.T) {
	q := New()
	q.Replace([]subsonic.Track{track("a"), track("b"), track("c")}, 1)
	if current, ok := q.Current(); !ok || current.ID != "b" {
		t.Fatalf("current = %v %v, want b true", current.ID, ok)
	}
	if next, ok := q.Next(); !ok || next.ID != "c" {
		t.Fatalf("next = %v %v, want c true", next.ID, ok)
	}
	if _, ok := q.Next(); ok {
		t.Fatalf("Next past end returned ok")
	}
	if prev, ok := q.Previous(); !ok || prev.ID != "b" {
		t.Fatalf("previous = %v %v, want b true", prev.ID, ok)
	}
}

func TestQueueRemoveKeepsCurrentTrack(t *testing.T) {
	q := New()
	q.Replace([]subsonic.Track{track("a"), track("b"), track("c")}, 2)
	if !q.Remove(0) {
		t.Fatalf("Remove returned false")
	}
	if q.CurrentIndex() != 1 {
		t.Fatalf("current index = %d, want 1", q.CurrentIndex())
	}
	if current, _ := q.Current(); current.ID != "c" {
		t.Fatalf("current = %s, want c", current.ID)
	}
}

func TestQueueMoveUpdatesCurrent(t *testing.T) {
	q := New()
	q.Replace([]subsonic.Track{track("a"), track("b"), track("c")}, 1)
	if !q.Move(1, -1) {
		t.Fatalf("Move returned false")
	}
	if q.CurrentIndex() != 0 {
		t.Fatalf("current index = %d, want 0", q.CurrentIndex())
	}
	tracks := q.Tracks()
	if tracks[0].ID != "b" || tracks[1].ID != "a" {
		t.Fatalf("tracks = %#v", tracks)
	}
}

func TestQueueSnapshotRestore(t *testing.T) {
	q := FromSnapshot(Snapshot{Tracks: []subsonic.Track{track("a"), track("b")}, Current: 7})
	if q.CurrentIndex() != 1 {
		t.Fatalf("current index = %d, want clamped 1", q.CurrentIndex())
	}
	if snapshot := q.Snapshot(); len(snapshot.Tracks) != 2 || snapshot.Current != 1 {
		t.Fatalf("snapshot = %#v", snapshot)
	}
}

func track(id string) subsonic.Track {
	return subsonic.Track{ID: id, Title: "Track " + id}
}
