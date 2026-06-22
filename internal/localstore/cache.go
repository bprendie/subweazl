package localstore

import (
	"encoding/json"
	"sort"
	"strings"
	"time"

	"github.com/bprendie/subweazl/internal/subsonic"
)

const SubsonicCacheFolderID = "subsonic-cache"

type CacheStatus struct {
	TrackCount          int
	LastScanCompletedAt string
}

func (s *Store) BeginSubsonicCacheSync() error {
	return s.UpsertFolder(SubsonicCacheFolderID, map[string]any{
		"kind": "subsonic-cache",
		"path": "subsonic://cache",
	})
}

func (s *Store) CompleteSubsonicCacheSync(presentIDs []string) error {
	if err := s.MarkMissingTracks(SubsonicCacheFolderID, presentIDs); err != nil {
		return err
	}
	return s.CompleteFolderScan(SubsonicCacheFolderID)
}

func (s *Store) UpsertSubsonicTrack(track subsonic.Track, starred bool) error {
	if track.ID == "" {
		return nil
	}
	return s.UpsertTrack(TrackRecord{
		ID:       track.ID,
		FolderID: SubsonicCacheFolderID,
		Payload: map[string]any{
			"id":        track.ID,
			"title":     track.Title,
			"artist":    track.Artist,
			"album":     track.Album,
			"album_id":  track.AlbumID,
			"cover_id":  track.CoverID,
			"duration":  track.Duration,
			"genre":     track.Genre,
			"year":      track.Year,
			"starred":   starred,
			"synced_at": time.Now().UTC().Format(time.RFC3339),
		},
	})
}

func (s *Store) CachedSubsonicSearch(query string, limit int) ([]subsonic.Track, error) {
	query = strings.TrimSpace(strings.ToLower(query))
	if query == "" {
		return nil, nil
	}
	if limit <= 0 {
		limit = 80
	}
	rows, err := s.db.Query(`
		select payload from tracks
		where folder_id = ? and missing = 0
		order by updated_at desc, id
	`, SubsonicCacheFolderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var matches []subsonic.Track
	for rows.Next() {
		var blob string
		if err := rows.Scan(&blob); err != nil {
			return nil, err
		}
		track, err := s.decryptCachedTrack(blob)
		if err != nil {
			return nil, err
		}
		if cachedTrackMatches(track, query) {
			matches = append(matches, track)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	sort.SliceStable(matches, func(i, j int) bool {
		return cachedTrackKey(matches[i]) < cachedTrackKey(matches[j])
	})
	if len(matches) > limit {
		matches = matches[:limit]
	}
	return matches, nil
}

func (s *Store) SubsonicCacheStatus() (CacheStatus, error) {
	var status CacheStatus
	err := s.db.QueryRow(`
		select count(*) from tracks
		where folder_id = ? and missing = 0
	`, SubsonicCacheFolderID).Scan(&status.TrackCount)
	if err != nil {
		return status, err
	}
	_ = s.db.QueryRow(`
		select coalesce(last_scan_completed_at, '')
		from folders where id = ?
	`, SubsonicCacheFolderID).Scan(&status.LastScanCompletedAt)
	return status, nil
}

func (s *Store) decryptCachedTrack(blob string) (subsonic.Track, error) {
	payload, err := s.decryptPayload(blob)
	if err != nil {
		return subsonic.Track{}, err
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return subsonic.Track{}, err
	}
	var track subsonic.Track
	if err := json.Unmarshal(raw, &track); err != nil {
		return subsonic.Track{}, err
	}
	track.CoverID = stringValue(payload, "cover_id")
	return track, nil
}

func cachedTrackMatches(track subsonic.Track, query string) bool {
	haystack := strings.ToLower(strings.Join([]string{
		track.Title,
		track.Artist,
		track.Album,
		track.Genre,
		track.ID,
	}, " "))
	for _, term := range strings.Fields(query) {
		if !strings.Contains(haystack, term) {
			return false
		}
	}
	return true
}

func cachedTrackKey(track subsonic.Track) string {
	return strings.ToLower(track.Artist) + "\x00" +
		strings.ToLower(track.Album) + "\x00" +
		strings.ToLower(track.Title) + "\x00" + track.ID
}
