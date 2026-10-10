package d2combat

import (
	"encoding/json"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
)

// TestOracleWeaponRoll compares RollPhysical with the real 0x579120.
func TestOracleWeaponRoll(t *testing.T) {
	var g struct {
		Stats []int             `json:"stats"`
		Cases []json.RawMessage `json:"cases"`
	}

	loadJSON(t, "wep_golden.json", &g)

	bad := 0

	for _, raw := range g.Cases {
		var st []int

		var usew, a1, active, mode, strb, dexb, mast, a2, a3, a4, a5, a6 int

		var lo, hi, rv, lo2, hi2 int64

		if err := json.Unmarshal(raw, &[]interface{}{&st, &usew, &a1, &active, &mode, &strb, &dexb, &mast, &a2, &a3, &a4, &a5, &a6, &lo, &hi, &rv, &lo2, &hi2}); err != nil {
			t.Fatal(err)
		}

		get := func(id int) int32 {
			for i, s := range g.Stats {
				if s == id {
					return int32(st[i])
				}
			}

			return 0
		}

		seed := &d2rand.Seed{Lo: uint32(lo), Hi: uint32(hi)}
		got := RollPhysical(seed, WeaponRollIn{
			Get: get, UseWeapon: usew == 1, Weapon: a1 != 0, ActiveWeapon: active != 0, HandMode: mode,
			StrBonus: int16(strb), DexBonus: int16(dexb), Mastery: int32(mast), Min: int32(a2), Max: int32(a3), PctAdd: int32(a4),
			Flat: int32(a5), Scale: uint8(a6),
		})

		if int64(got) != rv || int64(seed.Lo) != lo2 || int64(seed.Hi) != hi2 {
			bad++

			if bad <= 5 {
				t.Errorf("roll %v usew=%d a1=%d act=%d mode=%d: got %d seed %x:%x want %d seed %x:%x", st, usew, a1, active, mode, got, seed.Lo, seed.Hi, rv, lo2, hi2)
			}
		}
	}

	if bad > 0 {
		t.Errorf("%d of %d weapon roll cases differ", bad, len(g.Cases))
	}
}
