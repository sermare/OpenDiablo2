package d2combat

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
)

func TestOracleKnockback(t *testing.T) {
	var g struct {
		Cases [][]int64 `json:"cases"`
	}

	loadJSON(t, "knock_golden.json", &g)

	bad := 0

	for _, c := range g.Cases {
		stat, dk, rec, b5, res0 := c[0], c[1], c[2], c[3], c[4]
		seed := &d2rand.Seed{Lo: uint32(c[5]), Hi: uint32(c[6])}
		wantRet, wantRes := c[7], c[8]

		hit := RollKnockback(seed.Step, stat > 0, int(dk), rec == 1, byte(b5))
		ret, res := int64(0), res0

		if hit {
			ret, res = 1, res0|8
		}

		if ret != wantRet || res != wantRes || int64(seed.Lo) != c[9] || int64(seed.Hi) != c[10] {
			bad++

			if bad <= 5 {
				t.Errorf("knockback %v: got ret=%d res=%d seed=%x:%x", c, ret, res, seed.Lo, seed.Hi)
			}
		}
	}

	if bad > 0 {
		t.Errorf("%d knockback cases differ", bad)
	}
}
