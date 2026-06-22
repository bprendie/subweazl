package tui

import (
	"context"
	"fmt"
	"time"

	"github.com/bprendie/subweazl/internal/localstore"
	"github.com/bprendie/subweazl/internal/subsonic"
	tea "github.com/charmbracelet/bubbletea"
)

const cacheAlbumPageSize = 200

type cacheSyncMsg struct {
	tracks int
	status localstore.CacheStatus
}

func (m Model) syncSubsonicCache() tea.Cmd {
	return func() tea.Msg {
		if m.vaultStore == nil || !m.vaultStore.Unlocked() {
			return errMsg{fmt.Errorf("private vault is locked")}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		if err := m.vaultStore.BeginSubsonicCacheSync(); err != nil {
			return errMsg{err}
		}
		starred := map[string]bool{}
		if tracks, err := m.client.Starred(ctx); err == nil {
			for _, track := range tracks {
				if track.ID != "" {
					starred[track.ID] = true
				}
			}
		}
		present := []string{}
		seen := map[string]bool{}
		for offset := 0; ; offset += cacheAlbumPageSize {
			albums, err := m.client.Albums(ctx, offset, cacheAlbumPageSize)
			if err != nil {
				return errMsg{err}
			}
			if len(albums) == 0 {
				break
			}
			for _, album := range albums {
				tracks, err := m.client.Album(ctx, album.ID)
				if err != nil {
					return errMsg{err}
				}
				for _, track := range tracks {
					if track.ID == "" || seen[track.ID] {
						continue
					}
					seen[track.ID] = true
					present = append(present, track.ID)
					if err := m.vaultStore.UpsertSubsonicTrack(track, starred[track.ID]); err != nil {
						return errMsg{err}
					}
				}
			}
			if len(albums) < cacheAlbumPageSize {
				break
			}
		}
		if err := m.vaultStore.CompleteSubsonicCacheSync(present); err != nil {
			return errMsg{err}
		}
		status, err := m.vaultStore.SubsonicCacheStatus()
		if err != nil {
			return errMsg{err}
		}
		return cacheSyncMsg{tracks: len(present), status: status}
	}
}

func (m Model) searchCached(query string) ([]subsonic.Track, bool, error) {
	if m.vaultStore == nil || !m.vaultStore.Unlocked() {
		return nil, false, nil
	}
	status, err := m.vaultStore.SubsonicCacheStatus()
	if err != nil || status.TrackCount == 0 {
		return nil, false, err
	}
	tracks, err := m.vaultStore.CachedSubsonicSearch(query, 80)
	if err != nil {
		return nil, false, err
	}
	return tracks, len(tracks) > 0, nil
}

func (m *Model) refreshCacheStatus() {
	m.cacheStatus = localstore.CacheStatus{}
	if m.vaultStore == nil || !m.vaultStore.Unlocked() {
		return
	}
	status, err := m.vaultStore.SubsonicCacheStatus()
	if err == nil {
		m.cacheStatus = status
	}
}
