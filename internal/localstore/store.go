package localstore

import (
	"database/sql"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

type Store struct {
	db       *sql.DB
	key      []byte
	unlocked bool
}

func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}
	// Cache sync, playback history, and queue persistence can briefly write at
	// the same time. WAL lets readers continue during those writes, while the
	// busy timeout makes a competing writer wait instead of failing instantly.
	db, err := sql.Open("sqlite3", path+"?_foreign_keys=on&_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, err
	}
	store := &Store{db: db}
	if err := store.db.Ping(); err != nil {
		_ = store.Close()
		return nil, err
	}
	return store, nil
}

func OpenDefault() (*Store, error) {
	path, err := Path()
	if err != nil {
		return nil, err
	}
	return Open(path)
}

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *Store) RawDB() *sql.DB {
	return s.db
}
