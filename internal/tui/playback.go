package tui

import (
	"time"

	"github.com/bprendie/subweazl/internal/audio"
	"github.com/bprendie/subweazl/internal/localstore"
	"github.com/bprendie/subweazl/internal/player"
	"github.com/bprendie/subweazl/internal/state"
	"github.com/bprendie/subweazl/internal/subsonic"
	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) play(track subsonic.Track) tea.Cmd {
	stream := m.client.StreamURL(track.ID)
	if err := m.player.Play(stream); err != nil {
		m.err = err.Error()
		return noop
	}
	m.playing = &track
	m.playSource = stream
	m.paused = false
	m.trackTitle = ""
	m.titlePoll = time.Time{}
	m.coverID = coverArtID(track)
	m.coverArt = nil
	m.coverErr = ""
	stateErr := m.saveLastPlayed(track)
	historyErr := m.recordSubsonicPlay(track)
	meterErr := m.startMeter(stream)
	m.status = "playing " + track.Title
	if meterErr != nil {
		m.err = "visualizer: " + meterErr.Error()
	} else if stateErr != nil {
		m.err = stateErr.Error()
	} else if historyErr != nil {
		m.err = historyErr.Error()
	} else {
		m.err = ""
	}
	return m.loadCoverArt(m.coverID)
}

func (m *Model) saveLastPlayed(track subsonic.Track) error {
	last := state.FromTrack(track)
	if last.ID == "" {
		return nil
	}
	m.appState.LastPlayed = &last
	return state.Save(m.appState)
}

func (m *Model) recordSubsonicPlay(track subsonic.Track) error {
	if m.vaultStore == nil || !m.vaultStore.Unlocked() {
		return nil
	}
	return m.vaultStore.AddPlayHistory(localstore.PlayHistoryRecord{
		Source:  localstore.SourceSubsonic,
		TrackID: track.ID,
		Payload: map[string]any{
			"id":       track.ID,
			"title":    track.Title,
			"artist":   track.Artist,
			"album":    track.Album,
			"album_id": track.AlbumID,
			"cover_id": coverArtID(track),
			"duration": track.Duration,
			"genre":    track.Genre,
			"year":     track.Year,
		},
	})
}

func (m *Model) togglePause() {
	paused, err := m.player.TogglePause()
	if err != nil {
		m.err = err.Error()
		return
	}
	m.paused = paused
	if paused {
		m.stopMeter()
		m.status = "paused"
		return
	}
	if m.playSource != "" {
		if err := m.startMeter(m.playSource); err != nil {
			m.err = "visualizer: " + err.Error()
		}
		m.status = "playing " + m.playingLabel()
	}
}

func (m *Model) stop() {
	m.player.Stop()
	m.stopMeter()
	m.playing = nil
	m.playSource = ""
	m.paused = false
	m.trackTitle = ""
	m.titlePoll = time.Time{}
	m.coverID = ""
	m.coverArt = nil
	m.coverErr = ""
	m.status = "stopped"
}

func (m Model) isPlaying() bool {
	return m.playing != nil
}

func (m Model) playingLabel() string {
	if m.playing != nil {
		return m.playing.Title
	}
	return ""
}

func (m *Model) startMeter(url string) error {
	m.stopMeter()
	meter, err := audio.StartMeter(url)
	if err != nil {
		m.energy = audio.Sample{}
		return err
	}
	m.meter = meter
	return nil
}

func (m *Model) stopMeter() {
	if m.meter != nil {
		m.meter.Stop()
	}
	m.meter = nil
	m.energy = audio.Sample{}
}

func fetchTitle(player *player.Player) tea.Cmd {
	return func() tea.Msg {
		title, err := player.Title()
		if err != nil {
			return nil
		}
		return titleMsg{title: title}
	}
}
