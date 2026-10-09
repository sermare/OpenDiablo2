package d2hero

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2equip"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2hireling"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2statlist"
)

// gearFactory is a factory whose equip rules and base tables come from the tables in
// D2_TABLES (skips without them): GiveMercItem needs no game assets besides these.
func gearFactory(t *testing.T) *HeroStateFactory {
	t.Helper()

	dir := os.Getenv("D2_TABLES")
	if dir == "" {
		t.Skip("set D2_TABLES to run")
	}

	read := func(name string) []byte {
		b, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Skipf("%s: %v", name, err)
		}

		return b
	}

	armor, weapons, types, misc := read("armor.txt"), read("weapons.txt"), read("ItemTypes.txt"), read("misc.txt")

	ty, err := d2equip.ParseTypes(types)
	if err != nil {
		t.Fatal(err)
	}

	eb, err := d2equip.ParseBases(armor, weapons, misc)
	if err != nil {
		t.Fatal(err)
	}

	sb, err := d2statlist.ParseBases(armor, weapons)
	if err != nil {
		t.Fatal(err)
	}

	return &HeroStateFactory{equipTried: true, equipRules: d2equip.Rules{Types: ty}, equipBases: eb, statBases: sb}
}

func storedCode(code string) StoredItem {
	return StoredItem{Code: code, Identified: true, Quality: int(d2s.QualityNormal)}
}

func TestGiveAndTakeMercItems(t *testing.T) {
	f := gearFactory(t)
	m := &MercState{ID: 1, Type: 0}
	ref := MercRef{Class: d2equip.MercRogue, Base: d2hireling.Stats{Level: 30, Str: 100, Dex: 100, Defense: 50, MaxHP: 300}}

	if got := f.MercStatItems(m); got != nil {
		t.Errorf("no gear: %v", got)
	}

	// body armor, helm and a bow are fine for the rogue
	for _, tc := range []struct {
		code string
		loc  d2equip.Loc
	}{{"qui", d2equip.LocTorso}, {"cap", d2equip.LocHead}, {"sbw", d2equip.LocRightHand}} {
		res := f.GiveMercItem(m, ref, storedCode(tc.code))
		if !res.OK || res.Loc != tc.loc || res.Returned != nil {
			t.Fatalf("give %s: %+v", tc.code, res)
		}
	}

	if !m.ItemsDirty || len(m.Items) != 3 {
		t.Fatalf("items = %d dirty = %v", len(m.Items), m.ItemsDirty)
	}

	// refusals change nothing: a sword for a rogue, boots, an unidentified helm
	unid := storedCode("skp")
	unid.Identified = false

	for _, it := range []StoredItem{storedCode("ssd"), storedCode("lbt"), unid} {
		if res := f.GiveMercItem(m, ref, it); res.OK {
			t.Errorf("%s should be refused: %+v", it.Code, res)
		}
	}

	if len(m.Items) != 3 || m.MercItemAt(d2equip.LocHead).Code != "cap" {
		t.Errorf("a refusal changed the gear: %s", m.MercItemsSummary())
	}

	// the gear shows in the stat items and in ApplyGear
	items := f.MercStatItems(m)
	if len(items) != 3 {
		t.Fatalf("stat items = %d", len(items))
	}

	if g := d2hireling.ApplyGear(ref.Base, items); g.Defense <= ref.Base.Defense || g.DmgMax <= ref.Base.DmgMax {
		t.Errorf("gear gives nothing: %+v", g)
	}

	// replacing the helm returns the old one
	res := f.GiveMercItem(m, ref, storedCode("skp"))
	if !res.OK || res.Returned == nil || res.Returned.Code != "cap" || m.MercItemAt(d2equip.LocHead).Code != "skp" {
		t.Fatalf("replace: %+v", res)
	}

	// a requirement the merc does not meet (full plate mail needs 80 strength)
	weak := ref
	weak.Base.Str = 60

	if res := f.GiveMercItem(m, weak, storedCode("ful")); res.OK {
		t.Errorf("full plate for a 60 strength merc: %+v", res)
	}

	// healing potions are drunk, nothing is equipped
	if res := f.GiveMercItem(m, ref, StoredItem{Code: "hp1", Identified: true}); !res.OK || !res.Consumed || len(m.Items) != 3 {
		t.Errorf("potion: %+v items=%d", res, len(m.Items))
	}

	if it := m.TakeMercItem(d2equip.LocTorso); it == nil || it.Code != "qui" || len(m.Items) != 2 || m.TakeMercItem(d2equip.LocTorso) != nil {
		t.Errorf("take: %v", it)
	}
}

// mercSample returns the sample save with a 'jf' section, its tables and the imported merc
// items. The sample has a mercenary but no gear, so two of the hero's worn items are copied
// into the merc's section (the format does not care what class wore them).
func mercSample(t *testing.T) ([]byte, *d2s.ItemTables, *HeroState, *d2s.Character) {
	t.Helper()

	data, tables, state := realSave(t)

	c, err := d2s.Parse(data, tables)
	if err != nil {
		t.Fatal(err)
	}

	if state.Merc == nil {
		t.Skip("the sample save has no mercenary")
	}

	if len(c.MercItems) == 0 {
		for i := range c.Items {
			if it := c.Items[i]; it.Location == d2s.LocationEquipped && !it.Simple && (it.Equipped == 1 || it.Equipped == 3) {
				c.MercItems = append(c.MercItems, it)
			}
		}

		if len(c.MercItems) == 0 {
			t.Skip("the sample save has no head or torso item to lend")
		}

		if data, err = d2s.Write(c, tables); err != nil {
			t.Fatal(err)
		}

		if c, err = d2s.Parse(data, tables); err != nil || len(c.MercItems) == 0 {
			t.Fatalf("the synthetic jf section does not parse back: %v", err)
		}

		state.D2SBase = data
	}

	importMercItems(state.Merc, c.MercItems, func(string) bool { return true })

	return data, tables, state, c
}

// TestMercItemsImportAndExport: the 'jf' items import into the merc state, an export of
// untouched gear is byte-exact even when marked changed, and taking an item off removes it
// from the written file.
func TestMercItemsImportAndExport(t *testing.T) {
	data, tables, state, c := mercSample(t)

	if len(state.Merc.Items) != len(c.MercItems) {
		t.Fatalf("imported %d of %d merc items", len(state.Merc.Items), len(c.MercItems))
	}

	out, _, err := ExportD2SWithOptions(state, data, tables, ExportOptions{})
	if err != nil || !bytes.Equal(out, data) {
		t.Fatalf("untouched gear: err=%v equal=%v", err, bytes.Equal(out, data))
	}

	state.Merc.ItemsDirty = true

	out, _, err = ExportD2SWithOptions(state, data, tables, ExportOptions{})
	if err != nil || !bytes.Equal(out, data) {
		t.Fatalf("rewritten unchanged gear differs from the original: err=%v", err)
	}

	taken := state.Merc.TakeMercItem(d2equip.Loc(state.Merc.Items[0].X))
	if taken == nil {
		t.Fatal("take")
	}

	out, _, err = ExportD2SWithOptions(state, data, tables, ExportOptions{})
	if err != nil {
		t.Fatal(err)
	}

	got, err := d2s.Parse(out, tables)
	if err != nil {
		t.Fatal(err)
	}

	if len(got.MercItems) != len(c.MercItems)-1 {
		t.Errorf("merc items after take = %d, want %d", len(got.MercItems), len(c.MercItems)-1)
	}

	// the hero's own items are untouched
	if len(got.Items) != len(c.Items) {
		t.Errorf("hero items %d, want %d", len(got.Items), len(c.Items))
	}
}
