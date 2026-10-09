package d2monstats

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// oldScale is the pre-wiring engine model (d2monster.ResolveLevel without
// difficulty/expansion check, then base*pct/100), kept here as the oracle for
// the differential test below. Player count 1, expansion columns.
func (t MonLvl) oldScale(c *Class, diff, area int) Stats {
	lvl := c.Level[diff]
	if !(c.NoRatio || c.Boss || area <= 0) {
		lvl = area
	}

	row := t[clamp(lvl, 0, len(t)-1)]
	sc := func(base, pct int) int {
		if c.NoRatio {
			return pct
		}

		return base * pct / 100
	}

	s := Stats{Level: lvl}
	s.HPMin, s.HPMax = sc(row.HP[1][diff], c.MinHP[diff]), sc(row.HP[1][diff], c.MaxHP[diff])
	if s.HPMax < s.HPMin { // vitals.go clamped max up to min
		s.HPMax = s.HPMin
	}

	s.AC, s.XP = sc(row.AC[1][diff], c.AC[diff]), sc(row.XP[1][diff], c.Exp[diff])
	s.TH = sc(row.TH[1][diff], c.Attacks[A1][diff].TH)

	return s
}

// TestDifferentialOldVsNew compares the old engine model with the verified one
// for every class, difficulty and a spread of area levels (expansion game, one
// player) and lists every differing value. The only legitimate difference is
// the level rule: Normal never takes the area level.
func TestDifferentialOldVsNew(t *testing.T) {
	dir := os.Getenv("D2_TABLES")
	if dir == "" {
		t.Skip("D2_TABLES not set")
	}

	read := func(n string) []byte {
		b, err := os.ReadFile(filepath.Join(dir, "monsters", "patch_d2", n))
		if err != nil {
			t.Skip(err)
		}

		return b
	}

	ml, err := LoadMonLvl(read("monlvl.txt"))
	if err != nil {
		t.Fatal(err)
	}

	cl, err := LoadClasses(read("monstats.txt"))
	if err != nil {
		t.Fatal(err)
	}

	ids := make([]string, 0, len(cl))
	for id := range cl {
		ids = append(ids, id)
	}

	sort.Strings(ids)

	var total, differing int

	for _, id := range ids {
		c := cl[id]

		for diff := Normal; diff <= Hell; diff++ {
			for _, area := range []int{0, 1, 12, 25, 40, 55, 70, 85, 99} {
				total++

				o := ml.oldScale(c, diff, area)
				n := ml.Scale(c, diff, area, true, func(int) int { return 0 })
				ok := o.Level == n.Level && o.HPMin == n.HPMin && o.HPMax == n.HPMax &&
					o.AC == n.AC && o.XP == n.XP && o.TH == n.TH

				if ok {
					continue
				}

				differing++
				t.Logf("DIFF %s diff=%d area=%d: level %d->%d hp %d-%d -> %d-%d ac %d->%d xp %d->%d th %d->%d",
					id, diff, area, o.Level, n.Level, o.HPMin, o.HPMax, n.HPMin, n.HPMax, o.AC, n.AC, o.XP, n.XP, o.TH, n.TH)

				// Expected cause: Normal, ordinary class, area level differs from the stat level.
				if diff != Normal || c.NoRatio || c.Boss || area <= 0 || o.Level != area || n.Level != c.Level[Normal] {
					t.Errorf("unexplained difference for %s diff=%d area=%d", id, diff, area)
				}
			}
		}
	}

	t.Logf("%d cases, %d differ (all: Normal no longer takes the area level)", total, differing)
}
