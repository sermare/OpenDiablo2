package d2difficulty_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2combat"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2difficulty"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2hero"
)

// TestRealDifficultyLevels compares the built-in table and every engine
// consumer of it (resist penalty, death experience penalty, cold/freeze
// divisors) with the extracted DifficultyLevels.txt. D2_TABLES is the table
// directory; the file is d2exp/difficultylevels.txt. Skipped when unset.
//
// The d2exp file lacks the columns the 1.14b patch adds (damage bonuses,
// StaticFieldMin, gamble odds) and has Hell life/mana steal divisors of 2
// where patch_d2 has 3, so only columns present are compared and the steal
// divisors may differ by exactly that documented patch override.
func TestRealDifficultyLevels(t *testing.T) {
	root := os.Getenv("D2_TABLES")
	if root == "" {
		t.Skip("D2_TABLES not set")
	}

	data, err := os.ReadFile(filepath.Join(root, "d2exp", "difficultylevels.txt"))
	if err != nil {
		t.Skip(err)
	}

	got, err := d2difficulty.ParseTable(data)
	if err != nil {
		t.Fatal(err)
	}

	type col struct {
		name string
		g, w int
	}

	for l := d2difficulty.Normal; l < d2difficulty.Count; l++ {
		g, w := got[l], d2difficulty.Default[l]

		if g.Name != w.Name {
			t.Errorf("%v: name %q, built-in %q", l, g.Name, w.Name)
		}

		for _, c := range []col{
			{"ResistPenalty", g.ResistPenalty, w.ResistPenalty},
			{"DeathExpPenalty", g.DeathExpPenalty, w.DeathExpPenalty},
			{"UberCodeOddsNormal", g.UberCodeOddsNormal, w.UberCodeOddsNormal},
			{"UberCodeOddsGood", g.UberCodeOddsGood, w.UberCodeOddsGood},
			{"UltraCodeOddsNormal", g.UltraCodeOddsNormal, w.UltraCodeOddsNormal},
			{"UltraCodeOddsGood", g.UltraCodeOddsGood, w.UltraCodeOddsGood},
			{"MonsterSkillBonus", g.MonsterSkillBonus, w.MonsterSkillBonus},
			{"MonsterFreezeDivisor", g.MonsterFreezeDivisor, w.MonsterFreezeDivisor},
			{"MonsterColdDivisor", g.MonsterColdDivisor, w.MonsterColdDivisor},
			{"AiCurseDivisor", g.AiCurseDivisor, w.AiCurseDivisor},
			{"ExtraUniqueMonsters", g.ExtraUniqueMonsters, w.ExtraUniqueMonsters},
		} {
			if c.g != c.w {
				t.Errorf("%v %s: table %d, built-in %d", l, c.name, c.g, c.w)
			}
		}

		for _, c := range []col{
			{"LifeStealDivisor", g.LifeStealDivisor, w.LifeStealDivisor},
			{"ManaStealDivisor", g.ManaStealDivisor, w.ManaStealDivisor},
		} {
			if c.g != c.w && !(l == d2difficulty.Hell && c.g == 2 && c.w == 3) {
				t.Errorf("%v %s: table %d, built-in %d", l, c.name, c.g, c.w)
			}
		}

		// engine consumers
		if p := d2combat.LoDResistPenalty(int(l)); p != g.ResistPenalty {
			t.Errorf("%v: d2combat.LoDResistPenalty %d, table %d", l, p, g.ResistPenalty)
		}

		if p := d2difficulty.ResistPenalty(false, l); p != g.ResistPenalty {
			t.Errorf("%v: ResistPenalty %d, table %d", l, p, g.ResistPenalty)
		}

		if p := d2hero.DeathExpPenaltyPercent(int(l)); p != g.DeathExpPenalty {
			t.Errorf("%v: DeathExpPenaltyPercent %d, table %d", l, p, g.DeathExpPenalty)
		}

		if d2difficulty.ColdDivisor(l) != g.MonsterColdDivisor || d2difficulty.FreezeDivisor(l) != g.MonsterFreezeDivisor {
			t.Errorf("%v: cold/freeze divisors disagree with the table", l)
		}
	}
}
