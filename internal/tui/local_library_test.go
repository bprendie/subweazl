package tui

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/list"

	"github.com/bprendie/subweazl/internal/config"
	"github.com/bprendie/subweazl/internal/localstore"
)

func TestLocalVaultCreateRequiresMatchingConfirmation(t *testing.T) {
	store, err := localstore.Open(filepath.Join(t.TempDir(), "library.sqlite3"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	if err := store.Migrate(); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	m := New(config.Config{
		Server:   "https://example.test",
		Username: "user",
		Password: "pass",
	})
	m.localStore = store
	m.localVault = "create-password"
	m.input.SetValue("first")
	next, _ := m.submitLocalVault()
	if next.localVault != "create-confirm" {
		t.Fatalf("localVault = %q, want create-confirm", next.localVault)
	}
	next.input.SetValue("second")
	next, _ = next.submitLocalVault()
	if next.localVault != "create-password" {
		t.Fatalf("localVault after mismatch = %q, want create-password", next.localVault)
	}
	if next.err != "vault passwords did not match" {
		t.Fatalf("err = %q, want mismatch error", next.err)
	}
	hasVault, err := store.HasVault()
	if err != nil {
		t.Fatalf("HasVault: %v", err)
	}
	if hasVault {
		t.Fatal("vault was created despite password mismatch")
	}
}

func TestLocalSongDisplayUsesCleanFilenameFallback(t *testing.T) {
	it := item{
		kind: "local-song",
		local: localTrack{
			Path: "/very/long/path/Artist/Album/01 - Signal.flac",
		},
	}
	if got := it.Title(); got != "01 - Signal" {
		t.Fatalf("Title() = %q, want clean filename", got)
	}
	if got := it.Description(); got != "local file" {
		t.Fatalf("Description() = %q, want local file fallback", got)
	}
}

func TestLocalPaneRendersCompactTrackTable(t *testing.T) {
	m := New(config.Config{Server: "https://example.test", Username: "u", Password: "p"})
	m.mode = modeLocal
	m.width = 120
	m.height = 40
	m.list.SetItems([]list.Item{
		item{kind: "action", title: "Index configured folders", action: "local-index"},
		item{kind: "local-folder", folder: localFolder{Path: "/music/archive", Status: "indexed"}},
		item{kind: "local-song", local: localTrack{
			Title:  "Signal",
			Artist: "The Weazls",
			Album:  "Vault Songs",
			Path:   "/music/archive/01 - Signal.flac",
		}},
	})
	rendered := m.localPane(96, 16)
	for _, want := range []string{"LOCAL VAULT", "FOLDERS", "TRACKS 1", "TITLE", "ARTIST", "ALBUM", "Signal", "The Weazls"} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("local pane missing %q:\n%s", want, rendered)
		}
	}
	if strings.Contains(rendered, "/music/archive/01 - Signal.flac") {
		t.Fatalf("local pane exposed full path:\n%s", rendered)
	}
}

func TestLocalPaneSelectedFolderScopesTrackTable(t *testing.T) {
	m := New(config.Config{Server: "https://example.test", Username: "u", Password: "p"})
	m.mode = modeLocal
	m.list.SetItems([]list.Item{
		item{kind: "local-folder", folder: localFolder{Path: "/music/Artist/Album A", Status: "2 tracks"}},
		item{kind: "local-folder", folder: localFolder{Path: "/music/Artist/Album B", Status: "1 track"}},
		item{kind: "local-song", local: localTrack{
			Title: "A Song",
			Album: "Album A",
			Path:  "/music/Artist/Album A/01 - A Song.flac",
			Dir:   "/music/Artist/Album A",
		}},
		item{kind: "local-song", local: localTrack{
			Title: "B Song",
			Album: "Album B",
			Path:  "/music/Artist/Album B/01 - B Song.flac",
			Dir:   "/music/Artist/Album B",
		}},
	})
	m.list.Select(0)
	rendered := m.localPane(96, 16)
	if !strings.Contains(rendered, "TRACKS 1 - Album A") || !strings.Contains(rendered, "A Song") {
		t.Fatalf("selected folder did not scope table to Album A:\n%s", rendered)
	}
	if strings.Contains(rendered, "B Song") {
		t.Fatalf("selected folder table included another album:\n%s", rendered)
	}
}

func TestLocalFolderTreeCollapseHidesTracks(t *testing.T) {
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
	root := filepath.Join(t.TempDir(), "music")
	album := filepath.Join(root, "Artist", "Album")
	if err := store.UpsertFolder("folder_1", map[string]string{"path": root}); err != nil {
		t.Fatalf("UpsertFolder: %v", err)
	}
	if err := store.UpsertTrack(localstore.TrackRecord{
		ID:       "track_1",
		FolderID: "folder_1",
		Payload: map[string]any{
			"title": "Signal",
			"path":  filepath.Join(album, "01 - Signal.flac"),
		},
	}); err != nil {
		t.Fatalf("UpsertTrack: %v", err)
	}
	m := New(config.Config{
		Server:            "https://example.test",
		Username:          "user",
		Password:          "pass",
		LocalMusicFolders: []string{root},
	})
	m.localStore = store
	m.localOpen[root] = false
	items, _, err := m.localItems()
	if err != nil {
		t.Fatalf("localItems collapsed: %v", err)
	}
	if countKind(items, "local-song") != 0 {
		t.Fatalf("collapsed root exposed tracks: %#v", items)
	}
	m.localOpen[root] = true
	items, _, err = m.localItems()
	if err != nil {
		t.Fatalf("localItems expanded: %v", err)
	}
	if countKind(items, "local-song") != 1 {
		t.Fatalf("expanded root did not expose track: %#v", items)
	}
}

func countKind(items []list.Item, kind string) int {
	count := 0
	for _, row := range items {
		it, ok := row.(item)
		if ok && it.kind == kind {
			count++
		}
	}
	return count
}

func TestUnlockedLocalLibraryShowsIndexAction(t *testing.T) {
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
	m := New(config.Config{
		Server:            "https://example.test",
		Username:          "user",
		Password:          "pass",
		LocalMusicFolders: []string{"/music"},
	})
	m.localStore = store
	items, _, err := m.localItems()
	if err != nil {
		t.Fatalf("localItems: %v", err)
	}
	if len(items) == 0 {
		t.Fatal("localItems returned no rows")
	}
	it, ok := items[0].(item)
	if !ok || it.kind != "action" || it.action != "local-index" {
		t.Fatalf("first row = %#v, want local index action", items[0])
	}
}

func TestIndexLocalFoldersStartsSearching(t *testing.T) {
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
	m := New(config.Config{
		Server:            "https://example.test",
		Username:          "user",
		Password:          "pass",
		LocalMusicFolders: []string{t.TempDir()},
	})
	m.localStore = store
	next, cmd := m.indexLocalFolders()
	if !next.searching {
		t.Fatal("indexLocalFolders did not enter searching state")
	}
	if next.status != "indexing local folders" {
		t.Fatalf("status = %q, want indexing status", next.status)
	}
	if cmd == nil {
		t.Fatal("indexLocalFolders returned nil command")
	}
}
