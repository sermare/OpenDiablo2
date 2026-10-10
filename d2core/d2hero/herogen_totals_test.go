package d2hero

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2statlist"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2hero/herogen"
)

// The generated Barbarian (herogen) stores its current life, mana and stamina
// as the totals with the worn items. This proves the engine's own conversion of
// the saved items (StatItemsFromD2S) gives the same numbers, so the hero starts
// a game at full health. Needs D2_TABLES.
func TestGeneratedBarbarianTotalsMatchStoredCurrentValues(t *testing.T) {
	dir := os.Getenv("D2_TABLES")
	if dir == "" {
		t.Skip("D2_TABLES not set")
	}

	tb, err := herogen.LoadTables(dir)
	if err != nil {
		t.Fatal(err)
	}

	hero, err := tb.Generate(herogen.Barbarian("NokkaBarb"), nil)
	if err != nil {
		t.Fatal(err)
	}

	c, err := d2s.Parse(hero.Data, tb.Save)
	if err != nil {
		t.Fatal(err)
	}

	armor, err := os.ReadFile(filepath.Join(dir, "armor.txt"))
	if err != nil {
		t.Skip(err)
	}

	weapons, err := os.ReadFile(filepath.Join(dir, "weapons.txt"))
	if err != nil {
		t.Skip(err)
	}

	bases, err := d2statlist.ParseBases(armor, weapons)
	if err != nil {
		t.Fatal(err)
	}

	a := c.Body.Attributes
	h := d2statlist.Hero{
		Class: tb.Classes["Barbarian"], Level: int(a.Level), Str: int(a.Strength), Dex: int(a.Dexterity),
		Vit: int(a.Vitality), Ene: int(a.Energy), BaseLife: int(a.MaxHP), BaseMana: int(a.MaxMana),
		BaseStam: int(a.MaxStamina), Difficulty: 2,
	}

	tot := d2statlist.Compute(h, StatItemsFromD2S(c.Items, bases), nil)

	if tot.MaxLife != int(a.CurrentHP) || tot.MaxMana != int(a.CurrentMana) || tot.MaxStamina != int(a.CurrentStamina) {
		t.Errorf("engine totals %d/%d/%d, stored current values %d/%d/%d", tot.MaxLife, tot.MaxMana, tot.MaxStamina,
			a.CurrentHP, a.CurrentMana, a.CurrentStamina)
	}

	if tot.DamageMax != hero.Totals.DamageMax || tot.Defense != hero.Totals.Defense {
		t.Errorf("engine damage %d defense %d, generator %d / %d", tot.DamageMax, tot.Defense, hero.Totals.DamageMax,
			hero.Totals.Defense)
	}
}
