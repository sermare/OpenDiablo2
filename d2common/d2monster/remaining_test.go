package d2monster

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
)

// remWorld implements the optional extensions of postcheck.go.
type remWorld struct {
	*fakeWorld
	dest       Point
	destOK     bool
	heals      int
	casts      []Point
	castSkills []int
	threat     int
	reachable  bool
	alt        Target
	altDist    int
	altOK      bool
	tileFlag   bool
	overlays   []int
}

func (w *remWorld) TeleportDest(*Brain) (Point, bool) { return w.dest, w.destOK }
func (w *remWorld) HealByLevel(*Brain)                { w.heals++ }

func (w *remWorld) CastSkillAt(_ *Brain, s int, _ Mode, p Point) bool {
	w.casts = append(w.casts, p)
	w.castSkills = append(w.castSkills, s)

	return true
}

func (w *remWorld) LevelThreat(*Brain) int        { return w.threat }
func (w *remWorld) Reachable(*Brain, Target) bool { return w.reachable }
func (w *remWorld) OwnTileFlagged(*Brain) bool    { return w.tileFlag }
func (w *remWorld) ShowOverlay(_ *Brain, id int)  { w.overlays = append(w.overlays, id) }

func (w *remWorld) ThreatTarget(*Brain, Target) (Target, int, bool) {
	return w.alt, w.altDist, w.altOK
}

func newRem(dist int) *remWorld {
	return &remWorld{fakeWorld: newFake(dist, false), dest: Point{150, 160}, destOK: true, reachable: true}
}

// seedFor finds a unit seed whose first three roll(100) values satisfy pred,
// so tests force each RNG gate deterministically.
func seedFor(t *testing.T, pred func(r [3]int) bool) uint32 {
	t.Helper()

	for s := uint32(1); s < 100000; s++ {
		x := d2rand.New(s)
		r := [3]int{int(x.Roll(100)), int(x.Roll(100)), int(x.Roll(100))}

		if pred(r) {
			return s
		}
	}

	t.Fatal("no seed")

	return 0
}

func teleBrain(t *testing.T, hp int, melee bool, pred func(r [3]int) bool) *Brain {
	t.Helper()

	p := profile("Skeleton", 50, 10, 50, 50)
	p.Melee = melee
	b := brainAt(p)
	b.Seed = d2rand.New(seedFor(t, pred))
	b.CanTeleport = true
	b.HPPercent = hp

	return b
}

// TestRemainingWoundedTeleport pins 0x5aedc0: gate flag, roll(100)<40, the
// wounded (<30%) or non-melee-and-near (<10) condition, roll(100)<15, then
// the optional heal (roll<25 when wounded) and a MonTeleport (0xb8) cast.
func TestRemainingWoundedTeleport(t *testing.T) {
	pass := func(r [3]int) bool { return r[0] < 40 && r[1] < 15 && r[2] < 25 }
	noHeal := func(r [3]int) bool { return r[0] < 40 && r[1] < 15 && r[2] >= 25 }
	fail1 := func(r [3]int) bool { return r[0] >= 40 }
	fail2 := func(r [3]int) bool { return r[0] < 40 && r[1] >= 15 }

	for _, c := range []struct {
		name   string
		hp     int
		melee  bool
		class  int
		dist   int
		pred   func([3]int) bool
		flag   bool
		destOK bool
		want   bool
		heals  int
	}{
		{"wounded heals", 29, true, 1, 30, pass, true, true, true, 1},
		{"wounded no heal roll", 29, true, 1, 30, noHeal, true, true, true, 0},
		{"hp 30 melee far: gate closed", 30, true, 1, 30, pass, true, true, false, 0},
		{"healthy ranged near", 100, false, 1, 9, pass, true, true, true, 0},
		{"healthy ranged dist 10", 100, false, 1, 10, pass, true, true, false, 0},
		{"healthy bighead near never", 100, false, 10, 5, pass, true, true, false, 0},
		{"wounded bighead", 10, false, 10, 5, pass, true, true, true, 1},
		{"class 0x159 melee-forced", 100, false, 0x159, 5, pass, true, true, false, 0},
		{"class 0x22d melee-forced", 100, false, 0x22d, 5, pass, true, true, false, 0},
		{"first roll fails", 10, true, 1, 5, fail1, true, true, false, 0},
		{"second roll fails", 10, true, 1, 5, fail2, true, true, false, 0},
		{"no flag 0x20", 10, true, 1, 5, pass, false, true, false, 0},
		{"no destination", 10, true, 1, 5, pass, true, false, false, 0},
	} {
		w := newRem(c.dist)
		w.destOK = c.destOK
		b := teleBrain(t, c.hp, c.melee, c.pred)
		b.Class = c.class
		b.CanTeleport = c.flag

		ctx := &Ctx{B: b, W: w, Params: Params{Target: &w.target, Dist: c.dist}}

		if got := woundedTeleport(ctx); got != c.want {
			t.Errorf("%s: got %v want %v", c.name, got, c.want)
		}

		if w.heals != c.heals {
			t.Errorf("%s: heals %d want %d", c.name, w.heals, c.heals)
		}

		if c.want && (len(w.casts) != 1 || w.casts[0] != w.dest || w.castSkills[0] != skillMonTeleport) {
			t.Errorf("%s: casts %v skills %v", c.name, w.casts, w.castSkills)
		}
	}
}

// TestRemainingTeleportDeadAndHealBlock covers a dead caster and state 0x34.
func TestRemainingTeleportDeadAndHealBlock(t *testing.T) {
	pass := func(r [3]int) bool { return r[0] < 40 && r[1] < 15 && r[2] < 25 }

	w := newRem(5)
	b := teleBrain(t, 10, true, pass)
	b.Mode = ModeDying

	if woundedTeleport(&Ctx{B: b, W: w, Params: Params{Target: &w.target, Dist: 5}}) {
		t.Error("dying unit teleported")
	}

	w = newRem(5)
	w.states[stateNoHeal] = true
	b = teleBrain(t, 10, true, pass)

	if !woundedTeleport(&Ctx{B: b, W: w, Params: Params{Target: &w.target, Dist: 5}}) || w.heals != 0 {
		t.Errorf("state 0x34 must block the heal only (heals %d)", w.heals)
	}
}

// TestRemainingThreatRetarget pins the tail of 0x5aefc0.
func TestRemainingThreatRetarget(t *testing.T) {
	alt := Target{ID: 9, X: 120, Y: 100, Size: 1, IsPlayer: true}

	for _, c := range []struct {
		name      string
		threat    int
		reachable bool
		melee     bool
		noWalk    bool
		altOK     bool
		ends      bool
		wantID    uint32
		wantDist  int
	}{
		{"threat 0 off", 0, false, true, false, true, false, 1, 30},
		{"dist equal threat", 30, false, true, false, true, false, 1, 30},
		{"reachable keeps target", 20, true, true, false, true, false, 1, 30},
		{"not melee", 20, false, false, false, true, false, 1, 30},
		{"no walk mode", 20, false, true, true, true, false, 1, 30},
		{"retarget", 20, false, true, false, true, false, 9, 12},
		{"nobody: wander 4, end", 20, false, true, false, false, true, 1, 30},
	} {
		w := newRem(30)
		w.threat, w.reachable, w.alt, w.altDist, w.altOK = c.threat, c.reachable, alt, 12, c.altOK
		b := brainAt(profile("Skeleton", 50, 10, 50, 50))
		b.Profile.Melee, b.Profile.NoWalk = c.melee, c.noWalk
		ctx := &Ctx{B: b, W: w, Params: Params{Target: &w.target, Dist: 30, InRange: true}}

		if got := threatRetarget(ctx); got != c.ends {
			t.Errorf("%s: ends=%v want %v", c.name, got, c.ends)
		}

		if ctx.Target.ID != c.wantID || ctx.Dist != c.wantDist || !ctx.InRange {
			t.Errorf("%s: target %d dist %d inrange %v", c.name, ctx.Target.ID, ctx.Dist, ctx.InRange)
		}

		if c.ends && len(w.log) != 1 {
			t.Errorf("%s: expected one wander, log %v", c.name, w.log)
		}
	}
}

// TestRemainingIdleWander pins the no-target branches of 0x5dd6b0 (mode 1) and
// 0x5dd7f0 (modes 4/5).
func TestRemainingIdleWander(t *testing.T) {
	for _, c := range []struct {
		name       string
		mode       int
		aggressive bool
		tile       bool
		noWalk     bool
		wander     bool
		wake       int
	}{
		{"mode1 aggressive", TargetStandard, true, false, false, true, 0},
		{"mode1 aggressive no walk", TargetStandard, true, false, true, false, 25},
		{"mode1 tile flagged", TargetStandard, false, true, false, true, 0},
		{"mode1 plain", TargetStandard, false, false, false, false, 25},
		{"mode4 aggressive only: sleeps", TargetFindOrWait, true, false, false, false, 20},
		{"mode4 tile flagged", TargetFindOrWait, false, true, false, true, 0},
		{"mode5 tile flagged", TargetFindThenThink, false, true, false, true, 0},
		{"mode5 plain", TargetFindThenThink, false, false, false, false, 20},
	} {
		w := newRem(200)
		w.hasTarget, w.tileFlag = false, c.tile
		b := brainAt(profile("Skeleton", 50, 10, 50, 50))
		b.Def = &AIDef{Name: "x", TargetMode: c.mode, Think: func(*Ctx) {}}
		b.Aggressive, b.Profile.NoWalk = c.aggressive, c.noWalk

		before := shadow(b)

		Tick(w, b)

		if got := len(w.log) > 0; got != c.wander {
			t.Errorf("%s: wander=%v want %v (log %v)", c.name, got, c.wander, w.log)
		}

		if !c.wander && b.Wake != c.wake {
			t.Errorf("%s: wake %d want %d", c.name, b.Wake, c.wake)
		}

		if c.wander {
			// Four LCG steps: the Wander(5) of 0x5dcff0 (the think is empty).
			for i := 0; i < 4; i++ {
				before.Step()
			}

			if b.Seed.Lo != before.Lo {
				t.Errorf("%s: seed not advanced exactly 4 steps", c.name)
			}
		}
	}
}

// TestRemainingRaiderOverlay pins 0x622020: overlay 0x96 when aip6 == 1, else
// 0x2e, shown on the tick where the counter reaches aip5.
func TestRemainingRaiderOverlay(t *testing.T) {
	for _, c := range []struct{ aip6, want int }{{1, 0x96}, {0, 0x2e}, {2, 0x2e}} {
		p := profile("SandRaider", 0, 0, 0, 0, 2, c.aip6)
		b := brainAt(p)
		w := newRem(20)

		for i := 0; i < 3; i++ {
			b.Wake = 0
			Tick(w, b)
		}

		if len(w.overlays) != 1 || w.overlays[0] != c.want {
			t.Errorf("aip6 %d: overlays %v want [%#x]", c.aip6, w.overlays, c.want)
		}
	}
}
