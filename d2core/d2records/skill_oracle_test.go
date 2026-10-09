package d2records

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2calc"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2combat"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2txt"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2skill"
)

// The golden file holds numbers only. It was produced by running the real
// Game.exe 1.14b skill calc interpreter (0x648050) and field evaluator
// (0x6477d0) in an x86 emulator over every skills.txt calc cell and every
// skillcalc.txt field code (see docs/skill-oracle.md).

type skillCalcGolden struct {
	ULvl    int                                    `json:"ulvl"`
	Levels  []int                                  `json:"levels"`
	FLevels []int                                  `json:"flevels"`
	Eff     []int                                  `json:"eff"`
	Calcs   map[string]map[string]map[string][]int `json:"calcs"`
	Fields  map[string]map[string]map[string][]int `json:"fields"`
}

// oracleCalcColumns are the skills.txt calc columns the oracle evaluates.
var oracleCalcColumns = []string{
	"prgcalc1", "prgcalc2", "prgcalc3", "auralencalc", "aurarangecalc",
	"aurastatcalc1", "aurastatcalc2", "aurastatcalc3", "aurastatcalc4", "aurastatcalc5", "aurastatcalc6",
	"passivecalc1", "passivecalc2", "passivecalc3", "passivecalc4", "passivecalc5", "petmax",
	"sumsk1calc", "sumsk2calc", "sumsk3calc", "sumsk4calc", "sumsk5calc",
	"cltcalc1", "cltcalc2", "cltcalc3", "perdelay", "calc1", "calc2", "calc3", "calc4",
	"skpoints", "delay", "ToHitCalc", "DmgSymPerCalc", "EDmgSymPerCalc", "ELenSymPerCalc",
}

// oracleFieldCodes are the rows of skillcalc.txt in order (the id the game
// compiles a field code to).
var oracleFieldCodes = []string{
	"ln12", "dm12", "ln34", "dm34", "ln56", "dm56", "ln78", "dm78",
	"par1", "par2", "par3", "par4", "par5", "par6", "par7", "par8",
	"lvl", "edmn", "edmx", "edln", "toht", "mana", "mps", "math", "madm", "macr",
	"m1en", "m1ex", "m1el", "m2en", "m2ex", "m2el", "m3en", "m3ex", "m3el",
	"m1rn", "m2rn", "m3rn", "edns", "edxs", "ulvl", "blvl", "usmc",
	"m1eo", "m1ey", "m2eo", "m2ey", "me3o", "me3y", "enma", "exma", "edma", "enms", "exms",
	"len", "clc1", "clc2", "clc3", "clc4", "rng", "ast1", "ast2", "ast3", "ast4", "ast5", "ast6",
	"pst1", "pst2", "pst3", "pst4", "pst5", "pets", "skpt",
}

// oracleUnit is the caster of the oracle contexts. Context A knows only the
// evaluated skill (base level = evaluated level); context B has every skill
// at base (i*7+3)%21 with effective level taken from the golden.
type oracleUnit struct {
	d2skill.Unit
	ctxB   bool
	self   int
	lvl    int
	eff    []int
	ulevel int
	rng    *d2rand.Seed
}

func (u oracleUnit) Level() int { return u.ulevel }

func (u oracleUnit) SkillLevel(id int) int {
	if u.ctxB {
		if id >= 0 && id < len(u.eff) {
			return u.eff[id]
		}

		return 0
	}

	if id == u.self {
		return u.lvl
	}

	return 0
}

func (u oracleUnit) BaseSkillLevel(id int) int {
	if u.ctxB {
		if id < 0 || id >= len(u.eff) {
			return 0
		}

		return (id*7 + 3) % 21
	}

	if id == u.self {
		return u.lvl
	}

	return 0
}

func (u oracleUnit) Stat(name string) int {
	switch name {
	case "passive_fire_mastery":
		return 17
	case "passive_fire_pierce":
		return 9
	case "passive_ltng_mastery":
		return 11
	case "passive_cold_mastery":
		return 5
	case "passive_pois_mastery":
		return 3
	}

	return 0
}

func (u oracleUnit) Roller() d2combat.Roller {
	if u.rng == nil {
		return nil
	}

	return u.rng
}

func loadSkillOracleGolden(t *testing.T) skillCalcGolden {
	t.Helper()

	buf, err := os.ReadFile(filepath.Join("testdata", "skill_calc_golden.json"))
	if err != nil {
		t.Fatal(err)
	}

	var g skillCalcGolden
	if err = json.Unmarshal(buf, &g); err != nil {
		t.Fatal(err)
	}

	return g
}

// TestOracleSkillCalcs compares every skills.txt calc cell against the real
// game at several levels, with and without synergy skills.
func TestOracleSkillCalcs(t *testing.T) {
	rm := loadRealRecords(t)
	g := loadSkillOracleGolden(t)
	reg := rm.SkillTable()
	rows := skillRows(t)

	bad := map[string]int{}
	total := 0

	for _, skid := range sortedKeys(g.Calcs) {
		id, _ := strconv.Atoi(skid)
		sk := reg.ByID(id)
		if sk == nil {
			t.Fatalf("skill %d missing", id)
		}

		for col, ctxs := range g.Calcs[skid] {
			src := rows[id][col]
			prog := d2calc.Compile(src, d2calc.KindSkill)

			for ctx, vals := range ctxs {
				for i, want := range vals {
					lvl := g.Levels[i]
					u := oracleUnit{ctxB: ctx == "B", self: id, lvl: lvl, eff: g.Eff, ulevel: g.ULvl}

					if strings.Contains(src, "rand(") {
						if ctx == "B" { // the shared oracle unit's generator state drifts
							continue
						}

						u.rng = d2rand.New(1) // the oracle's fresh unit seed
					}

					got := d2skill.NewEnv(sk, lvl, u, reg).Eval(prog)
					total++

					if got != want {
						bad[col]++

						if bad[col] <= 2 {
							t.Errorf("skill %d (%s) %s ctx %s lvl %d: got %d want %d  src=%q", id, sk.Name, col, ctx, lvl, got, want, src)
						}
					}
				}
			}
		}
	}

	t.Logf("%d calc evaluations, mismatches by column: %v", total, bad)
}

// TestOracleSkillFields compares every skillcalc field code of every skill.
func TestOracleSkillFields(t *testing.T) {
	rm := loadRealRecords(t)
	g := loadSkillOracleGolden(t)
	reg := rm.SkillTable()

	bad := map[string]int{}
	total := 0

	for _, id := range reg.IDs() {
		sk := reg.ByID(id)
		fields := g.Fields[strconv.Itoa(id)]

		for fid, code := range oracleFieldCodes {
			ctxs := fields[strconv.Itoa(fid)]

			for _, ctx := range []string{"A", "B"} {
				for i, lvl := range g.FLevels {
					want := 0
					if v := ctxs[ctx]; v != nil {
						want = v[i]
					}

					u := oracleUnit{ctxB: ctx == "B", self: id, lvl: lvl, eff: g.Eff, ulevel: g.ULvl}
					got := d2skill.NewEnv(sk, lvl, u, reg).Field(code)
					total++

					if got != want {
						bad[code]++

						if bad[code] <= 2 {
							t.Errorf("skill %d (%s) field %s ctx %s lvl %d: got %d want %d", id, sk.Name, code, ctx, lvl, got, want)
						}
					}
				}
			}
		}
	}

	t.Logf("%d field evaluations, mismatches by code: %v", total, bad)
}

// skillRows returns skills.txt cells by skill id and lower-case column name.
func skillRows(t *testing.T) map[int]map[string]string {
	t.Helper()

	buf, err := os.ReadFile(filepath.Join(os.Getenv("D2_TABLES"), "skills", "patch_d2", "skills.txt"))
	if err != nil {
		t.Skip(err)
	}

	d := d2txt.LoadDataDictionary(buf)
	rows := map[int]map[string]string{}

	for d.Next() {
		r := map[string]string{}
		for _, c := range oracleCalcColumns {
			r[c] = d.String(c)
		}

		rows[d.Number("Id")] = r
	}

	return rows
}

func sortedKeys(m map[string]map[string]map[string][]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	return keys
}
