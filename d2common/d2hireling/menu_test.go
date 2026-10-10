package d2hireling

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
)

func TestSellerActMapping(t *testing.T) {
	tests := []struct{ act, seller int }{
		{1, 150}, {2, 198}, {3, 252}, {4, 0}, {5, 515}, {0, 0}, {6, 0},
	}

	for _, c := range tests {
		if got := SellerOfAct(c.act); got != c.seller {
			t.Errorf("SellerOfAct(%d) = %d, want %d", c.act, got, c.seller)
		}

		if c.seller != 0 && ActOfSeller(c.seller) != c.act {
			t.Errorf("ActOfSeller(%d) = %d, want %d", c.seller, ActOfSeller(c.seller), c.act)
		}
	}

	if ActOfSeller(0) != 0 || ActOfSeller(SellerTyrael) != 0 {
		t.Error("Tyrael and 0 hire nobody")
	}
}

func TestQuoteRevive(t *testing.T) {
	tests := []struct {
		level, gold, npc int
		cost             int
		afford, allowed  bool
	}{
		{1, 0, SellerKashya, 0, true, true}, // 1*1/2 = 0
		{10, 749, SellerGreiz, 750, false, true},
		{10, 750, SellerAsheara, 750, true, true},
		{50, 18750, SellerQualKehk, 18750, true, true},
		{98, 50000, SellerTyrael, 50000, true, true}, // capped at 50000
		{30, 99999, 148, 6750, true, false},          // Akara does not revive
	}

	for _, c := range tests {
		q := QuoteRevive(c.level, c.gold, c.npc)
		if q.Cost != c.cost || q.Afford != c.afford || q.Allowed != c.allowed {
			t.Errorf("QuoteRevive(%d,%d,%d) = %+v", c.level, c.gold, c.npc, q)
		}
	}
}

func TestReplaceTerms(t *testing.T) {
	if r := ReplaceTerms(); !r.DiscardsItems || r.Refund != 0 {
		t.Errorf("replace terms %+v", r)
	}
}

func synth() *Table {
	mk := func(id, act, seller, diff, lvl int, sub string, first, last string) *Record {
		r := &Record{Version: ExpansionVersion, ID: id, Act: act, Seller: seller, Difficulty: diff, Level: lvl, SubType: sub,
			HireDesc: sub, NameFirst: first, NameLast: last, Gold: 100, ExpPerLvl: 100, HP: 100, HPPerLvl: 10,
			Defense: 50, DefLvl: 5, DmgMin: 2, DmgMax: 4, DmgLvl: 8, Resist: 10, ResistLvl: 4}
		r.Skills[0] = SkillSlot{Name: "Prayer", Mode: 7, Level: 1, LvlPerLvl: 32}
		r.Skills[1] = SkillSlot{Name: "Defiance", Mode: 7, Level: 2, LvlPerLvl: 0}

		return r
	}

	return &Table{Rows: []*Record{
		mk(0, 1, SellerKashya, 1, 3, "fire", "merc01", "merc12"),
		mk(1, 1, SellerKashya, 1, 3, "cold", "merc01", "merc12"),
		mk(0, 1, SellerKashya, 1, 25, "fire", "merc01", "merc12"),
		mk(2, 2, SellerGreiz, 1, 9, "comb", "merca201", "merca221"),
	}}
}

func TestHireListPerActDifficulty(t *testing.T) {
	tab := synth()

	tests := []struct {
		seller, diff, want int
	}{
		{SellerKashya, 1, 2}, // two SubType variants of the first band, not the level-25 row
		{SellerKashya, 2, 0},
		{SellerGreiz, 1, 1},
		{SellerAsheara, 1, 0},
	}

	for _, c := range tests {
		if got := len(tab.HireList(c.seller, c.diff)); got != c.want {
			t.Errorf("HireList(%d,%d) = %d rows, want %d", c.seller, c.diff, got, c.want)
		}
	}
}

func TestHireMenuRows(t *testing.T) {
	tab := synth()
	seed := d2rand.New(1234)
	o := tab.NewOfferTable(seed, SellerKashya, 1)

	if o == nil {
		t.Fatal("no offer table")
	}

	rows := tab.HireMenu(o, 20, nil)
	if len(rows) != OffersShown {
		t.Fatalf("rows = %d, want %d (12 names, ten offered)", len(rows), OffersShown)
	}

	seen := map[int]bool{}

	for _, r := range rows {
		if seen[r.Slot] {
			t.Errorf("slot %d listed twice", r.Slot)
		}

		seen[r.Slot] = true

		if r.Level < 15 || r.Level > 19 {
			t.Errorf("level %d, want owner level 20 minus 1..5", r.Level)
		}

		d := r.Level - 3
		if want := (d*15 + 100) * 100 / 100; r.Price != want {
			t.Errorf("price %d at level %d, want %d", r.Price, r.Level, want)
		}

		if r.HP != 100+10*d {
			t.Errorf("hp %d want %d", r.HP, 100+10*d)
		}
	}

	// hiring a slot removes it from the list
	o.Slots[rows[0].Slot].Hired = true

	if after := tab.HireMenu(o, 20, nil); len(after) != len(rows)-1 {
		t.Errorf("after a hire %d rows, want %d", len(after), len(rows)-1)
	}

	if tab.HireMenu(nil, 20, nil) != nil {
		t.Error("nil table must give no rows")
	}
}

func TestSkillsAtLevel(t *testing.T) {
	r := synth().Rows[0] // base level 3
	req := func(n string) int { return map[string]int{"Prayer": 1, "Defiance": 20}[n] }

	tests := []struct {
		mlvl int
		want []SkillAt
	}{
		{3, []SkillAt{{"Prayer", 1, 7}}},
		{19, []SkillAt{{"Prayer", 17, 7}}},
		{20, []SkillAt{{"Prayer", 18, 7}, {"Defiance", 2, 7}}},
		{80, []SkillAt{{"Prayer", 32, 7}, {"Defiance", 2, 7}}}, // capped at 32
	}

	for _, c := range tests {
		got := r.SkillsAt(c.mlvl, req)
		if len(got) != len(c.want) {
			t.Errorf("level %d: %v, want %v", c.mlvl, got, c.want)

			continue
		}

		for i := range got {
			if got[i] != c.want[i] {
				t.Errorf("level %d slot %d: %v want %v", c.mlvl, i, got[i], c.want[i])
			}
		}
	}
}

func TestProgress(t *testing.T) {
	r := &Record{ExpPerLvl: 100}
	cur, next := int(ExpThreshold(10, 100)), int(ExpThreshold(11, 100))

	tests := []struct {
		level int
		exp   uint32
		want  Progress
	}{
		{10, uint32(cur), Progress{Exp: cur, Current: cur, Next: next, Percent: 0}},
		{10, uint32(cur + (next-cur)/2), Progress{Exp: cur + (next-cur)/2, Current: cur, Next: next, Percent: 50}},
		{10, uint32(next + 5), Progress{Exp: next + 5, Current: cur, Next: next, Percent: 100}},
		{MaxLevel, 1, Progress{Exp: 1, Current: int(ExpThreshold(MaxLevel, 100)), Max: true}},
	}

	for _, c := range tests {
		if got := r.ProgressOf(c.level, c.exp); got != c.want {
			t.Errorf("level %d exp %d: %+v want %+v", c.level, c.exp, got, c.want)
		}
	}
}

// TestRealHireMenus builds the hire list of every seller and difficulty of the real table: ten rows,
// levels owner-5..owner-1, positive prices, and the act of the seller matches the rows.
func TestRealHireMenus(t *testing.T) {
	tab := loadReal(t)

	for _, seller := range []int{SellerKashya, SellerGreiz, SellerAsheara, SellerQualKehk} {
		for diff := 1; diff <= 3; diff++ {
			o := tab.NewOfferTable(d2rand.New(uint32(seller*10+diff)), seller, diff)
			if o == nil {
				t.Fatalf("seller %d diff %d: no table", seller, diff)
			}

			rows := tab.HireMenu(o, 60, nil)
			if len(rows) != OffersShown {
				t.Errorf("seller %d diff %d: %d rows", seller, diff, len(rows))
			}

			for _, r := range rows {
				if r.Act != ActOfSeller(seller) {
					t.Errorf("seller %d: row act %d, want %d", seller, r.Act, ActOfSeller(seller))
				}

				if r.Level < 55 || r.Level > 59 || r.Price <= 0 || r.HP < 40 {
					t.Errorf("seller %d diff %d: bad row %+v", seller, diff, r)
				}
			}
		}
	}
}
