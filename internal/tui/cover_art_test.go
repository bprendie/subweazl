package tui

import (
	"testing"

	"github.com/bprendie/subweazl/internal/subsonic"
)

func TestCoverArtIDFallback(t *testing.T) {
	tests := []struct {
		name  string
		track subsonic.Track
		want  string
	}{
		{name: "cover art", track: subsonic.Track{ID: "song", AlbumID: "album", CoverID: "cover"}, want: "cover"},
		{name: "album id", track: subsonic.Track{ID: "song", AlbumID: "album"}, want: "album"},
		{name: "song id", track: subsonic.Track{ID: "song"}, want: "song"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := coverArtID(tt.track); got != tt.want {
				t.Fatalf("coverArtID() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDecodeCoverArtRejectsAPIErrorPayloads(t *testing.T) {
	for _, data := range [][]byte{
		[]byte(`{"subsonic-response":{"status":"failed"}}`),
		[]byte(`<subsonic-response status="failed"></subsonic-response>`),
	} {
		if _, err := decodeCoverArt(data); err == nil || err.Error() != "cover art unavailable" {
			t.Fatalf("decodeCoverArt err = %v, want cover art unavailable", err)
		}
	}
}

func TestDecodeCoverArtRejectsUnknownFormatCleanly(t *testing.T) {
	if _, err := decodeCoverArt([]byte("not an image")); err == nil || err.Error() != "unsupported cover art format" {
		t.Fatalf("decodeCoverArt err = %v, want unsupported cover art format", err)
	}
}
