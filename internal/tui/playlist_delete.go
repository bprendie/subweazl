package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/bprendie/subweazl/internal/localstore"
	"github.com/bprendie/subweazl/internal/subsonic"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

type playlistDeleteDestination int

const (
	playlistDeleteServer playlistDeleteDestination = iota
	playlistDeleteVault
)

type playlistDeleteDraft struct {
	destination playlistDeleteDestination
	id          string
	name        string
	previous    mode
	index       int
}

type playlistDeletedMsg struct {
	destination playlistDeleteDestination
	name        string
	index       int
	server      []subsonic.Playlist
	private     []localstore.PrivatePlaylist
}

func (m Model) startPlaylistDelete() (Model, tea.Cmd) {
	it, ok := m.list.SelectedItem().(item)
	if !ok {
		return m, noop
	}
	draft := playlistDeleteDraft{previous: m.mode, index: m.list.Index()}
	switch {
	case m.mode == modePlaylists && it.kind == "playlist":
		draft.destination, draft.id, draft.name = playlistDeleteServer, it.playlist.ID, it.playlist.Name
	case m.mode == modePrivatePlaylists && it.kind == "private_playlist":
		if m.vaultStore == nil || !m.vaultStore.Unlocked() {
			m.err = "private vault is locked"
			return m, noop
		}
		draft.destination, draft.id, draft.name = playlistDeleteVault, it.privatePlaylist.ID, it.privatePlaylist.Name
	default:
		m.err = "select a server or private playlist to delete"
		return m, noop
	}
	if draft.id == "" {
		m.err = "selected playlist has no ID"
		return m, noop
	}
	m.playlistDelete = draft
	m.mode = modePlaylistDelete
	m.err = ""
	m.status = "confirm playlist deletion"
	m.refreshTitle()
	return m, noop
}

func (m Model) handlePlaylistDeleteKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch strings.ToLower(msg.String()) {
	case "y":
		draft := m.playlistDelete
		m.mode = draft.previous
		m.playlistDelete = playlistDeleteDraft{}
		m.beginSearch("deleting " + draft.name)
		return m, m.deletePlaylistCmd(draft)
	case "n", "esc":
		return m.cancelPlaylistDelete()
	}
	return m, noop
}

func (m Model) cancelPlaylistDelete() (Model, tea.Cmd) {
	draft := m.playlistDelete
	m.playlistDelete = playlistDeleteDraft{}
	m.mode = draft.previous
	m.refreshTitle()
	if draft.index >= 0 && draft.index < len(m.list.Items()) {
		m.list.Select(draft.index)
	}
	m.status = "playlist deletion canceled"
	m.err = ""
	return m, noop
}

func (m Model) deletePlaylistCmd(draft playlistDeleteDraft) tea.Cmd {
	client, store := m.client, m.vaultStore
	return func() tea.Msg {
		msg := playlistDeletedMsg{destination: draft.destination, name: draft.name, index: draft.index}
		if draft.destination == playlistDeleteServer {
			if err := client.DeletePlaylist(context.Background(), draft.id); err != nil {
				return errMsg{err: fmt.Errorf("delete server playlist %s: %w", draft.name, err)}
			}
			playlists, err := client.Playlists(context.Background())
			if err != nil {
				return errMsg{err: fmt.Errorf("refresh server playlists: %w", err)}
			}
			msg.server = playlists
			return msg
		}
		if store == nil || !store.Unlocked() {
			return errMsg{err: fmt.Errorf("private vault is locked")}
		}
		if err := store.DeletePrivatePlaylist(draft.id); err != nil {
			return errMsg{err: fmt.Errorf("delete private playlist %s: %w", draft.name, err)}
		}
		playlists, err := store.PrivatePlaylists()
		if err != nil {
			return errMsg{err: fmt.Errorf("refresh private playlists: %w", err)}
		}
		msg.private = playlists
		return msg
	}
}

func (m Model) applyPlaylistDeleted(msg playlistDeletedMsg) Model {
	items := []list.Item{}
	if msg.destination == playlistDeleteServer {
		m.mode = modePlaylists
		for _, playlist := range msg.server {
			items = append(items, item{kind: "playlist", playlist: playlist})
		}
	} else {
		m.mode = modePrivatePlaylists
		for _, playlist := range msg.private {
			items = append(items, item{kind: "private_playlist", privatePlaylist: playlist})
		}
		if len(items) == 0 {
			items = append(items, item{kind: "empty", title: "No private playlists", desc: "press w from the queue to save one"})
		}
	}
	m.refreshTitle()
	m.list.SetItems(items)
	if len(items) > 0 {
		m.list.Select(min(msg.index, len(items)-1))
	}
	m.status = "deleted playlist: " + msg.name
	m.err = ""
	m.searching = false
	return m
}

func (m Model) playlistDeleteView(width int) string {
	destination := "Navidrome server"
	if m.playlistDelete.destination == playlistDeleteVault {
		destination = "private vault"
	}
	title := m.styles.header.Render("DELETE PLAYLIST")
	question := fmt.Sprintf("Delete %q from the %s?", m.playlistDelete.name, destination)
	warning := m.styles.error.Render("This cannot be undone.")
	help := m.styles.help.Render("[Y] yes, delete  [N/esc] cancel")
	return strings.Join([]string{title, "", ansi.Wordwrap(question, max(20, width-4), " /_-"), warning, "", help}, "\n")
}
