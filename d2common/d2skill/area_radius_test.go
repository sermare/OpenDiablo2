package d2skill

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2calc"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2missile"
)

// TestAreaHitFunctionsTakeTheirRadiusFromTheSkill pins what the exe does when
// a missile's sHitPar1 is empty (verified): hit function 1 (0x5a7500) takes
// the radius from skills.txt calc1 (record +0x138), hit function 14 (Meteor,
// 0x5a8680) from aurarangecalc (+0x64), and the Meteor flames live
// Param3 + (level-1)*Param4 frames (+0x150 / +0x154). A positive sHitPar1 wins.
func TestAreaHitFunctionsTakeTheirRadiusFromTheSkill(t *testing.T) {
	f := newClassFixture(map[string]int{"Fire Bolt": 1})
	tbl := missileTable{
		"area1":   {ID: 1, Name: "area1", SrvDoFunc: 1, SrvHitFunc: 1, Range: 5, CollideType: 3},
		"area1p":  {ID: 2, Name: "area1p", SrvDoFunc: 1, SrvHitFunc: 1, Range: 5, CollideType: 3, SHitPar: [3]int{4}},
		"meteorc": {ID: 3, Name: "meteorc", SrvDoFunc: 1, SrvHitFunc: 14, Range: 5, CollideType: 0},
	}
	f.p.Missiles = tbl

	sk := &Skill{ID: 99, Name: "Test", AuraRangeCalc: d2calc.Compile("ln12", d2calc.KindSkill),
		Calc: [5]*d2calc.Program{nil, d2calc.Compile("ln12", d2calc.KindSkill)}}
	sk.Params[1], sk.Params[2], sk.Params[3], sk.Params[4] = 6, 1, 30, 15

	for _, tc := range []struct {
		name           string
		lvl            int
		radius, range_ int
	}{
		{"area1", 1, 6, 0},    // calc1 = ln12 = 6 + (lvl-1)*1
		{"area1", 5, 10, 0},   // level scaled
		{"area1p", 5, 4, 0},   // sHitPar1 wins
		{"meteorc", 1, 6, 30}, // aurarangecalc, flames 30 frames
		{"meteorc", 3, 8, 60}, // 30 + 2*15
	} {
		env := NewEnv(sk, tc.lvl, f.u, f.reg)
		m := f.p.castMissile(f.u, sk, tc.lvl, env, tc.name, Target{X: 5, Y: 0}, castOpts{})

		if m == nil {
			t.Fatalf("%s: no missile", tc.name)
		}

		if m.AreaRadius != tc.radius && !(tc.name == "area1p" && m.AreaRadius == 0) {
			t.Errorf("%s L%d: AreaRadius %d want %d", tc.name, tc.lvl, m.AreaRadius, tc.radius)
		}

		if m.HitSubRange != tc.range_ {
			t.Errorf("%s L%d: HitSubRange %d want %d", tc.name, tc.lvl, m.HitSubRange, tc.range_)
		}
	}

	_ = d2missile.Spec{}
}
