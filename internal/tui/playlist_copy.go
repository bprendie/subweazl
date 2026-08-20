package tui

import (
	"context"
	"fmt"

	"github.com/bprendie/subweazl/internal/playqueue"
	"github.com/bprendie/subweazl/internal/subsonic"
	tea "github.com/charmbracelet/bubbletea"
)

type playlistCopiedMsg struct {
	name        string
	destination string
}

func (m Model) copySelectedPlaylist() (Model, tea.Cmd) {
	if m.vaultStore == nil || !m.vaultStore.Unlocked() {
		m.err = "private vault is locked"
		return m, noop
	}
	it, ok := m.list.SelectedItem().(item)
	if !ok {
		m.err = "select a playlist to copy"
		return m, noop
	}
	switch it.kind {
	case "playlist":
		playlist, store, client := it.playlist, m.vaultStore, m.client
		m.beginSearch("copying " + playlist.Name + " to private vault")
		return m, func() tea.Msg {
			tracks, err := client.Playlist(context.Background(), playlist.ID)
			if err != nil {
				return errMsg{err: fmt.Errorf("load server playlist: %w", err)}
			}
			_, err = store.SaveOrReplacePrivatePlaylist(playlist.Name, playqueue.Snapshot{Tracks: tracks, Current: 0})
			if err != nil {
				return errMsg{err: fmt.Errorf("copy playlist to vault: %w", err)}
			}
			return playlistCopiedMsg{name: playlist.Name, destination: "private vault"}
		}
	case "private_playlist":
		playlist, client := it.privatePlaylist, m.client
		m.beginSearch("copying " + playlist.Name + " to server")
		return m, func() tea.Msg {
			_, err := client.SaveOrReplacePlaylist(context.Background(), playlist.Name, append([]subsonic.Track(nil), playlist.Tracks...))
			if err != nil {
				return errMsg{err: fmt.Errorf("copy playlist to server: %w", err)}
			}
			return playlistCopiedMsg{name: playlist.Name, destination: "server"}
		}
	default:
		m.err = "select a server or private playlist to copy"
		return m, noop
	}
}
