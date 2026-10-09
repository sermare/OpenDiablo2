package d2records

import (
	"math"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2missile"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2path"
)

// emptyWorld is an open field without units.
type emptyWorld struct {
	g     *d2path.CellGrid
	frame int
}

func (w *emptyWorld) Flags(x, y int) uint16                          { return w.g.Flags(x, y) }
func (w *emptyWorld) Targets(int, int) []d2missile.Target            { return nil }
func (w *emptyWorld) IsEnemy(d2missile.Owner, d2missile.Target) bool { return true }
func (w *emptyWorld) Frame() int                                     { return w.frame }

// flyUntilDead creates the missile at level lvl and returns the frames it
// lived and the x it reached.
func flyUntilDead(t *testing.T, sp *d2missile.Spec, lvl int) (frames int, x float64) {
	t.Helper()

	w := &emptyWorld{g: d2path.NewCellGrid(-300, -300, 600, 600)}
	s := d2missile.NewSim(w, nil)

	m, err := s.Create(d2missile.CreateParams{Spec: sp, Level: lvl, DestX: 90})
	if err != nil {
		t.Fatal(err)
	}

	for frames < 3000 && !m.Dead() {
		w.frame++
		frames++
		s.Step()
	}

	return frames, m.X
}

// TestRealMissileOracle pins the verified missiles.txt semantics (see the RE
// notes missiles-pathing.md) for ten well-known missiles against the real
// table. Per frame travel = pathVel/4096 subtiles, pathVel =
// ((VelLev*lvl)/8+Vel)<<8 * 75/100 (verified); Range is frames at 25 Hz.
func TestRealMissileOracle(t *testing.T) {
	tbl := loadRealRecords(t).MissileTable()

	type row struct {
		name           string
		doFunc, hitFn  int
		vel, maxVel    int
		accel, rng     int
		collideType    int
		kill, last     bool
		pierce, toHit  bool
		alwaysExplode  bool
		nextDelay      int
		wantFrames     int
		wantDist       float64 // 0: not checked
		explosionChild string
	}

	dist := func(vel, frames int) float64 { return float64(frames) * float64(vel*256*75/100) / 4096 }

	rows := []row{
		{name: "firebolt", doFunc: 1, vel: 20, maxVel: 20, rng: 50, collideType: 3, kill: true, last: true, explosionChild: "fireexplode", wantFrames: 50, wantDist: dist(20, 50)},
		{name: "fireball", doFunc: 1, hitFn: 1, vel: 20, maxVel: 20, rng: 50, collideType: 3, kill: true, last: true, explosionChild: "explodingarrowexp", wantFrames: 50},
		// ring: 4 frames at 4608, 5 at 3608, 5 at 2608 (accel acts on the scaled velocity every 5th frame)
		{name: "frostnova", doFunc: 1, vel: 24, maxVel: 24, accel: -1000, rng: 14, collideType: 3, last: true, nextDelay: 4, wantFrames: 14, wantDist: (4*4608 + 5*3608 + 5*2608) / 4096.0},
		{name: "lightningbolt", doFunc: 1, vel: 30, maxVel: 30, rng: 25, collideType: 3, last: true, wantFrames: 25, wantDist: dist(30, 25)},
		{name: "magicarrow", doFunc: 1, vel: 24, maxVel: 24, rng: 40, collideType: 3, kill: true, last: true, pierce: true, toHit: true, explosionChild: "teethexplode", wantFrames: 40, wantDist: dist(24, 40)},
		{name: "guidedarrow", doFunc: 7, hitFn: 10, vel: 24, maxVel: 24, rng: 128, collideType: 3, kill: true, alwaysExplode: true, wantFrames: 128},
		{name: "multipleshotarrow", doFunc: 1, vel: 24, maxVel: 24, rng: 50, collideType: 3, kill: true, last: true, pierce: true, toHit: true, nextDelay: 4, wantFrames: 50},
		{name: "poisonjav", doFunc: 2, vel: 24, maxVel: 24, rng: 25, collideType: 3, kill: true, last: true, pierce: true, toHit: true, wantFrames: 25},
		{name: "meteorcenter", doFunc: 1, hitFn: 14, rng: 60, last: true, alwaysExplode: true, wantFrames: 60},
		{name: "hydra", doFunc: 1, vel: 16, maxVel: 16, rng: 30, collideType: 3, kill: true, last: true, explosionChild: "fireexplode", wantFrames: 30, wantDist: dist(16, 30)},
	}

	for _, r := range rows {
		r := r

		t.Run(r.name, func(t *testing.T) {
			sp := tbl.ByName(r.name)
			if sp == nil {
				t.Fatal("missile missing")
			}

			if sp.SrvDoFunc != r.doFunc || sp.SrvHitFunc != r.hitFn || sp.Vel != r.vel || sp.MaxVel != r.maxVel ||
				sp.Accel != r.accel || sp.Range != r.rng || sp.CollideType != r.collideType ||
				sp.CollideKill != r.kill || sp.LastCollide != r.last || sp.Pierce != r.pierce || sp.ToHit != r.toHit ||
				sp.AlwaysExplode != r.alwaysExplode || sp.ExplosionMissile != r.explosionChild ||
				(sp.NextHit && sp.NextDelay != r.nextDelay) {
				t.Fatalf("table row differs: %+v", sp)
			}

			frames, x := flyUntilDead(t, sp, 1)
			if frames != r.wantFrames {
				t.Errorf("lived %d frames, want %d", frames, r.wantFrames)
			}

			if r.wantDist > 0 && math.Abs(x-r.wantDist) > 0.02 {
				t.Errorf("travelled %.3f subtiles, want %.3f", x, r.wantDist)
			}
		})
	}
}

// TestRealMissileChains pins the Sub/Hit chains of the sample missiles.
func TestRealMissileChains(t *testing.T) {
	tbl := loadRealRecords(t).MissileTable()

	if got := tbl.ByName("poisonjav").SubMissile[0]; got != "poisonjavcloud" {
		t.Errorf("poison javelin SubMissile1 = %q", got)
	}

	mc := tbl.ByName("meteorcenter")
	if mc.HitSubMissile[0] != "meteorfire" || mc.SrvHitFunc != 14 {
		t.Errorf("meteorcenter chain %v hit func %d", mc.HitSubMissile, mc.SrvHitFunc)
	}

	mf := tbl.ByName("meteorfire")
	if !mf.SubLoop || mf.SubStart != 12 || mf.SubStop != 36 || mf.Range != 90 || mf.Size != 2 || mf.SrvDmgFunc != 3 {
		t.Errorf("meteorfire %+v", mf)
	}

	// Blessed Hammer: positive accel, MaxVel above Vel
	bh := tbl.ByName("blessedhammer")
	if bh.Vel != 18 || bh.MaxVel != 30 || bh.Accel != 250 {
		t.Errorf("blessedhammer %+v", bh)
	}
}
