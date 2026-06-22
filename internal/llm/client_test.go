package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bprendie/subweazl/internal/config"
)

func TestCompleteUsesConfiguredPathAndModel(t *testing.T) {
	var gotPath string
	var gotModel string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		var req map[string]any
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		gotModel, _ = req["model"].(string)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{{"message": map[string]string{"content": `{"track_ids":["a"]}`}}},
		})
	}))
	defer server.Close()

	client := New(config.LLMConfig{Provider: "test", BaseURL: server.URL, Model: "model-a", ChatPath: "chat"})
	got, err := client.Complete(context.Background(), []Message{{Role: "user", Content: "pick"}}, 20)
	if err != nil {
		t.Fatalf("complete: %v", err)
	}
	if got != `{"track_ids":["a"]}` || gotPath != "/chat" || gotModel != "model-a" {
		t.Fatalf("got content=%q path=%q model=%q", got, gotPath, gotModel)
	}
}

func TestCompleteRequiresConfig(t *testing.T) {
	_, err := New(config.LLMConfig{}).Complete(context.Background(), nil, 0)
	if err == nil {
		t.Fatal("expected config error")
	}
}
