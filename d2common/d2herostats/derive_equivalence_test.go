package d2herostats

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2combat"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2statlist"
)

// exeMax is the rule verified in the exe (verify-monster-hero.md, level-up
// 0x0056e770): the maxima live in 1/256 fixed point; every gained level adds
// the charstats byte * 0x40 (an exact quarter point), every attribute point
// likewise; the displayed value is the truncated total. Written independently
// of Class.BaseMax (which uses integer division of quarter sums).
func exeMax(c d2statlist.Class, level, vit, ene int) (life, mana, stam int) {
	l256 := (c.HpAdd + c.InitVit) << 8
	m256 := c.InitEne << 8
	s256 := c.InitStamina << 8

	for n := 2; n <= level; n++ { // level-up by level, as the engine does it
		l256 += c.LifePerLevel * 0x40
		m256 += c.ManaPerLevel * 0x40
		s256 += c.StaminaPerLevel * 0x40
	}

	for v := c.InitVit; v < vit; v++ {
		l256 += c.LifePerVit * 0x40
		s256 += c.StaminaPerVit * 0x40
	}

	for e := c.InitEne; e < ene; e++ {
		m256 += c.ManaPerEne * 0x40
	}

	return l256 >> 8, m256 >> 8, s256 >> 8
}

// TestDeriveEqualsEngineRecalc compares, for every class and level 1..99 over
// a range of vitality/energy/dexterity, Derive against (a) d2statlist.Compute
// (the engine's RecalcStats path, no items) and (b) the exe fixed-point rule.
// Result: all equal, no engine change needed.
func TestDeriveEqualsEngineRecalc(t *testing.T) {
	dir := os.Getenv("D2_TABLES")
	if dir == "" {
		t.Skip("D2_TABLES not set")
	}

	data, err := os.ReadFile(filepath.Join(dir, "CharStats.txt"))
	if err != nil {
		t.Skip(err)
	}

	classes, err := d2statlist.ParseClasses(data)
	if err != nil {
		t.Fatal(err)
	}

	if len(classes) != 7 {
		t.Fatalf("want 7 classes, got %d", len(classes))
	}

	extra := []int{0, 1, 5, 17, 50, 120, 300, 500}
	dexExtra := []int{0, 1, 3, 4, 25, 77, 200, 400}
	checked := 0

	for _, name := range d2statlist.ClassNames {
		c := classes[name]

		for level := 1; level <= 99; level++ {
			for i, ev := range extra {
				vit, ene, dex := c.InitVit+ev, c.InitEne+extra[(i+3)%len(extra)], c.InitDex+dexExtra[i]
				d := Derive(c, Attributes{Level: level, Dex: dex, Vit: vit, Ene: ene})
				tot := d2statlist.Compute(d2statlist.Hero{
					Class: c, Level: level, Dex: dex, Vit: vit, Ene: ene,
					BaseLife: d.MaxLife, BaseMana: d.MaxMana, BaseStam: d.MaxStamina,
				}, nil, nil)
				el, em, es := exeMax(c, level, vit, ene)

				if d.MaxLife != el || d.MaxMana != em || d.MaxStamina != es {
					t.Fatalf("%s L%d vit %d ene %d: Derive %d/%d/%d, exe rule %d/%d/%d",
						name, level, vit, ene, d.MaxLife, d.MaxMana, d.MaxStamina, el, em, es)
				}

				if tot.MaxLife != d.MaxLife || tot.MaxMana != d.MaxMana || tot.MaxStamina != d.MaxStamina {
					t.Fatalf("%s L%d: Compute %d/%d/%d, Derive %d/%d/%d", name, level,
						tot.MaxLife, tot.MaxMana, tot.MaxStamina, d.MaxLife, d.MaxMana, d.MaxStamina)
				}

				if tot.Defense != d.Defense || tot.AttackRating != d.AttackRating {
					t.Fatalf("%s L%d dex %d: Compute def/ar %d/%d, Derive %d/%d", name, level, dex,
						tot.Defense, tot.AttackRating, d.Defense, d.AttackRating)
				}

				if want := d2combat.PlayerAttackRating(0, dex, c.ToHitFactor); d.AttackRating != want {
					t.Fatalf("%s AR %d want %d", name, d.AttackRating, want)
				}

				checked++
			}
		}
	}

	t.Logf("%d class/level/attribute combinations equal", checked)
}

// TestDeriveEqualsEngineOnSave runs the same comparison on the level 94 save.
func TestDeriveEqualsEngineOnSave(t *testing.T) {
	dir, js := os.Getenv("D2_TABLES"), os.Getenv("D2S_SAVE_JSON")
	if js == "" && os.Getenv("D2S_SAMPLE_BODY") != "" {
		js = os.Getenv("D2S_SAMPLE_BODY") + ".json"
	}

	if dir == "" || js == "" {
		t.Skip("D2_TABLES / D2S_SAVE_JSON (or D2S_SAMPLE_BODY) not set")
	}

	data, err := os.ReadFile(filepath.Join(dir, "CharStats.txt"))
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
			MaxMana                                      int `json:"max_mana"`
			MaxStamina                                   int `json:"max_stamina"`
		}
	}
	if err = json.Unmarshal(raw, &save); err != nil {
		t.Fatal(err)
	}

	classes, err := d2statlist.ParseClasses(data)
	if err != nil {
		t.Fatal(err)
	}

	a, c := save.Attributes, classes["Sorceress"]
	d := Derive(c, Attributes{Level: a.Level, Dex: a.Dexterity, Vit: a.Vitality, Ene: a.Energy})
	el, em, es := exeMax(c, a.Level, a.Vitality, a.Energy)
	tot := d2statlist.Compute(d2statlist.Hero{Class: c, Level: a.Level, Dex: a.Dexterity, Vit: a.Vitality, Ene: a.Energy,
		BaseLife: d.MaxLife, BaseMana: d.MaxMana, BaseStam: d.MaxStamina}, nil, nil)

	if d.MaxLife != el || d.MaxMana != em || d.MaxStamina != es ||
		tot.MaxLife != el || tot.MaxMana != em || tot.MaxStamina != es ||
		tot.Defense != d.Defense || tot.AttackRating != d.AttackRating {
		t.Errorf("save: Derive %+v, Compute %+v, exe %d/%d/%d", d, tot, el, em, es)
	}

	if em != a.MaxMana || es != a.MaxStamina {
		t.Errorf("exe rule mana/stamina %d/%d vs save %d/%d", em, es, a.MaxMana, a.MaxStamina)
	}
}
