package tui

import (
	"fmt"
	"math/rand"

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
	m.queueSourceID = ""
	if m.mode == modeTracks {
		m.queueSourceID = m.playlistViewID
	}
	m.queueTitle = "queue"
	m.resetPlaybackTraversal()
	m.persistQueue()
	return m, m.play(it.track)
}

func (m *Model) followPlayingTrack() {
	if m.playing == nil {
		return
	}
	queueIndex := m.queue.CurrentIndex()
	if m.mode == modeQueue {
		if queueIndex >= 0 && queueIndex < len(m.list.Items()) {
			m.list.Select(queueIndex)
		}
		return
	}
	if m.mode != modeTracks || m.playlistViewID == "" || m.playlistViewID != m.queueSourceID {
		return
	}
	items := m.list.Items()
	if queueIndex >= 0 && queueIndex < len(items) {
		if it, ok := items[queueIndex].(item); ok && it.track.ID == m.playing.ID {
			m.list.Select(queueIndex)
			return
		}
	}
	for i, row := range items {
		if it, ok := row.(item); ok && it.track.ID == m.playing.ID {
			m.list.Select(i)
			return
		}
	}
}

func (m Model) playQueueIndex(index int) (Model, tea.Cmd) {
	track, ok := m.queue.SetCurrent(index)
	if !ok {
		m.err = "queue track is unavailable"
		return m, noop
	}
	m.persistQueue()
	m.resetPlaybackTraversal()
	m.refreshQueueView()
	return m, m.play(track)
}

func (m Model) playNext() (Model, tea.Cmd) {
	current := m.queue.CurrentIndex()
	var track subsonic.Track
	var ok bool
	if m.playbackMode == playbackShuffle || m.playbackMode == playbackShuffleRepeat {
		if len(m.shuffleUpcoming) == 0 && (!m.shuffleReady || m.playbackMode == playbackShuffleRepeat) {
			m.fillShuffleUpcoming()
		}
		if len(m.shuffleUpcoming) > 0 {
			index := m.shuffleUpcoming[0]
			m.shuffleUpcoming = m.shuffleUpcoming[1:]
			track, ok = m.queue.SetCurrent(index)
		}
	} else {
		track, ok = m.queue.Next()
		if !ok && m.playbackMode == playbackRepeat {
			track, ok = m.queue.SetCurrent(0)
		}
	}
	if !ok {
		m.stop()
		m.status = "end of queue"
		return m, noop
	}
	if current >= 0 && current != m.queue.CurrentIndex() {
		m.playHistory = append(m.playHistory, current)
	}
	m.persistQueue()
	m.refreshQueueView()
	return m, m.play(track)
}

func (m Model) playPrevious() (Model, tea.Cmd) {
	var track subsonic.Track
	var ok bool
	if (m.playbackMode == playbackShuffle || m.playbackMode == playbackShuffleRepeat) && len(m.playHistory) > 0 {
		last := len(m.playHistory) - 1
		track, ok = m.queue.SetCurrent(m.playHistory[last])
		m.playHistory = m.playHistory[:last]
	} else {
		track, ok = m.queue.Previous()
		if !ok && m.playbackMode == playbackRepeat {
			track, ok = m.queue.SetCurrent(len(m.queue.Tracks()) - 1)
		}
	}
	if !ok {
		m.status = "start of queue"
		return m, noop
	}
	m.persistQueue()
	m.refreshQueueView()
	return m, m.play(track)
}

func (m *Model) cyclePlaybackMode() {
	m.playbackMode = (m.playbackMode + 1) % 4
	m.resetPlaybackTraversal()
	m.status = "playback mode: " + m.playbackModeLabel()
	m.publishRemote(true)
}

func (m Model) playbackModeLabel() string {
	switch m.playbackMode {
	case playbackShuffle:
		return "shuffle"
	case playbackShuffleRepeat:
		return "shuffle/repeat"
	case playbackRepeat:
		return "repeat"
	default:
		return "off"
	}
}

func (m *Model) resetPlaybackTraversal() {
	m.shuffleUpcoming = nil
	m.shuffleReady = false
	m.playHistory = nil
}

func (m *Model) fillShuffleUpcoming() {
	m.shuffleReady = true
	tracks := m.queue.Tracks()
	if len(tracks) < 2 {
		if len(tracks) == 1 && m.playbackMode == playbackShuffleRepeat {
			m.shuffleUpcoming = []int{0}
		}
		return
	}
	current := m.queue.CurrentIndex()
	indexes := make([]int, 0, len(tracks)-1)
	for i := range tracks {
		if i != current {
			indexes = append(indexes, i)
		}
	}
	rand.Shuffle(len(indexes), func(i, j int) { indexes[i], indexes[j] = indexes[j], indexes[i] })
	m.shuffleUpcoming = indexes
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
