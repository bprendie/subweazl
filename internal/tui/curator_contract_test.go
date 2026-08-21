package tui

import (
	"testing"

	"github.com/bprendie/subweazl/internal/curator"
	"github.com/bprendie/subweazl/internal/subsonic"
	tea "github.com/charmbracelet/bubbletea"
)

func TestMoodRequestContractRemainsServerSide(t *testing.T) {
	seed := testTrack("helena")
	request := moodCuratorRequest(seed)
	if request.Mode != curator.ModeMood || request.Destination != curatorDestinationServer || request.PlaylistName != "Mood" || request.Limit != 20 || request.Seed.ID != seed.ID {
		t.Fatalf("request=%#v", request)
	}
}

func TestGrindageRequestContractIsPrivateAndQueueIndependent(t *testing.T) {
	request := grindageCuratorRequest()
	if request.Mode != curator.ModeAIMix || request.Destination != curatorDestinationVault || request.PlaylistName != "zero_tax_grindage" || request.Limit != 40 || request.Seed.ID != "" {
		t.Fatalf("request=%#v", request)
	}
}

func TestGChooserAndCancelPreservePlaybackAndQueue(t *testing.T) {
	m := newHomeTestModel(t)
	playing := testTrack("playing")
	m.playing = &playing
	m.paused = true
	m.queue.Replace([]subsonic.Track{playing, testTrack("next")}, 0)
	before := queueIDs(m)
	next, _ := m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	if next.mode != modeCuratorChoice {
		t.Fatalf("mode=%v", next.mode)
	}
	next, _ = next.handleKey(tea.KeyMsg{Type: tea.KeyEsc})
	if next.mode != modeHome || next.playing == nil || next.playing.ID != playing.ID || !next.paused {
		t.Fatalf("mode=%v playing=%#v paused=%v", next.mode, next.playing, next.paused)
	}
	assertIDs(t, queueIDs(next), before)
}

func TestPhaseZeroChooserCapturesAutogenerateWithoutStarting(t *testing.T) {
	m := newHomeTestModel(t)
	m.queue.Replace([]subsonic.Track{testTrack("keep")}, 0)
	m, _ = m.startCuratorChoice()
	m, _ = m.handleCuratorInputKey(tea.KeyMsg{Type: tea.KeyEnter})
	if m.mode != modeHome || m.curating || m.curatorDraft.Request.PlaylistName != "zero_tax_grindage" {
		t.Fatalf("mode=%v curating=%v request=%#v", m.mode, m.curating, m.curatorDraft.Request)
	}
	assertIDs(t, queueIDs(m), []string{"keep"})
}

func TestPhaseZeroPromptCapturesPrivateRequest(t *testing.T) {
	m := newHomeTestModel(t)
	m, _ = m.startCuratorChoice()
	m.curatorDraft.Choice = 1
	m, _ = m.handleCuratorInputKey(tea.KeyMsg{Type: tea.KeyEnter})
	if m.mode != modeCuratorPrompt || !m.input.Focused() {
		t.Fatalf("mode=%v focused=%v", m.mode, m.input.Focused())
	}
	m.input.SetValue("synthwave tracks for focus")
	m, _ = m.handleCuratorInputKey(tea.KeyMsg{Type: tea.KeyEnter})
	request := m.curatorDraft.Request
	if m.mode != modeHome || request.Destination != curatorDestinationVault || request.Prompt != "synthwave tracks for focus" || request.Limit != 40 {
		t.Fatalf("mode=%v request=%#v", m.mode, request)
	}
}

func TestVaultDestinationCannotReplaceQueueAtProgressOrCompletion(t *testing.T) {
	m := newHomeTestModel(t)
	m.queue.Replace([]subsonic.Track{testTrack("keep"), testTrack("playing-next")}, 0)
	m.activeCurator = grindageCuratorRequest()
	m.applyCuratorProgress(curatorProgressMsg{track: testTrack("generated"), accepted: 1, total: 40})
	assertIDs(t, queueIDs(m), []string{"keep", "playing-next"})
	m = m.applyLLMQueue(llmQueueMsg{
		request: grindageCuratorRequest(),
		result:  curator.Result{Tracks: []subsonic.Track{testTrack("generated")}, IDs: []string{"generated"}},
	})
	assertIDs(t, queueIDs(m), []string{"keep", "playing-next"})
}

func TestStaleCuratorProgressCannotMutateNewSession(t *testing.T) {
	m := newHomeTestModel(t)
	m.curatorID = "new"
	m.activeCurator = moodCuratorRequest(testTrack("seed"))
	m.queue.Replace([]subsonic.Track{testTrack("keep")}, 0)
	next, _ := m.Update(curatorProgressMsg{generation: "old", track: testTrack("stale"), accepted: 2, total: 20})
	got := next.(Model)
	assertIDs(t, queueIDs(got), []string{"keep"})
}

func assertIDs(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("ids=%v want=%v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("ids=%v want=%v", got, want)
		}
	}
}
