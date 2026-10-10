package d2records

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2combat"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2missile"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2path"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2skill"
)

// The skills audit walks every skills.txt row (357; 210 of them player skills)
// against the tables the skill engine depends on (missiles.txt, States.txt,
// Overlay.txt, skilldesc.txt, monstats.txt) and against the engine itself: it
// casts every skill through d2skill.Pipeline at levels 1 and 20 and records
// what comes out. D2_WRITE_SKILLS_AUDIT=<file> writes docs/skills-coverage.md.
// Needs D2_TABLES (skipped otherwise); no game data is stored in the repo.

// ---- raw tables ----

type rawTable struct {
	rows []map[string]string
}

func readRaw(t *testing.T, rel string) *rawTable {
	t.Helper()

	buf, err := os.ReadFile(filepath.Join(os.Getenv("D2_TABLES"), rel))
	if err != nil {
		t.Skip(err)
	}

	lines := strings.Split(strings.ReplaceAll(string(buf), "\r", ""), "\n")
	hdr := strings.Split(lines[0], "\t")
	rt := &rawTable{}

	for _, ln := range lines[1:] {
		if strings.TrimSpace(ln) == "" {
			continue
		}

		cols := strings.Split(ln, "\t")
		r := map[string]string{}

		for i, h := range hdr {
			if i < len(cols) && h != "" {
				r[strings.ToLower(h)] = cols[i]
			}
		}

		rt.rows = append(rt.rows, r)
	}

	return rt
}

func (rt *rawTable) set(key string) map[string]bool {
	out := map[string]bool{}

	for _, r := range rt.rows {
		if v := strings.ToLower(r[key]); v != "" {
			out[v] = true
		}
	}

	return out
}

// ---- engine fixture ----

type auditUnit struct {
	reg    *d2skill.Registry
	self   int
	lvl    int
	player bool
	rng    *d2rand.Seed
	mana   int
	cd     map[int]int
}

func (u *auditUnit) ID() string     { return "hero" }
func (u *auditUnit) IsPlayer() bool { return u.player }
func (u *auditUnit) Level() int     { return 30 }
func (u *auditUnit) SkillLevel(id int) int {
	if id == u.self {
		return u.lvl
	}

	return 0
}

func (u *auditUnit) BaseSkillLevel(id int) int {
	if id == u.self {
		return u.lvl
	}

	return 0
}
func (u *auditUnit) Stat(string) int             { return 0 }
func (u *auditUnit) Mana() int                   { return u.mana }
func (u *auditUnit) SetMana(v int)               { u.mana = v }
func (u *auditUnit) Pos() (int, int)             { return 50, 50 }
func (u *auditUnit) InTown() bool                { return false }
func (u *auditUnit) Roller() d2combat.Roller     { return u.rng }
func (u *auditUnit) AttackRating() int           { return 500 }
func (u *auditUnit) WeaponDamage() (int, int)    { return 10, 20 }
func (u *auditUnit) RangedWeaponMissile() string { return "" }
func (u *auditUnit) ThrownMissile() string       { return "" }
func (u *auditUnit) HasAmmo() bool               { return true }
func (u *auditUnit) ConsumeAmmo() bool           { return true }
func (u *auditUnit) Cooldown(id int) int         { return u.cd[id] }
func (u *auditUnit) SetCooldown(id, until int)   { u.cd[id] = until }

type auditFoe struct{ x, y int }

func (f *auditFoe) ID() string       { return "foe" }
func (f *auditFoe) IsPlayer() bool   { return false }
func (f *auditFoe) Alive() bool      { return true }
func (f *auditFoe) Level() int       { return 20 }
func (f *auditFoe) Defense(bool) int { return 0 }
func (f *auditFoe) Serial() int      { return 7 }

type auditWorld struct {
	grid *d2path.CellGrid
	foe  *auditFoe
}

func (w *auditWorld) Flags(x, y int) uint16                          { return w.grid.Flags(x, y) }
func (w *auditWorld) Frame() int                                     { return 0 }
func (w *auditWorld) IsEnemy(d2missile.Owner, d2missile.Target) bool { return true }
func (w *auditWorld) Targets(x, y int) []d2missile.Target {
	if x >= w.foe.x-1 && x <= w.foe.x+1 && y >= w.foe.y-1 && y <= w.foe.y+1 {
		return []d2missile.Target{w.foe}
	}

	return nil
}

// castOutcome is what a cast produced.
type castOutcome struct {
	StartOK, DoOK bool
	Reason        string
	Missiles      int
	Effects       []string
	Melee         bool
	Summon        string
	Cooldown      int
	Paid          int
	Panic         string
}

func (c castOutcome) produced() bool {
	return c.Missiles > 0 || len(c.Effects) > 0 || c.Melee
}

func castOnce(reg *d2skill.Registry, mt *MissileTable, id, lvl int) (out castOutcome) {
	defer func() {
		if r := recover(); r != nil {
			out.Panic = fmt.Sprint(r)
		}
	}()

	w := &auditWorld{grid: d2path.NewCellGrid(-50, -50, 300, 200), foe: &auditFoe{x: 55, y: 50}}
	sim := d2missile.NewSim(w, mt)
	p := &d2skill.Pipeline{Skills: reg, Missiles: mt, Sim: sim, Grid: w.grid, Frame: func() int { return 0 },
		Opt: d2skill.Options{IgnoreTown: true, StaticFieldMinPct: 25}}
	p.Near = func(x, y, r int) []d2skill.Foe {
		if abs(w.foe.x-x) <= r && abs(w.foe.y-y) <= r {
			return []d2skill.Foe{{Target: w.foe, X: w.foe.x, Y: w.foe.y}}
		}

		return nil
	}
	p.ApplyState = func(d2skill.Unit, d2missile.Target, string, int) {}
	p.After = func(int, func()) {}

	u := &auditUnit{reg: reg, self: id, lvl: lvl, player: true, rng: d2rand.New(1), mana: 5000 << 8, cd: map[int]int{}}
	tg := d2skill.Target{X: 55, Y: 50, Unit: w.foe, UX: 55, UY: 50, Corpse: true, CX: 55, CY: 50,
		CorpseID: "c1", CorpseHP: 200, CorpseKey: "zombie", CorpseLevel: 5}

	st := p.Start(u, id, tg)
	out.StartOK, out.Reason = st.OK, st.Reason

	if !st.OK {
		return out
	}

	res := p.Do(u, id, tg)
	out.DoOK, out.Missiles, out.Melee, out.Cooldown, out.Paid = res.OK, len(res.Missiles), res.Melee != nil, res.Cooldown, res.ManaPaid

	if !res.OK {
		out.Reason = res.Reason
	}

	for _, e := range res.Effects {
		out.Effects = append(out.Effects, e.Kind)

		if e.Summon != nil && out.Summon == "" {
			out.Summon = e.Summon.Key
		}
	}

	return out
}

func abs(a int) int {
	if a < 0 {
		return -a
	}

	return a
}

// ---- audit rows ----

type auditRow struct {
	ID                int
	Name, Class       string
	Player            bool
	St, Do            int
	DoState           string // handler | generic | none | MISSING
	Passive           bool
	Mana1, Mana20     int // 8.8
	Delay1, Delay20   int
	Missile           string
	MVel1, MVel20, MR int
	ClosureGaps       []string
	DmgKind           string
	DmgMin, DmgMax    int // 8.8 at level 20
	Elen20            int
	AuraLen20         int
	States            []string
	Summon            string
	PetMax20          int
	Syn               int
	ReqLvl, MaxLvl    int
	Req               []string
	Cast1, Cast20     castOutcome
	Unresolved        []string
	Stat1, StatN      int // passive stats at 1 / 20
	StandIn           string
}

var synRe = []string{"skill('", `skill("`}

func synergyNames(srcs ...string) map[string]bool {
	out := map[string]bool{}

	for _, s := range srcs {
		low := strings.ToLower(s)
		for _, pre := range synRe {
			rest := low
			for {
				i := strings.Index(rest, pre)
				if i < 0 {
					break
				}

				rest = rest[i+len(pre):]

				j := strings.IndexAny(rest, `'"`)
				if j < 0 {
					break
				}

				out[rest[:j]] = true
			}
		}
	}

	return out
}

// missileClosure lists the missiles a skill can fire (srvmissile*, then
// submissiles, hit submissiles and the explosion missile recursively).
func missileClosure(mt *MissileTable, roots []string) []*d2missile.Spec {
	seen := map[string]bool{}

	var out []*d2missile.Spec

	var walk func(n string, depth int)

	walk = func(n string, depth int) {
		sp := mt.ByName(n)
		if n == "" || sp == nil || seen[sp.Name] || depth > 4 {
			return
		}

		seen[sp.Name] = true
		out = append(out, sp)

		for _, s := range sp.SubMissile {
			walk(s, depth+1)
		}

		for _, s := range sp.HitSubMissile {
			walk(s, depth+1)
		}

		walk(sp.ExplosionMissile, depth+1)
	}

	for _, r := range roots {
		walk(r, 0)
	}

	return out
}

func buildAudit(t *testing.T) ([]auditRow, *RecordManager) {
	t.Helper()

	rm := loadRealRecords(t)
	reg := rm.SkillTable()
	mt := rm.MissileTable()

	raw := readRaw(t, "skills/patch_d2/skills.txt")
	states := readRaw(t, "states/patch_d2/States.txt").set("state")
	overlays := readRaw(t, "overlay/patch_d2/Overlay.txt").set("overlay")
	descs := readRaw(t, "skills/patch_d2/skilldesc.txt").set("skilldesc")
	mons := readRaw(t, "monsters/patch_d2/monstats.txt").set("id")
	names := raw.set("skill")

	var out []auditRow

	for _, r := range raw.rows {
		if r["skill"] == "" || r["id"] == "" {
			continue
		}

		id, _ := strconv.Atoi(r["id"])
		sk := reg.ByID(id)
		rec := rm.Skill.Details[id]

		if sk == nil || rec == nil {
			t.Fatalf("skill %d not loaded", id)
		}

		a := auditRow{ID: id, Name: sk.Name, Class: r["charclass"], Player: r["charclass"] != "", Passive: sk.Passive,
			St: sk.SrvStFunc, Do: sk.SrvDoFunc, ReqLvl: rec.Reqlevel, MaxLvl: rec.Maxlvl,
			Mana1: sk.ManaCost(1), Mana20: sk.ManaCost(20), StandIn: standIns[id]}

		a.Syn = len(synergyNames(r["calc1"], r["calc2"], r["calc3"], r["calc4"], r["dmgsymperc"+"alc"],
			r["edmgsymperc"+"alc"], r["elensymperc"+"alc"], r["auralencalc"], r["aurarangecalc"], r["aurastatcalc1"],
			r["aurastatcalc2"], r["aurastatcalc3"], r["aurastatcalc4"], r["aurastatcalc5"], r["aurastatcalc6"],
			r["passivecalc1"], r["passivecalc2"], r["passivecalc3"], r["passivecalc4"], r["passivecalc5"], r["petmax"],
			r["delay"], r["tohitcalc"]))

		// handler
		switch {
		case sk.Passive:
			a.DoState = "passive"
		case d2skill.Implemented(sk) && sk.SrvDoFunc != 0:
			a.DoState = "handler"
		case d2skill.Implemented(sk):
			a.DoState = "generic"
		case sk.SrvDoFunc == 0 && sk.SrvMissile == "":
			a.DoState = "none"
		default:
			a.DoState = "MISSING"
		}

		// delay, damage, aura
		u := &auditUnit{reg: reg, self: id, lvl: 1, player: true, rng: d2rand.New(1), cd: map[int]int{}}
		e1 := d2skill.NewEnv(sk, 1, u, reg)
		u20 := &auditUnit{reg: reg, self: id, lvl: 20, player: true, rng: d2rand.New(1), cd: map[int]int{}}
		e20 := d2skill.NewEnv(sk, 20, u20, reg)
		a.Delay1, a.Delay20 = e1.Eval(sk.Delay), e20.Eval(sk.Delay)

		if sk.EType != "" && sk.EType != "none" || sk.EMax != 0 || sk.EMaxLev[0] != 0 {
			a.DmgKind = "elem:" + sk.EType
			a.DmgMin, a.DmgMax = int(sk.ElemMin(e20, 20)), int(sk.ElemMax(e20, 20))
			a.Elen20 = sk.ElemLen(e20, 20)
		} else if sk.SrcDam != 0 || sk.MinDam != 0 || sk.MaxDam != 0 || sk.MaxLevDam[0] != 0 {
			a.DmgKind = "phys"
			a.DmgMin, a.DmgMax = int(sk.PhysMin(e20, 20, 10, true)), int(sk.PhysMax(e20, 20, 20, true))
		}

		if sk.AuraLenCalc != nil {
			a.AuraLen20 = e20.Eval(sk.AuraLenCalc)
		}

		if sk.PetMax != nil {
			a.PetMax20 = e20.Eval(sk.PetMax)
		}

		// missiles
		roots := []string{sk.SrvMissile, sk.SrvMissileA, sk.SrvMissileB, sk.SrvMissileC}
		for _, n := range roots {
			if n != "" && a.Missile == "" {
				a.Missile = n
			}
		}

		if sp := mt.ByName(a.Missile); sp != nil {
			a.MVel1, a.MVel20 = sp.Vel, min(sp.Vel+sp.VelLev*19, max(sp.MaxVel, sp.Vel))
			a.MR = sp.Range
		}

		gaps := map[string]bool{}

		for _, sp := range missileClosure(mt, roots) {
			if !d2missile.DoFuncModelled(sp.SrvDoFunc) {
				gaps[fmt.Sprintf("%s:do%d", sp.Name, sp.SrvDoFunc)] = true
			}

			if !d2missile.HitFuncModelled(sp.SrvHitFunc) {
				gaps[fmt.Sprintf("%s:hit%d", sp.Name, sp.SrvHitFunc)] = true
			}
		}

		for g := range gaps {
			a.ClosureGaps = append(a.ClosureGaps, g)
		}

		sort.Strings(a.ClosureGaps)

		// table resolution
		unres := func(kind, v string, set map[string]bool) {
			if v != "" && !set[strings.ToLower(v)] {
				a.Unresolved = append(a.Unresolved, kind+"="+v)
			}
		}

		for _, c := range []string{"srvmissile", "srvmissilea", "srvmissileb", "srvmissilec", "cltmissile", "cltmissilea",
			"cltmissileb", "cltmissilec", "cltmissiled"} {
			if v := r[c]; v != "" && mt.ByName(v) == nil {
				a.Unresolved = append(a.Unresolved, c+"="+v)
			}
		}

		for _, c := range []string{"aurastate", "auratargetstate", "passivestate", "state1", "state2", "state3"} {
			unres(c, r[c], states)

			if r[c] != "" {
				a.States = append(a.States, r[c])
			}
		}

		for _, c := range []string{"srvoverlay", "sumoverlay", "tgtoverlay", "prgoverlay", "castoverlay", "cltoverlaya",
			"cltoverlayb"} {
			unres(c, r[c], overlays)
		}

		unres("skilldesc", r["skilldesc"], descs)
		unres("summon", r["summon"], mons)

		for _, c := range []string{"reqskill1", "reqskill2", "reqskill3"} {
			unres(c, r[c], names)

			if r[c] != "" {
				a.Req = append(a.Req, r[c])
			}
		}

		for _, c := range []string{"sumskill1", "sumskill2", "sumskill3", "sumskill4", "sumskill5"} {
			unres(c, r[c], names)
		}

		for n := range synergyNames(r["calc1"], r["calc2"], r["calc3"], r["calc4"], r["edmgsymperccalc"],
			r["edmgsymperc"+"calc"], r["dmgsymperccalc"], r["auralencalc"], r["aurarangecalc"], r["aurastatcalc1"],
			r["passivecalc1"], r["petmax"], r["delay"]) {
			unres("synergy", n, names)
		}

		a.Summon = r["summon"]

		// casting
		if sk.Passive {
			a.Stat1 = len(passiveMods(reg, sk, 1))
			a.StatN = len(passiveMods(reg, sk, 20))
		} else {
			a.Cast1 = castOnce(reg, mt, id, 1)
			a.Cast20 = castOnce(reg, mt, id, 20)
		}

		out = append(out, a)
	}

	return out, rm
}

func min(a, b int) int {
	if a < b {
		return a
	}

	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}

	return b
}

func passiveMods(reg *d2skill.Registry, sk *d2skill.Skill, lvl int) []d2skill.StatMod {
	p := &d2skill.Pipeline{Skills: reg}
	u := &auditUnit{reg: reg, self: sk.ID, lvl: lvl, player: true, rng: d2rand.New(1), cd: map[int]int{}}

	return p.PassiveStats(u, sk.ID)
}

// standIns marks skills whose engine behaviour is a documented stand-in
// (kept in step with the STAND-IN comments in d2common/d2skill).
var standIns = map[int]string{}
