package d2skill

import (
	"math"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2combat"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2missile"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2path"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2state"
)

// cast is the context of one do function call.
type cast struct {
	p   *Pipeline
	u   Unit
	sk  *Skill
	lvl int
	tgt Target
	env *Env
	res *DoResult
}

// doFn is a server do function (SRVDO_*, skills-2.md section 9).
type doFn func(c *cast)

// doTable maps a skills.txt srvdofunc to the implementation. Entries cite the
// function name of the notes; "U" marks behaviour that was inferred from the
// skills.txt columns and the documented in-game effect, not read from the
// binary. The handlers share the helpers below (strike, castN, area effects,
// state builders, summon orders).
var doTable map[int]doFn

func init() {
	doTable = map[int]doFn{
		doAttack:   doAttackFn,
		doMelee:    doMeleeFn,
		doThrow:    doThrowFn,
		doLHThrow:  doThrowFn,
		doInner:    doInnerFn,
		doJab:      doMeleeFn,
		doCharged:  func(c *cast) { c.p.doChargedBolt(c.u, c.sk, c.lvl, c.tgt, c.env, c.res) },
		doFrozenAr: doArmorFn,
		doStatic: func(c *cast) {
			c.res.Effects = append(c.res.Effects, Effect{
				Kind: "area_damage", Radius: c.env.eval(c.sk.AuraRangeCalc), Filter: c.sk.AuraFilter,
				Pct: c.env.eval(c.sk.Calc[1]), MinDamage: c.env.eval(c.sk.Calc[2]) /* raw 8.8, see StaticFieldDamage */, FloorPct: c.p.Opt.StaticFieldMinPct,
				EType: c.sk.EType, ELen: c.sk.ElemLen(c.env, c.lvl),
			})
		},
		doNova: func(c *cast) { c.p.doNovaRing(c.u, c.sk, c.lvl, c.tgt, c.env, c.res) },

		// Barbarian
		9:  doStrikeBuffFn, // Frenzy (SRVDO_009_Frenzy) and, with 120, Maul / Feral Rage
		68: doShoutFn,      // Shout, Battle Orders, Battle Command, War Cry
		70: doDoubleSwingFn,
		71: doTauntFn, // Taunt
		74: doDoubleThrowFn,
		77: doLeapFn,
		78: doLeapAttackFn,

		// Amazon
		8:  doFanFn,    // Multiple Shot, Teeth, Shock Wave (SRVDO_008_MultipleShot)
		10: doHomingFn, // Guided Arrow, Bone Spirit
		11: doChargedStrikeFn,
		12: doStrafeFn,
		13: doMultiHitFn, // Zeal, Fury, Fend
		14: doLightningStrikeFn,

		// Sorceress
		19: doStreamFn, // Inferno, Arctic Blast
		23: doBlazeFn,  // Blaze, Energy Shield
		24: doFireWallFn,
		25: doArmorFn, // Enchant (a timed self state)
		26: doChainFn,
		27: doTeleportFn,
		28: doRainFn, // Meteor, Blizzard
		29: doStormFn,

		// Necromancer
		30: doCurseFn, // Amplify Damage, Dim Vision, Weaken, Iron Maiden, Terror, Life Tap, Decrepify, Lower Resist
		31: doSummonFn,
		32: doMeleeFn, // Poison Dagger: melee, poison from EType
		33: doPsychicHammerFn,
		55: doCorpseExplosionFn,
		56: doSummonFn, // golems
		58: doSummonFn, // Revive
		59: doCurseFn,  // Attract
		60: doWallFn,
		61: doCurseFn, // Confuse
		62: doWallFn,  // Bone Prison
		63: doCorpseExplosionFn,

		// Paladin
		64:  doSacrificeFn,
		65:  doAuraFn,
		66:  doAuraFn,
		67:  doChargeFn,
		73:  doBlessedHammerFn,
		80:  doFistFn,
		81:  doAuraFn,
		82:  doAuraFn,
		150: doSmiteFn,

		// Druid
		114: doSummonFn, // Raven
		116: doArmorFn,  // Werewolf / Werebear: a timed form state
		117: doFirestormFn,
		118: doTwisterFn,
		119: doSummonFn, // Oak Sage, Heart of Wolverine, Spirit of Barbs, spirit wolves, grizzly
		120: doStrikeBuffFn,
		121: doRabiesFn,
		123: doVolcanoFn,
		124: doStormFn, // Hurricane, Armageddon

		// Assassin
		34: doChargeUpFn, // Tiger Strike, Cobra Strike, Royal Strike
		35: doChargeUpFn, // Fists of Fire, Claws of Thunder, Blades of Ice
		42: doDragonFn,   // Dragon Talon
		43: doMineFn,     // Shock Web
		44: doSummonFn,   // Blade Sentinel
		45: doSummonFn,   // sentries
		46: doDragonFn,   // Dragon Claw
		47: doCloakFn,    // Cloak of Shadows
		49: doSummonFn,   // Shadow Warrior / Master
		50: doDragonFn,   // Dragon Tail
		51: doMindBlastFn,
		54: doArmorFn, // Blade Shield
	}
}

// Implemented reports whether the pipeline can run a skill: passives are not
// cast; skills with a do function need it to be a ported one, skills without
// one need a generic srvmissile.
func Implemented(sk *Skill) bool {
	if sk == nil || sk.Passive {
		return false
	}

	if sk.SrvDoFunc == 0 {
		return sk.SrvMissile != ""
	}

	if sk.SrvDoFunc == doNova {
		return sk.SrvMissileA != "" || sk.SrvMissile != ""
	}

	return doTable[sk.SrvDoFunc] != nil
}

// ---- helpers ----

func (c *cast) fail(reason string) { *c.res = DoResult{Reason: reason, Level: c.lvl} }

func (c *cast) calc(i int) int { return c.env.eval(c.sk.Calc[i]) }

// aim is the aim point: the target unit when there is one.
func (c *cast) aim() (int, int) {
	if c.tgt.Unit != nil {
		return c.tgt.UX, c.tgt.UY
	}

	return c.tgt.X, c.tgt.Y
}

func (c *cast) missileName() string {
	if c.sk.SrvMissileA != "" {
		return c.sk.SrvMissileA
	}

	return c.sk.SrvMissile
}

// castM creates a missile of the skill and records it.
func (c *cast) castM(name string, o castOpts) *d2missile.Missile {
	m := c.p.castMissile(c.u, c.sk, c.lvl, c.env, name, c.tgt, o)
	if m != nil {
		c.res.Missiles = append(c.res.Missiles, m)
	}

	return m
}

func (c *cast) effect(e Effect) {
	e.Level, e.SkillID, e.SkillName = c.lvl, c.sk.ID, c.sk.Name
	c.res.Effects = append(c.res.Effects, e)
}

// statsOf evaluates aurastat1..6 of a skill.
func statsOf(env *Env, sk *Skill) []StatMod {
	var out []StatMod

	for i := 1; i <= 6; i++ {
		if sk.AuraStat[i] != "" {
			out = append(out, StatMod{Stat: sk.AuraStat[i], Value: env.eval(sk.AuraStatCalc[i])})
		}
	}

	return out
}

// desc is the skill's damage descriptor with the caster's weapon.
func (c *cast) desc() *d2missile.DamageDesc {
	wmin, wmax := c.u.WeaponDamage()
	if wmin < 1 {
		wmin = 1
	}

	if wmax < wmin+1 {
		wmax = wmin + 1
	}

	d := c.sk.Descriptor(c.env, c.lvl, wmin, wmax, c.u.Stat(masteryStat[c.sk.EType]))
	d.DamagePct = int32(c.u.Stat("damagepercent") + missileMastery(c.u, c.sk))

	return &d
}

// fanAngles returns n angles (radians) spaced step degrees apart and centred
// on zero.
func fanAngles(n int, stepDeg float64) []float64 {
	out := make([]float64, n)
	for i := range out {
		out[i] = (float64(i) - float64(n-1)/2) * stepDeg * math.Pi / 180
	}

	return out
}

func (c *cast) rollN(n int32) int {
	if r := c.u.Roller(); r != nil && n > 0 {
		return int(r.Roll(n))
	}

	return 0
}

// ---- melee ----

// strikeOpt tunes one melee strike.
type strikeOpt struct {
	pct       int   // percent added to the weapon damage (skill part)
	toHitPct  int   // attack rating percent bonus
	autoHit   bool  // no to-hit roll
	skillElem bool  // add the skill's own elemental damage (EType)
	flat      int32 // 8.8 physical added after the percent
	toMagic   bool  // convert physical damage to magic (Berserk)
	// elemPct adds fire, cold and lightning damage equal to that percent of
	// the physical damage (Vengeance, U: added rather than converted).
	elemPct [3]int
}

func rollRange(r d2combat.Roller, lo, hi int32) int32 {
	if hi <= lo || r == nil {
		return lo
	}

	return lo + int32(r.Roll(hi-lo))
}

// AROperandAdjuster is implemented by a Unit that applies the attack rating
// operands of 0x57b8b0 (stats 0x73, 0x74, 0x7b, 0x7c) before the to-hit roll.
type AROperandAdjuster interface {
	AdjustAROperands(t d2missile.Target, ar, def int) (newAR, newDef int)
}

// strike resolves one melee strike of a skill against a target (the shared
// core of SRVDO_Attack / SRVDO_MeleeSkillResolveHit, skills-2.md 4.1, 4.2,
// 4.10, 4.14): to-hit roll, weapon damage plus percent, SrcDam scaling,
// strike chances, then the hero's added elemental damage (Enchant, stats
// firemindam...) and the skill's own elemental part.
func (p *Pipeline) strike(u Unit, sk *Skill, lvl int, t d2missile.Target, env *Env, o strikeOpt) *MeleeResult {
	mr := &MeleeResult{Target: t}

	if sk.Kick || o.autoHit {
		mr.Hit, mr.Chance = true, 100
	} else {
		ar, def := u.AttackRating(), t.Defense(false)
		if adj, ok := u.(AROperandAdjuster); ok {
			ar, def = adj.AdjustAROperands(t, ar, def) // 0x57b8b0, player attackers only
		}

		mr.Hit, mr.Chance, mr.Roll = d2combat.RollToHit(u.Roller(), d2combat.ToHitInput{
			AttackRating: ar, Defense: def,
			AttackerLevel: u.Level(), DefenderLevel: t.Level(), AttackRatingPct: o.toHitPct + u.Stat("item_tohit_percent") + masteryOf(u, MasteryToHit, sk),
		})
	}

	if !mr.Hit {
		return mr
	}

	r := u.Roller()
	dmg := d2combat.Damage{Result: d2combat.ResultHit, HitClass: int32(sk.HitClass)}

	if sk.Kick {
		k := u.Stat("strength") + u.Stat("dexterity") - 20
		if k < 1 {
			k = 1
		}

		dmg.Physical = int32(k/4) << 8
	} else {
		wmin, wmax := u.WeaponDamage()
		lo, hi := int32(wmin)<<8, int32(wmax)<<8

		if lo < 1<<8 {
			lo = 1 << 8
		}

		if hi < lo+1<<8 {
			hi = lo + 1<<8
		}

		ph := rollRange(r, lo, hi)
		ph += int32(mulDiv(int(ph), o.pct+u.Stat("damagepercent")+masteryOf(u, MasteryDamage, sk), 100))
		ph = d2combat.ScaleBySrcDam(ph, uint8(sk.SrcDam))
		ph += o.flat

		if ph < 0 {
			ph = 0
		}

		dmg.Physical = ph
	}

	if sk.EType != "" && o.skillElem {
		d := sk.Descriptor(env, lvl, 0, 0, u.Stat(masteryStat[sk.EType]))
		el := d.Roll(r)
		dmg.Fire, dmg.Lightning, dmg.Magic, dmg.Cold, dmg.ColdLen = el.Fire, el.Lightning, el.Magic, el.Cold, el.ColdLen
		dmg.Poison, dmg.PoisonLen, dmg.StunLen, dmg.FreezeLen = el.Poison, el.PoisonLen, el.StunLen, el.FreezeLen
		dmg.Burn, dmg.BurnLen = el.Burn, el.BurnLen
	}

	// elemental damage added to the weapon by states / items (Enchant: stats
	// firemindam/firemaxdam, whole points)
	addElem := func(minStat, maxStat string, into *int32) {
		if hi := u.Stat(maxStat); hi > 0 {
			*into += rollRange(r, int32(u.Stat(minStat))<<8, int32(hi)<<8)
		}
	}
	addElem("firemindam", "firemaxdam", &dmg.Fire)
	addElem("lightmindam", "lightmaxdam", &dmg.Lightning)
	addElem("magicmindam", "magicmaxdam", &dmg.Magic)

	if hi := u.Stat("coldmaxdam"); hi > 0 {
		addElem("coldmindam", "coldmaxdam", &dmg.Cold)

		if dmg.ColdLen == 0 {
			dmg.ColdLen = int32(u.Stat("coldlength"))
		}
	}

	if o.elemPct != [3]int{} {
		dmg.Fire += int32(mulDiv(int(dmg.Physical), o.elemPct[0], 100))
		dmg.Cold += int32(mulDiv(int(dmg.Physical), o.elemPct[1], 100))
		dmg.Lightning += int32(mulDiv(int(dmg.Physical), o.elemPct[2], 100))
	}

	// stun on hit from states (Maul: stunlength, frames)
	if st := u.Stat("stunlength"); st > 0 && !sk.Kick {
		dmg.StunLen += int32(st)
	}

	if o.toMagic {
		dmg.Magic += dmg.Physical
		dmg.Physical = 0
	}

	if !sk.Kick {
		dmg.ApplyStrike(r, d2combat.StrikeInput{
			WeaponChance: masteryOf(u, MasteryCrit, sk), CriticalChance: u.Stat("passive_critical_strike"), DeadlyChance: u.Stat("item_deadlystrike"),
		})
	}

	mr.Damage = dmg
	mr.Total = dmg.SumTotal(false)

	return mr
}

// meleeOpt builds the strike options of the plain melee skills: the percent
// from calc1 for the skills whose start function is Bash, Jab, Impale or
// Berserk (verified for Bash and Jab, the others by their skills.txt
// description: calc1 is "damage +x%"), the Vengeance elemental percents.
func (c *cast) meleeOpt() strikeOpt {
	sk := c.sk
	o := strikeOpt{toHitPct: sk.ToHitBonus(c.env, c.lvl), skillElem: true}

	switch {
	case sk.SrvStFunc == stBash, sk.SrvDoFunc == doJab, sk.SrvStFunc == 7, sk.SrvStFunc == 39:
		o.pct = c.calc(1)
	case sk.SrvStFunc == 35: // Vengeance
		o.elemPct = [3]int{c.calc(1), c.calc(2), c.calc(3)}
		o.skillElem = false
	}

	if sk.SrvStFunc == 39 { // Berserk: damage converted to magic
		o.toMagic = true
	}

	return o
}

func (c *cast) needTarget() bool {
	if c.tgt.Unit == nil || !c.tgt.Unit.Alive() {
		c.fail(ReasonTarget)
		return false
	}

	return true
}

func (c *cast) addMelee(m *MeleeResult) {
	if c.res.Melee == nil {
		c.res.Melee = m
	}

	c.res.Melees = append(c.res.Melees, m)
}

func doMeleeFn(c *cast) {
	if !c.needTarget() {
		return
	}

	m := c.p.strike(c.u, c.sk, c.lvl, c.tgt.Unit, c.env, c.meleeOpt())
	if c.sk.SrvStFunc == stBash && m.Hit {
		m.Damage.Physical += int32(c.calc(2)) << 8
		m.Total = m.Damage.SumTotal(false)
	}

	c.addMelee(m)

	// Impale (SRVST_007, VERIFIED): on a hit the held weapon may wear down, chance calc2 percent, amount
	// calc3 points (ITEM_ReduceDurabilityOrConsumeOnSkillUse 0x5d9580).
	if c.sk.SrvStFunc == stImpale && m.Hit {
		c.effect(Effect{Kind: "weapon_wear", Pct: c.calc(2), WearAmount: c.calc(3)})
	}
}

func doAttackFn(c *cast) {
	if mname := c.u.RangedWeaponMissile(); mname != "" {
		if c.castM(mname, castOpts{}) == nil {
			c.fail(ReasonMissile)
		}

		return
	}

	doMeleeFn(c)
}

func doThrowFn(c *cast) {
	name := c.u.ThrownMissile()
	if name == "" {
		c.fail(ReasonNoWeapon)
		return
	}

	if c.castM(name, castOpts{}) == nil {
		c.fail(ReasonMissile)
	}
}

// doDoubleSwingFn is SRVDO_070_DoubleSwing (U): two strikes per cast, each
// with the synergy percent of calc1.
func doDoubleSwingFn(c *cast) {
	if !c.needTarget() {
		return
	}

	for i := 0; i < 2; i++ {
		o := c.meleeOpt()
		o.pct = c.calc(1)
		c.addMelee(c.p.strike(c.u, c.sk, c.lvl, c.tgt.Unit, c.env, o))
	}
}

// doDoubleThrowFn is SRVDO_074_DoubleThrow (U): two thrown weapons.
func doDoubleThrowFn(c *cast) {
	name := c.u.ThrownMissile()
	if name == "" {
		c.fail(ReasonNoWeapon)
		return
	}

	for _, a := range fanAngles(2, 6) {
		c.castM(name, castOpts{angle: a})
	}

	if len(c.res.Missiles) == 0 {
		c.fail(ReasonMissile)
	}
}

// doMultiHitFn is SRVDO_013_Fend (Zeal / Fury / Fend share it, VERIFIED
// 0x5da910): one strike per attack animation event on a target in melee range,
// else on any enemy within melee range + 4 other than the last victim; calc1
// events in all (see Burst). With Options.EventBursts only the first event
// runs here and DoResult.Burst carries the rest.
func doMultiHitFn(c *cast) {
	if !c.needTarget() {
		return
	}

	n := c.calc(1)
	if n < 1 {
		n = 1
	}

	b := c.p.newMeleeBurst(c, n)
	b.opt.pct = c.calc(2)

	if !c.p.BurstEvent(b, c.res) && c.res.Melee == nil {
		// nobody in reach: the aimed target is struck anyway (the engine only
		// starts the skill with a target)
		c.addMelee(c.p.strike(c.u, c.sk, c.lvl, c.tgt.Unit, c.env, b.opt))
		b.left = 0
	}

	if c.p.Opt.EventBursts {
		if !b.Done() {
			c.res.Burst = b
		}

		return
	}

	c.p.RunBurst(b, c.res)
}

// foes lists enemies near a point; the pipeline without Near knows none.
func (p *Pipeline) foes(x, y, radius int) []Foe {
	if p.Near == nil {
		return nil
	}

	return p.Near(x, y, radius)
}

// doStrikeBuffFn: Frenzy (do 9) and Maul / Feral Rage (do 120). One melee
// strike with calc1 percent, then a timed, stacking self state from the
// aurastat columns (U: stacks up to 3 hits; the duration is auralencalc, 6 s when that is zero).
func doStrikeBuffFn(c *cast) {
	if !c.needTarget() {
		return
	}

	o := c.meleeOpt()
	o.pct = c.calc(1)
	m := c.p.strike(c.u, c.sk, c.lvl, c.tgt.Unit, c.env, o)
	c.addMelee(m)

	if !m.Hit {
		return
	}

	frames := c.env.eval(c.sk.AuraLenCalc)
	if frames <= 0 {
		frames = 150
	}

	// U: the bonus stacks up to 3 hits
	stack := 3

	c.effect(Effect{Kind: "self_state", State: c.sk.AuraState, Frames: frames, Stats: statsOf(c.env, c.sk), Stack: stack})
}

// doSacrificeFn is SRVDO_064_Sacrifice (U): a strike with calc1 percent damage
// that costs the caster calc2 percent of its maximum life.
func doSacrificeFn(c *cast) {
	if !c.needTarget() {
		return
	}

	o := c.meleeOpt()
	o.pct = c.calc(1)
	m := c.p.strike(c.u, c.sk, c.lvl, c.tgt.Unit, c.env, o)
	c.addMelee(m)

	if m.Hit {
		c.effect(Effect{Kind: "self_damage", SelfDamagePct: c.calc(2)})
	}
}

// doChargedStrikeFn is SRVDO_011_ChargedStrike (0x5da4a0, VERIFIED): a Power-Strike style melee hit, then
// calc1 charged bolts are created AT THE TARGET whether or not the hit landed (the exe never tests the
// outcome). Their direction is the point mirrored through the target (2*target - self); the per-bolt
// per-bolt hook (0x5c7340) reseeds each bolt with destX+index (ChargedBoltAim); scatter width UNVERIFIED.
func doChargedStrikeFn(c *cast) {
	if !c.needTarget() {
		return
	}

	o := c.meleeOpt()
	o.skillElem = false
	c.addMelee(c.p.strike(c.u, c.sk, c.lvl, c.tgt.Unit, c.env, o))

	bolts := c.calc(1)
	ux, uy := c.u.Pos()

	for i := 0; i < bolts; i++ {
		ang := ChargedBoltAim(ux, uy, c.tgt.UX, c.tgt.UY, i)
		x, y := c.tgt.UX, c.tgt.UY
		tg := c.tgt
		tg.Unit, tg.X, tg.Y = nil, x+int(math.Round(math.Cos(ang)*10)), y+int(math.Round(math.Sin(ang)*10))

		cc := *c
		cc.tgt = tg
		cc.castM(c.missileName(), castOpts{hasStart: true, startX: float64(x) + 0.5, startY: float64(y) + 0.5})
	}
}

// doLightningStrikeFn is SRVDO_014_LightningStrike (0x5dab30, VERIFIED): resolve the hit, then, hit or
// miss, look for ONE other enemy within calc1 subtiles of the target and start a single chain bolt
// there (calc2 = bolts allowed in the chain). Which enemy the exe's area scan returns first is not
// known; the nearest to the target is used (UNVERIFIED).
func doLightningStrikeFn(c *cast) {
	if !c.needTarget() {
		return
	}

	o := c.meleeOpt()
	o.skillElem = false
	c.addMelee(c.p.strike(c.u, c.sk, c.lvl, c.tgt.Unit, c.env, o))

	var best *Foe

	bd := 1 << 30

	for _, f := range c.p.foes(c.tgt.UX, c.tgt.UY, c.calc(1)) {
		f := f
		if f.Target.ID() == c.tgt.Unit.ID() || !f.Target.Alive() {
			continue
		}

		if d := cheb(f.X-c.tgt.UX, f.Y-c.tgt.UY); d < bd {
			bd, best = d, &f
		}
	}

	if best == nil {
		return
	}

	cc := *c
	cc.tgt = Target{Unit: best.Target, UX: best.X, UY: best.Y}
	cc.castM(c.sk.SrvMissileA, castOpts{hasStart: true, startX: float64(best.X) + 0.5, startY: float64(best.Y) + 0.5, chainLeft: c.calc(2)})
}

// ---- ranged / missile families ----

// doFanFn is SRVDO_008_MultipleShot (Multiple Shot, Teeth, Shock Wave): calc1
// missiles in a fan around the aim direction (8 degrees apart, U).
//
// Read from the binary (0x5d9fe0, read-only spot check): the count is calc1
// (verified); calc2 goes to the created missile's damage word and calc3 is the
// number of "real" middle missiles: the (calc1-calc3)/2 flanking ones on each side
// are created with flag 0x10000 (probably no ammo / no damage-scaling) and only
// the middle calc3 without it. The destinations are not an angular fan but
// points spaced along a line through the aim point (the step is the
// unit->aim vector after two helpers, 0x56b0e0 and 0x56b120, that were not
// read), so the 8 degree spacing, the missile B pick (bolt when the weapon is
// not a bow, via 0x623e60) and the equal treatment of all missiles here are
// approximations.
func doFanFn(c *cast) {
	name := c.sk.SrvMissileA
	if strings.Contains(c.u.RangedWeaponMissile(), "bolt") && c.sk.SrvMissileB != "" {
		name = c.sk.SrvMissileB
	}

	n := c.calc(1)
	if n < 1 {
		n = 1
	}

	for _, a := range fanAngles(n, 8) {
		c.castM(name, castOpts{angle: a})
	}

	if len(c.res.Missiles) == 0 {
		c.fail(ReasonMissile)
	}
}

// doHomingFn is SRVDO_010_GuidedArrow (Guided Arrow, Bone Spirit): a missile
// that steers onto the aimed unit.
func doHomingFn(c *cast) {
	if c.castM(c.missileName(), castOpts{home: c.tgt.Unit}) == nil {
		c.fail(ReasonMissile)
	}
}

// StrafeBurst is the number of arrows of a Strafe cast (SRVST_008, 0x5d9840, VERIFIED): with m =
// min(calc1, calc3), it is min(calc1, max(found, m)) where found is the number of enemies in the scan.
func StrafeBurst(calc1, calc3, found int) int {
	m := calc3
	if calc1 < m {
		m = calc1
	}

	if found > m {
		m = found
	}

	if calc1 < m {
		return calc1
	}

	return m
}

// doStrafeFn is SRVDO_012_Strafe: one arrow per animation event at a (new) enemy within aurarangecalc of
// the archer, the burst length from StrafeBurst. The exe spreads the arrows over attack events and picks
// the next enemy as any other than the previous one; here the burst is fired in one call, alternating over
// the enemies found, and with a single enemy only one arrow flies (the rescan excluding it ends the burst).
func doStrafeFn(c *cast) {
	rng := c.env.eval(c.sk.AuraRangeCalc)
	if rng <= 0 {
		rng = 14
	}

	ux, uy := c.u.Pos()
	foes := c.p.foes(ux, uy, rng)
	n := StrafeBurst(c.calc(1), c.calc(3), len(foes))

	if n < 1 {
		n = 1
	}

	if len(foes) == 1 && n > 1 {
		n = 1
	}

	for i := 0; i < n; i++ {
		tg := c.tgt
		if len(foes) > 0 {
			f := foes[i%len(foes)]
			tg.Unit, tg.UX, tg.UY = f.Target, f.X, f.Y
		}

		cc := *c
		cc.tgt = tg
		cc.castM(c.sk.SrvMissileA, castOpts{})
	}

	if len(c.res.Missiles) == 0 {
		c.fail(ReasonMissile)
	}
}

// doChainFn is SRVDO_026_ChainLightning (0x5c8320, VERIFIED): the cast stores
// calc1 in the bolt's data field 0x28 (the number of bolts allowed, this one
// included). The jumps themselves are hit function 12 of the missile, run by the
// missile simulation (d2missile chainHit, target pick verified by an emulator
// golden): the bolt scans around itself, picks the next higher unit id and
// spawns the next bolt with the field decremented.
func doChainFn(c *cast) {
	if c.castM(c.missileName(), castOpts{chainLeft: c.calc(1)}) == nil {
		c.fail(ReasonMissile)
	}
}

func (c *cast) chainStep(x, y int, hit map[string]bool, name string, rng int, hook func(*d2missile.Missile, d2missile.Target)) {
	var best *Foe

	bd := 1 << 30

	for _, f := range c.p.foes(x, y, rng) {
		f := f
		if hit[f.Target.ID()] || !f.Target.Alive() {
			continue
		}

		if d := cheb(f.X-x, f.Y-y); d < bd {
			bd, best = d, &f
		}
	}

	if best == nil {
		return
	}

	tg := Target{Unit: best.Target, UX: best.X, UY: best.Y}
	// start two subtiles towards the next victim so the bolt does not hit the
	// one it comes from again
	sx, sy := float64(x)+0.5, float64(y)+0.5
	if d := math.Hypot(float64(best.X-x), float64(best.Y-y)); d > 0 {
		sx += float64(best.X-x) / d * 2
		sy += float64(best.Y-y) / d * 2
	}

	_ = c.p.castFrom(c.u, c.sk, c.lvl, c.env, name, tg, sx, sy, hook)
}

// chainFrom fires n bolts from a point at distinct enemies around it.
func (c *cast) chainFrom(x, y int, first d2missile.Target, name string, n, rng int) {
	hit := map[string]bool{}
	if first != nil {
		hit[first.ID()] = true
	}

	for i := 0; i < n; i++ {
		c.chainStep(x, y, hit, name, rng, nil)
	}
}

func cheb(dx, dy int) int {
	if dx < 0 {
		dx = -dx
	}

	if dy < 0 {
		dy = -dy
	}

	if dx > dy {
		return dx
	}

	return dy
}

// castFrom creates a missile from an explicit start point at a target.
func (p *Pipeline) castFrom(u Unit, sk *Skill, lvl int, env *Env, name string, tg Target, sx, sy float64,
	hook func(*d2missile.Missile, d2missile.Target)) *d2missile.Missile {
	return p.castMissile(u, sk, lvl, env, name, tg, castOpts{hasStart: true, startX: sx, startY: sy, onHit: hook})
}

// doStreamFn is SRVDO_019_Inferno (Inferno, Arctic Blast): a burst of flame
// missiles along the aim line. In the game the skill is channelled (start
// function 11 re-runs it while the button is held); one cast here is a burst of
// six flames three frames apart whose range is aurarangecalc yards (U).
func doStreamFn(c *cast) {
	name := c.missileName()
	ms := c.p.Missiles.ByName(name)

	if ms == nil {
		c.fail(ReasonMissile)
		return
	}

	rng := 0

	if y := c.env.eval(c.sk.AuraRangeCalc); y > 0 {
		step := float64(d2combat.MissileStep(d2combat.MissileVelocity(uint8(ms.Vel), uint8(ms.VelLev), c.lvl, false, 0))) / 4096
		if step > 0 {
			rng = int(math.Ceil(float64(y) / step))
		}
	}

	for i := 0; i < 6; i++ {
		if i == 0 || c.p.After == nil {
			if m := c.castM(name, castOpts{rangeLife: rng}); m == nil {
				c.fail(ReasonMissile)
				return
			}

			continue
		}

		cc := *c
		cc.res = &DoResult{}
		c.p.After(i*3, func() {
			cc.p.castMissile(cc.u, cc.sk, cc.lvl, cc.env, name, cc.tgt, castOpts{rangeLife: rng})
		})
	}
}

// doFireWallFn is SRVDO_024_FireWall: a line of stationary fire missiles across
// the aim direction. The length (8 + level subtiles) and the five hits per
// second are U. The skill's damage is listed per second (Fire Wall L1 shows
// 15-20 and has HitShift 4): each of the five hits does 1<<HitShift/5 of
// the descriptor, so a target standing in the fire loses the listed amount
// per second (U: the missile damage function 3 that does this in the game was
// not read).
func doFireWallFn(c *cast) {
	name := c.sk.SrvMissileB
	if name == "" {
		name = c.sk.SrvMissileA
	}

	ax, ay := c.aim()
	hx, hy := c.u.Pos()
	dx, dy := float64(ax-hx), float64(ay-hy)

	d := math.Hypot(dx, dy)
	if d == 0 {
		dx, dy, d = 1, 0, 1
	}

	px, py := -dy/d, dx/d // perpendicular
	half := (8 + c.lvl) / 2

	for i := -half / 2; i <= half/2; i++ {
		cx, cy := float64(ax)+0.5+px*float64(i)*2, float64(ay)+0.5+py*float64(i)*2
		tg := Target{X: int(cx), Y: int(cy)}

		if c.p.Walkable != nil && !c.p.Walkable(tg.X, tg.Y) {
			continue
		}

		if m := c.p.castMissile(c.u, c.sk, c.lvl, c.env, name, tg, castOpts{
			hasStart: true, startX: cx, startY: cy, stationary: true, hitEvery: 5, scalePct: 100 * (1 << uint(c.sk.HitShift)) / 5,
		}); m != nil {
			c.res.Missiles = append(c.res.Missiles, m)
		}
	}

	if len(c.res.Missiles) == 0 {
		c.fail(ReasonMissile)
	}
}

// doTwisterFn is SRVDO_118_Twister (Twister, Tornado): calc1 twisters fanned 20
// degrees apart (U).
func doTwisterFn(c *cast) {
	n := c.calc(1)
	if n < 1 {
		n = 1
	}

	for _, a := range fanAngles(n, 20) {
		c.castM(c.missileName(), castOpts{angle: a})
	}
}

// doRabiesFn is SRVDO_121_Rabies (0x5c6b70, verified): a bite at the target
// (to-hit roll, then the poison damage from the skill's EType columns, taken
// from the generic melee strike here) that infects it (0x5c5dc0): the target
// gets auratargetstate for the skill's elemental length, unless it already
// carries it, and the plague missile (the skill's missile) starts on it. The
// plague belongs to the infected monster and marks the caster, see
// d2missile SrvDoFunc 30 / hit function 53, which spread the infection.
func doRabiesFn(c *cast) {
	if !c.needTarget() {
		return
	}

	m := c.p.strike(c.u, c.sk, c.lvl, c.tgt.Unit, c.env, c.meleeOpt())
	c.addMelee(m)

	if !m.Hit || c.sk.AuraTargetState == "" {
		return
	}

	if st, ok := c.tgt.Unit.(d2missile.Stateful); ok && st.HasStateNamed(c.sk.AuraTargetState) {
		return
	}

	if c.p.ApplyState != nil {
		c.p.ApplyState(c.u, c.tgt.Unit, c.sk.AuraTargetState, c.sk.ElemLen(c.env, c.lvl))
	}

	ow, ok := c.tgt.Unit.(d2missile.Ownable)
	if !ok {
		return
	}

	carrier, caster := ow.AsOwner(), c.p.owner(c.u)
	c.castM(c.missileName(), castOpts{owner: &carrier, markOwner: &caster, stationary: true, hasStart: true,
		startX: float64(c.tgt.UX) + 0.5, startY: float64(c.tgt.UY) + 0.5})
}

// doBlessedHammerFn is SRVDO_073_BlessedHammer: one hammer missile. Read from
// the binary (0x5ce5a0): the missile id is the progressive missile of the skill
// (SKILL_GetProgressiveMissileId), it is created at the cursor target with flag
// 0x20 and its path is switched to path type 0xe (the spiral), and when
// FUN_00647550 (a synergy/mastery test) holds, the missile's stats 0x34 and
// 0x35 are scaled by a percent. The spiral and that scaling are not modelled
// here: the hammer flies straight.
func doBlessedHammerFn(c *cast) {
	if c.castM(c.missileName(), castOpts{}) == nil {
		c.fail(ReasonMissile)
	}
}

// doMineFn is SRVDO_043_ShockField (Shock Web, U): a stationary trap missile at
// the aim point that zaps whatever walks into it every 12 frames.
func doMineFn(c *cast) {
	ax, ay := c.aim()
	tg := Target{X: ax, Y: ay}

	m := c.p.castMissile(c.u, c.sk, c.lvl, c.env, c.sk.SrvMissileA, tg, castOpts{
		hasStart: true, startX: float64(ax) + 0.5, startY: float64(ay) + 0.5, stationary: true, hitEvery: 12,
	})
	if m == nil {
		c.fail(ReasonMissile)
		return
	}

	c.res.Missiles = append(c.res.Missiles, m)
}

// ---- area effects ----

// doShoutFn is SRVDO_068_Shout: a nova ring of the skill's missile plus the
// timed self state (Shout, Battle Orders, Battle Command, Battle Cry). A skill
// whose missile has no hit function 18 / 21 and no state (War Cry) keeps the
// older approximation: it damages and stuns the enemies around the hero.
func doShoutFn(c *cast) {
	// SRVDO_068 (0x5d6e50, verified) creates the nova ring of the skill's
	// missile (the same ring as Howl) and then applies the skill's timed state
	// to the caster. The ring's missiles carry hit function 18 (Shout, Battle
	// Orders, Battle Command: allies get the state) or 21 (Battle Cry: enemies).
	if ms := c.p.Missiles.ByName(c.missileName()); ms != nil && (ms.SrvHitFunc == 18 || ms.SrvHitFunc == 21) {
		c.p.doNovaRing(c.u, c.sk, c.lvl, c.tgt, c.env, c.res)

		if len(c.res.Missiles) == 0 {
			c.fail(ReasonMissile)
			return
		}

		if c.sk.AuraState != "" {
			c.p.doState(c.sk, c.env, c.res, "self_state")
			last := &c.res.Effects[len(c.res.Effects)-1]
			last.Level, last.SkillID, last.SkillName = c.lvl, c.sk.ID, c.sk.Name
		}

		return
	}

	if c.sk.AuraState == "" {
		d := c.desc()
		d.StunLen = int32(c.env.eval(c.sk.Calc[1]))
		if d.StunLen == 0 {
			d.StunLen = int32(d2calcLN(c.sk.Params[1], c.sk.Params[2], c.lvl))
		}

		x, y := c.u.Pos()
		c.effect(Effect{Kind: "area_hit", Origin: "self", X: x, Y: y, Radius: 8, Desc: d})

		return
	}

	c.p.doState(c.sk, c.env, c.res, "self_state")
	last := &c.res.Effects[len(c.res.Effects)-1]
	last.Level, last.SkillID, last.SkillName = c.lvl, c.sk.ID, c.sk.Name
}

// doArmorFn handles the timed self states of do functions 18 (Frozen Armor,
// Bone Armor, Holy Shield, Fade, Venom), 25 (Enchant), 47 (Cloak), 54 (Blade
// Shield) and 116 (shapeshift forms).
func doArmorFn(c *cast) {
	c.p.doState(c.sk, c.env, c.res, "self_state")
	e := &c.res.Effects[len(c.res.Effects)-1]
	e.Level, e.SkillID, e.SkillName = c.lvl, c.sk.ID, c.sk.Name

	switch c.sk.AuraState { // calc1 is the chill time of the cold armors
	case "frozenarmor", "shiverarmor", "chillingarmor":
		e.Chill = c.calc(1)
	}
}

// doInnerFn is SRVDO_InnerSight (also Slow Missiles): an area state around the caster.
func doInnerFn(c *cast) {
	c.p.doState(c.sk, c.env, c.res, "area_state")
	e := &c.res.Effects[len(c.res.Effects)-1]
	e.State, e.Radius, e.Filter = c.sk.AuraTargetState, c.env.eval(c.sk.AuraRangeCalc), c.sk.AuraFilter
	e.Level, e.SkillID, e.SkillName = c.lvl, c.sk.ID, c.sk.Name
}

// doBlazeFn is SRVDO_023 (Blaze, Energy Shield). Blaze: a timed state (dm12
// frames) during which the caster leaves fire behind (the engine drops the
// srvmissilea every 3 frames). Energy Shield: a timed state whose synthetic
// stats tell the engine to absorb calc1 percent of the damage taken with
// mana, at calc2 mana per... U: the ratio is calc2/? and is passed as is.
func doBlazeFn(c *cast) {
	c.p.doState(c.sk, c.env, c.res, "self_state")
	e := &c.res.Effects[len(c.res.Effects)-1]
	e.Level, e.SkillID, e.SkillName = c.lvl, c.sk.ID, c.sk.Name

	if c.sk.SrvMissileA != "" {
		e.Missile, e.Interval = c.sk.SrvMissileA, 3
	} else {
		e.Stats = append(e.Stats, StatMod{Stat: "x_energyshield_pct", Value: c.calc(1)},
			StatMod{Stat: "x_energyshield_ratio", Value: c.calc(2)})
	}
}

// doTeleportFn is SRVDO_027_Teleport. VERIFIED (0x5c84d0): the caster's level
// must have the levels.txt Teleport column set (0 refuses), and with the value
// 2 the cast is refused when the line to the target is blocked for mask 0x804
// (walls and objects: no teleporting through them; U: the exe's helper answers
// non-zero for a blocked line). The caster then appears at the aim point
// (SERVER_MoveUnitToLevelPosition); U: the aim point must be walkable here.
func doTeleportFn(c *cast) {
	flag := 1
	if c.p.TeleportFlag != nil {
		flag = c.p.TeleportFlag()
	}

	x, y := c.aim()

	switch {
	case flag == 0:
		c.fail(ReasonLOS)
		return
	case flag == 2 && c.p.Grid != nil:
		hx, hy := c.u.Pos()
		if clear, _ := d2path.TraceLine(c.p.Grid, 0x804, d2path.Point{X: hx, Y: hy}, d2path.Point{X: x, Y: y}); !clear {
			c.fail(ReasonLOS)
			return
		}
	}

	if c.p.Walkable != nil && !c.p.Walkable(x, y) {
		c.fail(ReasonLOS)
		return
	}

	c.effect(Effect{Kind: "move", Mode: "teleport", X: x, Y: y})
}

// leapDistance is the longest Leap or Leap Attack, subtiles (U).
const leapDistance = 18

func (c *cast) clampMove(maxDist int) (int, int, bool) {
	x, y := c.aim()
	hx, hy := c.u.Pos()

	d := cheb(x-hx, y-hy)
	if d > maxDist {
		x = hx + (x-hx)*maxDist/d
		y = hy + (y-hy)*maxDist/d
	}

	if c.p.Walkable != nil && !c.p.Walkable(x, y) {
		return x, y, false
	}

	return x, y, true
}

// doLeapFn is SRVDO_077_Leap (U): the caster jumps to the aim point (at most
// leapDistance away).
func doLeapFn(c *cast) {
	x, y, ok := c.clampMove(leapDistance)
	if !ok {
		c.fail(ReasonLOS)
		return
	}

	c.effect(Effect{Kind: "move", Mode: "leap", X: x, Y: y})
}

// doLeapAttackFn is SRVDO_078_LeapAttack (U): a leap, and on landing a strike
// against every enemy within 5 subtiles with calc1 percent damage.
func doLeapAttackFn(c *cast) {
	x, y, ok := c.clampMove(leapDistance)
	if !ok {
		c.fail(ReasonLOS)
		return
	}

	c.effect(Effect{Kind: "move", Mode: "leap", X: x, Y: y})

	o := c.meleeOpt()
	o.pct = c.calc(1)

	for _, f := range c.p.foes(x, y, 5) {
		c.addMelee(c.p.strike(c.u, c.sk, c.lvl, f.Target, c.env, o))
	}
}

// doFistFn is SRVDO_080_FistOfTheHeavens (0x5cebd0, verified): without a target
// unit nothing happens. The skill's missile (fistoftheheavensdelay) is created
// at the target, marking it; when it ends (Range frames) hit function 22
// strikes the target with the lightning and sends holy bolts to the enemies
// around. (The exe also puts the skill's srvoverlay on the target: the engine
// draws that itself.)
func doFistFn(c *cast) {
	if !c.needTarget() {
		return
	}

	x, y := float64(c.tgt.UX)+0.5, float64(c.tgt.UY)+0.5
	if c.castM(c.missileName(), castOpts{hasStart: true, startX: x, startY: y, stationary: true, mark: c.tgt.Unit}) == nil {
		c.fail(ReasonMissile)
	}
}

// ---- curses and auras ----

// doCurseFn is the cast-at-a-point curse (SRVDO_030_AmplifyDamage and 59, 61,
// 71): every enemy within aurarangecalc of the aim point gets auratargetstate
// for auralencalc frames with the aurastat stats. Curses without stats get
// synthetic ones the engine reads (d2state.Stat*): Iron Maiden calc1, Life
// Tap calc1, Dim Vision, Confuse, Attract, Taunt.
func doCurseFn(c *cast) {
	sk := c.sk
	ax, ay := c.aim()
	e := Effect{
		Kind: "area_state", Origin: "aim", X: ax, Y: ay, Radius: c.env.eval(sk.AuraRangeCalc), Filter: sk.AuraFilter,
		State: sk.AuraTargetState, Frames: c.env.eval(sk.AuraLenCalc), Stats: statsOf(c.env, sk),
	}

	if e.Radius <= 0 {
		e.Radius = 10
	}

	switch sk.AuraTargetState {
	case "ironmaiden":
		e.Stats = append(e.Stats, StatMod{d2state.StatIronMaiden, c.calc(1)})
	case "lifetap":
		e.Stats = append(e.Stats, StatMod{d2state.StatLifeTap, c.calc(1)})
	case "dimvision":
		e.Stats = append(e.Stats, StatMod{d2state.StatBlind, 1})
	case "confuse":
		e.Stats = append(e.Stats, StatMod{d2state.StatConfused, 1})
	case "attract":
		e.Stats = append(e.Stats, StatMod{d2state.StatAttract, 1})
	case "taunt":
		e.Stats = append(e.Stats, StatMod{d2state.StatTaunted, 1})
	}

	if e.State == "" {
		c.fail(ReasonNoSkill)
		return
	}

	c.effect(e)
}

// enemyFilter says whether an aurafilter selects monsters (bits 0x80 and
// 0x100 of the filters the paladin aura rows use: 73731 for friendly auras
// has neither, 42883 / 42371 / 303747 / 59270 have them; U meaning of the
// bits).
func enemyFilter(f int) bool { return f&0x180 != 0 }

// doAuraFn is SRVDO_065_Might (Might, Prayer, Resist x, Thorns, ... SRVDO_066
// Holy Fire family, 81 Holy Freeze, 82 Redemption). The aura stays on until
// another aura is chosen; the engine pulses it (Interval frames): friendly
// auras keep the self state refreshed, enemy auras put auratargetstate on the
// monsters in range, damage auras (do 66 with an EType) hurt them.
func doAuraFn(c *cast) {
	sk := c.sk
	stats := statsOf(c.env, sk)
	e := Effect{Kind: "aura", State: sk.AuraState, Radius: c.env.eval(sk.AuraRangeCalc), Filter: sk.AuraFilter,
		Interval: 25, Mode: "friendly"}

	if enemyFilter(sk.AuraFilter) {
		e.Mode = "enemy"
		e.TargetState, e.TargetStats = sk.AuraTargetState, stats

		if e.TargetState == "" {
			e.TargetState = sk.AuraState
		}

		if sk.SrvDoFunc == 66 && sk.EType != "" {
			e.Mode = "damage"
			e.Desc = c.desc()
			// VERIFIED: the hit carries the skill's result flags (0x5cd880); bit 8
			// is the knockback (Sanctuary has ResultFlags 11).
			e.Knock = sk.ResultFlags&resultKnock != 0
		}
	} else {
		e.Stats = stats
		if sk.SrvDoFunc == 66 && sk.EType == "mag" { // Sanctuary: damages undead only; U: all enemies here
			e.Mode, e.Desc = "damage", c.desc()
		}
	}

	if sk.SrvDoFunc == 65 { // friendly aura: the upkeep is the aura parameter (VERIFIED 0x5cd4b0 + 0x645ed0)
		e.Cost = int(sk.calcMana(c.lvl, false))
	}

	if sk.SrvDoFunc == 82 { // Redemption
		e.Mode = "redemption"
		e.Heal, e.Dist = c.calc(2), c.calc(3)
		e.Stack = c.calc(1)
		// VERIFIED 0x5cf410: a pulse that redeemed something pays the aura
		// parameter (lvlmana*(lvl-1)+mana) << manashift; 0 for the shipped row.
		e.Cost = int(sk.calcMana(c.lvl, false))
	}

	c.effect(e)
}

// ---- storms and rains ----

// doRainFn is SRVDO_028 (Meteor, Blizzard, Eruption; 0x5c8560, VERIFIED): the
// skill's srvmissilea is created at the aim point when the cell is free of
// walk and wall bits for the missile's size (SKILL_IsSkillTargetCellWalkable);
// everything else happens in the missile. Meteor: meteorcenter lives its Range
// and ends in hit function 14 (area damage, then meteorfire on the ground).
// Blizzard: blizzardcenter (SrvDoFunc 10) drops a blizzard1 every calc2 frames
// at a random cell within calc1 of itself. Eruption: erruption center (SrvDoFunc
// 25) likewise with another cell mask.
func doRainFn(c *cast) {
	ax, ay := c.aim()

	if c.p.Walkable != nil && !c.p.Walkable(ax, ay) {
		c.fail(ReasonLOS)
		return
	}

	if c.castM(c.missileName(), castOpts{hasStart: true, startX: float64(ax) + 0.5, startY: float64(ay) + 0.5,
		stationary: true}) == nil {
		c.fail(ReasonMissile)
	}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}

	return b
}

// doFirestormFn is SRVDO_117_Firestorm (U): fire bursts in a line from the caster
// towards the aim, 2 subtiles apart, 2 frames apart.
func doFirestormFn(c *cast) {
	ax, ay := c.aim()
	hx, hy := c.u.Pos()
	dx, dy := float64(ax-hx), float64(ay-hy)

	d := math.Hypot(dx, dy)
	if d == 0 {
		d, dx = 1, 1
	}

	n := 5 + c.calc(1)
	desc := c.desc()

	var strikes []Strike

	for i := 1; i <= n; i++ {
		t := math.Min(float64(i)*2, d+4)
		strikes = append(strikes, Strike{Delay: i * 2, X: hx + int(math.Round(dx/d*t)), Y: hy + int(math.Round(dy/d*t)), Radius: 2})
	}

	c.effect(Effect{Kind: "strikes", Origin: "self", Desc: desc, Strikes: strikes})
}

// doVolcanoFn is SRVDO_123_Volcano (0x5c60c0, VERIFIED): the skill's srvmissile
// ("volcano") is created at the aim point if the cell is free for its size,
// with a random byte from the caster's seed stored in the missile's data field
// 0x28; the missile (SrvDoFunc 28) then lobs debris around itself.
func doVolcanoFn(c *cast) {
	ax, ay := c.aim()

	if c.p.Walkable != nil && !c.p.Walkable(ax, ay) {
		c.fail(ReasonLOS)
		return
	}

	seed := uint32(c.rollN(256))

	if c.castM(c.missileName(), castOpts{hasStart: true, startX: float64(ax) + 0.5, startY: float64(ay) + 0.5,
		stationary: true, data28: seed}) == nil {
		c.fail(ReasonMissile)
	}
}

// doStormFn handles the long lasting damage fields: Thunder Storm (do 29: a
// lightning strike on a nearby enemy every Param3 frames while the state
// lasts), Hurricane and Armageddon (do 124: every Param4 frames the area
// around the caster is hit, Armageddon at random points within Param3 and
// Hurricane on every enemy in range). All U.
func doStormFn(c *cast) {
	sk := c.sk
	frames := c.env.eval(sk.AuraLenCalc)
	e := Effect{Kind: "storm", State: sk.AuraState, Frames: frames, Desc: c.desc(), Origin: "self"}

	switch sk.SrvDoFunc {
	case 29:
		e.Mode, e.Interval, e.Radius = "nearest", maxInt(sk.Params[3], 12), 20
	default:
		e.Interval, e.Radius = maxInt(sk.Params[4], 10), maxInt(c.env.eval(sk.AuraRangeCalc), 6)
		e.Mode = "scatter"

		if sk.EType == "cold" {
			e.Mode = "aura"
		}
	}

	c.effect(e)
}

// ---- assassin ----

// chargeStats are the progressive (charge) states of the builders.
var chargeStats = []string{"progressive_fire", "progressive_lightning", "progressive_cold", "progressive_damage",
	"progressive_steal", "progressive_other"}

// doChargeUpFn is SRVDO_034 / 035 (Tiger Strike, Cobra Strike, Royal Strike,
// Fists of Fire, Claws of Thunder, Blades of Ice; U): a melee strike that,
// when it hits, adds one charge (up to 3) to the progressive state for
// auralencalc frames.
func doChargeUpFn(c *cast) {
	if !c.needTarget() {
		return
	}

	o := c.meleeOpt()
	o.pct = c.calc(1)
	o.skillElem = c.sk.SrvDoFunc == 35
	m := c.p.strike(c.u, c.sk, c.lvl, c.tgt.Unit, c.env, o)
	c.addMelee(m)

	if m.Hit && c.sk.AuraState != "" {
		frames := c.env.eval(c.sk.AuraLenCalc)
		if frames <= 0 {
			frames = 375
		}

		c.effect(Effect{Kind: "self_state", State: c.sk.AuraState, Frames: frames, Stack: 3,
			Stats: []StatMod{{Stat: c.sk.AuraState, Value: 1}}})
	}
}

// doDragonFn is SRVDO_042 / 046 / 050 (Dragon Talon, Claw, Tail; U): the
// finishing moves. Talon kicks calc1 times, Claw strikes twice, Tail once and
// explodes in fire around the target. Charges the caster holds are released
// as the missiles of their builder skill (srvmissilec / srvmissileb), one
// per charge, and consumed.
func doDragonFn(c *cast) {
	if !c.needTarget() {
		return
	}

	hits := 1

	switch c.sk.SrvDoFunc {
	case 42:
		hits = maxInt(c.calc(1), 1)
	case 46:
		hits = 2
	}

	for i := 0; i < hits; i++ {
		o := c.meleeOpt()
		o.pct = c.env.eval(c.sk.Calc[1])

		if c.sk.SrvDoFunc == 42 {
			o.pct = 0
		}

		o.skillElem = false
		c.addMelee(c.p.strike(c.u, c.sk, c.lvl, c.tgt.Unit, c.env, o))
	}

	if c.sk.SrvDoFunc == 42 && c.knockRoll(c.tgt.Unit, 100) { // 0x5d4370: the last kick may knock back
		c.knock(c.tgt.Unit)
	}

	if c.sk.SrvDoFunc == 50 {
		c.effect(Effect{Kind: "area_hit", Origin: "aim", X: c.tgt.UX, Y: c.tgt.UY,
			Radius: maxInt(c.env.eval(c.sk.AuraRangeCalc), 4), Desc: c.desc()})
	}

	c.releaseCharges()
}

// releaseCharges fires the missile of every builder whose charge the caster
// holds, and clears the charges.
func (c *cast) releaseCharges() {
	for _, st := range chargeStats {
		n := c.u.Stat(st)
		if n < 1 {
			continue
		}

		b := c.p.builderOf(st)
		if b == nil {
			continue
		}

		lvl := c.u.SkillLevel(b.ID)
		if lvl < 1 {
			continue
		}

		name := b.SrvMissileC
		if name == "" {
			name = b.SrvMissileB
		}

		env := c.p.env(b, lvl, c.u)

		for i := 0; i < n && name != ""; i++ {
			if m := c.p.castMissile(c.u, b, lvl, env, name, c.tgt, castOpts{angle: float64(i-1) * 0.3}); m != nil {
				c.res.Missiles = append(c.res.Missiles, m)
			}
		}

		c.effect(Effect{Kind: "clear_state", State: st})
	}
}

// builderOf finds the charge-up skill whose aurastate is the given progressive state.
func (p *Pipeline) builderOf(state string) *Skill {
	for _, id := range p.Skills.IDs() {
		if sk := p.Skills.ByID(id); sk.AuraState == state && sk.SrvDoFunc == 35 {
			return sk
		}
	}

	return nil
}

// ---- summons ----

// doSummonFn builds the summon order of do functions 31 (Raise Skeleton),
// 56/57 (golems), 58 (Revive), 114 (Raven), 119 (druid pets and totems), 44
// and 45 (assassin sentries), 49 (Shadow Warrior). The engine creates the
// monsters (d2monsters minions). Counts follow petmax: the caster tops up to
// that many (Raven fills all, other skills add one per cast).
func doSummonFn(c *cast) {
	sk := c.sk
	if sk.Summon == "" && sk.SrvDoFunc != 58 {
		c.fail(ReasonNoSkill)
		return
	}

	o := &SummonOrder{Key: sk.Summon, PetType: sk.PetType, Mode: sk.SumMode, Count: 1, Kind: "minion",
		Max: c.env.eval(sk.PetMax), Stats: statsOf(c.env, sk), TrapSkill: sk.SumSkill[1], HPPct: c.calc(1)}

	if o.Max < 1 {
		o.Max = 1
	}

	// VERIFIED (SRVDO_114 / 115 / 119): the monster's level is skills.txt calc2
	// (at least 1) through SKILL_ComputeSummonLevel; a blank calc2 keeps the
	// owner's level here.
	switch sk.SrvDoFunc {
	case 114, 115, 119:
		if !sk.Calc[2].Empty() {
			o.Level = maxInt(c.calc(2), 1)
		}
	}

	switch {
	case sk.SrvDoFunc == 114:
		o.Count = o.Max
	case sk.SrvDoFunc == 44 || sk.SrvDoFunc == 45 || sk.PetType == "assassintrap":
		o.Kind = "trap"
		o.Frames = 25 * 60
		ax, ay := c.aim()
		o.X, o.Y = ax, ay

		// the trap fires its own skill with the damage of the trap skill
		o.Desc = c.desc()
	case sk.PetType == "totem":
		o.Kind = "totem"
		ax, ay := c.aim()
		o.X, o.Y = ax, ay
	case sk.SrvDoFunc == 119 || sk.SrvDoFunc == 49 || sk.SrvDoFunc == 56:
		o.Count = 1
	}

	if sk.TargetCorpse || sk.SrvDoFunc == 58 {
		if !c.tgt.Corpse {
			c.fail(ReasonNoCorpse)
			return
		}

		o.CorpseID, o.X, o.Y = c.tgt.CorpseID, c.tgt.CX, c.tgt.CY
		o.UseCorpseType = sk.SrvDoFunc == 58
	}

	c.effect(Effect{Kind: "summon", Summon: o})
}

// doWallFn is SRVDO_060_BoneWall / 062 BonePrison. Bone Wall (read, 0x5c37a0):
// the first wall stands at the target, then two bonewallmaker missiles fly
// perpendicular to the cast line placing calc2/2 walls each (calc2 = "# of
// walls - 1", Param3 = 8 in the table); a skill without that missile (or
// calc2 < 2) keeps the older line of Param3 pieces (UNVERIFIED layout). Bone
// Prison (0x5c3c50, VERIFIED) is the 12-cell ring of BonePrisonOffsets. Walls last Param2 frames
// (MAX duration) and have calc1 percent extra life (skills.txt "hp %
// adjustment").
func doWallFn(c *cast) {
	ax, ay := c.aim()
	o := &SummonOrder{Key: c.sk.Summon, PetType: "none", Kind: "wall", Count: c.sk.Params[3], Max: 64,
		Frames: c.sk.Params[2], HPPct: c.calc(1), X: ax, Y: ay, Mode: c.sk.SumMode}

	if c.sk.SrvDoFunc == 62 {
		o.Count = len(BonePrisonOffsets) // 0x5c3c50, VERIFIED batch 4
		o.Mode = "ring"
	}

	if o.Count < 1 {
		o.Count = 8
	}

	if per := c.calc(2) / 2; c.sk.SrvDoFunc == 60 && per >= 1 && c.sk.SrvMissileA != "" && c.p.Missiles.ByName(c.sk.SrvMissileA) != nil {
		hx, hy := c.u.Pos()
		dx, dy := ax-hx, ay-hy

		// the perpendicular, one subtile per step on each axis (0x5c37a0 turns
		// the cast heading by 90 degrees; the exact rounding is UNVERIFIED)
		px, py := -sign(dy), sign(dx)
		if px == 0 && py == 0 {
			px, py = 0, 1
		}

		o.Count = 1
		o.Mode = ""
		o.Makers = &WallMakers{Missile: c.sk.SrvMissileA, PerMaker: per, FromX: ax, FromY: ay,
			Dirs: [2][2]int{{px * 16, py * 16}, {-px * 16, -py * 16}}}
	}

	c.effect(Effect{Kind: "summon", Summon: o})
}

// doCorpseExplosionFn is SRVDO_055_CorpseExplosion (U) and SRVDO_063 (Poison
// Explosion): the corpse at the aim point is consumed and every enemy within
// aurarangecalc of it takes a percentage of the corpse's life as fire damage
// (a roll between calc1 and calc2 percent; the real split between fire and
// physical was not read). Poison Explosion releases the skill's poison damage
// instead.
func doCorpseExplosionFn(c *cast) {
	if !c.tgt.Corpse {
		c.fail(ReasonNoCorpse)
		return
	}

	radius := c.env.eval(c.sk.AuraRangeCalc)
	if c.sk.SrvDoFunc == 55 {
		radius = (radius + 1) / 2 // VERIFIED 0x5c2c60: the units hit are within (aurarange+1)/2
	}

	e := Effect{Kind: "area_hit", Origin: "aim", X: c.tgt.CX, Y: c.tgt.CY, Radius: radius, CorpseID: c.tgt.CorpseID,
		CorpseHP: c.tgt.CorpseHP}

	if c.sk.SrvDoFunc == 63 {
		// VERIFIED (0x5c3dc0 + 0x5a6d70, batch 4): eight stationary clouds of srvmissilea around the corpse
		if name := c.missileName(); name != "" && c.p.Missiles != nil && c.p.Missiles.ByName(name) != nil && c.p.Sim != nil {
			for _, cell := range PoisonExplosionCells {
				x, y := c.tgt.CX+cell[0], c.tgt.CY+cell[1]
				c.tgt.X, c.tgt.Y = x, y
				c.castM(name, castOpts{hasStart: true, startX: float64(x) + 0.5, startY: float64(y) + 0.5})
			}

			c.effect(Effect{Kind: "area_hit", Origin: "aim", X: c.tgt.CX, Y: c.tgt.CY, Radius: 0, CorpseID: c.tgt.CorpseID})

			return
		}

		e.Desc = c.desc()
		c.effect(e)

		return
	}

	// VERIFIED (0x5c2c60): both bounds are percents of the corpse's life, and
	// the roll is taken over the life range lo..hi-1 (RAND_RollSeedModulo(hi-lo)).
	lo := mulDiv(c.tgt.CorpseHP, c.calc(1), 100)
	hi := mulDiv(c.tgt.CorpseHP, c.calc(2), 100)
	v := int32(lo)

	if hi > lo {
		v += int32(c.rollN(int32(hi - lo)))
	}

	d := &d2missile.DamageDesc{}

	// VERIFIED (SRVDO_055 0x5c2c60): a corpse above the caster's level deals
	// its damage scaled by caster level / corpse level.
	if cl := c.tgt.CorpseLevel; cl > 0 && cl > c.u.Level() {
		v = int32(mulDiv(int(v), c.u.Level(), cl))
	}

	v <<= 8
	phys, el := splitCorpseDamage(v, c.calc(3), c.sk.EType)
	d.PhysMin, d.PhysMax = phys, phys

	switch c.sk.EType {
	case "cold":
		d.Cold = el
	case "ltng":
		d.Lightning = el
	case "mag":
		d.Magic = el
	default:
		d.Fire = el
	}

	e.Desc = d
	// VERIFIED 0x5c2b90: beyond (aurarange/2)^2 squared subtiles from the
	// corpse the physical part is zeroed (the elemental part stays).
	r := c.env.eval(c.sk.AuraRangeCalc)
	e.Falloff, e.FalloffSq = true, (r/2)*(r/2)
	c.effect(e)
}
