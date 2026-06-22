package localstore

import (
	"encoding/json"
	"sort"
	"strconv"
	"strings"
)

type FolderSummary struct {
	ID                  string
	Path                string
	LastScanCompletedAt string
}

type TrackSummary struct {
	ID          string
	Title       string
	Artist      string
	Album       string
	Path        string
	DiscNumber  int
	TrackNumber int
	Missing     bool
}

func (s *Store) FolderSummaries() ([]FolderSummary, error) {
	rows, err := s.db.Query(`
		select id, payload, coalesce(last_scan_completed_at, '')
		from folders
		order by updated_at desc, id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var folders []FolderSummary
	for rows.Next() {
		var folder FolderSummary
		var payload string
		if err := rows.Scan(&folder.ID, &payload, &folder.LastScanCompletedAt); err != nil {
			return nil, err
		}
		values, err := s.decryptPayload(payload)
		if err != nil {
			return nil, err
		}
		folder.Path = stringValue(values, "path")
		folders = append(folders, folder)
	}
	return folders, rows.Err()
}

func (s *Store) TrackSummaries(limit int) ([]TrackSummary, error) {
	if limit <= 0 {
		limit = 200
	}
	rows, err := s.db.Query(`
		select id, payload, missing
		from tracks
		order by updated_at desc, id
		limit ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tracks []TrackSummary
	for rows.Next() {
		var track TrackSummary
		var payload string
		var missing int
		if err := rows.Scan(&track.ID, &payload, &missing); err != nil {
			return nil, err
		}
		values, err := s.decryptPayload(payload)
		if err != nil {
			return nil, err
		}
		track.Title = stringValue(values, "title")
		track.Artist = stringValue(values, "artist")
		track.Album = stringValue(values, "album")
		track.Path = stringValue(values, "path")
		track.DiscNumber = intValue(values, "disc_number")
		track.TrackNumber = intValue(values, "track_number")
		track.Missing = missing != 0
		tracks = append(tracks, track)
	}
	sort.SliceStable(tracks, func(i, j int) bool {
		return trackSortKey(tracks[i]) < trackSortKey(tracks[j])
	})
	return tracks, rows.Err()
}

func (s *Store) TrackCount() (int, error) {
	var count int
	err := s.db.QueryRow(`select count(*) from tracks where missing = 0`).Scan(&count)
	return count, err
}

func (s *Store) decryptPayload(blob string) (map[string]any, error) {
	plain, err := s.decrypt(blob)
	if err != nil {
		return nil, err
	}
	values := map[string]any{}
	if err := json.Unmarshal([]byte(plain), &values); err != nil {
		return nil, err
	}
	return values, nil
}

func stringValue(values map[string]any, key string) string {
	switch v := values[key].(type) {
	case string:
		return v
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case json.Number:
		return v.String()
	default:
		return ""
	}
}

func intValue(values map[string]any, key string) int {
	switch v := values[key].(type) {
	case float64:
		return int(v)
	case json.Number:
		n, _ := strconv.Atoi(v.String())
		return n
	case string:
		n, _ := strconv.Atoi(v)
		return n
	default:
		return 0
	}
}

func trackSortKey(track TrackSummary) string {
	return strings.ToLower(track.Artist) + "\x00" +
		strings.ToLower(track.Album) + "\x00" +
		sortInt(track.DiscNumber) + "\x00" +
		sortInt(track.TrackNumber) + "\x00" +
		strings.ToLower(track.Title) + "\x00" +
		strings.ToLower(track.Path)
}

func sortInt(n int) string {
	if n <= 0 {
		n = 9999
	}
	return strconv.FormatInt(int64(n+10000), 10)
}
