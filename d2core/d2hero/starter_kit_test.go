package d2hero

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/diablo2item"
)

// druidKit is the Druid row of CharStats.txt (item1..item6 with item<n>loc and item<n>count).
func druidKit() *HeroContainers {
	codes := []string{"clb", "buc", "hp1", "tsc", "isc", "0"}
	locs := []string{"rarm", "larm", "", "", "", ""}
	counts := []int{1, 1, 4, 1, 1, 0}

	return startingKit(codes, locs, counts, func(string) (int, bool) { return 1, true })
}

func TestStartingKitDruid(t *testing.T) {
	c := druidKit()

	if !c.EquippedSet || len(c.Equipped) != 2 {
		t.Fatalf("worn: set=%v items=%v", c.EquippedSet, c.Equipped)
	}

	if e := c.Equipped[0]; e.Code != "clb" || e.X != 4 || e.Page != PageEquipped || !e.Identified {
		t.Errorf("right hand: %+v", e)
	}

	if e := c.Equipped[1]; e.Code != "buc" || e.X != 5 {
		t.Errorf("left hand: %+v", e)
	}

	belt, inv := c.Page(PageBelt), c.Page(PageInventory)
	if len(belt) != 4 || len(inv) != 2 {
		t.Fatalf("belt %d inventory %d, want 4 and 2", len(belt), len(inv))
	}

	for i, b := range belt {
		if b.Code != "hp1" || b.X != i {
			t.Errorf("belt cell %d: %+v", i, b)
		}
	}

	if inv[0].Code != "tsc" || inv[1].Code != "isc" || inv[0].X == inv[1].X {
		t.Errorf("inventory: %+v", inv)
	}

	for _, s := range append(append([]StoredItem{}, c.Items...), c.Equipped...) {
		if !s.Identified {
			t.Errorf("%s is not identified", s.Code)
		}
	}
}

func TestStartingKitSkipsUnknownAndOverflow(t *testing.T) {
	known := func(code string) (int, bool) { return 1, code != "zzz" }
	c := startingKit([]string{"zzz", "hp1", "hp1", "tsc"}, []string{"rarm"}, []int{1, 3, 3, 1}, known)

	if len(c.Equipped) != 0 {
		t.Errorf("an unknown base item must not be worn: %v", c.Equipped)
	}

	// 4 belt cells; the extra potions go to the inventory
	if got := len(c.Page(PageBelt)); got != 4 {
		t.Errorf("belt %d, want 4", got)
	}

	if got := len(c.Page(PageInventory)); got != 3 {
		t.Errorf("inventory %d, want 3 (2 spare potions and the scroll)", got)
	}
}

// tsvRows reads a tab separated table of D2_TABLES into rows keyed by column name.
func tsvRows(t *testing.T, name string) []map[string]string {
	t.Helper()

	dir := os.Getenv("D2_TABLES")
	if dir == "" {
		t.Skip("set D2_TABLES to run")
	}

	b, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Skipf("%s: %v", name, err)
	}

	lines := strings.Split(strings.ReplaceAll(string(b), "\r", ""), "\n")
	head := strings.Split(lines[0], "\t")

	var rows []map[string]string

	for _, ln := range lines[1:] {
		if ln == "" {
			continue
		}

		row := map[string]string{}
		for i, f := range strings.Split(ln, "\t") {
			if i < len(head) {
				row[head[i]] = f
			}
		}

		rows = append(rows, row)
	}

	return rows
}

func kitInt(s string) int {
	n, _ := strconv.Atoi(strings.TrimSpace(s))

	return n
}

// TestStartingKitUsableByEveryClass builds the kit of all seven classes from the real CharStats.txt
// and checks that every worn item meets its strength, dexterity and level requirement at the class'
// starting attributes (no red slots), that nothing else is worn, and that the belt holds the potions.
func TestStartingKitUsableByEveryClass(t *testing.T) {
	type base struct{ w, reqStr, reqDex, reqLvl int }

	bases := map[string]base{}

	for _, file := range []string{"armor.txt", "weapons.txt", "misc.txt"} {
		for _, r := range tsvRows(t, file) {
			if r["code"] != "" {
				bases[r["code"]] = base{kitInt(r["invwidth"]), kitInt(r["reqstr"]), kitInt(r["reqdex"]), kitInt(r["levelreq"])}
			}
		}
	}

	classes := 0

	for _, r := range tsvRows(t, "CharStats.txt") {
		if r["class"] == "Expansion" || r["class"] == "" {
			continue
		}

		classes++

		var codes, locs []string

		var counts []int

		for i := 1; i <= 10; i++ {
			n := strconv.Itoa(i)
			codes = append(codes, r["item"+n])
			locs = append(locs, r["item"+n+"loc"])
			counts = append(counts, kitInt(r["item"+n+"count"]))
		}

		kit := startingKit(codes, locs, counts, func(c string) (int, bool) { b, ok := bases[c]; return b.w, ok })

		for _, w := range kit.Equipped {
			b := bases[w.Code]
			if b.reqStr > kitInt(r["str"]) || b.reqDex > kitInt(r["dex"]) || b.reqLvl > 1 {
				t.Errorf("%s wears %s (str %d dex %d lvl %d) at str %s dex %s level 1", r["class"], w.Code, b.reqStr, b.reqDex, b.reqLvl,
					r["str"], r["dex"])
			}
		}

		if len(kit.Equipped) == 0 {
			t.Errorf("%s: no weapon", r["class"])
		}

		if got := len(kit.Page(PageBelt)); got != 4 {
			t.Errorf("%s: %d potions in the belt, want the 4 of the table", r["class"], got)
		}
	}

	if classes != 7 {
		t.Errorf("%d classes in CharStats.txt, want 7", classes)
	}
}

// TestStarterWornSurvivesSave exports a new character with the Druid kit into a .d2s, reads it back and
// checks that the worn club and buckler, the belt potions and the scrolls are in the file; a second save
// of the same hero changes nothing, and a file that already holds a worn item is never given another.
func TestStarterWornSurvivesSave(t *testing.T) {
	_, tables, _ := realSave(t)

	data, err := d2s.NewCharacter("Druidy", d2s.Druid, d2s.NewCharacterFlags{Expansion: true, Created: time.Unix(1700000000, 0)},
		d2s.DefaultAppearance(d2s.Druid))
	if err != nil {
		t.Fatal(err)
	}

	kit := druidKit()
	facts := map[string]*diablo2item.ItemFacts{
		"clb": {Durability: 24, MaxDurability: 24}, "buc": {Defense: 5, Durability: 12, MaxDurability: 12},
		"hp1": {}, "tsc": {}, "isc": {},
	}

	for i := range kit.Equipped {
		kit.Equipped[i].Facts = facts[kit.Equipped[i].Code]
	}

	for i := range kit.Items {
		kit.Items[i].Facts = facts[kit.Items[i].Code]
	}

	state := &HeroState{HeroName: "Druidy", Act: 1, D2SBase: data, Containers: kit, Stats: &HeroStatsState{
		Level: 1, Strength: 15, Dexterity: 20, Vitality: 25, Energy: 20, Health: 40, MaxHealth: 40, Mana: 20, MaxMana: 20, MaxStamina: 100,
	}}
	opts := ExportOptions{Affixes: &AffixIDs{}, Known: func(string) bool { return true }}

	out, warnings, err := ExportD2SWithOptions(state, data, tables, opts)
	if err != nil {
		t.Fatal(err)
	}

	for _, w := range warnings {
		t.Errorf("warning: %s", w)
	}

	back, err := d2s.Parse(out, tables)
	if err != nil {
		t.Fatal(err)
	}

	worn := map[uint8]string{}
	belt, inv := 0, 0

	for i := range back.Items {
		it := &back.Items[i]

		switch {
		case it.Location == d2s.LocationEquipped:
			worn[it.Equipped] = strings.TrimSpace(it.Code)
		case it.Location == d2s.LocationBelt:
			belt++
		case it.Location == d2s.LocationStored && it.Page == uint8(PageInventory):
			inv++
		}
	}

	if len(worn) != 2 || worn[4] != "clb" || worn[5] != "buc" || belt != 4 || inv != 2 {
		t.Fatalf("file holds worn %v, belt %d, inventory %d", worn, belt, inv)
	}

	// the hero as the next load sees it: the worn items come from the file, and saving again changes nothing
	reloaded := &HeroState{HeroName: "Druidy", Act: 1, D2SBase: out, Stats: state.Stats, Containers: &HeroContainers{EquippedSet: true}}
	importEquipped(reloaded.Containers, out, back.Items, func(string) bool { return true })

	if len(reloaded.Containers.Equipped) != 2 {
		t.Fatalf("reloaded worn items: %v", reloaded.Containers.Equipped)
	}

	again, warnings, err := ExportD2SWithOptions(state, out, tables, opts)
	if err != nil {
		t.Fatal(err)
	}

	for _, w := range warnings {
		if !strings.Contains(w, "kept") {
			t.Errorf("second save warning: %s", w)
		}
	}

	if string(again) != string(out) {
		t.Errorf("a second save of the same hero changed the file (%d -> %d bytes)", len(out), len(again))
	}
}

// TestStarterSkillSurvivesSave: the Sorceress' short staff gives +1 to Fire Bolt (stat 107, parameter 36) and is flagged
// as a starter item; both are written to the .d2s and read back (PLAYER_CreateStartingItem 0x5326c0 / 0x5327a0).
func TestStarterSkillSurvivesSave(t *testing.T) {
	_, tables, _ := realSave(t)

	data, err := d2s.NewCharacter("Sorcy", d2s.Sorceress, d2s.NewCharacterFlags{Expansion: true, Created: time.Unix(1700000000, 0)},
		d2s.DefaultAppearance(d2s.Sorceress))
	if err != nil {
		t.Fatal(err)
	}

	kit := startingKitStacks([]string{"sst", "hp1", "tsc"}, []string{"rarm", "", ""}, []int{1, 1, 1},
		func(string) (int, bool) { return 1, true }, nil, 36)
	kit.Equipped[0].Facts = &diablo2item.ItemFacts{Durability: 20, MaxDurability: 20}
	kit.Items[0].Facts, kit.Items[1].Facts = &diablo2item.ItemFacts{}, &diablo2item.ItemFacts{}

	state := &HeroState{HeroName: "Sorcy", Act: 1, D2SBase: data, Containers: kit, Stats: &HeroStatsState{
		Level: 1, Strength: 10, Dexterity: 25, Vitality: 10, Energy: 35, Health: 40, MaxHealth: 40, Mana: 35, MaxMana: 35, MaxStamina: 74,
	}}

	out, warnings, err := ExportD2SWithOptions(state, data, tables, ExportOptions{Affixes: &AffixIDs{}, Known: func(string) bool { return true }})
	if err != nil {
		t.Fatal(err)
	}

	for _, w := range warnings {
		t.Errorf("warning: %s", w)
	}

	back, err := d2s.Parse(out, tables)
	if err != nil {
		t.Fatal(err)
	}

	found := false

	for i := range back.Items {
		it := &back.Items[i]
		if it.Location != d2s.LocationEquipped || strings.TrimSpace(it.Code) != "sst" {
			continue
		}

		found = true

		if !it.Starter {
			t.Error("the starting staff is not flagged as a starter item")
		}

		skill := false

		for _, p := range it.Properties {
			if p.ID == 107 && p.Param == 36 && p.Value == 1 {
				skill = true
			}
		}

		if !skill {
			t.Errorf("no +1 Fire Bolt on the staff: %+v", it.Properties)
		}
	}

	if !found {
		t.Fatal("the staff is not in the file")
	}
}
