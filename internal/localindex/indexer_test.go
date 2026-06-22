package localindex

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bprendie/subweazl/internal/localstore"
)

type fakeProber map[string]Metadata

func (f fakeProber) Probe(_ context.Context, path string) (Metadata, error) {
	meta, ok := f[path]
	if !ok {
		return Metadata{}, errors.New("unsupported")
	}
	return meta, nil
}

func TestIndexFolderStoresEncryptedTracksAndSkipsUnsupported(t *testing.T) {
	store := newStore(t)
	root := t.TempDir()
	song := filepath.Join(root, "song.flac")
	note := filepath.Join(root, "notes.txt")
	writeFile(t, song, "fake media")
	writeFile(t, note, "not media")
	setModTime(t, song, 1700000000)

	indexer := Indexer{
		Store: store,
		Prober: fakeProber{
			song: {Title: "Local Signal", Artist: "Private Artist"},
		},
	}
	result, err := indexer.IndexFolder(context.Background(), root)
	if err != nil {
		t.Fatalf("IndexFolder: %v", err)
	}
	if result.Indexed != 1 || result.Skipped != 1 {
		t.Fatalf("result = %#v, want one indexed and one skipped", result)
	}
	trackID := TrackID(song, int64(len("fake media")), 1700000000)
	var payload string
	var missing int
	err = store.RawDB().QueryRow(`select payload, missing from tracks where id = ?`, trackID).Scan(&payload, &missing)
	if err != nil {
		t.Fatalf("query indexed track: %v", err)
	}
	if missing != 0 {
		t.Fatalf("missing = %d, want 0", missing)
	}
	if strings.Contains(payload, "Local Signal") || strings.Contains(payload, song) {
		t.Fatalf("payload exposes plaintext: %q", payload)
	}
}

func TestIndexFolderMarksRemovedTracksMissing(t *testing.T) {
	store := newStore(t)
	root := t.TempDir()
	keep := filepath.Join(root, "keep.flac")
	gone := filepath.Join(root, "gone.flac")
	writeFile(t, keep, "keep")
	writeFile(t, gone, "gone")
	setModTime(t, keep, 1700000001)
	setModTime(t, gone, 1700000002)

	prober := fakeProber{
		keep: {Title: "Keep"},
		gone: {Title: "Gone"},
	}
	indexer := Indexer{Store: store, Prober: prober}
	if _, err := indexer.IndexFolder(context.Background(), root); err != nil {
		t.Fatalf("initial IndexFolder: %v", err)
	}
	if err := os.Remove(gone); err != nil {
		t.Fatalf("remove gone track: %v", err)
	}
	delete(prober, gone)
	if _, err := indexer.IndexFolder(context.Background(), root); err != nil {
		t.Fatalf("second IndexFolder: %v", err)
	}
	goneID := TrackID(gone, int64(len("gone")), 1700000002)
	var missing int
	err := store.RawDB().QueryRow(`select missing from tracks where id = ?`, goneID).Scan(&missing)
	if err != nil {
		t.Fatalf("query removed track: %v", err)
	}
	if missing != 1 {
		t.Fatalf("removed track missing = %d, want 1", missing)
	}
}

func newStore(t *testing.T) *localstore.Store {
	t.Helper()
	store, err := localstore.Open(filepath.Join(t.TempDir(), "library.sqlite3"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	if err := store.Migrate(); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if err := store.CreateVault("pw"); err != nil {
		t.Fatalf("CreateVault: %v", err)
	}
	return store
}

func writeFile(t *testing.T, path string, value string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(value), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func setModTime(t *testing.T, path string, unix int64) {
	t.Helper()
	when := time.Unix(unix, 0)
	if err := os.Chtimes(path, when, when); err != nil {
		t.Fatalf("chtimes %s: %v", path, err)
	}
}
