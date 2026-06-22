package playqueue

import "github.com/bprendie/subweazl/internal/subsonic"

type Queue struct {
	tracks  []subsonic.Track
	current int
}

type Snapshot struct {
	Tracks  []subsonic.Track `json:"tracks"`
	Current int              `json:"current"`
}

func New() Queue {
	return Queue{current: -1}
}

func FromSnapshot(snapshot Snapshot) Queue {
	q := Queue{current: -1}
	q.Replace(snapshot.Tracks, snapshot.Current)
	return q
}

func (q Queue) Tracks() []subsonic.Track {
	return append([]subsonic.Track(nil), q.tracks...)
}

func (q Queue) CurrentIndex() int {
	return q.current
}

func (q Queue) Current() (subsonic.Track, bool) {
	if q.current < 0 || q.current >= len(q.tracks) {
		return subsonic.Track{}, false
	}
	return q.tracks[q.current], true
}

func (q Queue) Snapshot() Snapshot {
	return Snapshot{Tracks: q.Tracks(), Current: q.current}
}

func (q *Queue) Replace(tracks []subsonic.Track, current int) {
	q.tracks = validTracks(tracks)
	if len(q.tracks) == 0 {
		q.current = -1
		return
	}
	if current < 0 {
		current = 0
	}
	if current >= len(q.tracks) {
		current = len(q.tracks) - 1
	}
	q.current = current
}

func (q *Queue) SetCurrent(index int) (subsonic.Track, bool) {
	if index < 0 || index >= len(q.tracks) {
		return subsonic.Track{}, false
	}
	q.current = index
	return q.tracks[q.current], true
}

func (q *Queue) Append(track subsonic.Track) bool {
	if track.ID == "" {
		return false
	}
	q.tracks = append(q.tracks, track)
	if q.current < 0 {
		q.current = 0
	}
	return true
}

func (q *Queue) Remove(index int) bool {
	if index < 0 || index >= len(q.tracks) {
		return false
	}
	q.tracks = append(q.tracks[:index], q.tracks[index+1:]...)
	if len(q.tracks) == 0 {
		q.current = -1
		return true
	}
	if q.current > index {
		q.current--
	}
	if q.current >= len(q.tracks) {
		q.current = len(q.tracks) - 1
	}
	return true
}

func (q *Queue) Clear() {
	q.tracks = nil
	q.current = -1
}

func (q *Queue) Next() (subsonic.Track, bool) {
	if q.current+1 >= len(q.tracks) {
		return subsonic.Track{}, false
	}
	q.current++
	return q.tracks[q.current], true
}

func (q *Queue) Previous() (subsonic.Track, bool) {
	if q.current <= 0 || len(q.tracks) == 0 {
		return subsonic.Track{}, false
	}
	q.current--
	return q.tracks[q.current], true
}

func (q *Queue) Move(index, delta int) bool {
	target := index + delta
	if index < 0 || index >= len(q.tracks) || target < 0 || target >= len(q.tracks) {
		return false
	}
	q.tracks[index], q.tracks[target] = q.tracks[target], q.tracks[index]
	if q.current == index {
		q.current = target
	} else if q.current == target {
		q.current = index
	}
	return true
}

func validTracks(tracks []subsonic.Track) []subsonic.Track {
	out := make([]subsonic.Track, 0, len(tracks))
	for _, track := range tracks {
		if track.ID != "" {
			out = append(out, track)
		}
	}
	return out
}
