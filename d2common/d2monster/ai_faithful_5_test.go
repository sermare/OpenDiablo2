package d2monster

import (
	"fmt"
	"strings"
	"testing"
)

// f5World adds the batch-5 host interfaces to a fake2.
type f5World struct {
	*fake2
	cands     []DiaCandidate
	useCands  bool
	auraState map[int]int
	blocked   map[int]bool
	footprint bool
	hooks     []string
	prison    int
	attacked  []Target
}

func newF5(dist int, inRange bool) *f5World {
	return &f5World{fake2: newFake2(dist, inRange), auraState: map[int]int{}, blocked: map[int]bool{}}
}

func (w *f5World) DiabloCandidates(*Brain) []DiaCandidate { return w.cands }
func (w *f5World) SlotAuraState(_ *Brain, slot int) int {
	if v, ok := w.auraState[slot]; ok {
		return v
	}

	return -1
}
func (w *f5World) LineBlocked(_ *Brain, _ Target, mask int) bool { return w.blocked[mask] }
func (w *f5World) FootprintBlocked(*Brain) bool                  { return w.footprint }
func (w *f5World) BossThink(_ *Brain, boss string)               { w.hooks = append(w.hooks, boss) }
func (w *f5World) PrisonTest(*Brain, Target) int                 { return w.prison }
func (w *f5World) Attack(b *Brain, m Mode, t Target) bool {
	w.attacked = append(w.attacked, t)

	return w.fake2.Attack(b, m, t)
}

// f5NoCands hides DiabloTargetSource so the tick target is the only candidate.
type f5Plain struct{ *f5World }

func (f5Plain) DiabloCandidates() {}

type f5Room struct {
	*f5World
	same bool
}

func (w f5Room) SameRoom(*Brain, Target) bool { return w.same }

type f5Cell struct {
	*f5World
	to Point
}

func (w f5Cell) NearestFreeCell(*Brain, Point) (Point, bool) { return w.to, true }

type f5Right struct {
	*f5World
	has bool
}

func (w f5Right) HasRightSkill(*Brain) bool { return w.has }

func TestBatch5Registered(t *testing.T) {
	for _, n := range append(FaithfulBatch5(), ReverifiedBatch5()...) {
		d, ok := Lookup(n)
		if !ok {
			t.Fatalf("%s not registered", n)
		}

		if m, ok := AITargetMode(n); !ok || m != d.TargetMode {
			t.Errorf("%s: target mode %d, exe %d", n, d.TargetMode, m)
		}
	}
}

// UberDiablo 0x5e8ea0 and UberMephisto 0x5f72f0 are 3-byte RETs in the exe.
func TestUberDiabloAndMephistoDoNothing(t *testing.T) {
	for _, n := range []string{"UberDiablo", "UberMephisto"} {
		w := newF5(5, true)
		b := brainAt(withSkills(profile(n), 0, 1, 2))
		def, _ := Lookup(n)
		b.SetAI(def)

		if !Tick(w, b) {
			t.Fatalf("%s: no think", n)
		}

		if len(w.log) != 0 || b.Wake != waitForever {
			t.Errorf("%s: log %v wake %d", n, w.log, b.Wake)
		}
	}
}

func TestDiabloScore(t *testing.T) {
	st := DiaStats{S24: 10, S25: 20, S27: 30, S29: 40, S2B: 50, S16: 100, S31: 10, S33: 20, S35: 30, S37: 40,
		S3A: 512, LifePct: 50, LeftRank: 2, LeftLevel: 10, RightRank: 1, RightLevel: 5}

	// by hand: melee: cold 50 + (10+40)*4 + 40 + 30 = 320 / 15 = 21; reach 100*5 = 500;
	// rank 25/4 = 6, damage (2+40+30+20+10+100)/2 = 101: (6+101)*2 = 214; 735/22 = 33
	lowLife, state := st, st
	lowLife.LifePct = 10
	state.HasState0xb = true

	monster := st
	monster.IsMonster, monster.Threat = true, 1

	threat2 := monster
	threat2.Threat = 2

	cases := []struct {
		name                      string
		s                         DiaStats
		melee, lineClear, leashed bool
		want                      int
	}{
		{"melee", st, true, true, false, 33},
		{"ranged with a clear line", st, false, true, false, 27}, // (18+375+214)/22
		{"ranged, hero outside the leash", st, false, true, true, 50},
		{"low life", lowLife, true, true, false, 47},
		{"state 0xb on the target", state, true, true, false, 51},
		{"harmless monster", monster, true, true, false, 0},
		{"threat 2 monster", threat2, true, true, false, 33},
		{"nothing scores 1", DiaStats{LifePct: 100}, false, false, false, 1},
	}

	for _, c := range cases {
		if got := DiabloScore(c.s, c.melee, c.lineClear, c.leashed); got != c.want {
			t.Errorf("%s: %d want %d", c.name, got, c.want)
		}
	}
}

func TestDiabloActionWeights(t *testing.T) {
	base := func() DiaSituation {
		return DiaSituation{HasTarget: true, Stats: DiaStats{LifePct: 100}, LineClear: true, Count: 3, Score: 50,
			Scaling: 2, EdgeDist: 10, PortalTest: 1}
	}

	w := func(s DiaSituation) [diaWeights]int { return DiabloActionWeights(s) }

	want := func(t *testing.T, name string, got [diaWeights]int, exp map[int]int) {
		t.Helper()

		for i, v := range got {
			if v != exp[i] {
				t.Errorf("%s: weight[%d]=%d want %d (%v)", name, i, v, exp[i], got)
			}
		}
	}

	// melee
	s := base()
	s.Melee = true
	want(t, "melee", w(s), map[int]int{diaA1: 40, diaA2: 70, diaLight: 40, diaFire: 24, diaCold: 40, diaWall: 15})

	s.Stats = DiaStats{LifePct: 10, S27: 50, S29: 20, HasState0xb: true}
	want(t, "melee hurt chilled", w(s), map[int]int{diaA1: 50, diaA2: 70, diaLight: 40, diaFire: 14})

	s = base()
	s.Melee, s.LineClear, s.PortalNear, s.HasPortalOb = true, false, true, true
	want(t, "melee blocked line + portal", w(s), map[int]int{diaA1: 40, diaA2: 70, diaCold: 40, diaWall: 15, diaPrisonAny: 10})

	// engaged, not in melee reach
	s = base()
	want(t, "engaged", w(s), map[int]int{diaLight: 25, diaFire: 25, diaPrison: 20, diaCircle: 20, diaWall: 15, diaRun: 10})

	s.EdgeDist = 30
	want(t, "engaged far", w(s), map[int]int{diaFire: 25, diaPrison: 20, diaCircle: 20, diaWall: 10, diaRun: 20})

	s = base()
	s.Count, s.Scaling = 1, 2
	want(t, "engaged alone", w(s), map[int]int{diaLight: 25, diaFire: 15, diaPrison: 20, diaCircle: 20, diaWall: 15, diaRun: 10})

	s.Scaling = 1 // few players: no prison
	if w(s)[diaPrison] != 0 {
		t.Errorf("prison with few players: %v", w(s))
	}

	s = base()
	s.Score = 70
	if w(s)[diaPrison] != 30 {
		t.Errorf("strong target: prison %d", w(s)[diaPrison])
	}

	s = base()
	s.GroundUser = true
	want(t, "ground-skill hero", w(s), map[int]int{diaLight: 25, diaFire: 35, diaPrison: 20, diaCircle: 20, diaWall: 15, diaRun: 30, diaFirewall: 15})

	s = base()
	s.TargetAway = true
	want(t, "hero outside the leash", w(s), map[int]int{diaLight: 25, diaFire: 25, diaPrison: 20, diaCircle: 10, diaWall: 15, diaFirewall: 15})

	s.TargetFar = true
	g := w(s)
	if g[diaPrison] != 20 || g[diaRunHome] != 60 {
		t.Errorf("far hero: %v", g)
	}

	s.PortalNear, s.HasPortalOb = true, true
	if g := w(s); g[diaPrisonAny] != 15 {
		t.Errorf("portal: %v", g)
	}

	s.PortalTest = 0 // the exe zeroes it when the test returns zero
	if g := w(s); g[diaPrisonAny] != 0 {
		t.Errorf("portal test zero: %v", g)
	}

	s = base()
	s.PrisonTest = 1
	if g := w(s); g[diaPrison] != 0 {
		t.Errorf("prison test nonzero: %v", g)
	}

	// no line to the target, out of reach
	s = base()
	s.LineClear = false
	want(t, "blocked", w(s), map[int]int{diaAttack11: 5, diaFire: 25, diaWall: 25, diaPrison: 40, diaCircle: 25})

	s.Count = 1
	want(t, "blocked alone", w(s), map[int]int{diaAttack11: 5, diaWalkHome: 25, diaWall: 20, diaCircle: 25})

	s = base()
	s.LineClear, s.TargetAway, s.Count = false, true, 1
	want(t, "blocked, alone, away", w(s), map[int]int{diaAttack11: 5, diaWall: 20, diaCircle: 15, diaFirewall: 25, diaPrison: 15})

	s = base()
	s.LineClear, s.GroundUser, s.TargetFar, s.PortalNear, s.HasPortalOb = false, true, true, true, true
	want(t, "blocked, ground hero, far", w(s), map[int]int{diaAttack11: 5, diaFire: 25, diaWall: 25, diaPrison: 40, diaCircle: 25,
		diaFirewall: 15, diaRunHome: 60, diaPrisonAny: 20})
}

func diaBrain(t *testing.T, diff Difficulty, slots ...int) *Brain {
	t.Helper()

	b := NewBrain(7, 1, diff, withSkills(profile("Diablo"), slots...), testSeed)
	b.X, b.Y = 100, 100
	def, _ := Lookup("Diablo")
	b.SetAI(def)

	return b
}

func TestDiabloActions(t *testing.T) {
	cases := []struct {
		name  string
		act   int
		slots []int
		diff  Difficulty
		log   string
		wake  int
	}{
		{"walk home goes to the target", diaWalkHome, nil, Normal, "walk-to(105,100)", -1},
		{"attack 1", diaA1, nil, Normal, "attack4", -1},
		{"attack 2", diaA2, nil, Normal, "attack5", -1},
		{"mode 0xb", diaAttack11, nil, Normal, "attack11", -1},
		{"fire", diaFire, []int{2}, Normal, "cast2", -1},
		{"fire without the skill waits 2", diaFire, nil, Normal, "", 2},
		{"cold", diaCold, []int{1}, Normal, "cast1", -1},
		{"wall", diaWall, []int{3}, Normal, "cast3", -1},
		{"run", diaRun, []int{4}, Normal, "cast4", -1},
		{"firewall", diaFirewall, []int{5}, Normal, "cast5", -1},
		{"prison", diaPrison, []int{6}, Normal, "cast6", -1},
		{"circle strafes", diaCircle, nil, Normal, "walk-to(", -1},
		{"wander", diaWander, nil, Normal, "walk-to(", -1},
		{"wait normal", diaWait, nil, Normal, "", 12},
		{"wait nightmare", diaWait, nil, Nightmare, "", 8},
		{"wait hell", diaWait, nil, Hell, "", 4},
		{"prison on a portal needs a player target", diaPrisonAny, []int{6}, Normal, "", 3},
	}

	for _, c := range cases {
		w := newF5(5, true)
		w.target.IsPlayer = c.name != "prison on a portal needs a player target"
		b := diaBrain(t, c.diff, c.slots...)
		b.Scratch[0] = c.act

		Tick(f5Plain{w}, b)

		if c.log != "" && !strings.HasPrefix(w.last(), c.log) {
			t.Errorf("%s: log %v", c.name, w.log)
		}

		if c.log == "" && len(w.log) != 0 {
			t.Errorf("%s: log %v", c.name, w.log)
		}

		if c.wake >= 0 && b.Wake != c.wake {
			t.Errorf("%s: wake %d want %d", c.name, b.Wake, c.wake)
		}

		if c.act != diaLight && b.Scratch[0] != 0 {
			t.Errorf("%s: the continuing action must be cleared, S0=%d", c.name, b.Scratch[0])
		}
	}
}

func TestDiabloLightChannelAndState(t *testing.T) {
	w := newF5(5, true)
	b := diaBrain(t, Normal, 0)
	b.Scratch[0] = diaLight
	Tick(w, b)

	if w.last() != "cast0" || b.Scratch[0] != diaLight {
		t.Fatalf("breath: %v S0=%d", w.log, b.Scratch[0])
	}

	// the breath's state 0xc ends the channel: cleared, wait 2
	w = newF5(5, true)
	w.states[0xc] = true
	b = diaBrain(t, Normal, 0)
	b.Scratch[0] = diaLight
	Tick(w, b)

	if len(w.log) != 0 || b.Wake != 2 || b.Scratch[0] != 0 {
		t.Fatalf("state 0xc: %v wake %d S0=%d", w.log, b.Wake, b.Scratch[0])
	}
}

func TestDiabloRunHome(t *testing.T) {
	// the first walk to the anchor
	w := newF5(5, true)
	b := diaBrain(t, Normal)
	b.AppendCommand(Command{Type: CmdAnchor, X: 60, Y: 80})
	b.Scratch[0] = diaRunHome
	Tick(w, b)

	if w.last() != "walk-to(60,80)" || b.Scratch[0] != 0 {
		t.Fatalf("run home: %v", w.log)
	}

	// when no path is queued the second try aims half the tile distance beyond
	// the anchor on his own side, then the tick waits 2
	w = newF5(5, true)
	w.failMove = true
	b = diaBrain(t, Normal)
	b.AppendCommand(Command{Type: CmdAnchor, X: 60, Y: 80})
	b.Scratch[0] = diaRunHome
	Tick(w, b)

	if len(w.log) != 0 || b.Wake != 2 {
		t.Fatalf("no path: %v wake %d", w.log, b.Wake)
	}
}

func TestDiabloNoTarget(t *testing.T) {
	// no candidate: the footprint test decides between wander (blocked) and wait
	w := newF5(5, true)
	w.useCands = true
	w.footprint = true
	b := diaBrain(t, Normal)
	Tick(f5Empty{w}, b)

	if !strings.HasPrefix(w.last(), "walk-to(") {
		t.Fatalf("blocked: %v", w.log)
	}

	w = newF5(5, true)
	b = diaBrain(t, Normal)
	Tick(f5Empty{w}, b)

	if len(w.log) != 0 || (b.Wake != 12 && b.Wake != 0) {
		t.Fatalf("free: %v wake %d", w.log, b.Wake)
	}
}

// f5Empty offers an empty candidate list.
type f5Empty struct{ *f5World }

func (f5Empty) DiabloCandidates(*Brain) []DiaCandidate { return nil }

func TestDiabloPicksBestScore(t *testing.T) {
	weak := DiaCandidate{Target: Target{ID: 1, X: 105, Y: 100, Size: 1, IsPlayer: true}, Stats: DiaStats{LifePct: 100, S16: 10}}
	hurt := DiaCandidate{Target: Target{ID: 2, X: 120, Y: 100, Size: 1, IsPlayer: true}, Stats: DiaStats{LifePct: 10, S16: 10}}
	dead := DiaCandidate{Target: Target{ID: 3, X: 101, Y: 100, Size: 1, IsPlayer: true}, Stats: DiaStats{LifePct: 100, S16: 999}, Dead: true}

	w := newF5(5, true)
	w.cands = []DiaCandidate{weak, dead, hurt}

	b := diaBrain(t, Normal)
	c := &Ctx{B: b, W: w}
	best, score, count, _ := c.diabloPick(nil)

	if best == nil || best.ID != 2 || count != 3 || score <= 0 {
		t.Fatalf("best=%v score=%d count=%d", best, score, count)
	}

	// without a host list the tick target is scored alone
	c = &Ctx{B: b, W: f5Plain{w}, Params: Params{Target: &weak.Target}}
	best, _, count, _ = c.diabloPick(nil)

	if best == nil || best.ID != 1 || count != 1 {
		t.Fatalf("fallback: %v %d", best, count)
	}
}

func TestDiabloBuffSkill8(t *testing.T) {
	w := newF5(5, true)
	w.auraState[slot8] = 9
	b := diaBrain(t, Normal, slot8)
	b.Scratch[0] = diaA1
	Tick(f5Plain{w}, b)

	if len(w.log) != 1 || w.log[0] != "cast7" {
		t.Fatalf("buff: %v", w.log)
	}

	// already up: the chosen action runs
	w = newF5(5, true)
	w.auraState[slot8] = 9
	w.states[9] = true
	b = diaBrain(t, Normal, slot8)
	b.Scratch[0] = diaA1
	Tick(f5Plain{w}, b)

	if w.last() != "attack4" {
		t.Fatalf("buff up: %v", w.log)
	}
}

func uberIzualBrain(aip ...int) *Brain {
	b := brainAt(withSkills(profile("UberIzual", aip...), 0, 1, 2))
	def, _ := Lookup("UberIzual")
	b.SetAI(def)

	return b
}

func TestUberIzualPreface(t *testing.T) {
	// buff missing: cast Skill2 first
	w := newF5(5, true)
	w.auraState[slot2] = 4
	Tick(w, uberIzualBrain(100, 0, 0, 0, 10, 3))

	if len(w.log) != 1 || w.log[0] != "cast1" {
		t.Fatalf("buff: %v", w.log)
	}

	// buff up, line blocked (mask 6): Skill3 toward the target
	w = newF5(15, false)
	w.auraState[slot2] = 4
	w.states[4] = true
	w.blocked[6] = true
	Tick(w, uberIzualBrain(100, 0, 0, 0, 10, 3))

	if len(w.log) != 1 || w.log[0] != "cast2" {
		t.Fatalf("blocked line: %v", w.log)
	}

	// otherwise Izual's body: in reach with aip1 100%: strike, no first-tick hook
	w = newF5(3, true)
	b := uberIzualBrain(100, 0, 0, 0, 10, 3)
	Tick(w, b)

	if w.last() != "attack4" || len(w.hooks) != 0 {
		t.Fatalf("body: %v hooks %v", w.log, w.hooks)
	}
}

func TestBossQuestHooks(t *testing.T) {
	for _, c := range []struct {
		ai    string
		ticks int
		want  string
	}{
		{"Izual", 3, "izual"},
		{"Summoner", 3, "summoner"},
		{"Nihlathak", 3, "nihlathak nihlathak nihlathak"},
	} {
		w := newF5(5, true)
		b := brainAt(withSkills(profile(c.ai, 100, 100, 100, 100, 100, 100), 0, 1, 2, 3, 4))
		def, _ := Lookup(c.ai)
		b.SetAI(def)

		for i := 0; i < c.ticks; i++ {
			b.Wake = 0
			Tick(w, b)
		}

		if got := strings.Join(w.hooks, " "); got != c.want {
			t.Errorf("%s: hooks %q want %q", c.ai, got, c.want)
		}
	}
}

func TestDurielRightSkill(t *testing.T) {
	for _, c := range []struct {
		name string
		w    func(*f5World) World
		want int
	}{
		{"no right skill yet", func(w *f5World) World { return f5Right{w, false} }, 2},
		{"right skill present", func(w *f5World) World { return f5Right{w, true} }, 0},
		{"without the host: once", func(w *f5World) World { return w }, 1},
	} {
		base := newF5(5, true)
		b := brainAt(withSkills(profile("Duriel", 4, 100, 100, 100, 0), 0, 1, 2, 3))
		def, _ := Lookup("Duriel")
		b.SetAI(def)

		w := c.w(base)
		Tick(w, b)
		b.Wake = 0
		Tick(w, b)

		if base.auras != c.want {
			t.Errorf("%s: aura started %d times, want %d", c.name, base.auras, c.want)
		}
	}
}

func vulture5(aip ...int) *Brain {
	b := brainAt(profile("Vulture", aip...))
	def, _ := Lookup("Vulture")
	b.SetAI(def)

	return b
}

func TestVultureWaypointMinimumDistance(t *testing.T) {
	// laps 2: the exe doubles the already incremented lap count: min = 2*(2+8) = 20
	// (the earlier port used 2*laps and stopped at 12)
	for id := uint32(1); id < 40; id++ {
		w := newF5(30, false)
		w.target.X = 130
		b := NewBrain(id, 1, Normal, profile("Vulture", 100, 8, 75, 100, 100), testSeed)
		b.X, b.Y = 100, 100
		def, _ := Lookup("Vulture")
		b.SetAI(def)
		b.Airborne = true
		b.Scratch[0] = 2

		Tick(w, b)

		if b.Scratch[0] != 1 {
			t.Fatalf("id %d: laps %d", id, b.Scratch[0])
		}

		if d := unitTileDist(b, b.Scratch[1], b.Scratch[2]); d < 20 {
			t.Fatalf("id %d: waypoint (%d,%d) only %d away", id, b.Scratch[1], b.Scratch[2], d)
		}
	}
}

func TestVultureKeepsAndSnapsWaypoint(t *testing.T) {
	w := newF5(30, false)
	w.target.X = 130
	b := vulture5(100, 8, 75, 100, 100)
	b.Airborne = true
	b.Scratch[0], b.Scratch[1], b.Scratch[2] = 10, 140, 110
	Tick(w, b)

	if w.last() != "walk-to(140,110)" || b.Scratch[0] != 9 || b.Scratch[1] != 140 {
		t.Fatalf("reuse: %v %v", w.log, b.Scratch)
	}

	// with the free-cell search the waypoint moves to the cell found
	w = newF5(30, false)
	w.target.X = 130
	b = vulture5(100, 8, 75, 100, 100)
	b.Airborne = true
	b.Scratch[0], b.Scratch[1], b.Scratch[2] = 10, 140, 110
	Tick(f5Cell{w, Point{141, 111}}, b)

	if w.last() != "walk-to(141,111)" || b.Scratch[1] != 141 || b.Scratch[2] != 111 {
		t.Fatalf("snap: %v %v", w.log, b.Scratch)
	}

	// reached waypoints (within 2 tiles) are replaced
	w = newF5(30, false)
	w.target.X = 130
	b = vulture5(100, 8, 75, 100, 100)
	b.Airborne = true
	b.Scratch[0], b.Scratch[1], b.Scratch[2] = 10, 101, 101
	Tick(w, b)

	if b.Scratch[1] == 101 && b.Scratch[2] == 101 {
		t.Fatal("a waypoint within 2 tiles must be replaced")
	}
}

func TestVultureRoomMismatch(t *testing.T) {
	// on the ground, hero in another room and far: walk toward him, never take off
	w := newF5(30, false)
	w.target.X = 130
	b := vulture5(100, 8, 75, 100, 100)
	Tick(f5Room{w, false}, b)

	if b.Airborne || !strings.HasPrefix(w.last(), "walk-to(") {
		t.Fatalf("ground: %v airborne=%v", w.log, b.Airborne)
	}

	// airborne with laps left: lands, settles toward the spot, laps -1, sleep 12
	w = newF5(30, false)
	w.target.X = 130
	b = vulture5(100, 8, 75, 100, 100)
	b.Airborne = true
	b.Scratch[0] = 5
	Tick(f5Room{w, false}, b)

	if b.Airborne || b.Scratch[0] != -1 || b.Wake != 12 {
		t.Fatalf("landing: airborne=%v S0=%d wake=%d", b.Airborne, b.Scratch[0], b.Wake)
	}
}

func bloodRaven5(p *Profile) *Brain {
	b := brainAt(p)
	def, _ := Lookup("BloodRaven")
	b.SetAI(def)

	return b
}

func TestBloodRavenSidestepScheme(t *testing.T) {
	p := profile("BloodRaven")
	w := newF5(30, false)
	w.target = Target{ID: 1, X: 130, Y: 100, Size: 1, IsPlayer: true}
	b := bloodRaven5(p)
	s := shadow(b)

	// distance 30 > 20 with the target near the anchor: sidestep of max(30/2, 12) = 15
	s1 := s.Step()
	r := int(s.Roll(15))
	sx, sy := s.Step(), s.Step()
	dx, dy := r, 15

	if s1&1 != 0 {
		dx, dy = 15, r
	}

	if sx&1 != 0 {
		dx = -dx
	}

	if sy&1 != 0 {
		dy = -dy
	}

	Tick(w, b)

	if want := fmt.Sprintf("run-to(%d,%d)", 130+dx, 100+dy); w.last() != want {
		t.Fatalf("sidestep %v, want %s", w.log, want)
	}
}

func TestBloodRavenSpotRolls(t *testing.T) {
	p := withSkills(profile("BloodRaven"), 0)
	w := newF5(15, false)
	w.target = Target{ID: 1, X: 115, Y: 100, Size: 1, IsPlayer: true}
	rec := &recordingWorld{fakeWorld: w.fake2.fakeWorld}
	b := bloodRaven5(p)
	b.Scratch[brCharge] = 100
	s := shadow(b)

	s.Roll(100) // the shot roll
	r := int(s.Roll(15)) + 5

	var ox, oy int
	if s.Step()&1 == 0 {
		ox, oy = int(s.Roll(int32(r))), r
	} else {
		ox, oy = r, int(s.Roll(int32(r)))
	}

	if s.Step()&1 != 0 {
		ox = -ox
	}

	if s.Step()&1 != 0 {
		oy = -oy
	}

	Tick(rec, b)

	if len(rec.casts) != 1 || rec.casts[0].X != 115+ox || rec.casts[0].Y != 100+oy {
		t.Fatalf("casts %v, want (%d,%d)", rec.casts, 115+ox, 100+oy)
	}
}

func TestBloodRavenLeashUsesTheTargetsDistance(t *testing.T) {
	// the monster is 40 from the anchor (within 50) but the hero is 80 away
	// from it: the exe runs home
	w := newF5(40, false)
	w.target = Target{ID: 1, X: 140, Y: 100, Size: 1, IsPlayer: true}
	p := profile("BloodRaven")
	p.AIDist = 55
	b := bloodRaven5(p)
	b.AppendCommand(Command{Type: CmdAnchor, X: 60, Y: 100})
	Tick(w, b)

	if w.last() != "run-to(60,100)" || b.Scratch[brFleeHome] != 1 {
		t.Fatalf("%v S2=%d", w.log, b.Scratch[brFleeHome])
	}
}

func TestNihlathakHelpRollWandersWhenNothingToHelp(t *testing.T) {
	// aip4 = 100: the helper branch is taken; no wounded ally and no Skill5
	w := newF5(5, false)
	b := brainAt(withSkills(profile("Nihlathak", 0, 0, 0, 100, 0), 1))
	def, _ := Lookup("Nihlathak")
	b.SetAI(def)
	Tick(w, b)

	if !strings.HasPrefix(w.last(), "walk-to(") || len(w.log) != 1 {
		t.Fatalf("expected the wander of the exe, got %v", w.log)
	}
}
