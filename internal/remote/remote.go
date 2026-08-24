package remote

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"
)

type Command string

const (
	Toggle    Command = "toggle"
	Next      Command = "next"
	Previous  Command = "previous"
	Stop      Command = "stop"
	CycleMode Command = "mode"
	Quit      Command = "quit"
)

type Snapshot struct {
	Running      bool      `json:"running"`
	State        string    `json:"state"`
	Title        string    `json:"title,omitempty"`
	Artist       string    `json:"artist,omitempty"`
	Album        string    `json:"album,omitempty"`
	Duration     int       `json:"duration,omitempty"`
	PlaybackMode string    `json:"playback_mode"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func StatePath() (string, error) {
	root := os.Getenv("XDG_STATE_HOME")
	if root == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		root = filepath.Join(home, ".local", "state")
	}
	return filepath.Join(root, "subweazl", "remote.json"), nil
}

func SocketPath() (string, error) {
	root := os.Getenv("XDG_RUNTIME_DIR")
	if root == "" {
		root = filepath.Join(os.TempDir(), "subweazl-"+os.Getenv("USER"))
	}
	return filepath.Join(root, "subweazl", "control.sock"), nil
}

func WriteSnapshot(snapshot Snapshot) error {
	path, err := StatePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	snapshot.UpdatedAt = time.Now().UTC()
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".remote-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(append(data, '\n')); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}

func ReadSnapshot() (Snapshot, error) {
	path, err := StatePath()
	if err != nil {
		return Snapshot{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return Snapshot{}, err
	}
	var snapshot Snapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return Snapshot{}, err
	}
	if snapshot.Running && time.Since(snapshot.UpdatedAt) > 15*time.Second {
		return snapshot, errors.New("Subweazl status is stale")
	}
	return snapshot, nil
}
