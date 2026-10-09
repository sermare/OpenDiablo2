package d2sfx

import (
	"errors"
	"math"
	"testing"
)

type fakeClock struct{ t int64 }

func (c *fakeClock) Now() int64 { return c.t }

type fakePlayer struct {
	row     *Row
	playing bool
	stopped bool
	vol     float64
	pan     float64
}

func (p *fakePlayer) Play()               { p.playing = true }
func (p *fakePlayer) Stop()               { p.playing, p.stopped = false, true }
func (p *fakePlayer) SetPan(v float64)    { p.pan = v }
func (p *fakePlayer) SetVolume(v float64) { p.vol = v }
func (p *fakePlayer) IsPlaying() bool     { return p.playing }

type harness struct {
	b       *Bank
	clock   *fakeClock
	players []*fakePlayer
	reports []Report
	rolls   []int
}

// rnd returns scripted values (each reduced modulo n) and then zeros.
func newHarness(rows []Row, voices int, rolls ...int) *harness {
	h := &harness{clock: &fakeClock{}, rolls: rolls}
	load := func(r *Row) (Player, error) {
		if r.FileName == "bad" {
			return nil, errors.New("missing")
		}

		p := &fakePlayer{row: r}
		h.players = append(h.players, p)

		return p, nil
	}
	rnd := func(n int) int {
		if len(h.rolls) == 0 {
			return 0
		}

		v := h.rolls[0] % n
		h.rolls = h.rolls[1:]

		return v
	}
	h.b = NewBank(NewTable(rows), voices, load, h.clock, rnd)
	h.b.OnReport = func(r Report) { h.reports = append(h.reports, r) }

	return h
}

func row(i int, f func(*Row)) Row {
	r := Row{Index: i, Handle: "s" + string(rune('a'+i%26)), FileName: "f", Volume: 255, Priority: 100}
	if f != nil {
		f(&r)
	}

	return r
}

func TestNoFileAndNoRow(t *testing.T) {
	h := newHarness([]Row{row(0, nil), row(1, func(r *Row) { r.FileName = "" })}, 2)

	for idx, want := range map[int]Decision{0: DecisionNoRow, 99: DecisionNoRow, 1: DecisionNoFile} {
		if got := h.b.Play(Request{Index: idx}).Report().Decision; got != want {
			t.Errorf("index %d: got %v want %v", idx, got, want)
		}
	}

	if h.b.ActiveVoices() != 0 {
		t.Error("nothing should play")
	}
}

func TestPriorityStealing(t *testing.T) {
	rows := []Row{row(0, nil),
		row(1, func(r *Row) { r.Priority = 50; r.Loop = true }),
		row(2, func(r *Row) { r.Priority = 80; r.Loop = true }),
		row(3, func(r *Row) { r.Priority = 255 }),
		row(4, func(r *Row) { r.Priority = 20 })}
	h := newHarness(rows, 2)
	low := h.b.Play(Request{Index: 1})
	mid := h.b.Play(Request{Index: 2})

	if !low.Playing() || !mid.Playing() {
		t.Fatal("both should play")
	}

	// lower priority than every voice: refused
	if d := h.b.Play(Request{Index: 4}).Report().Decision; d != DecisionNoVoice {
		t.Errorf("want no-voice, got %v", d)
	}

	// higher priority steals the weakest (index 1)
	hi := h.b.Play(Request{Index: 3})
	if hi.Report().Decision != DecisionStolen || hi.Report().Victim != low.Row().Handle {
		t.Fatalf("expected steal of low, got %+v", hi.Report())
	}

	if !mid.Playing() {
		t.Error("mid must survive")
	}

	// the stolen loop is waiting again and cannot steal back from higher priorities
	h.clock.t++
	h.b.Advance()

	if low.Playing() {
		t.Error("stolen loop must not be playing")
	}
}

func TestHeroPriorityBonusWraps(t *testing.T) {
	h := newHarness([]Row{row(0, nil), row(1, func(r *Row) { r.Priority = 80 }), row(2, func(r *Row) { r.Priority = 255 })}, 4)

	if p := h.b.Play(Request{Index: 1, Hero: true}).Report().Priority; p != 160 {
		t.Errorf("hero 80 -> 160, got %d", p)
	}

	h.clock.t += 5

	if p := h.b.Play(Request{Index: 2, Hero: true}).Report().Priority; p != (255+0x50)&0xff {
		t.Errorf("byte wrap expected, got %d", p)
	}
}

func TestGroupVariantAvoidsRecent(t *testing.T) {
	rows := []Row{row(0, nil)}
	for i := 1; i <= 4; i++ {
		rows = append(rows, row(i, nil))
	}

	rows[1].GroupSize = 4

	// g=4: window 2, no shortening roll. Rolls: pick 0 ->1, then pick 0 again (clash) -> redraw 1 ->2, ...
	h := newHarness(rows, 8, 0, 0, 1, 0, 1, 2, 3)
	var got []int

	for i := 0; i < 3; i++ {
		inst := h.b.Play(Request{Index: 1})
		got = append(got, inst.Report().Picked)
		h.clock.t += 10
		h.b.Advance()
	}

	want := []int{1, 2, 3}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("picks %v want %v", got, want)
		}
	}

	if h.b.PickVariant(2) != 2 { // GroupSize 0 rows are not groups
		t.Error("non-group row must return itself")
	}
}

func TestGroupWindowShrinksForSmallGroups(t *testing.T) {
	rows := []Row{row(0, nil), row(1, func(r *Row) { r.GroupSize = 2 }), row(2, nil)}
	// g=2: n=1; roll(3)==0 shrinks n to 0 so a repeat is allowed.
	h := newHarness(rows, 4, 1, 0, 0, 0, 0)
	h.b.history[1] = [2]int{1, 0}

	// shrink roll=1 (not 0): window 1 -> pick roll 0 -> index 1 clashes -> redraw 0?? scripted: 0 again clashes forever,
	// so script a 1 to escape.
	h.rolls = []int{1, 0, 1}

	if got := h.b.PickVariant(1); got != 2 {
		t.Errorf("window should force variant 2, got %d", got)
	}

	h.rolls = []int{0, 0} // shrink roll 0 -> no window -> repeat allowed

	if got := h.b.PickVariant(1); got != 1 {
		t.Errorf("shrunk window should allow repeat, got %d", got)
	}
}

func TestFixedVariantSkipsPick(t *testing.T) {
	rows := []Row{row(0, nil), row(1, func(r *Row) { r.GroupSize = 3 }), row(2, nil), row(3, nil)}
	h := newHarness(rows, 4, 2, 2)

	if got := h.b.Play(Request{Index: 1, FixedVariant: true}).Report().Picked; got != 1 {
		t.Errorf("fixed variant must stay on 1, got %d", got)
	}
}

func TestStopInstCutsOlder(t *testing.T) {
	rows := []Row{row(0, nil), row(1, func(r *Row) { r.StopInst = true; r.Loop = true })}
	h := newHarness(rows, 4)
	first := h.b.Play(Request{Index: 1})
	h.clock.t += 10
	second := h.b.Play(Request{Index: 1})

	if first.Alive() || !second.Playing() {
		t.Errorf("Stop Inst: older must die, newer play (older alive=%v)", first.Alive())
	}

	if !h.players[0].stopped {
		t.Error("older player not stopped")
	}
}

func TestDeferInstYieldsToPlaying(t *testing.T) {
	rows := []Row{row(0, nil), row(1, func(r *Row) { r.DeferInst = true; r.Loop = true })}
	h := newHarness(rows, 4)
	first := h.b.Play(Request{Index: 1})
	h.clock.t += 10
	second := h.b.Play(Request{Index: 1})

	if !first.Playing() || second.Alive() || second.Report().Decision != DecisionDeferred {
		t.Errorf("Defer Inst: new request must yield, got %+v", second.Report())
	}
}

func TestSameTickDuplicateDropped(t *testing.T) {
	rows := []Row{row(0, nil), row(1, func(r *Row) { r.Loop = true })}
	h := newHarness(rows, 4)
	h.b.Play(Request{Index: 1})
	h.clock.t++

	if d := h.b.Play(Request{Index: 1}).Report().Decision; d != DecisionSameTick {
		t.Errorf("want same-tick, got %v", d)
	}

	h.clock.t += 5

	if d := h.b.Play(Request{Index: 1}).Report().Decision; d != DecisionPlayed {
		t.Errorf("later copy should play, got %v", d)
	}
}

func TestFalloffRadiusAndInaudible(t *testing.T) {
	for f, want := range map[int]float64{0: 400, 1: 700, 2: 1000, 3: 1500, 4: 2000, 9: 700} {
		if FalloffRadius(f) != want {
			t.Errorf("falloff %d", f)
		}
	}

	rows := []Row{row(0, nil), row(1, func(r *Row) { r.Falloff = 0 }), row(2, func(r *Row) { r.Falloff = 0; r.Loop = true })}
	h := newHarness(rows, 4)
	h.b.SetListener(0, 0)

	if d := h.b.Play(Request{Index: 1, HasPos: true, X: 399}).Report().Decision; d != DecisionPlayed {
		t.Errorf("inside radius: %v", d)
	}

	if d := h.b.Play(Request{Index: 1, HasPos: true, X: 401}).Report().Decision; d != DecisionInaudible {
		t.Errorf("outside radius: %v", d)
	}

	// y counts double
	if d := h.b.Play(Request{Index: 1, HasPos: true, Y: 201}).Report().Decision; d != DecisionInaudible {
		t.Errorf("y doubled: %v", d)
	}

	// a looping sound waits and starts once the hero comes close
	loop := h.b.Play(Request{Index: 2, HasPos: true, X: 1000})
	if loop.Playing() || !loop.Alive() {
		t.Fatal("loop must wait")
	}

	h.clock.t++
	h.b.SetListener(900, 0)
	h.b.Advance()

	if !loop.Playing() {
		t.Error("loop should start when hero approaches")
	}

	// walking away stops it again, and it requeues
	h.clock.t++
	h.b.SetListener(0, 0)
	h.b.Advance()
	h.clock.t++
	h.b.Advance()

	if loop.Playing() || !loop.Alive() {
		t.Error("loop should be waiting after hero leaves")
	}
}

func TestPanAndDistanceGain(t *testing.T) {
	rows := []Row{row(0, nil), row(1, func(r *Row) { r.Falloff = 1 })}
	h := newHarness(rows, 2)
	h.b.Play(Request{Index: 1, HasPos: true, X: 350})

	p := h.players[0]
	if p.pan <= 0 || p.pan >= 1 {
		t.Errorf("right-of-hero pan should be in (0,1), got %v", p.pan)
	}

	if math.Abs(p.vol-0.5) > 1e-9 {
		t.Errorf("half radius -> 0.5 gain, got %v", p.vol)
	}

	if DistanceGain(700*700, 1) != 0 {
		t.Error("gain 0 at the radius")
	}
}

func TestFadeInAndOut(t *testing.T) {
	rows := []Row{row(0, nil), row(1, func(r *Row) { r.FadeIn = 10; r.FadeOut = 10; r.Loop = true })}
	h := newHarness(rows, 2)
	inst := h.b.Play(Request{Index: 1})

	if h.players[0].vol != 0 {
		t.Errorf("fade in starts silent, got %v", h.players[0].vol)
	}

	h.clock.t = 5
	h.b.Advance()

	if v := h.players[0].vol; math.Abs(v-127.0/255) > 0.01 {
		t.Errorf("half way, got %v", v)
	}

	h.clock.t = 10
	h.b.Advance()

	if h.players[0].vol != 1 {
		t.Errorf("full volume, got %v", h.players[0].vol)
	}

	h.b.Stop(inst)
	h.clock.t = 15
	h.b.Advance()

	if !inst.Alive() || h.players[0].vol > 0.51 {
		t.Errorf("fading out, got %v alive=%v", h.players[0].vol, inst.Alive())
	}

	h.clock.t = 21
	h.b.Advance()

	if inst.Alive() || !h.players[0].stopped {
		t.Error("should be stopped after fade out")
	}
}

func TestDurationAndFinish(t *testing.T) {
	rows := []Row{row(0, nil), row(1, func(r *Row) { r.Loop = true; r.Duration = 20 }), row(2, nil)}
	h := newHarness(rows, 4)
	loop := h.b.Play(Request{Index: 1})
	one := h.b.Play(Request{Index: 2})

	h.clock.t = 21
	h.b.Advance()

	if loop.Alive() {
		t.Error("Duration should end the loop")
	}

	h.players[1].playing = false // the file ended
	h.b.Advance()

	if one.Alive() || one.Report().Decision != DecisionFinished {
		t.Error("finished one-shot must be released")
	}

	if h.b.ActiveVoices() != 0 {
		t.Error("voices must be free")
	}
}

func TestDelayAndLoadFailure(t *testing.T) {
	rows := []Row{row(0, nil), row(1, nil), row(2, func(r *Row) { r.FileName = "bad" })}
	h := newHarness(rows, 2)
	inst := h.b.Play(Request{Index: 1, Delay: 5})

	if inst.Playing() {
		t.Fatal("delayed sound must wait")
	}

	h.clock.t = 5
	h.b.Advance()

	if !inst.Playing() {
		t.Error("should start after delay")
	}

	if d := h.b.Play(Request{Index: 2}).Report().Decision; d != DecisionLoadFailed {
		t.Errorf("got %v", d)
	}
}

func TestCompoundMerges(t *testing.T) {
	rows := []Row{row(0, nil), row(1, func(r *Row) { r.Compound = 10; r.Loop = true })}
	h := newHarness(rows, 4)
	a := h.b.Play(Request{Index: 1})
	h.clock.t = 5
	b2 := h.b.Play(Request{Index: 1})

	if a != b2 || h.b.QueueLen() != 1 {
		t.Error("requests inside the window merge")
	}

	h.clock.t = 50
	if c := h.b.Play(Request{Index: 1}); c == a {
		t.Error("requests outside the window are new")
	}
}

func TestSoloDucksOthers(t *testing.T) {
	rows := []Row{row(0, nil), row(1, func(r *Row) { r.Solo = true; r.Loop = true }), row(2, func(r *Row) { r.Loop = true })}
	h := newHarness(rows, 4)
	h.b.Play(Request{Index: 2})
	h.b.Play(Request{Index: 1})

	for i := 0; i < 40; i++ {
		h.clock.t++
		h.b.Advance()
	}

	if v := h.players[0].vol; math.Abs(v-0.7) > 1e-9 {
		t.Errorf("non-solo ducks to 70%%, got %v", v)
	}

	if h.players[1].vol != 1 {
		t.Errorf("solo stays at full, got %v", h.players[1].vol)
	}
}

func TestMasterVolumes(t *testing.T) {
	rows := []Row{row(0, nil), row(1, func(r *Row) { r.Volume = 128 }), row(2, func(r *Row) { r.MusicVol = true })}
	h := newHarness(rows, 4)
	h.b.SetVolumes(0.5, 0.25)
	h.b.Play(Request{Index: 1})
	h.b.Play(Request{Index: 2})

	if math.Abs(h.players[0].vol-128.0/255*0.5) > 1e-9 || math.Abs(h.players[1].vol-0.25) > 1e-9 {
		t.Errorf("volumes: %v %v", h.players[0].vol, h.players[1].vol)
	}
}

func TestGroupHeadPostPass(t *testing.T) {
	tb := NewTable([]Row{{Index: 0}, {Index: 1, GroupSize: 3}, {Index: 2}, {Index: 3}, {Index: 4}})
	for i, want := range []int{0, 1, 1, 1, 4} {
		if tb.Head(i) != want {
			t.Errorf("head(%d)=%d want %d", i, tb.Head(i), want)
		}
	}
}

func TestZeroPriorityStillPlays(t *testing.T) {
	h := newHarness([]Row{row(0, nil), row(1, func(r *Row) { r.Priority = 0 })}, 2)

	if got := h.b.Play(Request{Index: 1}).Report().Decision; got != DecisionPlayed {
		t.Errorf("Priority 0 (footsteps) got %v, want played", got)
	}
}
