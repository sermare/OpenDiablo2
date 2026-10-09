package d2autoscript

import (
	"strings"
	"testing"
)

// markHost remembers lines with their positions, like the game's log capture.
type markHost struct {
	fakeHost
	lines []string
}

func (h *markHost) LogMark() int { return len(h.lines) }
func (h *markHost) LogContainsSince(mark int, s string) bool {
	for _, l := range h.lines[mark:] {
		if strings.Contains(l, s) {
			return true
		}
	}

	return false
}

// Say logs a line of its own, like a game action that answers at once.
func (h *markHost) Say(c string) error {
	h.lines = append(h.lines, "game: did "+c)
	return h.fakeHost.Say(c)
}

func TestRunnerUntilSeesTheLinesOfThePreviousAction(t *testing.T) {
	h := &markHost{}
	steps, _ := Parse("say:talk;wait:1;until:did talk,5;say:next;exit")
	r := NewRunner(steps, h)

	for i := 0; i < 12; i++ {
		r.Advance(0.5)
	}

	if r.Failed() || !r.Done() || len(h.calls) != 2 {
		t.Fatalf("done=%v failed=%v calls=%v log=%v", r.Done(), r.Failed(), h.calls, h.log)
	}
}

func TestRunnerUntilIgnoresOldLines(t *testing.T) {
	h := &markHost{lines: []string{"NPC menu opened: Akara"}}
	steps, _ := Parse("until:NPC menu opened,5;say:next;exit")
	r := NewRunner(steps, h)

	r.Advance(0.5)
	r.Advance(0.5)

	if len(h.calls) != 0 {
		t.Fatalf("an old log line satisfied the until step: %v", h.calls)
	}

	h.lines = append(h.lines, "NPC menu opened: Akara again")
	r.Advance(0.5)
	r.Advance(0.5)

	if len(h.calls) != 1 || r.Failed() {
		t.Fatalf("calls=%v failed=%v", h.calls, r.Failed())
	}
}
