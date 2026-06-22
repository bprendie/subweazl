package localstore

import (
	"strings"
	"testing"
)

func TestSaveRecommendationRunEncryptsPayload(t *testing.T) {
	store := newMigratedStore(t)
	if err := store.CreateVault("pw"); err != nil {
		t.Fatalf("CreateVault: %v", err)
	}
	run, err := store.SaveRecommendationRun(RecommendationRun{
		Provider: "test-provider",
		Model:    "test-model",
		TrackIDs: []string{"a", "b"},
		Payload: map[string]any{
			"prompt":   "private listening summary",
			"response": `{"track_ids":["a","b"]}`,
		},
	})
	if err != nil {
		t.Fatalf("save run: %v", err)
	}
	if run.ID == "" || run.CreatedAt == "" {
		t.Fatalf("run metadata not populated: %#v", run)
	}
	var provider, model, payload string
	if err := store.RawDB().QueryRow(`select provider, model, payload from recommendation_runs where id = ?`, run.ID).Scan(&provider, &model, &payload); err != nil {
		t.Fatalf("query run: %v", err)
	}
	if provider != "test-provider" || model != "test-model" {
		t.Fatalf("provider/model = %q/%q", provider, model)
	}
	if strings.Contains(payload, "private listening summary") || strings.Contains(payload, "track_ids") {
		t.Fatalf("payload is plaintext: %q", payload)
	}
}

func TestSaveRecommendationRunRequiresProviderModelAndTracks(t *testing.T) {
	store := newMigratedStore(t)
	if err := store.CreateVault("pw"); err != nil {
		t.Fatalf("CreateVault: %v", err)
	}
	if _, err := store.SaveRecommendationRun(RecommendationRun{Model: "m", TrackIDs: []string{"a"}}); err == nil {
		t.Fatal("expected provider error")
	}
	if _, err := store.SaveRecommendationRun(RecommendationRun{Provider: "p", TrackIDs: []string{"a"}}); err == nil {
		t.Fatal("expected model error")
	}
	if _, err := store.SaveRecommendationRun(RecommendationRun{Provider: "p", Model: "m"}); err == nil {
		t.Fatal("expected tracks error")
	}
}
