package d2combat

import (
	"encoding/json"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
)

type melCase struct {
	Stats                                                 []int
	At, Dt, Dm, Un, Cls2, Scale, Flags0, Conv, Cpct, Skov int
	Lo, Hi                                                uint32
	Out                                                   []int
	Lo2, Hi2                                              uint32
	Phys, DStat, Sp                                       int
	Sets                                                  [][]int
}

func (m *melCase) UnmarshalJSON(buf []byte) error {
	return json.Unmarshal(buf, &[]interface{}{&m.Stats, &m.At, &m.Dt, &m.Dm, &m.Un, &m.Cls2, &m.Scale, &m.Flags0, &m.Conv, &m.Cpct, &m.Skov,
		&m.Lo, &m.Hi, &m.Out, &m.Lo2, &m.Hi2, &m.Phys, &m.DStat, &m.Sp, &m.Sets})
}

// TestOracleAttackerDamage compares BuildAttackerDamage with the real
// COMBAT_BuildAttackerDamage (0x5794e0) for unarmed attackers; the physical
// roll (0x579120) is stubbed to a fixed value in the golden.
func TestOracleAttackerDamage(t *testing.T) {
	var g struct {
		Stats []int     `json:"stats"`
		Cases []melCase `json:"cases"`
	}

	loadJSON(t, "melee_golden.json", &g)

	bad := 0

	for _, c := range g.Cases {
		get := func(id int) int32 {
			for i, s := range g.Stats {
				if s == id {
					return int32(c.Stats[i])
				}
			}

			return 0
		}

		seed := &d2rand.Seed{Lo: c.Lo, Hi: c.Hi}
		o := BuildAttackerDamage(seed, MeleeIn{
			Get: get, AttackerKind: c.At, AttackerMerc: c.Cls2 == 2, Special: c.Sp == 1, DefenderKind: c.Dt, DefenderDemon: c.Dm == 1,
			DefenderUndead: c.Un == 1, DefenderAC: int32(c.DStat), Scale: uint8(c.Scale), SkillOverride: c.Skov == 1, Phys: int32(c.Phys),
			Damage: Damage{Flags: uint32(c.Flags0)}, ConvClass: byte(c.Conv), ConvPct: int32(c.Cpct),
		})

		d := o.Damage
		got := []int32{int32(d.Flags), int32(d.Result), d.Physical, d.DamagePct, d.Fire, d.Burn, d.BurnLen, d.Lightning, d.Magic, d.Cold,
			d.Poison, d.PoisonLen, d.ColdLen, d.FreezeLen, d.LifeLeech, d.ManaLeech, d.StaminaLeech, d.StunLen}

		differs := seed.Lo != c.Lo2 || seed.Hi != c.Hi2

		for i, v := range got {
			if int(v) != c.Out[i] {
				differs = true
			}
		}

		// defender stat write
		switch {
		case len(c.Sets) == 0 && o.DefenderACSet:
			differs = true
		case len(c.Sets) > 0 && (!o.DefenderACSet || int(o.DefenderAC) != c.Sets[0][1]):
			differs = true
		}

		if differs {
			bad++

			if bad <= 5 {
				t.Errorf("melee at=%d dt=%d scale=%d flags=%#x conv=%d/%d sp=%d: got %v seed=%x:%x\n want %v seed=%x:%x sets=%v", c.At, c.Dt, c.Scale,
					c.Flags0, c.Conv, c.Cpct, c.Sp, got, seed.Lo, seed.Hi, c.Out[:18], c.Lo2, c.Hi2, c.Sets)
			}
		}
	}

	if bad > 0 {
		t.Errorf("%d of %d attacker damage cases differ", bad, len(g.Cases))
	}
}
