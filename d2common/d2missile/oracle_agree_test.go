package d2missile_test

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2combat"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2missile"
)

// The engine rolls missile damage through DamageDesc.Roll; the emulator-verified
// builder of the missile-damage-oracle is d2combat.BuildMissileDamage (golden
// dmg_bld_golden.json). Both implement 0x5a63c0 / 0x5a6690 and must agree on the
// same inputs, including the number of random steps they consume (a drift would
// shift every later roll of the owner's generator).

// stat ids read by BuildMissileDamage (names are unexported in d2combat).
const (
	sMinDmg, sMaxDmg, sDmgPct         = 0x15, 0x16, 0x19
	sFireMin, sFireMax                = 0x30, 0x31
	sLightMin, sLightMax              = 0x32, 0x33
	sMagicMin, sMagicMax              = 0x34, 0x35
	sColdMin, sColdMax, sColdLen      = 0x36, 0x37, 0x38
	sPoisonMin, sPoisonMax, sPoisonLn = 0x39, 0x3a, 0x3b
	sStunLen                          = 0x42
	sDeadly                           = 0x8d
	sBurnLen, sBurnMin, sBurnMax      = 0x13b, 0x13c, 0x13d
)

type seqRoller struct {
	state uint32
	calls int
}

func (r *seqRoller) Roll(n int32) uint32 {
	r.calls++
	r.state = r.state*1664525 + 1013904223

	if n <= 0 {
		return 0
	}

	return (r.state >> 8) % uint32(n)
}

func descStats(d d2missile.DamageDesc) map[int]int32 {
	m := map[int]int32{
		sMinDmg: d.PhysMin, sMaxDmg: d.PhysMax, sDmgPct: d.DamagePct,
		sFireMin: d.Fire.Min, sFireMax: d.Fire.Max,
		sLightMin: d.Lightning.Min, sLightMax: d.Lightning.Max,
		sMagicMin: d.Magic.Min, sMagicMax: d.Magic.Max,
		sColdMin: d.Cold.Min, sColdMax: d.Cold.Max, sColdLen: d.Cold.Len,
		sPoisonMin: d.Poison.Min, sPoisonMax: d.Poison.Max, sPoisonLn: d.Poison.Len,
		sBurnMin: d.Burn.Min, sBurnMax: d.Burn.Max, sBurnLen: d.Burn.Len,
		sStunLen: d.StunLen,
	}

	if d.Crit {
		m[sDeadly] = 1
	}

	return m
}

func TestDamageDescRollAgreesWithOracleBuilder(t *testing.T) {
	phys := []struct{ min, max int32 }{{0, 0}, {256, 256}, {256, 2560}, {3000, 1000}, {-5, 900}, {1, 1}}
	pcts := []int32{0, 25, 100, 250, -50, -90}
	elem := d2missile.Elem{Min: 512, Max: 2048, Len: 75}

	n := 0

	for _, p := range phys {
		for _, pct := range pcts {
			for _, crit := range []bool{false, true} {
				for _, withElem := range []bool{false, true} {
					d := d2missile.DamageDesc{PhysMin: p.min, PhysMax: p.max, DamagePct: pct, Crit: crit}
					if withElem {
						d.Fire, d.Lightning, d.Magic = elem, elem, elem
						d.Cold, d.Poison, d.Burn = elem, elem, elem
						d.StunLen = 30
					}

					for seed := uint32(1); seed <= 3; seed++ {
						n++

						engR := &seqRoller{state: seed * 7919}
						orcR := &seqRoller{state: seed * 7919}

						got := d.Roll(engR)
						st := descStats(d)
						want := d2combat.BuildMissileDamage(orcR, func(id int) int32 { return st[id] }, &d2combat.MissileTarget{})

						check := func(name string, g, w int32) {
							if g != w {
								t.Errorf("%s: phys=%+v pct=%d crit=%v elem=%v seed=%d: engine=%d oracle=%d",
									name, p, pct, crit, withElem, seed, g, w)
							}
						}

						check("physical", got.Physical, want.Physical)
						check("fire", got.Fire, want.Fire)
						check("lightning", got.Lightning, want.Lightning)
						check("magic", got.Magic, want.Magic)
						check("cold", got.Cold, want.Cold)
						check("coldLen", got.ColdLen, want.ColdLen)
						check("poison", got.Poison, want.Poison)
						check("poisonLen", got.PoisonLen, want.PoisonLen)
						check("burn", got.Burn, want.Burn)
						check("burnLen", got.BurnLen, want.BurnLen)
						check("stunLen", got.StunLen, want.StunLen)

						if engR.calls != orcR.calls {
							t.Errorf("random steps differ: phys=%+v pct=%d crit=%v elem=%v seed=%d: engine=%d oracle=%d",
								p, pct, crit, withElem, seed, engR.calls, orcR.calls)
						}

						// The critical result bit only has to agree when physical damage
						// is positive: the oracle sets it on any deadly strike, Roll only
						// when it doubled something (see the report on pass2-s2).
						if got.Physical > 0 {
							ge := got.Result&d2combat.ResultCritical != 0
							we := want.Result&d2combat.ResultCritical != 0

							if ge != we {
								t.Errorf("critical bit: phys=%+v pct=%d crit=%v seed=%d: engine=%v oracle=%v", p, pct, crit, seed, ge, we)
							}
						}
					}
				}
			}
		}
	}

	if n == 0 {
		t.Fatal("no cases ran")
	}
}
