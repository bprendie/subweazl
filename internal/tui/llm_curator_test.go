package tui

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/bprendie/subweazl/internal/curator"
	"github.com/bprendie/subweazl/internal/localstore"
	"github.com/bprendie/subweazl/internal/subsonic"
	tea "github.com/charmbracelet/bubbletea"
)

func TestGenerateLLMQueueOpensChooserBeforeConfigValidation(t *testing.T) {
	m := newHomeTestModel(t)
	got, cmd := m.generateLLMQueue()
	if cmd == nil {
		t.Fatal("expected noop command")
	}
	if got.mode != modeCuratorChoice || got.err != "" {
		t.Fatalf("mode=%v err=%q", got.mode, got.err)
	}
}

func TestMoodHasNoAuthoritativeAIMixGroundingAndDoesNotPanic(t *testing.T) {
	if got := authoritativeGrounding(nil); got != nil {
		t.Fatalf("grounding=%v, want nil", got)
	}
	anchors := []subsonic.Track{testTrack("primary"), testTrack("launch")}
	got := authoritativeGrounding(anchors)
	if len(got) != 1 || got[0].ID != "primary" {
		t.Fatalf("grounding=%v", got)
	}
}

func TestApplyLLMQueuePersistsQueue(t *testing.T) {
	m := newHomeTestModel(t)
	msg := llmQueueMsg{
		request:   curatorRequest{Mode: curator.ModeAIMix, Destination: curatorDestinationServer, PlaylistName: "AI Mix", Limit: 40},
		playlist:  subsonic.Playlist{ID: "ai-mix", Name: "AI Mix", Count: 2},
		playlists: []subsonic.Playlist{{ID: "ai-mix", Name: "AI Mix", Count: 2}},
		result: curator.Result{
			Tracks: []subsonic.Track{testTrack("a"), testTrack("b")},
			IDs:    []string{"a", "b"},
		},
		run: localstore.RecommendationRun{ID: "run-a"},
	}
	m = m.applyLLMQueue(msg)
	if m.mode != modePlaylists {
		t.Fatalf("mode = %v", m.mode)
	}
	if ids := queueIDs(m); len(ids) != 2 || ids[0] != "a" || ids[1] != "b" {
		t.Fatalf("queue ids = %#v", ids)
	}
	snapshot, ok, err := m.vaultStore.QueueSnapshot()
	if err != nil || !ok || len(snapshot.Tracks) != 2 {
		t.Fatalf("snapshot ok=%v err=%v snapshot=%#v", ok, err, snapshot)
	}
}

func TestMoodSeedPrefersPlayingThenQueuedTrack(t *testing.T) {
	m := newHomeTestModel(t)
	if seed, ok := m.moodSeed(); ok || seed.ID != "" {
		t.Fatalf("idle seed=%#v ok=%v", seed, ok)
	}
	m.queue.Replace([]subsonic.Track{testTrack("queued")}, 0)
	seed, ok := m.moodSeed()
	if !ok || seed.ID != "queued" {
		t.Fatalf("queued seed=%#v ok=%v", seed, ok)
	}
	playing := testTrack("teenagers")
	playing.Title = "Teenagers"
	m.playing = &playing
	seed, ok = m.moodSeed()
	if !ok || seed.ID != "teenagers" {
		t.Fatalf("playing seed=%#v ok=%v", seed, ok)
	}
}

func TestApplyMoodQueueFollowsNamedPlaylist(t *testing.T) {
	m := newHomeTestModel(t)
	seed := testTrack("teenagers")
	seed.Title = "Teenagers"
	m.playing = &seed
	m.paused = false
	m.queue.Replace([]subsonic.Track{testTrack("stale")}, 0)
	m = m.applyLLMQueue(llmQueueMsg{
		request:   moodCuratorRequest(seed),
		playlist:  subsonic.Playlist{ID: "mood-id", Name: "Mood", Count: 2},
		playlists: []subsonic.Playlist{{ID: "other", Name: "Other"}, {ID: "mood-id", Name: "Mood", Count: 2}},
		result: curator.Result{
			Tracks: []subsonic.Track{seed, testTrack("a")},
			IDs:    []string{seed.ID, "a"},
		},
	})
	selected, _ := m.list.SelectedItem().(item)
	if m.mode != modePlaylists || m.queueTitle != "Mood" || m.list.Title != "playlists" || selected.playlist.Name != "Mood" {
		t.Fatalf("mode=%v queueTitle=%q listTitle=%q", m.mode, m.queueTitle, m.list.Title)
	}
	if m.playing == nil || m.playing.ID != seed.ID || m.paused {
		t.Fatalf("Mood interrupted playback: playing=%#v paused=%v", m.playing, m.paused)
	}
	tracks := m.queue.Tracks()
	if len(tracks) != 2 || tracks[0].ID != seed.ID || tracks[1].ID != "a" || m.queue.CurrentIndex() != 0 {
		t.Fatalf("Mood queue=%#v current=%d", tracks, m.queue.CurrentIndex())
	}
}

func TestMoodProgressSplicesAheadOfSafetyBuffer(t *testing.T) {
	m := newHomeTestModel(t)
	seed, generated := testTrack("helena"), testTrack("jetset")
	m.playing = &seed
	m.queue.Replace([]subsonic.Track{seed, testTrack("old-next"), testTrack("old-last")}, 0)
	m.applyCuratorProgress(curatorProgressMsg{track: seed, accepted: 1, total: 20})
	m.applyCuratorProgress(curatorProgressMsg{track: generated, accepted: 2, total: 20})
	ids := queueIDs(m)
	want := []string{"helena", "jetset", "old-next", "old-last"}
	if len(ids) != len(want) {
		t.Fatalf("queue=%v", ids)
	}
	for i := range want {
		if ids[i] != want[i] {
			t.Fatalf("queue=%v want=%v", ids, want)
		}
	}
	if m.queue.CurrentIndex() != 0 || m.playing.ID != "helena" {
		t.Fatalf("current=%d playing=%s", m.queue.CurrentIndex(), m.playing.ID)
	}
}

func TestMoodStartedShowsServerPlaylistImmediately(t *testing.T) {
	m := newHomeTestModel(t)
	m.applyCuratorPlaylistStarted(curatorPlaylistStartedMsg{
		name:      "Mood",
		playlist:  subsonic.Playlist{ID: "mood", Name: "Mood", Count: 1},
		playlists: []subsonic.Playlist{{ID: "other", Name: "Other"}, {ID: "mood", Name: "Mood", Count: 1}},
	})
	selected, _ := m.list.SelectedItem().(item)
	if m.mode != modePlaylists || selected.playlist.ID != "mood" {
		t.Fatalf("mode=%v selected=%#v", m.mode, selected)
	}
}

func TestMoodProgressRefreshesPlaylistCountInPlace(t *testing.T) {
	m := newHomeTestModel(t)
	m.applyCuratorPlaylistStarted(curatorPlaylistStartedMsg{
		name:      "Mood",
		playlist:  subsonic.Playlist{ID: "mood", Name: "Mood", Count: 1},
		playlists: []subsonic.Playlist{{ID: "other", Name: "Other"}, {ID: "mood", Name: "Mood", Count: 1}},
	})
	m.list.Select(1)
	m.applyCuratorProgress(curatorProgressMsg{track: testTrack("next"), accepted: 4, total: 20})
	selected := m.list.SelectedItem().(item)
	if selected.playlist.ID != "mood" || selected.playlist.Count != 4 || m.list.Index() != 1 {
		t.Fatalf("selected=%#v index=%d", selected.playlist, m.list.Index())
	}
}

func TestOpenMoodAppendsTracksWithoutLeavingView(t *testing.T) {
	m := newHomeTestModel(t)
	m.mode = modeTracks
	m.playlistViewID = "mood"
	m.curatorPlaylistID = "mood"
	m.curatorPlaylistName = "Mood"
	m.curatorTracks = []subsonic.Track{testTrack("seed")}
	m.list.SetItems(trackItems(m.curatorTracks))
	m.applyCuratorProgress(curatorProgressMsg{track: testTrack("next"), accepted: 2, total: 20})
	if m.mode != modeTracks || m.playlistViewID != "mood" || len(m.list.Items()) != 2 {
		t.Fatalf("mode=%v playlist=%q items=%d", m.mode, m.playlistViewID, len(m.list.Items()))
	}
}

func TestMoodCompletionKeepsOpenPlaylistVisible(t *testing.T) {
	m := newHomeTestModel(t)
	m.mode = modeTracks
	m.playlistViewID = "mood"
	m.curatorPlaylistID = "mood"
	m.curatorPlaylistName = "Mood"
	m.list.SetItems(trackItems([]subsonic.Track{testTrack("seed")}))
	m = m.applyLLMQueue(llmQueueMsg{
		request:  curatorRequest{Mode: curator.ModeMood, Destination: curatorDestinationServer, PlaylistName: "Mood", Limit: 20},
		playlist: subsonic.Playlist{ID: "mood", Name: "Mood", Count: 2},
		result:   curator.Result{Tracks: []subsonic.Track{testTrack("seed"), testTrack("next")}},
	})
	if m.mode != modeTracks || len(m.list.Items()) != 2 {
		t.Fatalf("mode=%v items=%d", m.mode, len(m.list.Items()))
	}
}

func TestCuratorSpinnerSurvivesOrdinaryLoadCompletion(t *testing.T) {
	m := newHomeTestModel(t)
	m.searching = false
	m.curating = true
	m.curatorStarted = time.Now()
	line := m.statusLine()
	if !strings.Contains(line, m.spinner.View()) {
		t.Fatalf("curator status lost spinner: %q", line)
	}
}

func TestCuratorStageUpdatesStatusWithoutEndingSpinner(t *testing.T) {
	m := newHomeTestModel(t)
	m.curating = true
	m.curatorID = "generation"
	m.curatorEvents = make(chan tea.Msg)
	next, _ := m.Update(curatorStageMsg{generation: "generation", status: "judging 12 anchor candidates"})
	got := next.(Model)
	if got.status != "judging 12 anchor candidates" || !got.curating {
		t.Fatalf("status=%q curating=%v", got.status, got.curating)
	}
}

func TestCuratorFailureKeepsStageVisibleInStatusLine(t *testing.T) {
	m := newHomeTestModel(t)
	m.curating = true
	m.status = "judging 12 anchor candidates"
	next, _ := m.Update(errMsg{err: errors.New("established only 2/3 anchors")})
	got := next.(Model)
	if got.err != "judging 12 anchor candidates: established only 2/3 anchors" || !strings.Contains(got.statusLine(), "error:") {
		t.Fatalf("err=%q status line=%q", got.err, got.statusLine())
	}
}

func TestPublishedCuratorFailureReportsPreservedPartialCount(t *testing.T) {
	err := preservedPartialCuratorError(errors.New("provider stopped"), 17, 40)
	if !strings.Contains(err.Error(), "17/40") || !strings.Contains(err.Error(), "playable partial playlist preserved") {
		t.Fatalf("error=%q", err)
	}
}

func TestAIMixUsesNamedServerPlaylistAndLiveView(t *testing.T) {
	m := newHomeTestModel(t)
	m.applyCuratorPlaylistStarted(curatorPlaylistStartedMsg{
		name:      "AI Mix",
		playlist:  subsonic.Playlist{ID: "ai", Name: "AI Mix", Count: 1},
		playlists: []subsonic.Playlist{{ID: "ai", Name: "AI Mix", Count: 1}},
	})
	m.applyCuratorProgress(curatorProgressMsg{track: testTrack("new-track"), accepted: 1, total: 20})
	selected := m.list.SelectedItem().(item)
	if m.curatorPlaylistName != "AI Mix" || m.curatorPlaylistID != "ai" || selected.playlist.Count != 1 {
		t.Fatalf("name=%q id=%q selected=%#v", m.curatorPlaylistName, m.curatorPlaylistID, selected.playlist)
	}
	if m.queueTitle != "AI Mix (building)" {
		t.Fatalf("queue title=%q", m.queueTitle)
	}
}

func TestCuratorPlaylistNamesAreModeSpecific(t *testing.T) {
	if got := curatorPlaylistName(curator.ModeMood); got != "Mood" {
		t.Fatalf("Mood name=%q", got)
	}
	if got := curatorPlaylistName(curator.ModeAIMix); got != "AI Mix" {
		t.Fatalf("AI Mix name=%q", got)
	}
	if got := curatorPlaylistLimit(curator.ModeMood); got != 20 {
		t.Fatalf("Mood limit=%d", got)
	}
	if got := curatorPlaylistLimit(curator.ModeAIMix); got != 40 {
		t.Fatalf("AI Mix limit=%d", got)
	}
}
