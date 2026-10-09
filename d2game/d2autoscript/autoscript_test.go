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

	return runHost(t, spec, h, ticks)
}

func runHost(t *testing.T, spec string, h Host, ticks int) *Runner {
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

type fakeLevelHost struct {
	fakeHost
	level    int
	changeAt int // Level call count at which the level becomes 35
	ticks    int
	busyFor  int // Busy answers true this many times
}

func (h *fakeLevelHost) Busy() bool {
	if h.busyFor > 0 {
		h.busyFor--
		return true
	}

	return false
}
func (h *fakeLevelHost) Use(t string) error   { return h.rec("use " + t) }
func (h *fakeLevelHost) Waypoint(l int) error { return h.rec(fmt.Sprintf("waypoint %d", l)) }
func (h *fakeLevelHost) Level() (int, float64, float64) {
	h.ticks++
	if h.changeAt > 0 && h.ticks >= h.changeAt {
		return 35, 12.5, 7
	}

	return h.level, 12.5, 7
}

func TestParseLevelSteps(t *testing.T) {
	for _, spec := range []string{"use:Waypoint", "use:119", "waypoint:35", "expect:level=35"} {
		if _, err := Parse(spec); err != nil {
			t.Errorf("Parse(%q): %v", spec, err)
		}
	}

	for _, spec := range []string{"use:", "waypoint:", "waypoint:x", "waypoint:0", "expect:level=", "expect:level=x", "expect:level=0"} {
		if _, err := Parse(spec); err == nil {
			t.Errorf("Parse(%q) should fail", spec)
		}
	}

	s, _ := Parse("waypoint:35;expect:level=3")
	if s[0].Level != 35 || !s[1].HasLevel || s[1].Level != 3 {
		t.Errorf("steps %+v", s)
	}
}

func TestRunnerLevelSteps(t *testing.T) {
	h := &fakeLevelHost{level: 1}
	r := runHost(t, "use:Waypoint;waypoint:1;expect:level=1;exit", h, 10)

	if !r.Done() || r.Failed() {
		t.Fatalf("done=%v failed=%v log=%v", r.Done(), r.Failed(), h.log)
	}

	if strings.Join(h.calls, "|") != "use Waypoint|waypoint 1" {
		t.Errorf("calls %v", h.calls)
	}

	found := false

	for _, l := range h.log {
		found = found || l == "AUTOSCRIPT level=1 hero=(12.5,7.0) want=1"
	}

	if !found {
		t.Errorf("expect:level did not log the level and position: %v", h.log)
	}
}

func TestExpectLevelWaitsForChange(t *testing.T) {
	// the level becomes 35 only after a few polls: the step must wait, not fail
	h := &fakeLevelHost{level: 1, changeAt: 6}
	r := runHost(t, "expect:level=35;exit", h, 20)

	if !r.Done() || r.Failed() {
		t.Fatalf("done=%v failed=%v log=%v", r.Done(), r.Failed(), h.log)
	}

	// and it fails once the timeout passes
	h = &fakeLevelHost{level: 1}
	r = runHost(t, "expect:level=35;exit", h, 40)

	if !r.Done() || !r.Failed() {
		t.Fatalf("a level that never arrives must fail: done=%v failed=%v", r.Done(), r.Failed())
	}
}

func TestLevelStepsNeedLevelHost(t *testing.T) {
	h := &fakeHost{}
	r := run(t, "use:x;exit", h, 5)

	if !r.Failed() {
		t.Error("use on a plain host must fail")
	}
}

func TestRunnerHoldsWhileBusy(t *testing.T) {
	h := &fakeLevelHost{level: 1}
	steps, _ := Parse("use:Waypoint;waypoint:3")
	r := NewRunner(steps, h)

	r.Advance(0.5) // use runs
	h.busyFor = 3

	for i := 0; i < 3; i++ {
		r.Advance(0.5)

		if len(h.calls) != 1 {
			t.Fatalf("step ran while the host was busy: %v", h.calls)
		}
	}

	r.Advance(0.5)

	if strings.Join(h.calls, "|") != "use Waypoint|waypoint 3" {
		t.Errorf("calls %v", h.calls)
	}
}
