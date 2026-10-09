package d2drop

import (
	"reflect"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
)

type mapItems map[string]*ItemInfo

func (m mapItems) Item(code string) (*ItemInfo, bool) { i, ok := m[code]; return i, ok }

type fixedRatio struct{ r *Ratio }

func (f fixedRatio) ItemRatio(bool, bool) (*Ratio, bool) { return f.r, true }

func testTable() *TreasureTable {
	t := NewTreasureTable()
	t.Add(&TreasureClass{Name: "Leaf", Picks: 1, Entries: []Entry{{Code: "cap", Prob: 1}}})
	t.Add(&TreasureClass{Name: "Mid", Picks: 2, Mods: QualityMods{Unique: 100, Magic: 900},
		Entries: []Entry{{Code: "Leaf", Prob: 1}}})
	t.Add(&TreasureClass{Name: "Top", Picks: 1, Mods: QualityMods{Unique: 300, Magic: 10},
		Entries: []Entry{{Code: "Mid", Prob: 1}}})
	t.Add(&TreasureClass{Name: "Grp A", Group: 5, Level: 0, Picks: 1})
	t.Add(&TreasureClass{Name: "Grp B", Group: 5, Level: 20, Picks: 1})
	t.Add(&TreasureClass{Name: "Grp C", Group: 5, Level: 40, Picks: 1})
	t.Add(&TreasureClass{Name: "Other", Group: 6, Level: 10, Picks: 1})

	return t
}

func TestUpgrade(t *testing.T) {
	tab := testTable()
	a, _ := tab.TreasureClass("Grp A")

	for _, c := range []struct {
		lvl  int
		want string
	}{{0, "Grp A"}, {19, "Grp A"}, {20, "Grp B"}, {39, "Grp B"}, {99, "Grp C"}} {
		if got := tab.Upgrade(a, c.lvl).Name; got != c.want {
			t.Errorf("level %d -> %s, want %s", c.lvl, got, c.want)
		}
	}

	o, _ := tab.TreasureClass("Other")
	if tab.Upgrade(o, 50) != o {
		t.Error("a different group must not be walked into")
	}
}

func TestNestedModsAndItemLevel(t *testing.T) {
	d := &Dropper{TCs: testTable()}
	rng := d2rand.New(1)

	drops, err := d.Roll(&Context{RNG: rng, ILvl: 77}, "Top")
	if err != nil {
		t.Fatal(err)
	}

	if len(drops) != 2 {
		t.Fatalf("Top -> Mid(2 picks) -> Leaf = 2 items, got %d", len(drops))
	}

	for _, dr := range drops {
		if dr.ILvl != 77 || dr.Code != "cap" {
			t.Errorf("drop %+v", dr)
		}

		// max-merged down the stack: unique max(300,100), magic max(10,900)
		if dr.Mods.Unique != 300 || dr.Mods.Magic != 900 {
			t.Errorf("mods %+v", dr.Mods)
		}
	}
}

func TestMaxDropsCap(t *testing.T) {
	tab := NewTreasureTable()
	tab.Add(&TreasureClass{Name: "Many", Picks: 20, Entries: []Entry{{Code: "cap", Prob: 1}}})

	d := &Dropper{TCs: tab}
	drops, _ := d.Roll(&Context{RNG: d2rand.New(3)}, "Many")

	if len(drops) != DefaultMaxDrops {
		t.Errorf("got %d drops, want cap %d", len(drops), DefaultMaxDrops)
	}
}

func TestNegativePicksDeterministic(t *testing.T) {
	tab := NewTreasureTable()
	tab.Add(&TreasureClass{Name: "Det", Picks: -4, Entries: []Entry{
		{Code: "aaa", Prob: 2}, {Code: "bbb", Prob: 5}, {Code: "ccc", Prob: 1},
	}})

	d := &Dropper{TCs: tab}
	rng := &scriptRNG{}
	drops, _ := d.Roll(&Context{RNG: rng}, "Det")

	var codes []string
	for _, dr := range drops {
		codes = append(codes, dr.Code)
	}

	if want := []string{"aaa", "aaa", "bbb", "bbb"}; !reflect.DeepEqual(codes, want) {
		t.Errorf("got %v want %v", codes, want)
	}

	if len(rng.asked) != 0 {
		t.Error("deterministic mode must not roll")
	}
}

func TestNoDrop(t *testing.T) {
	tab := NewTreasureTable()
	tab.Add(&TreasureClass{Name: "N", Picks: 1, NoDrop: 3, Entries: []Entry{{Code: "aaa", Prob: 1}}})

	d := &Dropper{TCs: tab}
	// space 4: rolls 0..2 are no drop, 3 is the item.
	for r, want := range []int{0, 0, 0, 1} {
		drops, _ := d.Roll(&Context{RNG: &scriptRNG{vals: []uint32{uint32(r)}}}, "N")
		if len(drops) != want {
			t.Errorf("roll %d -> %d drops", r, len(drops))
		}
	}
}

func TestEffectiveNoDrop(t *testing.T) {
	if EffectiveNoDrop(10, 30, 1) != 10 || EffectiveNoDrop(0, 30, 8) != 0 {
		t.Error("one player / no nodrop must be unchanged")
	}

	// f = 10/40 = 1/4; n=2: 30*(1/16)/(15/16) = 2.
	if got := EffectiveNoDrop(10, 30, 2); got != 2 {
		t.Errorf("got %d", got)
	}

	if EffectiveNoDrop(10, 30, 8) >= EffectiveNoDrop(10, 30, 2) {
		t.Error("more players must mean fewer no-drops")
	}
}

func TestParseEntry(t *testing.T) {
	e := ParseEntry(`"gld,mul=1280"`, 11)
	if e.Code != "gld" || e.Mul != 1280 || e.Prob != 11 {
		t.Errorf("%+v", e)
	}

	e = ParseEntry("weap3,cu=800,cs=700,cr=600,cm=500,ce=2,cg=3", 1)
	if e.Mods != (QualityMods{Magic: 500, Rare: 600, Set: 700, Unique: 800, E: 2, G: 3}) {
		t.Errorf("%+v", e.Mods)
	}
}

func TestBuildTypeTreasureClasses(t *testing.T) {
	items := []*ItemInfo{
		{Code: "lvl3", Level: 3, Types: []string{"armo"}, Spawnable: true, Rarity: 2},
		{Code: "lvl4", Level: 4, Types: []string{"armo"}, Spawnable: true},
		{Code: "lvl6", Level: 6, Types: []string{"armo", "helm"}, Spawnable: true, Rarity: 5},
		{Code: "quest", Level: 5, Types: []string{"armo"}, Spawnable: true, Quest: true},
		{Code: "nosp", Level: 5, Types: []string{"armo"}},
		{Code: "wpn", Level: 5, Types: []string{"weap"}, Spawnable: true},
	}

	tcs := BuildTypeTreasureClasses(items, []string{"armo"})
	if len(tcs) != 32 {
		t.Fatalf("got %d classes", len(tcs))
	}

	byName := map[string]*TreasureClass{}
	for _, tc := range tcs {
		byName[tc.Name] = tc
	}

	// (N-3, N]: level 3 belongs to armo3 (0 < 3 <= 3), not armo6.
	armo3, armo6 := byName["armo3"], byName["armo6"]

	if len(armo3.Entries) != 1 || armo3.Entries[0].Code != "lvl3" || armo3.Entries[0].Prob != 2 {
		t.Errorf("armo3 = %+v", armo3.Entries)
	}

	var codes []string
	for _, e := range armo6.Entries {
		codes = append(codes, e.Code)
	}

	if want := []string{"lvl4", "lvl6"}; !reflect.DeepEqual(codes, want) {
		t.Errorf("armo6 = %v", codes)
	}

	if armo6.Entries[0].Prob != 1 {
		t.Error("probability is at least 1")
	}

	if armo6.Level != 3 || armo6.Picks != 1 || byName["armo96"] == nil || byName["armo99"] != nil {
		t.Error("generated class shape")
	}
}

func TestRollDeterministicBySeed(t *testing.T) {
	tab := testTable()
	items := mapItems{"cap": {Code: "cap", Level: 1, TypeRare: true}}
	d := &Dropper{TCs: tab, Items: items, Ratios: fixedRatio{testRatio}}

	roll := func(seed uint32) []Drop {
		out, err := d.Roll(&Context{RNG: d2rand.New(seed), ILvl: 80, MagicFind: 200}, "Top")
		if err != nil {
			t.Fatal(err)
		}

		return out
	}

	if !reflect.DeepEqual(roll(42), roll(42)) {
		t.Error("same seed must give the same drops")
	}
}

func TestUnknownAndCycle(t *testing.T) {
	tab := NewTreasureTable()
	tab.Add(&TreasureClass{Name: "Loop", Picks: 1, Entries: []Entry{{Code: "Loop", Prob: 1}}})

	d := &Dropper{TCs: tab}

	if _, err := d.Roll(&Context{RNG: d2rand.New(1)}, "Nope"); err == nil {
		t.Error("unknown class should error")
	}

	if _, err := d.Roll(&Context{RNG: d2rand.New(1)}, "Loop"); err == nil {
		t.Error("cycle should hit the nesting limit")
	}
}
