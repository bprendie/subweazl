package tui

import (
	"fmt"
	"strings"

	"github.com/bprendie/subweazl/internal/localstore"
	"github.com/bprendie/subweazl/internal/playqueue"
	"github.com/bprendie/subweazl/internal/recommend"
	"github.com/bprendie/subweazl/internal/subsonic"
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) generateRecommendedQueue() (Model, tea.Cmd) {
	if m.vaultStore == nil || !m.vaultStore.Unlocked() {
		m.err = "private vault is locked"
		return m, noop
	}
	cached, err := m.vaultStore.CachedSubsonicTracks(0)
	if err != nil {
		m.err = err.Error()
		return m, noop
	}
	if len(cached) == 0 {
		m.err = "sync the Subsonic cache first"
		return m, noop
	}
	recent, err := m.vaultStore.RecentSubsonicTrackIDs(100)
	if err != nil {
		m.err = err.Error()
		return m, noop
	}
	seed := m.recommendationSeed()
	result := recommend.Generate(seed, cached, recent, 40)
	if len(result.Tracks) == 0 {
		m.err = "no cached recommendation candidates"
		return m, noop
	}
	m.queue = playqueue.FromSnapshot(playqueue.Snapshot{Tracks: result.Tracks, Current: 0})
	m.persistQueue()
	if _, err := m.vaultStore.SaveRecommendationRecipe(recommendationRecipe(seed, result)); err != nil {
		m.err = err.Error()
		return m, noop
	}
	m.showQueue()
	m.status = fmt.Sprintf("generated %d-track queue: %s", len(result.Tracks), strings.Join(result.Rules, ", "))
	m.err = ""
	return m, noop
}

func (m Model) recommendationSeed() subsonic.Track {
	if track, ok := m.selectedOrPlayingTrack(); ok {
		return track
	}
	if m.appState.LastPlayed != nil {
		return m.appState.LastPlayed.Track()
	}
	return subsonic.Track{}
}

func recommendationRecipe(seed subsonic.Track, result recommend.Result) localstore.RecommendationRecipe {
	ids := make([]string, 0, len(result.Tracks))
	for _, track := range result.Tracks {
		ids = append(ids, track.ID)
	}
	name := "deterministic queue"
	if seed.Title != "" {
		name = "queue from " + seed.Title
	}
	return localstore.RecommendationRecipe{
		Name:     name,
		SeedID:   seed.ID,
		Rules:    append([]string(nil), result.Rules...),
		TrackIDs: ids,
		Payload: map[string]any{
			"engine": "deterministic-cache-v1",
		},
	}
}
