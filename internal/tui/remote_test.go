package tui

import (
	"testing"

	"github.com/bprendie/subweazl/internal/remote"
)

func TestRemoteSnapshotPublishesWeazlState(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	m := newHomeTestModel(t)
	m.EnableRemote()
	snapshot, err := remote.ReadSnapshot()
	if err != nil {
		t.Fatalf("read snapshot: %v", err)
	}
	if !snapshot.Running || snapshot.State != "idle" || snapshot.PlaybackMode != m.playbackModeLabel() {
		t.Fatalf("snapshot = %#v", snapshot)
	}
}

func TestRemoteQuitClosesWeazlVault(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	m := newHomeTestModel(t)
	if m.vaultStore == nil || !m.vaultStore.Unlocked() {
		t.Fatal("test vault is not unlocked")
	}
	next, cmd := m.Update(remote.Quit)
	got := next.(Model)
	if got.vaultStore != nil {
		t.Fatal("remote quit left the Weazl vault open")
	}
	if cmd == nil {
		t.Fatal("remote quit did not return a quit command")
	}
}
