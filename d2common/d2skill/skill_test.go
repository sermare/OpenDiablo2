package d2skill

import (
	"strconv"
	"strings"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2calc"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2combat"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2missile"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2path"
)

// ---- skills.txt rows (columns as in patch_d2 skills.txt) ----

type row map[string]string

func (r row) num(k string) int {
	n, _ := strconv.Atoi(r[k])
	return n
}

func (r row) calc(k string) *d2calc.Program { return d2calc.Compile(r[k], d2calc.KindSkill) }

// skillFromRow mirrors the d2records conversion for the columns the tests use.
func skillFromRow(r row) *Skill {
	s := &Skill{
		ID: r.num("Id"), Name: r["skill"],
		InTown: r.num("InTown") > 0, UseManaOnDo: r.num("usemanaondo") > 0, DecQuant: r.num("decquant") > 0,
		Lob: r.num("lob") > 0, Passive: r.num("passive") > 0, Kick: r.num("Kick") > 0, NoAmmo: r.num("noammo") > 0,
		SrvStFunc: r.num("srvstfunc"), SrvDoFunc: r.num("srvdofunc"),
		SrvMissile: r["srvmissile"], SrvMissileA: r["srvmissilea"], LineOfSight: r.num("LineOfSight"),
		MinMana: r.num("minmana"), ManaShift: r.num("manashift"), Mana: r.num("mana"), LvlMana: r.num("lvlmana"),
		Delay: r.calc("delay"), ToHit: r.num("ToHit"), LevToHit: r.num("LevToHit"), ToHitCalc: r.calc("ToHitCalc"),
		HitClass: r.num("HitClass"), AuraFilter: r.num("aurafilter"), AuraState: r["aurastate"],
		AuraTargetState: r["auratargetstate"], AuraLenCalc: r.calc("auralencalc"), AuraRangeCalc: r.calc("aurarangecalc"),
		PassiveState: r["passivestate"],
	}
	s.SrvMissileB, s.SrvMissileC = r["srvmissileb"], r["srvmissilec"]
	s.CharClass, s.Summon, s.PetType, s.SumMode = r["charclass"], r["summon"], r["pettype"], r["summode"]
	s.SumSkill[1] = r["sumskill1"]
	s.TargetCorpse, s.Aura, s.PetMax = r.num("TargetCorpse") > 0, r.num("aura") > 0, r.calc("petmax")

	for i := 1; i <= 8; i++ {
		s.Params[i] = r.num("Param" + strconv.Itoa(i))
	}

	for i := 1; i <= 4; i++ {
		s.Calc[i] = r.calc("calc" + strconv.Itoa(i))
	}

	for i := 1; i <= 6; i++ {
		s.AuraStat[i] = r["aurastat"+strconv.Itoa(i)]
		s.AuraStatCalc[i] = r.calc("aurastatcalc" + strconv.Itoa(i))
	}

	for i := 1; i <= 5; i++ {
		s.PassiveStat[i] = r["passivestat"+strconv.Itoa(i)]
		s.PassiveCalc[i] = r.calc("passivecalc" + strconv.Itoa(i))
	}

	s.HitShift, s.SrcDam, s.MinDam, s.MaxDam = r.num("HitShift"), r.num("SrcDam"), r.num("MinDam"), r.num("MaxDam")
	s.EType, s.EMin, s.EMax, s.ELen = r["EType"], r.num("EMin"), r.num("EMax"), r.num("ELen")

	for i := 0; i < 5; i++ {
		n := strconv.Itoa(i + 1)
		s.MinLevDam[i], s.MaxLevDam[i] = r.num("MinLevDam"+n), r.num("MaxLevDam"+n)
		s.EMinLev[i], s.EMaxLev[i] = r.num("EMinLev"+n), r.num("EMaxLev"+n)
	}

	for i := 0; i < 3; i++ {
		s.ELevLen[i] = r.num("ELevLen" + strconv.Itoa(i+1))
	}

	s.DmgSymPer, s.EDmgSymPer, s.ELenSymPer = r.calc("DmgSymPerCalc"), r.calc("EDmgSymPerCalc"), r.calc("ELenSymPerCalc")

	return s
}

var rows = []row{
	{"skill": "Attack", "Id": "0", "srvstfunc": "1", "srvdofunc": "1", "HitShift": "8", "SrcDam": "128", "manashift": "8"},
	{"skill": "Kick", "Id": "1", "srvstfunc": "2", "srvdofunc": "2", "Kick": "1", "HitShift": "8", "manashift": "8"},
	{"skill": "Throw", "Id": "2", "srvstfunc": "65", "srvdofunc": "3", "HitShift": "8", "SrcDam": "128", "manashift": "8"},
	{"skill": "Magic Arrow", "Id": "6", "srvmissile": "magicarrow", "noammo": "1", "minmana": "0", "manashift": "5",
		"mana": "12", "lvlmana": "-1", "ToHit": "10", "LevToHit": "9", "HitShift": "8", "SrcDam": "128", "MinDam": "1",
		"MinLevDam1": "1", "MinLevDam2": "1", "MinLevDam3": "1", "MinLevDam4": "1", "MinLevDam5": "1", "MaxDam": "1",
		"MaxLevDam1": "1", "MaxLevDam2": "1", "MaxLevDam3": "1", "MaxLevDam4": "1", "MaxLevDam5": "1"},
	{"skill": "Fire Arrow", "Id": "7", "srvstfunc": "4", "srvmissile": "firearrow", "decquant": "1", "minmana": "1",
		"manashift": "5", "mana": "24", "lvlmana": "1", "Param8": "12", "ToHit": "10", "LevToHit": "9", "HitShift": "8",
		"SrcDam": "128", "EType": "fire", "EMin": "1", "EMinLev1": "2", "EMinLev2": "3", "EMinLev3": "6", "EMinLev4": "12",
		"EMinLev5": "24", "EMax": "4", "EMaxLev1": "2", "EMaxLev2": "3", "EMaxLev3": "7", "EMaxLev4": "14", "EMaxLev5": "27",
		"EDmgSymPerCalc": "(skill('Exploding Arrow'.blvl)) * par8"},
	{"skill": "Inner Sight", "Id": "8", "srvdofunc": "6", "aurafilter": "34179", "auratargetstate": "innersight",
		"auralencalc": "ln34", "aurarangecalc": "ln56", "aurastat1": "armorclass", "aurastatcalc1": "-edmn", "LineOfSight": "4",
		"minmana": "1", "manashift": "7", "mana": "10", "Param1": "40", "Param2": "25", "Param3": "200", "Param4": "100",
		"Param5": "20", "Param6": "0", "HitShift": "8", "EMin": "40", "EMinLev1": "25", "EMinLev2": "45", "EMinLev3": "60",
		"EMinLev4": "80", "EMinLev5": "100"},
	{"skill": "Jab", "Id": "10", "srvstfunc": "5", "srvdofunc": "7", "minmana": "1", "manashift": "6", "mana": "8",
		"lvlmana": "1", "calc1": "ln34", "Param3": "-15", "Param4": "3", "ToHit": "10", "LevToHit": "9", "HitShift": "8", "SrcDam": "128"},
	{"skill": "Fire Bolt", "Id": "36", "srvmissile": "firebolt", "minmana": "1", "manashift": "7", "mana": "5", "Param8": "16",
		"HitShift": "7", "EType": "fire", "EMin": "6", "EMinLev1": "3", "EMinLev2": "4", "EMinLev3": "8", "EMinLev4": "18",
		"EMinLev5": "54", "EMax": "12", "EMaxLev1": "3", "EMaxLev2": "6", "EMaxLev3": "10", "EMaxLev4": "20", "EMaxLev5": "56",
		"EDmgSymPerCalc": "(skill('Fire Ball'.blvl)+skill('Meteor'.blvl))*par8"},
	{"skill": "Warmth", "Id": "37", "passivestate": "warmth", "passivestat1": "manarecoverybonus", "passivecalc1": "ln12",
		"InTown": "1", "passive": "1", "Param1": "30", "Param2": "12", "manashift": "8"},
	{"skill": "Charged Bolt", "Id": "38", "srvdofunc": "17", "srvmissilea": "chargedbolt", "minmana": "1", "manashift": "5",
		"mana": "24", "lvlmana": "4", "calc1": "min(24,ln12)", "Param1": "3", "Param2": "1", "Param8": "6", "HitShift": "7",
		"EType": "ltng", "EMin": "4", "EMinLev1": "1", "EMinLev2": "1", "EMinLev3": "2", "EMinLev4": "3", "EMinLev5": "4",
		"EMax": "8", "EMaxLev1": "1", "EMaxLev2": "1", "EMaxLev3": "2", "EMaxLev4": "3", "EMaxLev5": "4",
		"EDmgSymPerCalc": "(skill('Lightning'.blvl))*par8"},
	{"skill": "Ice Bolt", "Id": "39", "srvmissile": "icebolt", "minmana": "1", "manashift": "8", "mana": "3", "Param8": "15",
		"HitShift": "7", "EType": "cold", "EMin": "6", "EMinLev1": "2", "EMinLev2": "4", "EMinLev3": "6", "EMinLev4": "8",
		"EMinLev5": "10", "EMax": "10", "EMaxLev1": "3", "EMaxLev2": "5", "EMaxLev3": "7", "EMaxLev4": "9", "EMaxLev5": "11",
		"ELen": "150", "ELevLen1": "35", "ELevLen2": "35", "ELevLen3": "35",
		"EDmgSymPerCalc": "(skill('Frost Nova'.blvl)+skill('Ice Blast'.blvl)+skill('Glacial Spike'.blvl)+skill('Blizzard'.blvl)+skill('Frozen Orb'.blvl))*par8"},
	{"skill": "Frozen Armor", "Id": "40", "srvdofunc": "18", "aurastate": "frozenarmor", "InTown": "1",
		"auralencalc": "ln34+(skill('Shiver Armor'.blvl)+skill('Chilling Armor'.blvl))*par7", "aurastat1": "skill_armor_percent",
		"aurastatcalc1": "ln12", "minmana": "1", "manashift": "8", "mana": "7",
		"calc1": "ln56*(100+((skill('Shiver Armor'.blvl)+skill('Chilling Armor'.blvl))*par8))/100", "Param1": "30",
		"Param2": "5", "Param3": "3000", "Param4": "300", "Param5": "30", "Param6": "3", "Param7": "250", "Param8": "5"},
	{"skill": "Static Field", "Id": "42", "srvdofunc": "20", "aurafilter": "34691", "aurarangecalc": "ln12",
		"LineOfSight": "4", "minmana": "1", "manashift": "8", "mana": "9", "calc1": "par4", "calc2": "par3", "Param1": "5",
		"Param2": "1", "Param3": "0", "Param4": "25", "HitShift": "8", "EType": "ltng"},
	{"skill": "Frost Nova", "Id": "44", "srvdofunc": "22", "srvmissilea": "frostnova", "minmana": "1", "manashift": "8",
		"mana": "9", "lvlmana": "1", "Param1": "9", "Param2": "3", "Param8": "10", "HitShift": "7", "EType": "cold",
		"EMin": "4", "EMinLev1": "4", "EMinLev2": "6", "EMinLev3": "8", "EMinLev4": "10", "EMinLev5": "12", "EMax": "8",
		"EMaxLev1": "5", "EMaxLev2": "7", "EMaxLev3": "9", "EMaxLev4": "11", "EMaxLev5": "13", "ELen": "200",
		"ELevLen1": "25", "ELevLen2": "25", "ELevLen3": "25",
		"EDmgSymPerCalc": "(skill('Blizzard'.blvl)+skill('Frozen Orb'.blvl))*par8"},
	{"skill": "Bash", "Id": "126", "srvstfunc": "32", "srvdofunc": "2", "minmana": "1", "manashift": "8", "mana": "2",
		"calc1": "ln12+skill('Stun'.blvl)*par8", "calc2": "ln34", "Param1": "50", "Param2": "5", "Param3": "1", "Param4": "1",
		"Param7": "5", "Param8": "5", "ToHitCalc": "15+lvl*5+skill('Concentrate'.blvl)*par7", "HitClass": "112",
		"HitShift": "8", "SrcDam": "128"},
	{"skill": "Howl", "Id": "130", "srvdofunc": "22", "srvmissilea": "howl", "auratargetstate": "terror", "minmana": "1",
		"manashift": "8", "mana": "4", "calc1": "par1 * (lvl-1)", "Param1": "2", "Param2": "1", "Param3": "24", "Param4": "5",
		"Param5": "75", "Param6": "25", "HitShift": "8"},
	{"skill": "Stun", "Id": "139", "srvstfunc": "32", "srvdofunc": "2", "minmana": "1", "manashift": "8", "mana": "2",
		"calc1": "skill('Bash'.blvl)*par8", "Param8": "8", "ToHitCalc": "10+lvl*5+skill('Concentrate'.blvl)*par7",
		"HitClass": "96", "HitShift": "8", "SrcDam": "128", "EType": "stun", "ELen": "30", "ELevLen1": "5", "ELevLen2": "5",
		"ELevLen3": "2"},
	// skills that only exist so synergy calcs can find them
	{"skill": "Fire Ball", "Id": "950"}, {"skill": "Meteor", "Id": "951"}, {"skill": "Lightning", "Id": "952"},
	{"skill": "Ice Blast", "Id": "953"}, {"skill": "Glacial Spike", "Id": "954"}, {"skill": "Blizzard", "Id": "955"},
	{"skill": "Frozen Orb", "Id": "956"}, {"skill": "Shiver Armor", "Id": "957"}, {"skill": "Chilling Armor", "Id": "958"},
	{"skill": "Exploding Arrow", "Id": "959"}, {"skill": "Concentrate", "Id": "960"},
	{"skill": "Test Delay", "Id": "900", "srvmissile": "firebolt", "manashift": "8", "delay": "35"},
	{"skill": "Test Late", "Id": "901", "srvstfunc": "9", "srvmissile": "firebolt", "manashift": "8", "mana": "10",
		"usemanaondo": "1"},
	{"skill": "Test Start", "Id": "902", "srvstfunc": "9", "srvmissile": "firebolt", "manashift": "8", "mana": "10"},
}

func registry() *Registry {
	r := NewRegistry()
	for _, rw := range rows {
		r.Add(skillFromRow(rw))
	}

	return r
}

// ---- test unit ----

type testUnit struct {
	id        string
	player    bool
	level     int
	levels    map[string]int // skill name -> effective level
	base      map[string]int
	reg       *Registry
	stats     map[string]int
	mana      int
	x, y      int
	town      bool
	roller    d2combat.Roller
	ar        int
	wmin      int
	wmax      int
	ranged    string
	thrown    string
	ammo      int
	cooldowns map[int]int
}

type seq struct{ vals []uint32 }

func (s *seq) Roll(n int32) uint32 {
	v := uint32(0)
	if len(s.vals) > 0 {
		v, s.vals = s.vals[0], s.vals[1:]
	}

	if v >= uint32(n) {
		v = uint32(n) - 1
	}

	return v
}

func newUnit(reg *Registry, levels map[string]int) *testUnit {
	return &testUnit{id: "hero", player: true, level: 20, levels: levels, base: levels, reg: reg, mana: 100 << 8,
		roller: &seq{}, ar: 500, wmin: 5, wmax: 9, ammo: 1, cooldowns: map[int]int{}, stats: map[string]int{}}
}

func (u *testUnit) ID() string     { return u.id }
func (u *testUnit) IsPlayer() bool { return u.player }
func (u *testUnit) Level() int     { return u.level }
func (u *testUnit) SkillLevel(id int) int {
	return u.levels[u.reg.ByID(id).Name]
}
func (u *testUnit) BaseSkillLevel(id int) int   { return u.base[u.reg.ByID(id).Name] }
func (u *testUnit) Stat(n string) int           { return u.stats[n] }
func (u *testUnit) Mana() int                   { return u.mana }
func (u *testUnit) SetMana(v int)               { u.mana = v }
func (u *testUnit) Pos() (int, int)             { return u.x, u.y }
func (u *testUnit) InTown() bool                { return u.town }
func (u *testUnit) Roller() d2combat.Roller     { return u.roller }
func (u *testUnit) AttackRating() int           { return u.ar }
func (u *testUnit) WeaponDamage() (int, int)    { return u.wmin, u.wmax }
func (u *testUnit) RangedWeaponMissile() string { return u.ranged }
func (u *testUnit) ThrownMissile() string       { return u.thrown }
func (u *testUnit) HasAmmo() bool               { return u.ammo > 0 }
func (u *testUnit) Cooldown(id int) int         { return u.cooldowns[id] }
func (u *testUnit) SetCooldown(id, until int)   { u.cooldowns[id] = until }
func (u *testUnit) ConsumeAmmo() bool {
	if u.ammo > 0 {
		u.ammo--
		return true
	}

	return false
}

// ---- world ----

type testTarget struct {
	id      string
	alive   bool
	level   int
	defense int
	x, y    int
}

func (t *testTarget) ID() string       { return t.id }
func (t *testTarget) IsPlayer() bool   { return false }
func (t *testTarget) Alive() bool      { return t.alive }
func (t *testTarget) Level() int       { return t.level }
func (t *testTarget) Defense(bool) int { return t.defense }

type testWorld struct {
	grid    *d2path.CellGrid
	targets []*testTarget
	frame   int
}

func (w *testWorld) Flags(x, y int) uint16                          { return w.grid.Flags(x, y) }
func (w *testWorld) Frame() int                                     { return w.frame }
func (w *testWorld) IsEnemy(d2missile.Owner, d2missile.Target) bool { return true }
func (w *testWorld) Targets(x, y int) []d2missile.Target {
	var out []d2missile.Target

	for _, t := range w.targets {
		if t.alive && x >= t.x-1 && x <= t.x+1 && y >= t.y-1 && y <= t.y+1 {
			out = append(out, t)
		}
	}

	return out
}

type missileTable map[string]*d2missile.Spec

func (m missileTable) ByID(id int) *d2missile.Spec {
	for _, s := range m {
		if s.ID == id {
			return s
		}
	}

	return nil
}
func (m missileTable) ByName(n string) *d2missile.Spec { return m[strings.ToLower(n)] }

func missiles() missileTable {
	std := func(id int, name string, vel, rng int) *d2missile.Spec {
		return &d2missile.Spec{ID: id, Name: name, SrvDoFunc: 1, Vel: vel, MaxVel: vel, Range: rng, CollideType: 3,
			CollideKill: true, LastCollide: true, CanSlow: true, ExplosionMissile: "x"}
	}

	arrow := std(12, "firearrow", 24, 40)
	arrow.ToHit, arrow.Pierce = true, true

	return missileTable{
		"firebolt":    std(58, "firebolt", 20, 50),
		"icebolt":     std(59, "icebolt", 12, 50),
		"chargedbolt": std(56, "chargedbolt", 12, 98),
		"magicarrow":  std(27, "magicarrow", 24, 40),
		"firearrow":   arrow,
		"howl": {ID: 148, Name: "howl", SrvDoFunc: 1, Vel: 12, MaxVel: 12, Accel: -1000, Range: 12, CollideType: 3,
			LastCollide: true},
		"frostnova": {ID: 70, Name: "frostnova", SrvDoFunc: 1, Vel: 12, Range: 12, CollideType: 3},
	}
}

type fixture struct {
	reg   *Registry
	w     *testWorld
	sim   *d2missile.Sim
	p     *Pipeline
	u     *testUnit
	evs   []d2missile.Event
	state []string
}

func newFixture(levels map[string]int) *fixture {
	f := &fixture{reg: registry(), w: &testWorld{grid: d2path.NewCellGrid(-50, -50, 200, 100)}}
	f.u = newUnit(f.reg, levels)
	f.sim = d2missile.NewSim(f.w, missiles())
	f.sim.OnEvent = func(e d2missile.Event) { f.evs = append(f.evs, e) }
	f.p = &Pipeline{Skills: f.reg, Missiles: missiles(), Sim: f.sim, Grid: f.w.grid, Frame: func() int { return f.w.frame },
		Opt: Options{StaticFieldMinPct: 25}}
	f.p.ApplyState = func(_ Unit, t d2missile.Target, st string, frames int) {
		f.state = append(f.state, t.ID()+":"+st+":"+strconv.Itoa(frames))
	}

	return f
}

func (f *fixture) id(name string) int { return f.reg.ByName(name).ID }

func (f *fixture) cast(name string, tx, ty int) (StartResult, DoResult) {
	id := f.id(name)
	tg := Target{X: tx, Y: ty}
	st := f.p.Start(f.u, id, tg)

	if !st.OK {
		return st, DoResult{}
	}

	return st, f.p.Do(f.u, id, tg)
}

func (f *fixture) env(name string, lvl int) *Env {
	return NewEnv(f.reg.ByName(name), lvl, f.u, f.reg)
}

// ---- tests ----

func TestElementalDamagePerLevel(t *testing.T) {
	f := newFixture(map[string]int{})

	tests := []struct {
		skill          string
		lvl            int
		blvl           map[string]int
		min, max, elen int // 8.8 fixed (HitShift applied), frames
	}{
		// Fire Bolt 3-6 at level 1 (EMin 6 << 7 = 3.0), the in-game tooltip value
		{"Fire Bolt", 1, nil, 768, 1536, 0},
		// level 10: min 6+7*3+2*4 = 35 -> 17.5, max 12+7*3+2*6 = 45 -> 22.5
		{"Fire Bolt", 10, nil, 4480, 5760, 0},
		// level 20: min 6+21+32+4*8 = 91 -> 45.5, max 12+21+48+4*10 = 121 -> 60.5
		{"Fire Bolt", 20, nil, 11648, 15488, 0},
		// synergy: (Fire Ball 10 + Meteor 5) * 16 = +240% of the shifted base
		{"Fire Bolt", 10, map[string]int{"Fire Ball": 10, "Meteor": 5}, 4480 + 4480*240/100, 5760 + 5760*240/100, 0},
		// Frost Nova 2-4 at level 1 and 22-28.5 at level 10, chill 200 + 7*25+2*25
		{"Frost Nova", 1, nil, 512, 1024, 200},
		{"Frost Nova", 10, nil, 5632, 7296, 425},
		// Charged Bolt per bolt: 4-6 at level 5
		{"Charged Bolt", 5, nil, 1024, 1536, 0},
		// Ice Bolt level 3: min 6+2*2 = 10 -> 5.0; max 10+2*3 = 16 -> 8.0; length 150+2*35
		{"Ice Bolt", 3, nil, 1280, 2048, 220},
		// Fire Arrow level 2 (HitShift 8): min 1+2 = 3 -> 3.0, max 4+2 = 6
		{"Fire Arrow", 2, nil, 768, 1536, 0},
	}

	for _, tc := range tests {
		for k := range f.u.base {
			delete(f.u.base, k)
		}

		for k, v := range tc.blvl {
			f.u.base[k] = v
		}

		e := f.env(tc.skill, tc.lvl)
		sk := f.reg.ByName(tc.skill)

		if got := int(sk.ElemMin(e, tc.lvl)); got != tc.min {
			t.Errorf("%s L%d min = %d, want %d", tc.skill, tc.lvl, got, tc.min)
		}

		if got := int(sk.ElemMax(e, tc.lvl)); got != tc.max {
			t.Errorf("%s L%d max = %d, want %d", tc.skill, tc.lvl, got, tc.max)
		}

		if got := sk.ElemLen(e, tc.lvl); got != tc.elen {
			t.Errorf("%s L%d len = %d, want %d", tc.skill, tc.lvl, got, tc.elen)
		}
	}
}

func TestTierSums(t *testing.T) {
	l := [5]int{1, 10, 100, 1000, 10000}
	for lvl, want := range map[int]int{0: 0, 1: 0, 2: 1, 8: 7, 9: 17, 16: 87, 17: 187, 22: 687, 23: 1687, 28: 6687, 29: 16687, 30: 26687} {
		if got := tiers5(l, lvl); got != want {
			t.Errorf("tiers5 L%d = %d, want %d", lvl, got, want)
		}
	}
}

func TestManaCosts(t *testing.T) {
	f := newFixture(map[string]int{})
	for _, tc := range []struct {
		skill string
		lvl   int
		want  int // 8.8
	}{
		{"Fire Bolt", 1, 640},     // 5<<7 = 2.5 mana
		{"Fire Bolt", 20, 640},    // lvlmana 0
		{"Ice Bolt", 1, 3 * 256},  // 3<<8
		{"Charged Bolt", 1, 768},  // 24<<5 = 3.0
		{"Charged Bolt", 5, 1280}, // 40<<5 = 5.0
		{"Frost Nova", 10, 18 * 256},
		{"Magic Arrow", 1, 384},  // 12<<5 = 1.5
		{"Magic Arrow", 9, 128},  // 4<<5 = 0.5
		{"Magic Arrow", 20, 0},   // negative cost floors at minmana (0)
		{"Fire Arrow", 1, 768},   // 24<<5 = 3.0
		{"Inner Sight", 1, 1280}, // 10<<7 = 5.0
		{"Jab", 1, 512},          // 8<<6 = 2.0
		{"Jab", 5, 768},          // 12<<6 = 3.0
		{"Attack", 1, 0},         // free
	} {
		if got := f.reg.ByName(tc.skill).ManaCost(tc.lvl); got != tc.want {
			t.Errorf("%s L%d mana = %d, want %d", tc.skill, tc.lvl, got, tc.want)
		}
	}
}

func TestFireBoltCastPipeline(t *testing.T) {
	f := newFixture(map[string]int{"Fire Bolt": 10})
	f.u.base = map[string]int{"Fire Bolt": 10}

	// no srvst: mana is paid at the do frame, not at the start
	st := f.p.Start(f.u, f.id("Fire Bolt"), Target{X: 20, Y: 0})
	if !st.OK || st.ManaPaid != 0 || f.u.mana != 100<<8 {
		t.Fatalf("start %+v mana %d", st, f.u.mana)
	}

	do := f.p.Do(f.u, f.id("Fire Bolt"), Target{X: 20, Y: 0})
	if !do.OK || len(do.Missiles) != 1 || do.ManaPaid != 640 || f.u.mana != 100<<8-640 {
		t.Fatalf("do %+v mana %d", do, f.u.mana)
	}

	m := do.Missiles[0]
	if m.Spec.Name != "firebolt" || m.Level != 10 || m.SkillID != 36 || m.Velocity != 20<<8 || m.Life != 50 {
		t.Fatalf("missile %+v", m)
	}

	// damage descriptor: level 10 fire 17.5-22.5, no physical (SrcDam 0)
	if m.Damage.Fire.Min != 4480 || m.Damage.Fire.Max != 5760 || m.Damage.PhysMax != 0 {
		t.Fatalf("descriptor %+v", m.Damage)
	}
}

func TestFireBoltKillsMonsterInPath(t *testing.T) {
	f := newFixture(map[string]int{"Fire Bolt": 1})
	f.u.base = map[string]int{"Fire Bolt": 1}
	f.w.targets = []*testTarget{{id: "zombie", alive: true, level: 1, x: 12, y: 0}}

	_, do := f.cast("Fire Bolt", 30, 0)
	if !do.OK {
		t.Fatalf("%+v", do)
	}

	for i := 0; i < 30; i++ {
		f.w.frame++
		f.sim.Step()
	}

	hits := 0

	for _, e := range f.evs {
		if e.Kind == d2missile.EventHit {
			hits++

			// level 1 Fire Bolt: 3.0 - 6.0 fire, roller 0 -> min
			if e.Damage.Fire != 768 || e.Damage.SumTotal(true) != 768 {
				t.Fatalf("damage %+v", e.Damage)
			}
		}
	}

	if hits != 1 {
		t.Fatalf("hits = %d (events %d)", hits, len(f.evs))
	}
}

func TestStartRefusals(t *testing.T) {
	f := newFixture(map[string]int{"Fire Bolt": 1, "Warmth": 3, "Static Field": 1, "Fire Arrow": 1, "Test Delay": 1})
	f.u.base = f.u.levels
	fb := f.id("Fire Bolt")

	// unknown/level 0
	if st := f.p.Start(f.u, f.id("Ice Bolt"), Target{}); st.OK || st.Reason != ReasonNoSkill {
		t.Fatalf("%+v", st)
	}

	// town
	f.u.town = true
	if st := f.p.Start(f.u, fb, Target{}); st.OK || st.Reason != ReasonTown {
		t.Fatalf("%+v", st)
	}

	f.p.Opt.IgnoreTown = true
	if st := f.p.Start(f.u, fb, Target{}); !st.OK {
		t.Fatalf("%+v", st)
	}

	f.p.Opt.IgnoreTown = false

	// InTown skills work in town; passive skills are not cast
	if st := f.p.Start(f.u, f.id("Warmth"), Target{}); st.OK || st.Reason != ReasonPassive {
		t.Fatalf("%+v", st)
	}

	f.u.town = false

	// mana
	f.u.mana = 639
	if st := f.p.Start(f.u, fb, Target{}); st.OK || st.Reason != ReasonMana {
		t.Fatalf("%+v", st)
	}

	f.u.mana = 640
	if st := f.p.Start(f.u, fb, Target{}); !st.OK {
		t.Fatalf("exactly enough mana: %+v", st)
	}

	// line of sight (type 4 = mask 0x804): a closed door between caster and target
	f.u.mana = 100 << 8
	f.w.grid.Set(3, 0, d2path.FlagDoor)

	if st := f.p.Start(f.u, f.id("Static Field"), Target{X: 10, Y: 0}); st.OK || st.Reason != ReasonLOS {
		t.Fatalf("%+v", st)
	}

	if st := f.p.Start(f.u, f.id("Static Field"), Target{X: 10, Y: 8}); !st.OK {
		t.Fatalf("clear line: %+v", st)
	}

	// ammo for srvst 4
	f.u.ammo = 0
	if st := f.p.Start(f.u, f.id("Fire Arrow"), Target{}); st.OK || st.Reason != ReasonAmmo {
		t.Fatalf("%+v", st)
	}

	// cooldown
	do := f.p.Do(f.u, f.id("Test Delay"), Target{X: 5})
	if !do.OK || do.Cooldown != 35 {
		t.Fatalf("%+v", do)
	}

	f.w.frame = 34
	if st := f.p.Start(f.u, f.id("Test Delay"), Target{}); st.OK || st.Reason != ReasonCooldown {
		t.Fatalf("%+v", st)
	}

	f.w.frame = 35
	if st := f.p.Start(f.u, f.id("Test Delay"), Target{}); !st.OK {
		t.Fatalf("%+v", st)
	}
}

func TestManaTiming(t *testing.T) {
	f := newFixture(map[string]int{"Test Late": 1, "Test Start": 1})
	f.u.base = f.u.levels

	// srvst + usemanaondo: paid at the do frame
	late := f.id("Test Late")
	if st := f.p.Start(f.u, late, Target{X: 5}); !st.OK || st.ManaPaid != 0 || f.u.mana != 100<<8 {
		t.Fatalf("%+v mana %d", st, f.u.mana)
	}

	if do := f.p.Do(f.u, late, Target{X: 5}); !do.OK || do.ManaPaid != 10*256 || f.u.mana != 90<<8 {
		t.Fatalf("%+v mana %d", do, f.u.mana)
	}

	// srvst without usemanaondo: paid at the start, not again at the do frame
	f.u.mana = 100 << 8
	start := f.id("Test Start")

	if st := f.p.Start(f.u, start, Target{X: 5}); !st.OK || st.ManaPaid != 10*256 || f.u.mana != 90<<8 {
		t.Fatalf("%+v mana %d", st, f.u.mana)
	}

	if do := f.p.Do(f.u, start, Target{X: 5}); !do.OK || do.ManaPaid != 0 || f.u.mana != 90<<8 {
		t.Fatalf("%+v mana %d", do, f.u.mana)
	}

	// mana drained between start and do for a pay-at-do skill: the cast fails
	f.u.mana = 640
	fb := newFixture(map[string]int{"Fire Bolt": 1})
	fb.u.base = fb.u.levels
	fb.u.mana = 100

	if do := fb.p.Do(fb.u, fb.id("Fire Bolt"), Target{X: 5}); do.OK || do.Reason != ReasonMana {
		t.Fatalf("%+v", do)
	}
}

func TestChargedBoltFanOut(t *testing.T) {
	f := newFixture(map[string]int{"Charged Bolt": 5})
	f.u.base = f.u.levels

	_, do := f.cast("Charged Bolt", 20, 0)
	// calc1 = min(24, 3+(5-1)) = 7 bolts, 5.0 mana
	if !do.OK || len(do.Missiles) != 7 || f.u.mana != 100<<8-1280 {
		t.Fatalf("%d missiles, mana %d (%+v)", len(do.Missiles), f.u.mana, do.OK)
	}

	for _, m := range do.Missiles {
		if m.Damage.Lightning.Min != 1024 || m.Damage.Lightning.Max != 1536 {
			t.Fatalf("per-bolt damage %+v", m.Damage.Lightning)
		}
	}

	// the seeded roller returns 0 -> -40 degrees for every bolt here; a different roll fans out
	f2 := newFixture(map[string]int{"Charged Bolt": 2})
	f2.u.base = f2.u.levels
	f2.u.roller = &seq{vals: []uint32{0, 40, 80}}
	_, do = f2.cast("Charged Bolt", 20, 0)

	if len(do.Missiles) != 4 {
		t.Fatalf("level 2 -> 4 bolts, got %d", len(do.Missiles))
	}

	if a, b, c := do.Missiles[0].DY, do.Missiles[1].DY, do.Missiles[2].DY; !(a < b && b < c) {
		t.Fatalf("not fanned out: %v %v %v", a, b, c)
	}
}

func TestFrozenArmorState(t *testing.T) {
	f := newFixture(map[string]int{"Frozen Armor": 1})
	f.u.base = map[string]int{"Frozen Armor": 1, "Shiver Armor": 2, "Chilling Armor": 4}
	f.u.levels = f.u.base
	f.u.town = true // InTown skill

	_, do := f.cast("Frozen Armor", 0, 0)
	if !do.OK || len(do.Effects) != 1 {
		t.Fatalf("%+v", do)
	}

	e := do.Effects[0]
	// duration 3000 + 6*250 synergy; armor percent 30 at level 1; chill 30*(100+6*5)/100 = 39
	if e.Kind != "self_state" || e.State != "frozenarmor" || e.Frames != 4500 || e.Chill != 39 ||
		len(e.Stats) != 1 || e.Stats[0] != (StatMod{"skill_armor_percent", 30}) {
		t.Fatalf("%+v", e)
	}

	if f.u.mana != 100<<8-7*256 {
		t.Fatalf("mana %d", f.u.mana)
	}
}

func TestStaticFieldAndInnerSight(t *testing.T) {
	f := newFixture(map[string]int{"Static Field": 4, "Inner Sight": 3})
	f.u.base = f.u.levels

	_, do := f.cast("Static Field", 5, 5)
	if !do.OK || len(do.Effects) != 1 {
		t.Fatalf("%+v", do)
	}

	e := do.Effects[0]
	// radius ln12 = 5+3, 25% of current life, floor from the options
	if e.Kind != "area_damage" || e.Radius != 8 || e.Pct != 25 || e.MinDamage != 0 || e.FloorPct != 25 || e.EType != "ltng" {
		t.Fatalf("%+v", e)
	}

	_, do = f.cast("Inner Sight", 5, 5)
	e = do.Effects[0]
	// duration ln34 = 200+2*100, radius ln56 = 20, armor class -(40+2*25)
	if e.Kind != "area_state" || e.State != "innersight" || e.Frames != 400 || e.Radius != 20 ||
		e.Stats[0] != (StatMod{"armorclass", -90}) {
		t.Fatalf("%+v", e)
	}
}

func TestWarmthPassive(t *testing.T) {
	f := newFixture(map[string]int{"Warmth": 5})
	f.u.base = f.u.levels

	got := f.p.PassiveStats(f.u, f.id("Warmth"))
	// 30 + 12*(5-1) percent mana recovery
	if len(got) != 1 || got[0] != (StatMod{"manarecoverybonus", 78}) {
		t.Fatalf("%+v", got)
	}

	if f.p.PassiveStats(f.u, f.id("Fire Bolt")) != nil {
		t.Fatal("non passive")
	}
}

func TestHowlRing(t *testing.T) {
	f := newFixture(map[string]int{"Howl": 5})
	f.u.base = f.u.levels
	f.w.targets = []*testTarget{{id: "z1", alive: true, level: 1, x: 3, y: 0}}

	_, do := f.cast("Howl", 0, 0)
	if !do.OK || len(do.Missiles) != 64 {
		t.Fatalf("%d missiles %+v", len(do.Missiles), do.OK)
	}

	// velocity = (12 + par1*(lvl-1)) << 8
	if v := do.Missiles[0].Velocity; v != (12+8)<<8 {
		t.Fatalf("velocity %d", v)
	}

	for i := 0; i < 15; i++ {
		f.w.frame++
		f.sim.Step()
	}

	// the target is struck by the missiles passing it; fear lasts 75+25*(5-1) frames
	if len(f.state) == 0 || f.state[0] != "z1:terror:175" {
		t.Fatalf("states %v", f.state)
	}

	for _, e := range f.evs {
		if e.Kind == d2missile.EventHit && e.Damage.SumTotal(true) != 0 {
			t.Fatalf("howl deals damage %+v", e.Damage)
		}
	}
}

func TestMeleeSkills(t *testing.T) {
	// Jab level 1: to-hit +10%, damage -15%; weapon 5-9; roller 0 -> to-hit roll 0 (hit), damage min
	f := newFixture(map[string]int{"Jab": 1, "Bash": 3, "Stun": 2, "Attack": 1, "Kick": 1})
	f.u.base = f.u.levels
	f.u.stats["strength"], f.u.stats["dexterity"] = 50, 30
	target := &testTarget{id: "zombie", alive: true, level: 1, defense: 10}

	cast := func(name string) DoResult {
		id := f.id(name)
		tg := Target{Unit: target}
		f.p.Start(f.u, id, tg)

		return f.p.Do(f.u, id, tg)
	}

	// Attack: weapon 5..9, SrcDam 128 -> unchanged; roller 0 -> 5.0
	do := cast("Attack")
	if !do.OK || do.Melee == nil || !do.Melee.Hit || do.Melee.Total != 5*256 {
		t.Fatalf("attack %+v", do.Melee)
	}

	// Jab: 5.0 - 15% = 4.25 -> 1088 (8.8)
	do = cast("Jab")
	if !do.Melee.Hit || do.Melee.Damage.Physical != 5*256-5*256*15/100 {
		t.Fatalf("jab %+v", do.Melee)
	}

	// Bash level 3: calc1 = 50+10 + Stun(2)*5 = 70% bonus; calc2 = 1+2 = 3 flat added after
	do = cast("Bash")
	want := int32(5*256 + 5*256*70/100 + 3*256)
	if !do.Melee.Hit || do.Melee.Damage.Physical != want {
		t.Fatalf("bash %d want %d", do.Melee.Damage.Physical, want)
	}

	// Stun level 2: stun length 30+5 frames, damage bonus = Bash.blvl(3)*8 = 24%
	do = cast("Stun")
	if !do.Melee.Hit || do.Melee.Damage.StunLen != 35 || do.Melee.Damage.Physical != 5*256+5*256*24/100 {
		t.Fatalf("stun %+v", do.Melee.Damage)
	}

	// Kick: always hits; (50+30-20)/4 = 15 damage
	do = cast("Kick")
	if !do.Melee.Hit || do.Melee.Damage.Physical != 15*256 || do.Melee.Chance != 100 {
		t.Fatalf("kick %+v", do.Melee)
	}

	// no target
	id := f.id("Attack")
	if d := f.p.Do(f.u, id, Target{}); d.OK || d.Reason != ReasonTarget {
		t.Fatalf("%+v", d)
	}

	// a miss: huge defense, high roll
	target.defense = 1 << 20
	f.u.roller = &seq{vals: []uint32{99}}

	if do = cast("Attack"); do.Melee.Hit {
		t.Fatalf("expected a miss %+v", do.Melee)
	}
}

func TestBashToHitUsesCalc(t *testing.T) {
	f := newFixture(map[string]int{"Bash": 6})
	f.u.base = map[string]int{"Bash": 6, "Concentrate": 2}
	// ToHitCalc 15+lvl*5+Concentrate.blvl*par7 = 15+30+2*5
	e := f.env("Bash", 6)
	if got := f.reg.ByName("Bash").ToHitBonus(e, 6); got != 55 {
		t.Fatalf("bash to-hit bonus %d", got)
	}

	if got := f.reg.ByName("Jab").ToHitBonus(f.env("Jab", 4), 4); got != 10+9*3 {
		t.Fatalf("jab to-hit bonus %d", got)
	}
}

func TestArrowSkills(t *testing.T) {
	f := newFixture(map[string]int{"Magic Arrow": 3, "Fire Arrow": 2})
	f.u.base = map[string]int{"Magic Arrow": 3, "Fire Arrow": 2, "Exploding Arrow": 5}

	// Magic Arrow: no ammo needed (noammo), weapon 5-9 + MinDam 1 + 2 levels*1, pierce flag from the table
	_, do := f.cast("Magic Arrow", 20, 0)
	if !do.OK || len(do.Missiles) != 1 {
		t.Fatalf("%+v", do)
	}

	d := do.Missiles[0].Damage
	if d.PhysMin != (5+1+2)<<8 || d.PhysMax != (9+1+2)<<8 || d.ToHit != 10+9*2 {
		t.Fatalf("magic arrow %+v", d)
	}

	// Fire Arrow level 2: phys weapon, fire min (1+2) + 60% synergy (5*12), decquant uses an arrow
	f.u.levels = map[string]int{"Magic Arrow": 3, "Fire Arrow": 2}
	_, do = f.cast("Fire Arrow", 20, 0)

	if !do.OK || f.u.ammo != 0 {
		t.Fatalf("%+v ammo %d", do, f.u.ammo)
	}

	d = do.Missiles[0].Damage
	if d.PhysMin != 5<<8 || d.Fire.Min != 3*256+3*256*60/100 || d.Fire.Max != 6*256+6*256*60/100 {
		t.Fatalf("fire arrow %+v", d)
	}
}

func TestAttackWithBowFiresArrow(t *testing.T) {
	f := newFixture(map[string]int{"Attack": 1})
	f.u.base = f.u.levels
	f.u.ranged = "firearrow"
	f.w.targets = []*testTarget{{id: "z", alive: true, level: 1, x: 8, y: 0}}

	_, do := f.cast("Attack", 30, 0)
	if !do.OK || len(do.Missiles) != 1 || do.Missiles[0].Spec.Name != "firearrow" || do.Melee != nil {
		t.Fatalf("%+v", do)
	}

	// arrow damage is the weapon damage (SrcDam 128)
	if d := do.Missiles[0].Damage; d.PhysMin != 5<<8 || d.PhysMax != 9<<8 {
		t.Fatalf("%+v", d)
	}
}

func TestThrow(t *testing.T) {
	f := newFixture(map[string]int{"Throw": 1})
	f.u.base = f.u.levels

	if _, do := f.cast("Throw", 10, 0); do.OK || do.Reason != ReasonNoWeapon {
		t.Fatalf("%+v", do)
	}

	f.u.thrown = "firebolt" // any missile record stands in for a throwing knife
	if _, do := f.cast("Throw", 10, 0); !do.OK || len(do.Missiles) != 1 {
		t.Fatalf("%+v", do)
	}
}
