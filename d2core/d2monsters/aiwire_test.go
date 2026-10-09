package d2monsters

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2monster"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2path"
)

func TestPickTeleportDest(t *testing.T) {
	r := rect{1, 1, 18, 18}
	open := d2path.NewCellGrid(0, 0, 20, 20)

	for seed := uint32(1); seed < 50; seed++ {
		p, ok := pickTeleportDest(open, d2rand.New(seed), r, 1)
		if !ok || p.X < 1 || p.X > 18 || p.Y < 1 || p.Y > 18 {
			t.Fatalf("seed %d: %v %v", seed, p, ok)
		}

		q, _ := pickTeleportDest(open, d2rand.New(seed), r, 1)
		if p != q {
			t.Fatal("same seed must give the same destination")
		}
	}

	// half the map walled: every success lands on a free cell
	half := d2path.NewCellGrid(0, 0, 20, 20)
	for y := 0; y < 20; y++ {
		for x := 0; x < 10; x++ {
			half.Set(x, y, d2path.FlagWalk)
		}
	}

	for seed := uint32(1); seed < 50; seed++ {
		if p, ok := pickTeleportDest(half, d2rand.New(seed), r, 1); ok && p.X < 10 {
			t.Fatalf("landed in the wall: %v", p)
		}
	}

	full := d2path.NewCellGrid(0, 0, 20, 20)
	for i := range full.Cells {
		full.Cells[i] = d2path.FlagWalk
	}

	if _, ok := pickTeleportDest(full, d2rand.New(1), r, 1); ok {
		t.Error("blocked map must fail")
	}

	if _, ok := pickTeleportDest(open, d2rand.New(1), rect{5, 5, 4, 4}, 1); ok {
		t.Error("empty rect must fail")
	}
}

func TestTargetReachable(t *testing.T) {
	g := d2path.NewCellGrid(0, 0, 30, 30)
	from := d2monster.Point{X: 20, Y: 20}
	tg := d2monster.Target{X: 10, Y: 10}

	if !targetReachable(g, from, tg) {
		t.Error("open ground is reachable")
	}

	for _, k := range []int{0, 2, 4} {
		g.Set(10+k, 10+k, d2path.FlagWall)
	}

	if targetReachable(g, from, tg) {
		t.Error("all three probes blocked must be unreachable")
	}

	g2 := d2path.NewCellGrid(0, 0, 30, 30)
	for _, k := range []int{0, 2, 4} {
		g2.Set(10+k, 10+k, d2path.FlagDoor)
	}

	if targetReachable(g2, from, tg) {
		t.Error("closed doors block the probes")
	}

	g3 := d2path.NewCellGrid(0, 0, 30, 30)
	g3.Set(10, 10, d2path.FlagMonster)

	if !targetReachable(g3, from, tg) {
		t.Error("a unit footprint is not in mask 0x805")
	}
}

func TestOwnTileFlagged(t *testing.T) {
	d := &Director{fp: newFootprints(d2path.NewCellGrid(0, 0, 10, 10))}
	b := &d2monster.Brain{X: 3, Y: 3}

	if d.OwnTileFlagged(b) {
		t.Error("static map carries no 0x40")
	}

	d.fp.grid.Set(3, 3, d2path.FlagMissile)

	if !d.OwnTileFlagged(b) {
		t.Error("0x40 on the own tile must flag")
	}
}

func TestGatesOffByDefault(t *testing.T) {
	d := &Director{}
	b := &d2monster.Brain{}

	if d.LevelThreat(b) != 0 {
		t.Error("level threat must be off without Options.LevelThreat")
	}

	d.opt.LevelThreat = func(l int) int { return l + 10 }
	b.LevelID = 2

	if d.LevelThreat(b) != 12 {
		t.Error("host threat not used")
	}

	called := 0
	d.opt.OnOverlay = func(_ uint32, o int) { called = o }
	d.ShowOverlay(b, 0x96)

	if called != 0x96 {
		t.Error("overlay hook not called")
	}

	d.opt.IsTownLevel = func(int) bool { return true }

	if _, ok := d.TeleportDest(b); ok {
		t.Error("no teleport in town")
	}
}
