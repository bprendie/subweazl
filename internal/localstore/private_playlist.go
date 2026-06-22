package localstore

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/bprendie/subweazl/internal/playqueue"
	"github.com/bprendie/subweazl/internal/subsonic"
)

const PrivatePlaylistKind = "private"

type PrivatePlaylist struct {
	ID      string           `json:"id"`
	Name    string           `json:"name"`
	Tracks  []subsonic.Track `json:"tracks"`
	Current int              `json:"current"`
	Created string           `json:"created"`
	Updated string           `json:"updated"`
}

func (p PrivatePlaylist) Snapshot() playqueue.Snapshot {
	return playqueue.Snapshot{Tracks: append([]subsonic.Track(nil), p.Tracks...), Current: p.Current}
}

func (s *Store) SavePrivatePlaylist(name string, snapshot playqueue.Snapshot) (PrivatePlaylist, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return PrivatePlaylist{}, errors.New("private playlist name is required")
	}
	tracks := validPlaylistTracks(snapshot.Tracks)
	if len(tracks) == 0 {
		return PrivatePlaylist{}, errors.New("queue is empty")
	}
	now := time.Now().UTC().Format(time.RFC3339)
	playlist := PrivatePlaylist{
		ID:      newPlaylistID(),
		Name:    name,
		Tracks:  tracks,
		Current: clampPlaylistIndex(snapshot.Current, len(tracks)),
		Created: now,
		Updated: now,
	}
	return playlist, s.writePrivatePlaylist(playlist)
}

func (s *Store) RenamePrivatePlaylist(id, name string) error {
	playlist, ok, err := s.PrivatePlaylist(id)
	if err != nil || !ok {
		return err
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("private playlist name is required")
	}
	playlist.Name = name
	playlist.Updated = time.Now().UTC().Format(time.RFC3339)
	return s.writePrivatePlaylist(playlist)
}

func (s *Store) DeletePrivatePlaylist(id string) error {
	if strings.TrimSpace(id) == "" {
		return errors.New("private playlist id is required")
	}
	_, err := s.db.Exec(`delete from local_playlists where id = ? and kind = ?`, id, PrivatePlaylistKind)
	return err
}

func (s *Store) PrivatePlaylist(id string) (PrivatePlaylist, bool, error) {
	var blob string
	err := s.db.QueryRow(`select payload from local_playlists where id = ? and kind = ?`, id, PrivatePlaylistKind).Scan(&blob)
	if err != nil {
		if err == sql.ErrNoRows {
			return PrivatePlaylist{}, false, nil
		}
		return PrivatePlaylist{}, false, err
	}
	playlist, err := s.decryptPrivatePlaylist(blob)
	return playlist, err == nil, err
}

func (s *Store) PrivatePlaylists() ([]PrivatePlaylist, error) {
	rows, err := s.db.Query(`
		select payload from local_playlists
		where kind = ?
		order by updated_at desc, created_at desc
	`, PrivatePlaylistKind)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var playlists []PrivatePlaylist
	for rows.Next() {
		var blob string
		if err := rows.Scan(&blob); err != nil {
			return nil, err
		}
		playlist, err := s.decryptPrivatePlaylist(blob)
		if err != nil {
			return nil, err
		}
		playlists = append(playlists, playlist)
	}
	return playlists, rows.Err()
}

func (s *Store) writePrivatePlaylist(playlist PrivatePlaylist) error {
	blob, err := s.encryptedJSON(playlist)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`
		insert into local_playlists (id, kind, payload, updated_at)
		values (?, ?, ?, current_timestamp)
		on conflict(id) do update set
			kind = excluded.kind,
			payload = excluded.payload,
			updated_at = excluded.updated_at
	`, playlist.ID, PrivatePlaylistKind, blob)
	return err
}

func (s *Store) decryptPrivatePlaylist(blob string) (PrivatePlaylist, error) {
	payload, err := s.decryptPayload(blob)
	if err != nil {
		return PrivatePlaylist{}, err
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return PrivatePlaylist{}, err
	}
	var playlist PrivatePlaylist
	if err := json.Unmarshal(raw, &playlist); err != nil {
		return PrivatePlaylist{}, err
	}
	return playlist, nil
}

func validPlaylistTracks(tracks []subsonic.Track) []subsonic.Track {
	out := make([]subsonic.Track, 0, len(tracks))
	for _, track := range tracks {
		if track.ID != "" {
			out = append(out, track)
		}
	}
	return out
}

func clampPlaylistIndex(index, length int) int {
	if length == 0 {
		return -1
	}
	if index < 0 {
		return 0
	}
	if index >= length {
		return length - 1
	}
	return index
}

func newPlaylistID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return time.Now().UTC().Format("20060102150405.000000000")
	}
	return hex.EncodeToString(b[:])
}
