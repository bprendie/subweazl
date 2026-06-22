package tui

import (
	"fmt"
	"strings"

	"github.com/bprendie/subweazl/internal/localstore"
	"github.com/bprendie/subweazl/internal/subsonic"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

const (
	homeActionResume           = "resume"
	homeActionNewest           = "newest"
	homeActionPlaylists        = "playlists"
	homeActionRandom           = "random"
	homeActionSearch           = "search"
	homeActionQueue            = "queue"
	homeActionPrivatePlaylists = "private-playlists"
)

func (m *Model) showHome() {
	m.mode = modeHome
	m.clearNav()
	m.refreshTitle()
	m.list.SetItems(m.homeItems())
	m.status = "jump back in"
	m.err = ""
	m.searching = false
	m.input.Blur()
}

func (m Model) homeItems() []list.Item {
	items := []list.Item{}
	if m.appState.LastPlayed != nil && m.appState.LastPlayed.ID != "" {
		track := m.appState.LastPlayed.Track()
		items = append(items, item{
			kind:   "home",
			title:  "Resume " + track.Title,
			desc:   trackDescription(track),
			track:  track,
			action: homeActionResume,
		})
	}
	items = append(items, m.recentHomeItems()...)
	items = append(items,
		item{kind: "home", title: "Newest albums", desc: "browse recently added Subsonic albums", action: homeActionNewest},
		item{kind: "home", title: "Playlists", desc: "browse server playlists", action: homeActionPlaylists},
		item{kind: "home", title: "Random albums", desc: "shake loose a server discovery path", action: homeActionRandom},
		item{kind: "home", title: "Search", desc: "find tracks by song, artist, or album", action: homeActionSearch},
		item{kind: "home", title: "Last queue", desc: "restore or inspect the private queue", action: homeActionQueue},
		item{kind: "home", title: "Private playlists", desc: "load saved private queue playlists", action: homeActionPrivatePlaylists},
	)
	return items
}

func (m Model) recentHomeItems() []list.Item {
	if m.vaultStore == nil || !m.vaultStore.Unlocked() {
		return nil
	}
	entries, err := m.vaultStore.PlayHistory(3)
	if err != nil {
		return nil
	}
	items := make([]list.Item, 0, len(entries))
	for _, entry := range entries {
		if entry.Source != localstore.SourceSubsonic {
			continue
		}
		track := trackFromHistory(entry)
		if track.ID == "" {
			continue
		}
		items = append(items, item{
			kind:   "home",
			title:  "Recent " + track.Title,
			desc:   trackDescription(track),
			track:  track,
			action: homeActionResume,
		})
	}
	return items
}

func (m Model) handleHomeAction(it item) (Model, tea.Cmd) {
	switch it.action {
	case homeActionResume:
		if it.track.ID == "" {
			m.err = "no track is available to resume"
			return m, noop
		}
		return m, m.play(it.track)
	case homeActionNewest:
		m.beginSearch("loading newest albums")
		return m, m.loadNewest()
	case homeActionPlaylists:
		m.beginSearch("loading playlists")
		return m, m.loadPlaylists()
	case homeActionRandom:
		m.beginSearch("loading random albums")
		return m, m.loadRandomAlbums()
	case homeActionSearch:
		m.mode = modeSearch
		m.refreshTitle()
		m.input.Focus()
		m.status = "search server tracks"
		m.err = ""
		return m, noop
	case homeActionQueue:
		m.showQueue()
		return m, noop
	case homeActionPrivatePlaylists:
		m.showPrivatePlaylists()
		return m, noop
	default:
		m.err = "home action is unavailable"
		return m, noop
	}
}

func trackFromHistory(entry localstore.PlayHistoryEntry) subsonic.Track {
	track := subsonic.Track{ID: entry.TrackID}
	track.Title = stringPayload(entry.Payload, "title")
	track.Artist = stringPayload(entry.Payload, "artist")
	track.Album = stringPayload(entry.Payload, "album")
	track.AlbumID = stringPayload(entry.Payload, "album_id")
	track.CoverID = stringPayload(entry.Payload, "cover_id")
	track.Genre = stringPayload(entry.Payload, "genre")
	track.Duration = intPayload(entry.Payload, "duration")
	track.Year = intPayload(entry.Payload, "year")
	if track.Title == "" {
		track.Title = track.ID
	}
	return track
}

func stringPayload(payload map[string]any, key string) string {
	value, ok := payload[key]
	if !ok {
		return ""
	}
	s, ok := value.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(s)
}

func intPayload(payload map[string]any, key string) int {
	value, ok := payload[key]
	if !ok {
		return 0
	}
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	default:
		return 0
	}
}

func trackDescription(track subsonic.Track) string {
	parts := []string{}
	if track.Artist != "" {
		parts = append(parts, track.Artist)
	}
	if track.Album != "" {
		parts = append(parts, track.Album)
	}
	if len(parts) == 0 {
		return fmt.Sprintf("Subsonic track %s", track.ID)
	}
	return strings.Join(parts, "  ")
}
