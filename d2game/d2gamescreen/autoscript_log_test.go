package d2gamescreen

import (
	"strings"
	"testing"
)

func TestLogCaptureSince(t *testing.T) {
	c := &logCapture{}
	_, _ = c.Write([]byte("first line\nAUTOSCRIPT step 1: until:x\n"))
	mark := c.mark()
	_, _ = c.Write([]byte("AUTOSCRIPT step 2: say:x\nsecond line\n"))

	tests := []struct {
		substr string
		want   bool
	}{
		{"second", true},
		{"first", false},  // written before the mark
		{"step 2", false}, // the runner's own lines never count
		{"missing", false},
	}

	for _, tc := range tests {
		if got := c.containsSince(mark, tc.substr); got != tc.want {
			t.Errorf("containsSince(%q) = %v, want %v", tc.substr, got, tc.want)
		}
	}
}

func TestLogCaptureSinceSurvivesTrimming(t *testing.T) {
	c := &logCapture{}
	_, _ = c.Write([]byte("early\n"))
	mark := c.mark()

	// push the buffer past its limit so the head is cut off
	_, _ = c.Write([]byte(strings.Repeat("filler line\n", autoScriptLogLimit/8)))
	_, _ = c.Write([]byte("late marker\n"))

	if !c.containsSince(mark, "late marker") {
		t.Error("a line after the mark was lost by trimming")
	}

	if c.containsSince(mark, "early") {
		t.Error("a line before the mark was found")
	}
}
