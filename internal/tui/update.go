package tui

import (
	"fmt"
	"time"

	"github.com/bprendie/subweazl/internal/subsonic"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.resize(msg.Width, msg.Height)
	case tea.KeyMsg:
		next, cmd := m.handleKey(msg)
		m = next
		if cmd != nil {
			return m, cmd
		}
	case spinner.TickMsg:
		if m.searching {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}
	case loadedMsg:
		m.mode = msg.mode
		if msg.mode != modeStation {
			m.station = nil
		}
		m.refreshTitle()
		m.list.SetItems(msg.items)
		m.status = msg.status
		m.err = ""
		m.searching = false
		m.input.Blur()
	case errMsg:
		m.err = msg.err.Error()
		m.searching = false
	case stationMsg:
		m.station = &msg.playlist
		m.mode = modeStation
		m.refreshTitle()
		m.list.SetItems(trackItems(msg.tracks))
		m.status = fmt.Sprintf("saved station %q with %d tracks", msg.playlist.Name, len(msg.tracks))
		m.err = ""
		m.searching = false
		m.input.Blur()
	case renamedMsg:
		m.renameLoadedPlaylist(msg.id, msg.name)
		m.status = "renamed playlist"
		m.err = ""
		m.searching = false
	case setupSavedMsg:
		m.cfg = msg.cfg
		m.client = subsonic.New(msg.cfg.Server, msg.cfg.Username, msg.cfg.Password)
		m.status = msg.status
		m.err = ""
		m.searching = false
		if msg.cfg.Ready() {
			if err := m.prepareVault(); err != nil {
				m.err = err.Error()
			}
			m.refreshTitle()
			return m, tick()
		}
	case coverArtMsg:
		if msg.id != m.coverID {
			break
		}
		if msg.err != nil {
			m.coverArt = nil
			m.coverErr = msg.err.Error()
			break
		}
		m.coverArt = msg.img
		m.coverErr = ""
		if msg.img != nil {
			m.coverCache[msg.id] = msg.img
		}
	case titleMsg:
		if msg.title != "" {
			m.trackTitle = msg.title
		}
	case tickMsg:
		m.drainMeter()
		m.visualizer.Step(m.isPlaying() && !m.paused, m.energy)
		if m.isPlaying() && !m.paused && time.Time(msg).Sub(m.titlePoll) > 2*time.Second {
			m.titlePoll = time.Time(msg)
			cmds = append(cmds, fetchTitle(m.player))
		}
		cmds = append(cmds, tick())
	}
	next, cmd := m.updateFocused(msg)
	m = next
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

func (m Model) handleKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		m.stop()
		m.closeVaultStore()
		return m, tea.Quit
	}
	if m.mode == modeSetup {
		return m.handleSetupKey(msg)
	}
	if m.mode == modeVault {
		return m.handleVaultKey(msg)
	}
	if m.input.Focused() {
		switch msg.String() {
		case "enter":
			return m.handleEnter()
		case "esc":
			return m.back()
		}
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}
	switch msg.String() {
	case "q":
		m.stop()
		m.closeVaultStore()
		return m, tea.Quit
	case "h":
		m.showHome()
		return m, noop
	case "1":
		m.clearNav()
		m.beginSearch("loading newest albums")
		return m, m.loadNewest()
	case "2":
		m.clearNav()
		m.beginSearch("loading playlists")
		return m, m.loadPlaylists()
	case "3":
		m.clearNav()
		m.beginSearch("loading random albums")
		return m, m.loadRandomAlbums()
	case "4":
		m.showQueue()
		return m, noop
	case "/":
		m.pushNav()
		m.mode = modeSearch
		m.refreshTitle()
		m.input.Focus()
		return m, noop
	case "enter":
		return m.handleEnter()
	case "n":
		return m.playNext()
	case "p":
		return m.playPrevious()
	case "a":
		return m.enqueueSelected()
	case "x":
		return m.removeQueueSelection()
	case "c":
		return m.clearQueue()
	case "u":
		return m.moveQueueSelection(-1)
	case "d":
		return m.moveQueueSelection(1)
	case "r":
		return m.createStation()
	case "ctrl+r":
		return m.startRename()
	case "left", "esc":
		return m.back()
	case " ":
		m.togglePause()
	case "s":
		m.stop()
	}
	return m, nil
}

func (m Model) handleEnter() (Model, tea.Cmd) {
	if m.input.Focused() {
		if m.renaming != nil {
			name := m.input.Value()
			if name == "" {
				return m, nil
			}
			playlist := *m.renaming
			m.renaming = nil
			m.resetInput()
			m.beginSearch("renaming playlist")
			return m, m.renamePlaylist(playlist.ID, name)
		}
		q := m.input.Value()
		if q == "" {
			return m, nil
		}
		m.beginSearch("searching")
		return m, m.search(q)
	}
	if it, ok := m.list.SelectedItem().(item); ok {
		switch it.kind {
		case "home":
			return m.handleHomeAction(it)
		case "album":
			m.pushNav()
			m.beginSearch("loading album")
			return m, m.loadAlbum(it.album.ID)
		case "playlist":
			m.pushNav()
			m.beginSearch("loading playlist")
			return m, m.loadPlaylist(it.playlist.ID)
		case "song", "queue":
			return m.playSelectedTrack(it)
		default:
			return m, nil
		}
	}
	return m, nil
}

func (m Model) createStation() (Model, tea.Cmd) {
	seed, ok := m.selectedTrack()
	if !ok && m.playing != nil {
		seed, ok = *m.playing, true
	}
	if !ok {
		m.err = "select or play a song first"
		return m, noop
	}
	m.beginSearch("creating station")
	return m, m.createStationFrom(seed)
}

func (m Model) startRename() (Model, tea.Cmd) {
	if it, ok := m.list.SelectedItem().(item); ok && it.kind == "playlist" {
		playlist := it.playlist
		m.renaming = &playlist
		m.input.Prompt = "rename > "
		m.input.SetValue(playlist.Name)
		m.input.Focus()
		m.status = "enter playlist name"
		m.err = ""
		return m, noop
	}
	if m.station != nil {
		playlist := *m.station
		m.renaming = &playlist
		m.input.Prompt = "rename > "
		m.input.SetValue(playlist.Name)
		m.input.Focus()
		m.status = "enter station playlist name"
		m.err = ""
		return m, noop
	}
	m.err = "select a playlist to rename"
	return m, noop
}

func (m Model) selectedTrack() (subsonic.Track, bool) {
	it, ok := m.list.SelectedItem().(item)
	if !ok || (it.kind != "song" && it.kind != "queue") {
		return subsonic.Track{}, false
	}
	return it.track, true
}

func (m *Model) renameLoadedPlaylist(id, name string) {
	if m.station != nil && m.station.ID == id {
		m.station.Name = name
	}
	items := m.list.Items()
	for i, row := range items {
		it, ok := row.(item)
		if !ok || it.kind != "playlist" || it.playlist.ID != id {
			continue
		}
		it.playlist.Name = name
		items[i] = it
		m.list.SetItems(items)
		return
	}
}

func stationName(seed subsonic.Track) string {
	if seed.Artist == "" {
		return "Radio - " + seed.Title
	}
	return "Radio - " + seed.Artist + " - " + seed.Title
}

func (m *Model) beginSearch(status string) tea.Cmd {
	m.status = status
	m.err = ""
	m.searching = true
	return m.spinner.Tick
}
