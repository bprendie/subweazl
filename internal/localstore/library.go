package localstore

import (
	"encoding/json"
	"errors"
)

type TrackRecord struct {
	ID           string
	FolderID     string
	FileHash     string
	FileSize     int64
	ModifiedUnix int64
	Payload      any
}

func (s *Store) UpsertFolder(id string, payload any) error {
	if id == "" {
		return errors.New("folder id is required")
	}
	blob, err := s.encryptedJSON(payload)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`
		insert into folders (id, payload, last_scan_started_at, updated_at)
		values (?, ?, current_timestamp, current_timestamp)
		on conflict(id) do update set
			payload = excluded.payload,
			last_scan_started_at = current_timestamp,
			updated_at = current_timestamp
	`, id, blob)
	return err
}

func (s *Store) CompleteFolderScan(id string) error {
	if id == "" {
		return errors.New("folder id is required")
	}
	_, err := s.db.Exec(`
		update folders
		set last_scan_completed_at = current_timestamp,
			updated_at = current_timestamp
		where id = ?
	`, id)
	return err
}

func (s *Store) UpsertTrack(record TrackRecord) error {
	if record.ID == "" {
		return errors.New("track id is required")
	}
	if record.FolderID == "" {
		return errors.New("track folder id is required")
	}
	blob, err := s.encryptedJSON(record.Payload)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`
		insert into tracks (
			id, folder_id, payload, file_hash, file_size, modified_unix,
			missing, updated_at
		)
		values (?, ?, ?, ?, ?, ?, 0, current_timestamp)
		on conflict(id) do update set
			folder_id = excluded.folder_id,
			payload = excluded.payload,
			file_hash = excluded.file_hash,
			file_size = excluded.file_size,
			modified_unix = excluded.modified_unix,
			missing = 0,
			updated_at = current_timestamp
	`, record.ID, record.FolderID, blob, record.FileHash, record.FileSize, record.ModifiedUnix)
	return err
}

func (s *Store) MarkMissingTracks(folderID string, presentIDs []string) error {
	if folderID == "" {
		return errors.New("folder id is required")
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`
		update tracks
		set missing = 1,
			updated_at = current_timestamp
		where folder_id = ?
	`, folderID); err != nil {
		return err
	}
	for _, id := range presentIDs {
		if id == "" {
			continue
		}
		if _, err := tx.Exec(`
			update tracks
			set missing = 0,
				updated_at = current_timestamp
			where folder_id = ? and id = ?
		`, folderID, id); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) encryptedJSON(payload any) (string, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return s.encrypt(string(raw))
}
