package d2skill

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2missile"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2path"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2state"
)

// Rows with the real patch_d2 skills.txt values of the class skills under
// test (HitShift 8 unless noted). They are appended to the shared test
// registry.
func init() {
	rows = append(rows,
		row{"skill": "Amplify Damage", "Id": "300", "charclass": "nec", "srvdofunc": "30", "auratargetstate": "amplifydamage",
			"auralencalc": "ln34", "aurarangecalc": "ln12", "aurafilter": "3", "aurastat1": "damageresist",
			"aurastatcalc1": "-par5", "Param1": "3", "Param2": "1", "Param3": "200", "Param4": "75", "Param5": "100",
			"manashift": "8", "mana": "4"},
		row{"skill": "Iron Maiden", "Id": "301", "srvdofunc": "30", "auratargetstate": "ironmaiden", "auralencalc": "ln34",
			"aurarangecalc": "ln12", "aurafilter": "3", "calc1": "ln56", "Param1": "7", "Param3": "300", "Param4": "60",
			"Param5": "200", "Param6": "25", "manashift": "8"},
		row{"skill": "Terror", "Id": "302", "srvdofunc": "30", "auratargetstate": "terror", "auralencalc": "ln34",
			"aurarangecalc": "ln12", "aurafilter": "2", "Param1": "4", "Param3": "200", "Param4": "25", "manashift": "8"},
		row{"skill": "Double Swing", "Id": "303", "srvdofunc": "70", "calc1": "skill('Bash'.blvl)*par8", "calc3": "par5",
			"Param5": "50", "Param8": "5", "ToHit": "15", "HitShift": "8", "SrcDam": "128", "manashift": "8"},
		row{"skill": "Zeal", "Id": "304", "srvstfunc": "37", "srvdofunc": "13", "calc1": "min((par5 + lvl -1), par6)",
			"calc2": "((lvl < 5) ? 0 : ((lvl-4) * par4) )", "Param2": "100", "Param4": "6", "Param5": "2", "Param6": "5",
			"SrcDam": "128", "HitShift": "8", "manashift": "8"},
		row{"skill": "Fend", "Id": "305", "srvstfunc": "9", "srvdofunc": "13", "calc1": "12", "calc2": "ln34",
			"Param1": "1", "Param2": "60", "Param3": "70", "Param4": "10", "SrcDam": "128", "HitShift": "8", "ToHit": "40",
			"manashift": "8"},
		row{"skill": "Multiple Shot", "Id": "306", "srvstfunc": "4", "srvdofunc": "8", "srvmissilea": "multipleshotarrow",
			"calc1": "min(24,ln12)", "Param1": "2", "Param2": "1", "SrcDam": "96", "HitShift": "8", "manashift": "8"},
		row{"skill": "Might", "Id": "307", "charclass": "pal", "srvdofunc": "65", "aurastate": "might",
			"auratargetstate": "might", "aurarangecalc": "ln12", "aurafilter": "73731", "aura": "1", "aurastat1": "damagepercent",
			"aurastatcalc1": "ln34", "Param1": "16", "Param2": "2", "Param3": "40", "Param4": "10", "manashift": "8"},
		row{"skill": "Holy Freeze", "Id": "308", "srvdofunc": "81", "aurastate": "holywind", "auratargetstate": "holywindcold",
			"aurarangecalc": "ln12", "aurafilter": "303747", "aura": "1", "aurastat1": "velocitypercent",
			"aurastatcalc1": "-dm34", "Param1": "6", "Param2": "1", "Param3": "25", "Param4": "60", "manashift": "8",
			"EType": "cold", "EMin": "1", "EMax": "2", "HitShift": "8"},
		row{"skill": "Raise Skeleton", "Id": "309", "charclass": "nec", "srvstfunc": "15", "srvdofunc": "31",
			"calc1": "(lvl < 4) ? 0 : (par2 * (lvl - 3))", "Param2": "50", "summon": "necroskeleton", "pettype": "skeleton",
			"petmax": "(lvl < 4) ?lvl:(2+lvl/3)", "summode": "S1", "targetcorpse": "1", "TargetCorpse": "1",
			"aurastat1": "damagepercent", "aurastatcalc1": "((lvl < 4) ? 0 : ((lvl-3)*par3))", "Param3": "7", "manashift": "8",
			"passivestat1": "maxhp", "passivecalc1": "lvl * par2 * 256"},
		row{"skill": "Raven", "Id": "310", "srvdofunc": "114", "summon": "druidhawk", "pettype": "raven",
			"petmax": "min(lvl,par2)", "Param2": "5", "summode": "S1", "manashift": "8"},
		row{"skill": "Teleport", "Id": "311", "srvdofunc": "27", "manashift": "8"},
		row{"skill": "Fire Wall", "Id": "312", "srvdofunc": "24", "srvmissilea": "firewallmaker", "srvmissileb": "firewall",
			"LineOfSight": "4", "delay": "35", "EType": "fire", "EMin": "1", "EMax": "2", "HitShift": "8", "manashift": "8"},
		row{"skill": "Fists of Fire", "Id": "313", "srvstfunc": "23", "srvdofunc": "35", "srvmissilec": "fistsoffirefirewall",
			"aurastate": "progressive_fire", "auralencalc": "par3", "calc1": "lvl*3", "Param3": "375", "EType": "fire",
			"EMin": "4", "EMax": "6", "aurastat1": "progressive_fire", "SrcDam": "128", "HitShift": "8", "manashift": "8"},
		row{"skill": "Dragon Talon", "Id": "314", "srvstfunc": "24", "srvdofunc": "42", "calc1": "lvl/6+1", "calc2": "dm34",
			"Param3": "50", "Param4": "100", "HitShift": "8", "manashift": "8"},
		row{"skill": "Corpse Explosion", "Id": "315", "srvstfunc": "17", "srvdofunc": "55", "aurarangecalc": "ln34",
			"calc1": "par1", "calc2": "par2", "Param1": "70", "Param2": "120", "Param3": "8", "Param4": "1", "EType": "fire",
			"TargetCorpse": "1", "manashift": "8"},
		row{"skill": "Chain Lightning", "Id": "316", "srvdofunc": "26", "srvmissilea": "chainlightning",
			"aurarangecalc": "par1", "calc1": "ln34 / 5", "Param1": "20", "Param3": "26", "Param4": "1", "EType": "ltng",
			"EMin": "10", "EMax": "20", "HitShift": "8", "manashift": "8"},
		row{"skill": "Blizzard", "Id": "317", "srvdofunc": "28", "srvmissilea": "blizzardcenter", "calc1": "par1",
			"calc2": "par3", "Param1": "7", "Param3": "4", "EType": "cold", "EMin": "5", "EMax": "9", "HitShift": "8",
			"delay": "45", "manashift": "8"},
		row{"skill": "Meteor", "Id": "318", "srvdofunc": "28", "srvmissilea": "meteorcenter", "aurarangecalc": "ln12",
			"calc1": "ln12", "Param1": "6", "EType": "fire", "EMin": "30", "EMax": "50", "HitShift": "8", "manashift": "8"},
		row{"skill": "Sacrifice", "Id": "319", "srvstfunc": "29", "srvdofunc": "64", "calc1": "ln12", "calc2": "par3",
			"Param1": "180", "Param2": "15", "Param3": "8", "SrcDam": "128", "HitShift": "8", "manashift": "8"},
		row{"skill": "Frenzy", "Id": "320", "srvdofunc": "9", "aurastate": "frenzy", "auralencalc": "par7",
			"calc1": "ln12", "calc4": "3", "Param1": "90", "Param2": "5", "Param7": "150", "aurastat1": "velocitypercent",
			"aurastatcalc1": "dm34", "Param3": "20", "Param4": "200", "SrcDam": "128", "HitShift": "8", "manashift": "8"},
		row{"skill": "Guided Arrow", "Id": "321", "srvstfunc": "4", "srvdofunc": "10", "srvmissilea": "guidedarrow",
			"HitShift": "8", "manashift": "8"},
		row{"skill": "Blade Sentinel", "Id": "322", "srvdofunc": "44", "srvmissilea": "blade creeper", "summon": "bladecreeper",
			"pettype": "assassintrap", "petmax": "5", "summode": "S1", "sumskill1": "Blade Sentinel", "SrcDam": "48",
			"HitShift": "8", "manashift": "8"},
		row{"skill": "Dodge", "Id": "323", "passive": "1", "passivestat1": "passive_dodge", "passivecalc1": "dm12",
			"Param1": "10", "Param2": "65", "manashift": "8"},
		row{"skill": "Telekinesis Dummy", "Id": "324", "srvdofunc": "999"},
	)
}

func classMissiles() missileTable {
	m := missiles()
	std := func(id int, name string, vel, rng int) *d2missile.Spec {
		return &d2missile.Spec{ID: id, Name: name, SrvDoFunc: 1, Vel: vel, MaxVel: vel, Range: rng, CollideType: 3,
			CollideKill: true, CanSlow: true}
	}

	m["multipleshotarrow"] = std(200, "multipleshotarrow", 24, 50)
	m["chainlightning"] = std(201, "chainlightning", 30, 25)
	m["chainlightning"].SrvHitFunc = 12
	m["guidedarrow"] = std(202, "guidedarrow", 24, 128)
	m["firewall"] = &d2missile.Spec{ID: 203, Name: "firewall", SrvDoFunc: 5, Range: 90, CollideType: 3}
	m["fistsoffirefirewall"] = std(204, "fistsoffirefirewall", 12, 20)

	return m
}

type classFixture struct {
	*fixture
	foes []*testTarget
}

func newClassFixture(levels map[string]int) *classFixture {
	f := newFixture(levels)
	f.p.Missiles = classMissiles()
	f.sim = d2missile.NewSim(f.w, classMissiles())
	f.sim.OnEvent = func(e d2missile.Event) { f.evs = append(f.evs, e) }
	f.p.Sim = f.sim
	f.p.Walkable = func(x, y int) bool { return f.w.grid.Flags(x, y)&d2path.FlagWall == 0 }

	cf := &classFixture{fixture: f}
	f.p.Near = func(x, y, r int) []Foe {
		var out []Foe

		for _, t := range cf.foes {
			if t.alive && cheb(t.x-x, t.y-y) <= r {
				out = append(out, Foe{Target: t, X: t.x, Y: t.y})
			}
		}

		return out
	}

	return cf
}

func (cf *classFixture) addFoe(id string, x, y int) *testTarget {
	t := &testTarget{id: id, alive: true, level: 10, defense: 0, x: x, y: y, serial: len(cf.foes) + 1}
	cf.foes = append(cf.foes, t)
	cf.w.targets = append(cf.w.targets, t)

	return t
}

// castAt casts with a target unit.
func (cf *classFixture) castOn(name string, t *testTarget) (StartResult, DoResult) {
	id := cf.id(name)
	tg := Target{X: t.x, Y: t.y, Unit: t, UX: t.x, UY: t.y}
	st := cf.p.Start(cf.u, id, tg)

	if !st.OK {
		return st, DoResult{}
	}

	return st, cf.p.Do(cf.u, id, tg)
}

func effectOf(t *testing.T, r DoResult, kind string) Effect {
	t.Helper()

	for _, e := range r.Effects {
		if e.Kind == kind {
			return e
		}
	}

	t.Fatalf("no %q effect in %+v", kind, r.Effects)

	return Effect{}
}

func statOf(stats []StatMod, name string) (int, bool) {
	for _, s := range stats {
		if s.Stat == name {
			return s.Value, true
		}
	}

	return 0, false
}

func TestCurses(t *testing.T) {
	cf := newClassFixture(map[string]int{"Amplify Damage": 3, "Iron Maiden": 2, "Terror": 1})

	_, r := cf.cast("Amplify Damage", 20, 5)
	if !r.OK {
		t.Fatalf("cast failed: %s", r.Reason)
	}

	e := effectOf(t, r, "area_state")
	// ln12 = 3+1*(3-1) = 5 radius; ln34 = 200+75*(3-1) = 350 frames; damageresist -100
	if e.State != "amplifydamage" || e.Radius != 5 || e.Frames != 350 || e.Origin != "aim" || e.X != 20 || e.Y != 5 {
		t.Errorf("amplify effect %+v", e)
	}

	if v, ok := statOf(e.Stats, "damageresist"); !ok || v != -100 {
		t.Errorf("amplify stats %v", e.Stats)
	}

	// Iron Maiden: ln56 = 25 at level 2? Param5=200 Param6=25: 200+25*(2-1) = 225 percent
	_, r = cf.cast("Iron Maiden", 10, 0)
	e = effectOf(t, r, "area_state")

	if v, ok := statOf(e.Stats, d2state.StatIronMaiden); !ok || v != 225 {
		t.Errorf("iron maiden stats %v", e.Stats)
	}

	_, r = cf.cast("Terror", 10, 0)
	if e = effectOf(t, r, "area_state"); e.State != "terror" || e.Frames != 200 {
		t.Errorf("terror effect %+v", e)
	}
}

func TestDoubleSwingAndMultiHit(t *testing.T) {
	cf := newClassFixture(map[string]int{"Double Swing": 5, "Bash": 4, "Zeal": 7, "Fend": 3})
	z := cf.addFoe("z", 3, 0)
	z2 := cf.addFoe("z2", 5, 1)
	cf.addFoe("far", 40, 0)
	cf.u.roller = &seq{vals: make([]uint32, 200)}
	cf.u.ar = 100000 // never miss

	_, r := cf.castOn("Double Swing", z)
	if !r.OK || len(r.Melees) != 2 || r.Melee != r.Melees[0] {
		t.Fatalf("double swing: ok=%v melees=%d %s", r.OK, len(r.Melees), r.Reason)
	}

	// Zeal level 7: calc1 = min(2+7-1, 5) = 5 strikes
	_, r = cf.castOn("Zeal", z)
	if len(r.Melees) != 5 {
		t.Errorf("zeal strikes = %d, want 5", len(r.Melees))
	}

	// Fend hits every foe within 6 subtiles of the target once, not the far one
	_, r = cf.castOn("Fend", z)
	if len(r.Melees) != 2 {
		t.Fatalf("fend strikes = %d, want 2", len(r.Melees))
	}

	got := map[string]bool{}
	for _, m := range r.Melees {
		got[m.Target.ID()] = true
	}

	if !got["z"] || !got[z2.id] {
		t.Errorf("fend victims %v", got)
	}
}

func TestMultipleShotFan(t *testing.T) {
	cf := newClassFixture(map[string]int{"Multiple Shot": 5})
	cf.u.ranged = "bow"

	// level 5: calc1 = min(24, 2+1*4) = 6 arrows
	_, r := cf.cast("Multiple Shot", 30, 0)
	if !r.OK || len(r.Missiles) != 6 {
		t.Fatalf("ok=%v missiles=%d %s", r.OK, len(r.Missiles), r.Reason)
	}

	// symmetric fan: first and last directions mirror around the x axis
	a, b := r.Missiles[0], r.Missiles[len(r.Missiles)-1]
	if a.DY*b.DY >= 0 || abs(a.DY+b.DY) > 1e-9 {
		t.Errorf("fan not symmetric: %v %v", a.DY, b.DY)
	}
}

func abs(f float64) float64 {
	if f < 0 {
		return -f
	}

	return f
}

func TestAuras(t *testing.T) {
	cf := newClassFixture(map[string]int{"Might": 5, "Holy Freeze": 3})

	_, r := cf.cast("Might", 0, 0)
	e := effectOf(t, r, "aura")
	// ln34 = 40+10*4 = 80 percent damage on the caster
	if v, ok := statOf(e.Stats, "damagepercent"); !ok || v != 80 || e.Mode != "friendly" || e.State != "might" {
		t.Errorf("might %+v", e)
	}

	if e.Radius != 16+2*4 {
		t.Errorf("might radius %d", e.Radius)
	}

	// Holy Freeze: an enemy aura, stats go to the monsters, damage descriptor kept
	_, r = cf.cast("Holy Freeze", 0, 0)
	e = effectOf(t, r, "aura")

	if e.Mode != "enemy" || e.TargetState != "holywindcold" || len(e.Stats) != 0 {
		t.Errorf("holy freeze %+v", e)
	}

	if v, ok := statOf(e.TargetStats, "velocitypercent"); !ok || v >= 0 {
		t.Errorf("holy freeze must slow: %v", e.TargetStats)
	}
}

func TestSummons(t *testing.T) {
	cf := newClassFixture(map[string]int{"Raise Skeleton": 7, "Raven": 4, "Blade Sentinel": 3})

	// no corpse: refused at the start
	if st, _ := cf.cast("Raise Skeleton", 5, 5); st.OK || st.Reason != ReasonNoCorpse {
		t.Errorf("start without a corpse: %+v", st)
	}

	id := cf.id("Raise Skeleton")
	tg := Target{X: 5, Y: 5, Corpse: true, CX: 5, CY: 5, CorpseID: "c1", CorpseHP: 80}

	if st := cf.p.Start(cf.u, id, tg); !st.OK {
		t.Fatalf("start with corpse: %+v", st)
	}

	r := cf.p.Do(cf.u, id, tg)
	e := effectOf(t, r, "summon")

	// petmax (2+7/3) = 4, hp +50*(7-3) = 200 percent, damagepercent (7-3)*7 = 28
	if e.Summon.Key != "necroskeleton" || e.Summon.Max != 4 || e.Summon.HPPct != 200 || e.Summon.CorpseID != "c1" {
		t.Errorf("skeleton order %+v", e.Summon)
	}

	if v, ok := statOf(e.Summon.Stats, "damagepercent"); !ok || v != 28 {
		t.Errorf("skeleton stats %v", e.Summon.Stats)
	}

	// Raven fills up to min(lvl, 5) = 4 at once
	_, r = cf.cast("Raven", 5, 5)
	if e = effectOf(t, r, "summon"); e.Summon.Count != 4 || e.Summon.Kind != "minion" {
		t.Errorf("raven order %+v", e.Summon)
	}

	// sentinels are traps at the aim point with the skill's damage
	_, r = cf.cast("Blade Sentinel", 8, 3)
	if e = effectOf(t, r, "summon"); e.Summon.Kind != "trap" || e.Summon.X != 8 || e.Summon.Y != 3 || e.Summon.Desc == nil {
		t.Errorf("sentinel order %+v", e.Summon)
	}
}

func TestTeleportAndWalls(t *testing.T) {
	cf := newClassFixture(map[string]int{"Teleport": 1})
	cf.w.grid.Set(30, 0, d2path.FlagWall)

	_, r := cf.cast("Teleport", 20, 4)
	if e := effectOf(t, r, "move"); e.Mode != "teleport" || e.X != 20 || e.Y != 4 {
		t.Errorf("teleport %+v", e)
	}

	if _, r = cf.cast("Teleport", 30, 0); r.OK || r.Reason != ReasonLOS {
		t.Errorf("teleport into a wall: %+v", r)
	}
}

func TestFireWall(t *testing.T) {
	cf := newClassFixture(map[string]int{"Fire Wall": 10})
	cf.u.x, cf.u.y = 0, 0

	_, r := cf.cast("Fire Wall", 20, 0)
	if !r.OK || len(r.Missiles) < 5 {
		t.Fatalf("fire wall pieces: %d (%s)", len(r.Missiles), r.Reason)
	}

	for _, m := range r.Missiles {
		if m.Velocity != 0 || m.HitEvery != 5 {
			t.Fatalf("wall piece not stationary/periodic: %+v", m)
		}
	}

	// pieces are spread along y (perpendicular to the cast direction)
	ys := map[int]bool{}
	for _, m := range r.Missiles {
		ys[int(m.Y)] = true
	}

	if len(ys) < len(r.Missiles)/2 {
		t.Errorf("wall pieces not spread: %v", ys)
	}
}

func TestChargesAndFinisher(t *testing.T) {
	cf := newClassFixture(map[string]int{"Fists of Fire": 4, "Dragon Talon": 12})
	z := cf.addFoe("z", 2, 0)
	cf.u.roller = &seq{vals: make([]uint32, 100)}
	cf.u.ar = 100000

	_, r := cf.castOn("Fists of Fire", z)
	e := effectOf(t, r, "self_state")

	if e.State != "progressive_fire" || e.Stack != 3 || e.Frames != 375 {
		t.Errorf("charge effect %+v", e)
	}

	if r.Melee == nil || r.Melee.Damage.Fire == 0 {
		t.Errorf("fists of fire must add fire damage: %+v", r.Melee)
	}

	// the caster now holds 2 charges: Dragon Talon kicks lvl/6+1 = 3 times
	// and releases 2 firewall missiles, clearing the charges
	cf.u.stats["progressive_fire"] = 2
	cf.u.levels["Fists of Fire"] = 4

	_, r = cf.castOn("Dragon Talon", z)
	if len(r.Melees) != 3 {
		t.Errorf("talon kicks = %d", len(r.Melees))
	}

	if len(r.Missiles) != 2 {
		t.Errorf("released missiles = %d, want 2", len(r.Missiles))
	}

	if e := effectOf(t, r, "clear_state"); e.State != "progressive_fire" {
		t.Errorf("clear %+v", e)
	}
}

func TestCorpseExplosion(t *testing.T) {
	cf := newClassFixture(map[string]int{"Corpse Explosion": 5})
	cf.u.roller = &seq{vals: []uint32{0}}
	id := cf.id("Corpse Explosion")
	tg := Target{X: 9, Y: 9, Corpse: true, CX: 9, CY: 9, CorpseID: "c", CorpseHP: 100}
	cf.p.Start(cf.u, id, tg)

	r := cf.p.Do(cf.u, id, tg)
	e := effectOf(t, r, "area_hit")

	// the roll picks the low end: 70 percent of 100 life = 70 fire
	if e.Desc == nil || e.Desc.Fire.Min != 70<<8 || e.CorpseID != "c" || e.Radius != 8+4 {
		t.Errorf("corpse explosion %+v desc=%+v", e, e.Desc)
	}
}

func TestChainLightningJumps(t *testing.T) {
	cf := newClassFixture(map[string]int{"Chain Lightning": 10})
	cf.u.x, cf.u.y = 0, 0
	cf.u.ar = 100000
	cf.u.roller = &seq{vals: make([]uint32, 500)}

	a := cf.addFoe("a", 8, 0)
	cf.addFoe("b", 8, 10)
	cf.addFoe("c", 18, 10)

	_, r := cf.castOn("Chain Lightning", a)
	if !r.OK || len(r.Missiles) != 1 {
		t.Fatalf("cast: %+v", r)
	}

	for i := 0; i < 60; i++ {
		cf.w.frame++
		cf.sim.Step()
	}

	hit := map[string]bool{}

	for _, e := range cf.evs {
		if e.Kind == d2missile.EventHit {
			hit[e.Target.ID()] = true
		}
	}

	if !hit["a"] || !hit["b"] || !hit["c"] {
		t.Errorf("the bolt must chain a -> b -> c, hit %v", hit)
	}
}

// The chain follows unit ids (next higher, wrapping), not distance, and the
// first bolt counts: calc1 bolts in all (VERIFIED in 0x5a81c0 / 0x5c8320).
func TestChainLightningFollowsUnitIDs(t *testing.T) {
	cf := newClassFixture(map[string]int{"Chain Lightning": 1})
	cf.u.x, cf.u.y = 0, 0
	cf.u.ar = 100000
	cf.u.roller = &seq{vals: make([]uint32, 500)}

	// ids: far=1 is hit first; the next higher id is "high"(3) although "near"(2)... both near
	// enough; a bolt to the lowest id must only happen after wrapping
	a := cf.addFoe("a", 8, 0)
	b := cf.addFoe("b", 8, 6)
	c := cf.addFoe("c", 8, 3)
	a.serial, b.serial, c.serial = 2, 9, 5

	_, r := cf.castOn("Chain Lightning", a)
	if !r.OK {
		t.Fatalf("cast: %+v", r)
	}

	order := []string{}

	for i := 0; i < 80; i++ {
		cf.w.frame++
		cf.sim.Step()
	}

	for _, e := range cf.evs {
		if e.Kind == d2missile.EventHit && (len(order) == 0 || order[len(order)-1] != e.Target.ID()) {
			order = append(order, e.Target.ID())
		}
	}

	if len(order) != 3 || order[0] != "a" || order[1] != "c" || order[2] != "b" {
		t.Errorf("hit order %v, want a then c (next higher unit id 5, not the nearest)", order)
	}
}

func TestRainsAndStorms(t *testing.T) {
	cf := newClassFixture(map[string]int{"Blizzard": 5, "Meteor": 5})
	cf.u.roller = &seq{vals: make([]uint32, 400)}

	_, r := cf.cast("Blizzard", 12, 12)
	e := effectOf(t, r, "strikes")

	// 100 frames, one shard every calc2 = 4 frames
	if len(e.Strikes) != 25 || e.Desc == nil || e.Desc.Cold.Max == 0 {
		t.Errorf("blizzard strikes %d desc=%+v", len(e.Strikes), e.Desc)
	}

	for _, s := range e.Strikes {
		if cheb(s.X-12, s.Y-12) > 8 {
			t.Errorf("shard outside the radius: %+v", s)
		}
	}

	_, r = cf.cast("Meteor", 12, 12)
	e = effectOf(t, r, "strikes")

	if len(e.Strikes) != 1 || e.Strikes[0].Delay != 12 || e.Strikes[0].Radius != 6 || e.Desc.Fire.Max == 0 {
		t.Errorf("meteor %+v", e)
	}
}

func TestSacrificeAndFrenzy(t *testing.T) {
	cf := newClassFixture(map[string]int{"Sacrifice": 3, "Frenzy": 2})
	z := cf.addFoe("z", 2, 0)
	cf.u.roller = &seq{vals: make([]uint32, 100)}
	cf.u.ar = 100000

	_, r := cf.castOn("Sacrifice", z)
	if e := effectOf(t, r, "self_damage"); e.SelfDamagePct != 8 {
		t.Errorf("sacrifice %+v", e)
	}

	_, r = cf.castOn("Frenzy", z)
	e := effectOf(t, r, "self_state")

	if e.State != "frenzy" || e.Frames != 150 || e.Stack != 3 {
		t.Errorf("frenzy %+v", e)
	}
}

func TestGuidedArrowHomes(t *testing.T) {
	cf := newClassFixture(map[string]int{"Guided Arrow": 1})
	z := cf.addFoe("z", 10, 0)

	_, r := cf.castOn("Guided Arrow", z)
	if !r.OK || len(r.Missiles) != 1 || r.Missiles[0].Home == nil {
		t.Fatalf("homing missile: %+v", r)
	}
}

func TestSummonPassives(t *testing.T) {
	cf := newClassFixture(map[string]int{"Raise Skeleton": 7})
	id := cf.id("Raise Skeleton")

	// the skill is not a passive: PassiveStats stays empty, SummonPassives reads
	// the passivestat columns (fixture calc: lvl * par2 * 256 = 7*50*256)
	if got := cf.p.PassiveStats(cf.u, id); got != nil {
		t.Errorf("PassiveStats of a summon: %+v", got)
	}

	got := cf.p.SummonPassives(cf.u, id)
	if len(got) != 1 || got[0] != (StatMod{"maxhp", 7 * 50 * 256}) {
		t.Errorf("SummonPassives %+v", got)
	}

	if cf.p.SummonPassives(cf.u, 99999) != nil {
		t.Error("unknown skill")
	}

	cf0 := newClassFixture(map[string]int{})
	if cf0.p.SummonPassives(cf0.u, cf0.id("Raise Skeleton")) != nil {
		t.Error("level 0")
	}
}

func TestPassivesAndTable(t *testing.T) {
	cf := newClassFixture(map[string]int{"Dodge": 10})

	// Dodge passive: dm12 (10, 65) at level 10 = ((110*10)*(65-10))/(100*16)+10 = 47
	st := cf.p.PassiveStats(cf.u, cf.id("Dodge"))
	if len(st) != 1 || st[0].Stat != "passive_dodge" || st[0].Value != 47 {
		t.Errorf("dodge stats %+v", st)
	}

	if Implemented(cf.reg.ByName("Dodge")) {
		t.Error("a passive is not castable")
	}

	if Implemented(cf.reg.ByName("Telekinesis Dummy")) {
		t.Error("unknown do functions must not be implemented")
	}

	for _, n := range []string{"Amplify Damage", "Double Swing", "Might", "Raise Skeleton", "Teleport", "Fire Wall",
		"Fists of Fire", "Dragon Talon", "Corpse Explosion", "Chain Lightning", "Blizzard", "Sacrifice", "Frenzy"} {
		if !Implemented(cf.reg.ByName(n)) {
			t.Errorf("%s must be implemented", n)
		}
	}
}

// SubPos lets the chain and homing code find the test targets.
func (t *testTarget) SubPos() (float64, float64) { return float64(t.x) + 0.5, float64(t.y) + 0.5 }
