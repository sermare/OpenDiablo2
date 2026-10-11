package d2skill

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2calc"
)

// Rows with the patch_d2 values of the skills of batch 3.
func init() {
	rows = append(rows,
		row{"skill": "Feral Rage", "Id": "700", "srvstfunc": "56", "srvdofunc": "120", "aurastate": "feralrage",
			"auralencalc": "par1", "aurastat1": "velocitypercent", "aurastatcalc1": "dm34", "aurastat2": "lifedrainmindam",
			"aurastatcalc2": "par2 * lvl", "aurastat3": "lifedrainmaxdam", "aurastatcalc3": "par2 * lvl", "range": "h2h",
			"calc1": "ln56", "calc2": "lvl/par7 + par8", "Param1": "500", "Param2": "4", "Param3": "10", "Param4": "70",
			"Param5": "50", "Param6": "5", "Param7": "2", "Param8": "3", "ToHit": "20", "LevToHit": "10", "HitShift": "8",
			"SrcDam": "128", "manashift": "8", "mana": "3"},
		row{"skill": "Maul", "Id": "701", "srvstfunc": "56", "srvdofunc": "120", "aurastate": "maul", "auralencalc": "par4",
			"aurastat1": "damagepercent", "aurastatcalc1": "lvl*par3", "aurastat2": "stunlength", "aurastatcalc2": "dm56",
			"range": "h2h", "calc1": "0", "calc2": "lvl/par7 + par8", "Param3": "20", "Param4": "500", "Param5": "10",
			"Param6": "100", "Param7": "2", "Param8": "3", "ToHit": "20", "LevToHit": "10", "HitShift": "8", "SrcDam": "128",
			"manashift": "8", "mana": "3"},
		row{"skill": "Leap", "Id": "702", "srvstfunc": "40", "srvdofunc": "77", "srvmissilea": "leapknockback",
			"aurarangecalc": "dm12", "calc1": "ln34", "Param1": "4", "Param2": "30", "Param3": "4", "Param4": "1",
			"HitShift": "8", "manashift": "8", "mana": "2"},
		row{"skill": "Leap Attack", "Id": "703", "srvstfunc": "41", "srvdofunc": "78", "range": "none", "calc1": "ln34",
			"Param3": "100", "Param4": "30", "ToHit": "50", "LevToHit": "15", "HitShift": "8", "SrcDam": "128",
			"manashift": "8", "mana": "9"},
		row{"skill": "Dragon Claw", "Id": "704", "srvstfunc": "25", "srvdofunc": "46", "range": "h2h", "calc1": "ln12",
			"Param1": "50", "Param2": "5", "ToHit": "40", "LevToHit": "25", "HitShift": "8", "SrcDam": "128",
			"manashift": "8", "mana": "2"},
		row{"skill": "Dragon Tail", "Id": "705", "srvstfunc": "27", "srvdofunc": "50", "aurarangecalc": "par3",
			"range": "h2h", "calc1": "ln12", "Param1": "50", "Param2": "10", "Param3": "6", "Param4": "-40", "ToHit": "20",
			"LevToHit": "15", "Kick": "1", "HitShift": "8", "EType": "fire", "manashift": "8", "mana": "10"},
		row{"skill": "Dragon Flight", "Id": "706", "srvstfunc": "12", "srvdofunc": "52", "range": "h2h", "ToHit": "20",
			"LevToHit": "15", "HitShift": "8", "SrcDam": "128", "manashift": "8", "mana": "10"},
		row{"skill": "Wearwolf", "Id": "707", "srvdofunc": "116", "aurastate": "wolf", "auralencalc": "1000",
			"aurastat1": "skill_staminapercent", "aurastatcalc1": "par1", "aurastat2": "attackrate", "aurastatcalc2": "dm34",
			"delay": "25", "Param1": "25", "Param2": "25", "Param3": "10", "Param4": "80", "InTown": "1", "manashift": "8",
			"mana": "15"},
		row{"skill": "Thunder Storm", "Id": "708", "srvstfunc": "13", "srvdofunc": "29", "srvmissilea": "thunderstorm1",
			"aurastate": "thunderstorm", "auralencalc": "ln12", "periodic": "1", "Param1": "800", "Param2": "200",
			"Param3": "25", "Param4": "100", "Param7": "17", "EType": "ltng", "EMin": "1", "EMax": "100", "HitShift": "8",
			"manashift": "8", "mana": "19"},
	)
}

func b3Fixture(levels map[string]int) *classFixture {
	cf := newClassFixture(levels)
	cf.u.ar = 100000
	cf.u.roller = &seq{vals: make([]uint32, 200)}

	cf.reg.ByName("Thunder Storm").PerDelay = d2calc.Compile("(100-dm56) * par4/100 + par3", d2calc.KindSkill)

	return cf
}

func statValue(t *testing.T, e Effect, stat string) int {
	t.Helper()

	v, ok := statOf(e.Stats, stat)
	if !ok {
		t.Fatalf("no stat %q in %+v", stat, e.Stats)
	}

	return v
}

// Feral Rage evaluates its aura stats with the stack count as the level and
// grows the count by one per cast up to calc2; it keeps the other states of
// its group (0x5c57c0).
func TestFeralRageStacksAndLevels(t *testing.T) {
	cf := b3Fixture(map[string]int{"Feral Rage": 10, "Maul": 4})
	z := cf.addFoe("z", 3, 0)

	// calc2 = lvl/par7 + par8 = 10/2 + 3 = 8
	for i, want := range []int{1, 2, 3} {
		_, r := cf.castOn("Feral Rage", z)
		if !r.OK {
			t.Fatalf("cast %d: %+v", i, r)
		}

		e := effectOf(t, r, "self_state")
		if !e.NoGroup || e.Stack != 8 || e.Frames != 500 {
			t.Errorf("cast %d: effect %+v", i, e)
		}

		if got := statValue(t, e, "feralrage"); got != want {
			t.Errorf("cast %d: stack %d, want %d", i, got, want)
		}

		// par2 * lvl with lvl = the stack, not the skill level
		if got := statValue(t, e, "lifedrainmindam"); got != 4*want {
			t.Errorf("cast %d: leech %d, want %d", i, got, 4*want)
		}

		cf.u.stats["feralrage"] = want // what the engine reports after the cast
	}

	// at the limit the count stays
	cf.u.stats["feralrage"] = 8
	_, r := cf.castOn("Feral Rage", z)

	if got := statValue(t, effectOf(t, r, "self_state"), "feralrage"); got != 8 {
		t.Errorf("stack at the limit = %d, want 8", got)
	}

	// Maul: damagepercent = stack * par3, limit 4/2 + 3 = 5, applies even when the blow misses
	cf.u.ar = 0
	cf.u.stats["maul"] = 2
	_, r = cf.castOn("Maul", z)
	e := effectOf(t, r, "self_state")

	if statValue(t, e, "maul") != 3 || statValue(t, e, "damagepercent") != 60 || e.Stack != 5 || e.Frames != 500 {
		t.Errorf("maul %+v", e)
	}
}

func TestDragonTailFire(t *testing.T) {
	cases := []struct {
		kick        int32
		calc1, mast int
		want        int32
	}{{100 << 8, 50, 0, 50 << 8}, {100 << 8, 50, 30, 80 << 8}, {0, 50, 30, 0}, {-5, 50, 0, 0}, {200, 100, 0, 200}}

	for _, c := range cases {
		if got := DragonTailFire(c.kick, c.calc1, c.mast); got != c.want {
			t.Errorf("DragonTailFire(%d,%d,%d) = %d, want %d", c.kick, c.calc1, c.mast, got, c.want)
		}
	}
}

// Dragon Tail: the area is the explosion of the kick around the target, with
// the knockback bit, radius par3; calc1 is no bonus on the kick itself.
func TestDragonTailExplosion(t *testing.T) {
	cf := b3Fixture(map[string]int{"Dragon Tail": 1})
	z := cf.addFoe("z", 3, 0)
	cf.u.stats["passive_fire_mastery"] = 20
	cf.u.stats["strength"] = 100 // the kick damage is (str + dex - 20) / 4

	_, r := cf.castOn("Dragon Tail", z)
	if !r.OK || r.Melee == nil || !r.Melee.Hit {
		t.Fatalf("kick: %+v", r)
	}

	e := effectOf(t, r, "area_hit")
	want := DragonTailFire(r.Melee.Damage.Physical, 50, 20) // ln12 at level 1 = Param1 = 50

	if e.Radius != 6 || !e.Knock || e.Origin != "aim" || e.X != 3 || e.Y != 0 || e.Desc == nil ||
		e.Desc.Fire.Min != want || e.Desc.Fire.Max != want || want <= 0 {
		t.Errorf("explosion %+v desc %+v want fire %d", e, e.Desc, want)
	}

	// a kick that deals nothing leaves no explosion
	cf.u.stats["strength"] = 0
	cf.u.cooldowns = map[int]int{}
	_, r = cf.castOn("Dragon Tail", z)

	for _, ef := range r.Effects {
		if ef.Kind == "area_hit" {
			t.Errorf("a harmless kick exploded: %+v", ef)
		}
	}
}

func TestDragonClawTwoBlowsWithChargeBonus(t *testing.T) {
	cf := b3Fixture(map[string]int{"Dragon Claw": 3})
	z := cf.addFoe("z", 2, 0)

	_, r := cf.castOn("Dragon Claw", z)
	if !r.OK || len(r.Melees) != 2 {
		t.Fatalf("claw: ok=%v blows=%d", r.OK, len(r.Melees))
	}
}

func TestLeapAttackSingleVictim(t *testing.T) {
	cf := b3Fixture(map[string]int{"Leap Attack": 5})
	near := cf.addFoe("near", 10, 0)
	cf.addFoe("other", 11, 1)
	cf.addFoe("far", 40, 0)

	tg := Target{X: 10, Y: 0, Unit: near, UX: 10, UY: 0}
	id := cf.id("Leap Attack")
	cf.p.Start(cf.u, id, tg)
	r := cf.p.Do(cf.u, id, tg)

	if !r.OK || len(r.Melees) != 1 || r.Melee.Target.ID() != "near" {
		t.Fatalf("one victim expected: %+v melees=%d", r.Reason, len(r.Melees))
	}

	var kinds []string
	for _, e := range r.Effects {
		kinds = append(kinds, e.Kind)
	}

	for _, want := range []string{"move", "knockback", "overlay", "unit_clear_state"} {
		found := false

		for _, k := range kinds {
			found = found || k == want
		}

		if !found {
			t.Errorf("missing effect %q in %v", want, kinds)
		}
	}

	if e := effectOf(t, r, "unit_clear_state"); e.State != "stunned" || e.Target.ID() != "near" {
		t.Errorf("stun removal %+v", e)
	}

	// the skill's target is out of reach after the landing: the nearest unit is struck instead
	cf2 := b3Fixture(map[string]int{"Leap Attack": 5})
	cf2.addFoe("close", 12, 0)
	far := cf2.addFoe("far", 60, 0)
	tg = Target{X: 12, Y: 0, Unit: far, UX: 60, UY: 0}
	id = cf2.id("Leap Attack")
	cf2.p.Start(cf2.u, id, tg)
	r = cf2.p.Do(cf2.u, id, tg)

	if len(r.Melees) != 1 || r.Melee.Target.ID() != "close" {
		t.Errorf("scan victim: %+v", r.Melees)
	}

	// nobody around: the jump happens, nothing is struck
	cf3 := b3Fixture(map[string]int{"Leap Attack": 5})
	_, r = cf3.cast("Leap Attack", 10, 0)

	if !r.OK || len(r.Melees) != 0 {
		t.Errorf("empty landing: %+v", r)
	}

	effectOf(t, r, "move")
}

func TestLeapKnocksTheLanding(t *testing.T) {
	cf := b3Fixture(map[string]int{"Leap": 3})

	_, r := cf.cast("Leap", 10, 0)
	e := effectOf(t, r, "knock_area")

	// calc1 = ln34 = Param3 + Param4 * (lvl - 1) = 4 + 2
	if e.Radius != 6 || e.X != 10 || e.Y != 0 {
		t.Errorf("knock area %+v", e)
	}

	// a leaping monster does not hit the landing
	cf.u.player = false
	_, r = cf.cast("Leap", 10, 0)

	for _, ef := range r.Effects {
		if ef.Kind == "knock_area" {
			t.Errorf("monster leap knocked: %+v", ef)
		}
	}
}

func TestDragonFlightRules(t *testing.T) {
	cf := b3Fixture(map[string]int{"Dragon Flight": 4})
	z := cf.addFoe("z", 10, 0)

	// no target: nothing happens (the exe returns 0)
	if _, r := cf.cast("Dragon Flight", 10, 0); r.OK {
		t.Errorf("a cast without a target must fail: %+v", r)
	}

	// levels.txt Teleport 0: refused
	cf.p.TeleportFlag = func() int { return 0 }

	if _, r := cf.castOn("Dragon Flight", z); r.OK {
		t.Errorf("Teleport 0 must refuse: %+v", r)
	}

	// Teleport 1: lands next to the target and kicks
	cf.p.TeleportFlag = func() int { return 1 }

	_, r := cf.castOn("Dragon Flight", z)
	if !r.OK || r.Melee == nil {
		t.Fatalf("flight: %+v", r)
	}

	if e := effectOf(t, r, "move"); e.Mode != "teleport" || e.X != 10 || e.Y != 0 {
		t.Errorf("landing %+v", e)
	}
}

func TestFormIsASwitchAndFreeInTheForm(t *testing.T) {
	cf := b3Fixture(map[string]int{"Wearwolf": 5})

	_, r := cf.cast("Wearwolf", 0, 0)
	e := effectOf(t, r, "self_state")

	if !e.Toggle || e.State != "wolf" || e.Frames != 1000 {
		t.Errorf("form effect %+v", e)
	}

	sk := cf.reg.ByName("Wearwolf")
	free := &shiftUnit{testUnit: cf.u, shifted: true}

	if got, want := costOf(cf.u, sk, 5), sk.ManaCost(5); got != want || want == 0 {
		t.Errorf("cost outside a form = %d, want %d", got, want)
	}

	if got := costOf(free, sk, 5); got != 0 {
		t.Errorf("cost inside a form = %d, want 0", got)
	}

	// other skills still pay in a form
	other := cf.reg.ByName("Leap")
	if got := costOf(free, other, 3); got != other.ManaCost(3) {
		t.Errorf("only srvdofunc 116 is free: %d", got)
	}

	// the pipeline lets a hero without mana switch back
	cf.u.cooldowns = map[int]int{}
	free.mana = 0
	cf.u.mana = 0
	free.shifted = true

	if st := cf.p.Start(free, sk.ID, Target{}); !st.OK {
		t.Errorf("start inside a form with no mana: %+v", st)
	}

	if st := cf.p.Start(cf.u, sk.ID, Target{}); st.OK {
		t.Errorf("start outside a form with no mana must fail: %+v", st)
	}
}

type shiftUnit struct {
	*testUnit
	shifted bool
}

func (s *shiftUnit) Shapeshifted() bool { return s.shifted }

func TestThunderStormUsesPerDelayAndParam7(t *testing.T) {
	cf := b3Fixture(map[string]int{"Thunder Storm": 1})

	_, r := cf.cast("Thunder Storm", 0, 0)
	e := effectOf(t, r, "storm")

	// perdelay = (100 - dm56) * par4/100 + par3; Param5/6 are 0 so dm56 = 0 at level 1: 100 + 25
	if e.Interval < 25 || e.Radius != 17 || e.Mode != "nearest" {
		t.Errorf("storm %+v", e)
	}

	want := cf.env("Thunder Storm", 1).eval(cf.reg.ByName("Thunder Storm").PerDelay)
	if e.Interval != want {
		t.Errorf("interval %d, want perdelay %d", e.Interval, want)
	}
}
