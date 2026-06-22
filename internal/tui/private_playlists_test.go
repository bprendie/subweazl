package tui

import (
	"strings"
	"testing"

	"github.com/bprendie/subweazl/internal/playqueue"
	"github.com/bprendie/subweazl/internal/subsonic"
	"github.com/charmbracelet/bubbles/list"
)

func TestSaveQueueAsPrivatePlaylistShowsVaultList(t *testing.T) {
	m := newHomeTestModel(t)
	m.queue.Replace([]subsonic.Track{testTrack("a"), testTrack("b")}, 0)
	next, _ := m.saveQueueAsPrivatePlaylist("Road Mix")
	m = next
	if m.mode != modePrivatePlaylists {
		t.Fatalf("mode = %v, want private playlists", m.mode)
	}
	if !strings.Contains(privatePlaylistNames(m.list.Items()), "Road Mix") {
		t.Fatalf("private playlist names = %q", privatePlaylistNames(m.list.Items()))
	}
}

func TestLoadPrivatePlaylistIntoQueue(t *testing.T) {
	m := newHomeTestModel(t)
	playlist, err := m.vaultStore.SavePrivatePlaylist("Vault Mix", m.queueSnapshotForTest())
	if err != nil {
		t.Fatalf("SavePrivatePlaylist: %v", err)
	}
	m.queue.Clear()
	next, _ := m.loadPrivatePlaylist(item{kind: "private_playlist", privatePlaylist: playlist})
	m = next
	if m.mode != modeQueue {
		t.Fatalf("mode = %v, want queue", m.mode)
	}
	if current, ok := m.queue.Current(); !ok || current.ID != "b" {
		t.Fatalf("current = %s ok=%v, want b true", current.ID, ok)
	}
}

func TestRenameAndDeletePrivatePlaylist(t *testing.T) {
	m := newHomeTestModel(t)
	playlist, err := m.vaultStore.SavePrivatePlaylist("Old", m.queueSnapshotForTest())
	if err != nil {
		t.Fatalf("SavePrivatePlaylist: %v", err)
	}
	m.showPrivatePlaylists()
	next, _ := m.startPrivatePlaylistRename()
	m = next
	if !m.input.Focused() || m.privateRenaming != playlist.ID {
		t.Fatalf("rename state focused=%v id=%q", m.input.Focused(), m.privateRenaming)
	}
	next, _ = m.renamePrivatePlaylist("New")
	m = next
	if !strings.Contains(privatePlaylistNames(m.list.Items()), "New") {
		t.Fatalf("private playlist names after rename = %q", privatePlaylistNames(m.list.Items()))
	}
	next, _ = m.deletePrivatePlaylist()
	m = next
	if strings.Contains(privatePlaylistNames(m.list.Items()), "New") {
		t.Fatalf("private playlist names after delete = %q", privatePlaylistNames(m.list.Items()))
	}
}

func TestHomeActionPrivatePlaylistsShowsVaultList(t *testing.T) {
	m := newHomeTestModel(t)
	next, _ := m.handleHomeAction(item{kind: "home", action: homeActionPrivatePlaylists})
	if next.mode != modePrivatePlaylists {
		t.Fatalf("mode = %v, want private playlists", next.mode)
	}
}

func (m Model) queueSnapshotForTest() playqueue.Snapshot {
	return playqueue.Snapshot{Tracks: []subsonic.Track{testTrack("a"), testTrack("b")}, Current: 1}
}

func privatePlaylistNames(items []list.Item) string {
	var names []string
	for _, row := range items {
		if it, ok := row.(item); ok && it.kind == "private_playlist" {
			names = append(names, it.privatePlaylist.Name)
		}
	}
	return strings.Join(names, "|")
}
