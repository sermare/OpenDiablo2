package d2autoscript

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

type fakeHost struct {
	calls []string
	log   []string
	game  []string // lines logged by the game itself
	exit  []bool
	err   error
}

func (h *fakeHost) rec(s string) error { h.calls = append(h.calls, s); return h.err }

func (h *fakeHost) Move(x, y float64) error { return h.rec(fmt.Sprintf("move %g,%g", x, y)) }
func (h *fakeHost) Cast(s string, x, y float64, ok bool) error {
	return h.rec(fmt.Sprintf("cast %s %g,%g %v", s, x, y, ok))
}
func (h *fakeHost) MoveToNPC(n string) error { return h.rec("npc " + n) }
func (h *fakeHost) Panel(n string) error     { return h.rec("panel " + n) }
func (h *fakeHost) Say(c string) error       { return h.rec("say " + c) }
func (h *fakeHost) LogContains(s string) bool {
	for _, l := range append(h.game, h.log...) {
		if !strings.HasPrefix(l, "AUTOSCRIPT") && strings.Contains(l, s) {
			return true
		}
	}

	return false
}
func (h *fakeHost) Logf(f string, a ...interface{}) { h.log = append(h.log, fmt.Sprintf(f, a...)) }
func (h *fakeHost) Exit(p bool)                     { h.exit = append(h.exit, p) }

func TestParse(t *testing.T) {
	tests := []struct {
		spec string
		n    int
		bad  bool
	}{
		{"wait:2;move:10,20;cast:Fire Bolt@3,4;panel:inventory;say:fps;expect:log=hi;exit", 7, false},
		{"cast:Teleport", 1, false},
		{" wait:1 ; ; exit ", 2, false},
		{"", 0, true},
		{"wait:x", 0, true},
		{"wait:-1", 0, true},
		{"move:npc=Akara", 1, false},
		{"move:npc=", 0, true},
		{"move:1", 0, true},
		{"move:a,b", 0, true},
		{"panel:map", 0, true},
		{"expect:foo", 0, true},
		{"cast:", 0, true},
		{"say:", 0, true},
		{"dance", 0, true},
	}
	for _, tc := range tests {
		steps, err := Parse(tc.spec)
		if (err != nil) != tc.bad || len(steps) != tc.n {
			t.Errorf("Parse(%q) = %d steps, err %v", tc.spec, len(steps), err)
		}
	}
}

func TestParseCastTarget(t *testing.T) {
	s, err := Parse("cast:Fire Bolt@3,4.5")
	if err != nil || s[0].Arg != "Fire Bolt" || !s[0].HasXY || s[0].X != 3 || s[0].Y != 4.5 {
		t.Fatalf("got %+v %v", s, err)
	}
}

func run(t *testing.T, spec string, h *fakeHost, ticks int) *Runner {
	t.Helper()

	steps, err := Parse(spec)
	if err != nil {
		t.Fatal(err)
	}

	r := NewRunner(steps, h)
	for i := 0; i < ticks; i++ {
		r.Advance(0.5)
	}

	return r
}

func TestRunnerPass(t *testing.T) {
	h := &fakeHost{game: []string{"NPC menu opened: npc=\"Akara\""}}
	r := run(t, "wait:1;move:5,6;panel:inventory;say:fps;expect:log=npc=\"Akara\";exit", h, 20)

	if !r.Done() || r.Failed() || len(h.exit) != 1 || !h.exit[0] {
		t.Fatalf("done=%v failed=%v exit=%v", r.Done(), r.Failed(), h.exit)
	}

	want := []string{"move 5,6", "panel inventory", "say fps"}
	if strings.Join(h.calls, "|") != strings.Join(want, "|") {
		t.Errorf("calls %v", h.calls)
	}

	if h.log[len(h.log)-1] != "AUTOSCRIPT RESULT PASS" || h.log[0] != "AUTOSCRIPT step 1: wait:1" {
		t.Errorf("log %v", h.log)
	}
}

func TestRunnerWaitTakesTime(t *testing.T) {
	h := &fakeHost{}
	steps, _ := Parse("wait:2;move:1,1")
	r := NewRunner(steps, h)
	r.Advance(0.1) // runs the wait step
	r.Advance(1)
	r.Advance(0.5)

	if len(h.calls) != 0 {
		t.Fatalf("move ran during wait: %v", h.calls)
	}

	r.Advance(1)

	if len(h.calls) != 1 {
		t.Fatalf("move did not run after wait: %v", h.calls)
	}
}

func TestRunnerFailures(t *testing.T) {
	h := &fakeHost{}
	r := run(t, "expect:log=never seen", h, 10)

	if !r.Done() || !r.Failed() || h.exit[0] || h.log[len(h.log)-1] != "AUTOSCRIPT RESULT FAIL" {
		t.Errorf("expect: done=%v failed=%v exit=%v log=%v", r.Done(), r.Failed(), h.exit, h.log)
	}

	h = &fakeHost{err: errors.New("boom")}
	r = run(t, "move:1,1;exit", h, 10)

	if !r.Failed() || h.exit[0] {
		t.Errorf("host error should fail the script: %v", h.log)
	}

	r.Advance(1)

	if len(h.exit) != 1 {
		t.Errorf("finished script must not exit twice")
	}
}

func TestRunnerEndsWithoutExitStep(t *testing.T) {
	h := &fakeHost{}
	r := run(t, "panel:close", h, 5)

	if !r.Done() || r.Failed() || len(h.exit) != 1 {
		t.Errorf("done=%v failed=%v exit=%v", r.Done(), r.Failed(), h.exit)
	}
}
