package d2monsters

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2monster"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2asset"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2records"
)

// director with a monlvl.txt that has the real patch_d2 rows of level 1 and 99
// (the L-* columns, which the engine uses), and one monstats record.
func difficultyDirector(diff d2monster.Difficulty, area int) *Director {
	rec := &d2records.MonsterLevelRecord{Level: 1}
	rec.Ladder.Normal.Hitpoints, rec.Ladder.Normal.DefenseRating, rec.Ladder.Normal.AttackRating = 7, 6, 8
	rec.Ladder.Normal.Damage, rec.Ladder.Normal.Experience = 2, 30
	rec.Ladder.Nightmare.Hitpoints, rec.Ladder.Nightmare.DefenseRating, rec.Ladder.Nightmare.AttackRating = 107, 108, 108
	rec.Ladder.Nightmare.Damage, rec.Ladder.Nightmare.Experience = 3, 78
	rec.Ladder.Hell.Hitpoints, rec.Ladder.Hell.DefenseRating, rec.Ladder.Hell.AttackRating = 1107, 173, 216
	rec.Ladder.Hell.Damage, rec.Ladder.Hell.Experience = 4, 117

	am := &d2asset.AssetManager{Records: &d2records.RecordManager{}}
	am.Records.Monster.Levels = d2records.MonsterLevels{1: rec}

	return &Director{asset: am, opt: Options{Difficulty: diff}, areaLevel: area}
}

func ratioStat() *d2records.MonStatRecord {
	r := &d2records.MonStatRecord{Key: "t", ID: 1}
	r.LevelNormal, r.LevelNightmare, r.LevelHell = 1, 36, 69
	r.MinHPNormal, r.MaxHPNormal, r.MinHPNightmare, r.MaxHPNightmare, r.MinHPHell, r.MaxHPHell = 100, 100, 200, 200, 50, 50
	r.ArmorClassNormal, r.ArmorClassNightmare, r.ArmorClassHell = 100, 110, 120
	r.AttackRatingA1Normal, r.AttackRatingA1Nightmare, r.AttackRatingA1Hell = 100, 100, 80
	r.DamageMinA1Normal, r.DamageMaxA1Normal = 100, 100
	r.DamageMinA1Hell, r.DamageMaxA1Hell = 40, 100
	r.ExperienceNormal, r.ExperienceNightmare, r.ExperienceHell = 100, 100, 150
	r.TreasureClassNormal, r.TreasureClassNightmare, r.TreasureClassHell = "T", "T (N)", "T (H)"

	return r
}

func TestComputeVitalsPerDifficulty(t *testing.T) {
	// every number is monlvl value * ratio / 100, truncated (hand computed)
	for _, c := range []struct {
		diff                               d2monster.Difficulty
		level, hp, def, xp, th, dmin, dmax int
		tc                                 string
	}{
		{d2monster.Normal, 1, 7, 6, 30, 8, 2, 2, "T"},              // 7*100/100, 6*100/100, 30, 8, 2*100/100
		{d2monster.Nightmare, 1, 214, 118, 78, 108, 0, 0, "T (N)"}, // 107*200/100, 108*110/100=118.8->118
		{d2monster.Hell, 1, 553, 207, 175, 172, 1, 4, "T (H)"},     // 1107*50/100=553, 173*120/100=207, 117*150/100=175, 216*80/100=172, 4*40/100=1
	} {
		d := difficultyDirector(c.diff, 1)
		b := d2monster.NewBrain(1, 1, c.diff, &d2monster.Profile{}, 1)
		v := d.computeVitals(ratioStat(), b)

		if v.Level != c.level || v.MaxHP != c.hp || v.Defense != c.def || v.Experience != c.xp ||
			v.A1.ToHit != c.th || v.TreasureClass != c.tc {
			t.Errorf("%v: got level=%d hp=%d def=%d xp=%d th=%d tc=%q", c.diff, v.Level, v.MaxHP, v.Defense,
				v.Experience, v.A1.ToHit, v.TreasureClass)
		}

		// Nightmare has no damage ratio set: min=max=0 -> 0
		if c.diff != d2monster.Nightmare && (v.A1.Min != c.dmin || v.A1.Max != c.dmax) {
			t.Errorf("%v: damage %d-%d want %d-%d", c.diff, v.A1.Min, v.A1.Max, c.dmin, c.dmax)
		}
	}
}

func TestComputeVitalsLevelRule(t *testing.T) {
	// no area: the monstats Level of the difficulty (no monlvl row for 36/69 -> raw 100% fallback)
	d := difficultyDirector(d2monster.Hell, 0)
	b := d2monster.NewBrain(1, 1, d2monster.Hell, &d2monster.Profile{}, 1)

	if v := d.computeVitals(ratioStat(), b); v.Level != 69 {
		t.Errorf("monstats Level(H) = %d, want 69", v.Level)
	}

	// an area level replaces it for ordinary classes ...
	d = difficultyDirector(d2monster.Hell, 1)
	if v := d.computeVitals(ratioStat(), b); v.Level != 1 {
		t.Errorf("area MonLvl = %d, want 1", v.Level)
	}

	// ... but not for noRatio classes, which use the monstats values as they are
	st := ratioStat()
	st.IgnoreMonLevelTxt = true

	if v := d.computeVitals(st, b); v.Level != 69 {
		t.Errorf("noRatio level = %d, want 69", v.Level)
	}
}

func TestMonsterResists(t *testing.T) {
	r := &d2records.MonStatRecord{}
	r.ResistanceFireNormal, r.ResistanceFireNightmare, r.ResistanceFireHell = 0, 25, 75
	r.ResistanceColdHell = -20

	for diff, want := range [][6]int{{}, {0, 0, 25, 0, 0, 0}, {0, 0, 75, 0, -20, 0}} {
		if got := MonsterResists(r, d2monster.Difficulty(diff)); got != want {
			t.Errorf("diff %d: %v want %v", diff, got, want)
		}
	}
}
