package localstore

import "errors"

const (
	SourceLocal    = "local"
	SourceSubsonic = "subsonic"
)

type PlayHistoryRecord struct {
	Source  string
	TrackID string
	Payload any
}

type PlayHistoryEntry struct {
	ID      int64
	Source  string
	TrackID string
	Payload map[string]any
	Played  string
}

func (s *Store) AddPlayHistory(record PlayHistoryRecord) error {
	if record.Source != SourceLocal && record.Source != SourceSubsonic {
		return errors.New("play history source must be local or subsonic")
	}
	if record.TrackID == "" {
		return errors.New("play history track id is required")
	}
	blob, err := s.encryptedJSON(record.Payload)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`
		insert into play_history (source, track_id, payload)
		values (?, ?, ?)
	`, record.Source, record.TrackID, blob)
	return err
}

func (s *Store) PlayHistory(limit int) ([]PlayHistoryEntry, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.db.Query(`
		select id, source, track_id, payload, played_at
		from play_history
		order by played_at desc, id desc
		limit ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var entries []PlayHistoryEntry
	for rows.Next() {
		var entry PlayHistoryEntry
		var blob string
		if err := rows.Scan(&entry.ID, &entry.Source, &entry.TrackID, &blob, &entry.Played); err != nil {
			return nil, err
		}
		payload, err := s.decryptPayload(blob)
		if err != nil {
			return nil, err
		}
		entry.Payload = payload
		entries = append(entries, entry)
	}
	return entries, rows.Err()
}
