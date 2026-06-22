package localstore

import (
	"strings"
	"testing"

	"github.com/bprendie/subweazl/internal/subsonic"
)

func TestCachedSubsonicTracksAndRecipeStorage(t *testing.T) {
	store := newMigratedStore(t)
	if err := store.CreateVault("pw"); err != nil {
		t.Fatalf("CreateVault: %v", err)
	}
	if err := store.BeginSubsonicCacheSync(); err != nil {
		t.Fatalf("BeginSubsonicCacheSync: %v", err)
	}
	if err := store.UpsertSubsonicTrack(subsonic.Track{ID: "song_1", Title: "Signal"}, true); err != nil {
		t.Fatalf("UpsertSubsonicTrack: %v", err)
	}
	if err := store.CompleteSubsonicCacheSync([]string{"song_1"}); err != nil {
		t.Fatalf("CompleteSubsonicCacheSync: %v", err)
	}
	tracks, err := store.CachedSubsonicTracks(0)
	if err != nil {
		t.Fatalf("CachedSubsonicTracks: %v", err)
	}
	if len(tracks) != 1 || tracks[0].Track.ID != "song_1" || !tracks[0].Starred {
		t.Fatalf("tracks = %#v", tracks)
	}
	recipe, err := store.SaveRecommendationRecipe(RecommendationRecipe{Name: "Recipe", TrackIDs: []string{"song_1"}, Rules: []string{"starred"}})
	if err != nil {
		t.Fatalf("SaveRecommendationRecipe: %v", err)
	}
	var blob string
	if err := store.db.QueryRow(`select payload from station_recipes where id = ?`, recipe.ID).Scan(&blob); err != nil {
		t.Fatalf("query recipe: %v", err)
	}
	if strings.Contains(blob, "Recipe") || strings.Contains(blob, "song_1") {
		t.Fatalf("recipe exposes plaintext: %q", blob)
	}
}

func TestRecentSubsonicTrackIDs(t *testing.T) {
	store := newMigratedStore(t)
	if err := store.CreateVault("pw"); err != nil {
		t.Fatalf("CreateVault: %v", err)
	}
	if err := store.AddPlayHistory(PlayHistoryRecord{Source: SourceSubsonic, TrackID: "recent", Payload: map[string]any{"title": "Recent"}}); err != nil {
		t.Fatalf("AddPlayHistory: %v", err)
	}
	ids, err := store.RecentSubsonicTrackIDs(10)
	if err != nil {
		t.Fatalf("RecentSubsonicTrackIDs: %v", err)
	}
	if !ids["recent"] {
		t.Fatalf("ids = %#v", ids)
	}
}
