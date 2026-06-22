package localindex

import "testing"

func TestParseFFProbeMetadata(t *testing.T) {
	raw := []byte(`{
		"streams": [
			{"codec_type": "video", "codec_name": "png"},
			{"codec_type": "audio", "codec_name": "flac"}
		],
		"format": {
			"format_name": "flac",
			"duration": "245.125",
			"tags": {
				"TITLE": "Signal",
				"Artist": "The Weazls",
				"album": "Bare Metal",
				"album_artist": "The Weazls",
				"track": "3/11",
				"disc": "1/2",
				"date": "1999-04-02",
				"genre": "Noise"
			}
		}
	}`)
	meta, err := ParseFFProbe(raw)
	if err != nil {
		t.Fatalf("ParseFFProbe: %v", err)
	}
	if meta.Title != "Signal" || meta.Artist != "The Weazls" || meta.Album != "Bare Metal" {
		t.Fatalf("parsed basic tags = %#v", meta)
	}
	if meta.TrackNumber != 3 || meta.DiscNumber != 1 || meta.Year != 1999 {
		t.Fatalf("parsed numeric tags = %#v", meta)
	}
	if meta.Codec != "flac" || meta.Container != "flac" || meta.DurationSeconds != 245.125 {
		t.Fatalf("parsed format fields = %#v", meta)
	}
}

func TestIDsAreDeterministic(t *testing.T) {
	path := "/music/album/song.flac"
	if FolderID("/music/album") != FolderID("/music/album/") {
		t.Fatal("FolderID should normalize equivalent paths")
	}
	id := TrackID(path, 100, 200)
	if id != TrackID(path, 100, 200) {
		t.Fatal("TrackID is not deterministic")
	}
	if id == TrackID(path, 101, 200) {
		t.Fatal("TrackID should change when file size changes")
	}
}
