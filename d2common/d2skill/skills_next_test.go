package d2skill

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2state"
)

// Tests of skills_next.go. Skill rows carry the numbers of the 1.14 patch_d2
// skills.txt.

type bossTarget struct {
	*testTarget
	class KnockClass
}

func (b bossTarget) KnockClass() KnockClass { return b.class }

type shieldUnit struct {
	*testUnit
	lo, hi int
	has    bool
}

func (s shieldUnit) ShieldDamage() (int, int, bool) { return s.lo, s.hi, s.has }

type tallyRoller struct{ n int }

func (c *tallyRoller) Roll(int32) uint32 { c.n++; return 0 }

func nextFixture(levels map[string]int) *classFixture {
	cf := newClassFixture(levels)
	cf.u.ar = 100000

	for _, r := range []row{
		{"skill": "Psychic Hammer", "Id": "400", "srvdofunc": "33", "calc1": "par1", "calc2": "dm34", "calc3": "dm56",
			"calc4": "dm56", "Param1": "100", "Param3": "50", "Param4": "100", "Param5": "25", "Param6": "99",
			"EType": "mag", "EMin": "5", "EMax": "10", "manashift": "8"},
		{"skill": "Mind Blast", "Id": "401", "srvdofunc": "51", "aurafilter": "33667", "aurarangecalc": "par7",
			"Param3": "150", "Param4": "100", "Param5": "15", "Param6": "40", "Param7": "6", "EType": "stun", "manashift": "8"},
		{"skill": "Cloak of Shadows", "Id": "402", "srvdofunc": "47", "aurafilter": "57347", "aurastate": "cloak_of_shadows",
			"auratargetstate": "cloaked", "auralencalc": "ln34", "aurarangecalc": "dm12", "aurastat1": "skill_armor_percent",
			"aurastatcalc1": "-min(ln56,95)", "passivestat2": "item_absorbfire_percent", "passivecalc2": "ln12",
			"Param1": "30", "Param2": "30", "Param5": "15", "Param6": "3", "Param3": "200", "Param4": "25", "manashift": "8"},
		{"skill": "Smite", "Id": "403", "srvdofunc": "150", "calc1": "ln34", "calc2": "min(250,ln12)", "Param1": "15",
			"Param2": "5", "Param3": "15", "Param4": "15", "manashift": "8"},
		{"skill": "Charge", "Id": "404", "srvdofunc": "67", "calc1": "ln34", "Param1": "150", "Param3": "100", "Param4": "25",
			"HitShift": "8", "SrcDam": "128", "manashift": "8"},
		{"skill": "Dragon Talon", "Id": "314", "srvstfunc": "24", "srvdofunc": "42", "calc1": "lvl/6+1", "calc2": "dm34",
			"calc3": "dm56", "calc4": "dm56", "Param3": "50", "Param4": "100", "Param5": "25", "Param6": "99",
			"HitShift": "8", "manashift": "8"},
	} {
		cf.reg.Add(skillFromRow(r))
	}

	return cf
}

func knocked(r DoResult) bool {
	for _, e := range r.Effects {
		if e.Kind == "knockback" {
			return true
		}
	}

	return false
}

func TestDiminishing(t *testing.T) {
	cases := []struct{ lvl, lo, hi, want int }{
		{1, 25, 99, 25 + 15*74/100}, // 1*110/7 = 15
		{20, 25, 99, 87},
		{60, 25, 99, 99},
		{10, 15, 40, 32},
	}

	for _, c := range cases {
		if got := Diminishing(c.lvl, c.lo, c.hi); got != c.want {
			t.Errorf("Diminishing(%d,%d,%d) = %d, want %d", c.lvl, c.lo, c.hi, got, c.want)
		}
	}

	// the calc language's dm56 is the same curve
	cf := nextFixture(map[string]int{"Psychic Hammer": 20})
	if got := cf.env("Psychic Hammer", 20).eval(cf.reg.ByName("Psychic Hammer").Calc[3]); got != 87 {
		t.Errorf("dm56 at 20 = %d, want 87", got)
	}
}

func TestPsychicHammerKnockback(t *testing.T) {
	cases := []struct {
		name  string
		class KnockClass
		roll  uint32
		want  bool
	}{
		{"normal always", KnockNormal, 99, true},
		{"small low roll", KnockSmall, 10, true}, // calc2 = dm34(20) = 50 + 84*50/100 = 92
		{"small high roll", KnockSmall, 95, false},
		{"boss low roll", KnockBoss, 86, true}, // calc3 = 87
		{"boss high roll", KnockBoss, 87, false},
		{"player", KnockPlayer, 90, false},
	}

	for _, c := range cases {
		cf := nextFixture(map[string]int{"Psychic Hammer": 20})
		z := cf.addFoe("z", 3, 0)
		cf.u.roller = &seq{vals: []uint32{c.roll}}
		id := cf.id("Psychic Hammer")
		tg := Target{X: 3, Y: 0, Unit: bossTarget{z, c.class}, UX: 3, UY: 0}
		cf.p.Start(cf.u, id, tg)
		r := cf.p.Do(cf.u, id, tg)

		if !r.OK {
			t.Fatalf("%s: %+v", c.name, r)
		}

		hit := effectOf(t, r, "hit_unit")
		if hit.Desc == nil || hit.Desc.Magic.Max <= 0 {
			t.Errorf("%s: no magic damage %+v", c.name, hit.Desc)
		}

		if knocked(r) != c.want {
			t.Errorf("%s: knockback %v, want %v", c.name, knocked(r), c.want)
		}
	}
}

func TestPsychicHammerNeedsATarget(t *testing.T) {
	cf := nextFixture(map[string]int{"Psychic Hammer": 5})
	id := cf.id("Psychic Hammer")
	tg := Target{X: 9, Y: 9}
	cf.p.Start(cf.u, id, tg)

	if r := cf.p.Do(cf.u, id, tg); r.OK {
		t.Errorf("a cast into empty ground must fail: %+v", r)
	}
}

func TestMindBlastConvertsOrDamages(t *testing.T) {
	// level 10: chance = Diminishing(10, 15, 40) = 32; the roll is "at most"
	cases := []struct {
		roll    uint32
		convert bool
	}{{0, true}, {32, true}, {33, false}, {99, false}}

	for _, c := range cases {
		cf := nextFixture(map[string]int{"Mind Blast": 10})
		cf.addFoe("z", 2, 0)
		cf.u.roller = &seq{vals: []uint32{c.roll, 40}}
		id := cf.id("Mind Blast")
		tg := Target{X: 2, Y: 0}
		cf.p.Start(cf.u, id, tg)
		r := cf.p.Do(cf.u, id, tg)

		var conv, hit *Effect

		for i := range r.Effects {
			switch r.Effects[i].Kind {
			case "convert":
				conv = &r.Effects[i]
			case "hit_unit":
				hit = &r.Effects[i]
			}
		}

		if c.convert {
			if conv == nil || hit != nil || conv.Frames != 190 || conv.State != "conversion" {
				t.Errorf("roll %d: want a 190 frame conversion only, got %+v / %+v", c.roll, conv, hit)
			}
		} else if conv != nil || hit == nil {
			t.Errorf("roll %d: want damage only, got %+v / %+v", c.roll, conv, hit)
		}
	}
}

func TestMindBlastRadius(t *testing.T) {
	cf := nextFixture(map[string]int{"Mind Blast": 10})
	cf.addFoe("near", 5, 0)
	cf.addFoe("far", 9, 0)
	cf.u.roller = &seq{vals: []uint32{99, 99}}
	id := cf.id("Mind Blast")
	tg := Target{X: 0, Y: 0}
	cf.p.Start(cf.u, id, tg)
	r := cf.p.Do(cf.u, id, tg)

	n := 0

	for _, e := range r.Effects {
		if e.Kind == "hit_unit" {
			n++
		}
	}

	if n != 1 {
		t.Errorf("units hit = %d, want 1 (radius par7 = 6)", n)
	}
}

func TestCloakOfShadows(t *testing.T) {
	cf := nextFixture(map[string]int{"Cloak of Shadows": 10})
	id := cf.id("Cloak of Shadows")
	tg := Target{}
	cf.p.Start(cf.u, id, tg)
	r := cf.p.Do(cf.u, id, tg)

	self := effectOf(t, r, "self_state")
	if self.State != "cloak_of_shadows" || len(self.Stats) != 1 || self.Stats[0].Stat != "item_absorbfire_percent" {
		t.Errorf("self state %+v", self)
	}

	area := effectOf(t, r, "area_state")
	if area.State != "cloaked" || area.Origin != "self" || area.Frames != self.Frames || area.Radius <= 0 {
		t.Errorf("area state %+v", area)
	}

	if v, ok := statOf(area.Stats, d2state.StatBlind); !ok || v != 1 {
		t.Errorf("enemies must be blinded: %+v", area.Stats)
	}

	if v, ok := statOf(area.Stats, "skill_armor_percent"); !ok || v >= 0 {
		t.Errorf("armor stat %d %v", v, ok)
	}
}

func TestTauntHitsOneUnit(t *testing.T) {
	cf := nextFixture(map[string]int{"Taunt": 10})
	cf.reg.Add(skillFromRow(row{"skill": "Taunt", "Id": "405", "srvdofunc": "71", "aurafilter": "34179", "auratargetstate": "taunt",
		"aurastat1": "item_tohit_percent", "aurastatcalc1": "ln12", "Param1": "-5", "Param2": "-2",
		"Param3": "-5", "Param4": "-2", "manashift": "8"}))
	a := cf.addFoe("a", 4, 0)
	b := cf.addFoe("b", 2, 0)
	cf.addFoe("c", 30, 0) // beyond the 20 subtile search
	id := cf.id("Taunt")

	// no target unit: the nearest within 20 subtiles
	tg := Target{X: 50, Y: 50}
	cf.p.Start(cf.u, id, tg)
	r := cf.p.Do(cf.u, id, tg)
	e := effectOf(t, r, "unit_state")

	if e.Target.ID() != "b" || e.State != "taunt" {
		t.Errorf("taunt %+v", e)
	}

	if _, ok := statOf(e.Stats, d2state.StatTaunted); !ok {
		t.Errorf("the taunt marker is missing: %+v", e.Stats)
	}

	// a chosen live target wins over the nearest
	tg = Target{X: 4, Y: 0, Unit: a, UX: 4, UY: 0}
	r = cf.p.Do(cf.u, id, tg)

	if e = effectOf(t, r, "unit_state"); e.Target.ID() != "a" {
		t.Errorf("chosen target ignored: %+v", e.Target)
	}

	b.alive = false
	a.alive = false
	tg = Target{X: 50, Y: 50}

	if r = cf.p.Do(cf.u, id, tg); r.OK {
		t.Errorf("nothing to taunt must fail: %+v", r)
	}
}

func TestSmiteShieldDamage(t *testing.T) {
	cf := nextFixture(map[string]int{"Smite": 10})
	z := cf.addFoe("z", 1, 0)
	cf.u.roller = &seq{vals: []uint32{3 << 8}}
	id := cf.id("Smite")
	tg := Target{X: 1, Y: 0, Unit: z, UX: 1, UY: 0}

	// no shield: nothing happens
	su := shieldUnit{testUnit: cf.u}
	cf.p.Start(su, id, tg)

	if r := cf.p.Do(su, id, tg); r.OK {
		t.Errorf("Smite without a shield must fail: %+v", r)
	}

	su = shieldUnit{testUnit: cf.u, lo: 10, hi: 20, has: true}
	r := cf.p.Do(su, id, tg)

	if !r.OK || r.Melee == nil || !r.Melee.Hit {
		t.Fatalf("smite %+v", r)
	}

	// damage = roll(10<<8..20<<8) raised by calc1 = ln34 percent; the stun is calc2
	base := int32(10<<8 + 3<<8)
	pct := int32(cf.env("Smite", 10).eval(cf.reg.ByName("Smite").Calc[1]))
	want := base + base*pct/100

	if r.Melee.Damage.Physical != want || r.Melee.Damage.StunLen <= 0 {
		t.Errorf("physical %d stun %d, want %d and a stun", r.Melee.Damage.Physical, r.Melee.Damage.StunLen, want)
	}
}

func TestChargeEndsNextToTheTarget(t *testing.T) {
	cf := nextFixture(map[string]int{"Charge": 5})
	z := cf.addFoe("z", 10, 4)
	cf.u.roller = &seq{vals: make([]uint32, 20)}
	id := cf.id("Charge")

	// with a target unit: the rush ends one subtile short of it
	tg := Target{X: 10, Y: 4, Unit: z, UX: 10, UY: 4}
	cf.p.Start(cf.u, id, tg)
	r := cf.p.Do(cf.u, id, tg)
	m := effectOf(t, r, "move")

	if !r.OK || m.Mode != "charge" || m.X != 9 || m.Y != 3 || r.Melee == nil {
		t.Errorf("charge %+v melee %+v", m, r.Melee)
	}

	// without one the nearest unit within 3 subtiles is charged
	cf2 := nextFixture(map[string]int{"Charge": 5})
	cf2.addFoe("far", 3, 0)
	cf2.addFoe("close", 2, 0)
	cf2.u.roller = &seq{vals: make([]uint32, 20)}
	tg = Target{X: 30, Y: 30}
	cf2.p.Start(cf2.u, id, tg)
	r = cf2.p.Do(cf2.u, id, tg)

	if !r.OK || r.Melee == nil || r.Melee.Target.ID() != "close" {
		t.Errorf("fallback target %+v", r.Melee)
	}

	// nobody there: refused
	cf3 := nextFixture(map[string]int{"Charge": 5})
	cf3.p.Start(cf3.u, id, tg)

	if r = cf3.p.Do(cf3.u, id, tg); r.OK {
		t.Errorf("empty charge must fail: %+v", r)
	}
}

func TestDragonTalonKnockback(t *testing.T) {
	cases := []struct {
		class KnockClass
		roll  uint32
		want  bool
	}{
		{KnockNormal, 99, true}, // the exe's default chance is 100
		{KnockBoss, 10, true},
		{KnockBoss, 95, false},
		{KnockPlayer, 95, false},
	}

	for _, c := range cases {
		id := nextFixture(map[string]int{"Dragon Talon": 12}).id("Dragon Talon")

		// probe: count the rolls the kicks use, then put the knock roll right after them
		probe := nextFixture(map[string]int{"Dragon Talon": 12})
		pz := probe.addFoe("z", 1, 0)
		pr := &tallyRoller{}
		probe.u.roller = pr
		ptg := Target{X: 1, Y: 0, Unit: bossTarget{pz, c.class}, UX: 1, UY: 0}
		probe.p.Start(probe.u, id, ptg)
		probe.p.Do(probe.u, id, ptg)

		// the probe ran the knock roll when the class has a chance (always here)
		used := pr.n - 1

		cf := nextFixture(map[string]int{"Dragon Talon": 12})
		z := cf.addFoe("z", 1, 0)
		cf.u.roller = &seq{vals: append(make([]uint32, used), c.roll)}
		tg := Target{X: 1, Y: 0, Unit: bossTarget{z, c.class}, UX: 1, UY: 0}
		cf.p.Start(cf.u, id, tg)
		r := cf.p.Do(cf.u, id, tg)

		if knocked(r) != c.want {
			t.Errorf("class %d roll %d: knockback %v, want %v (kicks %d, rolls %d)", c.class, c.roll, knocked(r), c.want, len(r.Melees), used)
		}
	}
}
