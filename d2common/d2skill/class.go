package d2skill

import (
	"math"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2combat"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2missile"
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
				Pct: c.env.eval(c.sk.Calc[1]), MinDamage: c.env.eval(c.sk.Calc[2]), FloorPct: c.p.Opt.StaticFieldMinPct,
				EType: c.sk.EType, ELen: c.sk.ElemLen(c.env, c.lvl),
			})
		},
		doNova: func(c *cast) { c.p.doNovaRing(c.u, c.sk, c.lvl, c.tgt, c.env, c.res) },

		// Barbarian
		9:  doStrikeBuffFn, // Frenzy (SRVDO_009_Frenzy) and, with 120, Maul / Feral Rage
		68: doShoutFn,      // Shout, Battle Orders, Battle Command, War Cry
		70: doDoubleSwingFn,
		71: doCurseFn, // Taunt
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
		33: doHammerFn,
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
		47: doArmorFn,    // Cloak of Shadows
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
		mr.Hit, mr.Chance, mr.Roll = d2combat.RollToHit(u.Roller(), d2combat.ToHitInput{
			AttackRating: u.AttackRating(), Defense: t.Defense(false),
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
	mr.Total = dmg.SumTotal(true)

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
		m.Total = m.Damage.SumTotal(true)
	}

	c.addMelee(m)
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

// doMultiHitFn is SRVDO_013_Fend (Zeal / Fury / Fend, U): Zeal and Fury hit
// the target calc1 times with calc2 percent extra damage; Fend (start
// function 9) hits up to calc1 enemies around the target once each.
func doMultiHitFn(c *cast) {
	if !c.needTarget() {
		return
	}

	o := c.meleeOpt()
	o.pct = c.calc(2)

	if c.sk.SrvStFunc == 9 { // Fend
		foes := c.p.foes(c.tgt.UX, c.tgt.UY, 6)
		max := c.calc(1)

		for i, f := range foes {
			if i >= max {
				break
			}

			c.addMelee(c.p.strike(c.u, c.sk, c.lvl, f.Target, c.env, o))
		}

		if c.res.Melee == nil {
			c.addMelee(c.p.strike(c.u, c.sk, c.lvl, c.tgt.Unit, c.env, o))
		}

		return
	}

	n := c.calc(1)
	if n < 1 {
		n = 1
	}

	for i := 0; i < n; i++ {
		if !c.tgt.Unit.Alive() {
			break
		}

		c.addMelee(c.p.strike(c.u, c.sk, c.lvl, c.tgt.Unit, c.env, o))
	}
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

// doSmiteFn is SRVDO_150_Smite (U): the shield bash; damage is the shield-based
// range, here the shield is not modelled so it deals Smite's own physical
// MinDam/MaxDam range plus calc1 percent, and stuns for calc2 frames.
func doSmiteFn(c *cast) {
	if !c.needTarget() {
		return
	}

	o := c.meleeOpt()
	o.pct = c.calc(1)
	m := c.p.strike(c.u, c.sk, c.lvl, c.tgt.Unit, c.env, o)

	if m.Hit {
		m.Damage.StunLen = int32(c.calc(2))
	}

	c.addMelee(m)
}

// doChargedStrikeFn is SRVDO_011_ChargedStrike: a Power-Strike style melee hit;
// on a hit calc1 charged bolts fly out of the target in random directions.
func doChargedStrikeFn(c *cast) {
	if !c.needTarget() {
		return
	}

	o := c.meleeOpt()
	o.skillElem = false
	m := c.p.strike(c.u, c.sk, c.lvl, c.tgt.Unit, c.env, o)
	c.addMelee(m)

	if !m.Hit {
		return
	}

	bolts := c.calc(1)
	for i := 0; i < bolts; i++ {
		ang := float64(c.rollN(360)) * math.Pi / 180
		x, y := c.tgt.UX, c.tgt.UY
		tg := c.tgt
		tg.Unit, tg.X, tg.Y = nil, x+int(math.Round(math.Cos(ang)*10)), y+int(math.Round(math.Sin(ang)*10))

		cc := *c
		cc.tgt = tg
		cc.castM(c.sk.SrvMissileA, castOpts{hasStart: true, startX: float64(x) + 0.5, startY: float64(y) + 0.5})
	}
}

// doLightningStrikeFn is SRVDO_014_LightningStrike (U): a melee hit that
// chains lightning through enemies near the target.
func doLightningStrikeFn(c *cast) {
	if !c.needTarget() {
		return
	}

	o := c.meleeOpt()
	o.skillElem = false
	m := c.p.strike(c.u, c.sk, c.lvl, c.tgt.Unit, c.env, o)
	c.addMelee(m)

	if m.Hit {
		c.chainFrom(c.tgt.UX, c.tgt.UY, c.tgt.Unit, c.sk.SrvMissileA, 3, 10)
	}
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

// doStrafeFn is SRVDO_012_Strafe (U): calc1 arrows, each at a different enemy
// near the aim point, cycling when there are fewer enemies than arrows.
func doStrafeFn(c *cast) {
	ax, ay := c.aim()
	foes := c.p.foes(ax, ay, 14)
	n := c.calc(1)

	if n < 1 {
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

// doChainFn is SRVDO_026_ChainLightning: the bolt jumps to up to calc1 more
// enemies within aurarangecalc of the previous victim (hit function 12, U).
func doChainFn(c *cast) {
	jumps := c.calc(1)
	rng := c.env.eval(c.sk.AuraRangeCalc)
	name := c.missileName()
	hit := map[string]bool{}

	var hook func(m *d2missile.Missile, t d2missile.Target)

	hook = func(m *d2missile.Missile, t d2missile.Target) {
		if hit[t.ID()] {
			return
		}

		hit[t.ID()] = true

		if len(hit) > jumps {
			return
		}

		if pt, ok := t.(d2missile.Positioned); ok {
			x, y := pt.SubPos()
			c.chainStep(int(x), int(y), hit, name, rng, hook)
		}
	}

	if c.castM(name, castOpts{onHit: hook}) == nil {
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

// doRabiesFn is SRVDO_121_Rabies: the plague missile (poison from EType).
func doRabiesFn(c *cast) {
	if c.castM(c.missileName(), castOpts{}) == nil {
		c.fail(ReasonMissile)
	}
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

// doHammerFn is SRVDO_033_PsychicHammer (U): magic damage (EType mag) in a 3
// subtile radius around the aim and a stun of calc? frames is not applied.
func doHammerFn(c *cast) {
	ax, ay := c.aim()
	c.effect(Effect{Kind: "area_hit", Origin: "aim", X: ax, Y: ay, Radius: 3, Desc: c.desc()})
}

// doMindBlastFn is SRVDO_051_MindBlast (U): enemies in the radius par7 around the
// aim are stunned for the skill's stun length and take the magic damage.
func doMindBlastFn(c *cast) {
	ax, ay := c.aim()
	c.effect(Effect{Kind: "area_hit", Origin: "aim", X: ax, Y: ay, Radius: c.env.eval(c.sk.AuraRangeCalc), Desc: c.desc()})
}

// doShoutFn is SRVDO_068_Shout: Shout and Battle Orders are timed self states
// (the party members in range get them too; only the hero exists here).
// War Cry has no state: it damages and stuns the enemies around the hero.
func doShoutFn(c *cast) {
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

// doTeleportFn is SRVDO_027_Teleport: the caster appears at the aim point when
// it can stand there.
func doTeleportFn(c *cast) {
	x, y := c.aim()
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

// doChargeFn is SRVDO_067_Charge (U): rush to the target and strike it with
// calc1 percent damage.
func doChargeFn(c *cast) {
	if !c.needTarget() {
		return
	}

	hx, hy := c.u.Pos()
	c.effect(Effect{Kind: "move", Mode: "charge", X: (hx + c.tgt.UX) / 2, Y: (hy + c.tgt.UY) / 2})

	o := c.meleeOpt()
	o.pct = c.calc(1)
	c.addMelee(c.p.strike(c.u, c.sk, c.lvl, c.tgt.Unit, c.env, o))
}

// doFistFn is SRVDO_080_FistOfTheHeavens (U): a lightning bolt on the aim point,
// calc? holy bolts to enemies around are not modelled.
func doFistFn(c *cast) {
	ax, ay := c.aim()
	c.effect(Effect{Kind: "area_hit", Origin: "aim", X: ax, Y: ay, Radius: 3, Delay: 8, Desc: c.desc()})
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
		}
	} else {
		e.Stats = stats
		if sk.SrvDoFunc == 66 && sk.EType == "mag" { // Sanctuary: damages undead only; U: all enemies here
			e.Mode, e.Desc = "damage", c.desc()
		}
	}

	if sk.SrvDoFunc == 82 { // Redemption
		e.Mode = "redemption"
		e.Heal, e.Dist = c.calc(2), c.calc(3)
		e.Stack = c.calc(1)
	}

	c.effect(e)
}

// ---- storms and rains ----

// doRainFn is SRVDO_028 (Meteor, Blizzard). Meteor: one strike on the aim
// point after 12 frames, radius aurarangecalc. Blizzard: calc2 frames apart,
// shards fall at random points within calc1 of the aim for 100 frames (the
// blizzardcenter lifetime), each hitting a radius of 2. All U: the missiles'
// own do functions (10 / meteorcenter hit function 14) were not read.
func doRainFn(c *cast) {
	ax, ay := c.aim()
	d := c.desc()

	if c.sk.SrvMissileA == "meteorcenter" || c.sk.Name == "Meteor" {
		c.effect(Effect{Kind: "strikes", Origin: "aim", Desc: d, Strikes: []Strike{
			{Delay: 12, X: ax, Y: ay, Radius: maxInt(c.env.eval(c.sk.AuraRangeCalc), 3)},
		}})

		return
	}

	radius := c.calc(1)
	step := c.calc(2)

	if radius < 1 {
		radius = 7
	}

	if step < 1 {
		step = 4
	}

	var strikes []Strike

	for f := 0; f < 100; f += step {
		a := float64(c.rollN(360)) * math.Pi / 180
		r := math.Sqrt(float64(c.rollN(1000))/1000) * float64(radius)
		strikes = append(strikes, Strike{Delay: f + 6, X: ax + int(math.Round(math.Cos(a)*r)),
			Y: ay + int(math.Round(math.Sin(a)*r)), Radius: 2})
	}

	c.effect(Effect{Kind: "strikes", Origin: "aim", Desc: d, Strikes: strikes})
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

// doVolcanoFn is SRVDO_123_Volcano (Fissure, U): eruptions at random points
// within aurarangecalc of the aim for ~3 seconds.
func doVolcanoFn(c *cast) {
	ax, ay := c.aim()
	radius := maxInt(c.env.eval(c.sk.AuraRangeCalc), 4)
	desc := c.desc()

	var strikes []Strike

	for i := 0; i < 8; i++ {
		a := float64(c.rollN(360)) * math.Pi / 180
		r := math.Sqrt(float64(c.rollN(1000))/1000) * float64(radius)
		strikes = append(strikes, Strike{Delay: 8 + i*8, X: ax + int(math.Round(math.Cos(a)*r)),
			Y: ay + int(math.Round(math.Sin(a)*r)), Radius: 3})
	}

	c.effect(Effect{Kind: "strikes", Origin: "aim", Desc: desc, Strikes: strikes})
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

// doWallFn is SRVDO_060_BoneWall / 062 BonePrison (U): Param3 (8) stationary
// bone wall pieces in a line (Bone Wall) or a ring around the target (Bone
// Prison) lasting Param2 frames, each with calc1 extra life.
func doWallFn(c *cast) {
	ax, ay := c.aim()
	o := &SummonOrder{Key: c.sk.Summon, PetType: "none", Kind: "wall", Count: c.sk.Params[3], Max: 64,
		Frames: c.sk.Params[2], HPFlat: c.calc(1), X: ax, Y: ay, Mode: c.sk.SumMode}

	if c.sk.SrvDoFunc == 62 {
		o.Count = 8
		o.Mode = "ring"
	}

	if o.Count < 1 {
		o.Count = 8
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
	e := Effect{Kind: "area_hit", Origin: "aim", X: c.tgt.CX, Y: c.tgt.CY, Radius: radius, CorpseID: c.tgt.CorpseID,
		CorpseHP: c.tgt.CorpseHP}

	if c.sk.SrvDoFunc == 63 {
		e.Desc = c.desc()
		c.effect(e)

		return
	}

	lo, hi := c.calc(1), c.calc(2)
	pct := lo

	if hi > lo {
		pct = lo + c.rollN(int32(hi-lo+1))
	}

	d := &d2missile.DamageDesc{}
	v := int32(mulDiv(c.tgt.CorpseHP, pct, 100)) << 8
	d.Fire = d2missile.Elem{Min: v, Max: v}
	e.Desc = d
	c.effect(e)
}
