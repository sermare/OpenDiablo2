package d2drop

import (
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
)

// Real-data tests run only with D2_TABLES set to a directory holding the
// extracted game tables: armor.txt, weapons.txt, misc.txt, ItemTypes.txt and
// itemgen/patch_d2/{TreasureClassEx,ItemRatio}.txt (1.14b versions).

type tsv struct {
	cols map[string]int
	rows [][]string
}

func readTSV(t *testing.T, path string) *tsv {
	t.Helper()

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("missing table: %v", err)
	}

	lines := strings.Split(strings.ReplaceAll(string(raw), "\r", ""), "\n")
	out := &tsv{cols: map[string]int{}}

	for i, h := range strings.Split(lines[0], "\t") {
		if _, dup := out.cols[strings.ToLower(h)]; !dup {
			out.cols[strings.ToLower(h)] = i
		}
	}

	for _, l := range lines[1:] {
		if l != "" {
			out.rows = append(out.rows, strings.Split(l, "\t"))
		}
	}

	return out
}

func (d *tsv) s(row []string, col string) string {
	i, ok := d.cols[strings.ToLower(col)]
	if !ok || i >= len(row) {
		return ""
	}

	return strings.TrimSpace(row[i])
}

func (d *tsv) n(row []string, col string) int {
	v, _ := strconv.Atoi(d.s(row, col))

	return v
}

type realTables struct {
	tcs    *TreasureTable
	items  mapItems
	ratios map[[2]bool]*Ratio
	uniq   map[string]int // UniqueItems name -> row index
	sets   map[string]int // SetItems name -> row index
	base   map[string]string
	ver    map[string]int // unique/set row version
}

// rowIndex is the row index of a unique or set item (-1 if unknown).
func (r *realTables) rowIndex(q Quality, name string) int {
	m := r.uniq
	if q == QualitySet {
		m = r.sets
	}

	if i, ok := m[name]; ok {
		return i
	}

	return -1
}

func (r *realTables) ItemRatio(cs, uber bool) (*Ratio, bool) {
	x, ok := r.ratios[[2]bool{cs, uber}]

	return x, ok
}

func loadReal(t *testing.T) *realTables {
	t.Helper()

	root := os.Getenv("D2_TABLES")
	if root == "" {
		t.Skip("D2_TABLES not set")
	}

	rt := &realTables{tcs: NewTreasureTable(), items: mapItems{}, ratios: map[[2]bool]*Ratio{},
		uniq: map[string]int{}, sets: map[string]int{}}

	rt.base = map[string]string{}
	rt.ver = map[string]int{}

	for _, tab := range []struct {
		file, base string
		dst        map[string]int
	}{{"UniqueItems.txt", "code", rt.uniq}, {"SetItems.txt", "item", rt.sets}} {
		tt := readTSV(t, filepath.Join(root, "itemgen", "patch_d2", tab.file))
		n := 0

		for _, r := range tt.rows {
			name := tt.s(r, "index")
			if name == "" || name == "Expansion" {
				continue
			}

			if _, dup := tab.dst[name]; !dup {
				tab.dst[name] = n
				rt.base[name] = tt.s(r, tab.base)
				rt.ver[name] = tt.n(r, "version")
			}

			n++
		}
	}

	// Item types with their ancestors.
	types := readTSV(t, filepath.Join(root, "ItemTypes.txt"))
	parents := map[string][]string{}
	flags := map[string][3]bool{} // normal, magic, rare
	rarity := map[string]int{}
	throw := map[string]bool{}
	cls := map[string]bool{}

	var typeCodes []string

	for _, r := range types.rows {
		c := types.s(r, "Code")
		if c == "" {
			continue
		}

		parents[c] = []string{types.s(r, "Equiv1"), types.s(r, "Equiv2")}
		flags[c] = [3]bool{types.n(r, "Normal") == 1, types.n(r, "Magic") == 1, types.n(r, "Rare") == 1}
		cls[c] = types.s(r, "Class") != ""
		rarity[c] = types.n(r, "Rarity")
		throw[c] = types.n(r, "Throwable") > 0

		if types.n(r, "TreasureClass") == 1 {
			typeCodes = append(typeCodes, c)
		}
	}

	var chain func(c string, seen map[string]bool)

	chain = func(c string, seen map[string]bool) {
		if c == "" || seen[c] {
			return
		}

		seen[c] = true

		for _, p := range parents[c] {
			chain(p, seen)
		}
	}

	var list []*ItemInfo

	for _, f := range []string{"weapons", "armor", "misc"} {
		tab := readTSV(t, filepath.Join(root, f+".txt"))

		for _, r := range tab.rows {
			code := tab.s(r, "code")
			if code == "" {
				continue
			}

			seen := map[string]bool{}
			chain(tab.s(r, "type"), seen)
			chain(tab.s(r, "type2"), seen)

			info := &ItemInfo{
				Code: code, Level: tab.n(r, "level"), Rarity: tab.n(r, "rarity"),
				Spawnable: tab.n(r, "spawnable") == 1, Quest: tab.n(r, "quest") != 0,
				Unique: tab.n(r, "unique") == 1, MagicLevel: tab.n(r, "magic lvl"),
				Version: tab.n(r, "version"), Throwable: throw[tab.s(r, "type")],
			}

			for c := range seen {
				info.Types = append(info.Types, c)
			}

			info.ClassSpecific = cls[tab.s(r, "type")]
			info.Uber = UberTier(code, tab.s(r, "ubercode"), tab.s(r, "ultracode"), tab.s(r, "type"),
				info.Types, info.Quest)

			info.TypeRarity = rarity[tab.s(r, "type")]
			f := flags[tab.s(r, "type")]
			info.TypeNormal, info.TypeMagic, info.TypeRare = f[0], f[1], f[2]

			list = append(list, info)
			rt.items[code] = info
		}
	}

	tc := readTSV(t, filepath.Join(root, "itemgen", "patch_d2", "TreasureClassEx.txt"))

	for _, r := range tc.rows {
		name := tc.s(r, "Treasure Class")
		if name == "" {
			continue
		}

		c := &TreasureClass{
			Name: name, Group: tc.n(r, "group"), Level: tc.n(r, "level"), Picks: tc.n(r, "Picks"),
			NoDrop: tc.n(r, "NoDrop"),
			Mods: QualityMods{
				Unique: tc.n(r, "Unique"), Set: tc.n(r, "Set"), Rare: tc.n(r, "Rare"), Magic: tc.n(r, "Magic"),
			},
		}

		for i := 1; i <= 10; i++ {
			if code := tc.s(r, "Item"+strconv.Itoa(i)); code != "" {
				c.Entries = append(c.Entries, ParseEntry(code, tc.n(r, "Prob"+strconv.Itoa(i))))
			}
		}

		for i := range c.Entries {
			e := &c.Entries[i]
			if _, isItem := rt.items[e.Code]; isItem {
				continue
			}

			if _, isUnique := rt.uniq[e.Code]; isUnique {
				e.Kind, e.Base, e.Version = EntryUnique, rt.base[e.Code], rt.ver[e.Code]
			} else if _, isSet := rt.sets[e.Code]; isSet {
				e.Kind, e.Base, e.Version = EntrySet, rt.base[e.Code], rt.ver[e.Code]
			}
		}

		rt.tcs.Add(c)
	}

	for _, g := range BuildTypeTreasureClasses(list, typeCodes) {
		if _, exists := rt.tcs.TreasureClass(g.Name); !exists {
			rt.tcs.Add(g)
		}
	}

	ir := readTSV(t, filepath.Join(root, "itemgen", "patch_d2", "ItemRatio.txt"))

	for _, r := range ir.rows {
		if ir.n(r, "Version") == 0 {
			continue // keep the Lord of Destruction rows
		}

		dr := func(n string) DropRatio {
			return DropRatio{ir.n(r, n), ir.n(r, n+"Divisor"), ir.n(r, n+"Min")}
		}

		rt.ratios[[2]bool{ir.n(r, "Class Specific") == 1, ir.n(r, "Uber") == 1}] = &Ratio{
			Unique: dr("Unique"), Rare: dr("Rare"), Set: dr("Set"), Magic: dr("Magic"),
			HiQuality: DropRatio{ir.n(r, "HiQuality"), ir.n(r, "HiQualityDivisor"), 0},
			Normal:    DropRatio{ir.n(r, "Normal"), ir.n(r, "NormalDivisor"), 0},
		}
	}

	return rt
}

func TestRealTablesLoad(t *testing.T) {
	rt := loadReal(t)

	if len(rt.ratios) != 4 {
		t.Fatalf("expected the 4 LoD ItemRatio rows, got %d", len(rt.ratios))
	}

	r := rt.ratios[[2]bool{false, false}]
	if r.Unique != (DropRatio{400, 1, 6400}) || r.Magic != (DropRatio{34, 3, 192}) {
		t.Errorf("ratio row = %+v", r)
	}

	// Every nested reference resolves to a class, a base item, or is a
	// named unique/set item (which we cannot check without those tables).
	unresolved := 0

	for _, tc := range rt.tcs.list {
		for _, e := range tc.Entries {
			if _, isTC := rt.tcs.TreasureClass(e.Code); isTC {
				continue
			}

			if _, isItem := rt.items[e.Code]; !isItem {
				unresolved++
			}
		}
	}

	t.Logf("%d classes, %d base items, %d entries naming uniques/sets/unknown",
		rt.tcs.Len(), len(rt.items), unresolved)

	if unresolved > 400 {
		t.Errorf("suspiciously many unresolved entries: %d", unresolved)
	}

	// The (N-3, N] ranges: no level-3 helm may be in "armo6".
	if tc, ok := rt.tcs.TreasureClass("armo3"); !ok || len(tc.Entries) == 0 {
		t.Error("armo3 should exist and be populated")
	} else {
		for _, e := range tc.Entries {
			if lvl := rt.items[e.Code].Level; lvl < 1 || lvl > 3 {
				t.Errorf("armo3 contains %s of level %d", e.Code, lvl)
			}
		}
	}
}

func TestRealRollDeterministic(t *testing.T) {
	rt := loadReal(t)
	d := &Dropper{TCs: rt.tcs, Items: rt.items, Ratios: rt}

	for _, name := range []string{"Andariel", "Act 1 Equip A", "Mephisto (H)", "Act 5 (H) Good"} {
		if _, ok := rt.tcs.TreasureClass(name); !ok {
			continue
		}

		run := func() []Drop {
			out, err := d.Roll(&Context{RNG: d2rand.New(1234), ILvl: 85, Players: 1, MagicFind: 150}, name)
			if err != nil {
				t.Fatalf("%s: %v", name, err)
			}

			return out
		}

		a, b := run(), run()
		if !reflect.DeepEqual(a, b) {
			t.Errorf("%s: not deterministic", name)
		}

		t.Logf("%s -> %d drops", name, len(a))
	}
}

// TestRealEntryProbabilities rolls leaf classes (only base items) with a fixed
// seed and checks each entry is picked in proportion to Prob/(NoDrop+total),
// and that the probabilities (nodrop included) sum to one.
func TestRealEntryProbabilities(t *testing.T) {
	rt := loadReal(t)
	d := &Dropper{TCs: rt.tcs} // no Items: quality is not rolled

	tested := 0

	for _, tc := range rt.tcs.list {
		if tc.Picks != 1 || len(tc.Entries) < 2 || tc.TotalProb() == 0 || tc.TotalProb() > 400 {
			continue
		}

		leaf := true

		for _, e := range tc.Entries {
			if _, nested := rt.tcs.TreasureClass(e.Code); nested {
				leaf = false
			}
		}

		if !leaf || tested >= 5 {
			continue
		}

		tested++

		space := float64(tc.TotalProb() + tc.NoDrop)
		sum := float64(tc.NoDrop)
		counts := map[string]int{}
		rng := d2rand.New(99)

		const n = 150000

		for i := 0; i < n; i++ {
			out, err := d.Roll(&Context{RNG: rng, MaxDrops: 1}, tc.Name)
			if err != nil {
				t.Fatal(err)
			}

			if len(out) == 1 {
				counts[out[0].Code]++
			}
		}

		byCode := map[string]int{}
		for _, e := range tc.Entries {
			byCode[e.Code] += e.Prob
		}

		for code, prob := range byCode {
			p := float64(prob) / space
			sum += float64(prob)
			obs := float64(counts[code]) / n

			if math.Abs(obs-p) > 5*math.Sqrt(p*(1-p)/n)+1e-4 {
				t.Errorf("%s/%s: observed %.4f, expected %.4f", tc.Name, code, obs, p)
			}
		}

		if sum != space {
			t.Errorf("%s: probabilities sum to %v of %v", tc.Name, sum, space)
		}
	}

	if tested == 0 {
		t.Skip("no suitable leaf class")
	}
}

// TestRealQualityFrequencies rolls many qualities with the real 1.14b
// ItemRatio rows and compares the observed frequencies with the formulas.
func TestRealQualityFrequencies(t *testing.T) {
	rt := loadReal(t)
	ratio := rt.ratios[[2]bool{false, false}]

	checkQualityFrequencies(t, ratio)
}

func TestSyntheticQualityFrequencies(t *testing.T) {
	checkQualityFrequencies(t, testRatio)
}

// expect returns the probability that a tier test hits for a given chance.
func expectHit(chance, mod int) float64 {
	adj := mod * chance / 1024
	if chance == adj || chance-adj < 0 || chance-adj <= qualityFloor {
		return 1
	}

	return float64(qualityFloor) / float64(chance-adj)
}

func checkQualityFrequencies(t *testing.T, r *Ratio) {
	t.Helper()

	type scenario struct {
		name       string
		ilvl, qlvl int
		mf         int
		mods       QualityMods
	}

	scenarios := []scenario{
		{"base", 85, 60, 0, QualityMods{}},
		{"mf300", 85, 60, 300, QualityMods{}},
		{"boss", 85, 60, 0, QualityMods{Unique: 983, Set: 983, Rare: 983, Magic: 1024}},
		{"lowlvl", 20, 20, 0, QualityMods{}},
	}

	const n = 600000

	for _, sc := range scenarios {
		d := sc.ilvl - sc.qlvl

		pu := expectHit(tierChance(r.Unique, d, sc.mf, 250), sc.mods.Unique)
		ps := expectHit(tierChance(r.Set, d, sc.mf, 500), sc.mods.Set)
		pr := expectHit(tierChance(r.Rare, d, sc.mf, 600), sc.mods.Rare)
		pm := expectHit(magicChance(r.Magic, d, sc.mf), sc.mods.Magic)

		want := map[Quality]float64{}
		rest := 1.0
		want[QualityUnique], rest = rest*pu, rest*(1-pu)
		want[QualitySet], rest = rest*ps, rest*(1-ps)
		want[QualityRare], rest = rest*pr, rest*(1-pr)
		want[QualityMagic], rest = rest*pm, rest*(1-pm)

		// UNVERIFIED tail, see RollQuality.
		hq := float64((r.HiQuality.Base - d/r.HiQuality.Divisor) * 128)
		pSup := 1.0
		if hq > 0 {
			pSup = math.Min(1, 128/hq)
		}
		want[QualitySuperior], rest = rest*pSup, rest*(1-pSup)

		nm := float64((r.Normal.Base - d/r.Normal.Divisor) * 128)
		pNormal := 1.0
		if nm > 0 {
			pNormal = math.Min(1, 128/nm)
		}
		want[QualityNormal], want[QualityLow] = rest*pNormal, rest*(1-pNormal)

		rng := d2rand.New(2024)
		got := map[Quality]int{}

		for i := 0; i < n; i++ {
			got[RollQuality(rng, r, QualityInput{
				ILvl: sc.ilvl, QLvl: sc.qlvl, MagicFind: sc.mf, Mods: sc.mods, TypeRare: true,
			})]++
		}

		for q, p := range want {
			obs := float64(got[q]) / n
			sigma := math.Sqrt(p * (1 - p) / n)

			if math.Abs(obs-p) > 5*sigma+1e-4 {
				t.Errorf("%s quality %d: observed %.5f, expected %.5f (sigma %.5f)", sc.name, q, obs, p, sigma)
			}
		}
	}
}
