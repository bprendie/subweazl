package tui

import (
	"testing"

	"github.com/bprendie/subweazl/internal/subsonic"
)

func TestSearchUsesCachedTracks(t *testing.T) {
	m := newHomeTestModel(t)
	if err := m.vaultStore.BeginSubsonicCacheSync(); err != nil {
		t.Fatalf("BeginSubsonicCacheSync: %v", err)
	}
	track := subsonic.Track{ID: "cached-1", Title: "Cached Signal", Artist: "The Weazls", Album: "Vault"}
	if err := m.vaultStore.UpsertSubsonicTrack(track, false); err != nil {
		t.Fatalf("UpsertSubsonicTrack: %v", err)
	}
	if err := m.vaultStore.CompleteSubsonicCacheSync([]string{"cached-1"}); err != nil {
		t.Fatalf("CompleteSubsonicCacheSync: %v", err)
	}
	msg := m.search("cached")()
	loaded, ok := msg.(loadedMsg)
	if !ok {
		t.Fatalf("msg = %#v, want loadedMsg", msg)
	}
	if loaded.mode != modeSearch || loaded.status != "loaded 1 cached search tracks" {
		t.Fatalf("loaded = %#v", loaded)
	}
	if len(loaded.items) != 1 || loaded.items[0].(item).track.ID != "cached-1" {
		t.Fatalf("items = %#v", loaded.items)
	}
}

func TestRefreshCacheStatus(t *testing.T) {
	m := newHomeTestModel(t)
	if err := m.vaultStore.BeginSubsonicCacheSync(); err != nil {
		t.Fatalf("BeginSubsonicCacheSync: %v", err)
	}
	if err := m.vaultStore.UpsertSubsonicTrack(subsonic.Track{ID: "cached-1", Title: "Cached"}, false); err != nil {
		t.Fatalf("UpsertSubsonicTrack: %v", err)
	}
	if err := m.vaultStore.CompleteSubsonicCacheSync([]string{"cached-1"}); err != nil {
		t.Fatalf("CompleteSubsonicCacheSync: %v", err)
	}
	m.refreshCacheStatus()
	if m.cacheStatus.TrackCount != 1 {
		t.Fatalf("cacheStatus = %#v", m.cacheStatus)
	}
}
