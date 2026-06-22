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
