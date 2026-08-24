package llm

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bprendie/subweazl/internal/config"
)

func TestResolveConfigUsesCurrentOmarchyAgent(t *testing.T) {
	bin := t.TempDir()
	writeExecutable(t, bin, "omarchy", "#!/bin/sh\nprintf 'codex\\n'\n")
	writeExecutable(t, bin, "codex", "#!/bin/sh\nexit 0\n")
	t.Setenv("PATH", bin)
	got, err := ResolveConfig(context.Background(), config.LLMConfig{Provider: "omarchy"})
	if err != nil || got.Model != "codex" {
		t.Fatalf("config = %#v, err = %v", got, err)
	}
}

func TestResolveConfigExplainsMissingDefault(t *testing.T) {
	bin := t.TempDir()
	writeExecutable(t, bin, "omarchy", "#!/bin/sh\nexit 0\n")
	t.Setenv("PATH", bin)
	_, err := ResolveConfig(context.Background(), config.LLMConfig{Provider: "omarchy"})
	if err == nil || !strings.Contains(err.Error(), "omarchy default agent") {
		t.Fatalf("error = %v", err)
	}
}

func TestOmarchyCompletionUsesIsolatedDirectory(t *testing.T) {
	bin := t.TempDir()
	marker := filepath.Join(t.TempDir(), "cwd")
	writeExecutable(t, bin, "codex", "#!/bin/sh\npwd > \"$SUBWEAZL_TEST_CWD\"\nprintf '{\"track_ids\":[\"a\"]}'\n")
	t.Setenv("PATH", bin)
	t.Setenv("SUBWEAZL_TEST_CWD", marker)
	output, err := completeWithAgent(context.Background(), "codex", []Message{{Role: "user", Content: "choose"}}, 100)
	if err != nil || output != `{"track_ids":["a"]}` {
		t.Fatalf("output = %q, err = %v", output, err)
	}
	cwd, err := os.ReadFile(marker)
	if err != nil || !strings.Contains(string(cwd), "subweazl-agent-") {
		t.Fatalf("cwd = %q, err = %v", cwd, err)
	}
}

func TestAgentPromptForbidsToolsAndPreservesWeazlMessages(t *testing.T) {
	prompt := agentPrompt([]Message{{Role: "system", Content: "You are DJ-Weazl."}, {Role: "user", Content: "request"}}, 250)
	for _, required := range []string{"Do not use tools", "<SYSTEM>", "DJ-Weazl", "<USER>", "request", "250 tokens"} {
		if !strings.Contains(prompt, required) {
			t.Fatalf("prompt missing %q: %s", required, prompt)
		}
	}
}

func TestAgentCommandSupportsCodexAndRejectsUnknownDefaults(t *testing.T) {
	name, args, err := agentCommand("codex", "prompt")
	if err != nil || name != "codex" || !strings.Contains(strings.Join(args, " "), "--sandbox read-only") {
		t.Fatalf("name=%q args=%v err=%v", name, args, err)
	}
	if _, _, err := agentCommand("unknown", "prompt"); err == nil {
		t.Fatal("expected unsupported-agent error")
	}
}

func writeExecutable(t *testing.T, dir, name, contents string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(contents), 0o755); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}
