package d2autoscript

import (
	"fmt"
	"strings"
	"testing"
)

// playHost is a fakeHost that also plays.
type playHost struct {
	fakeHost
	busy int // Busy() answers true this many times
}

func (h *playHost) WalkToExit(level int) error  { return h.rec(fmt.Sprintf("walkto exit %d", level)) }
func (h *playHost) WalkToObject(n string) error { return h.rec("walkto object " + n) }
func (h *playHost) Menu(row string) error       { return h.rec("menu " + row) }
func (h *playHost) Kill(radius, secs float64) error {
	return h.rec(fmt.Sprintf("kill r=%g s=%g", radius, secs))
}
func (h *playHost) Use(string) error                 { return nil }
func (h *playHost) Waypoint(int) error               { return nil }
func (h *playHost) Level() (level int, x, y float64) { return 1, 0, 0 }
func (h *playHost) Busy() bool {
	if h.busy > 0 {
		h.busy--
		return true
	}

	return false
}

func TestParsePlaySteps(t *testing.T) {
	tests := []struct {
		spec string
		bad  bool
		want Step
	}{
		{"walkto:exit=2", false, Step{Kind: KindWalkTo, Target: "exit", Arg: "exit=2", Level: 2}},
		{"walkto:object=Waypoint", false, Step{Kind: KindWalkTo, Target: "object", Arg: "Waypoint"}},
		{"walkto:exit=x", true, Step{}},
		{"walkto:exit=0", true, Step{}},
		{"walkto:object=", true, Step{}},
		{"walkto:somewhere", true, Step{}},
		{"kill:all", false, Step{Kind: KindKill, Target: "all", Arg: "all", Seconds: DefaultKillSeconds}},
		{"kill:all,45", false, Step{Kind: KindKill, Target: "all", Arg: "all,45", Seconds: 45}},
		{"kill:near=12,30", false, Step{Kind: KindKill, Target: "near", Arg: "near=12,30", Radius: 12, Seconds: 30}},
		{"kill:near=0", true, Step{}},
		{"kill:everything", true, Step{}},
		{"kill:all,-1", true, Step{}},
		{"until:LEVEL CHANGE from=1 to=2,20", false, Step{Kind: KindUntil, Arg: "LEVEL CHANGE from=1 to=2", Seconds: 20}},
		{"until:no timeout", true, Step{}},
		{"until:x,0", true, Step{}},
		{"menu:Talk", false, Step{Kind: KindMenu, Arg: "Talk"}},
		{"menu:", true, Step{}},
	}

	for _, tc := range tests {
		steps, err := Parse(tc.spec)
		if (err != nil) != tc.bad {
			t.Errorf("Parse(%q): err %v, want bad=%v", tc.spec, err, tc.bad)
			continue
		}

		if tc.bad {
			continue
		}

		got := steps[0]
		if got.Kind != tc.want.Kind || got.Target != tc.want.Target || got.Arg != tc.want.Arg ||
			got.Level != tc.want.Level || got.Seconds != tc.want.Seconds || got.Radius != tc.want.Radius {
			t.Errorf("Parse(%q) = %+v, want %+v", tc.spec, got, tc.want)
		}
	}
}

func TestRunnerPlaySteps(t *testing.T) {
	h := &playHost{busy: 3}
	r := runHost(t, "menu:Talk;walkto:exit=2;kill:near=9,5;walkto:object=Waypoint;exit", h, 40)

	if !r.Done() || r.Failed() {
		t.Fatalf("done=%v failed=%v log=%v", r.Done(), r.Failed(), h.log)
	}

	want := "menu Talk|walkto exit 2|kill r=9 s=5|walkto object Waypoint"
	if got := strings.Join(h.calls, "|"); got != want {
		t.Errorf("calls %q, want %q", got, want)
	}
}

func TestRunnerPlayStepsNeedAPlayHost(t *testing.T) {
	h := &fakeHost{}
	r := run(t, "walkto:exit=2;exit", h, 10)

	if !r.Failed() {
		t.Fatal("a host that cannot play must fail the step")
	}
}

func TestRunnerUntil(t *testing.T) {
	h := &fakeHost{}
	steps, _ := Parse("until:ARRIVED,3;say:after;exit")
	r := NewRunner(steps, h)

	r.Advance(0.5) // the until step starts waiting
	r.Advance(0.5)

	if len(h.calls) != 0 {
		t.Fatalf("the next step ran before the line appeared: %v", h.calls)
	}

	h.game = append(h.game, "[Game] ARRIVED at the gate")
	r.Advance(0.5) // sees the line
	r.Advance(0.5) // runs say
	r.Advance(0.5) // exit

	if !r.Done() || r.Failed() || len(h.calls) != 1 {
		t.Fatalf("done=%v failed=%v calls=%v log=%v", r.Done(), r.Failed(), h.calls, h.log)
	}
}

func TestRunnerUntilTimesOut(t *testing.T) {
	h := &fakeHost{}
	r := run(t, "until:NEVER,2;exit", h, 20)

	if !r.Done() || !r.Failed() {
		t.Fatalf("done=%v failed=%v", r.Done(), r.Failed())
	}

	if !strings.Contains(strings.Join(h.log, "\n"), "FAIL: no log line containing \"NEVER\"") {
		t.Errorf("log %v", h.log)
	}
}

func TestRunnerUntilAlreadySeen(t *testing.T) {
	h := &fakeHost{game: []string{"already here"}}
	r := run(t, "until:already here,2;exit", h, 10)

	if !r.Done() || r.Failed() {
		t.Fatalf("done=%v failed=%v log=%v", r.Done(), r.Failed(), h.log)
	}
}
