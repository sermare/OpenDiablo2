package d2hero

import (
	"errors"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2records"
)

// VERIFIED (Game.exe 0x645b20, 0x645b90): the next point needs character level
// reqlevel + points already in the skill, so the 20th point of a row-6 skill
// (reqlevel 30) needs level 49. Real tables (D2_TABLES), every class skill
// without prerequisites checked at the boundary.
func TestRealPerPointLevelGate(t *testing.T) {
	for token, skills := range realTrees(t) {
		for _, s := range skills {
			if s.Reqskill1 != "" || s.Reqskill2 != "" || s.Reqskill3 != "" {
				continue
			}

			for _, have := range []int{0, 1, 9, 19} {
				s.SetPoints(have)
				need := s.Reqlevel + have

				lo := &HeroStatsState{Level: need - 1, SkillPoints: 5}
				if err := CanSpend(s.SkillRecord, PoolOf(skills, lo, token)); err == nil {
					t.Errorf("%s/%s: %d points at level %d allowed, needs %d", token, s.Skill, have, need-1, need)
				}

				hi := &HeroStatsState{Level: need, SkillPoints: 5}
				if err := CanSpend(s.SkillRecord, PoolOf(skills, hi, token)); err != nil {
					t.Errorf("%s/%s: %d points at level %d refused: %v", token, s.Skill, have, need, err)
				}
			}

			s.SetPoints(0)
		}
	}
}

// VERIFIED (0x645ce0): reqstr, reqint, reqdex and reqvit are compared with the
// hero's strength, energy, dexterity and vitality. The stock tables leave all
// four empty, so this uses a synthetic record.
func TestSpendStatRequirements(t *testing.T) {
	rec := &d2records.SkillRecord{Skill: "X", Charclass: "sor", Reqlevel: 1, Reqstr: 10, Reqdex: 20, Reqint: 30, Reqvit: 40}
	base := SkillPool{Class: "sor", Level: 1, Unused: 1, Str: 10, Dex: 20, Int: 30, Vit: 40, Points: func(string) int { return 0 }}

	if err := CanSpend(rec, base); err != nil {
		t.Fatalf("exact stats refused: %v", err)
	}

	for name, mut := range map[string]func(*SkillPool){
		"strength": func(p *SkillPool) { p.Str-- }, "dexterity": func(p *SkillPool) { p.Dex-- },
		"energy": func(p *SkillPool) { p.Int-- }, "vitality": func(p *SkillPool) { p.Vit-- },
	} {
		p := base
		mut(&p)

		var re *SkillReqError
		if err := CanSpend(rec, p); !errors.As(err, &re) || re.Stat != name {
			t.Errorf("%s one short: err = %v", name, err)
		}
	}
}
