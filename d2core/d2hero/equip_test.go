package d2hero

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2equip"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2statlist"
)

// fixtureCharms are the inventory charms of the save (they lift strength for the requirements).
var fixtureCharms []d2statlist.Item

// equipFixture builds a factory with the real tables and the worn items of the
// real level 94 Sorceress save (skips without D2_TABLES / D2S_SAMPLE_BODY).
func equipFixture(t *testing.T) (f *HeroStateFactory, c *HeroContainers, h d2statlist.Hero, data []byte, tables *d2s.ItemTables) {
	t.Helper()

	data, tables, state := realSave(t)
	dir := os.Getenv("D2_TABLES")

	read := func(n string) []byte {
		b, err := os.ReadFile(filepath.Join(dir, n))
		if err != nil {
			t.Skipf("%s: %v", n, err)
		}

		return b
	}

	types, err := d2equip.ParseTypes(read("ItemTypes.txt"))
	if err != nil {
		t.Fatal(err)
	}

	bases, err := d2equip.ParseBases(read("armor.txt"), read("weapons.txt"), read("misc.txt"))
	if err != nil {
		t.Fatal(err)
	}

	statBases, err := d2statlist.ParseBases(read("armor.txt"), read("weapons.txt"))
	if err != nil {
		t.Fatal(err)
	}

	classes, err := d2statlist.ParseClasses(read("CharStats.txt"))
	if err != nil {
		t.Fatal(err)
	}

	f = &HeroStateFactory{equipTried: true, equipRules: d2equip.Rules{Types: types}, equipBases: bases, statBases: statBases}

	parsed, err := d2s.Parse(data, tables)
	if err != nil {
		t.Fatal(err)
	}

	fixtureCharms = nil

	for _, it := range StatItemsFromD2S(parsed.Items, statBases) {
		if it.Charm {
			fixtureCharms = append(fixtureCharms, it)
		}
	}

	c = &HeroContainers{}
	importEquipped(c, data, parsed.Items, func(string) bool { return true })

	s := state.Stats
	cl := classes["Sorceress"]
	cl.ID = int(d2s.Sorceress)
	life, mana, stam := cl.BaseMax(s.Level, s.Vitality, s.Energy)
	h = d2statlist.Hero{
		Class: cl, Level: s.Level, Str: s.Strength, Dex: s.Dexterity, Vit: s.Vitality, Ene: s.Energy,
		BaseLife: life + 20, BaseMana: mana, BaseStam: stam,
	}

	return f, c, h, data, tables
}

func totalsOf(f *HeroStateFactory, c *HeroContainers, h d2statlist.Hero) (d2statlist.Totals, []EquipStatus) {
	items, status := f.ResolveEquipped(c, h, fixtureCharms)

	return d2statlist.Compute(h, append(items, fixtureCharms...), nil), status
}

func TestImportEquippedAndResolve(t *testing.T) {
	f, c, h, _, _ := equipFixture(t)

	if !c.EquippedSet || len(c.Equipped) < 10 || c.ActiveArms != 0 {
		t.Fatalf("equipped set=%v n=%d arms=%d", c.EquippedSet, len(c.Equipped), c.ActiveArms)
	}

	tot, status := totalsOf(f, c, h)
	for _, s := range status {
		if !s.Active && !s.Loc.IsSwap() {
			t.Errorf("%s %s is off: %s %s", s.Loc, s.Code, s.Reason, s.Detail)
		}
	}

	// the totals match the ones the stat list computes straight from the save (scenario 89: 1241 life)
	if tot.MaxLife != 1241 || tot.Defense < 1000 {
		t.Errorf("totals life=%d def=%d", tot.MaxLife, tot.Defense)
	}
}

func TestResolveRequirementsAndBroken(t *testing.T) {
	f, c, h, _, _ := equipFixture(t)
	base, _ := totalsOf(f, c, h)

	locOf := func(code string) int {
		for i := range c.Equipped {
			if c.Equipped[i].Code == code {
				return i
			}
		}

		t.Fatalf("no worn %s", code)

		return -1
	}

	// zero durability switches the item off and takes its defense away
	i := locOf("utp")
	zero := 0
	c.Equipped[i].Durability = &zero

	broken, status := totalsOf(f, c, h)
	if broken.Defense >= base.Defense {
		t.Errorf("broken torso: defense %d -> %d", base.Defense, broken.Defense)
	}

	found := false

	for _, s := range status {
		if s.Code == "utp" && s.Reason == d2equip.ReasonBroken && !s.Active {
			found = true
		}
	}

	if !found {
		t.Errorf("no broken verdict in %v", status)
	}

	// a damaged but not broken item still counts fully
	one := 1
	c.Equipped[i].Durability = &one

	if got, _ := totalsOf(f, c, h); got.Defense != base.Defense {
		t.Errorf("durability 1: defense %d, want %d", got.Defense, base.Defense)
	}

	c.Equipped[i].Durability = nil

	// a weak hero loses the items it cannot use but keeps the rest
	weak := h
	weak.Level = 5
	weak.Str, weak.Dex = 10, 10

	wt, wstatus := totalsOf(f, c, weak)
	off := 0

	for _, s := range wstatus {
		if !s.Active && !s.Loc.IsSwap() {
			off++
		}
	}

	if off == 0 || wt.Defense >= base.Defense {
		t.Errorf("a level 5 hero with 10 strength: %d items off, defense %d (full %d)", off, wt.Defense, base.Defense)
	}
}

func TestWeaponSetInStatItems(t *testing.T) {
	f, c, h, _, _ := equipFixture(t)
	set1, _ := totalsOf(f, c, h)

	// the logical hands of set II: the items in the file slots 11/12 are in the hands
	c.ActiveArms = 1
	set2, status := totalsOf(f, c, h)

	got := map[d2equip.Loc]string{}
	for _, s := range status {
		got[s.Loc] = s.Code
	}

	if got[d2equip.LocRightHand] == "" || got[d2equip.LocSwapRight] == "" {
		t.Fatalf("hands after the switch: %v", got)
	}

	if set1.DamageMax == set2.DamageMax && set1.Defense == set2.Defense {
		t.Errorf("the switch changed nothing: %+v", set2)
	}
}

func TestExportEquipmentWritesDurabilityAndArms(t *testing.T) {
	_, c, _, data, tables := equipFixture(t)

	for i := range c.Equipped {
		if c.Equipped[i].Code == "utp" {
			d := 7
			c.Equipped[i].Durability = &d
		}
	}

	c.ActiveArms = 1

	parsed, err := d2s.Parse(data, tables)
	if err != nil {
		t.Fatal(err)
	}

	exportEquipment(parsed, &HeroState{Containers: c}, func(string, ...interface{}) {})

	out, err := d2s.Write(parsed, tables)
	if err != nil {
		t.Fatal(err)
	}

	back, err := d2s.Parse(out, tables)
	if err != nil {
		t.Fatal(err)
	}

	if got := back.Header.Raw[activeArmsOffset]; got != 1 {
		t.Errorf("active arms byte %d", got)
	}

	seen := false

	for i := range back.Items {
		if it := &back.Items[i]; it.Location == d2s.LocationEquipped && trimCode(it.Code) == "utp" {
			seen = true

			if it.Durability != 7 {
				t.Errorf("torso durability %d, want 7", it.Durability)
			}
		}
	}

	if !seen {
		t.Error("torso missing after the export")
	}
}
