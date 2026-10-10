package d2combat

import "testing"

func TestOraclePierce(t *testing.T) {
	var g struct {
		Create [][]int32 `json:"create"`
		Spend  [][]int32 `json:"spend"`
	}

	loadJSON(t, "pierce_golden.json", &g)

	bad := 0

	for _, c := range g.Create {
		ot, mt, flag, sk, itm, init, want := c[0], c[1], c[2], c[3], c[4], c[5], c[6]
		if got := RollPierceCharges(sk+itm, int(ot), int(mt), flag == 1, 0, init); got != want {
			bad++

			if bad <= 5 {
				t.Errorf("create %v: got %d", c, got)
			}
		}
	}

	for _, c := range g.Spend {
		b, ns := SpendPierce(c[0] == 1, c[1])
		if int32(b) != c[2] || ns != c[3] {
			bad++

			if bad <= 8 {
				t.Errorf("spend %v: got %d %d", c, b, ns)
			}
		}
	}

	if bad > 0 {
		t.Errorf("%d pierce cases differ", bad)
	}
}
