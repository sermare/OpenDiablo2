package d2combat

import "testing"

// ApplyResistFull (resist.go, from the resist branches) and ReduceComponent
// (hit.go, from the missile-damage-oracle, golden dmg_res_golden.json) are two
// copies of COMBAT_ApplyResistToDamageType (0x579c90) without absorb. They
// must agree on every input; this pins them together so a fix to one cannot
// silently leave the other behind.
func TestApplyResistFullAgreesWithReduceComponent(t *testing.T) {
	dmgs := []int{-5, 0, 1, 100, 256, 5000, 123456}
	flats := []int{0, 1, 50, 256, 9999999}
	ress := []int{-150, -100, -60, -1, 0, 1, 25, 50, 75, 99, 100, 101, 250}

	n := 0

	for _, dmg := range dmgs {
		for _, flat := range flats {
			for _, res := range ress {
				for _, ign := range []bool{false, true} {
					n++

					want, heal := ReduceComponent(dmg, flat, res, ign, false, 0, 0)
					got := ApplyResistFull(dmg, flat, res, ign)

					if got != want || heal != 0 {
						t.Errorf("dmg=%d flat=%d res=%d unresistable=%v: ApplyResistFull=%d ReduceComponent=%d heal=%d",
							dmg, flat, res, ign, got, want, heal)
					}
				}
			}
		}
	}

	if n == 0 {
		t.Fatal("no cases")
	}
}

// ApplyResist (resist percent only) is ApplyResistFull with no flat reduction.
func TestApplyResistIsFullWithoutFlat(t *testing.T) {
	for _, dmg := range []int{-3, 0, 1, 77, 5000} {
		for _, res := range []int{-120, -1, 0, 10, 100, 130} {
			if a, b := ApplyResist(dmg, res), ApplyResistFull(dmg, 0, res, false); a != b {
				t.Errorf("dmg=%d res=%d: ApplyResist=%d ApplyResistFull=%d", dmg, res, a, b)
			}
		}
	}
}
