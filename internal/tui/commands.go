package tui

import (
	"context"
	"fmt"

	"github.com/bprendie/subweazl/internal/subsonic"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) loadNewest() tea.Cmd {
	return func() tea.Msg {
		albums, err := m.client.Newest(context.Background())
		if err != nil {
			return errMsg{err}
		}
		items := albumItems(albums)
		return loadedMsg{items: items, mode: modeNewest, status: fmt.Sprintf("loaded %d albums", len(items))}
	}
}

func (m Model) loadRandomAlbums() tea.Cmd {
	return func() tea.Msg {
		albums, err := m.client.RandomAlbums(context.Background())
		if err != nil {
			return errMsg{err}
		}
		items := albumItems(albums)
		return loadedMsg{items: items, mode: modeRandomAlbums, status: fmt.Sprintf("loaded %d random albums", len(items))}
	}
}

func (m Model) loadPlaylists() tea.Cmd {
	return func() tea.Msg {
		playlists, err := m.client.Playlists(context.Background())
		if err != nil {
			return errMsg{err}
		}
		items := make([]list.Item, 0, len(playlists))
		for _, playlist := range playlists {
			items = append(items, item{kind: "playlist", playlist: playlist})
		}
		return loadedMsg{items: items, mode: modePlaylists, status: fmt.Sprintf("loaded %d playlists", len(items))}
	}
}

func albumItems(albums []subsonic.Album) []list.Item {
	items := make([]list.Item, 0, len(albums))
	for _, album := range albums {
		items = append(items, item{kind: "album", album: album})
	}
	return items
}

func (m Model) loadAlbum(id string) tea.Cmd {
	return func() tea.Msg {
		tracks, err := m.client.Album(context.Background(), id)
		return tracksMsg(tracks, err, "album")
	}
}

func (m Model) loadPlaylist(id string) tea.Cmd {
	return func() tea.Msg {
		tracks, err := m.client.Playlist(context.Background(), id)
		return tracksMsg(tracks, err, "playlist")
	}
}

func (m Model) search(query string) tea.Cmd {
	return func() tea.Msg {
		tracks, err := m.client.Search(context.Background(), query)
		return tracksMsg(tracks, err, "search")
	}
}

func (m Model) createStationFrom(seed subsonic.Track) tea.Cmd {
	return func() tea.Msg {
		tracks, err := m.client.Similar(context.Background(), seed, 50)
		if err != nil {
			return errMsg{err}
		}
		name := stationName(seed)
		playlist, err := m.client.CreatePlaylist(context.Background(), name, tracks)
		if err != nil {
			return errMsg{err}
		}
		if playlist.Name == "" {
			playlist.Name = name
		}
		return stationMsg{playlist: playlist, tracks: tracks}
	}
}

func (m Model) renamePlaylist(id, name string) tea.Cmd {
	return func() tea.Msg {
		if err := m.client.RenamePlaylist(context.Background(), id, name); err != nil {
			return errMsg{err}
		}
		return renamedMsg{id: id, name: name}
	}
}

func tracksMsg(tracks []subsonic.Track, err error, label string) tea.Msg {
	if err != nil {
		return errMsg{err}
	}
	items := trackItems(tracks)
	nextMode := modeTracks
	if label == "search" {
		nextMode = modeSearch
	}
	return loadedMsg{items: items, mode: nextMode, status: fmt.Sprintf("loaded %d %s tracks", len(items), label)}
}

func trackItems(tracks []subsonic.Track) []list.Item {
	items := make([]list.Item, 0, len(tracks))
	for _, track := range tracks {
		items = append(items, item{kind: "song", track: track})
	}
	return items
}

func oneItem(it item) []list.Item {
	return []list.Item{it}
}
