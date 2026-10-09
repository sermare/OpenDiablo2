package d2skill

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// loadRealRegistry reads the 1.14 patch_d2 skills.txt from $D2_TABLES
// (skills/patch_d2/skills.txt) into a registry; the test is skipped when the
// variable is unset. No game data is stored in the repository.
func loadRealRegistry(t *testing.T) *Registry {
	t.Helper()

	root := os.Getenv("D2_TABLES")
	if root == "" {
		t.Skip("D2_TABLES not set")
	}

	f, err := os.Open(filepath.Join(root, "skills", "patch_d2", "skills.txt"))
	if err != nil {
		t.Skip(err)
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)

	var header []string

	reg := NewRegistry()

	for sc.Scan() {
		cols := strings.Split(sc.Text(), "\t")
		if header == nil {
			header = cols
			continue
		}

		r := row{}

		for i, c := range cols {
			if i < len(header) {
				r[header[i]] = c
			}
		}

		if r["skill"] != "" && r["Id"] != "" {
			reg.Add(skillFromRow(r))
		}
	}

	if reg.Len() < 200 {
		t.Fatalf("only %d skills read", reg.Len())
	}

	return reg
}

// realEnv builds an evaluation context for a skill at a level; base lists the
// caster's base levels of other skills (synergies read blvl).
func realEnv(reg *Registry, name string, lvl int, base map[string]int) *Env {
	u := newUnit(reg, map[string]int{})
	for k, v := range base {
		u.base[k] = v
	}

	return NewEnv(reg.ByName(name), lvl, u, reg)
}

// TestRealElementalDamage pins EMin/EMax/ELen and the mana cost of well-known
// skills from the real skills.txt. Expected values were computed by hand from
// the verified formulas (skills-2.md section 3: SKILL_SumLevelTierBonus
// 0x645f20, SKILL_GetElementalMin/Max/Length 0x646100/0x646200/0x6462f0,
// SKILL_PayManaCost 0x569d70): value = (E + tiers) << HitShift, plus the
// synergy percent of the shifted base. Damage is 8.8 fixed point.
//
// Fissure is internally "Eruption", Fire Blast is internally "Fire Trauma".
func TestRealElementalDamage(t *testing.T) {
	reg := loadRealRegistry(t)

	tests := []struct {
		skill    string
		lvl      int
		synergy  map[string]int
		min, max int32 // 8.8
		elen     int   // frames
		mana     int   // 8.8
	}{
		// Fire Bolt: HitShift 7 (half points), mana 5<<7 at every level.
		{"Fire Bolt", 1, nil, 768, 1536, 0, 640},
		{"Fire Bolt", 20, nil, 11648, 15488, 0, 640},
		{"Fire Bolt", 30, nil, 41344, 47744, 0, 640}, // tiers above 28 use column 5
		// +240%: (Fire Ball 10 + Meteor 5) * par8 16, of the shifted base
		{"Fire Bolt", 20, map[string]int{"Fire Ball": 10, "Meteor": 5}, 39603, 52659, 0, 640},
		// Frost Nova: chill ELen 200 + 25 per level from 2, mana 9+(l-1)
		{"Frost Nova", 1, nil, 512, 1024, 200, 2304},
		{"Frost Nova", 20, nil, 14336, 17280, 675, 7168},
		{"Frost Nova", 12, map[string]int{"Blizzard": 8, "Frozen Orb": 4}, 15769, 19993, 475, 5120},
		// Lightning: EMin 1 never grows and the synergy is skipped for the min
		// because base is not above 1.0 and EMinLev1 is 0 (0x646100 quirk).
		{"Lightning", 1, nil, 256, 10240, 0, 2048},
		{"Lightning", 20, nil, 256, 69632, 0, 4480},
		{"Lightning", 20, map[string]int{"Charged Bolt": 6, "Chain Lightning": 6, "Nova": 6}, 256, 169902, 0, 4480},
		// Poison Dagger: HitShift 1 and ELen 50 (poison); how the game turns
		// this into poison damage per frame is UNVERIFIED, only the columns
		// are pinned here.
		{"Poison Dagger", 1, nil, 36, 80, 50, 768},
		{"Poison Dagger", 20, nil, 576, 620, 240, 1984},
		{"Poison Dagger", 20, map[string]int{"Poison Explosion": 4, "Poison Nova": 4}, 1497, 1612, 240, 1984},
		// Fissure (Eruption): 15-25 fire, whole points, mana 15
		{"Eruption", 1, nil, 3840, 6400, 0, 3840},
		{"Eruption", 20, nil, 55552, 58112, 0, 3840},
		{"Eruption", 20, map[string]int{"Firestorm": 5, "Volcano": 5}, 122214, 127846, 0, 3840},
		// Fire Blast (Fire Trauma): HitShift 7, mana 24+(l-1) at shift 5
		{"Fire Trauma", 1, nil, 768, 1024, 0, 768},
		{"Fire Trauma", 20, nil, 21888, 29056, 0, 1376},
		{"Fire Trauma", 20, map[string]int{"Shock Field": 3, "Death Sentry": 3, "Charged Bolt Sentry": 3,
			"Lightning Sentry": 3, "Wake of Fire Sentry": 3, "Inferno Sentry": 3}, 57346, 76126, 0, 1376},
	}

	for _, tc := range tests {
		sk := reg.ByName(tc.skill)
		if sk == nil {
			t.Errorf("%s not in skills.txt", tc.skill)
			continue
		}

		e := realEnv(reg, tc.skill, tc.lvl, tc.synergy)

		if got := sk.ElemMin(e, tc.lvl); got != tc.min {
			t.Errorf("%s L%d %v EMin = %d, want %d", tc.skill, tc.lvl, tc.synergy, got, tc.min)
		}

		if got := sk.ElemMax(e, tc.lvl); got != tc.max {
			t.Errorf("%s L%d %v EMax = %d, want %d", tc.skill, tc.lvl, tc.synergy, got, tc.max)
		}

		if got := sk.ElemLen(e, tc.lvl); got != tc.elen {
			t.Errorf("%s L%d ELen = %d, want %d", tc.skill, tc.lvl, got, tc.elen)
		}

		if got := sk.ManaCost(tc.lvl); got != tc.mana {
			t.Errorf("%s L%d mana = %d, want %d", tc.skill, tc.lvl, got, tc.mana)
		}
	}
}

// TestRealPhysicalAndCalcs pins the physical columns (Raven) and the calc
// columns of the melee skills (Zeal, Sacrifice, Bash) from the real table.
func TestRealPhysicalAndCalcs(t *testing.T) {
	reg := loadRealRegistry(t)

	// Raven: MinDam 2, MaxDam 4 plus 1 per level, HitShift 8, no SrcDam, no
	// weapon part.
	raven := reg.ByName("Raven")
	for _, tc := range []struct {
		lvl      int
		min, max int32
	}{{1, 512, 1024}, {20, 5376, 5888}, {30, 7936, 8448}} {
		e := realEnv(reg, "Raven", tc.lvl, nil)
		if got := raven.PhysMin(e, tc.lvl, 99, true); got != tc.min {
			t.Errorf("Raven L%d PhysMin = %d, want %d", tc.lvl, got, tc.min)
		}

		if got := raven.PhysMax(e, tc.lvl, 99, true); got != tc.max {
			t.Errorf("Raven L%d PhysMax = %d, want %d", tc.lvl, got, tc.max)
		}
	}

	// SrcDam 128 means 100% of the weapon: Zeal adds the weapon unchanged.
	zeal := reg.ByName("Zeal")
	if got := zeal.PhysMin(realEnv(reg, "Zeal", 1, nil), 1, 10, true); got != 10<<8 {
		t.Errorf("Zeal weapon part = %d, want %d", got, 10<<8)
	}

	if got := zeal.PhysMin(realEnv(reg, "Zeal", 1, nil), 1, 10, false); got != 0 {
		t.Errorf("Zeal without weapon = %d, want 0", got)
	}

	tests := []struct {
		skill string
		calc  int // 1-based calc column
		lvl   int
		base  map[string]int
		want  int
	}{
		// Zeal calc1: strikes = min(par5 + lvl - 1, par6) = min(lvl+1, 5)
		{"Zeal", 1, 1, nil, 2},
		{"Zeal", 1, 4, nil, 5},
		{"Zeal", 1, 20, nil, 5},
		// Zeal calc2: damage percent, 0 below level 5, (lvl-4)*6 above, +12 per Sacrifice level
		{"Zeal", 2, 1, nil, 0},
		{"Zeal", 2, 4, nil, 0},
		{"Zeal", 2, 5, nil, 6},
		{"Zeal", 2, 20, nil, 96},
		{"Zeal", 2, 20, map[string]int{"Sacrifice": 10}, 216},
		// Sacrifice calc1: 180 + 15(l-1) + 15 per Redemption + 5 per Fanaticism
		{"Sacrifice", 1, 1, nil, 180},
		{"Sacrifice", 1, 20, nil, 465},
		{"Sacrifice", 1, 20, map[string]int{"Redemption": 5, "Fanaticism": 4}, 465 + 75 + 20},
		{"Sacrifice", 2, 7, nil, 8}, // self damage percent: par3
		// Bash calc1: damage percent 50 + 5(l-1) + 5 per Stun level; calc2 = ln34
		{"Bash", 1, 1, nil, 50},
		{"Bash", 1, 20, map[string]int{"Stun": 6}, 50 + 95 + 30},
		{"Bash", 2, 1, nil, 1},
		{"Bash", 2, 20, nil, 20},
		// Raven calc2 reads the caster level (ulvl 20 in the fixture): 20 - 2 + lvl
		{"Raven", 2, 5, nil, 23},
	}

	for _, tc := range tests {
		sk := reg.ByName(tc.skill)
		e := realEnv(reg, tc.skill, tc.lvl, tc.base)

		if got := e.Eval(sk.Calc[tc.calc]); got != tc.want {
			t.Errorf("%s calc%d L%d %v = %d, want %d (%q)", tc.skill, tc.calc, tc.lvl, tc.base, got, tc.want,
				sk.Calc[tc.calc].Source())
		}
	}
}

// TestRealToHitMana pins skill to-hit (percent AR bonus), ToHitCalc and the
// start-to-do cooldown column.
func TestRealToHitMana(t *testing.T) {
	reg := loadRealRegistry(t)

	for _, tc := range []struct {
		skill string
		lvl   int
		base  map[string]int
		want  int
	}{
		// ToHit + (lvl-1)*LevToHit when no ToHitCalc (0x645da0)
		{"Zeal", 1, nil, 10},
		{"Zeal", 20, nil, 200},
		{"Sacrifice", 10, nil, 20 + 9*7},
		{"Poison Dagger", 20, nil, 30 + 19*20},
		{"Raven", 3, nil, 100 + 2*15},
		// ToHitCalc wins over the columns: Bash 15 + 5 lvl + 5 per Concentrate level
		{"Bash", 1, nil, 20},
		{"Bash", 20, map[string]int{"Concentrate": 10}, 165},
		// Elemental bolts have no ToHit: they never roll (missile ToHit decides)
		{"Fire Bolt", 20, nil, 0},
		{"Lightning", 20, nil, 0},
	} {
		sk := reg.ByName(tc.skill)
		e := realEnv(reg, tc.skill, tc.lvl, tc.base)

		if got := sk.ToHitBonus(e, tc.lvl); got != tc.want {
			t.Errorf("%s L%d tohit = %d, want %d", tc.skill, tc.lvl, got, tc.want)
		}
	}

	// delay is a calc that compiles to a literal in the shipped data
	for _, tc := range []struct {
		skill string
		want  int
	}{{"Eruption", 50}, {"Fire Bolt", 0}, {"Zeal", 0}} {
		sk := reg.ByName(tc.skill)

		if got := realEnv(reg, tc.skill, 5, nil).Eval(sk.Delay); got != tc.want {
			t.Errorf("%s delay = %d, want %d", tc.skill, got, tc.want)
		}
	}
}

// TestCalcManaFields covers the skillcalc mana/mps/usmc fields (verified in
// SKILL_GetCalcFieldValue): no minmana floor, no free shortcut, 0 below L1.
func TestCalcManaFields(t *testing.T) {
	reg := loadRealRegistry(t)

	// Magic Arrow: mana 12, lvlmana -1, shift 5. At L20 the raw cost is
	// negative; SKILL_PayManaCost floors it at minmana (0) while the calc
	// field keeps the negative value (-224 >> 8 = -1, arithmetic shift).
	ma := reg.ByName("Magic Arrow")
	e := realEnv(reg, "Magic Arrow", 20, nil)

	for _, tc := range []struct {
		code string
		lvl  int
		want int
	}{
		{"usmc", 1, 384}, {"mana", 1, 1}, {"usmc", 20, -224}, {"mana", 20, -1},
		{"usmc", 0, 0}, {"mana", 0, 0},
		// mps: (12 * 25 / 2) << 5 = 150<<5 = 4800 -> >>8 = 18
		{"mps", 1, 18},
	} {
		e.Level = tc.lvl
		if got := e.Field(tc.code); got != tc.want {
			t.Errorf("Magic Arrow %s L%d = %d, want %d", tc.code, tc.lvl, got, tc.want)
		}
	}

	if got := ma.ManaCost(20); got != 0 {
		t.Errorf("pay cost at L20 = %d, want 0 (floored)", got)
	}

	// The cost actually paid still honours minmana: Frost Nova has minmana 1.
	if got := reg.ByName("Frost Nova").ManaCost(1); got != 9*256 {
		t.Errorf("Frost Nova L1 cost %d", got)
	}
}
