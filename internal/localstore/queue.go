package localstore

import (
	"database/sql"
	"encoding/json"

	"github.com/bprendie/subweazl/internal/playqueue"
)

const queueSnapshotID = 1

func (s *Store) SaveQueueSnapshot(snapshot playqueue.Snapshot) error {
	blob, err := s.encryptedJSON(snapshot)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`
		insert into queue_snapshot (id, payload, updated_at)
		values (?, ?, current_timestamp)
		on conflict(id) do update set
			payload = excluded.payload,
			updated_at = excluded.updated_at
	`, queueSnapshotID, blob)
	return err
}

func (s *Store) QueueSnapshot() (playqueue.Snapshot, bool, error) {
	var blob string
	err := s.db.QueryRow(`select payload from queue_snapshot where id = ?`, queueSnapshotID).Scan(&blob)
	if err != nil {
		if err == sql.ErrNoRows {
			return playqueue.Snapshot{}, false, nil
		}
		return playqueue.Snapshot{}, false, err
	}
	payload, err := s.decryptPayload(blob)
	if err != nil {
		return playqueue.Snapshot{}, false, err
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return playqueue.Snapshot{}, false, err
	}
	var snapshot playqueue.Snapshot
	if err := json.Unmarshal(raw, &snapshot); err != nil {
		return playqueue.Snapshot{}, false, err
	}
	return snapshot, true, nil
}
