package d2player

import "testing"

func TestSpeechBubbleSkipGrace(t *testing.T) {
	tests := []struct {
		name          string
		left, age     float64
		active, skips bool
	}{
		{"no speech", 0, 0, false, false},
		{"just started: key swallowed, line stays", 5, 0.05, true, false},
		{"after 100 ms the key cuts", 5, 0.1, true, true},
		{"long running", 1, 4, true, true},
	}

	for _, tt := range tests {
		s := &SpeechBubble{left: tt.left, age: tt.age}
		if s.Active() != tt.active || s.Skippable() != tt.skips {
			t.Errorf("%s: active=%v skippable=%v", tt.name, s.Active(), s.Skippable())
		}
	}

	s := &SpeechBubble{left: 3}
	s.Advance(0.2)

	if !s.Skippable() {
		t.Error("Advance must age the line")
	}

	s.Clear()

	if s.Active() {
		t.Error("Clear must end the line")
	}
}
