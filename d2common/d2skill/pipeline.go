package d2skill

import (
	"fmt"
	"math"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2combat"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2missile"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2path"
)

// Failure reasons returned by Start and Do.
const (
	ReasonNoSkill  = "no_skill"  // unknown skill or level < 1
	ReasonPassive  = "passive"   // passive skills are not cast
	ReasonTown     = "town"      // InTown flag clear and the caster is in town
	ReasonMana     = "mana"      // not enough mana
	ReasonCooldown = "cooldown"  // state 121 (delay) still running
	ReasonAmmo     = "no_ammo"   // ranged skill without ammunition
	ReasonLOS      = "los"       // LineOfSight test failed
	ReasonTarget   = "no_target" // melee skill without a target unit
	ReasonMissile  = "missile"   // missile creation failed
	ReasonNoWeapon = "no_throwable"
)

// srvst/srvdo function ids used by the implemented skills (skills.txt columns).
const (
	stAttack   = 1
	stKick     = 2
	stRanged   = 4 // SRVST_RangedAmmoCheck
	stJab      = 5
	stBash     = 32 // Bash, Stun, Concentrate...
	stThrow    = 65
	doAttack   = 1
	doMelee    = 2 // Kick, Power Strike, Bash, Stun
	doThrow    = 3
	doLHThrow  = 5
	doInner    = 6
	doJab      = 7
	doCharged  = 17
	doFrozenAr = 18
	doStatic   = 20
	doNova     = 22 // Frost Nova, Nova, Poison Nova, Howl
	novaCount  = 64
	novaRadius = 10.0
)

// los masks for skills.txt LineOfSight (verified).
var losMask = map[int]uint16{1: 0x4, 2: 0x1C09, 3: 0x180, 4: 0x804, 5: 0x805}

// Target is the aim of a cast.
type Target struct {
	X, Y int // aim point, subtile
	// Unit is the unit under the cursor (needed by melee skills), or nil.
	Unit d2missile.Target
	// UX, UY are its subtile position.
	UX, UY int
}

// Options tune the pipeline.
type Options struct {
	// IgnoreTown lets skills be cast in town (tests).
	IgnoreTown bool
	// StaticFieldMinPct is DifficultyLevels StaticFieldMin: the percent of a
	// monster's maximum life Static Field cannot reduce it below (U meaning).
	StaticFieldMinPct int
}

// Pipeline runs skills for casters.
type Pipeline struct {
	Skills   *Registry
	Missiles d2missile.Table
	Sim      *d2missile.Sim
	Grid     d2path.Grid
	Frame    func() int
	Opt      Options
	MissF    MissileFields

	// ApplyState is called when a missile of a skill applies a state to an
	// enemy (Howl's fear). frames is the duration.
	ApplyState func(owner Unit, t d2missile.Target, state string, frames int)
}

func (p *Pipeline) env(sk *Skill, lvl int, u Unit) *Env {
	e := NewEnv(sk, lvl, u, p.Skills)
	e.Missiles = p.MissF

	return e
}

// StartResult is the outcome of the start phase.
type StartResult struct {
	OK       bool
	Reason   string
	Level    int
	ManaPaid int // 8.8, 0 if nothing was paid yet
}

// payAtStart: mana is paid in the start function when the skill has a start
// function and not usemanaondo; otherwise in the do function (verified for
// the do side; the notes' step 4 of the start function is read as "when a
// start function exists", U for skills without one).
func payAtStart(sk *Skill) bool { return sk.SrvStFunc != 0 && !sk.UseManaOnDo }

func (p *Pipeline) frame() int {
	if p.Frame == nil {
		return 0
	}

	return p.Frame()
}

func (p *Pipeline) needsAmmo(u Unit, sk *Skill) bool {
	switch {
	case sk.NoAmmo:
		return false
	case sk.SrvStFunc == stRanged, sk.SrvStFunc == stThrow:
		return true
	case sk.SrvStFunc == stAttack:
		return u.RangedWeaponMissile() != ""
	}

	return false
}

// Start is SKILL_ServerRunStartFunc: it validates the cast and pays the mana
// of skills that pay at the start. It runs when the cast animation begins.
func (p *Pipeline) Start(u Unit, skillID int, tgt Target) StartResult {
	sk := p.Skills.ByID(skillID)
	if sk == nil {
		return StartResult{Reason: ReasonNoSkill}
	}

	lvl := u.SkillLevel(skillID)
	if lvl < 1 {
		return StartResult{Reason: ReasonNoSkill}
	}

	if sk.Passive {
		return StartResult{Reason: ReasonPassive, Level: lvl}
	}

	if !sk.InTown && u.InTown() && !p.Opt.IgnoreTown {
		return StartResult{Reason: ReasonTown, Level: lvl}
	}

	if until := u.Cooldown(skillID); until > p.frame() {
		return StartResult{Reason: ReasonCooldown, Level: lvl}
	}

	cost := sk.ManaCost(lvl)
	if u.IsPlayer() && cost > u.Mana() {
		return StartResult{Reason: ReasonMana, Level: lvl}
	}

	if sk.LineOfSight != 0 && p.Grid != nil {
		x, y := u.Pos()
		if clear, _ := d2path.TraceLine(p.Grid, losMask[sk.LineOfSight], d2path.Point{X: x, Y: y},
			d2path.Point{X: tgt.X, Y: tgt.Y}); !clear {
			return StartResult{Reason: ReasonLOS, Level: lvl}
		}
	}

	if p.needsAmmo(u, sk) && !u.HasAmmo() {
		return StartResult{Reason: ReasonAmmo, Level: lvl}
	}

	res := StartResult{OK: true, Level: lvl}

	if u.IsPlayer() && cost > 0 && payAtStart(sk) {
		rest, ok := d2combat.PayMana(u.Mana(), cost)
		if !ok {
			return StartResult{Reason: ReasonMana, Level: lvl}
		}

		u.SetMana(rest)
		res.ManaPaid = cost
	}

	return res
}

// StatMod is one stat given by a state.
type StatMod struct {
	Stat  string
	Value int
}

// Effect is a non-missile, non-melee outcome of a cast for the engine to apply.
type Effect struct {
	// Kind: "self_state" (Frozen Armor), "area_state" (Inner Sight),
	// "area_damage" (Static Field), "passive".
	Kind   string
	State  string
	Frames int
	Stats  []StatMod
	// Area effects.
	Radius int
	Filter int
	// Static Field: percent of current life, minimum damage (whole points),
	// the life floor percent (Options.StaticFieldMinPct), element and length.
	Pct, MinDamage, FloorPct int
	EType                    string
	ELen                     int
	// Chill is Frozen Armor's calc1: the frames a melee attacker is chilled.
	Chill int
}

// MeleeResult is a resolved melee strike.
type MeleeResult struct {
	Target d2missile.Target
	Hit    bool
	Chance int
	Roll   int
	Damage d2combat.Damage // rolled, before resists, 8.8
	Total  int32           // sum of the damage components, 8.8
}

// DoResult is the outcome of the do phase.
type DoResult struct {
	OK       bool
	Reason   string
	Level    int
	ManaPaid int
	Cooldown int // frames of delay applied
	Missiles []*d2missile.Missile
	Melee    *MeleeResult
	Effects  []Effect
}

// Do is SKILL_ServerRunSkillFunc: it runs at the animation's action frame.
// It checks the mana again for skills that pay here, runs the do function,
// creates the generic srvmissile, pays the mana, applies decquant and
// starts the delay cooldown.
func (p *Pipeline) Do(u Unit, skillID int, tgt Target) DoResult {
	sk := p.Skills.ByID(skillID)
	if sk == nil {
		return DoResult{Reason: ReasonNoSkill}
	}

	lvl := u.SkillLevel(skillID)
	if lvl < 1 {
		return DoResult{Reason: ReasonNoSkill}
	}

	cost := sk.ManaCost(lvl)
	payHere := u.IsPlayer() && cost > 0 && !payAtStart(sk)

	if payHere && cost > u.Mana() {
		return DoResult{Reason: ReasonMana, Level: lvl}
	}

	res := DoResult{OK: true, Level: lvl}
	env := p.env(sk, lvl, u)

	p.runDo(u, sk, lvl, tgt, env, &res)

	if !res.OK {
		return res
	}

	if sk.SrvMissile != "" {
		if m := p.castMissile(u, sk, lvl, env, sk.SrvMissile, tgt, castOpts{}); m != nil {
			res.Missiles = append(res.Missiles, m)
		} else {
			return DoResult{Reason: ReasonMissile, Level: lvl}
		}
	}

	if payHere {
		rest, _ := d2combat.PayMana(u.Mana(), cost)
		u.SetMana(rest)
		res.ManaPaid = cost
	}

	if sk.DecQuant {
		u.ConsumeAmmo()
	}

	if d := env.eval(sk.Delay); d > 0 && u.IsPlayer() {
		u.SetCooldown(skillID, p.frame()+d)
		res.Cooldown = d
	}

	return res
}

// runDo dispatches the srvdofunc.
func (p *Pipeline) runDo(u Unit, sk *Skill, lvl int, tgt Target, env *Env, res *DoResult) {
	switch sk.SrvDoFunc {
	case doAttack:
		if mname := u.RangedWeaponMissile(); mname != "" {
			if m := p.castMissile(u, sk, lvl, env, mname, tgt, castOpts{}); m != nil {
				res.Missiles = append(res.Missiles, m)
			} else {
				*res = DoResult{Reason: ReasonMissile, Level: lvl}
			}

			return
		}

		p.doMelee(u, sk, lvl, tgt, env, res)
	case doMelee, doJab:
		p.doMelee(u, sk, lvl, tgt, env, res)
	case doThrow, doLHThrow:
		name := u.ThrownMissile()
		if name == "" {
			*res = DoResult{Reason: ReasonNoWeapon, Level: lvl}
			return
		}

		if m := p.castMissile(u, sk, lvl, env, name, tgt, castOpts{}); m != nil {
			res.Missiles = append(res.Missiles, m)
		} else {
			*res = DoResult{Reason: ReasonMissile, Level: lvl}
		}
	case doCharged:
		p.doChargedBolt(u, sk, lvl, tgt, env, res)
	case doNova:
		p.doNovaRing(u, sk, lvl, tgt, env, res)
	case doFrozenAr:
		p.doState(sk, env, res, "self_state")
		res.Effects[len(res.Effects)-1].Chill = env.eval(sk.Calc[1])
	case doInner:
		p.doState(sk, env, res, "area_state")
		e := &res.Effects[len(res.Effects)-1]
		e.State, e.Radius, e.Filter = sk.AuraTargetState, env.eval(sk.AuraRangeCalc), sk.AuraFilter
	case doStatic:
		res.Effects = append(res.Effects, Effect{
			Kind: "area_damage", Radius: env.eval(sk.AuraRangeCalc), Filter: sk.AuraFilter,
			Pct: env.eval(sk.Calc[1]), MinDamage: env.eval(sk.Calc[2]), FloorPct: p.Opt.StaticFieldMinPct,
			EType: sk.EType, ELen: sk.ElemLen(env, lvl),
		})
	}
}

// doState builds the timed state of Frozen Armor / Inner Sight: duration from
// auralencalc and every aurastatN with its aurastatcalcN.
func (p *Pipeline) doState(sk *Skill, env *Env, res *DoResult, kind string) {
	e := Effect{Kind: kind, State: sk.AuraState, Frames: env.eval(sk.AuraLenCalc)}

	for i := 1; i <= 6; i++ {
		if sk.AuraStat[i] != "" {
			e.Stats = append(e.Stats, StatMod{Stat: sk.AuraStat[i], Value: env.eval(sk.AuraStatCalc[i])})
		}
	}

	res.Effects = append(res.Effects, e)
}

// PassiveStats returns the stats a passive skill gives at a level
// (passivestat1..5 with passivecalc1..5), e.g. Warmth's manarecoverybonus.
func (p *Pipeline) PassiveStats(u Unit, skillID int) []StatMod {
	sk := p.Skills.ByID(skillID)
	lvl := u.SkillLevel(skillID)

	if sk == nil || lvl < 1 || !sk.Passive {
		return nil
	}

	env := p.env(sk, lvl, u)

	var out []StatMod

	for i := 1; i <= 5; i++ {
		if sk.PassiveStat[i] != "" {
			out = append(out, StatMod{Stat: sk.PassiveStat[i], Value: env.eval(sk.PassiveCalc[i])})
		}
	}

	return out
}

type castOpts struct {
	angle    float64
	velocity int
	clamp    bool
	startX   float64
	startY   float64
	hasStart bool
	onHit    func(m *d2missile.Missile, t d2missile.Target)
}

func (p *Pipeline) owner(u Unit) d2missile.Owner {
	return d2missile.Owner{ID: u.ID(), IsPlayer: u.IsPlayer(), Level: u.Level(), AttackRating: u.AttackRating(),
		Roller: u.Roller()}
}

var masteryStat = map[string]string{
	"fire": "passive_fire_mastery", "ltng": "passive_ltng_mastery",
	"cold": "passive_cold_mastery", "pois": "passive_pois_mastery",
}

// castMissile is SKILL_ServerCastMissile: it creates one missile of the named
// record with the skill's damage descriptor, aimed at the target.
func (p *Pipeline) castMissile(u Unit, sk *Skill, lvl int, env *Env, name string, tgt Target, o castOpts) *d2missile.Missile {
	ms := p.Missiles.ByName(name)
	if ms == nil {
		return nil
	}

	wmin, wmax := u.WeaponDamage()
	if wmin < 1 {
		wmin = 1
	}

	if wmax < wmin+1 {
		wmax = wmin + 1
	}

	desc := sk.Descriptor(env, lvl, wmin, wmax, u.Stat(masteryStat[sk.EType]))

	x, y := u.Pos()
	sx, sy := float64(x)+0.5, float64(y)+0.5

	if o.hasStart {
		sx, sy = o.startX, o.startY
	}

	dx, dy := float64(tgt.X)+0.5, float64(tgt.Y)+0.5
	if tgt.Unit != nil {
		dx, dy = float64(tgt.UX)+0.5, float64(tgt.UY)+0.5
	}

	m, err := p.Sim.Create(d2missile.CreateParams{
		Spec: ms, Owner: p.owner(u), SkillID: sk.ID, Level: lvl, Damage: desc,
		X: sx, Y: sy, DestX: dx, DestY: dy, Angle: o.angle, Velocity: o.velocity, ClampToDest: o.clamp || sk.Lob,
		Pierce: u.Stat("pierce_idx"), OnHit: o.onHit,
	})
	if err != nil {
		return nil
	}

	return m
}

// doChargedBolt is SRVDO_ChargedBolt (0x5c73a0): calc1 bolts of srvmissilea,
// each in its own random direction. The game randomises direction and speed
// in a post-create callback (0x5c7340) that was not read; here every bolt
// is rotated by a uniform random angle within +-40 degrees (UNVERIFIED).
func (p *Pipeline) doChargedBolt(u Unit, sk *Skill, lvl int, tgt Target, env *Env, res *DoResult) {
	name := sk.SrvMissileA
	if name == "" {
		name = sk.SrvMissile
	}

	n := env.eval(sk.Calc[1])
	for i := 0; i < n; i++ {
		deg := 0
		if r := u.Roller(); r != nil {
			deg = int(r.Roll(81)) - 40
		}

		if m := p.castMissile(u, sk, lvl, env, name, tgt, castOpts{angle: float64(deg) * math.Pi / 180}); m != nil {
			res.Missiles = append(res.Missiles, m)
		}
	}

	if len(res.Missiles) == 0 {
		*res = DoResult{Reason: ReasonMissile, Level: lvl}
	}
}

// doNovaRing is SRVDO_NovaMissileRing (0x5c7c50) + SKILL_ServerCreateNovaRing:
// 64 missiles of srvmissilea around the caster. The explicit velocity is the
// missile's level velocity plus calc1 (Howl: par1*(lvl-1)). The exe takes the
// 64 destinations from tables at 0x6e2580/0x6e2680; here they are evenly
// spaced on a circle (UNVERIFIED). Howl's missile applies auratargetstate on
// hit (pSrvHitFunc 17; the duration LN(par5, par6) is UNVERIFIED).
func (p *Pipeline) doNovaRing(u Unit, sk *Skill, lvl int, _ Target, env *Env, res *DoResult) {
	name := sk.SrvMissileA
	if name == "" {
		name = sk.SrvMissile
	}

	ms := p.Missiles.ByName(name)
	if ms == nil {
		*res = DoResult{Reason: ReasonMissile, Level: lvl}
		return
	}

	vel := ((ms.VelLev*lvl)/8 + ms.Vel + env.eval(sk.Calc[1])) << 8
	x, y := u.Pos()
	frames := d2calcLN(sk.Params[5], sk.Params[6], lvl)

	var hook func(m *d2missile.Missile, t d2missile.Target)
	if sk.AuraTargetState != "" && p.ApplyState != nil {
		hook = func(_ *d2missile.Missile, t d2missile.Target) { p.ApplyState(u, t, sk.AuraTargetState, frames) }
	}

	for i := 0; i < novaCount; i++ {
		a := 2 * math.Pi * float64(i) / novaCount
		tg := Target{X: x + int(math.Round(math.Cos(a)*novaRadius)), Y: y + int(math.Round(math.Sin(a)*novaRadius))}

		if m := p.castMissile(u, sk, lvl, env, name, tg, castOpts{velocity: vel, onHit: hook}); m != nil {
			res.Missiles = append(res.Missiles, m)
		}
	}
}

func d2calcLN(a, b, lvl int) int {
	if lvl < 1 {
		return 0
	}

	return a + (lvl-1)*b
}

// doMelee resolves a melee strike against the target unit: SRVDO_Attack's
// melee branch, SRVST_Bash, SRVDO_Jab and SRVST_Kick.
//
// Verified shape (skills-2.md 4.1, 4.2, 4.10, 4.14): to-hit roll with the
// skill's to-hit bonus as a percent (Kick auto-hits, result 9); damage percent
// from calc1 for Bash/Stun/Jab; weapon damage scaled by SrcDam/128; Bash adds
// calc2*256 afterwards; elemental part from EType (Stun: stun length).
// Unverified: Kick damage ((str+dex-20)/4 is behind the Kick flag in 0x648f90
// but not reached from SRVST_Kick), and that the to-hit bonus is a percent.
func (p *Pipeline) doMelee(u Unit, sk *Skill, lvl int, tgt Target, env *Env, res *DoResult) {
	if tgt.Unit == nil || !tgt.Unit.Alive() {
		*res = DoResult{Reason: ReasonTarget, Level: lvl}
		return
	}

	t := tgt.Unit
	mr := &MeleeResult{Target: t}
	res.Melee = mr

	if sk.Kick {
		mr.Hit, mr.Chance = true, 100
	} else {
		pct := 0
		if sk.SrvStFunc == stBash || sk.SrvDoFunc == doJab {
			pct = sk.ToHitBonus(env, lvl)
		}

		mr.Hit, mr.Chance, mr.Roll = d2combat.RollToHit(u.Roller(), d2combat.ToHitInput{
			AttackRating: u.AttackRating(), Defense: t.Defense(false),
			AttackerLevel: u.Level(), DefenderLevel: t.Level(), AttackRatingPct: pct,
		})
	}

	if !mr.Hit {
		return
	}

	dmg := d2combat.Damage{Result: d2combat.ResultHit, HitClass: int32(sk.HitClass)}

	switch {
	case sk.Kick:
		k := u.Stat("strength") + u.Stat("dexterity") - 20
		if k < 1 {
			k = 1
		}

		dmg.Physical = int32(k/4) << 8
	default:
		wmin, wmax := u.WeaponDamage()
		lo := int32(wmin) << 8
		hi := int32(wmax) << 8

		if lo < 1<<8 {
			lo = 1 << 8
		}

		if hi < lo+1<<8 {
			hi = lo + 1<<8
		}

		ph := lo + int32(u.Roller().Roll(hi-lo))

		if sk.SrvStFunc == stBash || sk.SrvDoFunc == doJab {
			ph += int32(mulDiv(int(ph), env.eval(sk.Calc[1]), 100))
		}

		if sk.SrvStFunc == stBash || sk.SrvDoFunc == doJab || sk.SrvDoFunc == doAttack {
			ph = d2combat.ScaleBySrcDam(ph, uint8(sk.SrcDam))
		}

		if ph < 0 {
			ph = 0
		}

		dmg.Physical = ph
	}

	// elemental / stun part of the skill
	if sk.EType != "" {
		d := sk.Descriptor(env, lvl, 0, 0, 0)
		el := d.Roll(u.Roller())
		dmg.Fire, dmg.Lightning, dmg.Magic, dmg.Cold, dmg.ColdLen = el.Fire, el.Lightning, el.Magic, el.Cold, el.ColdLen
		dmg.Poison, dmg.PoisonLen, dmg.StunLen, dmg.FreezeLen = el.Poison, el.PoisonLen, el.StunLen, el.FreezeLen
	}

	if !sk.Kick {
		dmg.ApplyStrike(u.Roller(), d2combat.StrikeInput{
			CriticalChance: u.Stat("passive_critical_strike"), DeadlyChance: u.Stat("item_deadlystrike"),
		})
	}

	if sk.SrvStFunc == stBash {
		dmg.Physical += int32(env.eval(sk.Calc[2])) << 8
	}

	mr.Damage = dmg
	mr.Total = dmg.SumTotal(true)
}

// Implemented reports whether the pipeline can run a skill: passives are not
// cast; skills with a do function need it to be one of the ported ones, skills
// without one need a generic srvmissile.
func Implemented(sk *Skill) bool {
	if sk == nil || sk.Passive {
		return false
	}

	switch sk.SrvDoFunc {
	case 0:
		return sk.SrvMissile != ""
	case doAttack, doMelee, doThrow, doLHThrow, doInner, doJab, doCharged, doFrozenAr, doStatic:
		return true
	case doNova:
		return sk.SrvMissileA != "" || sk.SrvMissile != ""
	}

	return false
}

// String describes a result for logs.
func (r *MeleeResult) String() string {
	return fmt.Sprintf("hit=%v chance=%d roll=%d total=%.2f", r.Hit, r.Chance, r.Roll, float64(r.Total)/256)
}
