package tui

import (
	"os"
	"testing"
	"time"

	"github.com/bprendie/subweazl/internal/config"
	"github.com/bprendie/subweazl/internal/remote"
)

type countingPublisher struct{ calls int }

func (p *countingPublisher) Publish(remote.Snapshot) { p.calls++ }

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

func TestUnchangedRemoteStateSkipsWritesAndMediaSignals(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	m := New(config.Config{})
	publisher := &countingPublisher{}
	m.SetMediaPublisher(publisher)
	m.EnableRemote()
	path, err := remote.StatePath()
	if err != nil {
		t.Fatal(err)
	}
	before, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	m.publishRemote()
	after, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if !after.ModTime().Equal(before.ModTime()) {
		t.Fatal("unchanged status rewrote the snapshot")
	}
	if publisher.calls != 1 {
		t.Fatalf("media publish calls = %d, want 1", publisher.calls)
	}
	m.remotePublished = time.Now().Add(-31 * time.Second)
	m.publishRemote()
	heartbeat, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if !heartbeat.ModTime().After(after.ModTime()) {
		t.Fatal("heartbeat did not refresh the snapshot")
	}
	if publisher.calls != 1 {
		t.Fatalf("heartbeat emitted media update; calls = %d", publisher.calls)
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
