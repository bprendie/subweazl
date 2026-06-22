package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveLoadLocalMusicFolders(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	music := filepath.Join(t.TempDir(), "music")
	if err := os.Mkdir(music, 0o700); err != nil {
		t.Fatalf("mkdir music: %v", err)
	}
	cfg := Config{
		Server:            "https://example.test/",
		Username:          "user",
		Password:          "pass",
		LocalMusicFolders: []string{" " + music + " ", music, ""},
	}
	if err := Save(cfg); err != nil {
		t.Fatalf("save config: %v", err)
	}
	got, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if len(got.LocalMusicFolders) != 1 || got.LocalMusicFolders[0] != music {
		t.Fatalf("folders = %#v, want %q", got.LocalMusicFolders, music)
	}
}

func TestValidateLocalMusicFolders(t *testing.T) {
	music := t.TempDir()
	cfg := Config{LocalMusicFolders: []string{music}}
	if err := cfg.ValidateLocalMusicFolders(); err != nil {
		t.Fatalf("valid folder rejected: %v", err)
	}
	cfg.LocalMusicFolders = []string{filepath.Join(music, "missing")}
	if err := cfg.ValidateLocalMusicFolders(); err == nil {
		t.Fatal("missing folder accepted")
	}
}

func TestNormalizeFolderExpandsHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	got := normalizeFolders([]string{"~/Music"})
	want := filepath.Join(home, "Music")
	if len(got) != 1 || got[0] != want {
		t.Fatalf("folders = %#v, want %q", got, want)
	}
}
