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

func TestPromptMakesUserRequestPrimaryConstraint(t *testing.T) {
	prompt := promptText(Request{
		Mode:       ModeAIMix,
		UserPrompt: "synthwave tracks for focus",
		Candidates: []localstore.CachedTrack{{Track: subsonic.Track{ID: "a", Title: "Signal", Artist: "One"}}},
		Limit:      1,
	}, []localstore.CachedTrack{{Track: subsonic.Track{ID: "a", Title: "Signal", Artist: "One"}}}, 1, 1, nil)
	if !strings.Contains(prompt, "USER REQUEST:\nsynthwave tracks for focus") || !strings.Contains(prompt, "user request governs eligibility") {
		t.Fatalf("prompt missing user constraint:\n%s", prompt)
	}
}

func TestAIMixUsesDedicatedRequestFirstSystemPrompt(t *testing.T) {
	aiPrompt := systemPrompt(ModeAIMix)
	moodPrompt := systemPrompt(ModeMood)
	if aiPrompt == moodPrompt {
		t.Fatal("AI Mix and Mood unexpectedly share a system prompt")
	}
	for _, want := range []string{"USER INTENT IS LAW", "NEW IS ONLY A TIE-BREAKER", "Do not copy candidate order"} {
		if !strings.Contains(aiPrompt, want) {
			t.Fatalf("AI Mix system prompt missing %q", want)
		}
	}
}

func TestPromptedMixDoesNotInstructModelToPrioritizeNew(t *testing.T) {
	prompt := promptText(Request{Mode: ModeAIMix, UserPrompt: "synthwave like Information Society"}, nil, 40, 1, nil)
	if strings.Contains(prompt, "Prioritize NEW") || !strings.Contains(prompt, "NEW is only a tie-breaker") {
		t.Fatalf("prompt has wrong priority:\n%s", prompt)
	}
}

func TestAIMixPromptDistinguishesAuthoritativeSeedFromLaunchTracks(t *testing.T) {
	seed := subsonic.Track{ID: "seed", Title: "Supersonic", Artist: "Oasis"}
	launch := subsonic.Track{ID: "launch", Title: "Bittersweet Symphony", Artist: "The Verve"}
	prompt := promptText(Request{Mode: ModeAIMix, UserPrompt: "rock like Oasis", Anchors: []subsonic.Track{seed}, Initial: []subsonic.Track{seed, launch}}, nil, 37, 1, []string{"seed", "launch"})
	if !strings.Contains(prompt, "AUTHORITATIVE SEED") || !strings.Contains(prompt, "PROVISIONAL LAUNCH TRACKS") || !strings.Contains(prompt, "do not use them to broaden") {
		t.Fatalf("prompt lacks anchor roles:\n%s", prompt)
	}
}

func TestAIMixRepairPromptCarriesRemainingClassRequirements(t *testing.T) {
	var candidates []localstore.CachedTrack
	var accepted []string
	for i := 0; i < 24; i++ {
		id := fmt.Sprintf("new-%d", i)
		candidates = append(candidates, localstore.CachedTrack{Track: subsonic.Track{ID: id}, New: true})
		accepted = append(accepted, id)
	}
	for i := 0; i < 10; i++ {
		id := fmt.Sprintf("back-%d", i)
		candidates = append(candidates, localstore.CachedTrack{Track: subsonic.Track{ID: id}})
		accepted = append(accepted, id)
	}
	prompt := promptText(Request{Mode: ModeAIMix, Candidates: candidates, Limit: 40}, nil, 6, 2, accepted)
	if !strings.Contains(prompt, "at most 0 additional NEW") || !strings.Contains(prompt, "at least 6 additional BACK-NINE") {
		t.Fatalf("prompt lacks remaining class requirements:\n%s", prompt)
	}
}

func TestAIMixCompletionRequiresEightDistinctArtists(t *testing.T) {
	byID := map[string]localstore.CachedTrack{}
	var accepted []string
	for i := 0; i < 8; i++ {
		id := fmt.Sprintf("id-%d", i)
		accepted = append(accepted, id)
		byID[id] = localstore.CachedTrack{Track: subsonic.Track{ID: id, Artist: fmt.Sprintf("Artist %d", i%7)}}
	}
	if err := validateMixCompletion(ModeAIMix, accepted, byID); err == nil {
		t.Fatal("accepted mix with fewer than eight distinct artists")
	}
	byID["id-7"] = localstore.CachedTrack{Track: subsonic.Track{ID: "id-7", Artist: "Artist 7"}}
	if err := validateMixCompletion(ModeAIMix, accepted, byID); err != nil {
		t.Fatalf("rejected eight artists: %v", err)
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
	if len(window) != 240 || newCount != 144 {
		t.Fatalf("window=%d new=%d", len(window), newCount)
	}
	for i := 0; i+3 < len(window); i++ {
		if window[i].New && window[i+1].New && window[i+2].New && window[i+3].New {
			t.Fatalf("NEW candidates remain front-loaded at %d", i)
		}
	}
}

func TestAIMixSelectionEnforcesClassArtistAlbumAndAdjacencyCaps(t *testing.T) {
	byID := map[string]localstore.CachedTrack{}
	accepted := []string{"a1", "a2", "b1"}
	byID["a1"] = localstore.CachedTrack{Track: subsonic.Track{ID: "a1", Artist: "Artist A", Album: "Album A"}, New: true}
	byID["a2"] = localstore.CachedTrack{Track: subsonic.Track{ID: "a2", Artist: "Artist A", Album: "Album B"}, New: true}
	byID["b1"] = localstore.CachedTrack{Track: subsonic.Track{ID: "b1", Artist: "Artist B", Album: "Album C"}}
	byID["a0"] = localstore.CachedTrack{Track: subsonic.Track{ID: "a0", Artist: "Artist A", Album: "Album Z"}}
	artistCap := localstore.CachedTrack{Track: subsonic.Track{ID: "a3", Artist: "Artist A", Album: "Album D"}}
	byID["a3"] = artistCap
	if selectionAllowed(artistCap, append(accepted, "a0"), byID, ModeAIMix, 40) {
		t.Fatal("allowed fourth track by artist")
	}
	adjacent := localstore.CachedTrack{Track: subsonic.Track{ID: "b2", Artist: "Artist B", Album: "Album D"}}
	if selectionAllowed(adjacent, accepted, byID, ModeAIMix, 40) {
		t.Fatal("allowed adjacent same artist")
	}
	albumCap := localstore.CachedTrack{Track: subsonic.Track{ID: "a4", Artist: "Artist C", Album: "Album A"}}
	byID["a0"] = localstore.CachedTrack{Track: subsonic.Track{ID: "a0", Artist: "Artist D", Album: "Album A"}}
	if selectionAllowed(albumCap, append(accepted, "a0"), byID, ModeAIMix, 40) {
		t.Fatal("allowed third track from album")
	}
	var newAccepted []string
	for i := 0; i < 24; i++ {
		id := fmt.Sprintf("new-cap-%d", i)
		byID[id] = localstore.CachedTrack{Track: subsonic.Track{ID: id, Artist: fmt.Sprintf("Artist %d", i), Album: fmt.Sprintf("Album %d", i)}, New: true}
		newAccepted = append(newAccepted, id)
	}
	tooNew := localstore.CachedTrack{Track: subsonic.Track{ID: "too-new", Artist: "Different", Album: "Different"}, New: true}
	if selectionAllowed(tooNew, newAccepted, byID, ModeAIMix, 40) {
		t.Fatal("allowed more than 24 NEW tracks")
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

func TestGenerateStreamingKeepsPublishedAnchorsAndRequestsOnlyBackfill(t *testing.T) {
	anchors := []subsonic.Track{
		{ID: "anchor-a", Artist: "Oasis", Album: "Definitely Maybe"},
		{ID: "anchor-b", Artist: "Blur", Album: "Parklife"},
		{ID: "anchor-c", Artist: "The Verve", Album: "Urban Hymns"},
	}
	candidates := make([]localstore.CachedTrack, 0, 40)
	for _, anchor := range anchors {
		candidates = append(candidates, localstore.CachedTrack{Track: anchor})
	}
	var backfill []string
	for i := 0; i < 37; i++ {
		id := fmt.Sprintf("fill-%02d", i)
		backfill = append(backfill, id)
		candidates = append(candidates, localstore.CachedTrack{Track: subsonic.Track{ID: id, Artist: fmt.Sprintf("Artist %02d", i), Album: fmt.Sprintf("Album %02d", i)}})
	}
	client := &fakeStreamingClient{response: jsonIDs(t, backfill)}
	var progress []string
	result, err := GenerateStreaming(context.Background(), client, Request{Mode: ModeAIMix, Candidates: candidates, Initial: anchors, Limit: 40}, func(track subsonic.Track, _, _ int) error {
		progress = append(progress, track.ID)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.IDs) != 40 || len(progress) != 37 || result.IDs[0] != "anchor-a" || progress[0] != "fill-00" {
		t.Fatalf("result=%v progress=%v", result.IDs, progress)
	}
	if !strings.Contains(result.Attempts[0].Prompt, "exactly 37 track IDs") {
		t.Fatalf("prompt=%s", result.Attempts[0].Prompt)
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
