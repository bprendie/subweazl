package localstore

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"github.com/bprendie/subweazl/internal/subsonic"
)

type CachedTrack struct {
	Track   subsonic.Track
	Starred bool
}

type RecommendationRecipe struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	SeedID    string         `json:"seed_id"`
	Rules     []string       `json:"rules"`
	TrackIDs  []string       `json:"track_ids"`
	CreatedAt string         `json:"created_at"`
	Payload   map[string]any `json:"payload,omitempty"`
}

func (s *Store) CachedSubsonicTracks(limit int) ([]CachedTrack, error) {
	query := `select payload from tracks where folder_id = ? and missing = 0 order by id`
	args := []any{SubsonicCacheFolderID}
	if limit > 0 {
		query += ` limit ?`
		args = append(args, limit)
	}
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tracks []CachedTrack
	for rows.Next() {
		var blob string
		if err := rows.Scan(&blob); err != nil {
			return nil, err
		}
		payload, err := s.decryptPayload(blob)
		if err != nil {
			return nil, err
		}
		track, err := cachedTrackFromPayload(payload)
		if err != nil {
			return nil, err
		}
		if track.ID != "" {
			tracks = append(tracks, CachedTrack{Track: track, Starred: boolValue(payload, "starred")})
		}
	}
	return tracks, rows.Err()
}

func (s *Store) RecentSubsonicTrackIDs(limit int) (map[string]bool, error) {
	entries, err := s.PlayHistory(limit)
	if err != nil {
		return nil, err
	}
	ids := map[string]bool{}
	for _, entry := range entries {
		if entry.Source == SourceSubsonic && entry.TrackID != "" {
			ids[entry.TrackID] = true
		}
	}
	return ids, nil
}

func (s *Store) SaveRecommendationRecipe(recipe RecommendationRecipe) (RecommendationRecipe, error) {
	if recipe.Name == "" {
		return RecommendationRecipe{}, errors.New("recommendation recipe name is required")
	}
	if len(recipe.TrackIDs) == 0 {
		return RecommendationRecipe{}, errors.New("recommendation recipe has no tracks")
	}
	if recipe.ID == "" {
		recipe.ID = newRecommendationID()
	}
	if recipe.CreatedAt == "" {
		recipe.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	blob, err := s.encryptedJSON(recipe)
	if err != nil {
		return RecommendationRecipe{}, err
	}
	_, err = s.db.Exec(`
		insert into station_recipes (id, payload, updated_at)
		values (?, ?, current_timestamp)
		on conflict(id) do update set
			payload = excluded.payload,
			updated_at = excluded.updated_at
	`, recipe.ID, blob)
	return recipe, err
}

func cachedTrackFromPayload(payload map[string]any) (subsonic.Track, error) {
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

func boolValue(values map[string]any, key string) bool {
	v, ok := values[key]
	if !ok {
		return false
	}
	b, ok := v.(bool)
	return ok && b
}

func newRecommendationID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return time.Now().UTC().Format("20060102150405.000000000")
	}
	return hex.EncodeToString(b[:])
}
