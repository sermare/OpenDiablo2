package d2drop

import "testing"

// TestOracleUpgrade compares TreasureTable.Upgrade with
// ITEMGEN_GetTreasureClassByLevel of the real game for every grouped class
// and levels -1..120 (plus a few huge ones). The golden stores, per class, the
// levels where the result changes.
func TestOracleUpgrade(t *testing.T) {
	var g struct {
		Up map[string][][]interface{} `json:"up"`
	}

	readGolden(t, "tcup.json", &g)

	rt := loadReal(t)
	bad := 0

	var levels []int

	for i := -1; i <= 120; i++ {
		levels = append(levels, i)
	}

	levels = append(levels, 200, 1000, 32767)

	for name, trans := range g.Up {
		tc, ok := rt.tcs.TreasureClass(name)
		if !ok {
			t.Fatalf("unknown class %q", name)
		}

		want := ""
		ti := 0

		for _, lv := range levels {
			for ti < len(trans) && int(trans[ti][0].(float64)) <= lv {
				want = trans[ti][1].(string)
				ti++
			}

			if got := rt.tcs.Upgrade(tc, lv).Name; got != want {
				bad++

				if bad <= 8 {
					t.Errorf("%s level %d: got %q, real %q", name, lv, got, want)
				}
			}
		}
	}

	t.Logf("%d grouped classes, %d mismatches", len(g.Up), bad)
}
