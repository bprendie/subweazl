package state

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/bprendie/subweazl/internal/subsonic"
)

type State struct {
	LastPlayed *LastPlayed `json:"last_played,omitempty"`
}

type LastPlayed struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Artist   string `json:"artist,omitempty"`
	Album    string `json:"album,omitempty"`
	AlbumID  string `json:"album_id,omitempty"`
	CoverID  string `json:"cover_art,omitempty"`
	Duration int    `json:"duration,omitempty"`
}

func Load() (State, error) {
	path, err := Path()
	if err != nil {
		return State{}, err
	}
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return State{}, nil
	}
	if err != nil {
		return State{}, err
	}
	var st State
	if err := json.Unmarshal(b, &st); err != nil {
		return State{}, err
	}
	if st.LastPlayed != nil && st.LastPlayed.ID == "" {
		st.LastPlayed = nil
	}
	return st, nil
}

func Save(st State) error {
	path, err := Path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(b, '\n'), 0o600)
}

func Path() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "subweazl", "state.json"), nil
}

func FromTrack(track subsonic.Track) LastPlayed {
	return LastPlayed{
		ID:       strings.TrimSpace(track.ID),
		Title:    strings.TrimSpace(track.Title),
		Artist:   strings.TrimSpace(track.Artist),
		Album:    strings.TrimSpace(track.Album),
		AlbumID:  strings.TrimSpace(track.AlbumID),
		CoverID:  strings.TrimSpace(track.CoverID),
		Duration: track.Duration,
	}
}

func (lp LastPlayed) Track() subsonic.Track {
	return subsonic.Track{
		ID:       lp.ID,
		Title:    lp.Title,
		Artist:   lp.Artist,
		Album:    lp.Album,
		AlbumID:  lp.AlbumID,
		CoverID:  lp.CoverID,
		Duration: lp.Duration,
	}
}
