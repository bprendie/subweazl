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
	db, err := sql.Open("sqlite3", path+"?_foreign_keys=on")
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
