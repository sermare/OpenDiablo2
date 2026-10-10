package d2skill

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2calc"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2path"
)

// Rows with the real patch_d2 values of the skills under test.
func init() {
	rows = append(rows,
		row{"skill": "Conversion", "Id": "400", "charclass": "pal", "srvstfunc": "32", "srvdofunc": "79",
			"auratargetstate": "conversion", "auralencalc": "ln12", "range": "h2h", "calc1": "dm34", "Param1": "400",
			"Param2": "0", "Param3": "0", "Param4": "50", "HitShift": "8", "SrcDam": "128", "manashift": "8", "mana": "4"},
		row{"skill": "Hunger", "Id": "401", "charclass": "dru", "srvdofunc": "122", "range": "h2h", "calc1": "par5",
			"calc2": "dm12", "calc3": "dm34", "Param1": "50", "Param2": "200", "Param3": "50", "Param4": "200",
			"Param5": "-75", "ToHit": "50", "LevToHit": "10", "HitShift": "8", "SrcDam": "128", "manashift": "8"},
		row{"skill": "Hydra", "Id": "402", "charclass": "sor", "srvstfunc": "14", "srvdofunc": "144", "summon": "hydra1",
			"pettype": "hydra", "petmax": "99", "summode": "S2", "sumskill1": "HydraMissile", "Param1": "250",
			"EType": "fire", "EMin": "28", "EMax": "39", "HitShift": "7", "manashift": "7", "mana": "40"},
		row{"skill": "Plague Poppy", "Id": "403", "charclass": "dru", "srvdofunc": "115", "aurastate": "vine_beast",
			"summon": "plaguepoppy", "pettype": "vine", "petmax": "1", "summode": "S1", "sumskill1": "Vine Attack",
			"calc1": "par3 * (lvl-1)", "calc2": "ulvl * 3 / 4 + lvl", "Param3": "25", "EType": "pois", "EMin": "12",
			"EMax": "16", "ELen": "100", "manashift": "8", "mana": "8"},
		row{"skill": "Telekinesis", "Id": "404", "charclass": "sor", "srvstfunc": "12", "srvdofunc": "21",
			"aurarangecalc": "par1", "Param1": "25", "EType": "", "manashift": "8", "mana": "7"},
		row{"skill": "Raven2", "Id": "405", "srvdofunc": "114", "summon": "druidhawk", "pettype": "raven",
			"petmax": "min(lvl,par2)", "Param1": "-2", "Param2": "5", "summode": "S1", "calc2": "ulvl + par1 + lvl",
			"manashift": "8"},
		row{"skill": "Teleport Gate", "Id": "406", "srvdofunc": "27", "manashift": "8"},
	)
}

type fixedRoll struct{ v uint32 }

func (f fixedRoll) Roll(n int32) uint32 {
	if f.v >= uint32(n) {
		return uint32(n) - 1
	}

	return f.v
}

func mustCalc(src string) *d2calc.Program { return d2calc.Compile(src, d2calc.KindSkill) }

func TestConversionRollAndEffect(t *testing.T) {
	cf := newClassFixture(map[string]int{"Conversion": 5})
	foe := cf.addFoe("m1", 10, 10)
	cf.u.level = 30
	cf.u.roller = fixedRoll{v: 10}

	// chance 0: the roll fails and the blow resolves as a plain melee hit
	sk := cf.reg.ByName("Conversion")
	sk.Calc[1] = mustCalc("0")

	_, r := cf.castOn("Conversion", foe)
	if !r.OK || r.Melee == nil {
		t.Fatalf("failed roll must resolve as a plain melee blow: %+v", r)
	}

	if len(r.Effects) != 0 {
		t.Errorf("no conversion expected: %+v", r.Effects)
	}

	// chance 100: converted for auralencalc frames
	sk.Calc[1] = mustCalc("100")
	cf.u.roller = fixedRoll{v: 99}
	_, r = cf.castOn("Conversion", foe)

	e := effectOf(t, r, "convert")
	if e.Target != foe || e.State != "conversion" || e.CasterLevel != 30 || e.Frames < 1 {
		t.Errorf("convert effect %+v", e)
	}

	// the roll is "chance > random": 40 percent with a roll of 40 fails, 39 converts
	sk.Calc[1] = mustCalc("40")
	cf.u.roller = fixedRoll{v: 40}

	if _, r = cf.castOn("Conversion", foe); len(r.Effects) != 0 {
		t.Errorf("roll 40 of chance 40 converted")
	}

	cf.u.roller = fixedRoll{v: 39}

	if _, r = cf.castOn("Conversion", foe); len(r.Effects) == 0 {
		t.Errorf("roll 39 of chance 40 did not convert")
	}
}

func TestConversionRefusesNonConvertible(t *testing.T) {
	cf := newClassFixture(map[string]int{"Conversion": 5})
	cf.reg.ByName("Conversion").Calc[1] = mustCalc("100")
	cf.u.roller = fixedRoll{v: 0}
	foe := &noConvert{testTarget: cf.addFoe("boss", 10, 10)}

	id := cf.id("Conversion")
	tg := Target{X: 10, Y: 10, Unit: foe, UX: 10, UY: 10}
	cf.p.Start(cf.u, id, tg)

	r := cf.p.Do(cf.u, id, tg)
	for _, e := range r.Effects {
		if e.Kind == "convert" {
			t.Fatalf("a non convertible target was converted: %+v", e)
		}
	}
}

type noConvert struct{ *testTarget }

func (noConvert) CanConvert() bool { return false }

func TestHungerLeech(t *testing.T) {
	cf := newClassFixture(map[string]int{"Hunger": 3})
	foe := cf.addFoe("m1", 10, 10)
	cf.u.ar = 100000
	cf.u.wmin, cf.u.wmax = 10, 10
	cf.u.roller = fixedRoll{v: 0}

	sk := cf.reg.ByName("Hunger")
	sk.Calc[2], sk.Calc[3] = mustCalc("12"), mustCalc("34")

	_, r := cf.castOn("Hunger", foe)
	if !r.OK || r.Melee == nil || !r.Melee.Hit {
		t.Fatalf("hunger: %+v", r)
	}

	// calc1 = par5 = -75 percent of the weapon damage (the test table does not load SrcDam)
	if want := int32(10<<8) * 25 / 100; r.Melee.Damage.Physical != want {
		t.Errorf("physical %d, want %d", r.Melee.Damage.Physical, want)
	}

	if r.Melee.Damage.LifeLeech != 12 || r.Melee.Damage.ManaLeech != 34 {
		t.Errorf("leech %d/%d, want 12/34", r.Melee.Damage.LifeLeech, r.Melee.Damage.ManaLeech)
	}
}

func TestHydraOrder(t *testing.T) {
	cf := newClassFixture(map[string]int{"Hydra": 5})
	_, r := cf.cast("Hydra", 20, 10)
	e := effectOf(t, r, "summon").Summon

	if e.Key != "hydra1" || e.Count != 3 || e.Kind != "trap" || e.TrapSkill != "HydraMissile" || e.Frames != 250 ||
		e.X != 20 || e.Y != 10 || e.Desc == nil {
		t.Errorf("hydra order %+v", e)
	}

	want := [][2]int{{-1, -1}, {0, 0}, {1, -1}} // 0x6e46bc / 0x6e46b0 (VERIFIED)
	if len(e.Cells) != 3 || e.Cells[0] != want[0] || e.Cells[1] != want[1] || e.Cells[2] != want[2] {
		t.Errorf("hydra cells %v", e.Cells)
	}

	// refused in town
	cf.u.town = true
	if st, _ := cf.cast("Hydra", 20, 10); st.OK {
		t.Errorf("hydra cast in town: %+v", st)
	}
}

func TestPlaguePoppyOrder(t *testing.T) {
	cf := newClassFixture(map[string]int{"Plague Poppy": 5})
	cf.u.level = 40
	_, r := cf.cast("Plague Poppy", 12, 12)
	e := effectOf(t, r, "summon").Summon

	// calc1 = par3*(lvl-1) = 100 percent, calc2 = ulvl*3/4 + lvl = 35
	if e.Key != "plaguepoppy" || e.HPPct != 100 || e.Level != 35 || e.Max != 1 || e.Kind != "trap" || e.TrapSkill != "Vine Attack" {
		t.Errorf("poppy order %+v", e)
	}
}

func TestSummonLevelFromCalc2(t *testing.T) {
	cf := newClassFixture(map[string]int{"Raven2": 4, "Raven": 4})
	cf.u.level = 30

	// calc2 = ulvl + par1 + lvl = 30 - 2 + 4
	_, r := cf.cast("Raven2", 5, 5)
	if e := effectOf(t, r, "summon").Summon; e.Level != 32 {
		t.Errorf("raven level %d, want 32", e.Level)
	}

	// a skill without calc2 keeps the owner's level (0 = unset)
	_, r = cf.cast("Raven", 5, 5)
	if e := effectOf(t, r, "summon").Summon; e.Level != 0 {
		t.Errorf("level %d without calc2", e.Level)
	}
}

func TestTelekinesisDamagesTarget(t *testing.T) {
	cf := newClassFixture(map[string]int{"Telekinesis": 3})
	foe := cf.addFoe("m1", 9, 9)

	_, r := cf.castOn("Telekinesis", foe)
	e := effectOf(t, r, "area_hit")

	if e.X != 9 || e.Y != 9 || e.Desc == nil {
		t.Errorf("telekinesis %+v", e)
	}

	// no target: refused
	if _, r = cf.cast("Telekinesis", 9, 9); r.OK && len(r.Effects) > 0 {
		t.Errorf("telekinesis without a target: %+v", r)
	}
}

func TestTeleportLevelFlag(t *testing.T) {
	cf := newClassFixture(map[string]int{"Teleport Gate": 1})
	cf.u.x, cf.u.y = 0, 0

	flag := 0
	cf.p.TeleportFlag = func() int { return flag }

	// 0: not allowed on this level
	if _, r := cf.cast("Teleport Gate", 10, 0); r.OK || r.Reason != ReasonLOS {
		t.Errorf("flag 0: %+v", r)
	}

	// 1: allowed even through a wall in the way
	flag = 1
	cf.w.grid.Set(5, 0, d2path.FlagWall)

	if _, r := cf.cast("Teleport Gate", 10, 0); !r.OK {
		t.Errorf("flag 1: %+v", r)
	}

	// 2: not through walls
	flag = 2
	if _, r := cf.cast("Teleport Gate", 10, 0); r.OK {
		t.Errorf("flag 2 through a wall: %+v", r)
	}

	if _, r := cf.cast("Teleport Gate", 0, 8); !r.OK {
		t.Errorf("flag 2 with a clear line: %+v", r)
	}
}

func TestCorpseExplosionLevelScale(t *testing.T) {
	cf := newClassFixture(map[string]int{"Corpse Explosion": 5})
	cf.u.level = 10
	cf.u.roller = &seq{vals: []uint32{0}}
	id := cf.id("Corpse Explosion")
	tg := Target{X: 9, Y: 9, Corpse: true, CX: 9, CY: 9, CorpseID: "c", CorpseHP: 100, CorpseLevel: 20}
	cf.p.Start(cf.u, id, tg)

	r := cf.p.Do(cf.u, id, tg)
	e := effectOf(t, r, "area_hit")

	// 70 percent of 100 life = 70, scaled by caster 10 / corpse 20 = 35
	if e.Desc.Fire.Min != 35<<8 {
		t.Errorf("fire %d, want %d", e.Desc.Fire.Min, 35<<8)
	}
}
