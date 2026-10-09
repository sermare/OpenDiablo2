package d2cube

import (
	"strings"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/d2drop"
)

// Tiny hand written tables (not game data) so the engine is tested without
// D2_TABLES.

func tsvOf(rows ...string) []byte { return []byte(strings.Join(rows, "\n") + "\n") }

func synth() (*Catalog, *Table, *Runewords) {
	types := tsvOf("ItemType\tCode\tEquiv1\tEquiv2\tMaxSock1\tMaxSock25\tMaxSock40",
		"Any Armor\tarmo\t\t\t0\t0\t0",
		"Shield\tshld\tarmo\t\t3\t3\t4",
		"Weapon\tweap\t\t\t0\t0\t0",
		"Sword\tswor\tweap\t\t3\t4\t6",
		"Socket Filler\tsock\t\t\t0\t0\t0",
		"Gem\tgem\tsock\t\t0\t0\t0",
		"Chipped Gem\tgem0\tgem\t\t0\t0\t0",
		"Flawed Gem\tgem1\tgem\t\t0\t0\t0",
		"Rune\trune\tsock\t\t0\t0\t0")
	weapons := tsvOf("name\ttype\ttype2\tcode\tnormcode\tubercode\tultracode\tgemsockets\tlevel\tspawnable",
		"Short Sword\tswor\t\tssd\tssd\tcrs\tbsd\t6\t5\t1",
		"Crystal Sword\tswor\t\tcrs\tssd\tcrs\tbsd\t6\t30\t1",
		"Blade\tswor\t\tbsd\tssd\tcrs\tbsd\t6\t60\t1")
	armor := tsvOf("name\ttype\ttype2\tcode\tnormcode\tubercode\tultracode\tgemsockets\tlevel\tspawnable",
		"Kite\tshld\t\tkit\tkit\tkit\tkit\t3\t10\t1")
	misc := tsvOf("name\ttype\ttype2\tcode\tgemsockets\tlevel\tspawnable\tstackable\tmaxstack",
		"Chipped Amethyst\tgema\tgem0\tgcv\t0\t1\t0\t0\t0",
		"Flawed Amethyst\tgema\tgem1\tgfv\t0\t1\t0\t0\t0",
		"El Rune\trune\t\tr01\t0\t11\t0\t0\t0",
		"Eld Rune\trune\t\tr02\t0\t11\t0\t0\t0",
		"Tir Rune\trune\t\tr03\t0\t11\t0\t0\t0")
	cat := NewCatalog(types, weapons, armor, misc)

	cm := tsvOf("description\tenabled\tladder\tmin diff\tversion\top\tinput 1\tinput 2\toutput\tlvl\tplvl\tilvl\tmod 1\tmod 1 min\tmod 1 max",
		"gem up\t1\t\t\t0\t\t\"gcv,qty=3\"\t\tgfv\t\t\t\t\t\t",
		"rune up\t1\t\t\t100\t\t\"r01,qty=3\"\t\tr02\t\t\t\t\t\t",
		"ladder rune\t1\t1\t\t100\t\t\"r02,qty=2\"\t\tr03\t\t\t\t\t\t",
		"nm only\t1\t\t1\t0\t\tssd\tgcv\tcrs\t\t\t\t\t\t",
		"disabled\t\t\t\t0\t\tkit\tgcv\tcrs\t\t\t\t\t\t",
		"sock\t1\t\t\t0\t\t\"swor,mag\"\tgfv\t\"usetype,mag\"\t10\t100\t100\t\"sock\"\t2\t2",
		"upgrade\t1\t\t\t100\t\t\"swor,bas\"\tr01\t\"useitem,mod,exc\"\t\t\t\t\t\t",
		"quest\t1\t\t\t0\t28\t\"kit\"\t\t\"Cow Portal\"\t\t\t\t\t\t")
	// the quest row above uses the output column for the portal name
	tab := ParseTable(cm, cat)

	rw := ParseRunewords(tsvOf("Name\tRune Name\tcomplete\tserver\titype1\titype2\tetype1\tRune1\tRune2",
		"Runeword1\tSteel\t1\t\tswor\t\t\tr01\tr02",
		"Runeword2\tLadderOnly\t1\t1\tshld\t\t\tr01\tr03",
		"Runeword3\tBroken\t\t\tshld\t\t\tr02\tr03",
		"Runeword4\tNotShields\t1\t\tarmo\t\tshld\tr03\tr01"))

	return cat, tab, rw
}

func TestSyntheticParse(t *testing.T) {
	_, tab, _ := synth()

	if len(tab.Skipped) != 0 {
		t.Fatalf("skipped: %v", tab.Skipped)
	}

	if len(tab.Recipes) != 8 {
		t.Fatalf("recipes: %d", len(tab.Recipes))
	}
}

func TestSyntheticGemRuneAndGates(t *testing.T) {
	cat, tab, _ := synth()
	rng := d2rand.New(1)
	lod := &Context{PlayerLevel: 30, Expansion: true, Ladder: false}

	m := cat.Find(tab, lod, []Item{{Code: "gcv"}, {Code: "gcv"}, {Code: "gcv"}})
	if m == nil {
		t.Fatal("gem upgrade not found")
	}

	res, _ := cat.Execute(m, lod, []Item{{Code: "gcv"}, {Code: "gcv"}, {Code: "gcv"}}, rng)
	if res.Products[0].Item.Code != "gfv" || len(res.Consumed) != 3 {
		t.Errorf("%+v", res)
	}

	three := []Item{{Code: "r01"}, {Code: "r01"}, {Code: "r01"}}
	if cat.Find(tab, lod, three) == nil {
		t.Error("rune upgrade not found")
	}

	classic := &Context{PlayerLevel: 30}
	if cat.Find(tab, classic, three) != nil {
		t.Error("version 100 recipe in a classic game")
	}

	two := []Item{{Code: "r02"}, {Code: "r02"}}
	if cat.Find(tab, lod, two) != nil {
		t.Error("ladder recipe offline")
	}

	lad := *lod
	lad.Ladder = true

	if cat.Find(tab, &lad, two) == nil {
		t.Error("ladder recipe not found in a ladder game")
	}

	// min diff
	nm := []Item{{Code: "ssd"}, {Code: "gcv"}}
	if cat.Find(tab, lod, nm) != nil {
		t.Error("nightmare recipe in normal")
	}

	hell := *lod
	hell.Difficulty = 1

	if cat.Find(tab, &hell, nm) == nil {
		t.Error("nightmare recipe not found in nightmare")
	}

	// disabled recipe, wrong count and extra items never match
	if cat.Find(tab, &hell, []Item{{Code: "kit"}, {Code: "gcv"}}) != nil {
		t.Error("disabled recipe matched")
	}

	if cat.Find(tab, lod, append(three, Item{Code: "gcv"})) != nil {
		t.Error("extra item matched")
	}
}

func TestSyntheticSocketAndLevel(t *testing.T) {
	cat, tab, _ := synth()
	ctx := &Context{PlayerLevel: 50, Expansion: true}
	items := []Item{{Code: "crs", Quality: d2drop.QualityMagic, ILvl: 30}, {Code: "gfv"}}

	m := cat.Find(tab, ctx, items)
	if m == nil {
		t.Fatal("no match")
	}

	res, err := cat.Execute(m, ctx, items, d2rand.New(2))
	if err != nil {
		t.Fatal(err)
	}

	p := res.Products[0]
	if p.Item.Code != "crs" || p.Item.Sockets != 2 || !p.Roll || p.Item.Quality != d2drop.QualityMagic {
		t.Errorf("%+v", p)
	}

	// lvl 10 + 100% of the player level + 100% of the item level
	if want := 10 + 50 + 30; p.Item.ILvl != want {
		t.Errorf("ilvl %d want %d", p.Item.ILvl, want)
	}

	// the socket mod is capped by the base item's maximum at that level
	if mx := cat.MaxSockets("crs", 10); mx != 3 {
		t.Errorf("max sockets %d", mx)
	}
}

func TestSyntheticTierUpgradeKeepsIdentity(t *testing.T) {
	cat, tab, _ := synth()
	ctx := &Context{PlayerLevel: 50, Expansion: true}
	uni := Item{Code: "ssd", Quality: d2drop.QualityMagic, Prefixes: []string{"Sharp"}, Sockets: 1, Socketed: []string{"gcv"}}
	items := []Item{uni, {Code: "r01"}}

	m := cat.Find(tab, ctx, items)
	if m == nil {
		t.Fatal("no match")
	}

	res, _ := cat.Execute(m, ctx, items, d2rand.New(2))
	p := res.Products[0]

	if p.Item.Code != "crs" || p.Item.Prefixes[0] != "Sharp" || p.Item.Sockets != 1 || p.Source != 0 {
		t.Errorf("%+v", p)
	}
}

func TestSyntheticPortalOutput(t *testing.T) {
	cat, tab, _ := synth()
	ctx := &Context{PlayerLevel: 50}
	m := cat.Find(tab, ctx, []Item{{Code: "kit"}})

	if m == nil {
		t.Fatal("no match")
	}

	res, _ := cat.Execute(m, ctx, []Item{{Code: "kit"}}, d2rand.New(2))
	if res.Products[0].Portal != "Cow Portal" {
		t.Errorf("%+v", res.Products)
	}
}

func TestSyntheticRunewords(t *testing.T) {
	cat, _, rw := synth()

	sword := Item{Code: "ssd", Sockets: 2}
	if w, err := cat.Socket(&sword, "r01", rw, false); err != nil || w != nil {
		t.Fatal(w, err)
	}

	if w, err := cat.Socket(&sword, "r02", rw, false); err != nil || w == nil || w.Name != "Steel" || sword.Runeword != "Steel" {
		t.Fatalf("%v %v %+v", w, err, sword)
	}

	// ladder flag
	shield := Item{Code: "kit", Sockets: 2}
	_, _ = cat.Socket(&shield, "r01", rw, false)
	_, _ = cat.Socket(&shield, "r03", rw, false)

	if shield.Runeword != "" {
		t.Error("ladder-only runeword formed offline")
	}

	shield = Item{Code: "kit", Sockets: 2}
	_, _ = cat.Socket(&shield, "r01", rw, true)
	_, _ = cat.Socket(&shield, "r03", rw, true)

	if shield.Runeword != "LadderOnly" {
		t.Errorf("ladder runeword not formed: %q", shield.Runeword)
	}

	// incomplete and excluded-type runewords never form
	for _, c := range []struct {
		code  string
		runes []string
	}{{"kit", []string{"r02", "r03"}}, {"kit", []string{"r03", "r01"}}} {
		it := Item{Code: c.code, Sockets: 2}
		for _, r := range c.runes {
			_, _ = cat.Socket(&it, r, rw, true)
		}

		if it.Runeword != "" {
			t.Errorf("%v formed %q", c.runes, it.Runeword)
		}
	}

	// magic items cannot become runewords; errors
	magic := Item{Code: "ssd", Sockets: 2, Quality: d2drop.QualityMagic}
	_, _ = cat.Socket(&magic, "r01", rw, true)
	_, _ = cat.Socket(&magic, "r02", rw, true)

	if magic.Runeword != "" {
		t.Error("runeword in a magic item")
	}

	if _, err := cat.Socket(&magic, "r01", rw, true); err != ErrNoFreeSocket {
		t.Error(err)
	}

	if _, err := cat.Socket(&Item{Code: "ssd"}, "r01", rw, true); err != ErrNoSockets {
		t.Error(err)
	}

	if _, err := cat.Socket(&Item{Code: "ssd", Sockets: 1}, "ssd", rw, true); err != ErrNotSocketable {
		t.Error(err)
	}
}

func TestOutputLevelAndCrafted(t *testing.T) {
	cases := []struct {
		o                  Output
		player, main, want int
	}{
		{Output{PLevel: 50, ILevel: 50}, 80, 40, 60},
		{Output{Level: 30}, 80, 99, 30},
		{Output{}, 80, 45, 45},
		{Output{}, 80, 0, 80},
		{Output{Level: 90, PLevel: 100}, 80, 0, 99},
	}
	for i, c := range cases {
		if got := OutputLevel(&c.o, c.player, c.main); got != c.want {
			t.Errorf("case %d: %d want %d", i, got, c.want)
		}
	}

	rng := d2rand.New(5)
	for _, ilvl := range []int{10, 35, 55, 80} {
		floor := map[int]int{10: 1, 35: 2, 55: 3, 80: 4}[ilvl]

		for i := 0; i < 50; i++ {
			if n := CraftedAffixCount(rng, ilvl); n < floor || n > 4 {
				t.Fatalf("ilvl %d: %d affixes", ilvl, n)
			}
		}
	}
}
