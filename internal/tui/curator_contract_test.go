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

func TestPhaseOneChooserStartsPrivateAutogenerate(t *testing.T) {
	m := newHomeTestModel(t)
	m.queue.Replace([]subsonic.Track{testTrack("keep")}, 0)
	m, _ = m.startCuratorChoice()
	m, _ = m.handleCuratorInputKey(tea.KeyMsg{Type: tea.KeyEnter})
	if m.mode != modeHome || m.curating || m.curatorDraft.Request.PlaylistName != "zero_tax_grindage" {
		t.Fatalf("mode=%v curating=%v request=%#v", m.mode, m.curating, m.curatorDraft.Request)
	}
	assertIDs(t, queueIDs(m), []string{"keep"})
}

func TestPhaseOnePromptCapturesNamedPrivateRequest(t *testing.T) {
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
	if m.mode != modeHome || request.Destination != curatorDestinationVault || request.PlaylistName != "AI Mix: Synthwave Focus" || request.Prompt != "synthwave tracks for focus" || request.Limit != 40 {
		t.Fatalf("mode=%v request=%#v", m.mode, request)
	}
}

func TestPromptCuratorPlaylistNameIsDeterministicAndBounded(t *testing.T) {
	if got := promptCuratorPlaylistName("please make me some synthwave tracks for focus"); got != "AI Mix: Synthwave Focus" {
		t.Fatalf("name=%q", got)
	}
	if got := promptCuratorPlaylistName("the music playlist mix"); got != "AI Mix: Weazl Cut" {
		t.Fatalf("fallback name=%q", got)
	}
	if got := promptCuratorPlaylistName("cinematic instrumental post rock with enormous crescendos and atmospheric guitars"); len([]rune(got)) > len([]rune("AI Mix: "))+32 {
		t.Fatalf("unbounded name=%q", got)
	}
}

func TestPromptCuratorPlaylistNamePutsNamedReferenceFirst(t *testing.T) {
	for prompt, want := range map[string]string{
		"rock like Oasis":                        "AI Mix: Oasis Rock",
		"new wave like New Order":                "AI Mix: New Order New Wave",
		"synthwave like Slave to the Squarewave": "AI Mix: Slave to the Squarewave",
	} {
		if got := promptCuratorPlaylistName(prompt); got != want {
			t.Errorf("prompt=%q name=%q want=%q", prompt, got, want)
		}
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
