package d2drop

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
)

// The oracle goldens in testdata/ were produced by running the real Game.exe
// 1.14b code under an x86 emulator (see the d2-re-notes itemgen audit). They
// hold numbers and item codes only. The tests need D2_TABLES (to build the
// same tables the game had loaded) and skip without it.

func readGolden(t *testing.T, name string, v interface{}) {
	t.Helper()

	raw, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Skipf("golden %s missing: %v", name, err)
	}

	if err := json.Unmarshal(raw, v); err != nil {
		t.Fatalf("%s: %v", name, err)
	}
}

type qualityRecord struct {
	C  string    `json:"c"`
	IL int       `json:"il"`
	MF int       `json:"mf"`
	M  [4]int    `json:"m"` // magic, rare, set, unique (1/1024)
	S  uint32    `json:"s"`
	Q  []Quality `json:"q"`
	F  [2]uint32 `json:"f"`
}

// TestOracleQuality compares RollQuality with ITEMGEN_RollDropQuality: the
// qualities of six consecutive rolls and the generator state afterwards.
func TestOracleQuality(t *testing.T) {
	var recs []qualityRecord

	readGolden(t, "quality.json", &recs)

	rt := loadReal(t)
	bad := 0

	for _, r := range recs {
		info := rt.items[r.C]
		if info == nil {
			t.Fatalf("unknown item %q", r.C)
		}

		ratio, _ := rt.ItemRatio(info.ClassSpecific, info.Uber)
		rng := &d2rand.Seed{Lo: r.S, Hi: 0x29a}

		got := make([]Quality, 0, len(r.Q))

		for range r.Q {
			got = append(got, RollQuality(rng, ratio, QualityInput{
				ILvl: r.IL, QLvl: info.Level, MagicFind: r.MF,
				Mods:       QualityMods{Magic: r.M[0], Rare: r.M[1], Set: r.M[2], Unique: r.M[3]},
				TypeNormal: info.TypeNormal, TypeMagic: info.TypeMagic, TypeRare: info.TypeRare,
				Unique: info.Unique, Quest: info.Quest,
			}))
		}

		if fmt.Sprint(got) != fmt.Sprint(r.Q) || rng.Lo != r.F[0] || rng.Hi != r.F[1] {
			bad++

			if bad <= 10 {
				t.Errorf("%s ilvl %d mf %d mods %v seed %d: got %v end (%d,%d); real %v end %v",
					r.C, r.IL, r.MF, r.M, r.S, got, rng.Lo, rng.Hi, r.Q, r.F)
			}
		}
	}

	t.Logf("%d quality records, %d mismatches", len(recs), bad)

	if bad > 0 {
		t.Fail()
	}
}

type tcCase struct {
	TC string    `json:"tc"`
	IL int       `json:"il"`
	MF int       `json:"mf"`
	NP [3]int    `json:"np"` // party size, players in game, monster's players stat
	F4 int       `json:"f4"` // no-nodrop flag
	FQ Quality   `json:"fq"` // forced quality
	MX int       `json:"mx"` // max drops (0 = default)
	S  uint32    `json:"s"`
	D  []string  `json:"d"` // "code,quality,forcedID,flags"
	E  [2]uint32 `json:"e"`
}

type tcGolden struct {
	V     int      `json:"v"`
	Cases []tcCase `json:"cases"`
}

// fmtDrop formats a Drop like the golden: code, quality, forced row id (1 + row
// index of UniqueItems/SetItems), flags (4: ce roll hit, 0x10: cg roll hit).
func fmtDrop(d Drop, rt *realTables) string {
	fid, flags := 0, 0

	if d.ForcedID != "" {
		fid = rt.rowIndex(d.Quality, d.ForcedID) + 1
	}

	if d.EFlag {
		flags |= 4
	}

	if d.GFlag {
		flags |= 0x10
	}

	return fmt.Sprintf("%s,%d,%d,%d", d.Code, d.Quality, fid, flags)
}

func TestOracleTreasure(t *testing.T) { checkTreasureGolden(t, "treasure.json", false) }

// TestOracleTreasureLowQualityLevel rolls with the quality level 0..2 that
// chests hand the roller (Context.QualityLevel).
func TestOracleTreasureLowQualityLevel(t *testing.T) {
	checkTreasureGolden(t, "treasure_low.json", false)
}

// TestOracleTreasureClassic is the same comparison for a game without the
// expansion (game+0x70 == 0).
func TestOracleTreasureClassic(t *testing.T) { checkTreasureGolden(t, "treasure_classic.json", true) }

func checkTreasureGolden(t *testing.T, file string, classic bool) {
	t.Helper()

	var g tcGolden

	readGolden(t, file, &g)

	rt := loadReal(t)
	d := &Dropper{TCs: rt.tcs, Items: rt.items, Ratios: rt}
	bad := 0
	byTC := map[string]int{}

	for _, c := range g.Cases {
		rng := &d2rand.Seed{Lo: c.S, Hi: 0x29a}
		ctx := &Context{
			RNG: rng, ILvl: c.IL, MagicFind: c.MF, ForcedQuality: c.FQ, MaxDrops: c.MX,
			Players: NoDropPlayers(c.NP[0], c.NP[1], c.NP[2], true), NoNoDrop: c.F4 != 0, Classic: classic,
			QualityLevel: c.IL, UseQualityLevel: file == "treasure_low.json",
		}

		drops, err := d.Roll(ctx, c.TC)
		if err != nil {
			t.Fatalf("%s: %v", c.TC, err)
		}

		got := make([]string, 0, len(drops))
		for _, dr := range drops {
			got = append(got, fmtDrop(dr, rt))
		}

		if strings.Join(got, " ") != strings.Join(c.D, " ") || rng.Lo != c.E[0] || rng.Hi != c.E[1] {
			bad++
			byTC[c.TC]++

			if bad <= 8 {
				t.Errorf("%s ilvl %d mf %d np %v f4 %d fq %d mx %d seed %d:\n got  %v end (%d,%d)\n real %v end %v",
					c.TC, c.IL, c.MF, c.NP, c.F4, c.FQ, c.MX, c.S, got, rng.Lo, rng.Hi, c.D, c.E)
			}
		}
	}

	t.Logf("%d treasure cases, %d mismatches in %d classes", len(g.Cases), bad, len(byTC))

	if bad > 0 {
		t.Fail()
	}
}
