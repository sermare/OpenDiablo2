package diablo2item

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/d2drop"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2vendor"
)

type realisedItem struct {
	code     string
	quality  d2drop.Quality
	ilvl     int
	sockets  int
	ethereal bool
}

// realiseStock mirrors TradeWindow.realise: every generated entry goes
// through ItemFromCodeForVendor with seed+idx+1 and the window difficulty.
func realiseStock(t *testing.T, f *ItemFactory, s *d2vendor.Stock, seed uint32, diff int) []realisedItem {
	t.Helper()

	out := make([]realisedItem, 0, len(s.Items))

	for idx, e := range s.Items {
		it, err := f.ItemFromCodeForVendor(e.Code, e.Quality, e.ILvl, seed+uint32(idx)+1, diff)
		if err != nil {
			t.Fatalf("realise %q: %v", e.Code, err)
		}

		out = append(out, realisedItem{e.Code, e.Quality, it.itemLevel, it.NumSockets(), it.IsEthereal()})
	}

	return out
}

// A vendor's generated stock, realised the way the trade window does it,
// obeys the verified rules end to end and is reproducible for a fixed seed.
func TestVendorStockRealisedConsistency(t *testing.T) {
	f, _ := extrasFactory()

	bases := []d2vendor.Base{{
		Code: "cap", ReqLevel: 1, W: 2, H: 2, Gear: true, CanBeMagic: true,
		Vendor: d2vendor.Params{Min: 2, Max: 4, MagicMin: 1, MagicMax: 2, MagicLevel: 1},
	}}

	const (
		seed        = uint32(4242)
		playerLevel = 99
	)

	// Normal difficulty, act 1 vendor: item level cap 12 (VERIFIED table).
	opt := d2vendor.Options{PlayerLevel: playerLevel, Tier: 0, Difficulty: 0}

	var runs [2][]realisedItem

	for r := range runs {
		stock := d2vendor.GenerateSeeded(seed, bases, opt)
		runs[r] = realiseStock(t, f, stock, seed, 0)

		regular, magic := 0, 0

		for _, e := range stock.Items {
			if e.ILvl != 12 {
				t.Fatalf("stock ilvl %d, want the act 1 cap 12", e.ILvl)
			}

			if e.Quality == d2drop.QualityMagic {
				magic++
			} else {
				regular++
			}
		}

		// ilvl 12 < 25: regular pass runs, magic extra is 1 (exclusive bound).
		if regular < 2 || regular > 4 || magic < 1 || magic > 2 {
			t.Fatalf("counts regular=%d magic=%d outside [2,4]/[1,2]", regular, magic)
		}
	}

	if len(runs[0]) == 0 || len(runs[0]) != len(runs[1]) {
		t.Fatalf("stock sizes differ between runs: %d vs %d", len(runs[0]), len(runs[1]))
	}

	for i := range runs[0] {
		if runs[0][i] != runs[1][i] {
			t.Fatalf("item %d differs between identical runs: %+v vs %+v", i, runs[0][i], runs[1][i])
		}

		if runs[0][i].ethereal {
			t.Fatalf("vendor item %d ethereal", i)
		}

		if runs[0][i].ilvl != 12 || runs[0][i].sockets > 2 {
			t.Fatalf("item %d: ilvl %d sockets %d", i, runs[0][i].ilvl, runs[0][i].sockets)
		}
	}

	// Sockets must be possible: over many seeds some shop caps are socketed.
	socketed := 0

	for s := uint32(1); s <= 200; s++ {
		for _, it := range realiseStock(t, f, d2vendor.GenerateSeeded(s, bases, opt), s, 0) {
			if it.ethereal {
				t.Fatalf("seed %d: ethereal vendor item", s)
			}

			if it.ilvl > 12 {
				t.Fatalf("seed %d: ilvl %d above cap", s, it.ilvl)
			}

			if it.sockets > 0 {
				socketed++
			}
		}
	}

	if socketed == 0 {
		t.Error("no socketed vendor item over 200 stocks")
	}

	// Nightmare: the cap table no longer applies (VERIFIED), ilvl = level+5.
	nm := d2vendor.GenerateSeeded(seed, bases, d2vendor.Options{PlayerLevel: playerLevel, Tier: 0, Difficulty: 1})
	for _, it := range realiseStock(t, f, nm, seed, 1) {
		if it.ilvl != playerLevel+5 || it.ethereal {
			t.Fatalf("nightmare item ilvl %d ethereal %v", it.ilvl, it.ethereal)
		}
	}

	// Restock rule: strictly more than 240000 ms (VERIFIED).
	if d2vendor.RestockDue(d2vendor.RestockInterval) || !d2vendor.RestockDue(d2vendor.RestockInterval+1) {
		t.Error("restock must be due strictly after RestockInterval")
	}

	if d2vendor.RestockInterval != 240000 {
		t.Errorf("restock interval %d, want 240000", d2vendor.RestockInterval)
	}
}
