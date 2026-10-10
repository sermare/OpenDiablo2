package d2equip

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

const mercTypes = miniTypes +
	"Circlet\tcirc\thelm\t\t1\t1\thead\thead\t\t\t\n" +
	"Cloak\tcloa\ttors\t\t1\t1\ttors\ttors\t\t\t\n" +
	"Spear\tspea\tmele\t\t1\t1\trarm\tlarm\t\t\t\n" +
	"Polearm\tpole\tmele\t\t1\t1\trarm\tlarm\t\t\t\n" +
	"Mace\tmace\tblun\t\t1\t1\trarm\tlarm\t\t\t\n" +
	"Potion\tpoti\tmisc\t\t0\t0\t\t\t\t\t\n" +
	"Healing Potion\thpot\tpoti\t\t0\t0\t\t\t\t\t\n" +
	"Mana Potion\tmpot\tpoti\t\t0\t0\t\t\t\t\t\n" +
	"Rejuv Potion\trpot\thpot\t\t0\t0\t\t\t\t\t\n" +
	"Stamina Potion\tspot\tpoti\t\t0\t0\t\t\t\t\t\n"

func mercRules(t *testing.T) Rules {
	t.Helper()

	ty, err := ParseTypes([]byte(mercTypes))
	if err != nil {
		t.Fatal(err)
	}

	return Rules{Types: ty}
}

func TestMercAcceptsByClass(t *testing.T) {
	r := mercRules(t)

	tests := []struct {
		name  string
		class int
		it    *Item
		want  bool
	}{
		{"rogue bow", MercRogue, weapon("sbw", "bow", true), true},
		{"rogue crossbow", MercRogue, weapon("lxb", "xbow", false), false},
		{"rogue sword", MercRogue, weapon("ssd", "swor", false), false},
		{"rogue shield", MercRogue, armor("buc", "shie"), false},
		{"rogue armor", MercRogue, armor("qui", "tors"), true},
		{"rogue cloak", MercRogue, armor("cl1", "cloa"), true},
		{"rogue helm", MercRogue, armor("cap", "helm"), true},
		{"rogue circlet", MercRogue, armor("ci0", "circ"), true},
		{"rogue belt", MercRogue, armor("lbl", "belt"), false},
		{"rogue boots", MercRogue, armor("lbt", "boot"), false},
		{"rogue gloves", MercRogue, armor("lgl", "glov"), false},
		{"rogue ring", MercRogue, armor("rin", "ring"), false},
		{"rogue amulet", MercRogue, armor("amu", "amul"), false},
		{"desert spear", MercGuard, weapon("spr", "spea", false), true},
		{"desert polearm", MercGuard, weapon("bar", "pole", true), true},
		{"desert bow", MercGuard, weapon("sbw", "bow", true), false},
		{"desert shield", MercGuard, armor("buc", "shie"), false},
		{"iron wolf shield", MercIronWolf, armor("buc", "shie"), true},
		{"iron wolf 1h sword", MercIronWolf, weapon("ssd", "swor", false), true},
		{"iron wolf 2h sword", MercIronWolf, weapon("2hs", "swor", true), false},
		{"iron wolf mace", MercIronWolf, weapon("mac", "mace", false), false},
		{"iron wolf paladin shield", MercIronWolf, armor("pa1", "ashd"), false},
		{"barbarian 2hs any sword", MercBarbarian2, weapon("2hs", "swor", true), true},
		{"barbarian 2hs 1h sword", MercBarbarian2, weapon("ssd", "swor", false), true},
		{"barbarian 2hs shield", MercBarbarian2, armor("buc", "shie"), false},
		{"barbarian 2hs axe", MercBarbarian2, weapon("hax", "axe", false), false},
		{"barbarian 2hs primal helm", MercBarbarian2, armor("ba1", "phlm"), true},
		{"barbarian 1hs 1h axe", MercBarbarian, weapon("hax", "axe", false), true},
		{"barbarian 1hs 2h axe", MercBarbarian, weapon("gax", "axe", true), false},
		{"unknown class sword", 1, weapon("ssd", "swor", false), false},
		{"unknown class armor", 1, armor("qui", "tors"), true},
	}

	for _, tc := range tests {
		if got := r.MercAccepts(tc.class, tc.it); got != tc.want {
			t.Errorf("%s: MercAccepts = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestMercSlots(t *testing.T) {
	r := mercRules(t)

	tests := []struct {
		name  string
		class int
		it    *Item
		want  Loc
		ok    bool
	}{
		{"helm", MercRogue, armor("cap", "helm"), LocHead, true},
		{"torso", MercGuard, armor("qui", "tors"), LocTorso, true},
		{"bow", MercRogue, weapon("sbw", "bow", true), LocRightHand, true},
		{"iron wolf shield goes left", MercIronWolf, armor("buc", "shie"), LocLeftHand, true},
		{"iron wolf sword goes right", MercIronWolf, weapon("ssd", "swor", false), LocRightHand, true},
		{"belt has no slot", MercRogue, armor("lbl", "belt"), LocBelt, false},
		{"ring has no slot", MercRogue, armor("rin", "ring"), LocRightRing, false},
		{"amulet has no slot", MercRogue, armor("amu", "amul"), LocNeck, false},
	}

	for _, tc := range tests {
		got, ok := r.MercSlot(tc.class, tc.it)
		if got != tc.want || ok != tc.ok {
			t.Errorf("%s: MercSlot = %v,%v want %v,%v", tc.name, got, ok, tc.want, tc.ok)
		}
	}

	if len(MercLocs) != 4 || MercHasLoc(LocBelt) || MercHasLoc(LocNeck) || MercHasLoc(LocFeet) || MercHasLoc(LocGloves) ||
		MercHasLoc(LocLeftRing) || !MercHasLoc(LocHead) || !MercHasLoc(LocTorso) || !MercHasLoc(LocRightHand) || !MercHasLoc(LocLeftHand) {
		t.Errorf("MercLocs = %v", MercLocs)
	}
}

func TestMercGiveRequirements(t *testing.T) {
	r := mercRules(t)
	merc := Merc{Class: MercGuard, Level: 20, Str: 80, Dex: 60}

	spear := func(f func(*Item)) *Item {
		it := weapon("spr", "spea", false)
		it.ReqStr, it.ReqDex, it.ReqLevel = 50, 40, 15

		if f != nil {
			f(it)
		}

		return it
	}

	tests := []struct {
		name string
		m    Merc
		it   *Item
		want Reason
	}{
		{"ok", merc, spear(nil), ReasonOK},
		{"exact limits", Merc{Class: MercGuard, Level: 15, Str: 50, Dex: 40}, spear(nil), ReasonOK},
		{"too weak", Merc{Class: MercGuard, Level: 20, Str: 49, Dex: 60}, spear(nil), ReasonStrength},
		{"too clumsy", Merc{Class: MercGuard, Level: 20, Str: 80, Dex: 39}, spear(nil), ReasonDexterity},
		{"too low", Merc{Class: MercGuard, Level: 14, Str: 80, Dex: 60}, spear(nil), ReasonLevel},
		{"unidentified", merc, spear(func(i *Item) { i.Identified = false }), ReasonUnidentified},
		{"broken", merc, spear(func(i *Item) { i.MaxDurability, i.Durability = 30, 0 }), ReasonBroken},
		{"ethereal lowers by 10", Merc{Class: MercGuard, Level: 20, Str: 40, Dex: 30}, spear(func(i *Item) { i.Ethereal = true }), ReasonOK},
		{"requirements -20%", Merc{Class: MercGuard, Level: 20, Str: 40, Dex: 32}, spear(func(i *Item) { i.ReqPercent = -20 }), ReasonOK},
		{"zero strength fails", Merc{Class: MercGuard, Level: 20, Str: 0, Dex: 60}, spear(func(i *Item) { i.ReqStr = 0 }), ReasonStrength},
		{"wrong type", merc, weapon("sbw", "bow", true), ReasonBodyLoc},
		{"no slot", merc, armor("lbl", "belt"), ReasonBodyLoc},
	}

	for _, tc := range tests {
		got := r.MercGive(tc.m, map[Loc]*Item{}, tc.it, nil)
		if got.Reason != tc.want || got.OK != (tc.want == ReasonOK) {
			t.Errorf("%s: %+v (%s), want %s", tc.name, got.Decision, got.Detail, tc.want)
		}
	}
}

func TestMercClassItems(t *testing.T) {
	r := mercRules(t)
	phlm := armor("ba1", "phlm")
	amazon := weapon("abw", "abow", true)

	tests := []struct {
		name  string
		class int
		it    *Item
		want  Reason
	}{
		{"barbarian helm on barbarian", MercBarbarian2, phlm, ReasonOK},
		{"barbarian helm on guard", MercGuard, phlm, ReasonClass},
		{"barbarian helm on rogue", MercRogue, phlm, ReasonClass},
		{"amazon bow on rogue", MercRogue, amazon, ReasonClass},
		{"paladin shield on iron wolf", MercIronWolf, armor("pa1", "ashd"), ReasonBodyLoc},
	}

	for _, tc := range tests {
		got := r.MercGive(Merc{Class: tc.class, Level: 50, Str: 200, Dex: 200}, map[Loc]*Item{}, tc.it, nil)
		if got.Reason != tc.want {
			t.Errorf("%s: %s, want %s", tc.name, got.Reason, tc.want)
		}
	}
}

func TestMercPotions(t *testing.T) {
	r := mercRules(t)
	m := Merc{Class: MercRogue, Level: 1, Str: 10, Dex: 10}

	for typ, want := range map[string]bool{"hpot": true, "rpot": true, "mpot": false, "spot": false} {
		body := map[Loc]*Item{}
		got := r.MercGive(m, body, armor("p", typ), nil)

		if got.Consumed != want || got.OK != want || len(body) != 0 {
			t.Errorf("%s: consumed=%v ok=%v body=%v, want %v", typ, got.Consumed, got.OK, body, want)
		}
	}
}

func TestMercSwap(t *testing.T) {
	r := mercRules(t)
	oldBow := weapon("sbw", "bow", true)
	oldBow.ReqDex = 10
	newBow := weapon("lbw", "bow", true)
	newBow.ReqDex = 40
	m := Merc{Class: MercRogue, Level: 10, Str: 30, Dex: 45}

	body := map[Loc]*Item{LocRightHand: oldBow}

	got := r.MercGive(m, body, newBow, nil)
	if !got.OK || got.Returned != oldBow || body[LocRightHand] != newBow || got.Loc != LocRightHand {
		t.Fatalf("swap: %+v body=%v", got, body)
	}

	// the item being replaced gave the dexterity that the new one needs: the check
	// uses the stats without it, the swap fails and the old item stays
	body = map[Loc]*Item{LocRightHand: oldBow}
	without := func(removed *Item) Merc { m2 := m; m2.Dex -= 20; return m2 }

	got = r.MercGive(m, body, newBow, without)
	if got.OK || got.Reason != ReasonDexterity || body[LocRightHand] != oldBow || got.Returned != nil {
		t.Errorf("swap needing the old item's dexterity: %+v body=%v", got, body)
	}

	// other slots are untouched
	helm := armor("cap", "helm")
	body = map[Loc]*Item{LocRightHand: oldBow}

	if got = r.MercGive(m, body, helm, nil); !got.OK || got.Returned != nil || body[LocHead] != helm || body[LocRightHand] != oldBow {
		t.Errorf("helm: %+v body=%v", got, body)
	}

	// iron wolf: shield and sword live side by side
	wolf := Merc{Class: MercIronWolf, Level: 30, Str: 100, Dex: 100}
	body = map[Loc]*Item{}
	r.MercGive(wolf, body, weapon("ssd", "swor", false), nil)
	r.MercGive(wolf, body, armor("buc", "shie"), nil)

	if body[LocRightHand] == nil || body[LocLeftHand] == nil || len(body) != 2 {
		t.Errorf("iron wolf body = %v", body)
	}
}

// TestRealMercTables checks the rules against the real ItemTypes and hireling.txt: for
// every hireling row, a weapon type is accepted exactly when it is one of the row's
// WType1/WType2 (hireling.txt), and the Iron Wolf additionally takes shields.
func TestRealMercTables(t *testing.T) {
	dir := os.Getenv("D2_TABLES")
	if dir == "" {
		t.Skip("D2_TABLES not set")
	}

	data, err := os.ReadFile(filepath.Join(dir, "patch_d2", "ItemTypes.txt"))
	if err != nil {
		t.Skip(err)
	}

	ty, err := ParseTypes(data)
	if err != nil {
		t.Fatal(err)
	}

	r := Rules{Types: ty}

	f, err := os.Open(filepath.Join(dir, "hireling", "hireling.txt"))
	if err != nil {
		t.Skip(err)
	}
	defer f.Close()

	rd := csv.NewReader(f)
	rd.Comma, rd.FieldsPerRecord, rd.LazyQuotes = '\t', -1, true

	rows, err := rd.ReadAll()
	if err != nil {
		t.Fatal(err)
	}

	col := map[string]int{}
	for i, h := range rows[0] {
		col[h] = i
	}

	weaponTypes := []string{"bow", "xbow", "swor", "spea", "pole", "axe", "mace", "club", "hamm", "knif", "scep", "wand", "staf", "h2h", "orb"}
	seen := map[int]bool{}

	for _, row := range rows[1:] {
		if len(row) <= col["WType2"] {
			continue
		}

		class, _ := strconv.Atoi(row[col["Class"]])
		if seen[class] {
			continue
		}

		seen[class] = true

		// the exe checks 0x230 only as an axe class, but hireling.txt only uses 0x231
		want := map[string]bool{row[col["WType1"]]: true}
		if w2 := strings.TrimSpace(row[col["WType2"]]); w2 != "" {
			want[w2] = true
		}

		for _, wt := range weaponTypes {
			it := &Item{Code: wt, Type: wt, Weapon: true, Identified: true}
			if got := r.MercAccepts(class, it); got != want[wt] {
				t.Errorf("class %d (%s): weapon type %s accepted=%v, hireling.txt WType says %v", class, row[col["Hireling"]], wt, got, want[wt])
			}
		}

		if got, want := r.MercAccepts(class, &Item{Code: "buc", Type: "shie", Identified: true}), want["shie"]; got != want {
			t.Errorf("class %d: shield accepted=%v, WType says %v", class, got, want)
		}

		for _, other := range []string{"boot", "glov", "belt", "ring", "amul"} {
			if r.MercAccepts(class, &Item{Type: other, Identified: true}) {
				t.Errorf("class %d accepts %s", class, other)
			}
		}

		for typ, ok := range map[string]bool{"cloa": true, "circ": true, "helm": true, "tors": true, "pelt": true, "phlm": true} {
			if got := r.MercAccepts(class, &Item{Type: typ, Identified: true}); got != ok {
				t.Errorf("class %d type %s accepted=%v", class, typ, got)
			}
		}

		// class items: primal helms only on the barbarian
		if got, want := r.MercClassOK(class, &Item{Type: "phlm"}), class == MercBarbarian2; got != want {
			t.Errorf("class %d: primal helm class ok = %v", class, got)
		}

		if r.MercClassOK(class, &Item{Type: "pelt"}) || r.MercClassOK(class, &Item{Type: "ashd"}) || r.MercClassOK(class, &Item{Type: "abow"}) {
			t.Errorf("class %d takes another class's items", class)
		}
	}

	// javelins have Equiv2 = spea in ItemTypes.txt, so the Desert Guard accepts them if the
	// exe's ITEM_IsOfType follows Equiv2 as IsA does (UNVERIFIED, a data consequence)
	if !r.MercAccepts(MercGuard, &Item{Type: "jave"}) || r.MercAccepts(MercRogue, &Item{Type: "jave"}) {
		t.Errorf("javelin acceptance")
	}

	for _, c := range []int{MercRogue, MercGuard, MercIronWolf, MercBarbarian2} {
		if !seen[c] {
			t.Errorf("hireling.txt has no class %d", c)
		}
	}
}
