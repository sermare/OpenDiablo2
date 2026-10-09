package d2autoscript

import "testing"

func TestParseAutomapStep(t *testing.T) {
	steps, err := Parse("automap:on;automap:Full;automap:mini;automap:off;automap:stats")
	if err != nil {
		t.Fatal(err)
	}

	want := []string{"on", "full", "mini", "off", "stats"}
	for i, s := range steps {
		if s.Kind != KindAutomap || s.Arg != want[i] {
			t.Errorf("step %d = %+v", i, s)
		}
	}

	if _, err := Parse("automap:sideways"); err == nil {
		t.Error("unknown automap mode must fail")
	}

	if _, err := Parse("automap"); err == nil {
		t.Error("automap without a mode must fail")
	}
}

type recordHost struct{ said []string }

func (h *recordHost) Move(x, y float64) error                   { return nil }
func (h *recordHost) MoveToNPC(label string) error              { return nil }
func (h *recordHost) Cast(string, float64, float64, bool) error { return nil }
func (h *recordHost) Panel(string) error                        { return nil }
func (h *recordHost) Say(c string) error                        { h.said = append(h.said, c); return nil }
func (h *recordHost) LogContains(string) bool                   { return true }
func (h *recordHost) Logf(string, ...interface{})               {}
func (h *recordHost) Exit(bool)                                 {}

func TestAutomapStepRunsConsoleCommand(t *testing.T) {
	steps, _ := Parse("automap:full;automap:stats")
	h := &recordHost{}
	r := NewRunner(steps, h)

	for i := 0; i < 5; i++ {
		r.Advance(0.1)
	}

	if len(h.said) != 2 || h.said[0] != "automap full" || h.said[1] != "automap stats" {
		t.Errorf("said %v", h.said)
	}
}
