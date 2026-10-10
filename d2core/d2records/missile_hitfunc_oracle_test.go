package d2records

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2missile"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2path"
)

// hitWorld has one monster at a subtile on the missile's way.
type hitWorld struct {
	emptyWorld
	at [2]int
}

func (w *hitWorld) Targets(x, y int) []d2missile.Target {
	if x == w.at[0] && y == w.at[1] {
		return []d2missile.Target{oracleTarget{}}
	}

	return nil
}

type oracleTarget struct{}

func (oracleTarget) ID() string       { return "m" }
func (oracleTarget) IsPlayer() bool   { return false }
func (oracleTarget) Alive() bool      { return true }
func (oracleTarget) Level() int       { return 1 }
func (oracleTarget) Defense(bool) int { return 0 }

func countKind(evs []d2missile.Event, k d2missile.EventKind) (n int) {
	for _, e := range evs {
		if e.Kind == k {
			n++
		}
	}

	return n
}

// TestRealMissileHitFunctions pins the verified hit function semantics against
// the real table: the hit function's return value replaces the result bits of
// ProcessHitOrExpire (0x5aba10).
func TestRealMissileHitFunctions(t *testing.T) {
	tbl := loadRealRecords(t).MissileTable()

	run := func(w d2missile.World, name string, p d2missile.CreateParams, frames int) (evs []d2missile.Event) {
		p.Spec = tbl.ByName(name)
		p.Level = 1
		s := d2missile.NewSim(w, tbl)
		s.OnEvent = func(e d2missile.Event) { evs = append(evs, e) }

		if _, err := s.Create(p); err != nil {
			t.Fatal(err)
		}

		for i := 0; i < frames; i++ {
			s.Step()
		}

		return evs
	}

	grid := func() emptyWorld { return emptyWorld{g: d2path.NewCellGrid(-300, -300, 600, 600)} }

	// Fireball: hit func 1 returns 1 -> no direct damage, one area event of
	// radius sHitPar1 = 4 subtiles.
	hw := &hitWorld{emptyWorld: grid(), at: [2]int{20, 0}}
	evs := run(hw, "fireball", d2missile.CreateParams{DestX: 90}, 60)

	if countKind(evs, d2missile.EventHit) != 0 || countKind(evs, d2missile.EventArea) != 1 {
		t.Errorf("fireball on a target: %d hits, %d areas", countKind(evs, d2missile.EventHit), countKind(evs, d2missile.EventArea))
	}

	for _, e := range evs {
		if e.Kind == d2missile.EventArea && e.Radius != 4 {
			t.Errorf("fireball radius %d", e.Radius)
		}
	}

	// ... also where it runs out (the hit function runs on expiry)
	g := grid()
	evs = run(&g, "fireball", d2missile.CreateParams{DestX: 90}, 60)

	if countKind(evs, d2missile.EventExpire) != 1 || countKind(evs, d2missile.EventArea) != 1 {
		t.Errorf("fireball expiry: expire %d area %d", countKind(evs, d2missile.EventExpire), countKind(evs, d2missile.EventArea))
	}

	// Firebolt has no hit function: the direct hit deals damage, nothing else.
	evs = run(hw, "firebolt", d2missile.CreateParams{DestX: 90}, 60)

	if countKind(evs, d2missile.EventHit) != 1 || countKind(evs, d2missile.EventArea) != 0 {
		t.Errorf("firebolt: %d hits, %d areas", countKind(evs, d2missile.EventHit), countKind(evs, d2missile.EventArea))
	}

	// Meteor center (hit func 14, collide type 0): where it ends it drops
	// meteorfire at the 18 table offsets; it ignores the monster in its way.
	hw2 := &hitWorld{emptyWorld: grid(), at: [2]int{5, 0}}
	evs = run(hw2, "meteorcenter", d2missile.CreateParams{X: 5, DestX: 5, Stationary: true, HitSubRange: 40}, 61)
	fires := 0

	for _, e := range evs {
		if e.Kind == d2missile.EventCreate && e.Missile.Spec.Name == "meteorfire" {
			fires++

			if e.Missile.Total != 40 {
				t.Errorf("meteorfire lifetime %d", e.Missile.Total)
			}
		}
	}

	if fires != 18 || countKind(evs, d2missile.EventHit) != 0 {
		t.Errorf("meteor: %d meteorfire, %d hits", fires, countKind(evs, d2missile.EventHit))
	}

	// Guided Arrow table parameters read by SrvDoFunc 7 / hit func 10:
	// Param1 = re-aim period, Param2 = re-target radius.
	if ga := tbl.ByName("guidedarrow"); ga.Param[0] != 5 || ga.Param[1] != 15 {
		t.Errorf("guidedarrow params %v", ga.Param)
	}
}
