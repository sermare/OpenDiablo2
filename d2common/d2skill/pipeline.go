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
	ReasonNoCorpse = "no_corpse" // corpse skill without a corpse at the aim point
)

// srvst/srvdo function ids used by the implemented skills (skills.txt columns).
const (
	stAttack   = 1
	stKick     = 2
	stRanged   = 4 // SRVST_RangedAmmoCheck
	stStrafe   = 8 // SRVST_008_Strafe (ammo check as well)
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
	// Corpse is set when a corpse lies near the aim point (corpse skills:
	// Raise Skeleton, Corpse Explosion, Revive...). CX, CY is its subtile,
	// CorpseID an engine handle, CorpseHP its maximum life and CorpseKey its
	// monstats key.
	Corpse    bool
	CX, CY    int
	CorpseID  string
	CorpseHP  int
	CorpseKey string
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

	// Near lists the living enemies within a Chebyshev radius of a subtile
	// (area melee, chain lightning, Strafe). Optional: without it those
	// skills only affect the aimed target.
	Near func(x, y, radius int) []Foe
	// Walkable reports whether a subtile can be stood on (Teleport, Leap).
	// Optional: without it every cell is walkable.
	Walkable func(x, y int) bool
	// After runs fn after that many frames (staggered bursts such as Inferno).
	After func(frames int, fn func())
}

// Foe is an enemy unit near a point.
type Foe struct {
	Target d2missile.Target
	X, Y   int
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
	case sk.SrvStFunc == stRanged, sk.SrvStFunc == stStrafe, sk.SrvStFunc == stThrow:
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

	if sk.TargetCorpse && !tgt.Corpse {
		return StartResult{Reason: ReasonNoCorpse, Level: lvl}
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

	// ---- class skills (see class.go) ----

	// Origin of an area effect: "self" (the caster) or "aim" (the aim point).
	Origin string
	X, Y   int // aim point or destination, subtile
	// TargetState / TargetStats are the state and stats an area effect or aura
	// puts on enemies (curses, Holy Freeze, Conviction...).
	TargetState string
	TargetStats []StatMod
	// Stack > 0 makes the state a counter that grows by one per cast up to
	// Stack (assassin charges, Frenzy, Maul).
	Stack int
	// Desc is the damage an area_hit / strikes / storm effect deals to every
	// enemy it reaches; SkillName names the skill in logs.
	Desc      *d2missile.DamageDesc
	SkillName string
	// Delay is the frames before the effect lands.
	Delay int
	// Strikes lists the points of "strikes" (Blizzard, Firestorm...).
	Strikes []Strike
	// Interval, Mode and Missile configure storms, auras and trails: Interval
	// is frames between pulses; Mode "nearest" (one enemy in range), "scatter"
	// (random point) or "aura" (every enemy in range).
	Interval int
	Mode     string
	Missile  string
	// Summon describes a summon / trap / wall.
	Summon *SummonOrder
	// Corpse used by corpse skills (Corpse Explosion consumes it).
	CorpseID string
	CorpseHP int
	// SelfDamagePct: percent of own maximum life the caster loses (Sacrifice).
	SelfDamagePct int
	// Heal is a one-off life gain in whole points.
	Heal int
	// Level and SkillID of the casting skill.
	Level, SkillID int
	// Dist is the maximum distance of a move, subtiles.
	Dist int
}

// Strike is one delayed hit of a "strikes" effect.
type Strike struct {
	Delay  int
	X, Y   int
	Radius int
}

// SummonOrder tells the engine what to create.
type SummonOrder struct {
	// Key is the monstats key; PetType the group that PetMax limits.
	Key, PetType, Mode string
	// Count to create now and Max alive of that PetType.
	Count, Max int
	// Kind: "minion" (follows and fights), "trap" (stationary, fires),
	// "totem" (stationary aura), "wall" (stationary blocker).
	Kind string
	// Frames the summon lasts (0 = until it dies).
	Frames int
	// TrapSkill is the monster skill a trap or totem uses (sumskill1).
	TrapSkill string
	// Stats from the skill's aurastat columns (damagepercent, tohit, armorclass).
	Stats []StatMod
	// HP is an extra life bonus (percent for golems/walls, flat for walls).
	HPPct, HPFlat int
	// X, Y where to place it (aim point, corpse...).
	X, Y int
	// Corpse to consume (id), "" if none.
	CorpseID string
	// UseCorpseType summons the corpse's own monster type (Revive).
	UseCorpseType bool
	// Damage descriptor for traps (the trap skill's own damage).
	Desc *d2missile.DamageDesc
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
	// Melees is every strike of a multi-hit or area melee skill (Melee is the
	// first).
	Melees  []*MeleeResult
	Effects []Effect
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

// runDo dispatches the srvdofunc through the table in class.go.
func (p *Pipeline) runDo(u Unit, sk *Skill, lvl int, tgt Target, env *Env, res *DoResult) {
	if fn := doTable[sk.SrvDoFunc]; fn != nil {
		fn(&cast{p: p, u: u, sk: sk, lvl: lvl, tgt: tgt, env: env, res: res})
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

	return p.passiveStats(u, sk, lvl)
}

// SummonPassives evaluates the passivestat columns of a summoning skill at
// the caster's level (Raise Skeleton's maxhp from Skeleton Mastery, the golems'
// and druid pets' damage/tohit...). Unlike PassiveStats the skill itself is
// not a passive: the columns describe the minion, not the caster.
func (p *Pipeline) SummonPassives(u Unit, skillID int) []StatMod {
	sk := p.Skills.ByID(skillID)
	if sk == nil {
		return nil
	}

	lvl := u.SkillLevel(skillID)
	if lvl < 1 {
		return nil
	}

	return p.passiveStats(u, sk, lvl)
}

func (p *Pipeline) passiveStats(u Unit, sk *Skill, lvl int) []StatMod {
	env := p.env(sk, lvl, u)

	var out []StatMod

	for i := 1; i <= 5; i++ {
		if sk.PassiveStat[i] != "" {
			out = append(out, StatMod{Stat: sk.PassiveStat[i], Value: env.eval(sk.PassiveCalc[i])})
		}
	}

	return out
}

// TruePassiveMod is one stat of a true passive. Param is the passiveitype
// (weapon type) the stat is keyed to, "" when it is not weapon-keyed.
type TruePassiveMod struct {
	Stat  string
	Value int
	Param string
}

// TruePassiveStats returns the stats the game applies for a skill the unit
// has, the way the skill-change / join path does (0x648130, verified): the
// gate is the passivestate column (> 0), NOT the passive flag, so Resist
// Fire/Cold/Lightning and Blessed Aim count too. passivestat1..5 are
// evaluated with passivecalc1..5 at the skill's total level; the stat list
// is removed at level 0. passiveitype is not a gate: the exe stores it as the
// stat's param (layer), so a mastery's stats exist always and the combat code
// reads the entry for the equipped weapon type (the lookup side is unverified).
// Unverified: slot values of 0 are kept here (the aura writer 0x5c4d70 skips
// them); the removal when the unit has the aurastate (+0x80) is not modelled.
func (p *Pipeline) TruePassiveStats(u Unit, skillID int) []TruePassiveMod {
	sk := p.Skills.ByID(skillID)
	lvl := u.SkillLevel(skillID)

	if sk == nil || lvl < 1 || sk.PassiveState == "" {
		return nil
	}

	env := p.env(sk, lvl, u)

	var out []TruePassiveMod

	for i := 1; i <= 5; i++ {
		if sk.PassiveStat[i] == "" {
			break // the exe stops at the first invalid stat id
		}

		out = append(out, TruePassiveMod{Stat: sk.PassiveStat[i], Value: env.eval(sk.PassiveCalc[i]), Param: sk.PassiveIType})
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

	stationary bool
	hitEvery   int
	scalePct   int
	home       d2missile.Target
	rangeLife  int
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
	desc.DamagePct = int32(u.Stat("damagepercent"))

	x, y := u.Pos()
	sx, sy := float64(x)+0.5, float64(y)+0.5

	if o.hasStart {
		sx, sy = o.startX, o.startY
	}

	dx, dy := float64(tgt.X)+0.5, float64(tgt.Y)+0.5
	if tgt.Unit != nil {
		dx, dy = float64(tgt.UX)+0.5, float64(tgt.UY)+0.5
	}

	// Area hit functions without a table radius (sHitPar1 < 1) take it from the
	// casting skill (verified): hit function 1 (0x5a7500) from calc1 (skills
	// record +0x138), hit function 14 (Meteor, 0x5a8680) from aurarangecalc
	// (+0x64); Meteor's flames last Param3 + (level-1)*Param4 frames (record
	// +0x150 / +0x154, lifetime flag 0x8000).
	var areaRadius, hitSubRange int

	if ms.SHitPar[0] < 1 {
		switch ms.SrvHitFunc {
		case 1:
			areaRadius = env.eval(sk.Calc[1])
		case 14:
			areaRadius = env.eval(sk.AuraRangeCalc)
		}
	}

	if ms.SrvHitFunc == 14 {
		hitSubRange = sk.Params[3] + (lvl-1)*sk.Params[4]
	}

	m, err := p.Sim.Create(d2missile.CreateParams{
		Spec: ms, Owner: p.owner(u), SkillID: sk.ID, Level: lvl, Damage: desc,
		AreaRadius: areaRadius, HitSubRange: hitSubRange,
		X: sx, Y: sy, DestX: dx, DestY: dy, Angle: o.angle, Velocity: o.velocity, ClampToDest: o.clamp || sk.Lob,
		// the missile rolls its pierce charges (stat 0x148) from skill_pierce +
		// item_pierce at creation (0x59d4e0, verified)
		PierceChance: u.Stat("skill_pierce") + u.Stat("item_pierce"), OnHit: o.onHit,
		Stationary: o.stationary, HitEvery: o.hitEvery, ScalePct: o.scalePct, Home: o.home, Range: o.rangeLife,
	})
	if err != nil {
		return nil
	}

	return m
}

// doChargedBolt is SRVDO_ChargedBolt (0x5c73a0; spot check against the binary:
// the count calc1, the creation flags 0x21 and the per-bolt post-create
// callback 0x5c7340 are confirmed, the missile is the progressive missile of
// the skill rather than always srvmissilea): calc1 bolts of srvmissilea,
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

// String describes a result for logs.
func (r *MeleeResult) String() string {
	return fmt.Sprintf("hit=%v chance=%d roll=%d total=%.2f", r.Hit, r.Chance, r.Roll, float64(r.Total)/256)
}

// AuraRange evaluates a skill's aurarangecalc for a caster (radius of the
// splash of area missiles such as Glacial Spike).
func (p *Pipeline) AuraRange(u Unit, skillID int) int {
	sk := p.Skills.ByID(skillID)
	if sk == nil {
		return 0
	}

	return p.env(sk, u.SkillLevel(skillID), u).eval(sk.AuraRangeCalc)
}

// CastTrap fires a missile of a trap (a sentry) from its own position with
// the damage of the skill that created it, at the level the caster has in it.
func (p *Pipeline) CastTrap(u Unit, skillID int, missile string, fromX, fromY int, tgt Target) *d2missile.Missile {
	sk := p.Skills.ByID(skillID)
	if sk == nil {
		return nil
	}

	lvl := u.SkillLevel(skillID)

	return p.castMissile(u, sk, lvl, p.env(sk, lvl, u), missile, tgt,
		castOpts{hasStart: true, startX: float64(fromX) + 0.5, startY: float64(fromY) + 0.5})
}
