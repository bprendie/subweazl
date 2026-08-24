package llm

import (
	"context"
	"encoding/json"
	"errors"
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

func TestOmarchyStreamDeliversBufferedAgentOutput(t *testing.T) {
	bin := t.TempDir()
	writeExecutable(t, bin, "codex", "#!/bin/sh\nprintf '[\"weazl-track\"]'\n")
	t.Setenv("PATH", bin)
	client := New(config.LLMConfig{Provider: "omarchy", Model: "codex"})
	var got string
	err := client.StreamComplete(context.Background(), []Message{{Role: "system", Content: "You are DJ-Weazl."}}, 100, func(delta string) error {
		got += delta
		return nil
	})
	if err != nil || got != `["weazl-track"]` {
		t.Fatalf("got=%q err=%v", got, err)
	}
	want := errors.New("stop")
	err = client.StreamComplete(context.Background(), nil, 100, func(string) error { return want })
	if !errors.Is(err, want) {
		t.Fatalf("callback error = %v", err)
	}
}

func TestCompleteRequiresConfig(t *testing.T) {
	_, err := New(config.LLMConfig{}).Complete(context.Background(), nil, 0)
	if err == nil {
		t.Fatal("expected config error")
	}
}

func TestCompleteUsesOllamaChatShape(t *testing.T) {
	var gotPath string
	var gotStream bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		var req map[string]any
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatal(err)
		}
		gotStream, _ = req["stream"].(bool)
		_ = json.NewEncoder(w).Encode(map[string]any{"message": map[string]string{"content": `["a"]`}})
	}))
	defer server.Close()
	client := New(config.LLMConfig{Provider: "ollama", BaseURL: server.URL, Model: "mistral", ChatPath: "/api/chat"})
	got, err := client.Complete(context.Background(), []Message{{Role: "user", Content: "pick"}}, 20)
	if err != nil || got != `["a"]` || gotPath != "/api/chat" || gotStream {
		t.Fatalf("got=%q path=%q stream=%v err=%v", got, gotPath, gotStream, err)
	}
}

func TestStreamCompleteParsesVLLMSSE(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatal(err)
		}
		if stream, _ := req["stream"].(bool); !stream {
			t.Fatal("stream was not enabled")
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"[\\\"a\"}}]}\n\n"))
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"\\\",\\\"b\\\"]\"}}]}\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer server.Close()
	client := New(config.LLMConfig{Provider: "vllm", BaseURL: server.URL, Model: "m", ChatPath: "/v1/chat/completions"})
	var got string
	err := client.StreamComplete(context.Background(), []Message{{Role: "user", Content: "pick"}}, 100, func(delta string) error { got += delta; return nil })
	if err != nil || got != `["a","b"]` {
		t.Fatalf("got=%q err=%v", got, err)
	}
}

func TestFetchProviderModels(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/models":
			_ = json.NewEncoder(w).Encode(map[string]any{"data": []map[string]string{{"id": "vllm-model"}}})
		case "/api/tags":
			_ = json.NewEncoder(w).Encode(map[string]any{"models": []map[string]string{{"name": "ollama-model"}}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	for provider, want := range map[string]string{"vllm": "vllm-model", "ollama": "ollama-model"} {
		models, err := FetchProviderModels(context.Background(), provider, server.URL)
		if err != nil || len(models) != 1 || models[0] != want {
			t.Fatalf("provider=%s models=%v err=%v", provider, models, err)
		}
	}
}

func TestNormalizeProviderServerURL(t *testing.T) {
	if got := NormalizeServerURL("vllm", "https://host/v1/"); got != "https://host" {
		t.Fatalf("vllm url=%q", got)
	}
	if got := NormalizeServerURL("ollama", "http://host:11434/api/"); got != "http://host:11434" {
		t.Fatalf("ollama url=%q", got)
	}
}
