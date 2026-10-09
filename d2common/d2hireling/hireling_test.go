package d2hireling

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
)

func loadReal(t *testing.T) *Table {
	t.Helper()

	dir := os.Getenv("D2_TABLES")
	if dir == "" {
		t.Skip("D2_TABLES not set")
	}

	// the 1.14b patch_d2.mpq hireling.txt (extracted with mpqcli): the table the
	// game uses, with a Version column (0 classic, 100 expansion)
	f, err := os.Open(filepath.Join(dir, "hireling", "hireling_patch_d2.txt"))
	if err != nil {
		t.Skip("no hireling_patch_d2.txt:", err)
	}
	defer f.Close()

	tab, err := Parse(f)
	if err != nil {
		t.Fatal(err)
	}

	if len(tab.Rows) != 120 {
		t.Fatalf("rows = %d, want 120 (60 classic + 60 expansion)", len(tab.Rows))
	}

	return tab.ForVersion(ExpansionVersion)
}

func TestRealStatsHandComputed(t *testing.T) {
	tab := loadReal(t)

	if len(tab.Rows) != 60 {
		t.Fatalf("rows = %d, want 60", len(tab.Rows))
	}

	tests := []struct {
		name      string
		id, level int
		want      Stats
	}{
		// rogue fire, row 36, d=14: HP 342+18*14, Str 77+10*14/8, Dex 111+16*14/8, Def 279+15*14,
		// AR 406+24*14, dmg 9+4*14/8 / 11+7, resist 66+7*14/4, exp 51*50^2*100
		{"rogue50", 0, 50, Stats{Level: 50, MaxHP: 594, Str: 94, Dex: 139, Defense: 489, AR: 742,
			DmgMin: 16, DmgMax: 18, Resist: 90, Experience: 12750000, NextXP: 52 * 51 * 51 * 100}},
		// the base row itself
		{"rogue36", 0, 36, Stats{Level: 36, MaxHP: 342, Str: 77, Dex: 111, Defense: 279, AR: 406,
			DmgMin: 9, DmgMax: 11, Resist: 66, Experience: 37 * 36 * 36 * 100, NextXP: 38 * 37 * 37 * 100}},
		// desert off-nightmare (type 11, nokkasorc's merc), row 75, d=19
		{"desert94", 11, 94, Stats{Level: 94, MaxHP: 2127, Str: 202, Dex: 163, Defense: 1517, AR: 1837,
			DmgMin: 65, DmgMax: 72, Resist: 158, Experience: 100730400, NextXP: 96 * 95 * 95 * 120}},
		// Iron Wolf fire, row 15, d=25
		{"wolf40", 15, 40, Stats{Level: 40, MaxHP: 385, Str: 80, Dex: 65, Defense: 205, AR: 315,
			DmgMin: 13, DmgMax: 19, Resist: 68, Experience: 41 * 1600 * 110, NextXP: 42 * 41 * 41 * 110}},
	}

	for _, tc := range tests {
		got, rec := tab.StatsFor(tc.id, tc.level)
		if rec == nil || got != tc.want {
			t.Errorf("%s: got %+v, want %+v", tc.name, got, tc.want)
		}
	}
}

func TestRealNokkaMercLevel(t *testing.T) {
	tab := loadReal(t)

	// nokkasorc: desert Off-Nightmare (type 11), exp 100730580 = threshold(94)
	// + 180, so level 94 (the hero is level 94 too; the brief's "96" is not
	// what the formula gives).
	if got := tab.LevelFromExp(11, 100730580); got != 94 {
		t.Errorf("level = %d", got)
	}

	if got := tab.LevelFromExp(11, 100730399); got != 93 {
		t.Errorf("level just below the threshold = %d", got)
	}

	if got := tab.StartExp(11, 94); got != 100730400 {
		t.Errorf("StartExp = %d", got)
	}
}

func TestRealOffers(t *testing.T) {
	tab := loadReal(t)

	for _, tc := range []struct{ seller, names int }{{150, 41}, {198, 21}, {252, 20}, {515, 67}} {
		c := tab.Candidates(tc.seller, 1)
		if len(c) == 0 || NameCount(c[0]) != tc.names {
			t.Errorf("seller %d: names %v", tc.seller, c)
		}
	}

	// Kashya normal offers the two level-3 variants only
	if c := tab.Candidates(150, 1); len(c) != 2 || c[0].ID != 0 || c[1].ID != 1 {
		t.Errorf("kashya candidates %v", c)
	}

	o := tab.NewOfferTable(d2rand.New(12345), 150, 1)
	if len(o.Offered()) != OffersShown {
		t.Fatalf("offered %d", len(o.Offered()))
	}

	offer, _ := tab.MakeOffer(o, o.Offered()[0], 30)
	if offer.Stats.Level < 25 || offer.Stats.Level > 29 {
		t.Errorf("offer level %d", offer.Stats.Level)
	}

	// rogue Gold 100, base level 3: cost (15*(lvl-3)+100)
	if want := 15*(offer.Stats.Level-3) + 100; offer.Cost != want {
		t.Errorf("cost %d want %d", offer.Cost, want)
	}
}

const mini = "Hireling\tSubType\tId\tClass\tAct\tDifficulty\tLevel\tSeller\tNameFirst\tNameLast\tGold\tExp/Lvl\tHP\tHP/Lvl\t" +
	"DefaultChance\tSkill1\tMode1\tChance1\tChancePerLevel1\tLevel1\tLvlPerLvl1\tSkill2\tMode2\tChance2\tChancePerLevel2\tLevel2\tLvlPerLvl2\n" +
	"R\tFire\t0\t271\t1\t1\t3\t150\tmerc01\tmerc41\t100\t100\t45\t8\t75\tInner Sight\t4\t10\t0\t1\t10\tFire Arrow\t4\t25\t2\t1\t10\n" +
	"R\tFire\t0\t271\t1\t1\t25\t150\tmerc01\tmerc41\t100\t100\t221\t8\t75\tInner Sight\t4\t10\t0\t7\t10\tFire Arrow\t4\t69\t0\t7\t10\n"

func miniTable(t *testing.T) *Table {
	t.Helper()

	tab, err := Parse(strings.NewReader(mini))
	if err != nil {
		t.Fatal(err)
	}

	return tab
}

func TestCosts(t *testing.T) {
	tab := miniTable(t)
	r := tab.Rows[0]

	for _, tc := range []struct{ level, want int }{{2, 100}, {3, 100}, {4, 115}, {30, 505}} {
		if got := HireCost(r, tc.level); got != tc.want {
			t.Errorf("HireCost(%d) = %d, want %d", tc.level, got, tc.want)
		}
	}

	for _, tc := range []struct{ lvl, want int }{{1, 0}, {10, 750}, {30, 6750}, {80, 48000}, {85, 50000}, {99, 50000}} {
		if got := ReviveCost(tc.lvl); got != tc.want {
			t.Errorf("ReviveCost(%d) = %d, want %d", tc.lvl, got, tc.want)
		}
	}

	for _, tc := range []struct{ owner, roll, want int }{{30, 0, 25}, {30, 4, 29}, {3, 0, 2}, {1, 4, 2}} {
		if got := OfferLevel(tc.owner, tc.roll); got != tc.want {
			t.Errorf("OfferLevel(%d,%d) = %d, want %d", tc.owner, tc.roll, got, tc.want)
		}
	}
}

func TestLevelFromExpRoundTrip(t *testing.T) {
	tab := miniTable(t)

	for _, l := range []int{1, 2, 10, 25, 60, 98} {
		exp := uint32(ExpThreshold(l, 100))
		if l == 1 {
			exp = 0
		}

		if got := tab.LevelFromExp(0, exp); got != l {
			t.Errorf("LevelFromExp(thr(%d)) = %d", l, got)
		}

		if l > 1 {
			if got := tab.LevelFromExp(0, exp-1); got != l-1 {
				t.Errorf("LevelFromExp(thr(%d)-1) = %d", l, got)
			}
		}
	}
}

func TestOfferTableDeterministic(t *testing.T) {
	tab := miniTable(t)
	a := tab.NewOfferTable(d2rand.New(99), 150, 1)
	b := tab.NewOfferTable(d2rand.New(99), 150, 1)

	if len(a.Slots) != 41 || len(a.Offered()) != 10 {
		t.Fatalf("slots %d offered %d", len(a.Slots), len(a.Offered()))
	}

	for i := range a.Slots {
		if a.Slots[i] != b.Slots[i] {
			t.Fatal("not deterministic")
		}
	}

	// hiring all ten regenerates
	g := d2rand.New(5)
	for _, i := range a.Offered() {
		a.Slots[i].Hired = true
	}

	if !a.Regenerate(tab, g) || len(a.Offered()) != 10 {
		t.Error("expected regenerated table with ten offers")
	}

	if tab.NewOfferTable(g, 999, 1) != nil {
		t.Error("unknown seller must have no table")
	}
}

func TestNameKey(t *testing.T) {
	r := miniTable(t).Rows[0]

	for _, tc := range []struct {
		id   int
		want string
	}{{0, "merc01"}, {7, "merc08"}, {40, "merc41"}, {41, "merc01"}} {
		if got := NameKey(r, tc.id); got != tc.want {
			t.Errorf("NameKey(%d) = %s, want %s", tc.id, got, tc.want)
		}
	}
}

func TestChooseSkill(t *testing.T) {
	r := miniTable(t).Rows[1] // level 25, d=0: Default 75, Inner Sight 10, Fire Arrow 69
	fixed := func(v int) func(int) int { return func(int) int { return v } }

	for _, tc := range []struct {
		roll    int
		def     bool
		slot    int
		reqLvl  int
		blocked string
	}{
		{74, true, 0, 0, ""}, {75, false, 0, 0, ""}, {85, false, 0, 0, ""}, {86, false, 1, 0, ""}, {154, false, 1, 0, ""},
		{75, false, 1, 0, "Inner Sight"}, // buff already active: only Fire Arrow competes
	} {
		env := SkillEnv{StateActive: func(n string) bool { return n == tc.blocked }}
		c := r.ChooseSkill(25, env, fixed(tc.roll))

		if c.Default != tc.def || (!tc.def && c.Slot != tc.slot) {
			t.Errorf("roll %d blocked %q: %+v", tc.roll, tc.blocked, c)
		}
	}

	// the roll range is DefaultChance + sum of eligible chances + 1
	var n int

	r.ChooseSkill(25, SkillEnv{}, func(k int) int { n = k; return 0 })

	if n != 155 {
		t.Errorf("roll range %d, want 155", n)
	}

	// reqlevel gates skills
	c := r.ChooseSkill(25, SkillEnv{ReqLevel: func(string) int { return 99 }}, fixed(80))
	if !c.Default {
		t.Errorf("no eligible skill must fall back to the attack: %+v", c)
	}

	// skill level: Fire Arrow Level 7 at row 25 + 10*d>>5
	if got := r.SkillLevel(1, 25+32); got != 7+10 {
		t.Errorf("SkillLevel = %d", got)
	}
}
