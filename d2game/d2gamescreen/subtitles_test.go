package d2gamescreen

import "testing"

func TestSubtitleLine(t *testing.T) {
	tests := []struct {
		name, speaker string
		msg           int
		text          string
		want          string
	}{
		{"plain", "Akara", 3, "Greetings, stranger.", "SUBTITLE Akara [message 3]: Greetings, stranger."},
		{"line breaks and runs of spaces collapse", "Kashya", 12, "Rogues need\n  your help.\n", "SUBTITLE Kashya [message 12]: Rogues need your help."},
		{"no text", "Warriv", 40, "", "SUBTITLE Warriv [message 40]: (no text)"},
		{"only blanks", "Warriv", 41, " \n ", "SUBTITLE Warriv [message 41]: (no text)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := subtitleLine(tt.speaker, tt.msg, tt.text); got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}
