package d2combat

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
)

// Goldens (numbers only) come from running the real Game.exe 1.14b functions
// in an x86 emulator with fake units: ResolveComponent 0x579c90 (+0x579b10,
// 0x579c20), crushing blow 0x5bdbf0, open wounds 0x5bda80, missile damage
// builder 0x5a63c0 and the physical case of 0x5a6690.

func loadJSON(t *testing.T, name string, v interface{}) {
	t.Helper()

	buf, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}

	if err = json.Unmarshal(buf, v); err != nil {
		t.Fatal(err)
	}
}

func TestOracleComponent(t *testing.T) {
	var g struct {
		Comp [][]int `json:"comp"`
	}

	loadJSON(t, "dmg_res_golden.json", &g)

	bad := 0

	for _, c := range g.Comp {
		rs, maxs, pierce, apct, aflat := c[0], c[1], c[2], c[3], c[4]
		dmg, flat, res, mx, pv, ign, ap, af := c[5], c[6], c[7], c[8], c[9], c[10], c[11], c[12]
		expn, diff, penv, want, wantHeal := c[13], c[14], c[15], c[17], c[18]

		in := ResistInput{
			Resist: res, Pierce: pv, HasPierce: pierce != -1, MaxResistBonus: mx, HasMaxResist: maxs != -1,
			IsPhysical: rs == 36, NoDifficultyPenalty: rs == 36 || rs == 37, Ignore: ign == 1,
		}

		if expn == 1 {
			in.DifficultyPenalty = penv
		} else {
			in.DifficultyPenalty = ClassicResistPenalty(diff)
		}

		eff := EffectiveResist(in)
		a := int32(0)

		if af > 0 && aflat != -1 {
			a = int32(af)
		}

		got, heal := ResolveComponent(int32(dmg), int32(flat), eff, ign == 1, apct != -1, int32(ap), a)
		if int(got) != want || int(heal) != wantHeal {
			bad++

			if bad <= 8 {
				t.Errorf("component %v: got %d heal %d want %d heal %d", c, got, heal, want, wantHeal)
			}
		}
	}

	if bad > 0 {
		t.Errorf("%d of %d component cases differ", bad, len(g.Comp))
	}
}

func TestOracleCrushingBlow(t *testing.T) {
	var g struct {
		Crush [][]int `json:"crush"`
	}

	loadJSON(t, "dmg_cb_golden.json", &g)

	bad := 0

	for _, c := range g.Crush {
		kind, evt, chance, life, phys, pc, sp, bs, fl2 := c[0], c[1], c[2], c[3], c[4], c[5], c[6], c[7], c[8]
		seed := &d2rand.Seed{Lo: uint32(c[9]), Hi: uint32(c[10])}
		wantRet, wantLife, wantRes, wantKnock := c[11], c[12], c[13], c[14]

		o := RollCrushingBlowExe(seed, CrushingInput{
			Chance: chance, DefenderKind: kind, Special: sp == 1, Boss: bs == 1, DataFlag2: fl2 == 1,
			PlayerCount: pc, Event: evt, Life: int32(life), PhysResistRaw: phys,
		})

		res := 0
		if o.Died {
			res = 2
		}

		ret := 0
		if o.Hit {
			ret = 1
		}

		knock := 0
		if o.Knock {
			knock = 1
		}

		if ret != wantRet || int(o.Life) != wantLife || res != wantRes || knock != wantKnock ||
			seed.Lo != uint32(c[15]) || seed.Hi != uint32(c[16]) {
			bad++

			if bad <= 8 {
				t.Errorf("crush %v: got ret=%d life=%d res=%d knock=%d seed=%x:%x", c, ret, o.Life, res, knock, seed.Lo, seed.Hi)
			}
		}
	}

	if bad > 0 {
		t.Errorf("%d of %d crushing blow cases differ", bad, len(g.Crush))
	}
}

func TestOracleOpenWounds(t *testing.T) {
	var g struct {
		Wounds [][]int `json:"wounds"`
	}

	loadJSON(t, "dmg_cb_golden.json", &g)

	bad := 0

	for _, c := range g.Wounds {
		kind, evt, chance, lvl, fl := c[0], c[1], c[2], c[3], c[4]
		seed := &d2rand.Seed{Lo: uint32(c[5]), Hi: uint32(c[6])}
		wantRet, wantVal, wantLen := c[7], c[8], c[9]

		o := RollOpenWoundsExe(seed, OpenWoundsInput{Chance: chance, AttackerLevel: lvl, DefenderKind: kind, DefenderFlagC: fl == 1, Event: evt})

		ret, val, ln := 0, 0, 0
		if o.Hit {
			ret, val, ln = 1, -o.Value, OpenWoundsLength
		}

		if ret != wantRet || val != wantVal || ln != wantLen || seed.Lo != uint32(c[10]) || seed.Hi != uint32(c[11]) {
			bad++

			if bad <= 8 {
				t.Errorf("wounds %v: got ret=%d val=%d len=%d seed=%x:%x", c, ret, val, ln, seed.Lo, seed.Hi)
			}
		}
	}

	if bad > 0 {
		t.Errorf("%d of %d open wounds cases differ", bad, len(g.Wounds))
	}
}

type bldCase struct {
	Stats []int
	Tgt   int
	Dm    int
	Un    int
	Lo    uint32
	Hi    uint32
	Out   []int
	Lo2   uint32
	Hi2   uint32
}

func (b *bldCase) UnmarshalJSON(buf []byte) error {
	return json.Unmarshal(buf, &[]interface{}{&b.Stats, &b.Tgt, &b.Dm, &b.Un, &b.Lo, &b.Hi, &b.Out, &b.Lo2, &b.Hi2})
}

func checkBuild(t *testing.T, name string, ids []int, cases []bldCase, physOnly bool) {
	t.Helper()

	bad := 0

	for _, c := range cases {
		seed := &d2rand.Seed{Lo: c.Lo, Hi: c.Hi}
		get := func(id int) int32 {
			for i, s := range ids {
				if s == id {
					return int32(c.Stats[i])
				}
			}

			return 0
		}

		var tgt *MissileTarget
		if c.Tgt != 0 {
			tgt = &MissileTarget{Demon: c.Dm == 1, Undead: c.Un == 1}
		}

		var d Damage
		if physOnly {
			d = BuildMissilePhysical(seed, get, tgt)
		} else {
			d = BuildMissileDamage(seed, get, tgt)
		}

		got := []int32{int32(d.Flags), int32(d.Result), d.Physical, 0, d.Fire, d.Burn, d.BurnLen, d.Lightning, d.Magic, d.Cold,
			d.Poison, d.PoisonLen, d.ColdLen, 0, d.LifeLeech, d.ManaLeech, d.StaminaLeech, d.StunLen}

		differs := seed.Lo != c.Lo2 || seed.Hi != c.Hi2

		for i := range got {
			if i == 3 || i == 13 {
				continue
			}

			if int(got[i]) != c.Out[i] {
				differs = true
			}
		}

		if differs {
			bad++

			if bad <= 5 {
				t.Errorf("%s stats=%v tgt=%d: got %v want %v", name, c.Stats, c.Tgt, got, c.Out[:18])
			}
		}
	}

	if bad > 0 {
		t.Errorf("%s: %d of %d cases differ", name, bad, len(cases))
	}
}

func TestOracleMissileBuild(t *testing.T) {
	var g struct {
		Stats []int     `json:"stats"`
		Bld   []bldCase `json:"bld"`
		Phys  []bldCase `json:"phys"`
	}

	loadJSON(t, "dmg_bld_golden.json", &g)
	checkBuild(t, "build", g.Stats, g.Bld, false)
	checkBuild(t, "phys", g.Stats, g.Phys, true)
}
