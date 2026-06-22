package security

import "testing"

func TestSanitizeForTerminalRemovesControlCharacters(t *testing.T) {
	got := SanitizeForTerminal("ok\x1b[31m red\x00\nnext", 80)
	want := "ok[31m rednext"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestSanitizeForTerminalTruncatesByRune(t *testing.T) {
	got := SanitizeForTerminal("  åß∂ƒ  ", 3)
	want := "åß∂"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
