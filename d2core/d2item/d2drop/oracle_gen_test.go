package d2drop

import (
	"fmt"
	"strings"
	"testing"
)

type genClass struct {
	LV int             `json:"lv"`
	TC int             `json:"tc"` // total weight, classic game
	TX int             `json:"tx"` // total weight, expansion game
	E  [][]interface{} `json:"e"`  // code, expansion weight, flags (0x10: expansion only), classic weight
}

// TestOracleGeneratedClasses compares the item-type classes ("armo3" ...)
// with the ones the game builds at load time.
func TestOracleGeneratedClasses(t *testing.T) {
	var g map[string]genClass

	readGolden(t, "tcgen.json", &g)

	rt := loadReal(t)
	bad := 0

	for name, want := range g {
		tc, ok := rt.tcs.TreasureClass(name)
		if !ok {
			t.Errorf("%s: missing", name)

			bad++

			continue
		}

		var wantE, gotE []string

		for _, e := range want.E {
			wantE = append(wantE, fmt.Sprintf("%v:%v", e[0], e[1]))
		}

		for _, e := range tc.Entries {
			gotE = append(gotE, fmt.Sprintf("%s:%d", e.Code, e.Prob))
		}

		if strings.Join(wantE, " ") != strings.Join(gotE, " ") || tc.Level != want.LV || tc.TotalProb() != want.TX {
			bad++

			if bad <= 6 {
				t.Errorf("%s: level %d total %d entries %v; real level %d total %d entries %v",
					name, tc.Level, tc.TotalProb(), gotE, want.LV, want.TX, wantE)
			}
		}
	}

	t.Logf("%d generated classes, %d mismatches", len(g), bad)

	if bad > 0 {
		t.Fail()
	}
}
