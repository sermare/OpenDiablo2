package d2client

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2calc"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2skill"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2records"
)

func TestPlanClientShots(t *testing.T) {
	caster := d2skill.SubPoint{X: 0, Y: 0}
	target := d2skill.SubPoint{X: 10, Y: 0}

	t.Run("multiple shot fans calc1 arrows", func(t *testing.T) {
		rec := &d2records.SkillRecord{Cltdofunc: 17, Cltmissilea: "arrow", Cltmissileb: "bolt", Calc1: d2calc.Compile("par1+lvl", d2calc.KindSkill)}
		rec.Param1 = 2

		shots := planClientShots(rec, 1, caster, target)
		if len(shots) != 3 {
			t.Fatalf("%d shots, want 3", len(shots))
		}

		seen := map[d2skill.SubPoint]bool{}

		for _, s := range shots {
			if s.slot != 1 || s.from != caster {
				t.Errorf("shot %+v", s)
			}

			seen[s.to] = true
		}

		if len(seen) != 3 {
			t.Errorf("aim points not distinct: %v", seen)
		}
	})

	t.Run("charged strike bolts leave the target once each", func(t *testing.T) {
		rec := &d2records.SkillRecord{Cltdofunc: 19, Cltmissilea: "b", Cltmissileb: "b", Cltmissilec: "b", Calc1: d2calc.Compile("4", d2calc.KindSkill)}

		shots := planClientShots(rec, 1, caster, target)
		if len(shots) != 4 {
			t.Fatalf("%d shots, want 4 (not 3x the columns)", len(shots))
		}

		for _, s := range shots {
			if s.from != target || s.to != (d2skill.SubPoint{X: 20, Y: 0}) {
				t.Errorf("shot %+v", s)
			}
		}
	})

	t.Run("curse draws the cast effect at the target", func(t *testing.T) {
		rec := &d2records.SkillRecord{Cltstfunc: 18, Cltdofunc: 30, Cltmissilea: "curse", Cltmissilec: "cursecast"}

		shots := planClientShots(rec, 1, caster, target)
		if len(shots) != 2 || shots[0].from != target || shots[1].slot != 3 || shots[1].from != target {
			t.Errorf("%+v", shots)
		}
	})

	t.Run("undispatched skills keep every column at the caster", func(t *testing.T) {
		rec := &d2records.SkillRecord{Cltstfunc: 11, Cltmissile: "x", Cltmissiled: "y"}

		shots := planClientShots(rec, 1, caster, target)
		if len(shots) != 2 || shots[0].from != caster || shots[1].slot != 4 {
			t.Errorf("%+v", shots)
		}
	})
}
