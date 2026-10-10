package d2combat

import (
	"encoding/json"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
)

// TestOracleMonsterDouble compares Damage.RollMonsterDouble with the real 0x5a2f90.
func TestOracleMonsterDouble(t *testing.T) {
	var g struct {
		Cases [][]json.RawMessage `json:"cases"`
	}

	loadJSON(t, "mdouble_golden.json", &g)

	bad := 0

	for _, c := range g.Cases {
		var kind, chance int

		var lo, hi, lo2, hi2 uint32

		var pre, post []int32

		for i, v := range []interface{}{&kind, &chance, &lo, &hi, &pre, &post, &lo2, &hi2} {
			if err := json.Unmarshal(c[i], v); err != nil {
				t.Fatal(err)
			}
		}

		// pre order: +8 physical, +0x10 fire, +0x1c lightning, +0x20 magic, +0x24 cold, +0x28 poison, then untouched words
		d := Damage{Physical: pre[0], Fire: pre[1], Lightning: pre[2], Magic: pre[3], Cold: pre[4], Poison: pre[5]}
		seed := &d2rand.Seed{Lo: lo, Hi: hi}

		// the exe acts only for a monster attacker (unit type 1)
		if kind == 1 {
			d.RollMonsterDouble(seed, uint8(chance))
		}

		got := []int32{d.Physical, d.Fire, d.Lightning, d.Magic, d.Cold, d.Poison}

		differs := seed.Lo != lo2 || seed.Hi != hi2

		for i, v := range got {
			if v != post[i] {
				differs = true
			}
		}

		if differs {
			bad++

			if bad <= 5 {
				t.Errorf("kind=%d chance=%d pre=%v: got %v seed %x:%x want %v seed %x:%x", kind, chance, pre[:6], got, seed.Lo, seed.Hi, post[:6], lo2, hi2)
			}
		}
	}

	if bad > 0 {
		t.Errorf("%d monster double cases differ", bad)
	}
}
