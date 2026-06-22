package localstore

import (
	"errors"
	"time"
)

type RecommendationRun struct {
	ID        string         `json:"id"`
	Provider  string         `json:"provider"`
	Model     string         `json:"model"`
	TrackIDs  []string       `json:"track_ids"`
	Payload   map[string]any `json:"payload"`
	CreatedAt string         `json:"created_at"`
}

func (s *Store) SaveRecommendationRun(run RecommendationRun) (RecommendationRun, error) {
	if run.Provider == "" {
		return RecommendationRun{}, errors.New("recommendation run provider is required")
	}
	if run.Model == "" {
		return RecommendationRun{}, errors.New("recommendation run model is required")
	}
	if len(run.TrackIDs) == 0 {
		return RecommendationRun{}, errors.New("recommendation run has no tracks")
	}
	if run.ID == "" {
		run.ID = newRecommendationID()
	}
	if run.CreatedAt == "" {
		run.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	blob, err := s.encryptedJSON(run)
	if err != nil {
		return RecommendationRun{}, err
	}
	_, err = s.db.Exec(`
		insert into recommendation_runs (id, provider, model, payload, created_at)
		values (?, ?, ?, ?, current_timestamp)
	`, run.ID, run.Provider, run.Model, blob)
	return run, err
}
