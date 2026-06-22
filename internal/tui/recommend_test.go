package tui

import (
	"strings"
	"testing"

	"github.com/bprendie/subweazl/internal/localstore"
	"github.com/bprendie/subweazl/internal/subsonic"
)

func TestGenerateRecommendedQueueUsesCache(t *testing.T) {
	m := newHomeTestModel(t)
	seed := testTrack("seed")
	seed.Artist = "Artist A"
	seed.Genre = "Rock"
	seed.Year = 2000
	m.playing = &seed
	cacheTracks := []subsonic.Track{
		seed,
		{ID: "artist", Title: "Artist Match", Artist: "Artist A", Album: "Album", Genre: "Pop", Year: 1990},
		{ID: "genre", Title: "Genre Match", Artist: "Artist B", Album: "Album", Genre: "Rock", Year: 1980},
		{ID: "recent", Title: "Recent", Artist: "Artist A", Album: "Album", Genre: "Rock", Year: 2000},
	}
	seedCache(t, m, cacheTracks, map[string]bool{})
	if err := m.vaultStore.AddPlayHistory(localstore.PlayHistoryRecord{Source: localstore.SourceSubsonic, TrackID: "recent", Payload: map[string]any{"title": "Recent"}}); err != nil {
		t.Fatalf("AddPlayHistory: %v", err)
	}
	next, _ := m.generateRecommendedQueue()
	m = next
	if m.mode != modeQueue {
		t.Fatalf("mode = %v, want queue", m.mode)
	}
	ids := queueIDs(m)
	if strings.Contains(strings.Join(ids, ","), "recent") || !hasID(ids, "artist") || !hasID(ids, "genre") {
		t.Fatalf("queue ids = %#v", ids)
	}
	var recipes int
	if err := m.vaultStore.RawDB().QueryRow(`select count(*) from station_recipes`).Scan(&recipes); err != nil {
		t.Fatalf("count recipes: %v", err)
	}
	if recipes != 1 {
		t.Fatalf("recipes = %d, want 1", recipes)
	}
}

func TestGenerateRecommendedQueueRequiresCache(t *testing.T) {
	m := newHomeTestModel(t)
	next, _ := m.generateRecommendedQueue()
	if next.err != "sync the Subsonic cache first" {
		t.Fatalf("err = %q", next.err)
	}
}

func seedCache(t *testing.T, m Model, tracks []subsonic.Track, starred map[string]bool) {
	t.Helper()
	if err := m.vaultStore.BeginSubsonicCacheSync(); err != nil {
		t.Fatalf("BeginSubsonicCacheSync: %v", err)
	}
	ids := make([]string, 0, len(tracks))
	for _, track := range tracks {
		ids = append(ids, track.ID)
		if err := m.vaultStore.UpsertSubsonicTrack(track, starred[track.ID]); err != nil {
			t.Fatalf("UpsertSubsonicTrack: %v", err)
		}
	}
	if err := m.vaultStore.CompleteSubsonicCacheSync(ids); err != nil {
		t.Fatalf("CompleteSubsonicCacheSync: %v", err)
	}
}

func queueIDs(m Model) []string {
	tracks := m.queue.Tracks()
	ids := make([]string, 0, len(tracks))
	for _, track := range tracks {
		ids = append(ids, track.ID)
	}
	return ids
}

func hasID(ids []string, want string) bool {
	for _, id := range ids {
		if id == want {
			return true
		}
	}
	return false
}
