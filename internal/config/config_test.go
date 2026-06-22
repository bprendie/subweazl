package config

import "testing"

func TestDefaultConfigIsEmpty(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)

	got, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if got.Server != "" || got.Username != "" || got.Password != "" {
		t.Fatalf("default config = %#v", got)
	}
	if got.LLMReady() {
		t.Fatal("default llm config is ready")
	}
}

func TestSaveLoadSubsonicConfig(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	cfg := Config{
		Server:   " https://example.test/ ",
		Username: " user ",
		Password: "pass",
		LLM: LLMConfig{
			Provider:      " local ",
			BaseURL:       " https://llm.example/ ",
			Model:         " model-a ",
			ChatPath:      "v1/chat/completions",
			ModelsPath:    " /v1/models ",
			ContextWindow: 8192,
			APIKey:        " key ",
		},
	}
	if err := Save(cfg); err != nil {
		t.Fatalf("save config: %v", err)
	}
	got, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if got.Server != "https://example.test" {
		t.Fatalf("server = %q", got.Server)
	}
	if got.Username != "user" {
		t.Fatalf("username = %q", got.Username)
	}
	if got.Password != "pass" {
		t.Fatalf("password = %q", got.Password)
	}
	if !got.LLMReady() || got.LLM.ChatPath != "/v1/chat/completions" || got.LLM.ModelsPath != "/v1/models" {
		t.Fatalf("llm config = %#v", got.LLM)
	}
}

func TestEnvOverridesConfig(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	t.Setenv("SUBWEAZL_SERVER", "https://env.example/")
	t.Setenv("SUBWEAZL_USER", "env-user")
	t.Setenv("SUBWEAZL_PASSWORD", "env-pass")
	t.Setenv("SUBWEAZL_LLM_PROVIDER", "env-provider")
	t.Setenv("SUBWEAZL_LLM_BASE_URL", "https://llm-env.example/")
	t.Setenv("SUBWEAZL_LLM_MODEL", "env-model")
	t.Setenv("SUBWEAZL_LLM_CHAT_PATH", "chat")
	t.Setenv("SUBWEAZL_LLM_CONTEXT_WINDOW", "4096")
	if err := Save(Config{Server: "https://file.example", Username: "file", Password: "file-pass"}); err != nil {
		t.Fatalf("save config: %v", err)
	}
	got, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if got.Server != "https://env.example" || got.Username != "env-user" || got.Password != "env-pass" {
		t.Fatalf("config = %#v", got)
	}
	if got.LLM.Provider != "env-provider" || got.LLM.BaseURL != "https://llm-env.example" || got.LLM.Model != "env-model" || got.LLM.ChatPath != "/chat" || got.LLM.ContextWindow != 4096 {
		t.Fatalf("llm config = %#v", got.LLM)
	}
}

func TestReadyRequiresSubsonicCredentials(t *testing.T) {
	if (Config{}).Ready() {
		t.Fatal("empty config is ready")
	}
	if !(Config{Server: "https://example.test", Username: "user", Password: "pass"}).Ready() {
		t.Fatal("complete config is not ready")
	}
}
