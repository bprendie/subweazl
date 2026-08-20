package curator

import (
	"context"
	_ "embed"
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

const (
	maxCandidateTracks = 240
	maxAttempts        = 5
)

//go:embed prompts/dj_weazl_draft.md
var systemPromptText string

type Mode string

const (
	ModeAIMix Mode = "ai_mix"
	ModeMood  Mode = "mood"
)

type Client interface {
	Complete(ctx context.Context, messages []llm.Message, maxTokens int) (string, error)
}

type StreamingClient interface {
	StreamComplete(ctx context.Context, messages []llm.Message, maxTokens int, onDelta func(string) error) error
}

type Request struct {
	Seed       subsonic.Track
	Candidates []localstore.CachedTrack
	RecentIDs  map[string]bool
	Preferred  map[string]bool
	Limit      int
	Mode       Mode
}

type Attempt struct {
	Prompt   string   `json:"prompt"`
	Response string   `json:"response"`
	Accepted []string `json:"accepted_ids"`
	Rejected []string `json:"rejected_ids"`
}

type Result struct {
	Tracks   []subsonic.Track
	IDs      []string
	Prompt   string
	Raw      string
	Rejected []string
	Attempts []Attempt
	Fallback []string
}

func Generate(ctx context.Context, client Client, req Request) (Result, error) {
	if client == nil {
		return Result{}, errors.New("llm client is required")
	}
	limit := req.Limit
	if limit <= 0 {
		limit = 20
	}
	candidates := candidateList(req.Candidates, req.RecentIDs, req.Seed.ID, req.Preferred)
	if req.Mode == ModeMood && (req.Seed.ID == "" || !containsCandidate(candidates, req.Seed.ID)) {
		return Result{}, errors.New("currently playing track is not available in the synced cache")
	}
	if len(candidates) < limit {
		return Result{}, fmt.Errorf("need %d eligible cached tracks; found %d", limit, len(candidates))
	}
	accepted := make([]string, 0, limit)
	acceptedSet := map[string]bool{}
	if req.Mode == ModeMood {
		accepted = append(accepted, req.Seed.ID)
		acceptedSet[req.Seed.ID] = true
	}
	var attempts []Attempt
	var allRejected []string
	for attempt := 1; attempt <= maxAttempts && len(accepted) < limit; attempt++ {
		remaining := remainingCandidates(candidates, acceptedSet, req.Mode, req.Preferred)
		need := limit - len(accepted)
		prompt := promptText(req, remaining, need, attempt, accepted)
		raw, err := client.Complete(ctx, []llm.Message{
			{Role: "system", Content: systemPrompt()},
			{Role: "user", Content: prompt},
		}, 400)
		if err != nil {
			return resultFrom(candidates, accepted, attempts, allRejected), err
		}
		ids, err := extractIDs(raw)
		if err != nil {
			attempts = append(attempts, Attempt{Prompt: prompt, Response: raw, Rejected: []string{"invalid_json"}})
			allRejected = append(allRejected, "invalid_json")
			continue
		}
		allowed := candidateIDSet(remaining)
		var added, rejected []string
		seenResponse := map[string]bool{}
		for _, id := range ids {
			id = strings.TrimSpace(id)
			if id == "" || seenResponse[id] || acceptedSet[id] || !allowed[id] {
				if id != "" {
					rejected = append(rejected, id)
				}
				continue
			}
			seenResponse[id] = true
			acceptedSet[id] = true
			accepted = append(accepted, id)
			added = append(added, id)
			if len(accepted) == limit {
				break
			}
		}
		attempts = append(attempts, Attempt{Prompt: prompt, Response: raw, Accepted: added, Rejected: rejected})
		allRejected = append(allRejected, rejected...)
	}
	var fallback []string
	if len(accepted) != limit {
		for _, candidate := range remainingCandidates(candidates, acceptedSet, req.Mode, req.Preferred) {
			if acceptedSet[candidate.Track.ID] {
				continue
			}
			acceptedSet[candidate.Track.ID] = true
			accepted = append(accepted, candidate.Track.ID)
			fallback = append(fallback, candidate.Track.ID)
			if len(accepted) == limit {
				break
			}
		}
	}
	if len(accepted) != limit {
		return resultFrom(candidates, accepted, attempts, allRejected), fmt.Errorf("curator could only fill %d/%d unique cached tracks", len(accepted), limit)
	}
	result := resultFrom(candidates, accepted, attempts, allRejected)
	result.Fallback = fallback
	return result, nil
}

// GenerateStreaming validates opaque IDs as soon as their closing quote
// arrives. The provider request is cancelled once the playlist is full.
func GenerateStreaming(ctx context.Context, client StreamingClient, req Request, onAccept func(subsonic.Track, int, int) error) (Result, error) {
	if client == nil {
		return Result{}, errors.New("streaming llm client is required")
	}
	limit := req.Limit
	if limit <= 0 {
		limit = 20
	}
	candidates := candidateList(req.Candidates, req.RecentIDs, req.Seed.ID, req.Preferred)
	if req.Mode == ModeMood && (req.Seed.ID == "" || !containsCandidate(candidates, req.Seed.ID)) {
		return Result{}, errors.New("currently playing track is not available in the synced cache")
	}
	if len(candidates) < limit {
		return Result{}, fmt.Errorf("need %d eligible cached tracks; found %d", limit, len(candidates))
	}
	byID := map[string]subsonic.Track{}
	for _, candidate := range candidates {
		byID[candidate.Track.ID] = candidate.Track
	}
	accepted := make([]string, 0, limit)
	acceptedSet := map[string]bool{}
	if req.Mode == ModeMood {
		accepted = append(accepted, req.Seed.ID)
		acceptedSet[req.Seed.ID] = true
		if onAccept != nil {
			if err := onAccept(req.Seed, 1, limit); err != nil {
				return Result{}, err
			}
		}
	}
	var attempts []Attempt
	var allRejected []string
	for attempt := 1; attempt <= maxAttempts && len(accepted) < limit; attempt++ {
		remaining := remainingCandidates(candidates, acceptedSet, req.Mode, req.Preferred)
		prompt := promptText(req, remaining, limit-len(accepted), attempt, accepted)
		allowed := candidateIDSet(remaining)
		parser := &streamedStringParser{}
		var raw strings.Builder
		var added, rejected []string
		full := false
		roundCtx, cancel := context.WithCancel(ctx)
		err := client.StreamComplete(roundCtx, []llm.Message{{Role: "system", Content: systemPrompt()}, {Role: "user", Content: prompt}}, 3000, func(delta string) error {
			raw.WriteString(delta)
			for _, id := range parser.Feed(delta) {
				id = strings.TrimSpace(id)
				if id == "" || acceptedSet[id] || !allowed[id] {
					if id != "" {
						rejected = append(rejected, id)
						allRejected = append(allRejected, id)
					}
					continue
				}
				acceptedSet[id] = true
				accepted = append(accepted, id)
				added = append(added, id)
				if onAccept != nil {
					if err := onAccept(byID[id], len(accepted), limit); err != nil {
						return err
					}
				}
				if len(accepted) == limit {
					full = true
					cancel()
					break
				}
			}
			return nil
		})
		cancel()
		attempts = append(attempts, Attempt{Prompt: prompt, Response: raw.String(), Accepted: added, Rejected: rejected})
		if err != nil && !(full && errors.Is(err, context.Canceled)) {
			return resultFrom(candidates, accepted, attempts, allRejected), err
		}
	}
	var fallback []string
	if len(accepted) < limit {
		for _, candidate := range remainingCandidates(candidates, acceptedSet, req.Mode, req.Preferred) {
			id := candidate.Track.ID
			if acceptedSet[id] {
				continue
			}
			acceptedSet[id] = true
			accepted = append(accepted, id)
			fallback = append(fallback, id)
			if onAccept != nil {
				if err := onAccept(candidate.Track, len(accepted), limit); err != nil {
					return resultFrom(candidates, accepted, attempts, allRejected), err
				}
			}
			if len(accepted) == limit {
				break
			}
		}
	}
	if len(accepted) != limit {
		return resultFrom(candidates, accepted, attempts, allRejected), fmt.Errorf("curator could only fill %d/%d unique cached tracks", len(accepted), limit)
	}
	result := resultFrom(candidates, accepted, attempts, allRejected)
	result.Fallback = fallback
	return result, nil
}

type streamedStringParser struct {
	inString, escaped bool
	value             strings.Builder
}

func (p *streamedStringParser) Feed(delta string) []string {
	var values []string
	for _, r := range delta {
		if !p.inString {
			if r == '"' {
				p.inString = true
				p.value.Reset()
			}
			continue
		}
		if p.escaped {
			p.value.WriteRune(r)
			p.escaped = false
			continue
		}
		if r == '\\' {
			p.escaped = true
			continue
		}
		if r == '"' {
			values = append(values, p.value.String())
			p.inString = false
			continue
		}
		p.value.WriteRune(r)
	}
	return values
}

func resultFrom(candidates []localstore.CachedTrack, accepted []string, attempts []Attempt, rejected []string) Result {
	tracksByID := map[string]subsonic.Track{}
	for _, candidate := range candidates {
		tracksByID[candidate.Track.ID] = candidate.Track
	}
	tracks := make([]subsonic.Track, 0, len(accepted))
	for _, id := range accepted {
		tracks = append(tracks, tracksByID[id])
	}
	result := Result{Tracks: tracks, IDs: accepted, Rejected: rejected, Attempts: attempts}
	if len(attempts) > 0 {
		result.Prompt = attempts[0].Prompt
		result.Raw = attempts[len(attempts)-1].Response
	}
	return result
}

func systemPrompt() string {
	return strings.TrimSpace(systemPromptText)
}

func promptText(req Request, candidates []localstore.CachedTrack, limit, attempt int, accepted []string) string {
	rows := make([]string, 0, min(len(candidates), maxCandidateTracks)+8)
	mode := req.Mode
	if mode == "" {
		mode = ModeAIMix
	}
	rows = append(rows, "MODE: "+string(mode))
	if attempt > 1 {
		rows = append(rows, fmt.Sprintf("REPAIR ROUND %d: return exactly %d replacement track IDs.", attempt, limit))
		rows = append(rows, "Previously accepted IDs (never repeat): "+strings.Join(accepted, ","))
	} else {
		rows = append(rows, fmt.Sprintf("Create a playlist by returning exactly %d track IDs.", limit))
	}
	if req.Seed.ID != "" {
		rows = append(rows, "Seed: "+trackLine(localstore.CachedTrack{Track: req.Seed}))
	}
	if mode == ModeAIMix {
		rows = append(rows, "Prioritize NEW albums, then use BACK-NINE catalog and deep cuts while protecting momentum.")
	} else {
		rows = append(rows, "Preserve the seed's mood, energy, texture, and momentum. Include compatible NEW material and BACK-NINE deep cuts.")
	}
	rows = append(rows, "Return only a JSON array shaped like [\"id1\",\"id2\"].")
	rows = append(rows, "Candidates:")
	for i, candidate := range candidates {
		if i >= maxCandidateTracks {
			break
		}
		rows = append(rows, trackLine(candidate))
	}
	return strings.Join(rows, "\n")
}

func trackLine(candidate localstore.CachedTrack) string {
	track := candidate.Track
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
	if candidate.New {
		parts = append(parts, "class=NEW")
		if candidate.NewestRank > 0 {
			parts[len(parts)-1] = fmt.Sprintf("class=NEW rank=%d", candidate.NewestRank)
		}
	} else {
		parts = append(parts, "class=BACK-NINE")
	}
	if candidate.Starred {
		parts = append(parts, "starred=true")
	}
	return strings.Join(parts, " | ")
}

func clean(value string) string {
	value = strings.ReplaceAll(value, "\n", " ")
	value = strings.ReplaceAll(value, "\r", " ")
	return strings.TrimSpace(value)
}

func candidateList(cached []localstore.CachedTrack, recent map[string]bool, seedID string, preferred map[string]bool) []localstore.CachedTrack {
	seen := map[string]bool{}
	out := make([]localstore.CachedTrack, 0, len(cached))
	for _, candidate := range cached {
		id := candidate.Track.ID
		if id == "" || seen[id] || (recent[id] && id != seedID) {
			continue
		}
		seen[id] = true
		out = append(out, candidate)
	}
	sort.SliceStable(out, func(i, j int) bool {
		iPreferred, jPreferred := preferred[out[i].Track.ID], preferred[out[j].Track.ID]
		if iPreferred != jPreferred {
			return iPreferred
		}
		if out[i].New != out[j].New {
			return out[i].New
		}
		if out[i].New && out[i].NewestRank != out[j].NewestRank {
			return out[i].NewestRank < out[j].NewestRank
		}
		if out[i].Starred != out[j].Starred {
			return out[i].Starred
		}
		return sortKey(out[i].Track) < sortKey(out[j].Track)
	})
	return out
}

func containsCandidate(candidates []localstore.CachedTrack, id string) bool {
	for _, candidate := range candidates {
		if candidate.Track.ID == id {
			return true
		}
	}
	return false
}

func remainingCandidates(candidates []localstore.CachedTrack, accepted map[string]bool, mode Mode, preferred map[string]bool) []localstore.CachedTrack {
	var preferredTracks, newest, backNine []localstore.CachedTrack
	for _, candidate := range candidates {
		if accepted[candidate.Track.ID] {
			continue
		}
		switch {
		case preferred[candidate.Track.ID]:
			preferredTracks = append(preferredTracks, candidate)
		case candidate.New:
			newest = append(newest, candidate)
		default:
			backNine = append(backNine, candidate)
		}
	}
	out := make([]localstore.CachedTrack, 0, maxCandidateTracks)
	seen := map[string]bool{}
	appendFrom := func(source []localstore.CachedTrack, limit int) {
		for _, candidate := range source {
			if len(out) == maxCandidateTracks || limit == 0 {
				return
			}
			if seen[candidate.Track.ID] {
				continue
			}
			seen[candidate.Track.ID] = true
			out = append(out, candidate)
			limit--
		}
	}
	if mode == ModeMood {
		appendFrom(preferredTracks, 100)
		appendFrom(newest, 70)
		appendFrom(backNine, 70)
	} else {
		appendFrom(newest, 140)
		appendFrom(backNine, 100)
	}
	// Sparse buckets donate unused capacity so the model still sees the largest
	// useful closed-world pool available.
	appendFrom(preferredTracks, maxCandidateTracks)
	appendFrom(newest, maxCandidateTracks)
	appendFrom(backNine, maxCandidateTracks)
	return out
}

func candidateIDSet(candidates []localstore.CachedTrack) map[string]bool {
	ids := make(map[string]bool, len(candidates))
	for _, candidate := range candidates {
		ids[candidate.Track.ID] = true
	}
	return ids
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
		"attempts":     result.Attempts,
		"fallback_ids": result.Fallback,
		"created_at":   time.Now().UTC().Format(time.RFC3339),
	}
}
