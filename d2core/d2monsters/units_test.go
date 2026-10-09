package d2monsters

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2monster"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2path"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2records"
)

func TestFootprintsBlockUnitsNotCorpses(t *testing.T) {
	f := newFootprints(d2path.NewCellGrid(0, 0, 20, 20))

	f.Move(1, 5, 5, d2path.FlagMonster)
	f.Move(2, 6, 5, d2path.FlagMonster)
	f.Move(playerKeyBase+1, 7, 5, d2path.FlagPlayer)

	if !f.BlockedFor(2, 5, 5) || f.BlockedFor(1, 5, 5) {
		t.Error("a monster blocks others but not itself")
	}

	if !f.BlockedFor(1, 7, 5) {
		t.Error("a hero blocks monsters")
	}

	if f.Flags(5, 5)&d2path.FlagMonster == 0 || f.Flags(7, 5)&d2path.FlagPlayer == 0 {
		t.Errorf("flags 0x100/0x80 expected, got %x %x", f.Flags(5, 5), f.Flags(7, 5))
	}

	// moving frees the old cell
	f.Move(1, 5, 6, d2path.FlagMonster)

	if f.BlockedFor(2, 5, 5) || f.Flags(5, 5) != 0 {
		t.Error("old cell must be free")
	}

	// death: the monster footprint becomes a corpse (0x8000) that does not block
	f.Move(1, 5, 6, d2path.FlagCorpse)

	if f.BlockedFor(2, 5, 6) || f.Flags(5, 6) != d2path.FlagCorpse {
		t.Errorf("corpse must not block: %x", f.Flags(5, 6))
	}

	f.Remove(1)

	if f.Flags(5, 6) != 0 || f.Count() != 2 {
		t.Errorf("corpse removal: flags=%x count=%d", f.Flags(5, 6), f.Count())
	}
}

func TestFootprintsSharedCell(t *testing.T) {
	f := newFootprints(d2path.NewCellGrid(0, 0, 10, 10))
	f.Move(1, 3, 3, d2path.FlagMonster)
	f.Move(2, 3, 3, d2path.FlagCorpse)
	f.Remove(1)

	if f.Flags(3, 3) != d2path.FlagCorpse {
		t.Errorf("only the corpse bit should remain: %x", f.Flags(3, 3))
	}
}

func TestPathsAvoidUnits(t *testing.T) {
	g := d2path.NewCellGrid(0, 0, 30, 5)
	f := newFootprints(g)
	// a wall of monsters across the corridor, with a gap at y=4
	for y := 0; y < 4; y++ {
		f.Move(uint32(10+y), 15, y, d2path.FlagMonster)
	}

	from, to := d2path.Point{X: 2, Y: 1}, d2path.Point{X: 27, Y: 1}
	mask := d2path.MaskMonster | d2path.MaskUnits

	route, ok := d2path.FindPath(f.ignoring(from, to), mask, from, to)
	if !ok {
		t.Fatal("a path around the group exists")
	}

	for _, n := range route.Nodes {
		if n.X == 15 && n.Y < 4 {
			t.Fatalf("route walks through a monster at %v", n)
		}
	}

	// the target's own cell never blocks its own approach
	f.Move(99, 27, 1, d2path.FlagPlayer)

	if _, ok := d2path.FindPath(f.ignoring(from, to), mask, from, to); !ok {
		t.Error("the destination cell (where the hero stands) must not block the path")
	}
}

func TestBoltHitsUnitInItsWay(t *testing.T) {
	l := newBoltLauncher(d2path.NewCellGrid(0, 0, 60, 20))

	var hitAt *d2path.Point

	hit := false
	done := 0

	ok := l.Launch(Shot{
		From: d2path.Point{X: 2, Y: 10}, To: d2path.Point{X: 40, Y: 10}, Velocity: 2,
		Collide: func(x, y int) bool { return x == 20 && y == 10 },
		Impact: func(x, y int, h bool) {
			hitAt = &d2path.Point{X: x, Y: y}
			hit = h
			done++
		},
	})
	if !ok {
		t.Fatal("launch failed")
	}

	frames := 0

	for l.InFlight() > 0 && frames < 100 {
		l.Step()

		frames++
	}

	if done != 1 || !hit || hitAt == nil || hitAt.X != 20 {
		t.Fatalf("expected one hit at x=20, got done=%d hit=%v at=%v", done, hit, hitAt)
	}

	// 18 subtiles at 2 per frame: it takes about 9 frames, not one
	if frames < 8 || frames > 11 {
		t.Errorf("flight took %d frames, a projectile takes time", frames)
	}
}

func TestBoltStopsAtWallAndRange(t *testing.T) {
	g := d2path.NewCellGrid(0, 0, 60, 20)
	g.Set(25, 10, d2path.FlagWall)
	l := newBoltLauncher(g)

	var res []string

	rec := func(name string) func(x, y int, h bool) {
		return func(x, y int, h bool) {
			res = append(res, name)
			if h {
				res = append(res, "hit")
			}

			if name == "wall" && x != 25 {
				t.Errorf("wall impact at %d", x)
			}
		}
	}

	l.Launch(Shot{From: d2path.Point{X: 2, Y: 10}, To: d2path.Point{X: 40, Y: 10}, Impact: rec("wall")})
	l.Launch(Shot{From: d2path.Point{X: 2, Y: 12}, To: d2path.Point{X: 40, Y: 12}, Range: 10, Impact: rec("range")})

	for i := 0; i < 100 && l.InFlight() > 0; i++ {
		l.Step()
	}

	if len(res) != 2 || res[0] != "range" || res[1] != "wall" {
		t.Errorf("impacts (range expires first, wall second): %v", res)
	}

	if l.Launch(Shot{From: d2path.Point{X: 1, Y: 1}, To: d2path.Point{X: 1, Y: 1}, Impact: func(int, int, bool) {}}) {
		t.Error("a zero length shot cannot fly")
	}
}

func TestBoltMissesMovedTarget(t *testing.T) {
	l := newBoltLauncher(d2path.NewCellGrid(0, 0, 60, 20))
	hero := d2path.Point{X: 30, Y: 10}
	missed := false

	l.Launch(Shot{
		From: d2path.Point{X: 2, Y: 10}, To: hero, Velocity: 3,
		Collide: func(x, y int) bool { return x == hero.X && y == hero.Y },
		Impact:  func(_, _ int, h bool) { missed = !h },
	})

	for i := 0; i < 5; i++ {
		l.Step()
	}

	hero.Y = 15 // the hero stepped aside while the arrow flew

	for i := 0; i < 100 && l.InFlight() > 0; i++ {
		l.Step()
	}

	if !missed {
		t.Error("the arrow aimed at the old position should miss")
	}
}

func TestAttackReachPerMode(t *testing.T) {
	melee := &d2records.MonStatRecord{}
	bow := &d2records.MonStatRecord{MissileA1: "skarrow", IsRanged: true}
	mixed := &d2records.MonStatRecord{MissileA2: "bolt"} // A1 melee, A2 a spell

	for _, c := range []struct {
		name   string
		st     *d2records.MonStatRecord
		mode   d2monster.Mode
		ranged bool
	}{
		{"melee A1", melee, d2monster.ModeAttack1, false},
		{"melee skill", melee, d2monster.ModeSkill1, false},
		{"bow A1", bow, d2monster.ModeAttack1, true},
		{"rangedtype without missile, A1", &d2records.MonStatRecord{IsRanged: true}, d2monster.ModeAttack1, true},
		{"mixed A1", mixed, d2monster.ModeAttack1, false},
		{"mixed A2", mixed, d2monster.ModeAttack2, true},
	} {
		if got := attackIsRanged(c.st, c.mode); got != c.ranged {
			t.Errorf("%s: ranged=%v want %v", c.name, got, c.ranged)
		}
	}

	if attackReach(false) != 7 || attackReach(true) <= attackReach(false) {
		t.Errorf("reach melee=%d ranged=%d", attackReach(false), attackReach(true))
	}
}
