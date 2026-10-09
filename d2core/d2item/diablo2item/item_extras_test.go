package diablo2item

import (
	"math/rand"
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

	rec.Item.Treasure.Expansion["Caps"] = &d2records.TreasureClassRecord{
		Name: "Caps", NumPicks: 3,
		Treasures: []*d2records.Treasure{{Code: "cap", Probability: 1}, {Code: "skp", Probability: 1}},
	}

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

// Switching RollExtras on must not change anything but the extras: for a fixed
// seed the drop stream, codes, qualities, levels and seeds are identical.
func TestRollExtrasLeavesOtherFieldsUnchanged(t *testing.T) {
	f, _ := extrasFactory()

	type key struct {
		code     string
		ilvl     int
		seed     int64
		q        d2drop.Quality
		pre, suf int
	}

	flat := func(items []*Item) []key {
		var out []key
		for _, it := range items {
			out = append(out, key{it.CommonCode, it.itemLevel, it.Seed, it.genQuality, len(it.PrefixCodes), len(it.SuffixCodes)})
		}

		return out
	}

	extras := 0

	for seed := uint32(1); seed <= 300; seed++ {
		off, err := f.DropItems("Caps", DropOptions{Seed: seed, ILvl: 30, Players: 1, MaxDrops: 6})
		if err != nil {
			t.Fatal(err)
		}

		on, err := f.DropItems("Caps", DropOptions{Seed: seed, ILvl: 30, Players: 1, MaxDrops: 6, RollExtras: true})
		if err != nil {
			t.Fatal(err)
		}

		a, b := flat(off), flat(on)
		if len(a) != len(b) {
			t.Fatalf("seed %d: %d items off, %d on", seed, len(a), len(b))
		}

		for i := range a {
			if a[i] != b[i] {
				t.Fatalf("seed %d item %d: %+v != %+v", seed, i, a[i], b[i])
			}

			if on[i].IsEthereal() || on[i].NumSockets() > 0 {
				extras++
			}
		}
	}

	if extras == 0 {
		t.Error("no extras rolled in 300 seeds")
	}
}

// Sockets and the halved durability of an ethereal item survive Spec ->
// ItemFromSpec.
func TestSpecKeepsExtras(t *testing.T) {
	f, _ := extrasFactory()
	seenS, seenE := false, false

	for seed := uint32(1); seed <= 300; seed++ {
		items, err := f.DropItems("Caps", DropOptions{Seed: seed, ILvl: 30, Players: 1, MaxDrops: 6, RollExtras: true})
		if err != nil {
			t.Fatal(err)
		}

		for _, it := range items {
			again, err := f.ItemFromSpec(it.Spec())
			if err != nil {
				t.Fatal(err)
			}

			c1, m1 := it.Durability()
			c2, m2 := again.Durability()

			if again.NumSockets() != it.NumSockets() || again.IsEthereal() != it.IsEthereal() || c1 != c2 || m1 != m2 {
				t.Fatalf("seed %d: sockets %d/%d eth %v/%v dur %d/%d vs %d/%d", seed,
					it.NumSockets(), again.NumSockets(), it.IsEthereal(), again.IsEthereal(), c1, m1, c2, m2)
			}

			seenS = seenS || it.NumSockets() > 0
			seenE = seenE || it.IsEthereal()
		}
	}

	if !seenS || !seenE {
		t.Errorf("not exercised: sockets %v ethereal %v", seenS, seenE)
	}
}

func TestEtherealBonusTable(t *testing.T) {
	for _, tc := range []struct{ in, want int }{{0, 0}, {1, 1}, {2, 3}, {5, 7}, {11, 16}, {30, 45}, {101, 151}} {
		if got := etherealBoost(tc.in); got != tc.want {
			t.Errorf("etherealBoost(%d) = %d, want %d", tc.in, got, tc.want)
		}
	}

	a := &itemAttributes{
		damageOneHand: minMaxEnhanceable{min: 3, max: 7}, damageTwoHand: minMaxEnhanceable{min: 5, max: 11},
		damageMissile: minMaxEnhanceable{min: 1, max: 2}, defense: 31,
	}
	a.applyEtherialBonus()

	if a.damageOneHand.min != 4 || a.damageOneHand.max != 10 || a.damageTwoHand.min != 7 ||
		a.damageTwoHand.max != 16 || a.damageMissile.min != 1 || a.damageMissile.max != 3 || a.defense != 46 {
		t.Errorf("bonus wrong: %+v", a)
	}
}

// Ethereal items rolled by the generator get the bonus.
func TestEtherialDefenseFromRoll(t *testing.T) {
	f, tab := extrasFactory()
	f.asset.Records.Item.All["cap"].MinAC, f.asset.Records.Item.All["cap"].MaxAC = 10, 20

	seen := false

	for s := int64(1); s <= 600; s++ {
		it := newExtrasItem(s, d2drop.QualityNormal)
		it.factory = f
		it.rand = rand.New(rand.NewSource(s))
		it.updateItemAttributes()
		it.attributes.durable, it.attributes.currentDurability = true, 12

		base := it.attributes.defense
		f.rollExtras(tab, it, 0)

		if it.IsEthereal() {
			seen = true

			if it.attributes.defense != base*3/2 {
				t.Fatalf("defense %d, base %d", it.attributes.defense, base)
			}
		} else if it.attributes.defense != base {
			t.Fatalf("non-ethereal defense changed")
		}
	}

	if !seen {
		t.Error("no ethereal item rolled")
	}
}

// Shop stock: never ethereal, can be socketed, and apart from the socket
// count identical to the plain ItemFromCode item for the same seed.
func TestVendorItemsSocketsNoEthereal(t *testing.T) {
	f, _ := extrasFactory()
	socketed := 0

	for seed := uint32(1); seed <= 500; seed++ {
		plain, err := f.ItemFromCode("cap", d2drop.QualityNormal, 30, seed)
		if err != nil {
			t.Fatal(err)
		}

		shop, err := f.ItemFromCodeForVendor("cap", d2drop.QualityNormal, 30, seed, 0)
		if err != nil {
			t.Fatal(err)
		}

		if shop.IsEthereal() {
			t.Fatalf("seed %d: vendor item ethereal", seed)
		}

		pa, sa := *plain.attributes, *shop.attributes
		pa.numSockets, sa.numSockets = 0, 0

		if plain.Seed != shop.Seed || plain.itemLevel != shop.itemLevel || plain.quality != shop.quality ||
			plain.CommonCode != shop.CommonCode || pa.defense != sa.defense || pa.ethereal != sa.ethereal ||
			pa.currentDurability != sa.currentDurability || pa.durability != sa.durability {
			t.Fatalf("seed %d: item differs beyond sockets", seed)
		}

		if shop.NumSockets() > 2 {
			t.Fatalf("seed %d: %d sockets", seed, shop.NumSockets())
		}

		if shop.NumSockets() > 0 {
			socketed++
		}
	}

	if socketed < 100 || socketed > 230 {
		t.Errorf("%d of 500 vendor caps socketed, want about 33%%", socketed)
	}
}
