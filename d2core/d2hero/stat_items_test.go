package d2hero

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2statlist"
)

// A hero whose maxima are the item totals must still write the item-free
// maxima, or re-importing the save would count the items twice.
func TestExportWritesBaseMaxima(t *testing.T) {
	data, tables, state := realSave(t)

	s := state.Stats
	s.BaseMaxHealth, s.BaseMaxMana, s.BaseMaxStamina = s.MaxHealth, s.MaxMana, s.MaxStamina
	s.MaxHealth, s.MaxMana, s.MaxStamina = 1241, 477, 801

	out, _, err := ExportD2SWithOptions(state, data, tables, ExportOptions{})
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(out, data) {
		t.Fatal("exporting totals changed the stored maxima")
	}
}

// The totals travel to the client inside the hero stats JSON.
func TestStatsTotalsJSONRoundTrip(t *testing.T) {
	st := HeroStatsState{Level: 5, Totals: &d2statlist.Totals{Defense: 77, ResistShown: [4]int{1, 2, 3, 4}}}

	b, err := json.Marshal(st)
	if err != nil {
		t.Fatal(err)
	}

	var back HeroStatsState
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatal(err)
	}

	if back.Totals == nil || back.Totals.Defense != 77 || back.Totals.ResistShown[3] != 4 {
		t.Errorf("totals lost in %s", b)
	}
}

// TestRealSaveTotalsMatchStoredCurrentValues is the evidence for the
// "stored maxima are the values without items" finding: the real level 94
// Sorceress has hp 1241, mana 464 and stamina 801 (current, at full health)
// while the file stores maxima 869/221/525. Computing the equipment and
// inventory charm bonuses on top of the stored maxima must reproduce the
// current values.
func TestRealSaveTotalsMatchStoredCurrentValues(t *testing.T) {
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

	// the stored maxima follow the class formula (+20 life: Potion of Life quest reward)
	l, m, st := h.Class.BaseMax(h.Level, h.Vit, h.Ene)
	if l+20 != h.BaseLife || m != h.BaseMana || st != h.BaseStam {
		t.Errorf("class formula life=%d(+20) mana=%d stamina=%d, stored %d/%d/%d", l, m, st, h.BaseLife, h.BaseMana, h.BaseStam)
	}

	items := StatItemsFromD2S(c.Items, bases)
	tot := d2statlist.Compute(h, items, nil)

	t.Logf("items=%d life=%d mana=%d stamina=%d def=%d ar=%d dmg=%d-%d block=%d res=%v shown=%v str=%d dex=%d vit=%d ene=%d",
		len(items), tot.MaxLife, tot.MaxMana, tot.MaxStamina, tot.Defense, tot.AttackRating,
		tot.DamageMin, tot.DamageMax, tot.BlockPct, tot.Resist, tot.ResistShown, tot.Str, tot.Dex, tot.Vit, tot.Ene)

	// life and stamina were full when the save was written: the totals match exactly
	a := c.Body.Attributes
	if tot.MaxLife != int(a.CurrentHP) || tot.MaxStamina != int(a.CurrentStamina) {
		t.Errorf("life/stamina totals %d/%d, saved current values %d/%d", tot.MaxLife, tot.MaxStamina,
			a.CurrentHP, a.CurrentStamina)
	}

	// mana was not full (464 of 477): the current value only has to fit the total
	if int(a.CurrentMana) > tot.MaxMana || int(a.CurrentMana) <= h.BaseMana {
		t.Errorf("saved mana %d not in (%d, %d]", a.CurrentMana, h.BaseMana, tot.MaxMana)
	}
}
