package curator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/bprendie/subweazl/internal/llm"
	"github.com/bprendie/subweazl/internal/localstore"
	"github.com/bprendie/subweazl/internal/subsonic"
)

const maxCandidateTracks = 120

type Client interface {
	Complete(ctx context.Context, messages []llm.Message, maxTokens int) (string, error)
}

type Request struct {
	Seed       subsonic.Track
	Candidates []localstore.CachedTrack
	RecentIDs  map[string]bool
	Limit      int
}

type Result struct {
	Tracks   []subsonic.Track
	IDs      []string
	Prompt   string
	Raw      string
	Rejected []string
}

func Generate(ctx context.Context, client Client, req Request) (Result, error) {
	if client == nil {
		return Result{}, errors.New("llm client is required")
	}
	candidates := candidateList(req.Candidates, req.RecentIDs)
	if len(candidates) == 0 {
		return Result{}, errors.New("no cached candidate tracks available")
	}
	limit := req.Limit
	if limit <= 0 {
		limit = 40
	}
	prompt := promptText(req.Seed, candidates, limit)
	raw, err := client.Complete(ctx, []llm.Message{
		{Role: "system", Content: systemPrompt()},
		{Role: "user", Content: prompt},
	}, 900)
	if err != nil {
		return Result{}, err
	}
	ids, rejected, err := validTrackIDs(raw, candidates, limit)
	if err != nil {
		return Result{}, err
	}
	tracksByID := map[string]subsonic.Track{}
	for _, candidate := range candidates {
		tracksByID[candidate.Track.ID] = candidate.Track
	}
	tracks := make([]subsonic.Track, 0, len(ids))
	for _, id := range ids {
		tracks = append(tracks, tracksByID[id])
	}
	return Result{Tracks: tracks, IDs: ids, Prompt: prompt, Raw: raw, Rejected: rejected}, nil
}

func systemPrompt() string {
	return "You curate a private music queue. Return only compact JSON with a track_ids array. Use only provided IDs. Do not invent IDs."
}

func promptText(seed subsonic.Track, candidates []localstore.CachedTrack, limit int) string {
	rows := make([]string, 0, min(len(candidates), maxCandidateTracks)+8)
	rows = append(rows, fmt.Sprintf("Create a queue with up to %d tracks from these candidates.", limit))
	if seed.ID != "" {
		rows = append(rows, "Seed: "+trackLine(seed, false))
	}
	rows = append(rows, "Return JSON shaped like {\"track_ids\":[\"id1\",\"id2\"]}.")
	rows = append(rows, "Candidates:")
	for i, candidate := range candidates {
		if i >= maxCandidateTracks {
			break
		}
		rows = append(rows, trackLine(candidate.Track, candidate.Starred))
	}
	return strings.Join(rows, "\n")
}

func trackLine(track subsonic.Track, starred bool) string {
	parts := []string{
		"id=" + track.ID,
		"title=" + clean(track.Title),
		"artist=" + clean(track.Artist),
		"album=" + clean(track.Album),
	}
	if track.Genre != "" {
		parts = append(parts, "genre="+clean(track.Genre))
	}
	if track.Year > 0 {
		parts = append(parts, fmt.Sprintf("year=%d", track.Year))
	}
	if starred {
		parts = append(parts, "starred=true")
	}
	return strings.Join(parts, " | ")
}

func clean(value string) string {
	value = strings.ReplaceAll(value, "\n", " ")
	value = strings.ReplaceAll(value, "\r", " ")
	return strings.TrimSpace(value)
}

func candidateList(cached []localstore.CachedTrack, recent map[string]bool) []localstore.CachedTrack {
	seen := map[string]bool{}
	out := make([]localstore.CachedTrack, 0, len(cached))
	for _, candidate := range cached {
		id := candidate.Track.ID
		if id == "" || seen[id] || recent[id] {
			continue
		}
		seen[id] = true
		out = append(out, candidate)
	}
	sort.SliceStable(out, func(i, j int) bool {
		return sortKey(out[i].Track) < sortKey(out[j].Track)
	})
	return out
}

func sortKey(track subsonic.Track) string {
	return strings.ToLower(track.Artist) + "\x00" + strings.ToLower(track.Album) + "\x00" + strings.ToLower(track.Title) + "\x00" + track.ID
}

func validTrackIDs(raw string, candidates []localstore.CachedTrack, limit int) ([]string, []string, error) {
	allowed := map[string]bool{}
	for _, candidate := range candidates {
		allowed[candidate.Track.ID] = true
	}
	ids, err := extractIDs(raw)
	if err != nil {
		return nil, nil, err
	}
	seen := map[string]bool{}
	valid := make([]string, 0, min(len(ids), limit))
	var rejected []string
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		if !allowed[id] {
			rejected = append(rejected, id)
			continue
		}
		valid = append(valid, id)
		if len(valid) >= limit {
			break
		}
	}
	if len(valid) == 0 {
		return nil, rejected, errors.New("llm returned no valid cached track IDs")
	}
	return valid, rejected, nil
}

func extractIDs(raw string) ([]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, errors.New("empty llm response")
	}
	var obj struct {
		TrackIDs []string `json:"track_ids"`
		IDs      []string `json:"ids"`
	}
	if err := json.Unmarshal([]byte(raw), &obj); err == nil {
		if len(obj.TrackIDs) > 0 {
			return obj.TrackIDs, nil
		}
		if len(obj.IDs) > 0 {
			return obj.IDs, nil
		}
	}
	var ids []string
	if err := json.Unmarshal([]byte(raw), &ids); err == nil && len(ids) > 0 {
		return ids, nil
	}
	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start >= 0 && end > start {
		return extractIDs(raw[start : end+1])
	}
	return nil, errors.New("llm response did not contain track_ids JSON")
}

func RunPayload(provider, model string, result Result) map[string]any {
	return map[string]any{
		"provider":     provider,
		"model":        model,
		"track_ids":    result.IDs,
		"rejected_ids": result.Rejected,
		"prompt":       result.Prompt,
		"response":     result.Raw,
		"created_at":   time.Now().UTC().Format(time.RFC3339),
	}
}
