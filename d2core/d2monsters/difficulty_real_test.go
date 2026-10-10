package d2monsters

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2txt"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2monster"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2resource"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2util"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2asset"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2records"
)

// Real-table checks of the Nightmare / Hell monster rules. They read the
// extracted 1.14b tables from $D2_TABLES (monsters/patch_d2/{monstats,monlvl,
// levels}.txt) and compare, for every monster and every difficulty, what the
// engine computes with what the raw columns say. The expected values are
// derived here straight from the raw tab separated cells (not through the
// record loaders), so a wrong column name in a loader shows up as a mismatch.
// Skipped when D2_TABLES is unset.

type rawTable struct {
	col  map[string]int
	rows [][]string
}

func readRaw(t *testing.T, path string) *rawTable {
	t.Helper()

	buf, err := os.ReadFile(path)
	if err != nil {
		t.Skip(err)
	}

	lines := strings.Split(strings.ReplaceAll(string(buf), "\r", ""), "\n")
	rt := &rawTable{col: map[string]int{}}

	for i, h := range strings.Split(lines[0], "\t") {
		if _, dup := rt.col[strings.ToLower(h)]; !dup {
			rt.col[strings.ToLower(h)] = i
		}
	}

	for _, l := range lines[1:] {
		if strings.TrimSpace(l) != "" {
			rt.rows = append(rt.rows, strings.Split(l, "\t"))
		}
	}

	return rt
}

func (r *rawTable) s(row []string, name string) string {
	i, ok := r.col[strings.ToLower(name)]
	if !ok {
		panic("no column " + name)
	}

	if i >= len(row) {
		return ""
	}

	return row[i]
}

func (r *rawTable) n(row []string, name string) int {
	v, _ := strconv.Atoi(strings.TrimSpace(r.s(row, name)))

	return v
}

func loadReal(t *testing.T) (root string, rm *d2records.RecordManager) {
	t.Helper()

	root = os.Getenv("D2_TABLES")
	if root == "" {
		t.Skip("D2_TABLES not set")
	}

	rm, err := d2records.NewRecordManager(d2util.LogLevelNone)
	if err != nil {
		t.Fatal(err)
	}

	for _, f := range []struct{ path, file string }{
		{d2resource.MonStats, "monstats.txt"},
		{d2resource.MonsterLevel, "monlvl.txt"},
		{d2resource.LevelDetails, "levels.txt"},
	} {
		buf, err := os.ReadFile(filepath.Join(root, "monsters", "patch_d2", f.file))
		if err != nil {
			t.Skip(err)
		}

		if err = rm.Load(f.path, d2txt.LoadDataDictionary(buf)); err != nil {
			t.Fatalf("%s: %v", f.file, err)
		}
	}

	return root, rm
}

// every monster x difficulty x area: level, hit points, defense, attack
// ratings, damage, experience, treasure classes and resistances.
func TestRealDifficultyVitals(t *testing.T) {
	if testing.Short() {
		t.Skip("slow: every monster x difficulty x area")
	}

	root, rm := loadReal(t)

	ms := readRaw(t, filepath.Join(root, "monsters", "patch_d2", "monstats.txt"))
	ml := readRaw(t, filepath.Join(root, "monsters", "patch_d2", "monlvl.txt"))
	lv := readRaw(t, filepath.Join(root, "monsters", "patch_d2", "levels.txt"))

	// monlvl.txt L-* columns: [difficulty]{hp, ac, th, dm, xp}
	sfx := [3]string{"", "(N)", "(H)"}
	monlvl := map[int][3][5]int{}

	for _, r := range ml.rows {
		var e [3][5]int

		for d := 0; d < 3; d++ {
			for i, c := range []string{"HP", "AC", "TH", "DM", "XP"} {
				e[d][i] = ml.n(r, "L-"+c+sfx[d])
			}
		}

		monlvl[ml.n(r, "Level")] = e
	}

	if len(monlvl) < 100 {
		t.Fatalf("monlvl.txt has only %d rows", len(monlvl))
	}

	// the areas whose MonLvl(Ex) the monster level can follow: every row of
	// levels.txt, plus "no area"
	type area struct {
		id  int
		ex  [3]int
		any bool
	}

	areas := []area{{}}

	for _, r := range lv.rows {
		if lv.s(r, "Name") == "Expansion" {
			continue
		}

		areas = append(areas, area{id: lv.n(r, "Id"), any: true,
			ex: [3]int{lv.n(r, "MonLvl1Ex"), lv.n(r, "MonLvl2Ex"), lv.n(r, "MonLvl3Ex")}})
	}

	// areas are many; the monster level only depends on the (N)/(H) pair, so
	// de-duplicate by it to keep the loop small
	seen := map[[3]int]bool{}
	uniq := areas[:1]

	for _, a := range areas[1:] {
		if !seen[a.ex] {
			seen[a.ex] = true
			uniq = append(uniq, a)
		}
	}

	scale := func(a, pct int) int { return a * pct / 100 }
	checked := 0

	for _, row := range ms.rows {
		key := ms.s(row, "Id")
		if key == "Expansion" {
			continue // separator row
		}

		st := statOfRow(rm, ms.n(row, "hcIdx"), key)

		if st == nil {
			t.Errorf("%s: monstats row not loaded", key)

			continue
		}

		noRatio := ms.n(row, "noRatio") > 0
		boss := ms.n(row, "boss") > 0

		for d := 0; d < 3; d++ {
			diff := d2monster.Difficulty(d)
			s := sfx[d]

			// treasure classes, resists (no area needed)
			d0 := difficultyDirector(diff, 0)
			m := &d2mapentity.Monster{Stat: st}

			for kind, col := range []string{"TreasureClass1", "TreasureClass2", "TreasureClass3", "TreasureClass4"} {
				want := ms.s(row, col+s) // "TreasureClass1", "TreasureClass1(N)", ...

				if got := d0.TreasureClassOf(m, kind+1); got != want {
					t.Errorf("%s %v kind %d: TC %q, table %q", key, diff, kind+1, got, want)
				}
			}

			wantRes := [6]int{}
			for i, c := range []string{"ResDm", "ResMa", "ResFi", "ResLi", "ResCo", "ResPo"} {
				wantRes[i] = ms.n(row, c+s)
			}

			if got := MonsterResists(st, diff); got != wantRes {
				t.Errorf("%s %v: resists %v, table %v", key, diff, got, wantRes)
			}

			for _, a := range uniq {
				dir := realDirector(rm, diff, a.id, a.any)
				if dir.statByID[st.ID] != st {
					t.Fatalf("%s: director does not reach class %d", key, st.ID)
				}

				b := d2monster.NewBrain(1, 1, diff, &d2monster.Profile{}, 1)
				v := dir.computeVitals(st, b)

				// level rule (expansion game)
				wantLvl := ms.n(row, "Level"+s)
				if d > 0 && !noRatio && !boss && a.any && a.ex[d] > 0 {
					wantLvl = a.ex[d]
				}

				if v.Level != wantLvl {
					t.Errorf("%s %v area %d: level %d, table %d", key, diff, a.id, v.Level, wantLvl)

					continue
				}

				// scaling numbers: the monlvl row, or 100 for noRatio / missing rows
				n := [5]int{100, 100, 100, 100, 100}
				if r, ok := monlvl[wantLvl]; ok && !noRatio {
					n = r[d]
				}

				hpMin, hpMax := scale(n[0], ms.n(row, "MinHP"+s)), scale(n[0], ms.n(row, "MaxHP"+s))
				if hpMax < hpMin {
					hpMax = hpMin
				}

				if hpMin < 1 {
					hpMin = 1
					if hpMax < 1 {
						hpMax = 1
					}
				}

				if v.MaxHP < hpMin || v.MaxHP > hpMax {
					t.Errorf("%s %v lvl %d: hp %d outside table range %d..%d", key, diff, wantLvl, v.MaxHP, hpMin, hpMax)
				}

				if want := scale(n[1], ms.n(row, "AC"+s)); v.Defense != want {
					t.Errorf("%s %v lvl %d: defense %d, table %d", key, diff, wantLvl, v.Defense, want)
				}

				if want := scale(n[4], ms.n(row, "Exp"+s)); v.Experience != want {
					t.Errorf("%s %v lvl %d: exp %d, table %d", key, diff, wantLvl, v.Experience, want)
				}

				for ai, p := range []string{"A1", "A2"} {
					g := v.A1
					if ai == 1 {
						g = v.A2
					}

					wantMin, wantMax := scale(n[3], ms.n(row, p+"MinD"+s)), scale(n[3], ms.n(row, p+"MaxD"+s))

					if wantMax < wantMin {
						wantMax = wantMin
					}

					// casters take their first element damage as A1 when it has no
					// physical damage (documented UNVERIFIED reading): skip those
					if ai == 0 && wantMax == 0 && ms.s(row, "El1Mode") == "A1" {
						continue
					}

					if g.Min != wantMin || g.Max != wantMax {
						t.Errorf("%s %v lvl %d: %s damage %d-%d, table %d-%d", key, diff, wantLvl, p, g.Min, g.Max,
							wantMin, wantMax)
					}

					wantTH := scale(n[2], ms.n(row, p+"TH"+s))
					if wantTH == 0 && ai == 0 {
						wantTH = scale(n[2], 100)
					}

					if ai == 0 && g.ToHit != wantTH {
						t.Errorf("%s %v lvl %d: A1 to-hit %d, table %d", key, diff, wantLvl, g.ToHit, wantTH)
					}
				}

				checked++
			}
		}
	}

	if checked < len(ms.rows)*3 {
		t.Errorf("only %d combinations checked", checked)
	}

	t.Logf("%d monsters, %d (monster, difficulty, area) combinations", len(ms.rows), checked)
}

// the monster level of a class never indexes past monlvl.txt (a missing row
// would silently fall back to 100% ratios).
func TestRealMonsterLevelsHaveMonlvlRow(t *testing.T) {
	root, rm := loadReal(t)
	ms := readRaw(t, filepath.Join(root, "monsters", "patch_d2", "monstats.txt"))
	lv := readRaw(t, filepath.Join(root, "monsters", "patch_d2", "levels.txt"))

	need := map[int]string{}

	for _, r := range ms.rows {
		if ms.n(r, "noRatio") > 0 {
			continue
		}

		for _, s := range []string{"", "(N)", "(H)"} {
			if l := ms.n(r, "Level"+s); l > 0 {
				need[l] = ms.s(r, "Id")
			}
		}
	}

	for _, r := range lv.rows {
		for _, c := range []string{"MonLvl1Ex", "MonLvl2Ex", "MonLvl3Ex"} {
			if l := lv.n(r, c); l > 0 {
				need[l] = "levels.txt " + lv.s(r, "LevelName")
			}
		}
	}

	for l, who := range need {
		if rm.Monster.Levels[l] == nil {
			t.Errorf("no monlvl.txt row for level %d (%s)", l, who)
		}
	}
}

// the loaded levels.txt MonLvl columns agree with the table, via the
// director's AreaLevelOf, for every area, difficulty and game type.
func TestRealAreaLevelOf(t *testing.T) {
	root, rm := loadReal(t)
	lv := readRaw(t, filepath.Join(root, "monsters", "patch_d2", "levels.txt"))

	for _, r := range lv.rows {
		if lv.s(r, "Name") == "Expansion" {
			continue
		}

		id := lv.n(r, "Id")

		for d := 0; d < 3; d++ {
			for _, exp := range []bool{false, true} {
				col := "MonLvl" + strconv.Itoa(d+1)
				if exp {
					col += "Ex"
				}

				dir := realDirector(rm, d2monster.Difficulty(d), 0, false)
				dir.opt.Expansion = exp

				if got, want := dir.AreaLevelOf(id), lv.n(r, col); got != want {
					t.Errorf("area %d %s: %d, table %d", id, col, got, want)
				}
			}
		}
	}
}

func realDirector(rm *d2records.RecordManager, diff d2monster.Difficulty, areaID int, known bool) *Director {
	d := &Director{asset: &d2asset.AssetManager{Records: rm}, opt: Options{Difficulty: diff, Expansion: true},
		statByID: map[int]*d2records.MonStatRecord{}}

	for _, st := range rm.Monster.Stats {
		d.statByID[st.ID] = st
	}

	for _, st := range rm.Monster.Shadowed {
		if d.statByID[st.ID] == nil {
			d.statByID[st.ID] = st
		}
	}

	if det := rm.GetLevelDetails(areaID); known && det != nil {
		d.areaLevel = [3]int{det.MonsterLevelNormalEx, det.MonsterLevelNightmareEx, det.MonsterLevelHellEx}[diff]
	}

	return d
}

// statOfRow is the loaded record of a monstats row: by key, or, for a key the
// table repeats, the shadowed record with that class index.
func statOfRow(rm *d2records.RecordManager, hcIdx int, key string) *d2records.MonStatRecord {
	if st := rm.Monster.Stats[key]; st != nil && st.ID == hcIdx {
		return st
	}

	for _, st := range rm.Monster.Shadowed {
		if st.ID == hcIdx {
			return st
		}
	}

	return nil
}
