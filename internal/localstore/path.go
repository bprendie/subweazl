package localstore

import (
	"os"
	"path/filepath"
	"runtime"
)

const appName = "subweazl"

func Path() (string, error) {
	dir, err := DataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "vault.sqlite3"), nil
}

func DataDir() (string, error) {
	if dir := os.Getenv("SUBWEAZL_DATA_HOME"); dir != "" {
		return filepath.Join(dir, appName), nil
	}
	if dir := os.Getenv("XDG_DATA_HOME"); dir != "" {
		return filepath.Join(dir, appName), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", appName), nil
	case "windows":
		if appData := os.Getenv("APPDATA"); appData != "" {
			return filepath.Join(appData, appName), nil
		}
		return filepath.Join(home, "AppData", "Roaming", appName), nil
	default:
		return filepath.Join(home, ".local", "share", appName), nil
	}
}
