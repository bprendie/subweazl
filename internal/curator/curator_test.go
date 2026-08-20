package curator

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/bprendie/subweazl/internal/llm"
	"github.com/bprendie/subweazl/internal/localstore"
	"github.com/bprendie/subweazl/internal/subsonic"
)

type fakeClient struct {
	responses []string
	calls     int
}

type fakeStreamingClient struct {
	response string
	calls    int
}

func (f *fakeStreamingClient) StreamComplete(ctx context.Context, _ []llm.Message, _ int, onDelta func(string) error) error {
	f.calls++
	for start := 0; start < len(f.response); start += 17 {
		end := min(len(f.response), start+17)
		if err := onDelta(f.response[start:end]); err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
	}
	return nil
}

func (f *fakeClient) Complete(_ context.Context, messages []llm.Message, _ int) (string, error) {
	if len(messages) != 2 {
		return "", nil
	}
	response := f.responses[min(f.calls, len(f.responses)-1)]
	f.calls++
	return response, nil
}

func TestGenerateValidatesReturnedIDs(t *testing.T) {
	result, err := Generate(context.Background(), &fakeClient{responses: []string{`{"track_ids":["b","invented","a","b"]}`}}, Request{
		Candidates: []localstore.CachedTrack{
			{Track: subsonic.Track{ID: "a", Title: "A", Artist: "One"}},
			{Track: subsonic.Track{ID: "b", Title: "B", Artist: "Two"}},
		},
		Limit: 2,
	})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if len(result.Tracks) != 2 || result.Tracks[0].ID != "b" || result.Tracks[1].ID != "a" {
		t.Fatalf("tracks = %#v", result.Tracks)
	}
	if len(result.Rejected) != 1 || result.Rejected[0] != "invented" {
		t.Fatalf("rejected = %#v", result.Rejected)
	}
}

func TestGenerateRejectsAllInventedIDs(t *testing.T) {
	client := &fakeClient{responses: []string{`{"track_ids":["invented"]}`}}
	result, err := Generate(context.Background(), client, Request{
		Candidates: []localstore.CachedTrack{{Track: subsonic.Track{ID: "a", Title: "A"}}},
		Limit:      1,
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if client.calls != maxAttempts {
		t.Fatalf("calls = %d, want %d", client.calls, maxAttempts)
	}
	if len(result.IDs) != 1 || result.IDs[0] != "a" || len(result.Fallback) != 1 {
		t.Fatalf("fallback result = %#v", result)
	}
}

func TestGenerateRepairsHallucinationsWithoutRepeats(t *testing.T) {
	var candidates []localstore.CachedTrack
	for i := 0; i < 20; i++ {
		id := fmt.Sprintf("id-%02d", i)
		candidates = append(candidates, localstore.CachedTrack{Track: subsonic.Track{ID: id, Title: id}})
	}
	first := make([]string, 0, 20)
	for i := 0; i < 13; i++ {
		first = append(first, fmt.Sprintf("id-%02d", i))
	}
	for i := 0; i < 7; i++ {
		first = append(first, fmt.Sprintf("ghost-%02d", i))
	}
	second := make([]string, 0, 7)
	for i := 13; i < 20; i++ {
		second = append(second, fmt.Sprintf("id-%02d", i))
	}
	client := &fakeClient{responses: []string{jsonIDs(t, first), jsonIDs(t, second)}}
	result, err := Generate(context.Background(), client, Request{Candidates: candidates, Limit: 20})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if len(result.IDs) != 20 || client.calls != 2 {
		t.Fatalf("ids=%v calls=%d", result.IDs, client.calls)
	}
	seen := map[string]bool{}
	for _, id := range result.IDs {
		if seen[id] || strings.HasPrefix(id, "ghost-") {
			t.Fatalf("invalid final ids: %v", result.IDs)
		}
		seen[id] = true
	}
}

func jsonIDs(t *testing.T, ids []string) string {
	t.Helper()
	b, err := json.Marshal(ids)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestMoodKeepsSeedFirstAndRequestsRemainingTracks(t *testing.T) {
	seed := subsonic.Track{ID: "seed", Title: "Teenagers", Artist: "My Chemical Romance"}
	candidates := []localstore.CachedTrack{{Track: seed}}
	var ids []string
	for i := 0; i < 19; i++ {
		id := fmt.Sprintf("emo-%02d", i)
		ids = append(ids, id)
		candidates = append(candidates, localstore.CachedTrack{Track: subsonic.Track{ID: id, Title: id}})
	}
	client := &fakeClient{responses: []string{jsonIDs(t, ids)}}
	result, err := Generate(context.Background(), client, Request{
		Mode:       ModeMood,
		Seed:       seed,
		Candidates: candidates,
		Limit:      20,
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if len(result.IDs) != 20 || result.IDs[0] != "seed" {
		t.Fatalf("ids = %v", result.IDs)
	}
	if !strings.Contains(result.Attempts[0].Prompt, "exactly 19 track IDs") {
		t.Fatalf("prompt = %q", result.Attempts[0].Prompt)
	}
}

func TestAIMixCandidateWindowKeepsNewAndBackNine(t *testing.T) {
	var candidates []localstore.CachedTrack
	for i := 0; i < 160; i++ {
		candidates = append(candidates, localstore.CachedTrack{Track: subsonic.Track{ID: fmt.Sprintf("new-%03d", i)}, New: true, NewestRank: i/8 + 1})
		candidates = append(candidates, localstore.CachedTrack{Track: subsonic.Track{ID: fmt.Sprintf("back-%03d", i)}})
	}
	window := remainingCandidates(candidates, nil, ModeAIMix, nil)
	newCount := 0
	for _, candidate := range window {
		if candidate.New {
			newCount++
		}
	}
	if len(window) != 240 || newCount != 140 {
		t.Fatalf("window=%d new=%d", len(window), newCount)
	}
}

func TestGenerateStreamingStopsAfterRequiredValidIDs(t *testing.T) {
	var candidates []localstore.CachedTrack
	var emitted []string
	for i := 0; i < 120; i++ {
		id := fmt.Sprintf("opaque-navidrome-id-%03d", i)
		emitted = append(emitted, id)
		candidates = append(candidates, localstore.CachedTrack{Track: subsonic.Track{ID: id, Title: id}})
	}
	client := &fakeStreamingClient{response: jsonIDs(t, emitted)}
	var progress []string
	result, err := GenerateStreaming(context.Background(), client, Request{Candidates: candidates, Limit: 20}, func(track subsonic.Track, _, _ int) error {
		progress = append(progress, track.ID)
		return nil
	})
	if err != nil {
		t.Fatalf("GenerateStreaming: %v", err)
	}
	if len(result.IDs) != 20 || len(progress) != 20 {
		t.Fatalf("ids=%d progress=%d", len(result.IDs), len(progress))
	}
	if result.IDs[19] != emitted[19] {
		t.Fatalf("last id=%q want=%q", result.IDs[19], emitted[19])
	}
	if client.calls != 1 {
		t.Fatalf("calls=%d", client.calls)
	}
}

func TestStreamedStringParserWaitsForClosingQuote(t *testing.T) {
	p := &streamedStringParser{}
	if got := p.Feed(`["partial`); len(got) != 0 {
		t.Fatalf("premature values=%v", got)
	}
	got := p.Feed(`-id","complete"]`)
	if len(got) != 2 || got[0] != "partial-id" || got[1] != "complete" {
		t.Fatalf("values=%v", got)
	}
}

func TestCandidateListBlacklistsShortSilenceArtifacts(t *testing.T) {
	candidates := []localstore.CachedTrack{
		{Track: subsonic.Track{ID: "spacer", Title: "[silence]", Artist: "Bowling for Soup", Duration: 5}},
		{Track: subsonic.Track{ID: "real", Title: "Silence", Artist: "Portishead", Duration: 300}},
		{Track: subsonic.Track{ID: "song", Title: "A Real Song", Duration: 4}},
	}
	got := candidateList(candidates, nil, "", nil)
	if len(got) != 2 || got[0].Track.ID == "spacer" || got[1].Track.ID == "spacer" {
		t.Fatalf("candidate list = %#v", got)
	}
}

func TestShortSilenceArtifactCannotServeAsMoodSeed(t *testing.T) {
	seed := subsonic.Track{ID: "spacer", Title: "[silence]", Duration: 5}
	_, err := Generate(context.Background(), &fakeClient{responses: []string{`[]`}}, Request{
		Mode: ModeMood, Seed: seed, Limit: 1,
		Candidates: []localstore.CachedTrack{{Track: seed}},
	})
	if err == nil || !strings.Contains(err.Error(), "not available") {
		t.Fatalf("err=%v", err)
	}
}
