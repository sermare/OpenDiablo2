package d2missile

import (
	"encoding/json"
	"os"
	"testing"
)

func TestOracleAccept(t *testing.T) {
	buf, err := os.ReadFile("testdata/collide_golden.json")
	if err != nil {
		t.Fatal(err)
	}

	var g struct {
		Common [][]int `json:"common"`
		Typed  [][]int `json:"typed"`
	}

	if err = json.Unmarshal(buf, &g); err != nil {
		t.Fatal(err)
	}

	bad := 0

	for _, c := range g.Common {
		in := AcceptIn{TargetFlags: uint32(c[0]), NextHit: c[1] == 1, CollideFriend: c[2] == 1, TargetState56: c[3] == 1, LastHit: c[4] == 1,
			OwnerPresent: c[5] == 1, OwnerEnemy: c[6] == 1}
		if got := AcceptCommon(in); got != (c[7] == 1) {
			bad++

			if bad <= 5 {
				t.Errorf("common %v: got %v", c, got)
			}
		}
	}

	for _, c := range g.Typed {
		in := AcceptIn{TargetKind: c[1], TargetFlags: uint32(c[2]), NextHit: c[3] == 1, CollideFriend: c[4] == 1, TargetState56: c[5] == 1, LastHit: c[6] == 1,
			OwnerPresent: c[7] == 1, OwnerEnemy: c[8] == 1, Aligned: c[9] == 2}
		if got := AcceptTyped(c[0], in); got != (c[10] == 1) {
			bad++

			if bad <= 8 {
				t.Errorf("typed %v: got %v", c, got)
			}
		}
	}

	if bad > 0 {
		t.Errorf("%d target acceptance cases differ", bad)
	}
}
