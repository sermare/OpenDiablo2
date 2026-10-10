package d2skill

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2missile"
)

// Bone Wall's table numbers (skills.txt): Param1 25 (% life per level), Param2
// 600 (duration), Param3 8, calc1 = par1*(lvl-1), calc2 = par34 (8, "# of
// walls - 1"), srvmissilea bonewallmaker.
func boneWallFixture(withMaker bool) *fixture {
	f := newFixture(map[string]int{"Bone Wall": 5})
	f.addSkill(row{"skill": "Bone Wall", "Id": "78", "srvdofunc": "60", "srvmissilea": "bonewallmaker", "summon": "bonewall",
		"pettype": "none", "summode": "S1", "Param1": "25", "Param2": "600", "Param3": "8",
		"calc1": "par1 * (lvl-1)", "calc2": "par3", "minmana": "1", "manashift": "8", "mana": "17"})

	if withMaker {
		f.addMissile(&d2missile.Spec{ID: 207, Name: "bonewallmaker", SrvDoFunc: 13, Vel: 12, MaxVel: 12, Range: 7, LevRange: 2,
			CollideType: 8, LastCollide: true, Size: 1, CanSlow: true})
	}

	return f
}

func TestBoneWallOrderSplitsIntoLeaderAndMakers(t *testing.T) {
	cases := []struct {
		name       string
		withMaker  bool
		hx, hy     int // caster
		tx, ty     int // target
		wantCount  int
		wantPer    int
		wantDirs   [2][2]int
		wantMakers bool
	}{
		{"cast east", true, 0, 0, 10, 0, 1, 4, [2][2]int{{0, 16}, {0, -16}}, true},
		{"cast north-east", true, 0, 0, 10, 10, 1, 4, [2][2]int{{-16, 16}, {16, -16}}, true},
		{"cast south", true, 5, 5, 5, 12, 1, 4, [2][2]int{{-16, 0}, {16, 0}}, true},
		{"no maker missile: old line of Param3", false, 0, 0, 10, 0, 8, 0, [2][2]int{}, false},
	}

	for _, c := range cases {
		f := boneWallFixture(c.withMaker)
		f.u.x, f.u.y = c.hx, c.hy

		_, do := f.cast("Bone Wall", c.tx, c.ty)
		if !do.OK {
			t.Fatalf("%s: %+v", c.name, do)
		}

		o := effectOf(t, do, "summon").Summon
		// level 5: 25 * 4 percent extra life, 600 frames, summon key from the table
		if o.Key != "bonewall" || o.Kind != "wall" || o.Frames != 600 || o.HPPct != 100 || o.HPFlat != 0 || o.X != c.tx || o.Y != c.ty {
			t.Errorf("%s: order %+v", c.name, o)
		}

		if o.Count != c.wantCount || (o.Makers != nil) != c.wantMakers {
			t.Errorf("%s: count %d makers %v, want %d %v", c.name, o.Count, o.Makers, c.wantCount, c.wantMakers)
			continue
		}

		if c.wantMakers && (o.Makers.PerMaker != c.wantPer || o.Makers.Dirs != c.wantDirs || o.Makers.Missile != "bonewallmaker" ||
			o.Makers.FromX != c.tx || o.Makers.FromY != c.ty) {
			t.Errorf("%s: makers %+v", c.name, o.Makers)
		}
	}
}

// CastWallMaker puts a maker on the first wall; walking its range it summons
// exactly the walls it carries, each marked with the leader.
func TestCastWallMakerSummonsItsWalls(t *testing.T) {
	for _, walls := range []int{1, 4} {
		f := boneWallFixture(true)
		leader := &testTarget{id: "leader", alive: true, x: 20, y: 20}

		m := f.p.CastWallMaker(f.u, f.id("Bone Wall"), "bonewallmaker", 20, 20, 0, 16, walls, leader)
		if m == nil {
			t.Fatal("no maker missile")
		}

		if m.Data2C != uint32(walls) || m.Mark != d2missile.Target(leader) {
			t.Errorf("walls %d: data %d mark %v", walls, m.Data2C, m.Mark)
		}

		got := 0

		for i := 0; i < 60; i++ {
			f.w.frame++
			f.sim.Step()
		}

		for _, e := range f.evs {
			if e.Kind == d2missile.EventSummon && e.Target.ID() == "leader" {
				got++
			}
		}

		if got != walls || !m.Dead() {
			t.Errorf("walls %d: %d summon events, dead=%v", walls, got, m.Dead())
		}
	}
}
