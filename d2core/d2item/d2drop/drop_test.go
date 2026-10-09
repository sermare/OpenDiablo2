package d2drop

import (
	"reflect"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
)

// scriptRNG returns scripted Roll results and records the moduli it was asked.
type scriptRNG struct {
	vals  []uint32
	asked []int32
	coins []bool
}

func (s *scriptRNG) Roll(n int32) uint32 {
	s.asked = append(s.asked, n)

	if len(s.vals) == 0 {
		return 0
	}

	v := s.vals[0]
	s.vals = s.vals[1:]

	return v
}

func (s *scriptRNG) Chance() bool {
	if len(s.coins) == 0 {
		return true
	}

	c := s.coins[0]
	s.coins = s.coins[1:]

	return c
}

var testRatio = &Ratio{
	Unique:    DropRatio{400, 2, 6400},
	Rare:      DropRatio{160, 3, 3200},
	Set:       DropRatio{125, 6, 5600},
	Magic:     DropRatio{30, 16, 192},
	HiQuality: DropRatio{12, 16, 0},
	Normal:    DropRatio{4, 8, 0},
}

func TestDiminishMagicFind(t *testing.T) {
	cases := []struct{ x, f, want int }{
		{100, 250, 100},
		{110, 250, 110},
		{111, 250, 11*250/(11+250) + 100},
		{400, 250, 300*250/550 + 100}, // 236
		{400, 500, 300*500/800 + 100},
		{400, 600, 300*600/900 + 100},
	}

	for _, c := range cases {
		if got := DiminishMagicFind(c.x, c.f); got != c.want {
			t.Errorf("Diminish(%d,%d) = %d, want %d", c.x, c.f, got, c.want)
		}
	}

	if DiminishMagicFind(400, 250) != 236 {
		t.Error("300% MF unique factor should be 136%")
	}
}

func TestQualityDenominators(t *testing.T) {
	// ilvl 85, qlvl 60: d = 25. Unique: (400 - 25/2)*128.
	rng := &scriptRNG{vals: []uint32{0}}
	in := QualityInput{ILvl: 85, QLvl: 60, TypeRare: true}

	if q := RollQuality(rng, testRatio, in); q != QualityUnique {
		t.Fatalf("roll 0 should be unique, got %d", q)
	}

	if want := int32((400 - 12) * 128); rng.asked[0] != want {
		t.Errorf("unique denominator = %d, want %d", rng.asked[0], want)
	}

	// With 1/1024-unit modifier 512 the denominator halves.
	rng = &scriptRNG{vals: []uint32{0}}
	in.Mods.Unique = 512
	RollQuality(rng, testRatio, in)

	if want := int32((400 - 12) * 128 / 2); rng.asked[0] != want {
		t.Errorf("modded unique denominator = %d, want %d", rng.asked[0], want)
	}

	// MF 300 with diminishing: (400-12)*12800/236.
	rng = &scriptRNG{vals: []uint32{0}}
	in.Mods = QualityMods{}
	in.MagicFind = 300
	RollQuality(rng, testRatio, in)

	if want := int32((400 - 12) * 12800 / 236); rng.asked[0] != want {
		t.Errorf("MF unique denominator = %d, want %d", rng.asked[0], want)
	}
}

func TestQualityMinimumClamp(t *testing.T) {
	// Unique min 6400: at huge MF the denominator never goes below it.
	rng := &scriptRNG{vals: []uint32{0}}
	RollQuality(rng, testRatio, QualityInput{ILvl: 85, QLvl: 60, MagicFind: 100000})

	if rng.asked[0] < 6400 {
		t.Errorf("denominator %d below the table minimum", rng.asked[0])
	}
}

func TestQualityModifierSaturates(t *testing.T) {
	// A modifier of 1024 removes the whole denominator: always unique, no roll.
	rng := &scriptRNG{}
	q := RollQuality(rng, testRatio, QualityInput{ILvl: 50, QLvl: 50, Mods: QualityMods{Unique: 1024}})

	if q != QualityUnique || len(rng.asked) != 0 {
		t.Errorf("got quality %d after %d rolls", q, len(rng.asked))
	}
}

func TestQualityEarlyOuts(t *testing.T) {
	rng := &scriptRNG{}

	if RollQuality(rng, testRatio, QualityInput{TypeNormal: true}) != QualityNormal {
		t.Error("normal type")
	}

	if RollQuality(rng, testRatio, QualityInput{Unique: true}) != QualityUnique {
		t.Error("unique base")
	}

	if RollQuality(rng, testRatio, QualityInput{TypeMagic: true, Quest: true}) != QualityUnique {
		t.Error("magic-type quest item")
	}

	if len(rng.asked) != 0 {
		t.Error("early outs must not consume the generator")
	}

	// An always-magic type needs no magic test but still gets the others first.
	rng = &scriptRNG{vals: []uint32{999999, 999999}}
	if RollQuality(rng, testRatio, QualityInput{TypeMagic: true, ILvl: 10, QLvl: 10}) != QualityMagic {
		t.Error("magic type")
	}
}

func TestQualityTail(t *testing.T) {
	// All upper tests fail (rolls >= 128), then HiQ fails -> superior.
	big := uint32(1 << 20)
	rng := &scriptRNG{vals: []uint32{big, big, big, big, 5}}

	if q := RollQuality(rng, testRatio, QualityInput{TypeRare: true, ILvl: 10, QLvl: 10}); q != QualitySuperior {
		t.Errorf("want superior, got %d", q)
	}

	// HiQ passes (>=128), normal roll < 128 -> low.
	rng = &scriptRNG{vals: []uint32{big, big, big, big, big, 5}}
	if q := RollQuality(rng, testRatio, QualityInput{TypeRare: true, ILvl: 10, QLvl: 10}); q != QualityLow {
		t.Errorf("want low, got %d", q)
	}

	rng = &scriptRNG{vals: []uint32{big, big, big, big, big, big}}
	if q := RollQuality(rng, testRatio, QualityInput{TypeRare: true, ILvl: 10, QLvl: 10}); q != QualityNormal {
		t.Errorf("want normal, got %d", q)
	}

	// MF below -99 skips straight to the tail.
	rng = &scriptRNG{vals: []uint32{5}}
	if q := RollQuality(rng, testRatio, QualityInput{MagicFind: -100}); q != QualitySuperior {
		t.Errorf("MF<-99: want superior, got %d", q)
	}
}

func TestAffixLevel(t *testing.T) {
	cases := []struct{ ilvl, qlvl, ml, want int }{
		{50, 20, 0, 40},
		{90, 40, 0, 81}, // 90 >= 99-20: 2*90-99
		{1, 60, 0, 30},  // ilvl raised to qlvl 60: 60-30
		{40, 60, 0, 30}, // max(ilvl, qlvl) term (5bf2ed)
		{99, 0, 0, 99},
		{50, 20, 3, 53},
		{99, 0, 10, 99}, // clamped down
	}

	for _, c := range cases {
		if got := AffixLevel(c.ilvl, c.qlvl, c.ml); got != c.want {
			t.Errorf("AffixLevel(%d,%d,%d) = %d, want %d", c.ilvl, c.qlvl, c.ml, got, c.want)
		}
	}
}

func affixPool() []Affix {
	return []Affix{
		{ID: "low", Spawnable: true, Level: 1, MaxLevel: 10, Frequency: 5, Group: 1, IType: []string{"armo"}},
		{ID: "a", Spawnable: true, Level: 1, Frequency: 3, Group: 2, IType: []string{"armo"}},
		{ID: "a2", Spawnable: true, Level: 1, Frequency: 7, Group: 2, IType: []string{"armo"}},
		{ID: "wrongtype", Spawnable: true, Level: 1, Frequency: 100, Group: 3, IType: []string{"weap"}},
		{ID: "excl", Spawnable: true, Level: 1, Frequency: 100, Group: 4, IType: []string{"armo"}, EType: []string{"helm"}},
		{ID: "high", Spawnable: true, Level: 90, Frequency: 100, Group: 5, IType: []string{"armo"}},
		{ID: "off", Spawnable: false, Level: 1, Frequency: 100, Group: 6, IType: []string{"armo"}},
		{ID: "norare", Spawnable: true, Level: 1, Frequency: 100, Group: 7, IType: []string{"armo"}},
	}
}

func TestPickAffixFilters(t *testing.T) {
	pool := affixPool()
	pool[7].Rare = false
	it := &AffixItem{ILvl: 40, QLvl: 0, Types: []string{"helm", "armo"}, Version: 100, Quality: QualityRare}

	for i := range pool {
		pool[i].Rare = pool[i].ID != "norare" // all but one may be rare
	}

	var got []string

	for i := range pool {
		if pool[i].Eligible(it) {
			got = append(got, pool[i].ID)
		}
	}

	// low: MaxLevel 10 < 40; excl: helm excluded; high: level 90; off: not spawnable.
	if want := []string{"a", "a2"}; !reflect.DeepEqual(got, want) {
		t.Errorf("eligible = %v, want %v", got, want)
	}
}

func TestPickAffixWeightAndGroups(t *testing.T) {
	pool := affixPool()[1:3] // a (3), a2 (7), same group
	it := &AffixItem{ILvl: 40, Types: []string{"armo"}, Version: 100, Quality: QualityMagic}

	// sum = 10, roll over [0,10]: 0..2 -> a, 3..9 -> a2, 10 -> last (a2).
	counts := map[string]int{}

	for r := uint32(0); r <= 10; r++ {
		a := PickAffix(&scriptRNG{vals: []uint32{r}}, pool, it, map[int]bool{}, true)
		counts[a.ID]++
	}

	if counts["a"] != 3 || counts["a2"] != 8 {
		t.Errorf("weights = %v", counts)
	}

	// Used group excludes both.
	if PickAffix(&scriptRNG{}, pool, it, map[int]bool{2: true}, true) != nil {
		t.Error("group exclusion failed")
	}

	// The 50% gate.
	if PickAffix(&scriptRNG{coins: []bool{false}}, pool, it, nil, false) != nil {
		t.Error("gate should fail")
	}
}

func TestMagicAffixesAtLeastOne(t *testing.T) {
	pre := []Affix{{ID: "p", Prefix: true, Spawnable: true, Level: 1, Frequency: 1, Group: 1, IType: []string{"armo"}}}
	suf := []Affix{{ID: "s", Spawnable: true, Level: 1, Frequency: 1, Group: 2, IType: []string{"armo"}}}
	it := &AffixItem{ILvl: 40, Types: []string{"armo"}, Version: 100, Quality: QualityMagic}

	// Prefix gate fails; the suffix is forced (no coin consumed for it).
	rng := &scriptRNG{coins: []bool{false}}
	res := RollMagicAffixes(rng, pre, suf, it)

	if len(res.Prefixes) != 0 || len(res.Suffixes) != 1 {
		t.Errorf("got %d prefixes, %d suffixes", len(res.Prefixes), len(res.Suffixes))
	}

	// Both gates pass.
	res = RollMagicAffixes(&scriptRNG{coins: []bool{true, true}}, pre, suf, it)
	if len(res.Prefixes) != 1 || len(res.Suffixes) != 1 {
		t.Error("expected both")
	}

	// Prefix passes, suffix gate fails.
	res = RollMagicAffixes(&scriptRNG{coins: []bool{true, false}}, pre, suf, it)
	if len(res.Prefixes) != 1 || len(res.Suffixes) != 0 {
		t.Error("expected prefix only")
	}
}

func TestRareAffixCounts(t *testing.T) {
	want := map[uint32]int{0: 3, 1: 4, 2: 4, 3: 5, 4: 5, 5: 5, 6: 6, 7: 6}

	for r, n := range want {
		if got := RareAffixCount(&scriptRNG{vals: []uint32{r}}, false); got != n {
			t.Errorf("count[%d] = %d, want %d", r, got, n)
		}
	}

	if got := RareAffixCount(&scriptRNG{vals: []uint32{1}}, true); got != 4 {
		t.Errorf("jewel count = %d", got)
	}
}

func TestRollRareAffixes(t *testing.T) {
	var pre, suf []Affix

	for i := 1; i <= 8; i++ {
		pre = append(pre, Affix{ID: "p", Prefix: true, Spawnable: true, Rare: true, Level: 1, Frequency: 1, Group: i, IType: []string{"armo"}})
		suf = append(suf, Affix{ID: "s", Spawnable: true, Rare: true, Level: 1, Frequency: 1, Group: 100 + i, IType: []string{"armo"}})
	}

	it := &AffixItem{ILvl: 80, Types: []string{"armo"}, Version: 100, Quality: QualityRare}

	for seed := uint32(1); seed < 500; seed++ {
		res := RollRareAffixes(d2rand.New(seed), pre, suf, it)

		n := len(res.Prefixes) + len(res.Suffixes)
		if n < 3 || n > 6 || len(res.Prefixes) > 3 || len(res.Suffixes) > 3 {
			t.Fatalf("seed %d: %d prefixes %d suffixes", seed, len(res.Prefixes), len(res.Suffixes))
		}
	}

	// Only prefixes exist: the roll stops at 3.
	res := RollRareAffixes(d2rand.New(7), pre, nil, it)
	if len(res.Suffixes) != 0 || len(res.Prefixes) != 3 {
		t.Errorf("prefix-only: %d/%d", len(res.Prefixes), len(res.Suffixes))
	}
}

func TestPickRow(t *testing.T) {
	rows := []Row{
		{ID: "a", Base: "x", Rarity: 3, Level: 1, Enabled: true},
		{ID: "b", Base: "x", Rarity: 1, Level: 1, Enabled: true},
		{ID: "high", Base: "x", Rarity: 50, Level: 80, Enabled: true},
		{ID: "ladder", Base: "x", Rarity: 50, Level: 1, Enabled: true, Ladder: true},
		{ID: "off", Base: "x", Rarity: 50, Level: 1},
		{ID: "other", Base: "y", Rarity: 50, Level: 1, Enabled: true},
		{ID: "done", Base: "x", Rarity: 50, Level: 1, Enabled: true},
	}
	spawned := map[string]bool{"done": true}
	counts := map[string]int{}

	for r := uint32(0); r < 4; r++ {
		p := PickRow(&scriptRNG{vals: []uint32{r}}, rows, "x", 30, 100, false, spawned)
		counts[p.ID]++
	}

	if counts["a"] != 3 || counts["b"] != 1 || len(counts) != 2 {
		t.Errorf("counts %v", counts)
	}

	if PickRow(&scriptRNG{}, rows, "z", 30, 100, false, nil) != nil {
		t.Error("no row for base z")
	}

	rows[6].NoLimit = true

	if p := PickRow(&scriptRNG{vals: []uint32{3}}, rows[6:], "x", 30, 100, false, spawned); p == nil {
		t.Error("nolimit rows can respawn")
	}
}

func TestFallbackQuality(t *testing.T) {
	if FallbackQuality(QualityUnique) != QualityRare || FallbackQuality(QualitySet) != QualityMagic ||
		FallbackQuality(QualityRare) != QualityMagic || FallbackQuality(QualityMagic) != QualitySuperior {
		t.Error("fallback chain")
	}
}
