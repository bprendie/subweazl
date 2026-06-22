package localstore

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateVaultUnlocksStore(t *testing.T) {
	store := newMigratedStore(t)
	has, err := store.HasVault()
	if err != nil {
		t.Fatalf("HasVault: %v", err)
	}
	if has {
		t.Fatal("new store unexpectedly has a vault")
	}
	if err := store.CreateVault("swordfish"); err != nil {
		t.Fatalf("CreateVault: %v", err)
	}
	if !store.Unlocked() {
		t.Fatal("store was not unlocked after vault creation")
	}
	has, err = store.HasVault()
	if err != nil {
		t.Fatalf("HasVault after create: %v", err)
	}
	if !has {
		t.Fatal("vault was not created")
	}
}

func TestUnlockRejectsWrongPassword(t *testing.T) {
	path := filepath.Join(t.TempDir(), "library.sqlite3")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := store.Migrate(); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if err := store.CreateVault("right-password"); err != nil {
		t.Fatalf("CreateVault: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	store, err = Open(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer store.Close()
	if err := store.Unlock("wrong-password"); err == nil {
		t.Fatal("wrong password unlocked the store")
	}
	if store.Unlocked() {
		t.Fatal("store is unlocked after wrong password")
	}
	if err := store.Unlock("right-password"); err != nil {
		t.Fatalf("Unlock right password: %v", err)
	}
	if !store.Unlocked() {
		t.Fatal("store did not unlock with right password")
	}
}

func TestEncryptDecryptRequiresUnlock(t *testing.T) {
	store := newMigratedStore(t)
	if _, err := store.encrypt("secret"); !errors.Is(err, errLocked) {
		t.Fatalf("encrypt locked error = %v, want errLocked", err)
	}
	if _, err := store.decrypt("not-base64"); !errors.Is(err, errLocked) {
		t.Fatalf("decrypt locked error = %v, want errLocked", err)
	}
}

func TestEncryptDecryptRoundTrip(t *testing.T) {
	store := newMigratedStore(t)
	if err := store.CreateVault("pw"); err != nil {
		t.Fatalf("CreateVault: %v", err)
	}
	plain := `{"path":"/music/secret.flac","title":"Hidden Signal"}`
	blob, err := store.encrypt(plain)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	if blob == plain || strings.Contains(blob, "Hidden Signal") || strings.Contains(blob, "/music") {
		t.Fatalf("encrypted blob exposes plaintext: %q", blob)
	}
	got, err := store.decrypt(blob)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if got != plain {
		t.Fatalf("decrypt = %q, want %q", got, plain)
	}
}

func TestVaultHashDoesNotStorePlaintext(t *testing.T) {
	store := newMigratedStore(t)
	if err := store.CreateVault("plain-password"); err != nil {
		t.Fatalf("CreateVault: %v", err)
	}
	var hash string
	if err := store.db.QueryRow(`select password_hash from vault where id = 1`).Scan(&hash); err != nil {
		t.Fatalf("query hash: %v", err)
	}
	if hash == "plain-password" || strings.Contains(hash, "plain-password") {
		t.Fatalf("password hash exposes plaintext: %q", hash)
	}
}

func newMigratedStore(t *testing.T) *Store {
	t.Helper()
	store, err := Open(filepath.Join(t.TempDir(), "library.sqlite3"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	if err := store.Migrate(); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	return store
}
