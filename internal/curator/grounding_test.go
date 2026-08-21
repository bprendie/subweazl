package curator

import (
	"context"
	"testing"

	"github.com/bprendie/subweazl/internal/localstore"
	"github.com/bprendie/subweazl/internal/subsonic"
)

func TestInterpretIntentPreservesNamedArtistAndSearchTerms(t *testing.T) {
	client := &fakeClient{responses: []string{`{"artists":["New Order"],"tracks":[],"genres":["new wave","post-punk"],"era":"1980s","mood":["melancholic"],"texture":["synth"],"purpose":"focus","search_terms":["synth-pop"]}`}}
	intent, err := InterpretIntent(context.Background(), client, "new wave like New Order")
	if err != nil {
		t.Fatal(err)
	}
	if len(intent.SearchTerms) < 4 || intent.SearchTerms[0] != "New Order" {
		t.Fatalf("intent=%#v", intent)
	}
}

func TestInterpretIntentAcceptsArrayEraAndPurpose(t *testing.T) {
	client := &fakeClient{responses: []string{`{"artists":["Oasis"],"tracks":[],"genres":["rock"],"era":["1990s","Britpop"],"mood":[],"texture":[],"purpose":["driving"],"search_terms":["Oasis"]}`}}
	intent, err := InterpretIntent(context.Background(), client, "rock like Oasis")
	if err != nil {
		t.Fatal(err)
	}
	if string(intent.Era) != "1990s, Britpop" || string(intent.Purpose) != "driving" {
		t.Fatalf("intent=%#v", intent)
	}
}

func TestResolvePrimarySeedPrefersExactOasisArtist(t *testing.T) {
	candidates := []localstore.CachedTrack{
		{Track: subsonic.Track{ID: "cover", Title: "Wonderwall", Artist: "A Cover Band", Album: "Britpop Covers", Genre: "Rock"}},
		{Track: subsonic.Track{ID: "oasis", Title: "Supersonic", Artist: "Oasis", Album: "Definitely Maybe", Genre: "Rock"}},
		{Track: subsonic.Track{ID: "other", Title: "Rock Music", Artist: "Pixies", Genre: "Alternative"}},
	}
	seed, ok := ResolvePrimarySeed("rock like Oasis", candidates)
	if !ok || seed.ID != "oasis" {
		t.Fatalf("seed=%#v ok=%v", seed, ok)
	}
}

func TestSelectDeterministicAnchorsUsesDistinctArtistsAndAlbums(t *testing.T) {
	seed := subsonic.Track{ID: "oasis", Artist: "Oasis", Album: "Definitely Maybe"}
	neighbors := []subsonic.Track{
		{ID: "same-artist", Artist: "Oasis", Album: "Morning Glory"},
		{ID: "blur", Artist: "Blur", Album: "Parklife"},
		{ID: "blur-two", Artist: "Blur", Album: "13"},
		{ID: "verve", Artist: "The Verve", Album: "Urban Hymns"},
	}
	anchors, err := SelectDeterministicAnchors(seed, neighbors, map[string]bool{"oasis": true, "same-artist": true, "blur": true, "blur-two": true, "verve": true})
	if err != nil {
		t.Fatal(err)
	}
	if anchors[0].ID != "oasis" || anchors[1].ID != "blur" || anchors[2].ID != "verve" {
		t.Fatalf("anchors=%v", anchors)
	}
}

func TestSelectAnchorsIsClosedWorldAndRejectsRecordingVariants(t *testing.T) {
	candidates := []subsonic.Track{
		{ID: "new-order", Title: "Bizarre Love Triangle", Artist: "New Order", Album: "Brotherhood"},
		{ID: "depeche", Title: "Strangelove", Artist: "Depeche Mode", Album: "Music for the Masses"},
		{ID: "cure", Title: "A Forest", Artist: "The Cure", Album: "Seventeen Seconds"},
		{ID: "live", Title: "Blue Monday (Live)", Artist: "New Order", Album: "Live"},
	}
	client := &fakeClient{responses: []string{`{"track_ids":["ghost","live","new-order","depeche","cure"]}`}}
	anchors, err := SelectAnchors(context.Background(), client, "new wave like New Order", Intent{}, candidates)
	if err != nil {
		t.Fatal(err)
	}
	if len(anchors) != 3 || anchors[0].ID != "new-order" || anchors[2].ID != "cure" {
		t.Fatalf("anchors=%v", anchors)
	}
}

func TestMergeSimilarityCrateRanksMultiAnchorOverlap(t *testing.T) {
	anchors := []subsonic.Track{{ID: "a"}, {ID: "b"}, {ID: "c"}}
	shared := subsonic.Track{ID: "shared", Title: "Shared"}
	crate := MergeSimilarityCrate(anchors, [][]subsonic.Track{
		{shared, {ID: "one"}},
		{shared, {ID: "two"}},
		{{ID: "three"}},
	})
	if len(crate) != 7 || crate[3].ID != "shared" {
		t.Fatalf("crate=%v", crate)
	}
}
