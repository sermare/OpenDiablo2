package d2drop

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
)

// syntheticCreator is a tiny table set for the pure parts of slice A.
func syntheticCreator() *Creator {
	types := map[string]*ItemType{
		"weap": {Code: "weap", Ancestors: []string{"weap"}},
		"armo": {Code: "armo", Ancestors: []string{"armo"}},
		"sswd": {Code: "sswd", Ancestors: []string{"sswd", "weap"}, MaxSock1: 2, MaxSock25: 3, MaxSock40: 6},
		"bow":  {Code: "bow", Ancestors: []string{"bow", "weap"}, MaxSock1: 2, MaxSock25: 3, MaxSock40: 6},
		"shie": {Code: "shie", Ancestors: []string{"shie", "armo"}, MaxSock1: 1, MaxSock25: 2, MaxSock40: 4},
		"gold": {Code: "gold"},
	}

	for _, ty := range types {
		ty.Class = -1 // not class specific
	}

	items := []*BaseItem{
		{Code: "ssd", Type: "sswd", Durability: 20, MinDam: 2, MaxDam: 7, HasInv: true, GemSockets: 6, InvWidth: 1, InvHeight: 3},
		{Code: "bow", Type: "bow", Durability: 20, TwoHandMinDam: 3, TwoHandMaxDam: 9, NoDurability: true},
		{Code: "buc", Type: "shie", Durability: 12, MinAC: 3, MaxAC: 6, Block: 30, HasInv: true, GemSockets: 4, InvWidth: 2, InvHeight: 2},
		{Code: "gld", Type: "gold"},
	}

	it := &ItemTables{Items: items, ByCode: map[string]*BaseItem{}, Types: types}
	for i, b := range items {
		b.Class = i
		it.ByCode[b.Code] = b
	}

	return &Creator{Items: it, Quality: &QualityTables{
		Superior: []SuperiorRow{{Weapon: true}, {Armor: true, Shield: true}, {Bow: true}},
		Low:      4,
		Ratios: []RatioRow{
			{Version: 0, Ratio: Ratio{Normal: DropRatio{4, 8, 0}}},
			{Version: 1, Ratio: Ratio{Normal: DropRatio{2, 2, 0}}},
			{Version: 1, ClassSpecific: true, Ratio: Ratio{Normal: DropRatio{9, 9, 0}}},
		},
	}}
}

func TestQualityTablesRatio(t *testing.T) {
	q := syntheticCreator().Quality

	for _, tc := range []struct {
		cs, uber, lod bool
		want          int // Normal.Base, 0 for no row
	}{
		{false, false, false, 4}, // classic: the version 0 row
		{false, false, true, 2},  // Lord of Destruction: the highest version
		{true, false, true, 9},
		{true, false, false, 0}, // the classic game has no class specific rows
		{false, true, true, 0},
	} {
		r := q.ratio(tc.cs, tc.uber, tc.lod)
		got := 0

		if r != nil {
			got = r.Normal.Base
		}

		if got != tc.want {
			t.Errorf("ratio(%v,%v,%v) = %d, want %d", tc.cs, tc.uber, tc.lod, got, tc.want)
		}
	}
}

func TestSuperiorFits(t *testing.T) {
	c := syntheticCreator()
	row := func(i int) SuperiorRow { return c.Quality.Superior[i] }

	for _, tc := range []struct {
		code string
		row  int
		want bool
	}{
		{"ssd", 0, true},  // a sword takes weapon rows
		{"ssd", 1, false}, // but no armor rows
		{"buc", 1, true},  // a shield takes the armor row (and the shield flag)
		{"buc", 0, false},
		{"bow", 0, false}, // bows are not generic weapons...
		{"bow", 2, true},  // ...they take the bow column
	} {
		st := &itemState{c: c, base: c.Items.ByCode[tc.code]}
		if got := c.superiorFits(st, row(tc.row)); got != tc.want {
			t.Errorf("%s row %d: got %v, want %v", tc.code, tc.row, got, tc.want)
		}
	}
}

func TestSkillTiers(t *testing.T) {
	for _, tc := range []struct {
		ilvl int
		lod  bool
		want int
	}{{1, true, 1}, {12, true, 2}, {19, false, 3}, {25, false, 4}, {40, false, 4}, {40, true, 5}} {
		if got := skillTier(tc.ilvl, tc.lod); got != tc.want {
			t.Errorf("skillTier(%d,%v) = %d, want %d", tc.ilvl, tc.lod, got, tc.want)
		}
	}

	for _, tc := range []struct{ tier, roll, want int }{
		{3, 90, 4}, {3, 50, 3}, {3, 20, 2}, {3, 5, 1}, {1, 5, 1},
	} {
		if got := tierJitter(tc.tier, tc.roll); got != tc.want {
			t.Errorf("tierJitter(%d,%d) = %d, want %d", tc.tier, tc.roll, got, tc.want)
		}
	}
}

// rolledStat is the last unit stat written with the id.
func rolledStat(r *Rolled, id int) int {
	st := &itemState{writes: r.Writes}

	return st.stat(id)
}

func create(c *Creator, code string, ilvl int, q Quality, version int, flags RequestFlags, seed uint32) (*Rolled, error) {
	return c.Create(Request{
		Code: code, ILvl: ilvl, Quality: q, Version: version, Expansion: version >= 100,
		Flags: flags, GameSeed: d2rand.Seed{Lo: seed, Hi: 0x29a},
	})
}

func TestCreateBaseStats(t *testing.T) {
	c := syntheticCreator()

	for seed := uint32(1); seed < 200; seed++ {
		g, err := create(c, "gld", 10, QualityNormal, 100, 0, seed)
		if err != nil || len(g.Writes) != 1 || g.Writes[0].Stat != statGold {
			t.Fatalf("gold: %+v %v", g, err)
		}

		if v := g.Writes[0].Value; v < 10 || v >= 60 {
			t.Fatalf("gold amount %d outside [10,60)", v)
		}

		s, err := create(c, "buc", 10, QualityNormal, 100, 0, seed)
		if err != nil {
			t.Fatal(err)
		}

		// block, speed, durability (current, maximum), defense.
		if len(s.Writes) < 5 || s.Writes[0].Stat != statBlock || s.Writes[0].Value != 30 {
			t.Fatalf("shield writes %+v", s.Writes)
		}

		if cur, max := s.Writes[2].Value, s.Writes[3].Value; max != 12 || cur < 6 || cur > 12 {
			t.Fatalf("durability %d/%d", cur, max)
		}

		if def := s.Writes[4].Value; def < 3 || def > 6 {
			t.Fatalf("defense %d", def)
		}
	}
}

func TestCreateLowQualityAndFallbacks(t *testing.T) {
	c := syntheticCreator()

	g, err := create(c, "ssd", 20, QualityLow, 100, 0, 5)
	if err != nil || g.Quality != QualityLow {
		t.Fatalf("low quality sword: %+v %v", g, err)
	}

	// 75% of the damage, never below 2 and 1.
	if got := g.Writes[len(g.Writes)-2]; got.Stat != statSecMaxDam {
		t.Fatalf("low quality writes %+v", g.Writes)
	}

	// A magic request fails (no affix slice here) and ends superior or normal.
	m, err := create(c, "ssd", 20, QualityMagic, 100, 0, 5)
	if err != nil || (m.Quality != QualitySuperior && m.Quality != QualityNormal) {
		t.Fatalf("magic fallback: %+v %v", m, err)
	}

	// Gold cannot be low quality: it falls back to normal.
	l, err := create(c, "gld", 20, QualityLow, 100, 0, 5)
	if err != nil || l.Quality != QualityNormal {
		t.Fatalf("low quality gold: %+v %v", l, err)
	}

	// A classic game has no class specific ItemRatio row: the game stops
	// (error here), a Lord of Destruction game has one.
	c.Items.Types["sswd"].Class = 1

	if _, err := create(c, "ssd", 20, QualityNormal, 0, 0, 5); err != ErrNotCreated {
		t.Fatalf("classic class specific item: %v", err)
	}

	if _, err := create(c, "ssd", 20, QualityNormal, 100, 0, 5); err != nil {
		t.Fatal(err)
	}

	c.Items.Types["sswd"].Class = -1

	// The expansion item in a classic game is refused.
	c.Items.ByCode["ssd"].Version = 100

	if _, err := create(c, "ssd", 20, QualityNormal, 0, 0, 5); err != ErrNotCreated {
		t.Fatalf("classic game created an expansion item: %v", err)
	}
}

func TestCreateEtherealAndSockets(t *testing.T) {
	c := syntheticCreator()

	// Forced ethereal halves the durability (plus one) and adds 50% damage.
	g, err := create(c, "ssd", 30, QualityNormal, 100, FlagForceEthereal, 9)
	if err != nil {
		t.Fatal(err)
	}

	if g.Flags&flagEthereal == 0 {
		t.Fatalf("not ethereal: %#x", g.Flags)
	}

	if cur, max := rolledStat(g, statDurCur), rolledStat(g, statDurMax); max != 11 || cur != 11 {
		t.Fatalf("ethereal durability %d/%d", cur, max)
	}

	// Never ethereal when the request says so, never in a classic game.
	for _, tc := range []struct {
		v  int
		fl RequestFlags
	}{{100, FlagForceEthereal | FlagNoEthereal}, {0, FlagForceEthereal}} {
		e, err := create(c, "ssd", 30, QualityNormal, tc.v, tc.fl, 9)
		if err != nil || e.Flags&flagEthereal != 0 {
			t.Fatalf("version %d flags %#x: %+v %v", tc.v, tc.fl, e, err)
		}
	}

	// Forced sockets always give at least one, within the item's limits.
	for seed := uint32(1); seed < 100; seed++ {
		s, err := create(c, "ssd", 20, QualityNormal, 100, FlagForceSockets, seed)
		if err != nil {
			t.Fatal(err)
		}

		n := rolledStat(s, statSockets)
		if s.Flags&flagSocketed == 0 || n < 1 || n > 3 {
			t.Fatalf("sockets %d flags %#x (normal difficulty caps at 3, ilvl 20 allows 2)", n, s.Flags)
		}
	}

	if s, _ := create(c, "ssd", 20, QualityNormal, 100, FlagForceSockets|FlagNoSockets, 1); s.Flags&flagSocketed != 0 {
		t.Fatal("sockets despite the no sockets flag")
	}
}
