package tui

import (
	"testing"

	"github.com/bprendie/subweazl/internal/playqueue"
	"github.com/bprendie/subweazl/internal/subsonic"
)

func TestPlaySelectedTrackBuildsQueueContext(t *testing.T) {
	m := newHomeTestModel(t)
	tracks := []subsonic.Track{testTrack("a"), testTrack("b"), testTrack("c")}
	m.mode = modeTracks
	m.list.SetItems(trackItems(tracks))
	m.list.Select(1)
	context, index := m.trackContext("b")
	if index != 1 || len(context) != 3 || context[2].ID != "c" {
		t.Fatalf("context len=%d index=%d tracks=%#v", len(context), index, context)
	}
}

func TestPlaybackModeCycle(t *testing.T) {
	var m Model
	want := []playbackMode{playbackShuffle, playbackShuffleRepeat, playbackRepeat, playbackOff}
	for _, mode := range want {
		m.cyclePlaybackMode()
		if m.playbackMode != mode {
			t.Fatalf("mode = %v, want %v", m.playbackMode, mode)
		}
	}
}

func TestShuffleBuildsPermutationWithoutCurrentTrack(t *testing.T) {
	m := Model{queue: playqueue.FromSnapshot(playqueue.Snapshot{
		Tracks:  []subsonic.Track{testTrack("a"), testTrack("b"), testTrack("c")},
		Current: 0,
	}), playbackMode: playbackShuffle}
	m.fillShuffleUpcoming()
	if len(m.shuffleUpcoming) != 2 {
		t.Fatalf("upcoming = %v, want two tracks", m.shuffleUpcoming)
	}
	seen := map[int]bool{}
	for _, index := range m.shuffleUpcoming {
		if index == 0 || seen[index] {
			t.Fatalf("invalid shuffle permutation %v", m.shuffleUpcoming)
		}
		seen[index] = true
	}
}

func TestQueueSnapshotRestoresIntoModel(t *testing.T) {
	m := newHomeTestModel(t)
	snapshot := playqueue.Snapshot{Tracks: []subsonic.Track{testTrack("a"), testTrack("b")}, Current: 1}
	if err := m.vaultStore.SaveQueueSnapshot(snapshot); err != nil {
		t.Fatalf("SaveQueueSnapshot: %v", err)
	}
	m.queue.Clear()
	m.restoreQueueSnapshot()
	if current, ok := m.queue.Current(); !ok || current.ID != "b" {
		t.Fatalf("current = %s ok=%v, want b true", current.ID, ok)
	}
}

func TestShowQueueDisplaysCurrentTrack(t *testing.T) {
	m := newHomeTestModel(t)
	m.queue.Replace([]subsonic.Track{testTrack("a"), testTrack("b")}, 1)
	m.showQueue()
	if m.mode != modeQueue {
		t.Fatalf("mode = %v, want modeQueue", m.mode)
	}
	items := m.list.Items()
	if len(items) != 2 {
		t.Fatalf("items = %d, want 2", len(items))
	}
	it := items[1].(item)
	if it.kind != "queue" || it.queueIndex != 1 || it.Title() != "▶ Track b" {
		t.Fatalf("queue row = %#v title=%q", it, it.Title())
	}
}

func TestQueueRemoveAndMovePersist(t *testing.T) {
	m := newHomeTestModel(t)
	m.queue.Replace([]subsonic.Track{testTrack("a"), testTrack("b"), testTrack("c")}, 1)
	m.showQueue()
	m.list.Select(2)
	next, _ := m.moveQueueSelection(-1)
	m = next
	if got := m.queue.Tracks()[1].ID; got != "c" {
		t.Fatalf("track at index 1 = %s, want c", got)
	}
	next, _ = m.removeQueueSelection()
	m = next
	snapshot, ok, err := m.vaultStore.QueueSnapshot()
	if err != nil || !ok {
		t.Fatalf("QueueSnapshot err=%v ok=%v", err, ok)
	}
	if len(snapshot.Tracks) != 2 || snapshot.Tracks[1].ID != "b" {
		t.Fatalf("snapshot = %#v", snapshot)
	}
}

func TestPlayingTrackFollowPrefersQueueIndexForDuplicateTracks(t *testing.T) {
	m := newHomeTestModel(t)
	duplicate := testTrack("duplicate")
	tracks := []subsonic.Track{duplicate, duplicate, testTrack("last")}
	m.mode = modeTracks
	m.playlistViewID = "playlist-1"
	m.queueSourceID = "playlist-1"
	m.queue.Replace(tracks, 1)
	m.list.SetItems(trackItems(tracks))
	m.list.Select(0)
	m.playing = &duplicate
	m.followPlayingTrack()
	if m.list.Index() != 1 {
		t.Fatalf("selected=%d, want active queue index 1", m.list.Index())
	}
}

func TestPlayingTrackFollowDoesNotMoveDifferentOpenPlaylist(t *testing.T) {
	m := newHomeTestModel(t)
	playing := testTrack("playing")
	m.mode = modeTracks
	m.playlistViewID = "browsed-playlist"
	m.queueSourceID = "active-playlist"
	m.queue.Replace([]subsonic.Track{playing}, 0)
	m.list.SetItems(trackItems([]subsonic.Track{testTrack("other"), playing}))
	m.list.Select(0)
	m.playing = &playing
	m.followPlayingTrack()
	if m.list.Index() != 0 {
		t.Fatalf("selected=%d, should preserve manual browsing", m.list.Index())
	}
}

func TestPlayingTrackFollowMovesQueueCursor(t *testing.T) {
	m := newHomeTestModel(t)
	tracks := []subsonic.Track{testTrack("first"), testTrack("playing")}
	m.queue.Replace(tracks, 1)
	m.showQueue()
	m.list.Select(0)
	m.playing = &tracks[1]
	m.followPlayingTrack()
	if m.list.Index() != 1 {
		t.Fatalf("selected=%d, want 1", m.list.Index())
	}
}

func testTrack(id string) subsonic.Track {
	return subsonic.Track{ID: id, Title: "Track " + id, Artist: "Artist", Album: "Album"}
}
