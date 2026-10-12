package d2hero

import (
	"fmt"
	"strings"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2statlist"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2records"
)

// The new-character audit (w44): what a hero made in the menu starts with, against CharStats.txt and the exe's
// new-character code (PLAYER_InitBaseStatsFromClass 0x56e5d0: str/int/dex/vit from the row, life and max life
// (hpadd+vit), mana = int, stamina = stamina column, level 1, 0 unspent points; PLAYER_GiveStartingItems 0x532960 ->
// PLAYER_GiveStartingItemEntry 0x5327a0: items item1..item10 with their count, worn at item<n>loc, belt-able types to
// the belt, the rest to the inventory; PLAYER_CreateStartingItem 0x5326c0: stackables get a full stack).

// newcharWant is one class' expectation, written from the original table (and checked against it below).
type newcharWant struct {
	class                                   string
	life, mana, stamina, str, dex, vit, ene int
	worn                                    []string // item codes in body order (rarm first, then larm)
	quantity                                map[string]int
	beltPotions, inventoryItems             int
	startSkill                              string
	startSkillID                            int // skills.txt id the first starting item gives +1 to (0 none)
}

var newcharWants = []newcharWant{
	{class: "Amazon", life: 50, mana: 15, stamina: 84, str: 20, dex: 25, vit: 20, ene: 15, worn: []string{"jav", "buc"},
		quantity: map[string]int{"jav": 60}, beltPotions: 4, inventoryItems: 2},
	{class: "Sorceress", life: 40, mana: 35, stamina: 74, str: 10, dex: 25, vit: 10, ene: 35, worn: []string{"sst"},
		beltPotions: 4, inventoryItems: 2, startSkill: "fire bolt", startSkillID: 36},
	{class: "Necromancer", life: 45, mana: 25, stamina: 79, str: 15, dex: 25, vit: 15, ene: 25, worn: []string{"wnd"},
		beltPotions: 4, inventoryItems: 2, startSkill: "raise skeleton", startSkillID: 70},
	{class: "Paladin", life: 55, mana: 15, stamina: 89, str: 25, dex: 20, vit: 25, ene: 15, worn: []string{"ssd", "buc"},
		beltPotions: 4, inventoryItems: 2},
	{class: "Barbarian", life: 55, mana: 10, stamina: 92, str: 30, dex: 20, vit: 25, ene: 10, worn: []string{"hax", "buc"},
		beltPotions: 4, inventoryItems: 2},
	{class: "Druid", life: 55, mana: 20, stamina: 84, str: 15, dex: 20, vit: 25, ene: 20, worn: []string{"clb", "buc"},
		beltPotions: 4, inventoryItems: 2},
	{class: "Assassin", life: 50, mana: 25, stamina: 95, str: 20, dex: 20, vit: 20, ene: 25, worn: []string{"ktr", "buc"},
		beltPotions: 4, inventoryItems: 2},
}

// TestNewCharStatsAllClasses checks the level 1 life, mana and stamina of every class against PLAYER_InitBaseStatsFromClass
// (life (hpadd+vit), mana int, stamina column) with the real CharStats.txt, and against the values written above.
func TestNewCharStatsAllClasses(t *testing.T) {
	rows := map[string]map[string]string{}
	for _, r := range tsvRows(t, "CharStats.txt") {
		rows[r["class"]] = r
	}

	for _, w := range newcharWants {
		r := rows[w.class]
		if r == nil {
			t.Errorf("%s: no CharStats.txt row", w.class)
			continue
		}

		c := d2statlist.Class{
			InitStr: kitInt(r["str"]), InitDex: kitInt(r["dex"]), InitVit: kitInt(r["vit"]), InitEne: kitInt(r["int"]),
			InitStamina: kitInt(r["stamina"]), HpAdd: kitInt(r["hpadd"]),
			LifePerVit: kitInt(r["LifePerVitality"]), ManaPerEne: kitInt(r["ManaPerMagic"]), StaminaPerVit: kitInt(r["StaminaPerVitality"]),
			LifePerLevel: kitInt(r["LifePerLevel"]), ManaPerLevel: kitInt(r["ManaPerLevel"]), StaminaPerLevel: kitInt(r["StaminaPerLevel"]),
		}

		life, mana, stam := c.BaseMax(1, c.InitVit, c.InitEne)
		if life != c.HpAdd+c.InitVit || mana != c.InitEne || stam != c.InitStamina {
			t.Errorf("%s: level 1 life/mana/stamina %d/%d/%d, the exe has %d/%d/%d", w.class, life, mana, stam,
				c.HpAdd+c.InitVit, c.InitEne, c.InitStamina)
		}

		got := fmt.Sprint(life, mana, stam, c.InitStr, c.InitDex, c.InitVit, c.InitEne)
		want := fmt.Sprint(w.life, w.mana, w.stamina, w.str, w.dex, w.vit, w.ene)

		if got != want {
			t.Errorf("%s: life mana stamina str dex vit ene = %s, want %s", w.class, got, want)
		}

		if got := strings.ToLower(strings.TrimSpace(r["StartSkill"])); got != w.startSkill {
			t.Errorf("%s: StartSkill %q, want %q", w.class, got, w.startSkill)
		}
	}
}

// TestNewCharKitAllClasses builds each class' kit with the real tables (invwidth, stackable and maxstack of the base
// items) and checks the worn items and their stack, the four belt potions and the two scrolls and their cells.
func TestNewCharKitAllClasses(t *testing.T) {
	type base struct {
		w, max int
		stack  bool
	}

	bases := map[string]base{}

	for _, file := range []string{"armor.txt", "weapons.txt", "misc.txt"} {
		for _, r := range tsvRows(t, file) {
			if r["code"] != "" {
				bases[r["code"]] = base{kitInt(r["invwidth"]), kitInt(r["maxstack"]), kitInt(r["stackable"]) > 0}
			}
		}
	}

	rows := map[string]map[string]string{}
	for _, r := range tsvRows(t, "CharStats.txt") {
		rows[r["class"]] = r
	}

	for _, w := range newcharWants {
		r := rows[w.class]

		var codes, locs []string

		var counts []int

		for i := 1; i <= 10; i++ {
			n := fmt.Sprint(i)
			codes, locs, counts = append(codes, r["item"+n]), append(locs, r["item"+n+"loc"]), append(counts, kitInt(r["item"+n+"count"]))
		}

		kit := startingKitStacks(codes, locs, counts, func(c string) (int, bool) { b, ok := bases[c]; return b.w, ok },
			func(c string) int {
				if b := bases[c]; b.stack {
					return b.max
				}

				return 0
			}, w.startSkillID)

		var worn []string
		for _, e := range kit.Equipped {
			worn = append(worn, e.Code)

			if want := w.quantity[e.Code]; e.Quantity != want {
				t.Errorf("%s: %s quantity %d, want %d (a full stack for a stackable, 0 otherwise)", w.class, e.Code, e.Quantity, want)
			}
		}

		for i, e := range append(append([]StoredItem{}, kit.Equipped...), kit.Items...) {
			// every starting item is flagged (0x20000); only the first row's item gives the start skill
			want := 0
			if e.Code == w.worn[0] && i == 0 {
				want = w.startSkillID
			}

			if !e.Starter || e.StartSkill != want {
				t.Errorf("%s: %s starter=%v startSkill=%d, want true/%d", w.class, e.Code, e.Starter, e.StartSkill, want)
			}
		}

		if strings.Join(worn, ",") != strings.Join(w.worn, ",") {
			t.Errorf("%s worn %v, want %v", w.class, worn, w.worn)
		}

		if got := len(kit.Page(PageBelt)); got != w.beltPotions {
			t.Errorf("%s: %d belt potions, want %d", w.class, got, w.beltPotions)
		}

		inv := kit.Page(PageInventory)
		if len(inv) != w.inventoryItems {
			t.Errorf("%s: %d inventory items, want %d", w.class, len(inv), w.inventoryItems)
		}

		// tome and scroll of town portal / identify, one each, side by side from the top left
		for i, want := range []string{"tsc", "isc"} {
			if i < len(inv) && (inv[i].Code != want || inv[i].X != i || inv[i].Y != 0) {
				t.Errorf("%s: inventory item %d is %s at %d,%d, want %s at %d,0", w.class, i, inv[i].Code, inv[i].X, inv[i].Y, want, i)
			}
		}

		for i, b := range kit.Page(PageBelt) {
			if b.Code != "hp1" || b.X != i {
				t.Errorf("%s: belt cell %d holds %s at %d, want hp1 at %d", w.class, i, b.Code, b.X, i)
			}
		}
	}
}

// TestStartingKitStackQuantity: a stackable starter gets the full stack, others keep 0 (no quantity).
func TestStartingKitStackQuantity(t *testing.T) {
	maxStack := func(c string) int {
		if c == "jav" {
			return 60
		}

		return 0
	}
	kit := startingKitStacks([]string{"jav", "buc", "hp1"}, []string{"rarm", "larm", ""}, []int{1, 1, 1},
		func(string) (int, bool) { return 1, true }, maxStack, 0)

	if len(kit.Equipped) != 2 || kit.Equipped[0].Quantity != 60 || kit.Equipped[1].Quantity != 0 {
		t.Errorf("worn %+v", kit.Equipped)
	}

	if b := kit.Page(PageBelt); len(b) != 1 || b[0].Quantity != 0 {
		t.Errorf("belt %+v", b)
	}
}

// TestSelectableByItemLevel: a skill the hero has no point in can be put on a button when its items give it (the
// Sorceress' staff gives Fire Bolt), a passive or an unknown skill cannot.
func TestSelectableByItemLevel(t *testing.T) {
	rec := &d2records.SkillRecord{}

	for _, c := range []struct {
		name string
		s    *HeroSkill
		want bool
	}{
		{"points", &HeroSkill{SkillRecord: rec, SkillPoints: 1}, true},
		{"item level only", &HeroSkill{SkillRecord: rec, ItemLevel: 1}, true},
		{"neither", &HeroSkill{SkillRecord: rec}, false},
		{"nil", nil, false},
	} {
		if got := Selectable(c.s, false); got != c.want {
			t.Errorf("%s: Selectable = %v, want %v", c.name, got, c.want)
		}
	}
}
