package tui

import (
	"strings"
	"testing"

	"github.com/bprendie/subweazl/internal/playqueue"
	"github.com/bprendie/subweazl/internal/subsonic"
	tea "github.com/charmbracelet/bubbletea"
)

func TestPlaylistDeleteCancelPreservesPlaylistQueueAndPlayback(t *testing.T) {
	m := newHomeTestModel(t)
	playlist, err := m.vaultStore.SavePrivatePlaylist("Keep Me", playqueue.Snapshot{Tracks: []subsonic.Track{testTrack("saved")}})
	if err != nil {
		t.Fatal(err)
	}
	m.queue.Replace([]subsonic.Track{testTrack("playing"), testTrack("next")}, 0)
	playing := testTrack("playing")
	m.playing = &playing
	m.showPrivatePlaylists()
	next, _ := m.startPlaylistDelete()
	if next.mode != modePlaylistDelete || !strings.Contains(next.playlistDeleteView(80), `Delete "Keep Me"`) {
		t.Fatalf("delete confirmation not shown: mode=%v view=%q", next.mode, next.playlistDeleteView(80))
	}
	next, _ = next.handlePlaylistDeleteKey(tea.KeyMsg{Type: tea.KeyEsc})
	if next.mode != modePrivatePlaylists || next.playing == nil || next.playing.ID != "playing" {
		t.Fatalf("cancel changed state: mode=%v playing=%#v", next.mode, next.playing)
	}
	assertIDs(t, queueIDs(next), []string{"playing", "next"})
	playlists, err := next.vaultStore.PrivatePlaylists()
	if err != nil || len(playlists) != 1 || playlists[0].ID != playlist.ID {
		t.Fatalf("playlist deleted on cancel: playlists=%v err=%v", playlists, err)
	}
}

func TestDeleteOnlyStartsFromPlaylistLists(t *testing.T) {
	m := newHomeTestModel(t)
	m.showQueue()
	next, _ := m.startPlaylistDelete()
	if next.mode == modePlaylistDelete || !strings.Contains(next.err, "select a server or private playlist") {
		t.Fatalf("mode=%v err=%q", next.mode, next.err)
	}
}
