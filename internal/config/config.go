package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Config struct {
	Server   string    `json:"server"`
	Username string    `json:"username"`
	Password string    `json:"password"`
	LLM      LLMConfig `json:"llm,omitempty"`
}

type LLMConfig struct {
	Provider      string `json:"provider,omitempty"`
	BaseURL       string `json:"base_url,omitempty"`
	Model         string `json:"model,omitempty"`
	ChatPath      string `json:"chat_path,omitempty"`
	ModelsPath    string `json:"models_path,omitempty"`
	ContextWindow int    `json:"context_window,omitempty"`
	APIKey        string `json:"api_key,omitempty"`
}

func Load() (Config, error) {
	cfg := envConfig()
	path, err := Path()
	if err != nil {
		return cfg, err
	}
	if err := ensureConfigDir(filepath.Dir(path)); err != nil {
		return cfg, err
	}
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return cfg, Save(cfg)
	}
	if err != nil {
		return cfg, err
	}
	if err := json.Unmarshal(b, &cfg); err != nil {
		return cfg, err
	}
	cfg.applyEnv()
	cfg.normalize()
	return cfg, nil
}

func Save(cfg Config) error {
	path, err := Path()
	if err != nil {
		return err
	}
	if err := ensureConfigDir(filepath.Dir(path)); err != nil {
		return err
	}
	cfg.normalize()
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(b, '\n'), 0o600)
}

func Path() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "subweazl", "config.json"), nil
}

func (c Config) Ready() bool {
	return c.Server != "" && c.Username != "" && c.Password != ""
}

func (c Config) LLMReady() bool {
	return c.LLM.Provider != "" && c.LLM.BaseURL != "" && c.LLM.Model != "" && c.LLM.ChatPath != ""
}

func (c *Config) normalize() {
	c.Server = strings.TrimRight(strings.TrimSpace(c.Server), "/")
	c.Username = strings.TrimSpace(c.Username)
	c.LLM.normalize()
}

func (l *LLMConfig) NormalizeForClient() {
	l.normalize()
}

func (l *LLMConfig) normalize() {
	l.Provider = strings.TrimSpace(l.Provider)
	l.BaseURL = strings.TrimRight(strings.TrimSpace(l.BaseURL), "/")
	l.Model = strings.TrimSpace(l.Model)
	l.ChatPath = cleanPath(l.ChatPath)
	l.ModelsPath = cleanPath(l.ModelsPath)
	l.APIKey = strings.TrimSpace(l.APIKey)
}

func cleanPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return path
}

func (c *Config) applyEnv() {
	if v := os.Getenv("SUBWEAZL_SERVER"); v != "" {
		c.Server = v
	}
	if v := os.Getenv("SUBWEAZL_USER"); v != "" {
		c.Username = v
	}
	if v := os.Getenv("SUBWEAZL_PASSWORD"); v != "" {
		c.Password = v
	}
	c.applyLLMEnv()
}

func (c *Config) applyLLMEnv() {
	if v := os.Getenv("SUBWEAZL_LLM_PROVIDER"); v != "" {
		c.LLM.Provider = v
	}
	if v := os.Getenv("SUBWEAZL_LLM_BASE_URL"); v != "" {
		c.LLM.BaseURL = v
	}
	if v := os.Getenv("SUBWEAZL_LLM_MODEL"); v != "" {
		c.LLM.Model = v
	}
	if v := os.Getenv("SUBWEAZL_LLM_CHAT_PATH"); v != "" {
		c.LLM.ChatPath = v
	}
	if v := os.Getenv("SUBWEAZL_LLM_MODELS_PATH"); v != "" {
		c.LLM.ModelsPath = v
	}
	if v := os.Getenv("SUBWEAZL_LLM_API_KEY"); v != "" {
		c.LLM.APIKey = v
	}
	if v := os.Getenv("SUBWEAZL_LLM_CONTEXT_WINDOW"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			c.LLM.ContextWindow = n
		}
	}
}

func envConfig() Config {
	cfg := Config{
		Username: os.Getenv("SUBWEAZL_USER"),
		Password: os.Getenv("SUBWEAZL_PASSWORD"),
	}
	cfg.applyEnv()
	cfg.normalize()
	return cfg
}

func ensureConfigDir(dir string) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	return os.Chmod(dir, 0o700)
}
