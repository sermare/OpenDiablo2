package d2drop

import "testing"

func TestEligibleBoundaries(t *testing.T) {
	base := Affix{Spawnable: true, Level: 20, MaxLevel: 40, Rare: true, Frequency: 1, IType: []string{"armo"}}

	cases := []struct {
		name string
		mod  func(a *Affix, it *AffixItem)
		want bool
	}{
		{"alvl equals level", func(a *Affix, it *AffixItem) { it.ILvl = 20 }, true},
		{"alvl below level", func(a *Affix, it *AffixItem) { it.ILvl = 19 }, false},
		{"alvl equals maxlevel", func(a *Affix, it *AffixItem) { it.ILvl = 40 }, true},
		{"alvl above maxlevel", func(a *Affix, it *AffixItem) { it.ILvl = 41 }, false},
		{"alvl uses qlvl/2", func(a *Affix, it *AffixItem) { it.ILvl = 30; it.QLvl = 22 }, false}, // 30-11=19
		{"magic level lifts alvl", func(a *Affix, it *AffixItem) { it.ILvl = 18; it.MagicLevel = 2 }, true},
		{"classic item, LoD affix", func(a *Affix, it *AffixItem) { a.Version = 100; it.Version = 0 }, false},
		{"LoD item, LoD affix", func(a *Affix, it *AffixItem) { a.Version = 100 }, true},
		{"rare-only flag blocks rare", func(a *Affix, it *AffixItem) { a.Rare = false; it.Quality = QualityRare }, false},
		{"rare-only flag blocks crafted", func(a *Affix, it *AffixItem) { a.Rare = false; it.Quality = QualityCrafted }, false},
		{"rare flag irrelevant on magic", func(a *Affix, it *AffixItem) { a.Rare = false; it.Quality = QualityMagic }, true},
		{"class restriction mismatch", func(a *Affix, it *AffixItem) { a.Class = "ama"; it.Class = "sor" }, false},
		{"class restriction match", func(a *Affix, it *AffixItem) { a.Class = "ama"; it.Class = "ama" }, true},
		{"class restriction on classless item", func(a *Affix, it *AffixItem) { a.Class = "ama" }, false},
		{"excluded ancestor", func(a *Affix, it *AffixItem) { a.EType = []string{"armo"} }, false},
	}

	for _, c := range cases {
		a := base
		it := &AffixItem{ILvl: 30, Types: []string{"armo"}, Version: 100, Quality: QualityMagic}
		c.mod(&a, it)

		if got := a.Eligible(it); got != c.want {
			t.Errorf("%s: Eligible = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestMagicLevelWeightsByLevel(t *testing.T) {
	pool := []Affix{
		{ID: "x", Spawnable: true, Level: 2, Frequency: 3, Group: 1, IType: []string{"armo"}},
		{ID: "y", Spawnable: true, Level: 10, Frequency: 1, Group: 2, IType: []string{"armo"}},
	}
	it := &AffixItem{ILvl: 30, MagicLevel: 1, Types: []string{"armo"}, Version: 100, Quality: QualityMagic}

	// weights: 3*2=6 and 1*10=10; roll over [0,16].
	counts := map[string]int{}

	for r := uint32(0); r <= 16; r++ {
		counts[PickAffix(&scriptRNG{vals: []uint32{r}}, pool, it, nil, true).ID]++
	}

	if counts["x"] != 6 || counts["y"] != 11 {
		t.Errorf("weights = %v", counts)
	}

	rng := &scriptRNG{vals: []uint32{0}}
	PickAffix(rng, pool, it, nil, true)

	if rng.asked[0] != 17 {
		t.Errorf("roll modulus should be sum+1, asked %v", rng.asked)
	}
}

func TestRareGroupsNotRepeated(t *testing.T) {
	pre := []Affix{
		{ID: "p1", Prefix: true, Spawnable: true, Rare: true, Level: 1, Frequency: 1, Group: 1, IType: []string{"armo"}},
		{ID: "p2", Prefix: true, Spawnable: true, Rare: true, Level: 1, Frequency: 1, Group: 1, IType: []string{"armo"}},
	}
	it := &AffixItem{ILvl: 50, Types: []string{"armo"}, Version: 100, Quality: QualityRare}

	res := RollRareAffixes(&scriptRNG{vals: []uint32{0}}, pre, nil, it)
	if len(res.Prefixes) != 1 {
		t.Fatalf("got %d prefixes from a single group", len(res.Prefixes))
	}
}

func TestPickRareName(t *testing.T) {
	if PickRareName(&scriptRNG{}, 0) != -1 {
		t.Error("empty table")
	}

	r := &scriptRNG{vals: []uint32{4}}
	if PickRareName(r, 5) != 4 || r.asked[0] != 5 {
		t.Errorf("asked %v", r.asked)
	}
}

func TestPickAutoMagic(t *testing.T) {
	pool := []Affix{
		{ID: "g5a", Spawnable: true, Level: 1, Frequency: 1, Group: 5, IType: []string{"armo"}},
		{ID: "g5b", Spawnable: true, Level: 1, Frequency: 1, Group: 5, IType: []string{"armo"}},
		{ID: "g6", Spawnable: true, Level: 1, Frequency: 1, Group: 6, IType: []string{"armo"}},
	}
	it := &AffixItem{ILvl: 10, Types: []string{"armo"}, Version: 100, Quality: QualityNormal}

	if PickAutoMagic(&scriptRNG{}, pool, it, 0) != nil {
		t.Error("no auto prefix")
	}

	// A failed gate coin must not matter: automagic is forced.
	a := PickAutoMagic(&scriptRNG{coins: []bool{false}, vals: []uint32{1}}, pool, it, 5)
	if a == nil || a.ID != "g5b" {
		t.Errorf("got %+v", a)
	}

	if PickAutoMagic(&scriptRNG{}, pool, it, 9) != nil {
		t.Error("unknown group")
	}
}

func TestAutoMagicQuality(t *testing.T) {
	yes := []Quality{QualityLow, QualityNormal, QualitySuperior, QualityMagic, QualityRare, QualityCrafted, 9}
	no := []Quality{QualityNone, QualitySet, QualityUnique}

	for _, q := range yes {
		if !AutoMagicQuality(q) {
			t.Errorf("quality %d should get automagic", q)
		}
	}

	for _, q := range no {
		if AutoMagicQuality(q) {
			t.Errorf("quality %d should not", q)
		}
	}
}
