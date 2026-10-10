package d2herostats

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2combat"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2statlist"
)

// Progression audit: charstats.txt and Experience.txt against pinned numbers
// (the numbers of the 1.14 tables, which the real files reproduce). The pinned
// tables run without game data; the *Real tests read D2_TABLES and skip.

type charRow struct {
	name                        string
	str, dex, vit, ene, stamina int
	hpAdd                       int
	lifeLvl, manaLvl, stamLvl   int // quarters per level
	lifeVit, stamVit, manaEne   int // quarters per point
	toHit, block, runDrain      int
	startLife, startMana        int
}

var charRows = []charRow{
	{"Amazon", 20, 25, 20, 15, 84, 30, 8, 6, 4, 12, 4, 6, 5, 25, 20, 50, 15},
	{"Sorceress", 10, 25, 10, 35, 74, 30, 4, 8, 4, 8, 4, 8, -15, 20, 20, 40, 35},
	{"Necromancer", 15, 25, 15, 25, 79, 30, 6, 8, 4, 8, 4, 8, -10, 20, 20, 45, 25},
	{"Paladin", 25, 20, 25, 15, 89, 30, 8, 6, 4, 12, 4, 6, 20, 30, 20, 55, 15},
	{"Barbarian", 30, 20, 25, 10, 92, 30, 8, 4, 4, 16, 4, 4, 20, 25, 20, 55, 10},
	{"Druid", 15, 20, 25, 20, 84, 30, 6, 8, 4, 8, 4, 8, 5, 20, 20, 55, 20},
	{"Assassin", 20, 20, 20, 25, 95, 30, 8, 6, 5, 12, 5, 7, 15, 25, 15, 50, 25},
}

func (r charRow) class(id int) d2statlist.Class {
	return d2statlist.Class{
		ID: id, InitStr: r.str, InitDex: r.dex, InitVit: r.vit, InitEne: r.ene, InitStamina: r.stamina,
		HpAdd: r.hpAdd, LifePerVit: r.lifeVit, ManaPerEne: r.manaEne, StaminaPerVit: r.stamVit,
		LifePerLevel: r.lifeLvl, ManaPerLevel: r.manaLvl, StaminaPerLevel: r.stamLvl,
		ToHitFactor: r.toHit, BlockFactor: r.block,
	}
}

func TestPinnedStartingVitals(t *testing.T) {
	for i, r := range charRows {
		c := r.class(i)
		life, mana, stam := c.BaseMax(1, r.vit, r.ene)

		if life != r.startLife || mana != r.startMana || stam != r.stamina {
			t.Errorf("%s level 1: life %d mana %d stamina %d, want %d %d %d", r.name, life, mana, stam, r.startLife, r.startMana, r.stamina)
		}
	}
}

func TestLevelAndPointGains(t *testing.T) {
	for i, r := range charRows {
		c := r.class(i)

		// four levels grant exactly lifePerLevel/mana/stamina whole points
		l1, m1, s1 := c.BaseMax(1, r.vit, r.ene)
		l5, m5, s5 := c.BaseMax(5, r.vit, r.ene)

		if l5-l1 != r.lifeLvl || m5-m1 != r.manaLvl || s5-s1 != r.stamLvl {
			t.Errorf("%s: 4 levels gave %d/%d/%d, want %d/%d/%d", r.name, l5-l1, m5-m1, s5-s1, r.lifeLvl, r.manaLvl, r.stamLvl)
		}

		// four vitality / energy points
		lv, _, sv := c.BaseMax(1, r.vit+4, r.ene)
		_, me, _ := c.BaseMax(1, r.vit, r.ene+4)

		if lv-l1 != r.lifeVit || sv-s1 != r.stamVit || me-m1 != r.manaEne {
			t.Errorf("%s: 4 points gave life %d stamina %d mana %d, want %d %d %d", r.name, lv-l1, sv-s1, me-m1, r.lifeVit, r.stamVit, r.manaEne)
		}

		// derived and per-step resource application agree: spending n vitality
		// points from base grows max life by n*lifeVit/4 (in quarter points)
		res := ApplyVitality(c, Resources{Life: Pool{Cur: 40 << 8, Max: 40 << 8}}, 4)
		if res.Life.Max-(40<<8) != r.lifeVit*4*0x40 {
			t.Errorf("%s: ApplyVitality(4) grew max life by %d raw", r.name, res.Life.Max-(40<<8))
		}
	}

	if StatPointsPerLevel != 5 || SkillPointsPerLevel != 1 {
		t.Errorf("points per level %d/%d, want 5/1", StatPointsPerLevel, SkillPointsPerLevel)
	}
}

func TestDerivedRatingsPerClass(t *testing.T) {
	for i, r := range charRows {
		c := r.class(i)
		// attack rating (dex-7)*5 + ToHitFactor, defense dex/4, block (dex-15)*(toBlock+factor)/(2*level)
		d := Derive(c, Attributes{Level: 1, Dex: r.dex, Vit: r.vit, Ene: r.ene})

		if want := (r.dex-7)*5 + r.toHit; d.AttackRating != want {
			t.Errorf("%s attack rating %d, want %d", r.name, d.AttackRating, want)
		}

		if want := r.dex / 4; d.Defense != want {
			t.Errorf("%s defense %d, want %d", r.name, d.Defense, want)
		}

		blk := d2combat.PlayerBlockChance(d2combat.PlayerBlockInput{
			HasShield: true, ToBlock: 30, ClassBlockFactor: r.block, Dex: r.dex, Level: 10, IncludeDex: true,
		})
		if want := (r.dex - 15) * (30 + r.block) / 20; want > 75 {
			want = 75
		} else if blk != want {
			t.Errorf("%s block %d, want %d", r.name, blk, want)
		}
	}
}

func TestPinnedExperienceRows(t *testing.T) {
	// first and last rows of Experience.txt (identical for the seven classes)
	rows := map[int]int64{1: 500, 2: 1500, 3: 3750, 10: 72144, 25: 2050449, 50: 51767302, 75: 481128591, 90: 1764543065, 98: 3520485254, 99: 3837739017}
	tb := NewExpTable(99, func(l int) int64 { return rows[l] })

	cases := []struct {
		exp  int64
		want int
	}{
		{0, 1}, {499, 1}, {500, 2}, {1499, 2}, {1500, 3}, {3750, 4}, {1764543064, 76}, {3520485254, 99}, {1 << 40, 99},
	}

	for _, tc := range cases {
		// only rows pinned above matter for these exp values
		got := lookupLevel(rows, tc.exp)
		if got != tc.want {
			t.Errorf("exp %d: level %d, want %d", tc.exp, got, tc.want)
		}
	}

	if tb.ExpCap() != 3520485254 {
		t.Errorf("cap %d", tb.ExpCap())
	}
}

// lookupLevel is LevelFor over a sparse set of rows: the level is the largest
// l+1 whose row l threshold is <= exp, bounded by 99.
func lookupLevel(rows map[int]int64, exp int64) int {
	lvl := 1

	for l := 1; l <= 99; l++ {
		if v, ok := rows[l]; ok && exp >= v {
			lvl = l + 1
		}
	}

	if lvl > 99 {
		lvl = 99
	}

	return lvl
}

func readTSV(t *testing.T, name string) [][]string {
	t.Helper()

	dir := os.Getenv("D2_TABLES")
	if dir == "" {
		t.Skip("D2_TABLES not set")
	}

	data, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Skip(err)
	}

	var out [][]string

	for _, l := range strings.Split(strings.ReplaceAll(string(data), "\r", ""), "\n") {
		if l != "" {
			out = append(out, strings.Split(l, "\t"))
		}
	}

	return out
}

func TestRealCharStatsMatchPinned(t *testing.T) {
	rows := readTSV(t, "CharStats.txt")
	col := map[string]int{}

	for i, h := range rows[0] {
		col[strings.ToLower(h)] = i
	}

	byName := map[string][]string{}
	for _, r := range rows[1:] {
		byName[r[0]] = r
	}

	num := func(r []string, name string) int {
		i, ok := col[strings.ToLower(name)]
		if !ok || i >= len(r) {
			t.Fatalf("column %q missing", name)
		}

		v, err := strconv.Atoi(r[i])
		if err != nil {
			t.Fatalf("%s %s: %v", r[0], name, err)
		}

		return v
	}

	for _, want := range charRows {
		r := byName[want.name]
		if r == nil {
			t.Fatalf("%s missing from CharStats.txt", want.name)
		}

		got := charRow{
			name: want.name, str: num(r, "str"), dex: num(r, "dex"), vit: num(r, "vit"), ene: num(r, "int"),
			stamina: num(r, "stamina"), hpAdd: num(r, "hpadd"), lifeLvl: num(r, "LifePerLevel"), manaLvl: num(r, "ManaPerLevel"),
			stamLvl: num(r, "StaminaPerLevel"), lifeVit: num(r, "LifePerVitality"), stamVit: num(r, "StaminaPerVitality"),
			manaEne: num(r, "ManaPerMagic"), toHit: num(r, "ToHitFactor"), block: num(r, "BlockFactor"), runDrain: num(r, "RunDrain"),
			startLife: want.startLife, startMana: want.startMana,
		}
		if got != want {
			t.Errorf("%s: file %+v, pinned %+v", want.name, got, want)
		}

		if n := num(r, "StatPerLevel"); n != StatPointsPerLevel {
			t.Errorf("%s StatPerLevel %d, engine grants %d", want.name, n, StatPointsPerLevel)
		}

		if num(r, "WalkVelocity") != 6 || num(r, "RunVelocity") != 9 {
			t.Errorf("%s walk/run velocity %d/%d", want.name, num(r, "WalkVelocity"), num(r, "RunVelocity"))
		}
	}
}

func TestRealExperienceTable(t *testing.T) {
	data := readTSV(t, "Experience.txt")
	joined := make([]string, 0, len(data))

	for _, r := range data {
		joined = append(joined, strings.Join(r, "\t"))
	}

	tabs, err := ParseExperience([]byte(strings.Join(joined, "\n")))
	if err != nil {
		t.Fatal(err)
	}

	amazon := tabs["Amazon"]

	for _, c := range ClassNames {
		tb := tabs[c]
		if tb.MaxLevel != 99 {
			t.Errorf("%s max level %d", c, tb.MaxLevel)
		}

		for l := 1; l <= 99; l++ {
			if tb.Threshold[l] <= tb.Threshold[l-1] {
				t.Fatalf("%s: row %d (%d) not above row %d (%d)", c, l, tb.Threshold[l], l-1, tb.Threshold[l-1])
			}

			if tb.Threshold[l] != amazon.Threshold[l] {
				t.Errorf("%s row %d differs from Amazon", c, l)
			}
		}

		// level boundaries: exactly at a threshold the hero is one level higher
		for l := 1; l < 99; l++ {
			if got := tb.LevelFor(tb.Threshold[l]); got != l+1 {
				t.Errorf("%s: exp %d (row %d) gives level %d, want %d", c, tb.Threshold[l], l, got, l+1)
			}

			if got := tb.LevelFor(tb.Threshold[l] - 1); got != l {
				t.Errorf("%s: exp %d gives level %d, want %d", c, tb.Threshold[l]-1, got, l)
			}
		}

		// the experience that first reaches level 99 is row 98
		if got := tb.LevelFor(tb.Threshold[98]); got != 99 {
			t.Errorf("%s: row 98 gives level %d", c, got)
		}

		p := Progress{Level: 1}
		if n := tb.AddExperience(&p, tb.Threshold[98]); n != 98 || p.Level != 99 || p.StatPoints != 5*98 || p.SkillPoints != 98 {
			t.Errorf("%s: 1 -> 99 gave %d levels, %+v", c, n, p)
		}
	}
}

func TestStaminaDrainPerStep(t *testing.T) {
	cases := []struct {
		name                        string
		runDrain, weight, pct, want int
		town                        bool
	}{
		{"barbarian bare", 20, -1, 0, 40, false},
		{"assassin bare", 15, -1, 0, 30, false},
		{"light armor weight 0", 20, 0, 0, 40, false},
		{"weight 9 still x1", 20, 9, 0, 40, false},
		{"weight 10 x2", 20, 10, 0, 80, false},
		{"weight 35 x4", 20, 35, 0, 160, false},
		{"drain percent 50", 20, -1, 50, 20, false},
		{"drain percent 100 floors at 1", 20, -1, 100, 1, false},
		{"town is free", 20, 50, 0, 0, true},
	}

	for _, tc := range cases {
		if got := StaminaDrainPerStep(tc.runDrain, tc.weight, tc.pct, tc.town); got != tc.want {
			t.Errorf("%s: %d, want %d", tc.name, got, tc.want)
		}
	}
}

func TestStaminaRegenPerTick(t *testing.T) {
	max := 100 << 8

	cases := []struct {
		name                  string
		cur, mode, pct, wantv int
	}{
		{"standing", 10 << 8, ModeNeutral, 0, max >> 8},
		{"standing with 50 percent recovery", 10 << 8, ModeNeutral, 50, max>>8 + (max>>8)*50/100},
		{"town standing", 10 << 8, ModeTownNeutral, 0, max >> 8},
		{"town walking is half", 10 << 8, ModeTownWalk, 0, max >> 9},
		{"walking is half", 10 << 8, ModeWalk, 0, max >> 9},
		{"walking at empty stamina does not recover", 100, ModeWalk, 0, 0},
		{"running does not recover", 10 << 8, ModeRun, 0, 0},
		{"full stamina", max, ModeNeutral, 0, 0},
	}

	for _, tc := range cases {
		if got := StaminaRegenPerTick(max, tc.cur, tc.mode, tc.pct); got != tc.wantv {
			t.Errorf("%s: %d, want %d", tc.name, got, tc.wantv)
		}
	}

	if got := ClampStamina(max-1, 50, max); got != max {
		t.Errorf("clamp: %d", got)
	}
}

func TestLevelDiffPenaltyPinned(t *testing.T) {
	// full table of 0x006e2960 / 0x006e298c plus the clvl >= 25 ratio rule
	char := [11]int{256, 256, 256, 256, 256, 256, 207, 159, 110, 61, 13}
	mon := [11]int{256, 256, 256, 256, 256, 256, 225, 174, 92, 38, 5}

	if CharAboveTable() != char || MonsterAboveTable() != mon {
		t.Fatal("level difference tables changed")
	}

	const xp = 25600

	for d := 0; d <= 12; d++ {
		idx := d
		if idx > 10 {
			idx = 10
		}

		if got, want := LevelScaleXP(xp, 50-d, 50), xp*char[idx]/256; got != want {
			t.Errorf("monster %d below clvl 50: %d, want %d", d, got, want)
		}

		if got, want := LevelScaleXP(xp, 10+d, 10), xp*mon[idx]/256; got != want {
			t.Errorf("monster %d above clvl 10: %d, want %d", d, got, want)
		}
	}

	// clvl >= 25 and a stronger monster: xp * clvl / mlvl, no table
	if got := LevelScaleXP(1000, 60, 30); got != 500 {
		t.Errorf("monster 60 vs clvl 30: %d, want 500", got)
	}

	// at the cap nothing is gained; +% experience applies after the scaling
	if KillXP(1000, 30, 99, 99, 0) != 0 || KillXP(1000, 30, 98, 99, 50) != LevelScaleXP(1000, 30, 98)*3/2 {
		t.Error("cap or item experience wrong")
	}
}
