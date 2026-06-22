package tui

import (
	"strings"
	"testing"

	"github.com/bprendie/subweazl/internal/curator"
	"github.com/bprendie/subweazl/internal/localstore"
	"github.com/bprendie/subweazl/internal/subsonic"
)

func TestGenerateLLMQueueRequiresConfig(t *testing.T) {
	m := newHomeTestModel(t)
	got, cmd := m.generateLLMQueue()
	if cmd == nil {
		t.Fatal("expected noop command")
	}
	if !strings.Contains(got.err, "not configured") {
		t.Fatalf("err = %q", got.err)
	}
}

func TestApplyLLMQueuePersistsQueue(t *testing.T) {
	m := newHomeTestModel(t)
	msg := llmQueueMsg{
		result: curator.Result{
			Tracks: []subsonic.Track{testTrack("a"), testTrack("b")},
			IDs:    []string{"a", "b"},
		},
		run: localstore.RecommendationRun{ID: "run-a"},
	}
	m = m.applyLLMQueue(msg)
	if m.mode != modeQueue {
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
