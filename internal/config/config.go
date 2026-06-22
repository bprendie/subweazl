package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	Server   string `json:"server"`
	Username string `json:"username"`
	Password string `json:"password"`
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

func (c *Config) normalize() {
	c.Server = strings.TrimRight(strings.TrimSpace(c.Server), "/")
	c.Username = strings.TrimSpace(c.Username)
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
}

func envConfig() Config {
	cfg := Config{
		Server:   "https://weazltunes.prendie.io",
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
