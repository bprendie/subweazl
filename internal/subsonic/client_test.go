package subsonic

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"path"
	"testing"
)

func TestSimilarPrefersSimilarSongs(t *testing.T) {
	var methods []string
	client := New("https://example.test", "u", "p")
	client.http = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		methods = append(methods, path.Base(r.URL.Path))
		switch path.Base(r.URL.Path) {
		case "getSimilarSongs2.view":
			return jsonResponse(t, map[string]any{
				"similarSongs2": map[string]any{"song": []Track{
					{ID: "similar-1", Title: "Similar 1"},
					{ID: "similar-2", Title: "Similar 2"},
				}},
			}), nil
		case "search3.view":
			return jsonResponse(t, map[string]any{
				"searchResult3": map[string]any{"song": []Track{{ID: "artist-1", Title: "Artist 1"}}},
			}), nil
		default:
			return jsonResponse(t, map[string]any{}), nil
		}
	})}
	got, err := client.Similar(context.Background(), Track{ID: "seed", Title: "Seed", Artist: "Artist"}, 4)
	if err != nil {
		t.Fatalf("similar: %v", err)
	}
	ids := trackIDs(got)
	want := []string{"seed", "similar-1", "similar-2", "artist-1"}
	if !sameStrings(ids, want) {
		t.Fatalf("ids = %#v, want %#v", ids, want)
	}
	if methods[0] != "getSimilarSongs2.view" {
		t.Fatalf("first method = %q, want getSimilarSongs2.view", methods[0])
	}
}

func TestRandomSongsByYearQuery(t *testing.T) {
	client := New("https://example.test", "u", "p")
	client.http = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if got := r.URL.Query().Get("fromYear"); got != "1998" {
			t.Fatalf("fromYear = %q, want 1998", got)
		}
		if got := r.URL.Query().Get("toYear"); got != "2002" {
			t.Fatalf("toYear = %q, want 2002", got)
		}
		return jsonResponse(t, map[string]any{
			"randomSongs": map[string]any{"song": []Track{{ID: "year-1", Title: "Year 1"}}},
		}), nil
	})}
	got, err := client.RandomSongsByYear(context.Background(), 2000, 10)
	if err != nil {
		t.Fatalf("random by year: %v", err)
	}
	if len(got) != 1 || got[0].ID != "year-1" {
		t.Fatalf("tracks = %#v", got)
	}
}

func TestRandomAlbumsQuery(t *testing.T) {
	client := New("https://example.test", "u", "p")
	client.http = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if got := r.URL.Query().Get("type"); got != "random" {
			t.Fatalf("type = %q, want random", got)
		}
		writeAlbumListResponse := map[string]any{
			"albumList2": map[string]any{"album": []Album{{ID: "album-1", Name: "Album 1"}}},
		}
		return jsonResponse(t, writeAlbumListResponse), nil
	})}
	got, err := client.RandomAlbums(context.Background())
	if err != nil {
		t.Fatalf("random albums: %v", err)
	}
	if len(got) != 1 || got[0].ID != "album-1" {
		t.Fatalf("albums = %#v", got)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func jsonResponse(t *testing.T, payload map[string]any) *http.Response {
	t.Helper()
	body := map[string]any{"subsonic-response": map[string]any{"status": "ok"}}
	for key, value := range payload {
		body["subsonic-response"].(map[string]any)[key] = value
	}
	var b bytes.Buffer
	if err := json.NewEncoder(&b).Encode(body); err != nil {
		t.Fatalf("encode response: %v", err)
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       ioNopCloser{&b},
	}
}

type ioNopCloser struct{ *bytes.Buffer }

func (c ioNopCloser) Close() error { return nil }

func trackIDs(tracks []Track) []string {
	ids := make([]string, 0, len(tracks))
	for _, track := range tracks {
		ids = append(ids, track.ID)
	}
	return ids
}

func sameStrings(a, b []string) bool {
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
