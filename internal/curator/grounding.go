package curator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"unicode"

	"github.com/bprendie/subweazl/internal/llm"
	"github.com/bprendie/subweazl/internal/localstore"
	"github.com/bprendie/subweazl/internal/subsonic"
)

const groundingSystemPrompt = `You are DJ-Weazl's library grounding engine. Convert a music request into compact search concepts, then choose only real supplied recordings. Never invent an ID. Return only the requested JSON.`

type Intent struct {
	Artists     []string     `json:"artists"`
	Tracks      []string     `json:"tracks"`
	Genres      []string     `json:"genres"`
	Era         FlexibleText `json:"era"`
	Mood        []string     `json:"mood"`
	Texture     []string     `json:"texture"`
	Purpose     FlexibleText `json:"purpose"`
	SearchTerms []string     `json:"search_terms"`
}

type FlexibleText string

func (f *FlexibleText) UnmarshalJSON(data []byte) error {
	var scalar string
	if err := json.Unmarshal(data, &scalar); err == nil {
		*f = FlexibleText(scalar)
		return nil
	}
	var values []string
	if err := json.Unmarshal(data, &values); err != nil {
		return errors.New("expected a string or array of strings")
	}
	*f = FlexibleText(strings.Join(values, ", "))
	return nil
}

func ResolvePrimarySeed(query string, candidates []localstore.CachedTrack) (subsonic.Track, bool) {
	queryKey := searchable(query)
	allowVariants := recordingVariant(subsonic.Track{Title: query})
	bestScore, bestKey := 0, ""
	var best subsonic.Track
	for _, candidate := range candidates {
		track := candidate.Track
		if track.ID == "" || (!allowVariants && recordingVariant(track)) || curatorBlacklisted(track) {
			continue
		}
		score := seedScore(queryKey, candidate)
		key := sortKey(track)
		if score > bestScore || (score == bestScore && score > 0 && key < bestKey) {
			bestScore, bestKey, best = score, key, track
		}
	}
	return best, bestScore >= 300
}

func DiscoveryPrimarySeed(candidates []localstore.CachedTrack) (subsonic.Track, bool) {
	for _, requireStarred := range []bool{true, false} {
		for _, candidate := range candidates {
			if candidate.Track.ID == "" || candidate.New || curatorBlacklisted(candidate.Track) || recordingVariant(candidate.Track) || (requireStarred && !candidate.Starred) {
				continue
			}
			return candidate.Track, true
		}
	}
	return subsonic.Track{}, false
}

func SelectDeterministicAnchors(seed subsonic.Track, neighborhood []subsonic.Track, cachedIDs map[string]bool) ([]subsonic.Track, error) {
	if seed.ID == "" || !cachedIDs[seed.ID] {
		return nil, errors.New("primary seed is not in the synced cache")
	}
	anchors := []subsonic.Track{seed}
	seenArtist := map[string]bool{normalized(seed.Artist): true}
	seenAlbum := map[string]bool{normalized(seed.Album): true}
	for _, track := range neighborhood {
		artist, album := normalized(track.Artist), normalized(track.Album)
		if track.ID == "" || !cachedIDs[track.ID] || track.ID == seed.ID || recordingVariant(track) || (artist != "" && seenArtist[artist]) || (album != "" && seenAlbum[album]) {
			continue
		}
		anchors = append(anchors, track)
		seenArtist[artist], seenAlbum[album] = true, true
		if len(anchors) == 3 {
			return anchors, nil
		}
	}
	return nil, fmt.Errorf("Navidrome established only %d/3 distinct anchors", len(anchors))
}

func seedScore(query string, candidate localstore.CachedTrack) int {
	if query == "" {
		return 0
	}
	track := candidate.Track
	score := 0
	for value, weight := range map[string]int{searchable(track.Artist): 1000, searchable(track.Title): 900, searchable(track.Album): 550, searchable(track.Genre): 400} {
		if value != "" && containsPhrase(query, value) {
			score += weight + len(value)
		}
	}
	for _, word := range strings.Fields(query) {
		if len(word) < 4 || groundingStopWords[word] {
			continue
		}
		if containsPhrase(searchable(track.Artist), word) {
			score += 120
		}
		if containsPhrase(searchable(track.Genre), word) {
			score += 100
		}
		if containsPhrase(searchable(track.Title), word) {
			score += 50
		}
	}
	if candidate.Starred {
		score += 10
	}
	return score
}

var groundingStopWords = map[string]bool{"like": true, "with": true, "music": true, "tracks": true, "songs": true, "playlist": true, "focus": true}

func searchable(value string) string {
	value = strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			return unicode.ToLower(r)
		}
		return ' '
	}, value)
	return strings.Join(strings.Fields(value), " ")
}

func containsPhrase(text, phrase string) bool { return strings.Contains(" "+text+" ", " "+phrase+" ") }

func InterpretIntent(ctx context.Context, client Client, query string) (Intent, error) {
	if client == nil {
		return Intent{}, errors.New("llm client is required")
	}
	prompt := `Interpret this request for searching a private music library.
Return one JSON object with keys artists, tracks, genres, era, mood, texture, purpose, and search_terms.
All values except era and purpose are arrays of short strings. Include named artists verbatim. Search terms must be useful against track title, artist, album, and genre metadata.
Request: ` + clean(query)
	raw, err := client.Complete(ctx, []llm.Message{{Role: "system", Content: groundingSystemPrompt}, {Role: "user", Content: prompt}}, 400)
	if err != nil {
		return Intent{}, err
	}
	var intent Intent
	if err := decodeJSONObject(raw, &intent); err != nil {
		return Intent{}, fmt.Errorf("interpret music request: %w", err)
	}
	intent.SearchTerms = uniqueTerms(append(append(append(intent.Artists, intent.Tracks...), intent.Genres...), intent.SearchTerms...))
	if len(intent.SearchTerms) == 0 {
		return Intent{}, errors.New("DJ-Weazl could not derive library search terms")
	}
	return intent, nil
}

func SelectAnchors(ctx context.Context, client Client, query string, intent Intent, candidates []subsonic.Track) ([]subsonic.Track, error) {
	if len(candidates) < 3 {
		return nil, fmt.Errorf("need three credible anchor candidates; found %d", len(candidates))
	}
	if len(candidates) > 120 {
		candidates = candidates[:120]
	}
	var rows []string
	for _, track := range candidates {
		rows = append(rows, anchorTrackLine(track))
	}
	intentJSON, _ := json.Marshal(intent)
	prompt := fmt.Sprintf(`Choose exactly three anchor recordings for the user's request.
Each anchor must independently fit. Prefer different artists and allow at most one track per album.
Reject covers, remixes, live, acoustic, novelty, karaoke, and tribute versions unless explicitly requested.
Return only {"track_ids":["id1","id2","id3"]}.
User request: %s
Interpreted intent: %s
ANCHOR CANDIDATES:
%s`, clean(query), intentJSON, strings.Join(rows, "\n"))
	raw, err := client.Complete(ctx, []llm.Message{{Role: "system", Content: groundingSystemPrompt}, {Role: "user", Content: prompt}}, 500)
	if err != nil {
		return nil, err
	}
	var payload struct {
		TrackIDs []string `json:"track_ids"`
	}
	if err := decodeJSONObject(raw, &payload); err != nil {
		return nil, fmt.Errorf("select anchors: %w", err)
	}
	byID := make(map[string]subsonic.Track, len(candidates))
	for _, track := range candidates {
		byID[track.ID] = track
	}
	var anchors []subsonic.Track
	seenID, seenArtist, seenAlbum := map[string]bool{}, map[string]bool{}, map[string]bool{}
	for _, id := range payload.TrackIDs {
		track, ok := byID[strings.TrimSpace(id)]
		artist, album := normalized(track.Artist), normalized(track.Album)
		if !ok || seenID[track.ID] || recordingVariant(track) || (artist != "" && seenArtist[artist]) || (album != "" && seenAlbum[album]) {
			continue
		}
		seenID[track.ID], seenArtist[artist], seenAlbum[album] = true, true, true
		anchors = append(anchors, track)
		if len(anchors) == 3 {
			return anchors, nil
		}
	}
	return nil, fmt.Errorf("DJ-Weazl established only %d/3 credible distinct anchors", len(anchors))
}

func MergeSimilarityCrate(anchors []subsonic.Track, neighborhoods [][]subsonic.Track) []subsonic.Track {
	type ranked struct {
		track subsonic.Track
		hits  int
		first int
	}
	byID := map[string]*ranked{}
	order := 0
	for _, anchor := range anchors {
		if anchor.ID != "" {
			byID[anchor.ID] = &ranked{track: anchor, hits: len(neighborhoods) + 1, first: order}
			order++
		}
	}
	for _, neighborhood := range neighborhoods {
		seenNeighborhood := map[string]bool{}
		for _, track := range neighborhood {
			if track.ID == "" || seenNeighborhood[track.ID] || recordingVariant(track) {
				continue
			}
			seenNeighborhood[track.ID] = true
			if existing := byID[track.ID]; existing != nil {
				existing.hits++
				continue
			}
			byID[track.ID] = &ranked{track: track, hits: 1, first: order}
			order++
		}
	}
	rankedTracks := make([]ranked, 0, len(byID))
	for _, entry := range byID {
		rankedTracks = append(rankedTracks, *entry)
	}
	sort.SliceStable(rankedTracks, func(i, j int) bool {
		if rankedTracks[i].hits != rankedTracks[j].hits {
			return rankedTracks[i].hits > rankedTracks[j].hits
		}
		return rankedTracks[i].first < rankedTracks[j].first
	})
	out := make([]subsonic.Track, 0, len(rankedTracks))
	for _, entry := range rankedTracks {
		out = append(out, entry.track)
	}
	return out
}

func decodeJSONObject(raw string, target any) error {
	start, end := strings.Index(raw, "{"), strings.LastIndex(raw, "}")
	if start < 0 || end < start {
		return errors.New("response did not contain a JSON object")
	}
	return json.Unmarshal([]byte(raw[start:end+1]), target)
}

func uniqueTerms(values []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, value := range values {
		value = strings.TrimSpace(value)
		key := normalized(value)
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, value)
	}
	return out
}

func normalized(value string) string { return strings.ToLower(strings.TrimSpace(value)) }

func recordingVariant(track subsonic.Track) bool {
	text := normalized(track.Title + " " + track.Album)
	for _, marker := range []string{" live", "remix", "acoustic", "karaoke", "tribute", "cover version"} {
		if strings.Contains(" "+text, marker) {
			return true
		}
	}
	return false
}

func anchorTrackLine(track subsonic.Track) string {
	return strings.Join([]string{"id=" + track.ID, "title=" + clean(track.Title), "artist=" + clean(track.Artist), "album=" + clean(track.Album), "genre=" + clean(track.Genre)}, " | ")
}
