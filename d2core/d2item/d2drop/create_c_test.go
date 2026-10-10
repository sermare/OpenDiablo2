package d2drop

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
)

// Golden cases of slice C (testdata/create_c.json), produced by running the
// real game's ITEMMODS_ApplyPropertyGroup and the unique / set pick on an
// emulated item. See gen_create_c.py in the oracle.

type cWrite [5]interface{} // kind, stat, value, param, list selector

type cCase struct {
	T   string    `json:"t"`
	K   int       `json:"k"`  // property group kind
	C   string    `json:"c"`  // base item code
	IL  int       `json:"il"` // item level
	V   int       `json:"v"`  // item version
	Df  int       `json:"df"`
	Q   int       `json:"q"`
	Fl  uint32    `json:"fl"` // item flags before
	R   int       `json:"r"`  // record: unique / set item row, affix id, QualityItems row
	SB  [2]uint32 `json:"sb"`
	SA  [2]uint32 `json:"sa"`
	Pre [][2]int  `json:"pre"`
	W   []cWrite  `json:"w"`
	// fn cases: one property function.
	F   int    `json:"f"`
	I   [3]int `json:"i"` // param, min, max
	Set int    `json:"set"`
	St  int    `json:"st"`
	Val int    `json:"val"`
	Pv  int    `json:"pv"`
	Sel int    `json:"sel"`
	// pick cases.
	Fid int   `json:"fid"`
	Rf  int   `json:"rf"`
	Lad int   `json:"lad"`
	M0  []int `json:"m0"`
	M1  []int `json:"m1"`
	Iv  int   `json:"iv"`
	Uid int   `json:"uid"`
	P   int   `json:"p"` // cfn cases: the property row
	// Ov lists [row, xor] changes of the flags byte of unique rows (1 enabled,
	// 2 nolimit, 8 ladder-only) the golden ran with.
	Ov [][2]int `json:"ov"`
	// full cases: the request (Fl is the request flags) and the game's result.
	X  int    `json:"x"`
	GS uint32 `json:"gs"`
	O  *struct {
		OK  int       `json:"ok"`
		Q   int       `json:"q"`
		FL  uint32    `json:"fl"`
		IL  int       `json:"il"`
		Pre [3]int    `json:"pre"`
		Suf [3]int    `json:"suf"`
		Au  int       `json:"au"`
		Rn  [2]int    `json:"rn"`
		Uid int       `json:"uid"`
		Gfx [2]int    `json:"gfx"`
		Us  [2]uint32 `json:"us"`
		Is  [2]uint32 `json:"is_"`
		W   []cWrite  `json:"w"`
		Ws  []int     `json:"ws"`
		M1  []int     `json:"m1"`
	} `json:"o"`
	// results of fn and pick cases.
	Ret *int   `json:"ret"`
	Fl2 uint32 `json:"fl2"`
}

type cFile struct {
	V     int     `json:"v"`
	Cases []cCase `json:"cases"`
}

func cNum(x interface{}) int { return int(x.(float64)) }

// cLoadCreator builds a creator with every table: items, quality (slice A),
// affixes (slice B), properties and unique / set items (slice C).
func cLoadCreator(t *testing.T) *Creator {
	t.Helper()

	pt := loadPropTables(t)
	pt.Affix = loadTestAffixes(t, pt)
	pt.Quality = loadTestQuality(t, pt)

	return &Creator{
		Items: loadItemTables(t), Quality: loadQualityTables(t), Affixes: loadAffixTables(t),
		Props: pt, Uniques: loadUniqueTables(t, pt),
	}
}

func readCFile(t *testing.T) *cFile {
	t.Helper()

	name := "create_c.json"
	if p := os.Getenv("D2_CREATE_C"); p != "" {
		raw, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}

		var f cFile
		if err := json.Unmarshal(raw, &f); err != nil {
			t.Fatal(err)
		}

		return &f
	}

	var f cFile

	readGolden(t, name, &f)

	return &f
}

// newPropState builds the item state the property group starts from.
func newPropState(c *Creator, cs *cCase) *itemState {
	st := &itemState{
		c: c, base: c.Items.ByCode[cs.C], ilvl: cs.IL, flags: cs.Fl, quality: Quality(cs.Q),
		req:  Request{Code: cs.C, Version: cs.V, Difficulty: cs.Df},
		item: d2rand.Seed{Lo: cs.SB[0], Hi: cs.SB[1]}, uniqueRow: -1,
	}

	for _, p := range cs.Pre {
		st.write('S', p[0], p[1], 0)
	}

	return st
}

func cFmtWrites(ws []StatWrite) string {
	s := ""
	for _, w := range ws {
		s += fmt.Sprintf(" %c%d=%d/%d@%d", w.Kind, w.Stat, w.Value, uint32(w.Param), w.List)
	}

	return s
}

func (cs *cCase) want() []StatWrite {
	var out []StatWrite

	for _, w := range cs.W {
		out = append(out, StatWrite{Kind: w[0].(string)[0], Stat: cNum(w[1]), Value: cNum(w[2]), Param: cNum(w[3]), List: cNum(w[4])})
	}

	return out
}

// TestPropertyGroups replays the property groups of unique items, set items,
// affixes and QualityItems rows against the game.
func TestPropertyGroups(t *testing.T) {
	f := readCFile(t)
	c := cLoadCreator(t)
	bad, n := 0, 0

	for i := range f.Cases {
		cs := &f.Cases[i]
		if cs.T != "grp" {
			continue
		}

		n++

		st := newPropState(c, cs)
		pre := len(st.writes)

		var props []PropInst

		switch cs.K {
		case PropKindUnique, PropKindSetItem:
			st.uniqueRow = cs.R
		case PropKindAffix:
			props = c.Props.Affix[cs.R-1][:]
		case PropKindQuality:
			props = c.Props.Quality[cs.R][:]
		}

		c.applyProps(st, cs.K, props)

		got, want := st.writes[pre:], cs.want()
		wantSeed := d2rand.Seed{Lo: cs.SA[0], Hi: cs.SA[1]}

		if cFmtWrites(got) != cFmtWrites(want) || st.item != wantSeed {
			bad++
			if bad <= 12 {
				t.Errorf("case %d kind %d %s row %d ilvl %d:\n got %s seed %v\nwant %s seed %v", i, cs.K, cs.C, cs.R, cs.IL, cFmtWrites(got), st.item, cFmtWrites(want), wantSeed)
			}
		}
	}

	if n == 0 {
		t.Skip("no group cases")
	}

	t.Logf("%d property groups, %d mismatches", n, bad)

	if bad > 0 {
		t.Fatalf("%d of %d property groups differ from the game", bad, n)
	}
}
