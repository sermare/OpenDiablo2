package d2monsters

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2monster"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2records"
)

func TestProfileFromRecordPicksDifficulty(t *testing.T) {
	r := &d2records.MonStatRecord{Key: "x", ID: 3, AiKey: "Skeleton"}
	r.AiParameterNormal1, r.AiParameterNightmare1, r.AiParameterHell1 = 60, 65, 70
	r.AiDistanceNormal, r.AiDistanceHell = 30, 50
	r.SkillId1, r.SkillAnimation1 = "Foo", "S2"

	for diff, want := range []int{60, 65, 70} {
		p := profileFromRecord(r, d2monster.Difficulty(diff))
		if p.AIP[1] != want || p.AI != "Skeleton" || p.Class != 3 {
			t.Errorf("diff %d: %+v", diff, p)
		}
	}

	p := profileFromRecord(r, d2monster.Hell)
	if p.AIDist != 50 || !p.Skills[0].Used() || p.Skills[0].Mode != d2monster.ModeSkill2 {
		t.Errorf("hell profile %+v", p)
	}
}

func TestAttackFor(t *testing.T) {
	v := &struct{}{}
	_ = v
	if a := MonsterAttackFrom(10, 5, 3); a.Max != 5 {
		t.Errorf("max must be raised to min: %+v", a)
	}
}
