package d2drop

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
)

type createCase struct {
	C  string `json:"c"`
	IL int    `json:"il"`
	Q  int    `json:"q"`
	DF int    `json:"df"`
	GS uint32 `json:"gs"`
	FL int    `json:"fl"`
	V  int    `json:"v"`
	X  int    `json:"x"`
	O  struct {
		OK  int           `json:"ok"`
		Q   int           `json:"q"`
		FL  uint32        `json:"fl"`
		IL  int           `json:"il"`
		Pre [3]int        `json:"pre"`
		Suf [3]int        `json:"suf"`
		Au  int           `json:"au"`
		Rn  [2]int        `json:"rn"`
		Uid int           `json:"uid"`
		Gfx [2]int        `json:"gfx"`
		Us  [2]uint32     `json:"us"`
		Is  [2]uint32     `json:"is_"`
		W   []goldenWrite `json:"w"`
	} `json:"o"`
}

type createGolden struct {
	V     int          `json:"v"`
	Cases []createCase `json:"cases"`
}

// rolledSetOrUnique reports whether the first quality roll of a request is
// set or unique (replaying the seeds of Create).
func (c *Creator) rolledSetOrUnique(req Request) bool {
	g := req.GameSeed
	st := &itemState{c: c, req: req, base: c.Items.ByCode[req.Code], ilvl: req.ILvl}
	st.unit = stepInit(&g)
	st.item = stepInit(&g)

	if st.base == nil {
		return false
	}

	q := c.rollExistingQuality(st)

	return q == QualitySet || q == QualityUnique
}

func fmtWrites(ws []StatWrite, kinds string) string {
	var sb strings.Builder

	for _, w := range ws {
		if strings.IndexByte(kinds, w.Kind) >= 0 {
			fmt.Fprintf(&sb, "%c%d=%d/%d ", w.Kind, w.Stat, w.Value, w.Param)
		}
	}

	return sb.String()
}

// goldenWrite is [kind, stat, value, param] with the kind a one letter string.
type goldenWrite struct {
	Kind              byte
	Stat, Value, Para int64
}

func (g *goldenWrite) UnmarshalJSON(b []byte) error {
	var raw []interface{}

	if err := json.Unmarshal(b, &raw); err != nil || len(raw) != 4 {
		return fmt.Errorf("bad write %s", b)
	}

	g.Kind = raw[0].(string)[0]
	g.Stat, g.Value, g.Para = int64(raw[1].(float64)), int64(raw[2].(float64)), int64(raw[3].(float64))

	return nil
}

func goldenWrites(w []goldenWrite, kinds string) string {
	var sb strings.Builder

	for _, x := range w {
		if strings.IndexByte(kinds, x.Kind) >= 0 {
			fmt.Fprintf(&sb, "%c%d=%d/%d ", x.Kind, x.Stat, x.Value, x.Para)
		}
	}

	return sb.String()
}

// TestOracleCreateA compares slice A with ITEMGEN_CreateItemFromRequest of
// the real game (noprops mode: the property engine is not run) for
// qualities low, normal and superior over every item class, both game
// versions and the request flags.
func TestOracleCreateA(t *testing.T) {
	var g createGolden

	if p := os.Getenv("D2_CREATE_A"); p != "" {
		// A bigger local run (gen_create_a.py in the oracle directory).
		raw, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}

		if err := json.Unmarshal(raw, &g); err != nil {
			t.Fatal(err)
		}
	} else {
		readGolden(t, "create_a.json", &g)
	}

	items := loadItemTables(t)
	c := &Creator{Items: items, Quality: loadQualityTables(t), Affixes: loadAffixTables(t)}

	bad, ok, fails, skipped := 0, 0, 0, 0

	for _, tc := range g.Cases {
		req := Request{
			Code: tc.C, ILvl: tc.IL, Quality: Quality(tc.Q), Difficulty: tc.DF, Version: tc.V,
			Expansion: tc.X == 1, Flags: RequestFlags(tc.FL), GameSeed: d2rand.Seed{Lo: tc.GS, Hi: 0x29a},
		}

		if tc.V < 1 && tc.Q == 0 && c.rolledSetOrUnique(req) {
			// The classic set and unique picks (slice C) change the
			// durability of the item even when they fail.
			skipped++

			continue
		}

		got, err := c.Create(req)
		if tc.O.OK == 0 {
			if err == nil {
				bad++

				if bad <= 10 {
					t.Errorf("%+v: created, the game refuses", req)
				}
			}

			fails++

			continue
		}

		if err != nil {
			bad++

			if bad <= 10 {
				t.Errorf("%+v: %v, the game creates it", req, err)
			}

			continue
		}

		ok++

		var want []string

		chk := func(name string, a, b interface{}) {
			if !reflect.DeepEqual(a, b) {
				want = append(want, fmt.Sprintf("%s got %v want %v", name, a, b))
			}
		}

		chk("quality", int(got.Quality), tc.O.Q)
		chk("flags", fmt.Sprintf("%#x", got.Flags), fmt.Sprintf("%#x", tc.O.FL))
		chk("ilvl", got.ILvl, tc.O.IL)
		chk("prefix", got.Prefix, tc.O.Pre)
		chk("suffix", got.Suffix, tc.O.Suf)
		chk("auto", got.Auto, tc.O.Au)
		chk("uid", got.Unique, tc.O.Uid)
		chk("gfx", got.Gfx, tc.O.Gfx)
		chk("S writes", fmtWrites(got.Writes, "S"), goldenWrites(tc.O.W, "S"))
		chk("L writes", fmtWrites(got.Writes, "LM"), goldenWrites(tc.O.W, "LM"))
		chk("unit seed", [2]uint32{got.UnitSeed.Lo, got.UnitSeed.Hi}, tc.O.Us)
		chk("item seed", [2]uint32{got.ItemSeed.Lo, got.ItemSeed.Hi}, tc.O.Is)

		if len(want) > 0 {
			bad++

			if bad <= 10 {
				t.Errorf("%s ilvl %d q %d diff %d v %d x %d fl %#x seed %d:\n\t%s", tc.C, tc.IL, tc.Q, tc.DF,
					tc.V, tc.X, tc.FL, tc.GS, strings.Join(want, "\n\t"))
			}
		}
	}

	t.Logf("%d cases compared, %d refused by the game, %d skipped (classic set/unique), %d mismatches",
		ok, fails, skipped, bad)

	if bad > 0 {
		t.Fail()
	}
}
