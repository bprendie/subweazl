package security

import (
	"strings"
	"unicode"
)

const (
	maxNameLen        = 120
	maxDescriptionLen = 500
	maxGenreLen       = 160
	maxNowPlayingLen  = 240
	maxURLLen         = 2048
)

func SanitizeStationName(s string) string {
	return SanitizeForTerminal(s, maxNameLen)
}

func SanitizeDescription(s string) string {
	return SanitizeForTerminal(s, maxDescriptionLen)
}

func SanitizeGenre(s string) string {
	return SanitizeForTerminal(s, maxGenreLen)
}

func SanitizeNowPlaying(s string) string {
	return SanitizeForTerminal(s, maxNowPlayingLen)
}

func SanitizeURL(s string) string {
	return SanitizeForTerminal(s, maxURLLen)
}

func SanitizeForTerminal(s string, maxLen int) string {
	s = strings.ToValidUTF8(s, "")
	var b strings.Builder
	for _, r := range s {
		if unicode.IsControl(r) {
			continue
		}
		b.WriteRune(r)
	}
	return limitRunes(strings.TrimSpace(b.String()), maxLen)
}

func limitRunes(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	if len([]rune(s)) <= maxLen {
		return s
	}
	runes := []rune(s)
	return string(runes[:maxLen])
}
