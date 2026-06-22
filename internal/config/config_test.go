package config

import "testing"

func TestSaveLoadSubsonicConfig(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	cfg := Config{
		Server:   " https://example.test/ ",
		Username: " user ",
		Password: "pass",
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
}

func TestEnvOverridesConfig(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	t.Setenv("SUBWEAZL_SERVER", "https://env.example/")
	t.Setenv("SUBWEAZL_USER", "env-user")
	t.Setenv("SUBWEAZL_PASSWORD", "env-pass")
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
}

func TestReadyRequiresSubsonicCredentials(t *testing.T) {
	if (Config{}).Ready() {
		t.Fatal("empty config is ready")
	}
	if !(Config{Server: "https://example.test", Username: "user", Password: "pass"}).Ready() {
		t.Fatal("complete config is not ready")
	}
}
