//go:build !windows

package mpris

import (
	"testing"

	"github.com/bprendie/subweazl/internal/remote"
	"github.com/godbus/dbus/v5"
)

func TestPlayerPropertiesExposePlaybackWithoutPrivateState(t *testing.T) {
	s := &service{state: remote.Snapshot{
		Running: true, State: "playing", Title: "Weazl Signal", Artist: "Artist",
		Album: "Album", Duration: 240, PlaybackMode: "shuffle/repeat",
	}}
	props := s.playerPropertiesLocked()
	if got := props["PlaybackStatus"].Value(); got != "Playing" {
		t.Fatalf("playback status = %v", got)
	}
	if got := props["LoopStatus"].Value(); got != "Playlist" {
		t.Fatalf("loop status = %v", got)
	}
	if got := props["Shuffle"].Value(); got != true {
		t.Fatalf("shuffle = %v", got)
	}
	metadata, ok := props["Metadata"].Value().(map[string]dbus.Variant)
	if !ok {
		t.Fatalf("metadata type = %T", props["Metadata"].Value())
	}
	allowed := map[string]bool{"mpris:trackid": true, "mpris:length": true, "xesam:title": true, "xesam:artist": true, "xesam:album": true}
	for key := range metadata {
		if !allowed[key] {
			t.Fatalf("MPRIS metadata exposed unexpected field %q", key)
		}
	}
}

func TestMPRISMethodsRouteCommands(t *testing.T) {
	commands := make(chan remote.Command, 4)
	s := &service{action: func(command remote.Command) { commands <- command }, state: remote.Snapshot{State: "playing"}}
	s.next()
	s.pause()
	s.stop()
	for _, want := range []remote.Command{remote.Next, remote.Toggle, remote.Stop} {
		if got := <-commands; got != want {
			t.Fatalf("command = %q, want %q", got, want)
		}
	}
}

func TestIntrospectionContainsRequiredInterfaces(t *testing.T) {
	node := introspectionNode()
	found := map[string]bool{}
	for _, iface := range node.Interfaces {
		found[iface.Name] = true
	}
	for _, name := range []string{rootInterface, playerInterface, propsInterface} {
		if !found[name] {
			t.Fatalf("introspection missing %s", name)
		}
	}
}
