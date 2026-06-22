package recommend

import (
	"hash/fnv"
	"sort"
	"strings"

	"github.com/bprendie/subweazl/internal/localstore"
	"github.com/bprendie/subweazl/internal/subsonic"
)

type Result struct {
	Tracks []subsonic.Track
	Rules  []string
}

func Generate(seed subsonic.Track, cached []localstore.CachedTrack, recent map[string]bool, limit int) Result {
	if limit <= 0 {
		limit = 40
	}
	seen := map[string]bool{}
	var out []subsonic.Track
	var rules []string
	appendRule := func(rule string, tracks []localstore.CachedTrack) {
		before := len(out)
		for _, candidate := range tracks {
			track := candidate.Track
			if track.ID == "" || seen[track.ID] || track.ID == seed.ID || recent[track.ID] {
				continue
			}
			seen[track.ID] = true
			out = append(out, track)
			if len(out) >= limit {
				break
			}
		}
		if len(out) > before {
			rules = append(rules, rule)
		}
	}
	if seed.Artist != "" {
		appendRule("same artist", filterCached(cached, func(c localstore.CachedTrack) bool {
			return strings.EqualFold(c.Track.Artist, seed.Artist)
		}))
	}
	if len(out) < limit && seed.Genre != "" {
		appendRule("same genre", filterCached(cached, func(c localstore.CachedTrack) bool {
			return strings.EqualFold(c.Track.Genre, seed.Genre)
		}))
	}
	if len(out) < limit && seed.Year > 0 {
		appendRule("nearby year", filterCached(cached, func(c localstore.CachedTrack) bool {
			return c.Track.Year >= seed.Year-2 && c.Track.Year <= seed.Year+2
		}))
	}
	if len(out) < limit {
		appendRule("starred", filterCached(cached, func(c localstore.CachedTrack) bool { return c.Starred }))
	}
	if len(out) < limit {
		unseen := filterCached(cached, func(c localstore.CachedTrack) bool { return !recent[c.Track.ID] })
		sort.SliceStable(unseen, func(i, j int) bool { return stableScore(unseen[i].Track.ID) < stableScore(unseen[j].Track.ID) })
		appendRule("random unseen", unseen)
	}
	return Result{Tracks: out, Rules: rules}
}

func filterCached(cached []localstore.CachedTrack, keep func(localstore.CachedTrack) bool) []localstore.CachedTrack {
	var out []localstore.CachedTrack
	for _, candidate := range cached {
		if keep(candidate) {
			out = append(out, candidate)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return trackKey(out[i].Track) < trackKey(out[j].Track) })
	return out
}

func trackKey(track subsonic.Track) string {
	return strings.ToLower(track.Artist) + "\x00" + strings.ToLower(track.Album) + "\x00" + strings.ToLower(track.Title) + "\x00" + track.ID
}

func stableScore(id string) uint32 {
	h := fnv.New32a()
	_, _ = h.Write([]byte(id))
	return h.Sum32()
}
