package d2hero

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2statlist"
)

// A hero without skill buffs is computed exactly as before: no hook, an empty
// hook and an empty skill list all give the totals the items give alone
// (level 94 save oracle), and the factory builds no skill env for them.
func TestNoSkillBuffsLeavesTotalsUnchanged(t *testing.T) {
	data, tables, state := realSave(t)
	dir := os.Getenv("D2_TABLES")

	read := func(n string) []byte {
		b, err := os.ReadFile(filepath.Join(dir, n))
		if err != nil {
			t.Skipf("%s: %v", n, err)
		}

		return b
	}

	bases, err := d2statlist.ParseBases(read("armor.txt"), read("weapons.txt"))
	if err != nil {
		t.Fatal(err)
	}

	classes, err := d2statlist.ParseClasses(read("CharStats.txt"))
	if err != nil {
		t.Fatal(err)
	}

	c, err := d2s.Parse(data, tables)
	if err != nil {
		t.Fatal(err)
	}

	s := state.Stats
	h := d2statlist.Hero{
		Class: classes["Sorceress"], Level: s.Level, Str: s.Strength, Dex: s.Dexterity, Vit: s.Vitality,
		Ene: s.Energy, BaseLife: s.MaxHealth, BaseMana: s.MaxMana, BaseStam: s.MaxStamina,
	}
	items := StatItemsFromD2S(c.Items, bases)

	want := d2statlist.Compute(h, items, nil)

	for name, env := range map[string]*d2statlist.Env{
		"empty env": {}, "empty skill list": {Skill: d2statlist.NewList()},
	} {
		got := d2statlist.Compute(h, items, env)
		if got.Defense != want.Defense || got.MaxLife != want.MaxLife || got.MaxMana != want.MaxMana ||
			got.MaxStamina != want.MaxStamina || got.Resist != want.Resist || got.AttackRating != want.AttackRating {
			t.Errorf("%s changed the totals: %+v vs %+v", name, got, want)
		}
	}

	// a Frozen Armor buff raises the defense by its percent (stat 171)
	buff := d2statlist.NewList()
	buff.Add(d2statlist.StatSkillArmorPct, 0, 30)

	if got := d2statlist.Compute(h, items, &d2statlist.Env{Skill: buff}); got.Defense <= want.Defense {
		t.Errorf("defense with Frozen Armor %d, without %d", got.Defense, want.Defense)
	}
}

func TestSkillEnvNilWithoutBuffs(t *testing.T) {
	f := &HeroStateFactory{}

	if f.skillEnv(&HeroStatsState{}) != nil {
		t.Error("no hook must give no env")
	}

	if f.skillEnv(&HeroStatsState{SkillStats: func() map[string]int { return nil }}) != nil {
		t.Error("an empty hook must give no env")
	}
}
