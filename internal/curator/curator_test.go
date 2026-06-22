package curator

import (
	"context"
	"strings"
	"testing"

	"github.com/bprendie/subweazl/internal/llm"
	"github.com/bprendie/subweazl/internal/localstore"
	"github.com/bprendie/subweazl/internal/subsonic"
)

type fakeClient struct{ response string }

func (f fakeClient) Complete(_ context.Context, messages []llm.Message, _ int) (string, error) {
	if len(messages) != 2 || !strings.Contains(messages[1].Content, "id=a") {
		return "", nil
	}
	return f.response, nil
}

func TestGenerateValidatesReturnedIDs(t *testing.T) {
	result, err := Generate(context.Background(), fakeClient{response: `{"track_ids":["b","invented","a","b"]}`}, Request{
		Candidates: []localstore.CachedTrack{
			{Track: subsonic.Track{ID: "a", Title: "A", Artist: "One"}},
			{Track: subsonic.Track{ID: "b", Title: "B", Artist: "Two"}},
		},
		Limit: 10,
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
	_, err := Generate(context.Background(), fakeClient{response: `{"track_ids":["invented"]}`}, Request{
		Candidates: []localstore.CachedTrack{{Track: subsonic.Track{ID: "a", Title: "A"}}},
	})
	if err == nil {
		t.Fatal("expected error")
	}
}
