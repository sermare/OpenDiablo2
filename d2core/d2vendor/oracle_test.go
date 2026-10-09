package d2vendor

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2txt"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2resource"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2trade"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2util"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/d2drop"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2records"
)

// Oracle tests on the real 1.14b tables. Rules are marked VERIFIED (read from
// Game.exe and the tables, see d2-re-notes/inventory-trade.md) or UNVERIFIED
// (hypothesis kept on purpose, with the exe address still to confirm).

// loadRecords builds a RecordManager from real 1.14b tables found under the
// directory in D2_TABLES: patch_d2/{weapons,armor,misc,ItemTypes}.txt and
// d2exp/{gamble,difficultylevels,npc}.txt. The test is skipped when the
// variable or a file is missing; no game data is committed.
func loadRecords(t *testing.T) *d2records.RecordManager {
	t.Helper()

	root := os.Getenv("D2_TABLES")
	if root == "" {
		t.Skip("D2_TABLES not set")
	}

	rec, err := d2records.NewRecordManager(d2util.LogLevelNone)
	if err != nil {
		t.Fatal(err)
	}

	files := []struct{ path, file string }{
		{d2resource.Weapons, "patch_d2/weapons.txt"},
		{d2resource.Armor, "patch_d2/armor.txt"},
		{d2resource.Misc, "patch_d2/misc.txt"},
		{d2resource.ItemTypes, "patch_d2/ItemTypes.txt"},
		{d2resource.Gamble, "d2exp/gamble.txt"},
		{d2resource.DifficultyLevels, "d2exp/difficultylevels.txt"},
		{d2resource.NPC, "d2exp/npc.txt"},
	}

	for _, f := range files {
		data, err := os.ReadFile(filepath.Join(root, f.file))
		if err != nil {
			t.Skip(err)
		}

		if err := rec.Load(f.path, d2txt.LoadDataDictionary(data)); err != nil {
			t.Fatalf("%s: %v", f.file, err)
		}
	}

	return rec
}

func rawHeader(t *testing.T, file string) []string {
	t.Helper()

	data, err := os.ReadFile(filepath.Join(os.Getenv("D2_TABLES"), file))
	if err != nil {
		t.Skip(err)
	}

	return strings.Split(strings.SplitN(string(data), "\n", 2)[0], "\t")
}

func TestVendorCatalogOracle(t *testing.T) {
	rec := loadRecords(t)

	// ClassID is the monstats hcIdx (VERIFIED against MonStats.txt).
	data, err := os.ReadFile(filepath.Join(os.Getenv("D2_TABLES"), "itemgen/patch_d2/MonStats.txt"))
	if err != nil {
		t.Skip(err)
	}

	ids := map[string]string{}

	for _, line := range strings.Split(string(data), "\n")[1:] {
		f := strings.Split(line, "\t")
		if len(f) > 2 {
			ids[strings.ToLower(f[0])] = f[1]
		}
	}

	for _, v := range All() {
		if got := ids[v.NPC]; got != strconv.Itoa(v.ClassID) {
			t.Errorf("%s: class id %d, MonStats hcIdx %q", v.Name, v.ClassID, got)
		}

		if rec.NPCs[v.NPC] == nil {
			t.Errorf("%s: no npc.txt row %q", v.Name, v.NPC)
		}

		if len(BasesFor(rec, v)) == 0 {
			t.Errorf("%s: no stock columns in the item tables", v.Name)
		}
	}

	// Vendors without item columns in armor/weapons/misc have no stock to
	// model (VERIFIED against the headers). Cain has columns but only identifies.
	for _, file := range []string{"patch_d2/armor.txt", "patch_d2/weapons.txt", "patch_d2/misc.txt"} {
		cols := strings.Join(rawHeader(t, file), ",")
		for _, none := range []string{"NihlathakMin", "GreizMin", "QualKehkMin", "JerhynMin"} {
			if strings.Contains(cols, none) {
				t.Errorf("%s has %s: an unmodelled vendor sells stock", file, none)
			}
		}
	}

	for _, name := range []string{"nihlathak", "greiz", "qual-kehk"} {
		if _, ok := ByName(name); ok {
			t.Errorf("%s must not be a stock vendor", name)
		}
	}

	// Which item types each vendor sells (VERIFIED from the tables).
	sells := map[string][]string{
		"Akara":   {"ibk", "tbk", "tsc", "isc", "hp1", "mp1", "wnd", "sst", "scp"},
		"Charsi":  {"aqv", "cqv", "hax", "buc"},
		"Gheed":   {"aqv", "cqv", "key"},
		"Alkor":   {"vps", "wms", "yps"},
		"Larzuk":  {"aqv", "cqv"},
		"Malah":   {"hp4", "mp4", "tsc", "isc"},
		"Jamella": {"hp4", "mp4", "tsc", "isc"},
		"Ormus":   {"hp3", "mp3", "tsc", "isc"},
	}

	for name, codes := range sells {
		v, _ := ByName(name)
		have := map[string]bool{}

		for _, b := range BasesFor(rec, v) {
			have[b.Code] = true
		}

		for _, c := range codes {
			if !have[c] {
				t.Errorf("%s should sell %s", name, c)
			}
		}
	}

	// Alkor sells only potions (permanent bases, no magic pass).
	alkor, _ := ByName("Alkor")
	for _, b := range BasesFor(rec, alkor) {
		if !b.Permanent || b.CanBeMagic {
			t.Errorf("Alkor base %s must be a permanent potion", b.Code)
		}
	}
}

func TestStockInvariantsOracle(t *testing.T) {
	rec := loadRecords(t)

	for _, v := range All() {
		bases := BasesFor(rec, v)
		byCode := map[string]Base{}

		for _, b := range bases {
			byCode[b.Code] = b
		}

		for _, level := range []int{1, 8, 15, 19, 20, 24, 30, 60, 99} {
			ilvl := ItemLevel(level, -1)
			if ilvl != level+5 {
				t.Fatalf("ilvl %d for level %d", ilvl, level)
			}

			for seed := uint32(1); seed <= 25; seed++ {
				s := GenerateSeeded(seed, bases, Options{PlayerLevel: level, Tier: -1})

				if len(s.Items) > GridCols*GridRows {
					t.Fatalf("%s: %d items exceed the grid", v.Name, len(s.Items))
				}

				regular := map[string]int{}
				magic := map[string]int{}
				perm := map[string]bool{}
				occupied := map[[2]int]string{}

				for _, it := range s.Items {
					b, ok := byCode[it.Code]
					if !ok {
						t.Fatalf("%s: item %s is not in the vendor's columns", v.Name, it.Code)
					}

					if it.ILvl != ilvl {
						t.Errorf("%s %s: ilvl %d, want %d", v.Name, it.Code, it.ILvl, ilvl)
					}

					for x := it.X; x < it.X+it.W; x++ {
						for y := it.Y; y < it.Y+it.H; y++ {
							if o, taken := occupied[[2]int{x, y}]; taken {
								t.Fatalf("%s: %s overlaps %s at %d,%d", v.Name, it.Code, o, x, y)
							}

							occupied[[2]int{x, y}] = it.Code
						}
					}

					switch {
					case it.Permanent:
						perm[it.Code] = true

						if b.Ammo && it.Quantity != b.MaxStack {
							t.Errorf("%s %s: permanent ammo quantity %d, want full stack %d", v.Name, it.Code, it.Quantity, b.MaxStack)
						}
					case it.Quality == d2drop.QualityMagic:
						magic[it.Code]++

						if b.Vendor.MagicLevel > ilvl || !b.CanBeMagic {
							t.Errorf("%s: magic %s at ilvl %d (MagicLvl %d)", v.Name, it.Code, ilvl, b.Vendor.MagicLevel)
						}
					default:
						regular[it.Code]++

						if ilvl >= regularItemLevelLimit {
							t.Errorf("%s: regular %s at ilvl %d (>=25 has only the magic pass)", v.Name, it.Code, ilvl)
						}

						if b.ReqLevel > ilvl {
							t.Errorf("%s: %s requires level %d > ilvl %d", v.Name, it.Code, b.ReqLevel, ilvl)
						}
					}
				}

				for code, n := range regular {
					if n > byCode[code].Vendor.Max {
						t.Errorf("%s %s: %d regular copies, max column %d", v.Name, code, n, byCode[code].Vendor.Max)
					}
				}

				for code, n := range magic {
					// rolled in [0, MagicMax+1) below ilvl 25, [0, MagicMax+1..3) from 25 on
					limit := byCode[code].Vendor.MagicMax
					if ilvl > 24 {
						limit += 2
					}

					if n > limit {
						t.Errorf("%s %s: %d magic copies, limit %d", v.Name, code, n, limit)
					}
				}

				for _, b := range bases {
					if b.Permanent && b.Vendor.Max > 0 && !perm[b.Code] {
						t.Errorf("%s: permanent %s missing at level %d seed %d", v.Name, b.Code, level, seed)
					}
				}
			}
		}
	}
}

type fixedRNG struct{ v int }

func (f *fixedRNG) Roll(n int32) uint32 { return uint32(f.v) % uint32(n) }
func (f *fixedRNG) Chance() bool        { return false }

// Quality odds of regular vendor items (VERIFIED thresholds of FUN_00574710:
// low 9% below ilvl 5, superior 14% for ilvl 5..9 and 25% from ilvl 10).
func TestRollQualityOddsOracle(t *testing.T) {
	cases := []struct {
		ilvl    int
		q       d2drop.Quality
		percent int
	}{
		{1, d2drop.QualityLow, 9},
		{7, d2drop.QualitySuperior, 14},
		{10, d2drop.QualitySuperior, 25},
		{45, d2drop.QualitySuperior, 25},
	}

	for _, c := range cases {
		hit := 0

		for r := 0; r < 100; r++ {
			if RollQuality(&fixedRNG{r}, c.ilvl) == c.q {
				hit++
			}
		}

		if hit != c.percent {
			t.Errorf("ilvl %d: %d of 100 rolls give quality %d, want %d", c.ilvl, hit, c.q, c.percent)
		}
	}
}

func TestStockSeedReproducible(t *testing.T) {
	seen := map[uint32]string{}

	for _, v := range All() {
		for _, gamble := range []bool{false, true} {
			a := StockSeed(12345, v.ClassID, 0, gamble)
			if a != StockSeed(12345, v.ClassID, 0, gamble) {
				t.Fatal("StockSeed is not deterministic")
			}

			key := v.Name
			if gamble {
				key += "#gamble"
			}

			if o, dup := seen[a]; dup {
				t.Errorf("%s and %s share a stock seed", key, o)
			}

			seen[a] = key

			if StockSeed(12345, v.ClassID, 1, gamble) == a {
				t.Errorf("%s: a restock must change the seed", key)
			}
		}
	}

	if StockSeed(1, 148, 0, false) == StockSeed(2, 148, 0, false) {
		t.Error("different games must give different stocks")
	}
}

func TestVendorStockDeterministicOracle(t *testing.T) {
	rec := loadRecords(t)
	v, _ := ByName("Charsi")
	bases := BasesFor(rec, v)
	seed := StockSeed(99, v.ClassID, 0, false)

	codes := func(s *Stock) string {
		var out []string
		for _, it := range s.Items {
			out = append(out, it.Code+"/"+strconv.Itoa(int(it.Quality)))
		}

		return strings.Join(out, ",")
	}

	a := codes(GenerateSeeded(seed, bases, Options{PlayerLevel: 20, Tier: -1}))
	if a != codes(GenerateSeeded(seed, bases, Options{PlayerLevel: 20, Tier: -1})) {
		t.Error("stock is not reproducible from its seed")
	}

	if a == codes(GenerateSeeded(StockSeed(99, v.ClassID, 1, false), bases, Options{PlayerLevel: 20, Tier: -1})) {
		t.Error("restock produced the same stock")
	}
}

func TestGambleOracle(t *testing.T) {
	rec := loadRecords(t)

	pool, ring, amulet, ok := GamblePoolFor(rec)
	if !ok || ring.Code != "rin" || amulet.Code != "amu" {
		t.Fatalf("pool ok=%v ring=%q amulet=%q", ok, ring.Code, amulet.Code)
	}

	if pool[0].Level > gambleMinILvl {
		t.Errorf("no gamble base at level <= %d (first %s level %d): low level players would get a short stock", gambleMinILvl, pool[0].Code, pool[0].Level)
	}

	// The expansion's difficultylevels.txt has no Gamble* columns; the values
	// of the 1.14b .bin must be used (VERIFIED, identical for all difficulties).
	for d := 0; d < 3; d++ {
		if p := GambleParamsFor(rec, d); p != ShippedGambleParams {
			t.Errorf("difficulty %d: %+v, want the shipped bin values %+v", d, p, ShippedGambleParams)
		}
	}

	// Item level of every gamble item: level - 5 + rand(10), clamped to 5..99.
	for level := 1; level <= 99; level++ {
		for seed := uint32(1); seed <= 20; seed++ {
			items := GenerateGamble(d2rand.New(seed), pool, ring, amulet, ShippedGambleParams, level, true)
			if len(items) != GambleItems {
				t.Fatalf("level %d seed %d: %d items, want %d", level, seed, len(items), GambleItems)
			}

			lo, hi := level-gambleILvlShift, level-gambleILvlShift+gambleILvlSpan-1
			if lo < gambleMinILvl {
				lo = gambleMinILvl
			}

			if hi < gambleMinILvl {
				hi = gambleMinILvl
			}

			if hi > gambleMaxILvl {
				hi = gambleMaxILvl
			}

			for i, it := range items {
				if it.ILvl < lo || it.ILvl > hi {
					t.Fatalf("level %d: item %d ilvl %d outside %d..%d", level, i, it.ILvl, lo, hi)
				}
			}
		}
	}
}

// Buy and sell multipliers per vendor straight from npc.txt (VERIFIED: the
// loader order is swapped, the file's "sell mult" is what the player pays).
// The quest group columns are mapped by the same swap; that is UNVERIFIED.
func TestNPCPricingOracle(t *testing.T) {
	rec := loadRecords(t)

	tests := []struct {
		name         string
		pays         int
		maxBuyNormal int
	}{
		{"Gheed", 1088, 5000}, {"Charsi", 960, 5000}, {"Akara", 1024, 5000},
		{"Lysander", 1024, 10000}, {"Drognan", 1024, 10000}, {"Elzix", 1024, 10000}, {"Fara", 1024, 10000},
		{"Hralti", 1024, 15000}, {"Alkor", 1024, 15000}, {"Ormus", 1024, 15000}, {"Asheara", 1024, 15000},
		{"Jamella", 1024, 20000}, {"Halbu", 1024, 20000},
		{"Malah", 2048, 25000}, {"Drehya", 2048, 25000}, {"Larzuk", 2048, 25000},
	}

	for _, tc := range tests {
		v, _ := ByName(tc.name)
		n := NPCPricing(rec, v, nil)

		if n.PlayerPays != tc.pays || n.VendorPays != 512 || n.Repair != 128 {
			t.Errorf("%s: pays %d vendor pays %d repair %d, want %d/512/128", tc.name, n.PlayerPays, n.VendorPays, n.Repair, tc.pays)
		}

		if n.MaxBuy != [3]int{tc.maxBuyNormal, 25000, 25000} {
			t.Errorf("%s: max buy %v", tc.name, n.MaxBuy)
		}

		// Whatever an item is worth, selling it never fetches more than the cap.
		item := &d2trade.Item{BaseCost: 10_000_000, Quantity: 1, MaxStack: 1, Identified: true}
		for d, limit := range n.MaxBuy {
			if got := d2trade.ItemPrice(item, d2trade.Params{Mode: d2trade.ModeSell, Difficulty: d, NPC: n}); got != limit {
				t.Errorf("%s difficulty %d: sell-back %d, want the cap %d", tc.name, d, got, limit)
			}
		}
	}

	// A 100 gold item costs cost*PlayerPays/1024 (Gheed 1088 -> 106).
	gheed, _ := ByName("Gheed")
	tp := &d2trade.Item{BaseCost: 100, Quantity: 1, MaxStack: 1, Identified: true}

	if got := d2trade.ItemPrice(tp, d2trade.Params{Mode: d2trade.ModeBuy, NPC: NPCPricing(rec, gheed, nil)}); got != 106 {
		t.Errorf("100 gold item at Gheed costs %d, want 106 (1088/1024)", got)
	}
}
