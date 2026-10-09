package d2autoscript

import "testing"

type logHost struct {
	recordHost
	has  bool
	exit []bool
}

func (h *logHost) LogContains(string) bool { return h.has }
func (h *logHost) Exit(pass bool)          { h.exit = append(h.exit, pass) }

func TestWaitLogHoldsUntilTheLineAppears(t *testing.T) {
	steps, err := Parse("waitlog:PARTY joined;say:go;exit")
	if err != nil {
		t.Fatal(err)
	}

	h := &logHost{}
	r := NewRunner(steps, h)

	for i := 0; i < 20; i++ {
		r.Advance(0.5)
	}

	if len(h.said) != 0 {
		t.Fatalf("the step after waitlog ran before the line appeared: %v", h.said)
	}

	h.has = true

	for i := 0; i < 5; i++ {
		r.Advance(0.5)
	}

	if len(h.said) != 1 || !r.Done() || r.Failed() || len(h.exit) != 1 || !h.exit[0] {
		t.Fatalf("said=%v done=%v failed=%v exit=%v", h.said, r.Done(), r.Failed(), h.exit)
	}
}

func TestWaitLogTimesOutAsAFailure(t *testing.T) {
	steps, _ := Parse("waitlog:never;exit")
	h := &logHost{}
	r := NewRunner(steps, h)

	for i := 0; i < int(WaitLogTimeout*2)+10 && !r.Done(); i++ {
		r.Advance(0.5)
	}

	if !r.Done() || !r.Failed() || len(h.exit) != 1 || h.exit[0] {
		t.Fatalf("done=%v failed=%v exit=%v", r.Done(), r.Failed(), h.exit)
	}

	if _, err := Parse("waitlog:"); err == nil {
		t.Fatal("waitlog needs a substring")
	}

	if _, err := Parse("panel:party"); err != nil {
		t.Fatalf("the party panel is a script panel: %v", err)
	}
}
