package tui

import (
	"context"
	"errors"
	"fmt"
	"strings"
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
	generation       string
	result           curator.Result
	run              localstore.RecommendationRun
	request          curatorRequest
	playlist         subsonic.Playlist
	playlists        []subsonic.Playlist
	privatePlaylist  localstore.PrivatePlaylist
	privatePlaylists []localstore.PrivatePlaylist
}

type curatorProgressMsg struct {
	generation string
	track      subsonic.Track
	accepted   int
	total      int
}

type curatorStageMsg struct {
	generation string
	status     string
}

type curatorPlaylistStartedMsg struct {
	generation string
	name       string
	playlist   subsonic.Playlist
	playlists  []subsonic.Playlist
}

type curatorPrivateStartedMsg struct {
	generation string
	name       string
	playlist   localstore.PrivatePlaylist
	playlists  []localstore.PrivatePlaylist
}

type curatorAnchorsStartedMsg struct {
	generation string
	name       string
	playlist   localstore.PrivatePlaylist
	playlists  []localstore.PrivatePlaylist
	anchors    []subsonic.Track
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
		var err error
		cfg, err = llm.ResolveConfig(ctx, cfg)
		if err != nil {
			return errMsg{err: err}
		}
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
		llmClient := llm.New(cfg)
		var groundingAnchors []subsonic.Track
		var groundingIntent curator.Intent
		groundingMethod := "mood_seed"
		candidateSnapshotID := curator.CandidateSnapshotID(cached)
		playlistName := request.PlaylistName
		playlistLimit := request.Limit
		var priorPrivate *localstore.PrivatePlaylist
		var partialPrivateID string
		published := false
		if request.Destination == curatorDestinationVault {
			playlists, listErr := store.PrivatePlaylists()
			if listErr != nil {
				return errMsg{err: listErr}
			}
			for _, playlist := range playlists {
				if strings.EqualFold(strings.TrimSpace(playlist.Name), strings.TrimSpace(playlistName)) {
					copy := playlist
					priorPrivate = &copy
					break
				}
			}
		}
		rollbackPrivate := func() {
			if request.Destination != curatorDestinationVault || partialPrivateID == "" {
				return
			}
			if priorPrivate != nil {
				_, _ = store.SaveOrReplacePrivatePlaylist(priorPrivate.Name, priorPrivate.Snapshot())
			} else {
				_ = store.DeletePrivatePlaylist(partialPrivateID)
			}
		}
		if request.Mode == curator.ModeMood {
			if request.Seed.ID == "" {
				return errMsg{err: fmt.Errorf("play a track before creating a Mood playlist")}
			}
			if similar, err := client.Similar(ctx, request.Seed, 80); err == nil {
				for _, track := range similar {
					preferred[track.ID] = true
				}
			}
		} else {
			query := strings.TrimSpace(request.Prompt)
			intent := curator.Intent{Purpose: "cohesive discovery without taxing the gig"}
			cachedIDs := make(map[string]bool, len(cached))
			for _, candidate := range cached {
				cachedIDs[candidate.Track.ID] = true
			}
			events <- curatorStageMsg{generation: generation, status: "grounding three playable anchors"}
			primary, fastGrounded := curator.DiscoveryPrimarySeed(cached, recent)
			if query != "" {
				primary, fastGrounded = curator.ResolvePrimarySeed(query, cached, recent)
			}
			var anchors []subsonic.Track
			var authoritativeNeighborhood []subsonic.Track
			if fastGrounded {
				groundingMethod = "deterministic_cache_and_navidrome"
				similar, similarErr := client.Similar(ctx, primary, 160)
				if similarErr == nil {
					anchors, similarErr = curator.SelectDeterministicAnchors(primary, similar, cachedIDs, recent)
					authoritativeNeighborhood = similar
				}
				fastGrounded = similarErr == nil
			}
			if !fastGrounded && query != "" {
				groundingMethod = "llm_fallback"
				events <- curatorStageMsg{generation: generation, status: "interpreting your request"}
				intent, err = curator.InterpretIntent(ctx, llmClient, query)
				if err != nil {
					return errMsg{err: fmt.Errorf("interpret AI mix request: %w", err)}
				}
				groundingIntent = intent
				seen := map[string]bool{}
				var anchorCandidates []subsonic.Track
				for _, term := range intent.SearchTerms {
					matches, searchErr := store.CachedSubsonicSearch(term, 40)
					if searchErr != nil {
						return errMsg{err: fmt.Errorf("search anchor candidates: %w", searchErr)}
					}
					for _, track := range matches {
						if !seen[track.ID] {
							seen[track.ID] = true
							anchorCandidates = append(anchorCandidates, track)
						}
					}
				}
				events <- curatorStageMsg{generation: generation, status: fmt.Sprintf("judging %d fallback anchor candidates", len(anchorCandidates))}
				anchors, err = curator.SelectAnchors(ctx, llmClient, query, intent, anchorCandidates)
				if err != nil {
					return errMsg{err: err}
				}
				authoritativeNeighborhood, err = client.Similar(ctx, anchors[0], 200)
				if err != nil {
					return errMsg{err: fmt.Errorf("find music similar to authoritative seed %s: %w", anchors[0].Title, err)}
				}
			} else if !fastGrounded {
				return errMsg{err: errors.New("could not establish three discovery anchors")}
			}
			groundingAnchors = anchors
			playlist, saveErr := store.SaveOrReplacePrivatePlaylist(playlistName, playqueue.Snapshot{Tracks: anchors, Current: 0})
			if saveErr != nil {
				return errMsg{err: fmt.Errorf("publish anchor playlist: %w", saveErr)}
			}
			partialPrivateID = playlist.ID
			playlists, listErr := store.PrivatePlaylists()
			if listErr != nil {
				rollbackPrivate()
				return errMsg{err: fmt.Errorf("refresh anchor playlist: %w", listErr)}
			}
			events <- curatorAnchorsStartedMsg{generation: generation, name: playlistName, playlist: playlist, playlists: playlists, anchors: anchors}
			published = true
			events <- curatorStageMsg{generation: generation, status: "building from authoritative seed " + anchors[0].Artist + " - " + anchors[0].Title}
			crate := curator.MergeSimilarityCrate(anchors, [][]subsonic.Track{authoritativeNeighborhood})
			cachedByID := make(map[string]localstore.CachedTrack, len(cached))
			for _, candidate := range cached {
				cachedByID[candidate.Track.ID] = candidate
			}
			grounded := make([]localstore.CachedTrack, 0, len(crate))
			for _, track := range crate {
				candidate, ok := cachedByID[track.ID]
				if !ok {
					continue
				}
				grounded = append(grounded, candidate)
				preferred[track.ID] = true
			}
			if len(grounded) < playlistLimit {
				return errMsg{err: preservedPartialCuratorError(fmt.Errorf("authoritative seed produced only %d cached candidates; need %d", len(grounded), playlistLimit), len(anchors), playlistLimit)}
			}
			cached = grounded
			candidateSnapshotID = curator.CandidateSnapshotID(cached)
			events <- curatorStageMsg{generation: generation, status: fmt.Sprintf("curating a grounded crate of %d tracks", len(grounded))}
		}
		progressive := append([]subsonic.Track(nil), groundingAnchors...)
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
		result, err := curator.GenerateStreaming(ctx, llmClient, curator.Request{
			Seed:       request.Seed,
			Candidates: cached,
			RecentIDs:  recent,
			Preferred:  preferred,
			Limit:      playlistLimit,
			Mode:       request.Mode,
			UserPrompt: request.Prompt,
			Anchors:    authoritativeGrounding(groundingAnchors),
			Initial:    groundingAnchors,
		}, func(track subsonic.Track, accepted, total int) error {
			progressive = append(progressive, track)
			if request.Destination == curatorDestinationVault {
				playlist, saveErr := store.SaveOrReplacePrivatePlaylist(playlistName, playqueue.Snapshot{Tracks: progressive, Current: 0})
				if saveErr != nil {
					return saveErr
				}
				partialPrivateID = playlist.ID
				if accepted == 1 {
					playlists, listErr := store.PrivatePlaylists()
					if listErr != nil {
						return listErr
					}
					events <- curatorPrivateStartedMsg{generation: generation, name: playlistName, playlist: playlist, playlists: playlists}
				}
			}
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
			if !published {
				rollbackPrivate()
			}
			payload := enrichedCuratorRunPayload(cfg.Provider, cfg.Model, result, request, groundingMethod, groundingIntent, groundingAnchors, candidateSnapshotID, len(cached))
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
			if published {
				err = preservedPartialCuratorError(err, len(result.IDs), playlistLimit)
			}
			return errMsg{err: err}
		}
		run := localstore.RecommendationRun{
			Provider: cfg.Provider,
			Model:    cfg.Model,
			TrackIDs: result.IDs,
			Status:   "complete",
			Payload:  enrichedCuratorRunPayload(cfg.Provider, cfg.Model, result, request, groundingMethod, groundingIntent, groundingAnchors, candidateSnapshotID, len(cached)),
		}
		run, err = store.SaveRecommendationRun(run)
		if err != nil {
			if !published {
				rollbackPrivate()
			}
			if published {
				err = preservedPartialCuratorError(err, len(result.IDs), playlistLimit)
			}
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
		if request.Destination == curatorDestinationVault {
			msg.privatePlaylist, err = store.SaveOrReplacePrivatePlaylist(playlistName, playqueue.Snapshot{Tracks: result.Tracks, Current: 0})
			if err != nil {
				if !published {
					rollbackPrivate()
				}
				err = fmt.Errorf("save private %s playlist: %w", playlistName, err)
				if published {
					err = preservedPartialCuratorError(err, len(result.IDs), playlistLimit)
				}
				return errMsg{err: err}
			}
			msg.privatePlaylists, err = store.PrivatePlaylists()
			if err != nil {
				if !published {
					rollbackPrivate()
				}
				err = fmt.Errorf("refresh private playlists: %w", err)
				if published {
					err = preservedPartialCuratorError(err, len(result.IDs), playlistLimit)
				}
				return errMsg{err: err}
			}
		}
		return msg
	}
}

func discoveryAnchorCandidates(cached []localstore.CachedTrack, limit int) []subsonic.Track {
	var newest, backNine []subsonic.Track
	for _, candidate := range cached {
		if candidate.New {
			newest = append(newest, candidate.Track)
		} else {
			backNine = append(backNine, candidate.Track)
		}
	}
	out := make([]subsonic.Track, 0, min(limit, len(cached)))
	for i, j := 0, 0; len(out) < limit && (i < len(newest) || j < len(backNine)); {
		if i < len(newest) {
			out = append(out, newest[i])
			i++
		}
		if len(out) == limit {
			break
		}
		if j < len(backNine) {
			out = append(out, backNine[j])
			j++
		}
	}
	return out
}

func anchorSummary(anchors []subsonic.Track) string {
	names := make([]string, 0, len(anchors))
	for _, anchor := range anchors {
		label := strings.TrimSpace(anchor.Artist + " - " + anchor.Title)
		if label != "-" && label != "" {
			names = append(names, label)
		}
	}
	return strings.Join(names, ", ")
}

func authoritativeGrounding(anchors []subsonic.Track) []subsonic.Track {
	if len(anchors) == 0 {
		return nil
	}
	return anchors[:1]
}

func preservedPartialCuratorError(err error, accepted, total int) error {
	return fmt.Errorf("curation stopped at %d/%d; playable partial playlist preserved: %w", accepted, total, err)
}

func enrichedCuratorRunPayload(provider, model string, result curator.Result, request curatorRequest, method string, intent curator.Intent, launchTracks []subsonic.Track, candidateSnapshotID string, candidateCount int) map[string]any {
	payload := curator.RunPayload(provider, model, result)
	payload["curator_mode"] = request.Mode
	payload["user_prompt"] = request.Prompt
	payload["grounding_method"] = method
	payload["interpreted_intent"] = intent
	payload["candidate_snapshot_id"] = candidateSnapshotID
	payload["candidate_count"] = candidateCount
	if len(launchTracks) > 0 {
		payload["authoritative_seed"] = launchTracks[0]
		payload["launch_tracks"] = launchTracks
	}
	return payload
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
	if msg.request.Destination == curatorDestinationVault {
		m.queueTitle = msg.request.PlaylistName
		m.persistQueue()
		items := make([]list.Item, 0, len(msg.privatePlaylists))
		for _, playlist := range msg.privatePlaylists {
			items = append(items, item{kind: "private_playlist", privatePlaylist: playlist})
		}
		m.clearNav()
		m.mode = modePrivatePlaylists
		m.refreshTitle()
		m.list.SetItems(items)
		for i, row := range m.list.Items() {
			if it, ok := row.(item); ok && it.kind == "private_playlist" && it.privatePlaylist.ID == msg.privatePlaylist.ID {
				m.list.Select(i)
				break
			}
		}
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

func (m *Model) applyCuratorPrivateStarted(msg curatorPrivateStartedMsg) {
	m.curatorPlaylistID = msg.playlist.ID
	m.curatorPlaylistName = msg.name
	items := make([]list.Item, 0, len(msg.playlists))
	for _, playlist := range msg.playlists {
		items = append(items, item{kind: "private_playlist", privatePlaylist: playlist})
	}
	m.clearNav()
	m.mode = modePrivatePlaylists
	m.refreshTitle()
	m.list.SetItems(items)
	for i, row := range m.list.Items() {
		if it, ok := row.(item); ok && it.kind == "private_playlist" && it.privatePlaylist.ID == msg.playlist.ID {
			m.list.Select(i)
			break
		}
	}
	m.status = fmt.Sprintf("%s created in private vault; curating", msg.name)
}

func (m Model) applyCuratorAnchorsStarted(msg curatorAnchorsStartedMsg) (Model, tea.Cmd) {
	m.applyCuratorPrivateStarted(curatorPrivateStartedMsg{
		generation: msg.generation,
		name:       msg.name,
		playlist:   msg.playlist,
		playlists:  msg.playlists,
	})
	m.curatorTracks = append([]subsonic.Track(nil), msg.anchors...)
	m.queue.Replace(msg.anchors, 0)
	m.queueTitle = msg.name + " (building)"
	m.queueSourceID = msg.playlist.ID
	m.resetPlaybackTraversal()
	m.persistQueue()
	m.status = fmt.Sprintf("playing anchor 1/3; curating %s 3/40", msg.name)
	if len(msg.anchors) == 0 {
		return m, noop
	}
	return m, m.play(msg.anchors[0])
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
	if m.activeCurator.Destination == curatorDestinationServer || (m.activeCurator.Destination == curatorDestinationVault && len(m.curatorTracks) >= 3) {
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
	if m.mode == modePrivatePlaylists {
		cursor := m.list.Index()
		items := m.list.Items()
		for i, row := range items {
			it, ok := row.(item)
			if !ok || it.kind != "private_playlist" || it.privatePlaylist.ID != m.curatorPlaylistID {
				continue
			}
			it.privatePlaylist.Tracks = append([]subsonic.Track(nil), m.curatorTracks...)
			it.privatePlaylist.Current = 0
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
