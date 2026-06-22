package tui

import (
	"fmt"

	"github.com/bprendie/subweazl/internal/playqueue"
	"github.com/bprendie/subweazl/internal/subsonic"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) restoreQueueSnapshot() {
	if m.vaultStore == nil || !m.vaultStore.Unlocked() {
		return
	}
	snapshot, ok, err := m.vaultStore.QueueSnapshot()
	if err != nil {
		m.err = err.Error()
		return
	}
	if ok {
		m.queue = playqueue.FromSnapshot(snapshot)
	}
}

func (m *Model) persistQueue() {
	if m.vaultStore == nil || !m.vaultStore.Unlocked() {
		return
	}
	if err := m.vaultStore.SaveQueueSnapshot(m.queue.Snapshot()); err != nil {
		m.err = err.Error()
	}
}

func (m *Model) showQueue() {
	m.mode = modeQueue
	m.clearNav()
	m.refreshTitle()
	m.list.SetItems(m.queueItems())
	m.status = fmt.Sprintf("queue: %d tracks", len(m.queue.Tracks()))
	m.err = ""
	m.searching = false
	m.input.Blur()
}

func (m Model) queueItems() []list.Item {
	tracks := m.queue.Tracks()
	items := make([]list.Item, 0, len(tracks))
	for i, track := range tracks {
		title := track.Title
		if title == "" {
			title = track.ID
		}
		if i == m.queue.CurrentIndex() {
			title = "▶ " + title
		}
		items = append(items, item{kind: "queue", title: title, desc: trackDescription(track), track: track, queueIndex: i})
	}
	if len(items) == 0 {
		items = append(items, item{kind: "empty", title: "Queue is empty", desc: "play or enqueue tracks to build a queue"})
	}
	return items
}

func (m *Model) refreshQueueView() {
	if m.mode != modeQueue {
		return
	}
	cursor := m.list.Index()
	items := m.queueItems()
	m.list.SetItems(items)
	if cursor < len(items) {
		m.list.Select(cursor)
	}
}

func (m Model) playSelectedTrack(it item) (Model, tea.Cmd) {
	if m.mode == modeQueue {
		return m.playQueueIndex(it.queueIndex)
	}
	tracks, index := m.trackContext(it.track.ID)
	m.queue.Replace(tracks, index)
	m.persistQueue()
	return m, m.play(it.track)
}

func (m Model) playQueueIndex(index int) (Model, tea.Cmd) {
	track, ok := m.queue.SetCurrent(index)
	if !ok {
		m.err = "queue track is unavailable"
		return m, noop
	}
	m.persistQueue()
	m.refreshQueueView()
	return m, m.play(track)
}

func (m Model) playNext() (Model, tea.Cmd) {
	track, ok := m.queue.Next()
	if !ok {
		m.status = "end of queue"
		return m, noop
	}
	m.persistQueue()
	m.refreshQueueView()
	return m, m.play(track)
}

func (m Model) playPrevious() (Model, tea.Cmd) {
	track, ok := m.queue.Previous()
	if !ok {
		m.status = "start of queue"
		return m, noop
	}
	m.persistQueue()
	m.refreshQueueView()
	return m, m.play(track)
}

func (m Model) enqueueSelected() (Model, tea.Cmd) {
	track, ok := m.selectedOrPlayingTrack()
	if !ok {
		m.err = "select or play a song to enqueue"
		return m, noop
	}
	if !m.queue.Append(track) {
		m.err = "track is unavailable"
		return m, noop
	}
	m.persistQueue()
	m.refreshQueueView()
	m.status = "enqueued " + track.Title
	m.err = ""
	return m, noop
}

func (m Model) removeQueueSelection() (Model, tea.Cmd) {
	if m.mode != modeQueue {
		return m, nil
	}
	it, ok := m.list.SelectedItem().(item)
	if !ok || it.kind != "queue" || !m.queue.Remove(it.queueIndex) {
		m.err = "select a queue track to remove"
		return m, noop
	}
	m.persistQueue()
	m.refreshQueueView()
	m.status = "removed from queue"
	m.err = ""
	return m, noop
}

func (m Model) clearQueue() (Model, tea.Cmd) {
	if m.mode != modeQueue {
		return m, nil
	}
	m.queue.Clear()
	m.persistQueue()
	m.refreshQueueView()
	m.status = "queue cleared"
	m.err = ""
	return m, noop
}

func (m Model) moveQueueSelection(delta int) (Model, tea.Cmd) {
	if m.mode != modeQueue {
		return m, nil
	}
	it, ok := m.list.SelectedItem().(item)
	if !ok || it.kind != "queue" || !m.queue.Move(it.queueIndex, delta) {
		m.err = "queue track cannot move farther"
		return m, noop
	}
	m.persistQueue()
	m.refreshQueueView()
	m.list.Select(it.queueIndex + delta)
	m.status = "queue reordered"
	m.err = ""
	return m, noop
}

func (m Model) trackContext(selectedID string) ([]subsonic.Track, int) {
	items := m.list.Items()
	tracks := make([]subsonic.Track, 0, len(items))
	index := 0
	for _, row := range items {
		it, ok := row.(item)
		if !ok || it.kind != "song" || it.track.ID == "" {
			continue
		}
		if it.track.ID == selectedID {
			index = len(tracks)
		}
		tracks = append(tracks, it.track)
	}
	if len(tracks) == 0 {
		return []subsonic.Track{{ID: selectedID}}, 0
	}
	return tracks, index
}

func (m Model) selectedOrPlayingTrack() (subsonic.Track, bool) {
	if track, ok := m.selectedTrack(); ok {
		return track, true
	}
	if m.playing != nil {
		return *m.playing, true
	}
	return subsonic.Track{}, false
}
