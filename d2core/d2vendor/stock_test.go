package d2vendor

import (
	"reflect"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/d2drop"
)

// seqRNG returns scripted rolls (reduced modulo n) and records the moduli.
type seqRNG struct {
	vals []uint32
	i    int
	mods []int32
}

func (s *seqRNG) Roll(n int32) uint32 {
	s.mods = append(s.mods, n)

	if n < 1 {
		return 0
	}

	v := s.vals[s.i%len(s.vals)]
	s.i++

	return v % uint32(n)
}

func (s *seqRNG) Chance() bool { return false }

func TestItemLevel(t *testing.T) {
	tests := []struct{ level, tier, want int }{
		{1, -1, 6},
		{30, -1, 35},
		{30, 0, 12},
		{30, 1, 20},
		{30, 2, 28},
		{30, 3, 35}, // 35 < 36: below the cap
		{40, 3, 36},
		{80, 4, 45},
		{3, 0, 8}, // below the cap stays
		{10, 9, 15},
	}

	for _, tt := range tests {
		if got := ItemLevel(tt.level, tt.tier); got != tt.want {
			t.Errorf("ItemLevel(%d,%d)=%d want %d", tt.level, tt.tier, got, tt.want)
		}
	}
}

func TestRollQuality(t *testing.T) {
	tests := []struct {
		name string
		ilvl int
		roll uint32
		want d2drop.Quality
	}{
		{"ilvl 3 roll 90 normal", 3, 90, d2drop.QualityNormal},
		{"ilvl 3 roll 91 low", 3, 91, d2drop.QualityLow},
		{"ilvl 7 roll 85 normal", 7, 85, d2drop.QualityNormal},
		{"ilvl 7 roll 86 superior", 7, 86, d2drop.QualitySuperior},
		{"ilvl 9 roll 99 superior", 9, 99, d2drop.QualitySuperior},
		{"ilvl 10 roll 74 normal", 10, 74, d2drop.QualityNormal},
		{"ilvl 10 roll 75 superior", 10, 75, d2drop.QualitySuperior},
		{"ilvl 40 roll 0 normal", 40, 0, d2drop.QualityNormal},
	}

	for _, tt := range tests {
		r := &seqRNG{vals: []uint32{tt.roll}}
		if got := RollQuality(r, tt.ilvl); got != tt.want {
			t.Errorf("%s: got %v want %v", tt.name, got, tt.want)
		}
	}
}

func testBases() []Base {
	return []Base{
		{Code: "aqv", Ammo: true, MaxStack: 350, W: 1, H: 2, Permanent: true, Vendor: Params{Min: 3, Max: 5, MagicLevel: 255}},
		{Code: "cap", ReqLevel: 1, W: 2, H: 2, Gear: true, CanBeMagic: true, Vendor: Params{Min: 1, Max: 2, MagicMin: 1, MagicMax: 2, MagicLevel: 5}},
		{Code: "plt", ReqLevel: 40, W: 2, H: 3, Gear: true, CanBeMagic: true, Vendor: Params{Max: 1, MagicMax: 1, MagicLevel: 1}},
		{Code: "hp1", W: 1, H: 1, Permanent: true, Vendor: Params{Min: 1, Max: 1, MagicLevel: 255}},
	}
}

func TestGenerateDeterministic(t *testing.T) {
	a := GenerateSeeded(42, testBases(), Options{PlayerLevel: 20, Tier: -1})
	b := GenerateSeeded(42, testBases(), Options{PlayerLevel: 20, Tier: -1})

	sig := func(s *Stock) [][5]interface{} {
		out := [][5]interface{}{}
		for _, it := range s.Items {
			out = append(out, [5]interface{}{it.Code, it.Quality, it.X, it.Y, it.Quantity})
		}

		return out
	}

	if !reflect.DeepEqual(sig(a), sig(b)) {
		t.Fatal("same seed gave different stock")
	}

	if len(a.Items) == 0 {
		t.Fatal("empty stock")
	}
}

func TestGenerateRules(t *testing.T) {
	for seed := uint32(1); seed < 200; seed++ {
		s := GenerateSeeded(seed, testBases(), Options{PlayerLevel: 20, Tier: -1}) // ilvl 25: magic pass only
		perm := map[string]int{}

		for _, it := range s.Items {
			if it.ILvl != 25 {
				t.Fatalf("ilvl %d want 25", it.ILvl)
			}

			if it.Permanent {
				perm[it.Code]++

				if it.Quality != d2drop.QualityNormal {
					t.Errorf("permanent %s quality %v", it.Code, it.Quality)
				}
			} else if it.Quality != d2drop.QualityMagic {
				t.Errorf("seed %d: %s quality %v; at ilvl >= 25 only magic items are rolled", seed, it.Code, it.Quality)
			}

			if it.Code == "aqv" && it.Quantity != 350 {
				t.Errorf("arrows quantity %d want a full stack", it.Quantity)
			}

			if it.Code == "plt" && it.ILvl < 40 {
				// reqlevel gates only the regular pass; the magic pass is gated by MagicLvl
				continue
			}
		}

		if perm["aqv"] != 1 || perm["hp1"] != 1 {
			t.Fatalf("seed %d: permanent items %v", seed, perm)
		}
	}
}

func TestGenerateRegularPassUnderLevel25(t *testing.T) {
	// level 1 => ilvl 6: regular cap only (plt needs level 40, cap magic level 5 <= 6)
	r := &seqRNG{vals: []uint32{99}}
	s := Generate(r, testBases(), Options{PlayerLevel: 1, Tier: -1})

	counts := map[string]int{}
	for _, it := range s.Items {
		counts[it.Code]++
	}

	// cap: regular count = Min 1 + 99 % (Max+1-Min = 2) = 2; magic: MagicMin 1 +
	// 99 % (MagicMax 2 + 1 - 1 = 2) = 2; that is 4 copies (Min is read, VERIFIED).
	// plt (needs level 40 > ilvl 6): neither pass runs (VERIFIED: the
	// reqlevel gate of 0x574814 wraps the magic pass too).
	if counts["cap"] != 4 || counts["plt"] != 0 {
		t.Errorf("counts %v", counts)
	}

	r = &seqRNG{vals: []uint32{1}}
	s = Generate(r, testBases(), Options{PlayerLevel: 1, Tier: -1})
	counts = map[string]int{}

	for _, it := range s.Items {
		counts[it.Code]++
	}

	if counts["cap"] < 2 { // 1 regular + 1 magic
		t.Errorf("counts %v", counts)
	}

	if counts["aqv"] != 1 || counts["hp1"] != 1 {
		t.Errorf("permanent items missing: %v", counts)
	}
}

func TestStockPlaceRemove(t *testing.T) {
	s := NewStock()
	a := &Item{Code: "a", W: 2, H: 3}
	b := &Item{Code: "b", W: 1, H: 1}

	if !s.Place(a) || !s.Place(b) {
		t.Fatal("place failed")
	}

	if s.ItemAt(a.X, a.Y) != a || s.ItemAt(b.X, b.Y) != b {
		t.Error("ItemAt wrong")
	}

	s.Remove(a)

	if s.ItemAt(a.X, a.Y) != nil || len(s.Items) != 1 {
		t.Error("remove failed")
	}

	// a 10x10 grid holds exactly 100 cells
	full := NewStock()

	for i := 0; i < GridCols*GridRows; i++ {
		if !full.Place(&Item{W: 1, H: 1}) {
			t.Fatalf("cell %d should fit", i)
		}
	}

	if full.Place(&Item{W: 1, H: 1}) {
		t.Error("grid should be full")
	}
}

func TestGenerateStopsAfterFailures(t *testing.T) {
	huge := []Base{{Code: "big", W: 10, H: 10, Gear: true, Vendor: Params{Max: 5}}}
	r := &seqRNG{vals: []uint32{5}}

	s := Generate(r, huge, Options{PlayerLevel: 1, Tier: -1})
	if len(s.Items) != 1 {
		t.Errorf("one 10x10 item should fill the grid, got %d", len(s.Items))
	}
}

var _ d2drop.RNG = (*d2rand.Seed)(nil)
