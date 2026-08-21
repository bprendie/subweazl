package tui

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bprendie/subweazl/internal/curator"
	"github.com/bprendie/subweazl/internal/llm"
	"github.com/bprendie/subweazl/internal/localstore"
	"github.com/bprendie/subweazl/internal/playqueue"
	"github.com/bprendie/subweazl/internal/subsonic"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

type llmQueueMsg struct {
	generation string
	result     curator.Result
	run        localstore.RecommendationRun
	request    curatorRequest
	playlist   subsonic.Playlist
	playlists  []subsonic.Playlist
}

type curatorProgressMsg struct {
	generation string
	track      subsonic.Track
	accepted   int
	total      int
}

type curatorPlaylistStartedMsg struct {
	generation string
	name       string
	playlist   subsonic.Playlist
	playlists  []subsonic.Playlist
}

func (m Model) generateLLMQueue() (Model, tea.Cmd) {
	return m.startCuratorChoice()
}

func (m Model) generateMoodPlaylist() (Model, tea.Cmd) {
	seed, ok := m.moodSeed()
	if !ok {
		m.err = "play or queue a track before creating Mood"
		return m, noop
	}
	return m.generateLLMCuration(moodCuratorRequest(seed))
}

func (m Model) generateLLMCuration(request curatorRequest) (Model, tea.Cmd) {
	if !m.cfg.LLMReady() {
		m.err = "llm curator is not configured"
		return m, noop
	}
	if m.vaultStore == nil || !m.vaultStore.Unlocked() {
		m.err = "private vault is locked"
		return m, noop
	}
	m.beginSearch("curating AI mix with llm")
	if request.Mode == curator.ModeMood {
		m.status = "curating Mood from " + request.Seed.Title
	}
	m.curating = true
	m.curatorStarted = time.Now()
	if m.curatorCancel != nil {
		m.curatorCancel()
	}
	m.curatorID = fmt.Sprintf("%d", time.Now().UnixNano())
	// Unbuffered delivery keeps final success/failure ordered behind every
	// accepted-track UI update.
	m.curatorEvents = make(chan tea.Msg)
	m.curatorTracks = nil
	m.curatorPlaylistID = ""
	m.curatorPlaylistName = request.PlaylistName
	m.activeCurator = request
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	m.curatorCancel = cancel
	spinnerCmd := m.spinner.Tick
	return m, tea.Batch(spinnerCmd, m.runLLMCurator(ctx, m.curatorID, m.curatorEvents, request), waitCuratorEvent(m.curatorEvents))
}

func (m Model) moodSeed() (subsonic.Track, bool) {
	if m.playing != nil && m.playing.ID != "" {
		return *m.playing, true
	}
	if track, ok := m.queue.Current(); ok {
		return track, true
	}
	return m.selectedTrack()
}

func (m Model) runLLMCurator(ctx context.Context, generation string, events chan tea.Msg, request curatorRequest) tea.Cmd {
	store := m.vaultStore
	cfg := m.cfg.LLM
	client := m.client
	return func() tea.Msg {
		defer close(events)
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
		preferred := map[string]bool{}
		playlistName := request.PlaylistName
		playlistLimit := request.Limit
		if request.Mode == curator.ModeMood {
			if request.Seed.ID == "" {
				return errMsg{err: fmt.Errorf("play a track before creating a Mood playlist")}
			}
			if similar, err := client.Similar(ctx, request.Seed, 80); err == nil {
				for _, track := range similar {
					preferred[track.ID] = true
				}
			}
		}
		progressive := make([]subsonic.Track, 0, playlistLimit)
		started := false
		if request.Destination == curatorDestinationServer && request.Mode == curator.ModeMood {
			playlist, saveErr := client.SaveOrReplacePlaylist(ctx, playlistName, []subsonic.Track{request.Seed})
			if saveErr != nil {
				return errMsg{err: fmt.Errorf("start server %s playlist: %w", playlistName, saveErr)}
			}
			playlists, listErr := client.Playlists(ctx)
			if listErr != nil {
				return errMsg{err: fmt.Errorf("refresh server playlists: %w", listErr)}
			}
			events <- curatorPlaylistStartedMsg{generation: generation, name: playlistName, playlist: playlist, playlists: playlists}
			started = true
		}
		result, err := curator.GenerateStreaming(ctx, llm.New(cfg), curator.Request{
			Seed:       request.Seed,
			Candidates: cached,
			RecentIDs:  recent,
			Preferred:  preferred,
			Limit:      playlistLimit,
			Mode:       request.Mode,
		}, func(track subsonic.Track, accepted, total int) error {
			progressive = append(progressive, track)
			if request.Destination == curatorDestinationServer && !started {
				playlist, saveErr := client.SaveOrReplacePlaylist(ctx, playlistName, progressive)
				if saveErr != nil {
					return saveErr
				}
				playlists, listErr := client.Playlists(ctx)
				if listErr != nil {
					return listErr
				}
				events <- curatorPlaylistStartedMsg{generation: generation, name: playlistName, playlist: playlist, playlists: playlists}
				started = true
			}
			events <- curatorProgressMsg{generation: generation, track: track, accepted: accepted, total: total}
			if request.Destination == curatorDestinationServer && accepted > 1 && (accepted%4 == 0 || accepted == total) {
				_, err := client.SaveOrReplacePlaylist(ctx, playlistName, progressive)
				return err
			}
			return nil
		})
		if err != nil {
			payload := curator.RunPayload(cfg.Provider, cfg.Model, result)
			payload["status"] = "failed"
			payload["error"] = err.Error()
			_, _ = store.SaveRecommendationRun(localstore.RecommendationRun{
				Provider: cfg.Provider,
				Model:    cfg.Model,
				TrackIDs: result.IDs,
				Status:   "failed",
				Payload:  payload,
			})
			if errors.Is(err, context.DeadlineExceeded) {
				err = fmt.Errorf("DJ-Weazl timed out; check the LLM server and try again")
			}
			return errMsg{err: err}
		}
		run := localstore.RecommendationRun{
			Provider: cfg.Provider,
			Model:    cfg.Model,
			TrackIDs: result.IDs,
			Status:   "complete",
			Payload:  curator.RunPayload(cfg.Provider, cfg.Model, result),
		}
		run, err = store.SaveRecommendationRun(run)
		if err != nil {
			return errMsg{err: err}
		}
		msg := llmQueueMsg{generation: generation, result: result, run: run, request: request}
		if request.Destination == curatorDestinationServer {
			msg.playlist, err = client.SaveOrReplacePlaylist(ctx, playlistName, result.Tracks)
			if err != nil {
				return errMsg{err: fmt.Errorf("save server %s playlist: %w", playlistName, err)}
			}
			msg.playlists, err = client.Playlists(ctx)
			if err != nil {
				return errMsg{err: fmt.Errorf("refresh server playlists: %w", err)}
			}
		}
		return msg
	}
}

func waitCuratorEvent(events <-chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-events
		if !ok {
			return nil
		}
		return msg
	}
}

func curatorPlaylistName(mode curator.Mode) string {
	if mode == curator.ModeMood {
		return "Mood"
	}
	return "AI Mix"
}

func curatorPlaylistLimit(mode curator.Mode) int {
	if mode == curator.ModeMood {
		return 20
	}
	return 40
}

func (m Model) applyLLMQueue(msg llmQueueMsg) Model {
	snapshot := playqueue.Snapshot{Tracks: msg.result.Tracks, Current: 0}
	if msg.request.Destination == curatorDestinationServer && m.playing != nil {
		current := -1
		for i, track := range snapshot.Tracks {
			if track.ID == m.playing.ID {
				current = i
				break
			}
		}
		if current >= 0 {
			snapshot.Current = current
		} else {
			snapshot.Tracks = append([]subsonic.Track{*m.playing}, snapshot.Tracks...)
			snapshot.Current = 0
		}
	}
	if msg.request.Destination == curatorDestinationServer {
		m.queue = playqueue.FromSnapshot(snapshot)
		m.queueTitle = msg.request.PlaylistName
		m.persistQueue()
	}
	if msg.request.Destination == curatorDestinationServer {
		if m.mode == modeTracks && m.playlistViewID == msg.playlist.ID {
			cursor := m.list.Index()
			m.list.SetItems(trackItems(msg.result.Tracks))
			if len(msg.result.Tracks) > 0 {
				m.list.Select(min(cursor, len(msg.result.Tracks)-1))
			}
		} else {
			items := make([]list.Item, 0, len(msg.playlists))
			for _, playlist := range msg.playlists {
				items = append(items, item{kind: "playlist", playlist: playlist})
			}
			m.clearNav()
			m.mode = modePlaylists
			m.refreshTitle()
			m.list.SetItems(items)
			for i, row := range m.list.Items() {
				if it, ok := row.(item); ok && it.kind == "playlist" && it.playlist.ID == msg.playlist.ID {
					m.list.Select(i)
					break
				}
			}
		}
	}
	m.status = fmt.Sprintf("DJ-Weazl curated %d tracks", len(msg.result.Tracks))
	if len(msg.result.Rejected) > 0 {
		m.status += fmt.Sprintf("; rejected %d invented ids", len(msg.result.Rejected))
	}
	if len(msg.result.Fallback) > 0 {
		m.status += fmt.Sprintf("; library-filled %d slots", len(msg.result.Fallback))
	}
	m.err = ""
	m.searching = false
	m.curating = false
	m.curatorCancel = nil
	m.curatorID = ""
	return m
}

func (m *Model) applyCuratorPlaylistStarted(msg curatorPlaylistStartedMsg) {
	m.curatorPlaylistID = msg.playlist.ID
	m.curatorPlaylistName = msg.name
	items := make([]list.Item, 0, len(msg.playlists))
	for _, playlist := range msg.playlists {
		items = append(items, item{kind: "playlist", playlist: playlist})
	}
	m.clearNav()
	m.mode = modePlaylists
	m.refreshTitle()
	m.list.SetItems(items)
	for i, row := range m.list.Items() {
		if it, ok := row.(item); ok && it.kind == "playlist" && it.playlist.ID == msg.playlist.ID {
			m.list.Select(i)
			break
		}
	}
	m.status = fmt.Sprintf("%s created on server; curating", msg.name)
}

func (m *Model) applyCuratorProgress(msg curatorProgressMsg) {
	seen := false
	for _, track := range m.curatorTracks {
		if track.ID == msg.track.ID {
			seen = true
			break
		}
	}
	if !seen {
		m.curatorTracks = append(m.curatorTracks, msg.track)
	}
	m.refreshStreamingCuratorView(msg.accepted)
	if m.activeCurator.Destination == curatorDestinationServer {
		m.spliceCuratorTracks()
	}
	m.status = fmt.Sprintf("curating %s %d/%d", m.curatorPlaylistName, msg.accepted, msg.total)
}

func (m *Model) refreshStreamingCuratorView(accepted int) {
	if m.mode == modePlaylists {
		cursor := m.list.Index()
		items := m.list.Items()
		for i, row := range items {
			it, ok := row.(item)
			if !ok || it.kind != "playlist" || it.playlist.ID != m.curatorPlaylistID {
				continue
			}
			it.playlist.Count = accepted
			items[i] = it
			m.list.SetItems(items)
			m.list.Select(cursor)
			return
		}
	}
	if m.mode == modeTracks && m.playlistViewID == m.curatorPlaylistID {
		cursor := m.list.Index()
		m.list.SetItems(trackItems(m.curatorTracks))
		if len(m.curatorTracks) > 0 {
			m.list.Select(min(cursor, len(m.curatorTracks)-1))
		}
	}
}

func (m *Model) spliceCuratorTracks() {
	if len(m.curatorTracks) == 0 {
		return
	}
	old := m.queue.Tracks()
	current := m.queue.CurrentIndex()
	if m.playing != nil {
		for i, track := range old {
			if track.ID == m.playing.ID {
				current = i
				break
			}
		}
	}
	var prefix []subsonic.Track
	if current >= 0 && current < len(old) {
		prefix = append(prefix, old[:current+1]...)
	} else if m.playing != nil {
		prefix = append(prefix, *m.playing)
		current = 0
	}
	seen := map[string]bool{}
	for _, track := range prefix {
		seen[track.ID] = true
	}
	combined := append([]subsonic.Track(nil), prefix...)
	for _, track := range m.curatorTracks {
		if track.ID != "" && !seen[track.ID] {
			combined = append(combined, track)
			seen[track.ID] = true
		}
	}
	start := current + 1
	if start < 0 {
		start = 0
	}
	for _, track := range old[start:] {
		if track.ID != "" && !seen[track.ID] {
			combined = append(combined, track)
			seen[track.ID] = true
		}
	}
	if len(prefix) == 0 {
		current = 0
	}
	m.queue.Replace(combined, current)
	m.queueTitle = m.curatorPlaylistName + " (building)"
	m.persistQueue()
}
