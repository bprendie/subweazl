package recommend

import (
	"testing"

	"github.com/bprendie/subweazl/internal/localstore"
	"github.com/bprendie/subweazl/internal/subsonic"
)

func TestGenerateUsesSeedRulesAndAvoidsRecent(t *testing.T) {
	seed := track("seed", "Seed", "A", "G", 2001)
	cached := []localstore.CachedTrack{
		{Track: track("seed", "Seed", "A", "G", 2001)},
		{Track: track("recent", "Recent", "A", "G", 2001)},
		{Track: track("artist", "Artist", "A", "X", 1990)},
		{Track: track("genre", "Genre", "B", "G", 1980)},
		{Track: track("year", "Year", "C", "Y", 2002)},
		{Track: track("star", "Star", "D", "Z", 1970), Starred: true},
	}
	result := Generate(seed, cached, map[string]bool{"recent": true}, 4)
	ids := ids(result.Tracks)
	want := []string{"artist", "genre", "year", "star"}
	if !same(ids, want) {
		t.Fatalf("ids = %#v, want %#v", ids, want)
	}
	for _, rule := range []string{"same artist", "same genre", "nearby year", "starred"} {
		if !contains(result.Rules, rule) {
			t.Fatalf("rules = %#v missing %q", result.Rules, rule)
		}
	}
}

func TestGenerateFallsBackToRandomUnseen(t *testing.T) {
	cached := []localstore.CachedTrack{{Track: track("a", "A", "", "", 0)}, {Track: track("b", "B", "", "", 0)}}
	result := Generate(subsonic.Track{}, cached, nil, 2)
	if len(result.Tracks) != 2 || !contains(result.Rules, "random unseen") {
		t.Fatalf("result = %#v", result)
	}
}

func track(id, title, artist, genre string, year int) subsonic.Track {
	return subsonic.Track{ID: id, Title: title, Artist: artist, Album: "Album", Genre: genre, Year: year}
}

func ids(tracks []subsonic.Track) []string {
	out := make([]string, 0, len(tracks))
	for _, track := range tracks {
		out = append(out, track.ID)
	}
	return out
}

func same(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
