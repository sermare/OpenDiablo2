package d2monster

import (
	"strings"
	"testing"
)

type spWorld struct {
	*fakeWorld
	holeSpawns, roomSpawns int
	holeFail               bool
	class                  int
	classOK                bool
	scan                   *Target
	scanQ                  []FBXScanQuery
	missile                int
	cellsBlock             bool
	raw                    []int
}

func newSP(dist int, inRange bool) *spWorld {
	return &spWorld{fakeWorld: newFake(dist, inRange), class: 453, classOK: true}
}

func (w *spWorld) SpawnHoleMinion(*Brain) bool {
	if w.holeFail {
		return false
	}

	w.holeSpawns++

	return true
}
func (w *spWorld) SpawnRoomUnit(*Brain) bool { w.roomSpawns++; return true }
func (w *spWorld) PickGenericSpawnClass(*Brain) (int, bool) {
	return w.class, w.classOK
}
func (w *spWorld) SpawnerCellsFree(*Brain) bool { return !w.cellsBlock }
func (w *spWorld) MissileS1Range(*Brain) int    { return w.missile }
func (w *spWorld) FBXScan(_ *Brain, q FBXScanQuery) FBXScanResult {
	w.scanQ = append(w.scanQ, q)
	if w.scan == nil {
		return FBXScanResult{}
	}

	return FBXScanResult{Found: true, T: *w.scan}
}

func (w *spWorld) CastSkillID(_ *Brain, id int, _ Mode, _ *Target, _ *Point) bool {
	w.raw = append(w.raw, id)
	w.log = append(w.log, "rawcast")

	return true
}

func spBrain(ai string, skills []int, aip ...int) *Brain {
	b := NewBrain(7, 1, Normal, withSkills(profile(ai, aip...), skills...), testSeed)
	b.X, b.Y = 100, 100
	def, _ := Lookup(ai)
	b.SetAI(def)

	return b
}

func TestSpawnersRegistered(t *testing.T) {
	for _, n := range FaithfulSpawners() {
		d, ok := Lookup(n)
		if !ok || !d.Implemented && d.Think == nil {
			t.Fatalf("%s not registered", n)
		}

		if m, ok := AITargetMode(n); !ok || m != d.TargetMode {
			t.Errorf("%s: target mode %d, exe %d", n, d.TargetMode, m)
		}

		if _, stand := genericAIs[n]; stand {
			t.Errorf("%s is still a stand-in", n)
		}
	}
}

func TestEvilHoleModesAndCounter(t *testing.T) {
	// aip1 = 2 minions, aip2 = 30 frame gap.
	w := newSP(9, false)
	b := spBrain("EvilHole", nil, 2, 30)

	// NU, target far: sleep 5, no action.
	Tick(w, b)

	if len(w.log) != 0 || b.Wake != 5 {
		t.Fatalf("far: log %v wake %d", w.log, b.Wake)
	}

	if b.Scratch[0] != 30 || b.Scratch[1] != 2 {
		t.Fatalf("init scratch %v", b.Scratch)
	}

	// NU, target near: S3 requested, wake 20.
	w.frame, w.dist = 10, 4
	w.target.X = 104
	Tick(w, b)

	if w.last() != "attack10" || b.Wake != 30 {
		t.Fatalf("near: log %v wake %d", w.log, b.Wake)
	}

	// S3 -> S4.
	b.Mode, w.frame = ModeSkill3, 30
	Tick(w, b)

	if w.last() != "attack11" {
		t.Fatalf("S3: %v", w.log)
	}

	// S4: scratch0 = 30, frame 30: not yet (strict).
	b.Mode = ModeSkill4
	w.frame = 30
	b.Wake = 0
	Tick(w, b)

	if w.holeSpawns != 0 || b.Wake != 60 {
		t.Fatalf("S4 early: spawns %d wake %d", w.holeSpawns, b.Wake)
	}

	// frame 31 > 30: spawn, timer re-armed to 61.
	w.frame, b.Wake = 31, 0
	Tick(w, b)

	if w.holeSpawns != 1 || b.Scratch[1] != 1 || b.Scratch[0] != 61 {
		t.Fatalf("spawn1: n=%d scratch=%v", w.holeSpawns, b.Scratch)
	}

	// A failed spawn does not count but still re-arms the timer.
	w.holeFail = true
	w.frame, b.Wake = 62, 0
	Tick(w, b)

	if b.Scratch[1] != 1 || b.Scratch[0] != 92 {
		t.Fatalf("failed spawn: scratch=%v", b.Scratch)
	}

	w.holeFail = false
	w.frame, b.Wake = 93, 0
	Tick(w, b)

	if w.holeSpawns != 2 || b.Scratch[1] != 0 {
		t.Fatalf("spawn2: n=%d scratch=%v", w.holeSpawns, b.Scratch)
	}

	// Counter exhausted in S4: death mode.
	w.frame, b.Wake = 130, 0
	Tick(w, b)

	if w.last() != "attack0" {
		t.Fatalf("done: %v", w.log)
	}

	// Any other mode (here walk) also ends in the death mode.
	w2 := newSP(3, false)
	b2 := spBrain("EvilHole", nil, 2, 30)
	b2.Mode = ModeWalk
	Tick(w2, b2)

	if w2.last() != "attack0" {
		t.Fatalf("other mode: %v", w2.log)
	}
}

func TestHighPriestOpening(t *testing.T) {
	// In range, aip1 = 100: the roll is always below, so attack A1, engaged.
	w := newSP(2, true)
	b := spBrain("HighPriest", nil, 100, 0, 0, 0, 0, 0, 0, 0)
	Tick(w, b)

	if w.last() != "attack4" || b.Scratch[0] != 1 {
		t.Fatalf("open attack: %v %v", w.log, b.Scratch)
	}

	// aip1 = 0: always walks away, stays unengaged.
	w = newSP(2, true)
	b = spBrain("HighPriest", nil, 0)
	Tick(w, b)

	if !strings.HasPrefix(w.last(), "walk-to") || b.Scratch[0] != 0 {
		t.Fatalf("open retreat: %v %v", w.log, b.Scratch)
	}
}

func TestHighPriestHeal(t *testing.T) {
	ally := &Target{ID: 55, X: 110, Y: 100, Size: 1}
	w := newSP(8, false)
	w.scan = ally
	// aip2 heal 100%, aip3 gap 40.
	b := spBrain("HighPriest", []int{1}, 0, 100, 40)
	w.frame = 5
	Tick(w, b)

	if w.last() != "cast1" || b.Scratch[1] != 45 {
		t.Fatalf("heal: %v scratch %v", w.log, b.Scratch)
	}

	if len(w.scanQ) != 1 || w.scanQ[0].Kind != FBXScanWoundedAlly || w.scanQ[0].Radius2 != 2500 ||
		w.scanQ[0].LifeBelow != 75 {
		t.Fatalf("scan query %+v", w.scanQ)
	}

	// Still on cooldown (S1 = 45 is not below frame 45): no scan, no heal.
	w.log, w.scanQ = nil, nil
	w.frame, b.Wake = 45, 0
	Tick(w, b)

	if len(w.scanQ) != 0 || w.last() == "cast1" {
		t.Fatalf("cooldown: %v %v", w.log, w.scanQ)
	}
}

func TestHighPriestSkillAtOffset(t *testing.T) {
	w := newSP(8, false)
	// Skill1 used, aip4 100, aip8 15.
	b := spBrain("HighPriest", []int{0}, 0, 0, 0, 100, 0, 0, 0, 15)
	w.frame = 7
	Tick(w, b)

	if w.last() != "cast0" || b.Scratch[1] != 107 {
		t.Fatalf("skill1: %v scratch %v", w.log, b.Scratch)
	}

	// Too far for aip8: no cast.
	w = newSP(20, false)
	b = spBrain("HighPriest", []int{0}, 0, 0, 0, 100, 0, 0, 0, 15)
	Tick(w, b)

	if w.last() == "cast0" {
		t.Fatalf("far cast: %v", w.log)
	}
}

func TestHighPriestMissileAndTail(t *testing.T) {
	w := newSP(6, false)
	w.missile = 12
	b := spBrain("HighPriest", nil, 0, 0, 0, 0, 100)
	Tick(w, b)

	if w.last() != "attack8" {
		t.Fatalf("missile S1: %v", w.log)
	}

	// Beyond the range - 2: the missile branch is skipped.
	w = newSP(10, false)
	w.missile = 12
	b = spBrain("HighPriest", nil, 0, 0, 0, 0, 100)
	Tick(w, b)

	if w.last() == "attack8" && b.Scratch[0] == 0 {
		t.Fatalf("missile out of range fired: %v", w.log)
	}

	// Engaged, near, aip7 above any roll: S1 attack.
	w = newSP(3, false)
	b = spBrain("HighPriest", nil, 0, 0, 0, 0, 0, 0, 101, 0)
	b.Scratch[0] = 1
	Tick(w, b)

	if w.last() != "attack8" {
		t.Fatalf("near S1: %v", w.log)
	}

	// Engaged, far, retreat 100%: back to the opening phase after 10 frames.
	w = newSP(9, false)
	b = spBrain("HighPriest", nil, 0, 0, 0, 0, 0, 100, 0, 0)
	b.Scratch[0] = 1
	Tick(w, b)

	if b.Scratch[0] != 0 || b.Wake != 10 || len(w.log) != 0 {
		t.Fatalf("retreat: %v scratch %v wake %d", w.log, b.Scratch, b.Wake)
	}

	// Engaged, in range, aip7 0 and retreat 100: walk away and reopen.
	w = newSP(2, true)
	b = spBrain("HighPriest", nil, 0, 0, 0, 0, 0, 100, 0, 0)
	b.Scratch[0] = 1
	Tick(w, b)

	if !strings.HasPrefix(w.last(), "walk-to") || b.Scratch[0] != 0 {
		t.Fatalf("in-range retreat: %v %v", w.log, b.Scratch)
	}

	// Engaged, in range, aip7 101: S1 attack.
	w = newSP(2, true)
	b = spBrain("HighPriest", nil, 0, 0, 0, 0, 0, 0, 101, 0)
	b.Scratch[0] = 1
	Tick(w, b)

	if w.last() != "attack8" {
		t.Fatalf("in-range S1: %v", w.log)
	}
}

func TestGenericSpawner(t *testing.T) {
	// aip1 gap 10, aip3 limit 2; Skill1 used (the raw cast falls back to it
	// only without RawCaster, here the fake has none).
	w := newSP(10, false)
	b := spBrain("GenericSpawner", []int{0}, 10, 0, 2)
	w.frame = 3
	Tick(w, b)

	// Pre-hook: last lay = 3, count 0, class chosen from the host.
	if b.SpawnClass != 453 || b.Scratch[0] != 3 || b.Scratch[1] != 0 {
		t.Fatalf("pre: class %d scratch %v", b.SpawnClass, b.Scratch)
	}

	// gap 0 < 10: nothing laid, sleep 20.
	if b.Wake != 23 || len(w.log) != 0 {
		t.Fatalf("early: wake %d log %v", b.Wake, w.log)
	}

	// Gap reached: lay one.
	w.frame, b.Wake = 13, 0
	Tick(w, b)

	if w.last() != "rawcast" || len(w.raw) != 1 || w.raw[0] != 167 || b.Scratch[1] != 1 || b.Scratch[0] != 13 {
		t.Fatalf("lay1: %v %v", w.log, b.Scratch)
	}

	// Blocked cells: the timer moves, the count does not.
	w.cellsBlock = true
	w.frame, b.Wake = 40, 0
	w.log = nil
	Tick(w, b)

	if len(w.log) != 0 || b.Scratch[1] != 1 || b.Scratch[0] != 40 || b.Wake != 60 {
		t.Fatalf("blocked: %v %v wake %d", w.log, b.Scratch, b.Wake)
	}

	// Second unit, then the limit ends the spawner.
	w.cellsBlock = false
	w.frame, b.Wake = 55, 0
	Tick(w, b)

	if b.Scratch[1] != 2 {
		t.Fatalf("lay2: %v", b.Scratch)
	}

	w.frame, b.Wake = 80, 0
	Tick(w, b)

	if b.Mode != ModeDying {
		t.Fatalf("limit: %v", w.log)
	}
}

func TestGenericSpawnerOutOfRangeAndNoClass(t *testing.T) {
	w := newSP(21, false)
	b := spBrain("GenericSpawner", []int{0}, 0, 0, 5)
	Tick(w, b)

	if len(w.log) != 0 || b.Wake != 20 {
		t.Fatalf("dist 21: %v wake %d", w.log, b.Wake)
	}

	w = newSP(5, false)
	w.classOK = false
	b = spBrain("GenericSpawner", []int{0}, 0, 0, 5)
	Tick(w, b)

	if b.Mode != ModeDying {
		t.Fatalf("no class: %v", w.log)
	}
}

func TestInvisoSpawner(t *testing.T) {
	// aip1 = 2 units, aip2 range 30, aip3 gap 50.
	w := newSP(10, false)
	b := spBrain("InvisoSpawner", nil, 2, 30, 50)
	Tick(w, b)

	if w.roomSpawns != 1 || b.Scratch[1] != 1 || b.Scratch[0] != 50 || b.Wake != 15 {
		t.Fatalf("first: n=%d scratch=%v wake=%d", w.roomSpawns, b.Scratch, b.Wake)
	}

	// Before the gap passes nothing happens (and the counter is not reset).
	w.frame, b.Wake = 15, 0
	Tick(w, b)

	if w.roomSpawns != 1 || b.Scratch[1] != 1 {
		t.Fatalf("gap: n=%d scratch=%v", w.roomSpawns, b.Scratch)
	}

	w.frame, b.Wake = 50, 0
	Tick(w, b)

	if w.roomSpawns != 2 || b.Scratch[1] != 0 || b.Scratch[0] != 100 {
		t.Fatalf("second: n=%d scratch=%v", w.roomSpawns, b.Scratch)
	}

	// Out of units: death mode, no sleep.
	w.frame, b.Wake = 100, 0
	Tick(w, b)

	if b.Mode != ModeDying {
		t.Fatalf("done: %v", w.log)
	}

	// Beyond aip2 nothing spawns and the unit waits 15.
	w = newSP(31, false)
	b = spBrain("InvisoSpawner", nil, 2, 30, 50)
	Tick(w, b)

	if w.roomSpawns != 0 || b.Wake != 15 {
		t.Fatalf("far: n=%d wake=%d", w.roomSpawns, b.Wake)
	}
}
