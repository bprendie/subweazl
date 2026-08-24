package tui

import (
	"time"

	"github.com/bprendie/subweazl/internal/remote"
)

func (m *Model) EnableRemote() {
	m.remoteEnabled = true
	m.publishRemote(true)
}

func (m *Model) publishRemote(force ...bool) {
	if !m.remoteEnabled {
		return
	}
	if len(force) == 0 && time.Since(m.remotePublished) < time.Second {
		return
	}
	snapshot := remote.Snapshot{Running: true, State: "idle", PlaybackMode: m.playbackModeLabel()}
	if m.playing != nil {
		snapshot.State = "playing"
		if m.paused {
			snapshot.State = "paused"
		}
		snapshot.Title = m.playing.Title
		if m.trackTitle != "" {
			snapshot.Title = m.trackTitle
		}
		snapshot.Artist = m.playing.Artist
		snapshot.Album = m.playing.Album
		snapshot.Duration = m.playing.Duration
	}
	if remote.WriteSnapshot(snapshot) == nil {
		m.remotePublished = time.Now()
	}
}
