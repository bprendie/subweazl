package tui

import (
	"strings"
	"testing"

	"github.com/bprendie/subweazl/internal/config"
	"github.com/bprendie/subweazl/internal/localstore"
	"github.com/bprendie/subweazl/internal/state"
	"github.com/bprendie/subweazl/internal/subsonic"
)

func newHomeTestModel(t *testing.T) Model {
	t.Helper()
	t.Setenv("SUBWEAZL_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	m := New(config.Config{Server: "https://example.test", Username: "user", Password: "pass"})
	m.vaultInput.SetValue("thisguy47")
	next, _ := m.submitVault()
	m = next
	m.vaultInput.SetValue("thisguy47")
	next, _ = m.submitVault()
	m = next
	if m.mode != modeHome {
		t.Fatalf("mode = %v, want home", m.mode)
	}
	return m
}

func TestHomeHasDiscoveryFallbacks(t *testing.T) {
	m := newHomeTestModel(t)
	titles := homeTitles(m)
	for _, want := range []string{"Newest albums", "Playlists", "Random albums", "Search", "Last queue", "Private playlists"} {
		if !strings.Contains(titles, want) {
			t.Fatalf("home titles %q missing %q", titles, want)
		}
	}
}

func TestHomeShowsLastPlayed(t *testing.T) {
	m := newHomeTestModel(t)
	m.appState.LastPlayed = &state.LastPlayed{ID: "song-1", Title: "Private Signal", Artist: "The Weazls"}
	m.showHome()
	it, ok := m.list.Items()[0].(item)
	if !ok || it.action != homeActionResume || it.track.ID != "song-1" {
		t.Fatalf("first home item = %#v", m.list.Items()[0])
	}
}

func TestHomeShowsVaultedRecentHistory(t *testing.T) {
	m := newHomeTestModel(t)
	if err := m.vaultStore.AddPlayHistory(localstore.PlayHistoryRecord{
		Source:  localstore.SourceSubsonic,
		TrackID: "song-2",
		Payload: map[string]any{"title": "Recent Signal", "artist": "The Weazls"},
	}); err != nil {
		t.Fatalf("AddPlayHistory: %v", err)
	}
	m.showHome()
	if !strings.Contains(homeTitles(m), "Recent Recent Signal") {
		t.Fatalf("home titles = %q", homeTitles(m))
	}
}

func TestHomeActionSearchFocusesInput(t *testing.T) {
	m := newHomeTestModel(t)
	next, _ := m.handleHomeAction(item{kind: "home", action: homeActionSearch})
	if next.mode != modeSearch || !next.input.Focused() {
		t.Fatalf("mode=%v focused=%v", next.mode, next.input.Focused())
	}
}

func TestRecordSubsonicPlayWritesVaultedHistory(t *testing.T) {
	m := newHomeTestModel(t)
	track := subsonic.Track{ID: "song-3", Title: "Logged Signal", Artist: "The Weazls"}
	if err := m.recordSubsonicPlay(track); err != nil {
		t.Fatalf("recordSubsonicPlay: %v", err)
	}
	entries, err := m.vaultStore.PlayHistory(10)
	if err != nil {
		t.Fatalf("PlayHistory: %v", err)
	}
	if len(entries) != 1 || entries[0].Payload["title"] != "Logged Signal" {
		t.Fatalf("history = %#v", entries)
	}
}

func homeTitles(m Model) string {
	var titles []string
	for _, row := range m.list.Items() {
		if it, ok := row.(item); ok {
			titles = append(titles, it.Title())
		}
	}
	return strings.Join(titles, "|")
}
