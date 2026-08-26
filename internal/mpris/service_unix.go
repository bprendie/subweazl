//go:build !windows

package mpris

import (
	"crypto/sha256"
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/bprendie/subweazl/internal/remote"
	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
)

const (
	busName         = "org.mpris.MediaPlayer2.subweazl"
	objectPath      = dbus.ObjectPath("/org/mpris/MediaPlayer2")
	rootInterface   = "org.mpris.MediaPlayer2"
	playerInterface = "org.mpris.MediaPlayer2.Player"
	propsInterface  = "org.freedesktop.DBus.Properties"
)

type service struct {
	conn   *dbus.Conn
	action func(remote.Command)
	mu     sync.RWMutex
	state  remote.Snapshot
}

func Start(action func(remote.Command)) (Service, error) {
	conn, err := dbus.SessionBusPrivate()
	if err != nil {
		return nil, fmt.Errorf("connect to session bus: %w", err)
	}
	if err := conn.Auth(nil); err != nil {
		conn.Close()
		return nil, fmt.Errorf("authenticate session bus: %w", err)
	}
	if err := conn.Hello(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("join session bus: %w", err)
	}
	reply, err := conn.RequestName(busName, dbus.NameFlagDoNotQueue)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("claim MPRIS name: %w", err)
	}
	if reply != dbus.RequestNameReplyPrimaryOwner {
		conn.Close()
		return nil, fmt.Errorf("Subweazl MPRIS player is already running")
	}
	s := &service{conn: conn, action: action, state: remote.Snapshot{Running: true, State: "idle", PlaybackMode: "off"}}
	if err := conn.ExportMethodTable(s.rootMethods(), objectPath, rootInterface); err != nil {
		conn.Close()
		return nil, err
	}
	if err := conn.ExportMethodTable(s.playerMethods(), objectPath, playerInterface); err != nil {
		conn.Close()
		return nil, err
	}
	if err := conn.ExportMethodTable(s.propertyMethods(), objectPath, propsInterface); err != nil {
		conn.Close()
		return nil, err
	}
	node := introspectionNode()
	if err := conn.Export(introspect.NewIntrospectable(node), objectPath, introspect.IntrospectData.Name); err != nil {
		conn.Close()
		return nil, err
	}
	return s, nil
}

func (s *service) Close() error {
	_, _ = s.conn.ReleaseName(busName)
	return s.conn.Close()
}

func (s *service) Publish(snapshot remote.Snapshot) {
	s.mu.Lock()
	before := s.playerPropertiesLocked()
	s.state = snapshot
	after := s.playerPropertiesLocked()
	s.mu.Unlock()
	changed := make(map[string]dbus.Variant)
	for key, value := range after {
		if old, ok := before[key]; !ok || !reflect.DeepEqual(old.Value(), value.Value()) {
			changed[key] = value
		}
	}
	if len(changed) > 0 {
		_ = s.conn.Emit(objectPath, propsInterface+".PropertiesChanged", playerInterface, changed, []string{})
	}
}

func (s *service) Get(iface, property string) (dbus.Variant, *dbus.Error) {
	properties, ok := s.properties(iface)
	if !ok {
		return dbus.Variant{}, dbus.NewError("org.freedesktop.DBus.Error.UnknownInterface", []any{iface})
	}
	value, ok := properties[property]
	if !ok {
		return dbus.Variant{}, dbus.NewError("org.freedesktop.DBus.Error.UnknownProperty", []any{property})
	}
	return value, nil
}

func (s *service) GetAll(iface string) (map[string]dbus.Variant, *dbus.Error) {
	properties, ok := s.properties(iface)
	if !ok {
		return nil, dbus.NewError("org.freedesktop.DBus.Error.UnknownInterface", []any{iface})
	}
	return properties, nil
}

func (s *service) Set(iface, property string, value dbus.Variant) *dbus.Error {
	return dbus.NewError("org.freedesktop.DBus.Error.PropertyReadOnly", []any{iface + "." + property})
}

func (s *service) properties(iface string) (map[string]dbus.Variant, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	switch iface {
	case rootInterface:
		return rootProperties(), true
	case playerInterface:
		return s.playerPropertiesLocked(), true
	default:
		return nil, false
	}
}

func rootProperties() map[string]dbus.Variant {
	return variants(map[string]any{
		"CanQuit": false, "CanRaise": false, "HasTrackList": false,
		"Identity": "Subweazl", "DesktopEntry": "subweazl",
		"SupportedUriSchemes": []string{}, "SupportedMimeTypes": []string{},
	})
}

func (s *service) playerPropertiesLocked() map[string]dbus.Variant {
	state := s.state
	status := "Stopped"
	if state.State == "playing" {
		status = "Playing"
	} else if state.State == "paused" {
		status = "Paused"
	}
	metadata := map[string]dbus.Variant{
		"mpris:trackid": dbus.MakeVariant(trackPath(state)),
	}
	if state.Title != "" {
		metadata["xesam:title"] = dbus.MakeVariant(state.Title)
	}
	if state.Artist != "" {
		metadata["xesam:artist"] = dbus.MakeVariant([]string{state.Artist})
	}
	if state.Album != "" {
		metadata["xesam:album"] = dbus.MakeVariant(state.Album)
	}
	if state.Duration > 0 {
		metadata["mpris:length"] = dbus.MakeVariant(int64(state.Duration) * 1_000_000)
	}
	active := state.Running && state.Title != ""
	loopStatus := "None"
	if strings.Contains(state.PlaybackMode, "repeat") {
		loopStatus = "Playlist"
	}
	return variants(map[string]any{
		"PlaybackStatus": status, "LoopStatus": loopStatus, "Rate": 1.0,
		"Shuffle":  strings.Contains(state.PlaybackMode, "shuffle"),
		"Metadata": metadata, "Volume": 1.0, "Position": int64(0),
		"MinimumRate": 1.0, "MaximumRate": 1.0,
		"CanGoNext": active, "CanGoPrevious": active, "CanPlay": active,
		"CanPause": active, "CanSeek": false, "CanControl": true,
	})
}

func variants(values map[string]any) map[string]dbus.Variant {
	result := make(map[string]dbus.Variant, len(values))
	for key, value := range values {
		result[key] = dbus.MakeVariant(value)
	}
	return result
}

func trackPath(snapshot remote.Snapshot) dbus.ObjectPath {
	if snapshot.Title == "" {
		return dbus.ObjectPath("/org/mpris/MediaPlayer2/TrackList/NoTrack")
	}
	sum := sha256.Sum256([]byte(snapshot.Title + "\x00" + snapshot.Artist + "\x00" + snapshot.Album))
	return dbus.ObjectPath(fmt.Sprintf("/org/mpris/MediaPlayer2/track/%x", sum[:8]))
}

func (s *service) raise() *dbus.Error    { return nil }
func (s *service) quit() *dbus.Error     { s.send(remote.Quit); return nil }
func (s *service) next() *dbus.Error     { s.send(remote.Next); return nil }
func (s *service) previous() *dbus.Error { s.send(remote.Previous); return nil }
func (s *service) pause() *dbus.Error {
	if s.playbackStatus() == "Playing" {
		s.send(remote.Toggle)
	}
	return nil
}
func (s *service) playPause() *dbus.Error { s.send(remote.Toggle); return nil }
func (s *service) stop() *dbus.Error      { s.send(remote.Stop); return nil }
func (s *service) play() *dbus.Error {
	if s.playbackStatus() == "Paused" {
		s.send(remote.Toggle)
	}
	return nil
}
func (s *service) seek(int64) *dbus.Error                         { return unsupported("Seek") }
func (s *service) setPosition(dbus.ObjectPath, int64) *dbus.Error { return unsupported("SetPosition") }
func (s *service) openURI(string) *dbus.Error                     { return unsupported("OpenUri") }

func (s *service) rootMethods() map[string]any {
	return map[string]any{"Raise": s.raise, "Quit": s.quit}
}

func (s *service) playerMethods() map[string]any {
	return map[string]any{
		"Next": s.next, "Previous": s.previous, "Pause": s.pause,
		"PlayPause": s.playPause, "Stop": s.stop, "Play": s.play,
		"Seek": s.seek, "SetPosition": s.setPosition, "OpenUri": s.openURI,
	}
}

func (s *service) propertyMethods() map[string]any {
	return map[string]any{"Get": s.Get, "GetAll": s.GetAll, "Set": s.Set}
}

func (s *service) send(command remote.Command) {
	if s.action != nil {
		s.action(command)
	}
}

func (s *service) playbackStatus() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.state.State == "playing" {
		return "Playing"
	}
	if s.state.State == "paused" {
		return "Paused"
	}
	return "Stopped"
}

func unsupported(method string) *dbus.Error {
	return dbus.NewError("org.mpris.MediaPlayer2.Error.NotSupported", []any{method + " is not supported"})
}

func introspectionNode() *introspect.Node {
	rootProps := []introspect.Property{
		{Name: "CanQuit", Type: "b", Access: "read"}, {Name: "CanRaise", Type: "b", Access: "read"},
		{Name: "HasTrackList", Type: "b", Access: "read"}, {Name: "Identity", Type: "s", Access: "read"},
		{Name: "DesktopEntry", Type: "s", Access: "read"}, {Name: "SupportedUriSchemes", Type: "as", Access: "read"},
		{Name: "SupportedMimeTypes", Type: "as", Access: "read"},
	}
	playerProps := []introspect.Property{
		{Name: "PlaybackStatus", Type: "s", Access: "read"}, {Name: "LoopStatus", Type: "s", Access: "read"},
		{Name: "Rate", Type: "d", Access: "read"}, {Name: "Shuffle", Type: "b", Access: "read"},
		{Name: "Metadata", Type: "a{sv}", Access: "read"}, {Name: "Volume", Type: "d", Access: "read"},
		{Name: "Position", Type: "x", Access: "read"}, {Name: "MinimumRate", Type: "d", Access: "read"},
		{Name: "MaximumRate", Type: "d", Access: "read"}, {Name: "CanGoNext", Type: "b", Access: "read"},
		{Name: "CanGoPrevious", Type: "b", Access: "read"}, {Name: "CanPlay", Type: "b", Access: "read"},
		{Name: "CanPause", Type: "b", Access: "read"}, {Name: "CanSeek", Type: "b", Access: "read"},
		{Name: "CanControl", Type: "b", Access: "read"},
	}
	return &introspect.Node{Name: string(objectPath), Interfaces: []introspect.Interface{
		introspect.IntrospectData,
		{Name: rootInterface, Methods: []introspect.Method{{Name: "Raise"}, {Name: "Quit"}}, Properties: rootProps},
		{Name: playerInterface, Methods: []introspect.Method{
			{Name: "Next"}, {Name: "Previous"}, {Name: "Pause"}, {Name: "PlayPause"}, {Name: "Stop"}, {Name: "Play"},
			{Name: "Seek", Args: []introspect.Arg{{Name: "Offset", Type: "x", Direction: "in"}}},
			{Name: "SetPosition", Args: []introspect.Arg{{Name: "TrackId", Type: "o", Direction: "in"}, {Name: "Position", Type: "x", Direction: "in"}}},
			{Name: "OpenUri", Args: []introspect.Arg{{Name: "Uri", Type: "s", Direction: "in"}}},
		}, Signals: []introspect.Signal{
			{Name: "Seeked", Args: []introspect.Arg{{Name: "Position", Type: "x"}}},
		}, Properties: playerProps},
		{Name: propsInterface, Methods: []introspect.Method{
			{Name: "Get", Args: []introspect.Arg{{Name: "Interface", Type: "s", Direction: "in"}, {Name: "Property", Type: "s", Direction: "in"}, {Name: "Value", Type: "v", Direction: "out"}}},
			{Name: "Set", Args: []introspect.Arg{{Name: "Interface", Type: "s", Direction: "in"}, {Name: "Property", Type: "s", Direction: "in"}, {Name: "Value", Type: "v", Direction: "in"}}},
			{Name: "GetAll", Args: []introspect.Arg{{Name: "Interface", Type: "s", Direction: "in"}, {Name: "Properties", Type: "a{sv}", Direction: "out"}}},
		}, Signals: []introspect.Signal{
			{Name: "PropertiesChanged", Args: []introspect.Arg{{Name: "Interface", Type: "s"}, {Name: "Changed", Type: "a{sv}"}, {Name: "Invalidated", Type: "as"}}},
		}},
	}}
}
