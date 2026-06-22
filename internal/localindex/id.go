package localindex

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"
)

func FolderID(path string) string {
	return prefixedID("folder", filepath.Clean(path))
}

func TrackID(path string, size int64, modifiedUnix int64) string {
	clean := filepath.Clean(path)
	return prefixedID("track", fmt.Sprintf("%s|%d|%d", clean, size, modifiedUnix))
}

func prefixedID(prefix string, value string) string {
	sum := sha256.Sum256([]byte(value))
	return prefix + "_" + hex.EncodeToString(sum[:10])
}
