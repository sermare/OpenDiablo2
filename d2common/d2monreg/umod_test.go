package d2monreg

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
)

const testMonType = "type\tequiv1\tequiv2\tequiv3\tstrsing\tstrplur\t*eol\n" +
	"\t\t\t\t\t\t0\n" +
	"undead\t\t\t\t\t\t0\n" +
	"skeleton\tundead\t\t\t\t\t0\n" +
	"leaper\t\t\t\t\t\t0\n"

const testMonUMod = "uniquemod\tid\tenabled\tversion\txfer\tchampion\tfPick\texclude1\texclude2\tcpick\tcpick (N)\tcpick (H)\tupick\tupick (N)\tupick (H)\tconstants\n" +
	"none\t0\t0\t0\t1\t\t\t\t\t\t\t\t\t\t\t20\n" +
	"a\t1\t1\t0\t1\t\t\t\t\t\t\t\t5\t5\t5\t0\n" +
	"b\t2\t1\t0\t0\t\t\t\t\t\t\t\t3\t3\t3\t0\n" +
	"champ\t3\t1\t0\t1\t1\t\t\t\t1\t1\t1\t\t\t\t0\n" +
	"exp\t4\t1\t100\t0\t\t\t\t\t\t\t\t4\t4\t4\t0\n" +
	"fast\t5\t1\t0\t0\t\t3\t\t\t\t\t\t6\t6\t6\t0\n" +
	"multi\t6\t1\t0\t0\t\t2\t\t\t\t\t\t6\t6\t6\t0\n" +
	"light\t7\t1\t0\t0\t\t\tundead\t\t\t\t\t6\t6\t6\t0\n"

const testMonStats = "Id\tBaseId\tNextInClass\tMonStatsEx\tspawn\tminion1\tminion2\tPartyMin\tPartyMax\tRarity\tMinGrp\tMaxGrp\tsparsePopulate\tLevel\tLevel(N)\tLevel(H)\tMonType\tisMelee\tisSpawn\n" +
	"walker\t\t\twalker\t\t\t\t0\t0\t1\t1\t1\t0\t1\t1\t1\tskeleton\t1\t1\n" +
	"shooter\t\t\tshooter\t\t\t\t0\t0\t1\t1\t1\t0\t1\t1\t1\tleaper\t0\t1\n"

const testMonStats2 = "Id\tSizeX\tSizeY\tspawnCol\tmDT\tmNU\tmWL\tmGH\tmA1\tmA2\n" +
	"walker\t1\t1\t0\t1\t1\t1\t1\t1\t1\n" +
	"shooter\t1\t1\t0\t1\t1\t0\t1\t1\t1\n"

const testLevels = "Name\tId\tAct\tMonDen\tMonUMin\tMonUMax\tNumMon\n" +
	"x\t1\t0\t1\t0\t0\t1\n"

func synthTables(t *testing.T) *Tables {
	t.Helper()

	tb, err := ParseTables([]byte(testMonStats), []byte(testMonStats2), []byte(testLevels))
	if err != nil {
		t.Fatal(err)
	}

	if err := tb.LoadUMods([]byte(testMonUMod), []byte(testMonType)); err != nil {
		t.Fatal(err)
	}

	return tb
}

func TestUModEligible(t *testing.T) {
	tb := synthTables(t)
	walker, shooter := tb.MonByKey("walker"), tb.MonByKey("shooter")

	cases := []struct {
		name      string
		row       int
		class     int
		expansion bool
		want      bool
	}{
		{"plain row", 1, walker, false, true},
		{"disabled row 0", 0, walker, false, false},
		{"expansion row in classic", 4, walker, false, false},
		{"expansion row in lod", 4, walker, true, true},
		{"fPick 3 needs the walk mode: walker has it", 5, walker, false, true},
		{"fPick 3 shooter has no walk mode", 5, shooter, false, false},
		{"fPick 2 (multishot) refuses melee classes", 6, walker, false, false},
		{"fPick 2 allows ranged classes", 6, shooter, false, true},
		{"exclude1 undead rejects a skeleton (equiv chain)", 7, walker, false, false},
		{"exclude1 undead keeps other types", 7, shooter, false, true},
	}

	for _, c := range cases {
		if got := tb.UMods.Eligible(tb, c.row, c.class, c.expansion); got != c.want {
			t.Errorf("%s: got %v want %v", c.name, got, c.want)
		}
	}
}

func TestUModRollShape(t *testing.T) {
	tb := synthTables(t)
	walker := tb.MonByKey("walker")

	if got := tb.UMods.ChampionChance(); got != 20 {
		t.Fatalf("champion chance %d", got)
	}

	champs := 0

	for s := uint32(1); s <= 4000; s++ {
		mods, champ := tb.UMods.Roll(tb, d2rand.New(s), 0, walker, false, true, nil)

		if champ {
			champs++

			// the only champion row of a classic game
			if !reflect.DeepEqual(mods, []int{3}) {
				t.Fatalf("champion mods %v", mods)
			}

			continue
		}

		if len(mods) != 1 { // normal: Roll(1)+1+0 = 1 modifier
			t.Fatalf("seed %d: normal rare got %v", s, mods)
		}

		if m := mods[0]; m == 0 || m == 3 || m == 4 || m == 6 || m == 7 {
			// 3 is a champion row, 4 expansion-only, 6 multishot (melee), 7 excluded undead
			t.Fatalf("seed %d: ineligible modifier %d", s, m)
		}
	}

	if frac := float64(champs) / 4000; frac < 0.17 || frac > 0.23 {
		t.Errorf("champion share %.3f, want about 0.20", frac)
	}

	// Hell: three modifiers, all distinct
	for s := uint32(1); s <= 500; s++ {
		mods, _ := tb.UMods.Roll(tb, d2rand.New(s), 2, tb.MonByKey("shooter"), true, false, nil)
		if len(mods) != 3 {
			t.Fatalf("hell rare got %v", mods)
		}

		sort.Ints(mods)

		for i := 1; i < len(mods); i++ {
			if mods[i] == mods[i-1] {
				t.Fatalf("duplicate modifier in %v", mods)
			}
		}
	}
}

func TestUModWeights(t *testing.T) {
	tb := synthTables(t)
	walker := tb.MonByKey("walker")
	count := map[int]int{}

	for s := uint32(1); s <= 9000; s++ {
		count[tb.UMods.PickNonBoss(tb, d2rand.New(s), 0, walker, false, nil)]++
	}

	// eligible rows for a classic walker: 1 (upick 5), 2 (3), 5 (6): shares 5/14, 3/14, 6/14
	for row, want := range map[int]float64{1: 5.0 / 14, 2: 3.0 / 14, 5: 6.0 / 14} {
		if got := float64(count[row]) / 9000; got < want-0.03 || got > want+0.03 {
			t.Errorf("row %d share %.3f want %.3f", row, got, want)
		}
	}

	// an empty pool returns 0 and takes no seed step
	used := map[int]bool{1: true, 2: true, 5: true}
	seed := d2rand.New(7)
	before := *seed

	if id := tb.UMods.PickNonBoss(tb, seed, 0, walker, false, used); id != 0 || *seed != before {
		t.Errorf("empty pool: id %d, seed moved %v", id, *seed != before)
	}
}

func TestUModXfer(t *testing.T) {
	tb := synthTables(t)

	// rows 1 and 3 are xfer, row 2 is not
	if got := tb.UMods.XferMods([]int{1, 2, 3}); !reflect.DeepEqual(got, []int{1, 3}) {
		t.Errorf("xfer mods %v", got)
	}
}

func realUModTables(t *testing.T) *Tables {
	t.Helper()

	tb := realTables(t)
	dir := filepath.Join(os.Getenv("D2_TABLES"), "monsters", "patch_d2")

	mu, err1 := os.ReadFile(filepath.Join(dir, "monumod.txt"))
	mt, err2 := os.ReadFile(filepath.Join(dir, "montype.txt"))

	if err1 != nil || err2 != nil {
		t.Skip("monumod.txt / montype.txt not under D2_TABLES/monsters/patch_d2")
	}

	if err := tb.LoadUMods(mu, mt); err != nil {
		t.Fatal(err)
	}

	return tb
}

// The real monumod table: the shape the exe decompilation gives.
func TestRealUModTable(t *testing.T) {
	tb := realUModTables(t)
	u := tb.UMods

	if u.ChampionChance() != 20 {
		t.Errorf("champion chance %d, want 20", u.ChampionChance())
	}

	boss := func(expansion bool) []int {
		var ids []int

		for i := range u.Rows {
			if u.Rows[i].Champion && u.Rows[i].CPick[0] > 0 && u.Rows[i].Enabled && (expansion || u.Rows[i].Version < 100) {
				ids = append(ids, i)
			}
		}

		return ids
	}

	if got := boss(false); !reflect.DeepEqual(got, []int{16}) {
		t.Errorf("classic champion rows %v, want [16]", got)
	}

	if got := boss(true); !reflect.DeepEqual(got, []int{16, 36, 37, 38, 39}) {
		t.Errorf("expansion champion rows %v", got)
	}

	// rows 1 and 2 (name seed, hit points) are never picked: no upick weight
	if u.Rows[1].UPick[0] != 0 || u.Rows[2].UPick[0] != 0 {
		t.Error("rows 1/2 must have no upick weight")
	}

	if u.Rows[16].Name != "champion" || u.Rows[17].Name != "lightning" {
		t.Errorf("row names %q %q: ids must equal row indices", u.Rows[16].Name, u.Rows[17].Name)
	}

	// lightning enchanted excludes the sand leaper type and nothing else
	leaper := tb.MonByKey("sandleaper1")
	if leaper < 0 {
		t.Skip("no sandleaper1")
	}

	if u.Eligible(tb, 17, leaper, false) {
		t.Error("sand leaper must not be lightning enchanted")
	}

	if !u.Eligible(tb, 17, tb.MonByKey("zombie1"), false) {
		t.Error("zombie1 may be lightning enchanted")
	}

	// multishot (fPick 2) needs a ranged class
	if u.Eligible(tb, 29, tb.MonByKey("zombie1"), false) {
		t.Error("a melee zombie must not get multishot")
	}
}

// Loading the monumod table must not move any seed of the emulator-compared
// creation sequence: every pick still takes exactly one step.
func TestOracleNaturalWithRealMods(t *testing.T) {
	tb := realUModTables(t)

	files, _ := filepath.Glob(filepath.Join("testdata", "natural_*.json.gz"))
	if len(files) == 0 {
		t.Skip("no goldens")
	}

	for _, f := range files {
		f := f
		t.Run(filepath.Base(f), func(t *testing.T) { checkNatural(t, tb, f) })
	}
}
