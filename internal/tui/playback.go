package tui

import (
	"strings"
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
	m.localPlay = nil
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

func (m *Model) playLocal(track localTrack) tea.Cmd {
	if track.Missing {
		m.err = "local track is missing"
		return noop
	}
	if track.Path == "" {
		m.err = "local track path is empty"
		return noop
	}
	if err := m.player.Play(track.Path); err != nil {
		m.err = err.Error()
		return noop
	}
	m.playing = nil
	m.localPlay = &track
	m.playSource = track.Path
	m.paused = false
	m.trackTitle = ""
	m.titlePoll = time.Time{}
	m.coverID = ""
	m.coverArt = nil
	m.coverErr = ""
	historyErr := m.recordLocalPlay(track)
	meterErr := m.startMeter(track.Path)
	m.status = "playing " + localTrackLabel(track)
	if meterErr != nil {
		m.err = "visualizer: " + meterErr.Error()
	} else if historyErr != nil {
		m.err = historyErr.Error()
	} else {
		m.err = ""
	}
	return noop
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
	if err := m.ensureLocalStore(); err != nil {
		return err
	}
	if !m.localStore.Unlocked() {
		return nil
	}
	return m.localStore.AddPlayHistory(localstore.PlayHistoryRecord{
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

func (m *Model) recordLocalPlay(track localTrack) error {
	if err := m.ensureLocalStore(); err != nil {
		return err
	}
	if !m.localStore.Unlocked() {
		return nil
	}
	return m.localStore.AddPlayHistory(localstore.PlayHistoryRecord{
		Source:  localstore.SourceLocal,
		TrackID: track.ID,
		Payload: map[string]any{
			"id":      track.ID,
			"title":   track.Title,
			"artist":  track.Artist,
			"album":   track.Album,
			"path":    track.Path,
			"missing": track.Missing,
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
	m.localPlay = nil
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
	return m.playing != nil || m.localPlay != nil
}

func (m Model) playingLabel() string {
	if m.playing != nil {
		return m.playing.Title
	}
	if m.localPlay != nil {
		return localTrackLabel(*m.localPlay)
	}
	return ""
}

func localTrackLabel(track localTrack) string {
	if track.Artist != "" && track.Title != "" {
		return track.Artist + " - " + track.Title
	}
	if track.Title != "" {
		return track.Title
	}
	return track.Path
}

func localTrackDescription(track localTrack) string {
	var parts []string
	if track.Artist != "" {
		parts = append(parts, track.Artist)
	}
	if track.Album != "" {
		parts = append(parts, track.Album)
	}
	if track.Missing {
		parts = append(parts, "missing")
	}
	if len(parts) == 0 {
		return "local file"
	}
	return strings.Join(parts, "  ")
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
