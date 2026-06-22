package tui

import (
	"fmt"
	"time"

	"github.com/bprendie/subweazl/internal/playqueue"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) showPrivatePlaylists() {
	m.mode = modePrivatePlaylists
	m.clearNav()
	m.refreshTitle()
	m.list.SetItems(m.privatePlaylistItems())
	m.status = "private vault playlists"
	m.err = ""
	m.searching = false
	m.input.Blur()
}

func (m Model) privatePlaylistItems() []list.Item {
	if m.vaultStore == nil || !m.vaultStore.Unlocked() {
		return []list.Item{item{kind: "empty", title: "Private playlists locked", desc: "unlock the vault to use private playlists"}}
	}
	playlists, err := m.vaultStore.PrivatePlaylists()
	if err != nil {
		return []list.Item{item{kind: "empty", title: "Private playlists unavailable", desc: err.Error()}}
	}
	items := make([]list.Item, 0, len(playlists))
	for _, playlist := range playlists {
		items = append(items, item{kind: "private_playlist", privatePlaylist: playlist})
	}
	if len(items) == 0 {
		items = append(items, item{kind: "empty", title: "No private playlists", desc: "press w from the queue to save one"})
	}
	return items
}

func (m Model) startSaveQueue() (Model, tea.Cmd) {
	if len(m.queue.Tracks()) == 0 {
		m.err = "queue is empty"
		return m, noop
	}
	m.savingQueue = true
	m.privateRenaming = ""
	m.input.Prompt = "save queue > "
	m.input.SetValue(defaultPrivatePlaylistName())
	m.input.Focus()
	m.status = "name private playlist"
	m.err = ""
	return m, noop
}

func (m Model) saveQueueAsPrivatePlaylist(name string) (Model, tea.Cmd) {
	if m.vaultStore == nil || !m.vaultStore.Unlocked() {
		m.err = "private vault is locked"
		return m, noop
	}
	playlist, err := m.vaultStore.SavePrivatePlaylist(name, m.queue.Snapshot())
	m.savingQueue = false
	m.resetInput()
	if err != nil {
		m.err = err.Error()
		return m, noop
	}
	m.showPrivatePlaylists()
	m.status = "saved private playlist: " + playlist.Name
	return m, noop
}

func (m Model) loadPrivatePlaylist(it item) (Model, tea.Cmd) {
	playlist := it.privatePlaylist
	if playlist.ID == "" {
		m.err = "select a private playlist"
		return m, noop
	}
	m.queue = playqueue.FromSnapshot(playlist.Snapshot())
	m.persistQueue()
	m.showQueue()
	m.status = fmt.Sprintf("loaded private playlist: %s", playlist.Name)
	return m, noop
}

func (m Model) startPrivatePlaylistRename() (Model, tea.Cmd) {
	it, ok := m.list.SelectedItem().(item)
	if !ok || it.kind != "private_playlist" {
		m.err = "select a private playlist to rename"
		return m, noop
	}
	m.privateRenaming = it.privatePlaylist.ID
	m.savingQueue = false
	m.input.Prompt = "rename private > "
	m.input.SetValue(it.privatePlaylist.Name)
	m.input.Focus()
	m.status = "enter private playlist name"
	m.err = ""
	return m, noop
}

func (m Model) renamePrivatePlaylist(name string) (Model, tea.Cmd) {
	if m.vaultStore == nil || !m.vaultStore.Unlocked() {
		m.err = "private vault is locked"
		return m, noop
	}
	id := m.privateRenaming
	m.privateRenaming = ""
	m.resetInput()
	if err := m.vaultStore.RenamePrivatePlaylist(id, name); err != nil {
		m.err = err.Error()
		return m, noop
	}
	m.showPrivatePlaylists()
	m.status = "renamed private playlist"
	return m, noop
}

func (m Model) deletePrivatePlaylist() (Model, tea.Cmd) {
	if m.mode != modePrivatePlaylists {
		return m, nil
	}
	it, ok := m.list.SelectedItem().(item)
	if !ok || it.kind != "private_playlist" {
		m.err = "select a private playlist to delete"
		return m, noop
	}
	if m.vaultStore == nil || !m.vaultStore.Unlocked() {
		m.err = "private vault is locked"
		return m, noop
	}
	if err := m.vaultStore.DeletePrivatePlaylist(it.privatePlaylist.ID); err != nil {
		m.err = err.Error()
		return m, noop
	}
	m.showPrivatePlaylists()
	m.status = "deleted private playlist"
	return m, noop
}

func defaultPrivatePlaylistName() string {
	return "Queue " + time.Now().Format("2006-01-02 15:04")
}
