package tui

import (
	"testing"
	"time"

	"github.com/bprendie/subweazl/internal/subsonic"
)

func TestTickIntervalTracksPlaybackWork(t *testing.T) {
	m := Model{terminalVisible: true}
	if got := m.tickInterval(); got != time.Second {
		t.Fatalf("idle interval = %v", got)
	}
	m.playing = &subsonic.Track{ID: "track"}
	m.terminalVisible = true
	if got := m.tickInterval(); got != time.Second/15 {
		t.Fatalf("playing interval = %v", got)
	}
	m.paused = true
	if got := m.tickInterval(); got != time.Second {
		t.Fatalf("paused interval = %v", got)
	}
	m.paused = false
	m.terminalVisible = false
	if got := m.tickInterval(); got != time.Second {
		t.Fatalf("detached interval = %v", got)
	}
	m.playing = nil
	if got := m.tickInterval(); got != 30*time.Second {
		t.Fatalf("detached idle interval = %v", got)
	}
}
