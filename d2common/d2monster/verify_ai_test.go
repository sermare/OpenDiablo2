package d2monster

import (
	"fmt"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
)

// Pinning tests of the monster-AI verification pass (every behaviour here was
// re-read from Game.exe 1.14b; addresses in the comments).

// awake marks a Summoner as already woken so tests of its think function see
// the think and not the one-shot wake-up (0x5aed10).
func awake(b *Brain) *Brain {
	b.WakeShouted = true

	return b
}

// TestVerifyStateTableModes pins the forced-state table at 0x73a548 (read from
// memory): state 12 is mode 0 (the think acquires itself), 13 and 16 mode 1,
// 17 mode 2.
func TestVerifyStateTableModes(t *testing.T) {
	for state, want := range map[int]int{
		StateWander: 1, StateFormation: 1, StateTownGuard: 0, StateAttack25: 1, StateRelease: 1,
		StateBlind: 1, StateFear: 1, StateTaunted: 0, StateLeash: 1, StateCharge: 1,
		StateImp: 1, StateBlind2: 2,
	} {
		d := StateDef(state)
		if d == nil {
			t.Errorf("state %d not registered", state)

			continue
		}

		if d.TargetMode != want {
			t.Errorf("state %d: mode %d, exe table says %d", state, d.TargetMode, want)
		}
	}
}

// TestVerifyAggroStrict pins 0x5dc560: a player at exactly the aggro radius is
// not acquired (the compare is "below"), one step closer is.
func TestVerifyAggroStrict(t *testing.T) {
	for _, c := range []struct {
		dist int
		want bool
	}{{34, true}, {35, false}, {36, false}} {
		w := newFake(c.dist, false)
		b := brainAt(profile("Skeleton", 50, 10, 50, 50))
		b.Profile.AIDist = 35

		Tick(w, b)

		if b.HasTarget != c.want {
			t.Errorf("dist %d: acquired=%v want %v", c.dist, b.HasTarget, c.want)
		}
	}
}

// TestVerifyFindModes pins target modes 4 and 5 (0x5af2a5, 0x5af2cb,
// 0x5dd7f0): with nobody in range both sleep 20 frames; mode 4 ends the tick,
// mode 5 runs the think anyway.
func TestVerifyFindModes(t *testing.T) {
	for _, c := range []struct {
		mode int
		want bool
	}{{TargetFindOrWait, false}, {TargetFindThenThink, true}} {
		thought := false
		w := newFake(200, false)
		b := brainAt(profile("Skeleton"))
		b.Def = &AIDef{Name: "x", TargetMode: c.mode, Think: func(*Ctx) { thought = true }}

		ran := Tick(w, b)

		if ran != c.want || thought != c.want {
			t.Errorf("mode %d: ran=%v thought=%v want %v", c.mode, ran, thought, c.want)
		}

		if c.mode == TargetFindOrWait && b.Wake != waitNoTargetFrames {
			t.Errorf("mode 4 wake %d want 20", b.Wake)
		}
	}
}

// TestVerifyWander pins 0x5dcff0: parity picks the full-n axis, roll(n) the
// other, then one sign step for x and one for y.
func TestVerifyWander(t *testing.T) {
	for seed := uint32(1); seed <= 100; seed++ {
		w := newFake(5, true)
		b := brainAt(profile("Idle"))
		b.Seed.Init(seed)
		r := d2rand.New(seed)
		(&Ctx{B: b, W: w}).Wander(5)

		s1 := r.Step()
		rr := int(r.Roll(5))
		sx, sy := r.Step()&1, r.Step()&1

		dx, dy := rr, 5
		if s1&1 != 0 {
			dx, dy = 5, rr
		}

		if sx != 0 {
			dx = -dx
		}

		if sy != 0 {
			dy = -dy
		}

		if want := fmt.Sprintf("walk-to(%d,%d)", 100+dx, 100+dy); w.last() != want {
			t.Fatalf("seed %d: %s want %s", seed, w.last(), want)
		}
	}
}

// TestVerifyCircleDirection pins the 5/6 split of 0x5de5e0 on the low byte of
// the LCG (threshold 0x80), not on bit 0.
func TestVerifyCircleDirection(t *testing.T) {
	seen := map[bool]bool{}

	for seed := uint32(1); seed <= 400; seed++ {
		b := brainAt(profile("Idle"))
		b.Seed.Init(seed)
		lo := d2rand.New(seed).Step() & 0xff

		w := newFake(10, true)
		(&Ctx{B: b, W: w}).Circle(Target{X: 110, Y: 100}, 4)

		want := "walk-to(100,104)"
		if lo >= 0x80 {
			want = "walk-to(100,96)"
		}

		if w.last() != want {
			t.Fatalf("seed %d lo %#x: %s want %s", seed, lo, w.last(), want)
		}

		seen[lo >= 0x80] = true
	}

	if !seen[true] || !seen[false] {
		t.Fatal("both directions must occur")
	}
}

// TestVerifySummonerWake pins 0x5aed10: the first player sighting shouts once,
// sleeps 20 frames and ends the tick; the +0x14 counter (not the distance)
// gates it; other classes never wake.
func TestVerifySummonerWake(t *testing.T) {
	w := newFake2(8, false)
	b := brainAt(summonerProfile(100, 0, 100, 40, 120, 0, 10, 40))

	if Tick(w, b) || b.Wake != 20 || w.shouts != 1 || !b.WakeShouted {
		t.Fatalf("first sight: wake %d shouts %d", b.Wake, w.shouts)
	}

	b.Wake = 0
	Tick(w, b)

	if w.shouts != 1 {
		t.Fatal("wake-up must be one-shot")
	}

	b2 := brainAt(summonerProfile(100, 0, 100, 40, 120, 0, 10, 40))
	b2.Scratch[0] = 20
	w2 := newFake2(8, false)
	Tick(w2, b2)

	if w2.shouts != 0 {
		t.Fatal("counter >= 20 must skip the wake-up")
	}

	w3 := newFake(8, false)
	Tick(w3, brainAt(profile("Skeleton", 50, 10, 50, 50)))

	if w3.shouts != 0 {
		t.Fatal("non-summoner shouted")
	}
}

// TestVerifyMinionCommand pins 0x5e09c0: a type-1 command is live only while
// count > frame (strict).
func TestVerifyMinionCommand(t *testing.T) {
	run := func(frame, count int) *Brain {
		w := &cmdWorld{fakeWorld: newFake(10, true)}
		w.frame = frame
		b := brainAt(profile("Minion", 100, 100, 0, 9))
		b.AppendCommand(Command{Type: CmdAlert, Count: count, Target: 1})
		Tick(w, b)

		return b
	}

	if run(50, 50).PeekCommand() != nil {
		t.Error("command at count == frame must be expired")
	}

	if run(49, 50).PeekCommand() == nil {
		t.Error("command at count > frame must be kept")
	}
}

// TestVerifyRaiderCharge pins the SandRaider counter (0x5ef800): the tick on
// which the counter reaches aip5 sleeps aidel+1; later ticks set the charged
// flag.
func TestVerifyRaiderCharge(t *testing.T) {
	p := profile("SandRaider", 0, 0, 0, 0, 3)
	p.AIDel = 4
	b := brainAt(p)
	w := newFake(20, false)

	for i := 0; i < 2; i++ {
		b.Wake = 0
		Tick(w, b)
	}

	b.Wake = 0
	Tick(w, b) // counter == aip5

	if b.Scratch[raiderCount] != 3 || b.Wake != w.frame+5 {
		t.Fatalf("count %d wake %d", b.Scratch[raiderCount], b.Wake)
	}

	b.Wake = 0
	w.log = nil
	Tick(w, b) // counter > aip5: charged, out of reach: walks

	if b.Scratch[raiderCharged] != 1 || len(w.log) == 0 {
		t.Fatalf("charged=%d log=%v", b.Scratch[raiderCharged], w.log)
	}
}

// cmdWorld resolves every commanded unit id to the fake's target.
type cmdWorld struct{ *fakeWorld }

func (w *cmdWorld) UnitTarget(*Brain, uint32) (Target, int, bool) { return w.target, w.dist, true }
