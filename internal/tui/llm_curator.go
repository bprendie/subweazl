package tui

import (
	"context"
	"fmt"
	"time"

	"github.com/bprendie/subweazl/internal/curator"
	"github.com/bprendie/subweazl/internal/llm"
	"github.com/bprendie/subweazl/internal/localstore"
	"github.com/bprendie/subweazl/internal/playqueue"
	tea "github.com/charmbracelet/bubbletea"
)

type llmQueueMsg struct {
	result curator.Result
	run    localstore.RecommendationRun
}

func (m Model) generateLLMQueue() (Model, tea.Cmd) {
	if !m.cfg.LLMReady() {
		m.err = "llm curator is not configured"
		return m, noop
	}
	if m.vaultStore == nil || !m.vaultStore.Unlocked() {
		m.err = "private vault is locked"
		return m, noop
	}
	m.beginSearch("curating queue with llm")
	return m, m.runLLMCurator()
}

func (m Model) runLLMCurator() tea.Cmd {
	store := m.vaultStore
	cfg := m.cfg.LLM
	seed := m.recommendationSeed()
	return func() tea.Msg {
		cached, err := store.CachedSubsonicTracks(0)
		if err != nil {
			return errMsg{err: err}
		}
		if len(cached) == 0 {
			return errMsg{err: fmt.Errorf("sync the Subsonic cache first")}
		}
		recent, err := store.RecentSubsonicTrackIDs(100)
		if err != nil {
			return errMsg{err: err}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		result, err := curator.Generate(ctx, llm.New(cfg), curator.Request{
			Seed:       seed,
			Candidates: cached,
			RecentIDs:  recent,
			Limit:      40,
		})
		if err != nil {
			return errMsg{err: err}
		}
		run := localstore.RecommendationRun{
			Provider: cfg.Provider,
			Model:    cfg.Model,
			TrackIDs: result.IDs,
			Payload:  curator.RunPayload(cfg.Provider, cfg.Model, result),
		}
		run, err = store.SaveRecommendationRun(run)
		if err != nil {
			return errMsg{err: err}
		}
		return llmQueueMsg{result: result, run: run}
	}
}

func (m Model) applyLLMQueue(msg llmQueueMsg) Model {
	m.queue = playqueue.FromSnapshot(playqueue.Snapshot{Tracks: msg.result.Tracks, Current: 0})
	m.persistQueue()
	m.showQueue()
	m.status = fmt.Sprintf("llm curated %d-track queue", len(msg.result.Tracks))
	if len(msg.result.Rejected) > 0 {
		m.status += fmt.Sprintf("; rejected %d invented ids", len(msg.result.Rejected))
	}
	m.err = ""
	m.searching = false
	return m
}
