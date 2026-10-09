package diablo2item

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/d2drop"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2records"
)

func extrasFactory() (*ItemFactory, *dropTables) {
	f := testDropFactory()
	rec := f.asset.Records
	rec.Item.All["cap"].Durability = 12
	rec.Item.All["cap"].HasInventory = true
	rec.Item.All["cap"].GemSockets = 2
	rec.Item.All["cap"].InventoryWidth, rec.Item.All["cap"].InventoryHeight = 2, 2
	rec.Item.Types["helm"] = &d2records.ItemTypeRecord{Code: "helm", Rare: true, MaxSock1: 2, MaxSock25: 2, MaxSock40: 3}
	rec.Item.Types["armo"] = &d2records.ItemTypeRecord{Code: "armo", Rare: true}

	return f, f.dropTables()
}

func newExtrasItem(seed int64, q d2drop.Quality) *Item {
	return &Item{
		CommonCode: "cap", Seed: seed, itemLevel: 30, genQuality: q,
		attributes: &itemAttributes{durable: true, currentDurability: 12, durability: minMaxEnhanceable{min: 12, max: 12}},
	}
}

// Over many seeds the extras happen at roughly the verified rates and always
// obey the verified limits.
func TestRollExtrasRatesAndLimits(t *testing.T) {
	f, tab := extrasFactory()

	eth, sock := 0, 0
	const n = 4000

	for s := int64(1); s <= n; s++ {
		it := newExtrasItem(s*7919, d2drop.QualityNormal)
		f.rollExtras(tab, it, 0)

		if it.IsEthereal() {
			eth++

			if _, m := it.Durability(); m != 7 {
				t.Fatalf("ethereal max durability %d want 7", m)
			}
		}

		if c := it.NumSockets(); c > 0 {
			sock++

			if c > 2 {
				t.Fatalf("%d sockets on a gemsockets 2 helm", c)
			}
		}
	}

	if eth < n*3/100 || eth > n*7/100 {
		t.Errorf("ethereal %d of %d, want about 5%%", eth, n)
	}

	if sock < n*28/100 || sock > n*38/100 {
		t.Errorf("sockets %d of %d, want about 33%%", sock, n)
	}
}

func TestRollExtrasEligibility(t *testing.T) {
	f, tab := extrasFactory()

	for s := int64(1); s <= 500; s++ {
		for _, q := range []d2drop.Quality{d2drop.QualityLow, d2drop.QualitySet} {
			it := newExtrasItem(s, q)
			f.rollExtras(tab, it, 2)

			if it.IsEthereal() {
				t.Fatalf("quality %d ethereal", q)
			}
		}

		for _, q := range []d2drop.Quality{d2drop.QualityLow, d2drop.QualityMagic, d2drop.QualityRare, d2drop.QualityUnique} {
			it := newExtrasItem(s, q)
			f.rollExtras(tab, it, 2)

			if it.NumSockets() != 0 {
				t.Fatalf("quality %d got sockets", q)
			}
		}
	}

	// not generated (no quality): untouched
	it := newExtrasItem(1, d2drop.QualityNone)
	f.rollExtras(tab, it, 0)

	if it.IsEthereal() || it.NumSockets() != 0 {
		t.Error("ungenerated item changed")
	}
}

// Same seed, same result; the feature is off unless asked for.
func TestRollExtrasDeterministicAndOptIn(t *testing.T) {
	f, tab := extrasFactory()

	a, b := newExtrasItem(12345, d2drop.QualityNormal), newExtrasItem(12345, d2drop.QualityNormal)
	f.rollExtras(tab, a, 1)
	f.rollExtras(tab, b, 1)

	if a.NumSockets() != b.NumSockets() || a.IsEthereal() != b.IsEthereal() {
		t.Error("not deterministic")
	}

	if (DropOptions{}).RollExtras {
		t.Error("extras must be off by default")
	}
}
