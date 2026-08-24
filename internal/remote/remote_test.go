//go:build !windows

package remote

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRemoteCommandsAreValid(t *testing.T) {
	for _, command := range []Command{Toggle, Next, Previous, Stop, CycleMode, Quit} {
		if !validCommand(command) {
			t.Fatalf("command %q should be accepted", command)
		}
	}
}

func TestSnapshotRoundTripUsesPrivateAtomicFile(t *testing.T) {
	stateHome := t.TempDir()
	t.Setenv("XDG_STATE_HOME", stateHome)
	want := Snapshot{Running: true, State: "playing", Title: "Weazl Signal", Artist: "Artist", Album: "Album", Duration: 240, PlaybackMode: "shuffle"}
	if err := WriteSnapshot(want); err != nil {
		t.Fatalf("write snapshot: %v", err)
	}
	got, err := ReadSnapshot()
	if err != nil || got.Title != want.Title || got.PlaybackMode != want.PlaybackMode || got.UpdatedAt.IsZero() {
		t.Fatalf("snapshot = %#v, err = %v", got, err)
	}
	path, _ := StatePath()
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %v, err = %v", info.Mode().Perm(), err)
	}
	matches, _ := filepath.Glob(filepath.Join(filepath.Dir(path), ".remote-*"))
	if len(matches) != 0 {
		t.Fatalf("temporary files remain: %v", matches)
	}
}

func TestSnapshotRejectsStaleRunningState(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	if err := WriteSnapshot(Snapshot{Running: true, State: "playing"}); err != nil {
		t.Fatal(err)
	}
	path, _ := StatePath()
	data := []byte(`{"running":true,"state":"playing","updated_at":"2000-01-01T00:00:00Z"}`)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadSnapshot(); err == nil {
		t.Fatal("expected stale-state error")
	}
}

func TestUnixTransportDeliversCommandAndRejectsSecondServer(t *testing.T) {
	t.Setenv("XDG_RUNTIME_DIR", t.TempDir())
	received := make(chan Command, 1)
	server, err := Listen(func(command Command) { received <- command })
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer server.Close()
	if second, err := Listen(func(Command) {}); err == nil {
		second.Close()
		t.Fatal("expected duplicate server error")
	}
	if err := Send(Next); err != nil {
		t.Fatalf("send: %v", err)
	}
	select {
	case got := <-received:
		if got != Next {
			t.Fatalf("command = %q", got)
		}
	case <-time.After(time.Second):
		t.Fatal("command was not delivered")
	}
}
