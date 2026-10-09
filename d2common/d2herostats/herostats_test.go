package d2herostats

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2statlist"
)

// miniExp builds a small Experience.txt with the given thresholds for every
// class (rows 0..len-1, MaxLvl = len-1).
func miniExp(th []int64) []byte {
	var b strings.Builder

	b.WriteString(strings.Join(append([]string{"Level"}, ClassNames...), "\t") + "\r\n")

	row := func(first string, v int64) {
		b.WriteString(first)

		for range ClassNames {
			fmt.Fprintf(&b, "\t%d", v)
		}

		b.WriteString("\r\n")
	}

	row("MaxLvl", int64(len(th)-1))

	for i, v := range th {
		row(fmt.Sprint(i), v)
	}

	return []byte(b.String())
}

func TestExperienceLevelUp(t *testing.T) {
	tabs, err := ParseExperience(miniExp([]int64{0, 500, 1500, 3750, 7000}))
	if err != nil {
		t.Fatal(err)
	}

	tab := tabs["Sorceress"]

	for _, tc := range []struct {
		exp  int64
		want int
	}{{0, 1}, {499, 1}, {500, 2}, {1499, 2}, {1500, 3}, {3750, 4}, {7000, 4}, {99999, 4}} {
		if got := tab.LevelFor(tc.exp); got != tc.want {
			t.Errorf("LevelFor(%d) = %d, want %d", tc.exp, got, tc.want)
		}
	}

	if got := tab.NextLevelExp(1); got != 500 {
		t.Errorf("NextLevelExp(1) = %d", got)
	}

	if got := tab.NextLevelExp(4); got != 7000 {
		t.Errorf("NextLevelExp(max) = %d", got)
	}

	p := &Progress{Level: 1}
	if n := tab.AddExperience(p, 1600); n != 2 || p.Level != 3 || p.StatPoints != 10 || p.SkillPoints != 2 {
		t.Errorf("after +1600: n=%d %+v", n, p)
	}

	if n := tab.AddExperience(p, 1_000_000); n != 1 || p.Level != 4 || p.Experience != 3750 {
		t.Errorf("cap: n=%d %+v", n, p)
	}

	// Death penalty never drops the level.
	if n := tab.AddExperience(p, -100000); n != 0 || p.Level != 4 || p.Experience != 3750 {
		t.Errorf("penalty: n=%d %+v", n, p)
	}
}

func TestParseExperienceErrors(t *testing.T) {
	for name, in := range map[string]string{
		"empty":     "",
		"no class":  "Level\tFoo\nMaxLvl\t3\n0\t0\n",
		"bad count": strings.Replace(string(miniExp([]int64{0, 5, 9})), "MaxLvl\t2", "MaxLvl\t5", 1),
	} {
		if _, err := ParseExperience([]byte(in)); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}

func TestBreakpointModels(t *testing.T) {
	n := 0

	for _, e := range Tables {
		if e.Model == nil {
			continue
		}

		n++

		if got := e.Model.Table(); !reflect.DeepEqual(got, e.Table) {
			t.Errorf("%s %s %s: model gives %v, table %v", e.Class, e.Kind, e.Note, got, e.Table)
		}
	}

	if n < 8 {
		t.Errorf("only %d modelled tables", n)
	}
}

func TestBreakpointTablesWellFormed(t *testing.T) {
	for _, c := range ClassNames {
		for _, k := range []Kind{FHR, FCR, FBR} {
			e, ok := Lookup(c, k)
			if !ok {
				t.Errorf("no %s table for %s", k, c)
				continue
			}

			if e.Table[0] != 0 {
				t.Errorf("%s %s does not start at 0", c, k)
			}

			for i := 1; i < len(e.Table); i++ {
				if e.Table[i] <= e.Table[i-1] {
					t.Errorf("%s %s not increasing: %v", c, k, e.Table)
				}
			}
		}
	}
}

func TestReachedAndFrames(t *testing.T) {
	sorcFCR, _ := Lookup("Sorceress", FCR)
	for _, tc := range []struct{ pct, idx, frames int }{
		{0, 0, 14}, {8, 0, 14}, {9, 1, 13}, {105, 5, 9}, {200, 6, 8}, {500, 6, 8},
	} {
		if got := Reached(sorcFCR.Table, tc.pct); got != tc.idx {
			t.Errorf("Reached(%d) = %d, want %d", tc.pct, got, tc.idx)
		}

		if got := Frames(14, 256, tc.pct); got != tc.frames {
			t.Errorf("Frames(%d%%) = %d, want %d", tc.pct, got, tc.frames)
		}
	}

	if EffectivePct(0) != 0 || EffectivePct(120) != 60 || EffectivePct(-5) != 0 {
		t.Error("EffectivePct")
	}

	// IAS (unverified formula): monotonic, never below 1, capped by +75.
	prev := AttackFrames(16, 0, 0, 0)
	for ias := 0; ias <= 500; ias += 5 {
		f := AttackFrames(16, ias, 0, 0)
		if f > prev || f < 1 {
			t.Fatalf("AttackFrames not monotonic at %d: %d after %d", ias, f, prev)
		}

		prev = f
	}

	if AttackFrames(16, 0, 20, 0) <= AttackFrames(16, 0, -20, 0) {
		t.Error("a slow weapon must not attack faster than a fast one")
	}
}

func TestDeriveStartingHeroes(t *testing.T) {
	// Sorceress, from charstats.txt: start life = hpadd+vit = 40, mana 35,
	// stamina 74; level 1 and starting attributes.
	c := d2statlist.Class{
		InitStr: 10, InitDex: 25, InitVit: 10, InitEne: 35, InitStamina: 74, HpAdd: 30,
		LifePerVit: 8, ManaPerEne: 8, StaminaPerVit: 4,
		LifePerLevel: 4, ManaPerLevel: 8, StaminaPerLevel: 4, ToHitFactor: -15,
	}

	d := Derive(c, Attributes{Level: 1, Str: 10, Dex: 25, Vit: 10, Ene: 35})
	if d.MaxLife != 40 || d.MaxMana != 35 || d.MaxStamina != 74 {
		t.Errorf("level 1: %+v", d)
	}

	if d.AttackRating != (25-7)*5-15 || d.Defense != 6 {
		t.Errorf("AR/defense: %+v", d)
	}

	// One level-up and one vitality point.
	d = Derive(c, Attributes{Level: 2, Vit: 11, Ene: 35, Dex: 25})
	if d.MaxLife != 40+1+2 || d.MaxStamina != 74+1+1 || d.MaxMana != 35+2 {
		t.Errorf("level 2: %+v", d)
	}

	if l, m, s := LevelUpGain(c); l != 4 || m != 8 || s != 4 {
		t.Error("LevelUpGain")
	}

	if l, s, m := PointGain(c); l != 8 || s != 4 || m != 8 {
		t.Error("PointGain")
	}
}

// TestRealLevel94Sorceress checks the oracle: the real level 94 save and the
// real tables. D2_TABLES must hold Experience.txt and CharStats.txt,
// D2S_SAVE_JSON the parsed save (nokka-d2s-ref examples/nokkasorc.json).
func TestRealLevel94Sorceress(t *testing.T) {
	dir, js := os.Getenv("D2_TABLES"), os.Getenv("D2S_SAVE_JSON")
	if dir == "" || js == "" {
		t.Skip("D2_TABLES / D2S_SAVE_JSON not set")
	}

	expData, err := os.ReadFile(filepath.Join(dir, "Experience.txt"))
	if err != nil {
		t.Skip("no Experience.txt:", err)
	}

	csData, err := os.ReadFile(filepath.Join(dir, "CharStats.txt"))
	if err != nil {
		t.Skip(err)
	}

	raw, err := os.ReadFile(js)
	if err != nil {
		t.Skip(err)
	}

	var save struct {
		Attributes struct {
			Strength, Energy, Dexterity, Vitality, Level int
			Experience                                   int64
			MaxHP                                        int `json:"max_hp"`
			MaxMana                                      int `json:"max_mana"`
			MaxStamina                                   int `json:"max_stamina"`
		}
	}
	if err = json.Unmarshal(raw, &save); err != nil {
		t.Fatal(err)
	}

	a := save.Attributes

	tabs, err := ParseExperience(expData)
	if err != nil {
		t.Fatal(err)
	}

	tab := tabs["Sorceress"]
	if tab.MaxLevel != 99 || tab.Threshold[93] != 2286478756 || tab.Threshold[94] != 2492671933 {
		t.Fatalf("unexpected table: max %d", tab.MaxLevel)
	}

	if got := tab.LevelFor(a.Experience); got != a.Level {
		t.Errorf("LevelFor(%d) = %d, save says %d", a.Experience, got, a.Level)
	}

	// Gaining the missing experience levels the hero up exactly once.
	p := &Progress{Level: a.Level, Experience: a.Experience}
	if n := tab.AddExperience(p, tab.NextLevelExp(a.Level)-a.Experience); n != 1 || p.Level != 95 {
		t.Errorf("level-up from 94: n=%d %+v", n, p)
	}

	classes, err := d2statlist.ParseClasses(csData)
	if err != nil {
		t.Fatal(err)
	}

	d := Derive(classes["Sorceress"], Attributes{Level: a.Level, Str: a.Strength, Dex: a.Dexterity, Vit: a.Vitality, Ene: a.Energy})

	// Mana and stamina match exactly; the stored life carries the +20 of the
	// Act 3 Potion of Life quest reward on top of the class formula.
	if d.MaxMana != a.MaxMana || d.MaxStamina != a.MaxStamina || d.MaxLife+20 != a.MaxHP {
		t.Errorf("derived %+v vs save life %d mana %d stamina %d", d, a.MaxHP, a.MaxMana, a.MaxStamina)
	}

	if d.Defense != a.Dexterity/4 {
		t.Errorf("defense %d", d.Defense)
	}
}

func TestLevelUpGrantsAccumulateQuarters(t *testing.T) {
	// VERIFIED 0x0056e770: each level adds charstats quarter points to the
	// 1/256 fixed-point maxima; the shown value is the truncated total, so
	// per-level rounding never drifts from the closed form.
	c := d2statlist.Class{InitVit: 10, HpAdd: 30, LifePerLevel: 5, InitEne: 10, ManaPerLevel: 6, InitStamina: 80, StaminaPerLevel: 4}
	q := 0

	for lvl := 1; lvl <= 40; lvl++ {
		if lvl > 1 {
			q += c.LifePerLevel
		}

		life, _, _ := c.BaseMax(lvl, c.InitVit, c.InitEne)
		if want := c.HpAdd + c.InitVit + q/4; life != want {
			t.Fatalf("level %d life %d want %d", lvl, life, want)
		}
	}
}

func TestPointSpendGrants(t *testing.T) {
	// VERIFIED 0x0056ea50 / 0x0056e970.
	c := d2statlist.Class{LifePerVit: 8, StaminaPerVit: 4, ManaPerEne: 6}
	r := Resources{Life: Pool{1000, 2000}, Stamina: Pool{500, 600}, Mana: Pool{50, 100}}

	v := ApplyVitality(c, r, 3)
	if v.Life != (Pool{1000 + 8*3*64, 2000 + 8*3*64}) || v.Stamina != (Pool{500 + 4*3*64, 600 + 4*3*64}) {
		t.Errorf("vit +3: %+v", v)
	}

	e := ApplyEnergy(c, r, 2)
	if e.Mana != (Pool{50 + 6*2*64, 100 + 6*2*64}) {
		t.Errorf("ene +2: %+v", e.Mana)
	}

	// a refund lowers the maximum only and clamps current to it
	low := Resources{Life: Pool{2000, 2000}}
	if got := ApplyVitality(c, low, -2).Life; got != (Pool{2000 - 8*2*64, 2000 - 8*2*64}) {
		t.Errorf("refund: %+v", got)
	}
}

func TestAttackRatingFormula(t *testing.T) {
	// VERIFIED 0x00622710: (dex-7)*5 + stat 0x13 (gear to-hit) + charstats ToHitFactor.
	c := d2statlist.Class{ToHitFactor: 5}
	if got := Derive(c, Attributes{Level: 1, Dex: 25, ItemToHit: 10}).AttackRating; got != (25-7)*5+10+5 {
		t.Errorf("attack rating %d", got)
	}
}
