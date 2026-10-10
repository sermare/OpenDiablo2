package d2skill

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2calc"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2missile"
)

// Rows with the real patch_d2 values of the skills class_abc.go adds (ids
// 400+, so they never collide with the shared registry).
func init() {
	rows = append(rows,
		row{"skill": "Dopplezon", "Id": "400", "charclass": "ama", "srvdofunc": "15", "summon": "dopplezon",
			"pettype": "dopplezon", "petmax": "1", "summode": "NU", "calc1": "lvl*par4", "calc2": "ln12", "calc3": "par3",
			"Param1": "250", "Param2": "125", "Param3": "50", "Param4": "10", "manashift": "8"},
		row{"skill": "Valkyrie", "Id": "401", "charclass": "ama", "srvdofunc": "16", "summon": "valkyrie",
			"pettype": "valkyrie", "petmax": "1", "summode": "NU", "calc1": "par1 * (lvl - 1) + skill('Dopplezon'.blvl) * par8",
			"calc2": "ln56", "Param1": "20", "Param8": "20", "aurastat1": "strength", "aurastatcalc1": "lvl * par2",
			"Param2": "25", "manashift": "8", "delay": "150"},
		row{"skill": "Blade Fury", "Id": "402", "charclass": "ass", "srvstfunc": "26", "srvdofunc": "48",
			"srvmissilea": "bladefragment1", "Param3": "3", "Param4": "5", "usemanaondo": "1", "HitShift": "8", "manashift": "8"},
		row{"skill": "Dragon Flight", "Id": "403", "charclass": "ass", "srvstfunc": "12", "srvdofunc": "52", "Kick": "1",
			"Param7": "27", "aurarangecalc": "par7", "ToHit": "60", "LevToHit": "25", "HitShift": "8", "manashift": "8"},
		row{"skill": "Find Potion", "Id": "404", "charclass": "bar", "srvstfunc": "33", "srvdofunc": "69", "TargetCorpse": "1",
			"calc1": "dm12", "Param2": "100", "Param3": "30", "Param4": "10", "manashift": "8"},
		row{"skill": "Find Item", "Id": "405", "charclass": "bar", "srvstfunc": "34", "srvdofunc": "72", "TargetCorpse": "1",
			"calc1": "dm12", "Param1": "5", "Param2": "60", "Param3": "30", "Param4": "5", "manashift": "8"},
		row{"skill": "Grim Ward", "Id": "406", "charclass": "bar", "srvstfunc": "33", "srvdofunc": "75", "TargetCorpse": "1",
			"auratargetstate": "terror", "auralencalc": "par6", "aurarangecalc": "ln12", "calc1": "ln34", "Param1": "3",
			"Param2": "1", "Param6": "60", "manashift": "8"},
		row{"skill": "Blade Shield", "Id": "408", "charclass": "ass", "srvstfunc": "28", "srvdofunc": "54", "aurastate": "bladeshield",
			"auralencalc": "ln12", "aurarangecalc": "par4", "periodic": "1", "perdelay": "par3", "Param1": "500", "Param2": "125",
			"Param3": "25", "Param4": "6", "SrcDam": "32", "MinDam": "1", "MaxDam": "30", "HitShift": "8", "manashift": "8"},
		row{"skill": "Whirlwind", "Id": "407", "charclass": "bar", "srvstfunc": "38", "srvdofunc": "76", "aurastate": "whirlwind",
			"calc1": "ln12", "Param1": "-50", "Param2": "8", "LevToHit": "5", "SrcDam": "128", "HitShift": "8", "manashift": "8"},
	)
}

func abcFixture(levels map[string]int) *classFixture {
	cf := newClassFixture(levels)
	tbl := classMissiles()
	tbl["bladefragment1"] = &d2missile.Spec{ID: 210, Name: "bladefragment1", SrvDoFunc: 1, Vel: 22, MaxVel: 22, Range: 40,
		CollideType: 3, CollideKill: true, CanSlow: true}
	cf.p.Missiles = tbl
	cf.sim = d2missile.NewSim(cf.w, tbl)
	cf.sim.OnEvent = func(e d2missile.Event) { cf.evs = append(cf.evs, e) }
	cf.p.Sim = cf.sim

	return cf
}

func corpseTarget(id string) Target {
	return Target{X: 8, Y: 4, Corpse: true, CX: 8, CY: 4, CorpseID: id, CorpseHP: 40, CorpseKey: "zombie1"}
}

func (cf *classFixture) castCorpse(name string, tg Target) (StartResult, DoResult) {
	id := cf.id(name)
	st := cf.p.Start(cf.u, id, tg)

	if !st.OK {
		return st, DoResult{}
	}

	return st, cf.p.Do(cf.u, id, tg)
}

func TestFindPotionCodeTable(t *testing.T) {
	// golden: the table at 0x73e988 of Game.exe 1.14b (act, difficulty, column)
	tests := []struct {
		act, diff, col int
		want           string
	}{
		{1, 0, 0, "hp2"}, {1, 0, 1, "mp2"}, {1, 0, 2, "rvs"},
		{2, 0, 0, "hp3"}, {3, 0, 2, "rvs"}, {4, 0, 2, "rvl"}, {5, 0, 0, "hp4"},
		{1, 1, 0, "hp4"}, {2, 1, 1, "mp5"}, {2, 1, 0, "hp4"}, {3, 1, 0, "hp5"},
		{1, 2, 0, "hp5"}, {5, 2, 2, "rvl"},
		{0, 0, 0, ""}, {6, 0, 0, ""}, {1, 3, 0, ""}, {1, 0, 3, ""},
	}

	for _, tc := range tests {
		if got := FindPotionCode(tc.act, tc.diff, tc.col); got != tc.want {
			t.Errorf("FindPotionCode(%d,%d,%d) = %q, want %q", tc.act, tc.diff, tc.col, got, tc.want)
		}
	}
}

func TestFindPotion(t *testing.T) {
	// dm12 at level 5 with Param1 0, Param2 100: 110*5*100/(100*11) = 50 percent
	tests := []struct {
		name         string
		rolls        []uint32
		drop         bool
		col          int
		wantRollsLen int
	}{
		{"roll above the chance drops nothing", []uint32{50}, false, 0, 0},
		{"health", []uint32{49, 80}, true, 0, 0},
		{"mana below Param3", []uint32{0, 29}, true, 1, 0},
		{"rejuvenation in the next Param4", []uint32{10, 30}, true, 2, 0},
		{"rejuvenation upper edge", []uint32{10, 39}, true, 2, 0},
		{"health at the edge", []uint32{10, 40}, true, 0, 0},
	}

	for _, tc := range tests {
		cf := abcFixture(map[string]int{"Find Potion": 5})
		cf.u.roller = &seq{vals: tc.rolls}

		_, r := cf.castCorpse("Find Potion", corpseTarget("c1"))
		if !r.OK {
			t.Fatalf("%s: %s", tc.name, r.Reason)
		}

		o := effectOf(t, r, "loot").Loot
		if o.Chance != 50 || o.Kind != "potion" || o.Drop != tc.drop || o.CorpseID != "c1" || (tc.drop && o.Column != tc.col) {
			t.Errorf("%s: order %+v", tc.name, o)
		}

		if len(cf.u.roller.(*seq).vals) != tc.wantRollsLen {
			t.Errorf("%s: %d rolls left, want %d", tc.name, len(cf.u.roller.(*seq).vals), tc.wantRollsLen)
		}
	}
}

func TestFindItemBuckets(t *testing.T) {
	// Param1..4 = 5, 60, 30, 5: [0,5) type 1, [5,65) 2, [65,95) 3, [95,100) 4 (0x5d7210)
	tests := []struct{ r, want int }{{0, 1}, {4, 1}, {5, 2}, {64, 2}, {65, 3}, {94, 3}, {95, 4}, {99, 4}}

	for _, tc := range tests {
		cf := abcFixture(map[string]int{"Find Item": 5})
		cf.u.roller = &seq{vals: []uint32{0, uint32(tc.r)}}

		_, r := cf.castCorpse("Find Item", corpseTarget("c1"))
		o := effectOf(t, r, "loot").Loot

		if !o.Drop || o.Kind != "item" || o.TCType != tc.want {
			t.Errorf("roll %d: type %d, want %d (%+v)", tc.r, o.TCType, tc.want, o)
		}
	}

	// the chance roll can fail
	cf := abcFixture(map[string]int{"Find Item": 5})
	cf.u.roller = &seq{vals: []uint32{99}}

	_, r := cf.castCorpse("Find Item", corpseTarget("c1"))
	if o := effectOf(t, r, "loot").Loot; o.Drop {
		t.Errorf("a failed chance roll must not drop: %+v", o)
	}
}

func TestFindNeedsAnUnlootedCorpse(t *testing.T) {
	for _, name := range []string{"Find Potion", "Find Item", "Grim Ward"} {
		cf := abcFixture(map[string]int{name: 3})

		if st, _ := cf.cast(name, 5, 5); st.OK || st.Reason != ReasonNoCorpse {
			t.Errorf("%s without a corpse: %+v", name, st)
		}

		if name == "Grim Ward" {
			continue // a ward is also raised from a looted corpse (it only sets state 0x76)
		}

		tg := corpseTarget("c9")
		tg.CorpseLooted = true

		_, r := cf.castCorpse(name, tg)
		if r.OK || r.Reason != ReasonNoCorpse {
			t.Errorf("%s on a looted corpse: ok=%v %q", name, r.OK, r.Reason)
		}
	}
}

func TestGrimWard(t *testing.T) {
	cf := abcFixture(map[string]int{"Grim Ward": 4})

	_, r := cf.castCorpse("Grim Ward", corpseTarget("c2"))
	if !r.OK {
		t.Fatal(r.Reason)
	}

	w := effectOf(t, r, "ward").Ward
	// ln12 = 3 + 3*1 = 6 subtiles, auralencalc par6 = 60 frames, 6 frame pulses for 200 frames
	if w.Radius != 6 || w.Fear != 60 || w.Period != 6 || w.Life != 200 || w.State != "terror" || w.X != 8 || w.Y != 4 || w.CorpseID != "c2" {
		t.Errorf("ward %+v", w)
	}
}

func TestDopplezonAndValkyrie(t *testing.T) {
	cf := abcFixture(map[string]int{"Dopplezon": 4, "Valkyrie": 3})

	_, r := cf.cast("Dopplezon", 5, 5)
	if !r.OK {
		t.Fatal(r.Reason)
	}

	o := effectOf(t, r, "summon").Summon
	// calc2 ln12 = 250 + 3*125 frames of life, calc3 par3 = 50 percent of the owner's life
	if o.Key != "dopplezon" || o.Frames != 625 || o.OwnerHPPct != 50 || o.HPPct != 0 || o.Max != 1 || o.Kind != "minion" {
		t.Errorf("dopplezon order %+v", o)
	}

	// Valkyrie: life bonus calc1 = 20*(3-1) + Dopplezon blvl 4 * 20 = 120 percent
	_, r = cf.cast("Valkyrie", 5, 5)
	if !r.OK {
		t.Fatal(r.Reason)
	}

	o = effectOf(t, r, "summon").Summon
	if o.Key != "valkyrie" || o.HPPct != 120 || o.Max != 1 || o.Count != 1 {
		t.Errorf("valkyrie order %+v", o)
	}

	if v, ok := statOf(o.Stats, "strength"); !ok || v != 75 {
		t.Errorf("valkyrie stats %v", o.Stats)
	}

	// only a player summons the Valkyrie (0x5daf10 returns 0 for a monster)
	cf.u.player = false
	cf.u.cooldowns = map[int]int{}

	if _, r = cf.cast("Valkyrie", 5, 5); r.OK {
		t.Error("a non player cast a Valkyrie")
	}
}

func TestBladeFuryStream(t *testing.T) {
	cf := abcFixture(map[string]int{"Blade Fury": 6})
	cf.w.frame = 100

	_, r := cf.cast("Blade Fury", 30, 0)
	if !r.OK || len(r.Missiles) != 1 || r.Missiles[0].Spec.Name != "bladefragment1" {
		t.Fatalf("first cast: ok=%v %q missiles=%d", r.OK, r.Reason, len(r.Missiles))
	}

	// the next blade is blocked for prgcalc1 (par4 = 5) - 1 frames
	cf.w.frame = 103
	if st, _ := cf.cast("Blade Fury", 30, 0); st.OK || st.Reason != ReasonCooldown {
		t.Errorf("a blade 3 frames later: %+v", st)
	}

	cf.w.frame = 104

	if st, r := cf.cast("Blade Fury", 30, 0); !st.OK || !r.OK || len(r.Missiles) != 1 {
		t.Errorf("a blade 4 frames later: %+v ok=%v", st, r.OK)
	}
}

func TestDragonFlight(t *testing.T) {
	cf := abcFixture(map[string]int{"Dragon Flight": 4})
	foe := cf.addFoe("z", 20, 0)
	cf.u.roller = &seq{vals: make([]uint32, 50)}

	// with a target: arrive on the caster's side of it, then the kick lands (kicks never miss)
	_, r := cf.castOn("Dragon Flight", foe)
	if !r.OK || r.Melee == nil || !r.Melee.Hit {
		t.Fatalf("flight at a target: ok=%v %q", r.OK, r.Reason)
	}

	if e := effectOf(t, r, "move"); e.Mode != "teleport" || e.X != 19 || e.Y != 0 {
		t.Errorf("landing %+v", e)
	}

	// without a target: a teleport to the aim point, refused on a wall
	_, r = cf.cast("Dragon Flight", 12, 3)
	if e := effectOf(t, r, "move"); !r.OK || e.X != 12 || e.Y != 3 || r.Melee != nil {
		t.Errorf("flight to a point: %+v %+v", r, e)
	}

	cf.p.Walkable = func(x, y int) bool { return x != 40 }

	if _, r = cf.cast("Dragon Flight", 40, 3); r.OK || r.Reason != ReasonLOS {
		t.Errorf("flight into a wall: %+v", r)
	}
}

func TestBladeShield(t *testing.T) {
	cf := abcFixture(map[string]int{"Blade Shield": 3})
	// the shared test loader does not read perdelay
	cf.reg.ByName("Blade Shield").PerDelay = d2calc.Compile("par3", d2calc.KindSkill)

	_, r := cf.cast("Blade Shield", 5, 5)
	if !r.OK {
		t.Fatal(r.Reason)
	}

	// ln12 = 500 + 2*125 frames, par4 = 6 subtiles, perdelay par3 = 25
	st := effectOf(t, r, "self_state")
	sh := effectOf(t, r, "shield")

	if st.State != "bladeshield" || st.Frames != 750 || sh.Frames != 750 || sh.Radius != 6 || sh.Interval != 25 || sh.Desc == nil {
		t.Errorf("state %+v shield %+v", st, sh)
	}
}

func TestWhirlwindOrderAndDelay(t *testing.T) {
	cf := abcFixture(map[string]int{"Whirlwind": 5})
	cf.u.wmin, cf.u.wmax = 0, 0

	_, r := cf.cast("Whirlwind", 25, 0)
	if !r.OK {
		t.Fatal(r.Reason)
	}

	w := effectOf(t, r, "whirl").Whirl
	// calc1 ln12 = -50 + 4*8 = -18 percent; bare handed delay is 10 frames; reach 5 subtiles
	if w.Pct != -18 || w.Delay != 10 || w.Radius != 5 || w.X != 25 || w.State != "whirlwind" {
		t.Errorf("whirl order %+v", w)
	}

	// SKILL_WhirlwindComputeNextStepDelay (0x5d7de0): weapon frames to delay
	for _, tc := range []struct {
		frames int
		armed  bool
		want   int
	}{{0, false, 10}, {11, true, 4}, {12, true, 6}, {14, true, 6}, {15, true, 8}, {17, true, 8}, {18, true, 10}, {19, true, 10},
		{20, true, 12}, {22, true, 12}, {23, true, 14}, {25, true, 14}, {26, true, 16}} {
		if got := WhirlDelay(tc.frames, tc.armed); got != tc.want {
			t.Errorf("WhirlDelay(%d,%v) = %d, want %d", tc.frames, tc.armed, got, tc.want)
		}
	}

	// a hit of the whirl uses calc1 as the damage percent
	foe := cf.addFoe("z", 3, 0)
	cf.u.roller = &seq{vals: make([]uint32, 50)}
	cf.u.ar = 100000

	if m := cf.p.WhirlStrike(cf.u, cf.id("Whirlwind"), foe); m == nil || !m.Hit {
		t.Errorf("whirl strike %+v", m)
	}
}

// Every non passive skill of the three classes has a ported do function.
func TestABCSkillsAreImplemented(t *testing.T) {
	reg := loadRealRegistry(t)

	n := 0

	for _, id := range reg.IDs() {
		sk := reg.ByID(id)
		if sk.CharClass != "ama" && sk.CharClass != "bar" && sk.CharClass != "ass" {
			continue
		}

		n++

		if !sk.Passive && !Implemented(sk) {
			t.Errorf("%s (%s, srvdofunc %d) is not implemented", sk.Name, sk.CharClass, sk.SrvDoFunc)
		}
	}

	if n != 90 {
		t.Errorf("%d skills of the three classes, want 90", n)
	}
}
