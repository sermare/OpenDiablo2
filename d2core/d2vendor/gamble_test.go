package d2vendor

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/d2drop"
)

func TestGambleQuality(t *testing.T) {
	p := GambleParams{Rare: 10000, Set: 100, Unique: 50}
	tests := []struct {
		roll uint32
		want d2drop.Quality
	}{
		{0, d2drop.QualityUnique},
		{49, d2drop.QualityUnique},
		{50, d2drop.QualitySet},
		{149, d2drop.QualitySet},
		{150, d2drop.QualityRare},
		{10149, d2drop.QualityRare},
		{10150, d2drop.QualityMagic},
		{99999, d2drop.QualityMagic},
	}

	for _, tt := range tests {
		if got := GambleQuality(&seqRNG{vals: []uint32{tt.roll}}, p); got != tt.want {
			t.Errorf("roll %d: got %d want %d", tt.roll, got, tt.want)
		}
	}

	r := &seqRNG{vals: []uint32{0}}
	if GambleQuality(r, GambleParams{}) != d2drop.QualityMagic || len(r.mods) != 0 {
		t.Error("zero windows: magic, no roll")
	}
}

func TestGambleItemLevel(t *testing.T) {
	tests := []struct {
		level int
		roll  uint32
		want  int
	}{{30, 0, 25}, {30, 9, 34}, {1, 0, 5}, {99, 9, 99}, {10, 5, 10}}

	for _, tt := range tests {
		if got := GambleItemLevel(&seqRNG{vals: []uint32{tt.roll}}, tt.level); got != tt.want {
			t.Errorf("level %d roll %d: %d want %d", tt.level, tt.roll, got, tt.want)
		}
	}
}

func TestUpgradeGambleBase(t *testing.T) {
	b := GambleBase{Code: "hax", Exc: "9ha", ExcLvl: 20, Elite: "7ha", EliteL: 40}
	p := GambleParams{Uber: 90, Ultra: 33}
	tests := []struct {
		name  string
		ilvl  int
		rolls []uint32
		want  string
	}{
		{"below exc level", 10, []uint32{0}, "hax"},
		{"exc hit", 25, []uint32{450}, "9ha"}, // 5*90+1 = 451 > 450
		{"exc miss, elite level too low", 25, []uint32{451}, "hax"},
		{"elite hit", 50, []uint32{9999, 330}, "7ha"}, // 10*33+1 = 331 > 330
		{"elite miss", 50, []uint32{9999, 331}, "hax"},
	}

	for _, tt := range tests {
		if got := UpgradeGambleBase(&seqRNG{vals: tt.rolls}, b, tt.ilvl, p); got != tt.want {
			t.Errorf("%s: %q want %q", tt.name, got, tt.want)
		}
	}

	if UpgradeGambleBase(&seqRNG{vals: []uint32{0}}, GambleBase{Code: "x"}, 50, p) != "x" {
		t.Error("no upgrade row")
	}
}

func TestGenerateGamble(t *testing.T) {
	pool := []GambleBase{
		{Code: "d", Level: 10}, {Code: "a", Level: 1}, {Code: "c", Level: 5}, {Code: "b", Level: 5},
	}
	SortGamblePool(pool)

	if pool[0].Code != "a" || pool[1].Code != "b" || pool[2].Code != "c" || pool[3].Code != "d" {
		t.Fatalf("pool order %v", pool)
	}

	ring, amu := GambleBase{Code: "rin"}, GambleBase{Code: "amu"}
	p := GambleParams{Rare: 10000, Set: 100, Unique: 50}

	a := GenerateGamble(d2rand.New(7), pool, ring, amu, p, 20, true)
	b := GenerateGamble(d2rand.New(7), pool, ring, amu, p, 20, true)

	if len(a) != GambleItems {
		t.Fatalf("got %d items", len(a))
	}

	for i := range a {
		if a[i] != b[i] {
			t.Fatal("not deterministic")
		}
	}

	if a[0].Code != "rin" || a[1].Code != "amu" {
		t.Errorf("slots 0/1 = %s %s", a[0].Code, a[1].Code)
	}

	for _, it := range a[2:] {
		if it.ILvl < 15 || it.ILvl > 24 {
			t.Errorf("item level %d outside 15..24", it.ILvl)
		}
	}

	// at player level 5 (item levels 5..9) the level-10 base is never picked
	for _, it := range GenerateGamble(d2rand.New(3), pool, ring, amu, p, 5, false)[2:] {
		if it.Code == "d" {
			t.Error("level 10 base picked at item level < 10")
		}
	}
}
