package d2skill

import (
	"math"

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
		doJab:      doJabFn,
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
	wmin, wmax := missileWeapon(c.u)

	d := c.sk.Descriptor(c.env, c.lvl, wmin, wmax, c.u.Stat(masteryStat[c.sk.EType]))
	d.DamagePct = descriptorPct(c.u, c.sk, 0)

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
	// convClass / convPct move that percent of the physical damage into an element (damage struct +0x65 / +0x68,
	// skill EType and calc4; Berserk, Concentrate, Frenzy: magic).
	convClass byte
	convPct   int
	// elemPct adds fire, cold and lightning damage equal to that percent of
	// the bare weapon damage roll (Vengeance, SRVST_035 0x5ce340, VERIFIED wave 23:
	// added, not converted; see vengeanceElem).
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

	// shield block: COMBAT_RollAttackOutcome (0x57cc10) rolls it after a successful to-hit roll, on the
	// DEFENDER's seed (the target's own); a blocked blow clears the hit bit. Only after a rolled to-hit (the
	// autohit and kick paths here roll none; whether the exe still blocks them is UNVERIFIED).
	if mr.Hit && !sk.Kick && !o.autoHit {
		if b, ok := t.(d2missile.Blocker); ok && b.RollBlock(false) {
			mr.Hit, mr.Blocked = false, true
		}
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
		// COMBAT_RollPhysicalDamage 0x579120 (VERIFIED wave 46): STATS_GetUnitFullStat(unit, 0x6f), the "+N to
		// damage" property (item_normaldamage), is added to both ends BEFORE the 1 / min+1 floors and the percent.
		flat := normalDamageFlat(u)
		lo, hi := (int32(wmin)+flat)<<8, (int32(wmax)+flat)<<8

		if lo < 1<<8 {
			lo = 1 << 8
		}

		if hi < lo+1<<8 {
			hi = lo + 1<<8
		}

		ph := rollRange(r, lo, hi)
		ph += int32(mulDiv(int(ph), o.pct+u.Stat("damagepercent")+meleeBonusPct(u)+masteryOf(u, MasteryDamage, sk), 100))
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

	// poison damage of the weapon: the stat is the per-frame 8.8 value, rolled raw (BuildAttackerDamage)
	if hi := u.Stat("poisonmaxdam"); hi > 0 && !sk.Kick {
		dmg.Poison += rollRange(r, int32(u.Stat("poisonmindam")), int32(hi))
		dmg.PoisonLen += int32(u.Stat("poisonlength"))
	}

	// life and mana leech of the attacker's stats ride on the struct (BuildAttackerDamage, players)
	if !sk.Kick {
		dmg.LifeLeech += int32(u.Stat("lifedrainmindam"))
		dmg.ManaLeech += int32(u.Stat("manadrainmindam"))
	}

	if o.elemPct != [3]int{} {
		p.vengeanceElem(u, sk, lvl, env, o, &dmg)
	}

	// stun on hit from states (Maul: stunlength, frames)
	if st := u.Stat("stunlength"); st > 0 && !sk.Kick {
		dmg.StunLen += int32(st)
	}

	if o.convClass > 0 && o.convPct > 0 {
		d2combat.ConvertPhysical(&dmg, r, o.convClass, int32(o.convPct))
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
	case sk.SrvStFunc == stBash, sk.SrvDoFunc == doJab, sk.SrvStFunc == 7, sk.SrvStFunc == 39,
		sk.SrvStFunc == stPowerStrike, sk.SrvStFunc == stLightningStrike:
		o.pct = c.calc(1)
	case sk.SrvStFunc == 35: // Vengeance
		o.elemPct = [3]int{c.calc(1), c.calc(2), c.calc(3)}
		o.skillElem = false
	}

	// Bash, Stun, Concentrate, Berserk, Frenzy and Double Swing (SRVST_Bash 0x5d6910, SRVST_039 0x5d82a0,
	// 0x5d75b0, VERIFIED): with an EType and calc4 > 0 that percent of the physical damage becomes that element.
	if sk.SrvStFunc == stBash || sk.SrvStFunc == 39 || sk.SrvDoFunc == 9 || sk.SrvDoFunc == 70 {
		if cls := ElemConvClass(sk.EType); cls > 0 {
			if pct := c.calc(4); pct > 0 {
				o.convClass, o.convPct = cls, pct
			}
		}
	}

	return o
}

// corpseOK applies the corpse tests of the exe to the corpse in the target when the engine described it
// (Target.CorpseInfo): Raise Skeleton and Raise Skeletal Mage use MONSTER_IsCorpseUsableForRaise (0x6468b0), the
// other corpse skills SKILL_CanReviveOrExplodeCorpse (0x646a10).
func (c *cast) corpseOK() bool {
	f := c.tgt.CorpseInfo
	if f == nil {
		return true
	}

	if c.sk.SrvDoFunc == 31 {
		return CorpseRaisable(*f, false)
	}

	return CorpseExplodable(*f)
}

func (c *cast) needTarget() bool {
	if c.tgt.Unit == nil || !c.tgt.Unit.Alive() {
		c.fail(ReasonTarget)
		return false
	}

	return true
}

// outOfReach is the range gate of the attack outcome (AttackInReach) when Options.MeleeGate is on: the blow finds
// no hit when the target is farther than the table distance allows (no to-hit roll is made then, 0x57cc10 returns 0
// before it).
func (c *cast) outOfReach() bool {
	if !c.p.Opt.MeleeGate || c.tgt.Unit == nil {
		return false
	}

	ux, uy := c.u.Pos()
	sa, sb := HeroUnitSize, UnitSizeOf(c.tgt.Unit)

	if !c.u.IsPlayer() {
		sa, sb = 1, 1 // a monster attacker keeps the unsized distance of the earlier port
	}

	extra := 1
	if !c.u.IsPlayer() {
		extra = 3
	}

	d := TableDistanceSized(c.tgt.UX-ux, c.tgt.UY-uy, sa, sb)
	if d < 1 {
		return false
	}

	if d > c.p.meleeRange()+1+extra {
		return true
	}

	// COMBAT_IsTargetWithinMeleeRange 0x622e40: in range, the line trace with mask 0x804 must find no blocker
	if c.p.Grid != nil {
		return !MeleeLineClear(c.p.Grid, MeleeTraceMask, ux, uy, sa, c.tgt.UX, c.tgt.UY, sb)
	}

	return false
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

	if c.outOfReach() {
		c.addMelee(&MeleeResult{Target: c.tgt.Unit})

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

	// Fend only (SRVST_009 0x5d99b0, VERIFIED wave 23): the start function counts the enemies in reach (melee
	// range + 4, filter 0x20003, the target included) and keeps min(calc1, found) hits; Zeal (SRVST_037
	// 0x5d9ac0) keeps calc1 whatever is around. A count of 0 (the engine's aimed target is not listed) still
	// strikes once.
	if c.sk.SrvStFunc == stFend && c.p.Near != nil {
		ux, uy := c.u.Pos()

		found := 0

		for _, f := range c.p.Near(ux, uy, FendReach(c.p.meleeRange())) {
			if f.Target.Alive() {
				found++
			}
		}

		n = StrikeCount(n, found)
	}

	if n < 1 {
		n = 1
	}

	b := c.p.newMeleeBurst(c, n)
	b.opt.pct = c.calc(2)

	// the animation clock of the hero's attack: every event reschedules the animation by Param2 percent of the
	// elapsed time (SRVDO_013 0x5da910 passes SKILLS_GetPerLevelIncrementOfBaseValue = dwParam2 to 0x551b90)
	if clk := c.p.clockFor(c.u, c.sk); clk != nil {
		b.attachClock(clk, c.sk.Params[2])
	}

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

// LifeHolder is implemented by a Target that can say how much life it has (whole points). Sacrifice caps the
// damage its self cost is taken from by the target's current life.
type LifeHolder interface {
	LifePoints() int
}

// doSacrificeFn is SRVDO_064_Sacrifice (0x5cccb0) with the blow of SRVST_029 (0x5ccb50), VERIFIED wave 23: the blow
// is a plain melee hit with calc1 percent damage (the percent goes into the weapon roll, SrcDam 0, and the skill's
// element only when calc4 is above zero). The do function then finds the queued hit and charges the caster calc2
// percent of the physical damage that hit rolled (before the target's resists), capped by the life the target had
// (stat 6); a miss rolled no damage, so it costs nothing. The charge is a damage struct flagged 0x1000 applied to
// the caster. It is NOT a share of the caster's maximum life (the older stand-in).
func doSacrificeFn(c *cast) {
	if !c.needTarget() {
		return
	}

	o := c.meleeOpt()
	o.pct = c.calc(1)
	m := c.p.strike(c.u, c.sk, c.lvl, c.tgt.Unit, c.env, o)
	c.addMelee(m)

	if m.Hit {
		dealt := int(m.Damage.Physical)
		if lh, ok := c.tgt.Unit.(LifeHolder); ok {
			if life := lh.LifePoints() << 8; life < dealt {
				dealt = life
			}
		}

		c.effect(Effect{Kind: "self_damage", SelfDamagePct: c.calc(2), SelfDamageFixed: mulDiv(dealt, c.calc(2), 100)})
	}
}

// doChargedStrikeFn is SRVDO_011_ChargedStrike (0x5da4a0, VERIFIED): a Power-Strike style melee hit, then
// calc1 charged bolts are created AT THE TARGET whether or not the hit landed (the exe never tests the
// outcome). Their direction is the point mirrored through the target (2*target - self); the per-bolt
// per-bolt hook (0x5c7340) reseeds each bolt with destX+index (ChargedBoltSeed) and walks path type 10.
func doChargedStrikeFn(c *cast) {
	if !c.needTarget() {
		return
	}

	// The blow is SRVST_006 (0x5d9470, shared with Power Strike, VERIFIED wave 23): calc1 goes into the damage
	// percent field (for Charged Strike that is the bolt count, a quirk of the table) and the skill's own
	// lightning damage (SKILL_RollElementalSkillDamage) is part of the blow as well as of every bolt.
	o := c.meleeOpt()
	c.addMelee(c.p.strike(c.u, c.sk, c.lvl, c.tgt.Unit, c.env, o))

	bolts := c.calc(1)
	ux, uy := c.u.Pos()

	// SRVDO_011 0x5da4a0 + hook 0x5c7340 (VERIFIED): start at the target, destination 2*target - caster, seed =
	// destination x + index, path type 10 (the same wobbling path as Charged Bolt, d2missile.PathType10Nodes).
	x, y := c.tgt.UX, c.tgt.UY
	dx, dy := 2*x-ux, 2*y-uy

	for i := 0; i < bolts; i++ {
		tg := c.tgt
		tg.Unit, tg.X, tg.Y = nil, dx, dy

		cc := *c
		cc.tgt = tg
		cc.castM(c.missileName(), castOpts{hasStart: true, startX: float64(x) + 0.5, startY: float64(y) + 0.5,
			pathType10: true, pathSeed: ChargedBoltSeed(dx, i)})
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

	// The blow is SRVST_010 (0x5d9bb0, VERIFIED wave 23): calc1 (the radius, 20) goes into the damage percent
	// field like Power Strike's, and the lightning of the skill (min + roll, SKILL_GetElementalMin/MaxDamage) is
	// part of the blow.
	o := c.meleeOpt()
	c.addMelee(c.p.strike(c.u, c.sk, c.lvl, c.tgt.Unit, c.env, o))

	// The scan is SKILL_ScanRadiusForTargets around the struck unit with the struck unit's id as the excluded id
	// (0x5dab30, VERIFIED wave 23): its callback 0x569a40 keeps the smallest unit id above the struck unit's, else
	// the smallest id of all (d2missile.ScanNext), not the nearest. The struck unit itself is a candidate and is
	// what the scan returns when nothing else is in reach.
	foes := c.p.foes(c.tgt.UX, c.tgt.UY, c.calc(1))
	cands := make([]d2missile.Target, 0, len(foes))
	byID := map[string]*Foe{}

	for i := range foes {
		cands = append(cands, foes[i].Target)
		byID[foes[i].Target.ID()] = &foes[i]
	}

	pick := d2missile.ScanNext(cands, c.tgt.Unit)
	if pick == nil {
		return
	}

	best := byID[pick.ID()]

	cc := *c
	cc.tgt = Target{Unit: best.Target, UX: best.X, UY: best.Y}
	cc.castM(c.sk.SrvMissileA, castOpts{hasStart: true, startX: float64(best.X) + 0.5, startY: float64(best.Y) + 0.5, chainLeft: c.calc(2)})
}

// ---- ranged / missile families ----

// doFanFn is SRVDO_008_MultipleShot (Multiple Shot, Teeth, Shock Wave): calc1
// missiles, each aimed at its own point of a row through the aim point and
// perpendicular to the line of fire (FanSpacing / FanDestinations, V wave 22:
// the helpers 0x56b0e0 and 0x56b120 were read).
//
// Read from the binary (0x5d9fe0): the count is calc1; calc2 goes to the
// created missile's word at +0x44 (role UNVERIFIED) and calc3 is the number of
// "real" middle missiles: the (calc1-calc3)/2 flanking ones on each side are
// created with flag 0x10000 (meaning UNVERIFIED, not modelled) and only the
// middle calc3 without it (calc3 0 means all). The missile B pick is the exe's
// "attack weapon class is not 1" test, ported as UsesMissileB (VERIFIED; the hero reports its COF weapon class).
// Flag 0x10000 of the created missiles is set on the flanking ones and on the middle ones it is clear; what it
// does inside MISSILE_CreateServerMissile is not traced (UNVERIFIED).
func doFanFn(c *cast) {
	name := c.sk.SrvMissileA
	if UsesMissileB(c.u, c.sk) {
		name = c.sk.SrvMissileB
	}

	n := c.calc(1)
	if n < 1 {
		n = 1
	}

	ux, uy := c.u.Pos()
	ax, ay := c.aim()
	sx, sy := FanSpacing(ax-ux, ay-uy)

	// create flag 0x800 with calc2 in the word at +0x44: the arrows can collide after calc2 frames instead of
	// the missile's Activate column (0x59d5d0, VERIFIED; calc2 = par3 = 1 for Multiple Shot).
	act := c.calc(2)

	for _, d := range FanDestinations(ax, ay, n, sx, sy) {
		c.castM(name, castOpts{hasDest: true, destX: float64(d[0]) + 0.5, destY: float64(d[1]) + 0.5,
			activate: act, hasActivate: true})
	}

	if len(c.res.Missiles) == 0 {
		c.fail(ReasonMissile)
	}
}

// doHomingFn is SRVDO_010_GuidedArrow (Guided Arrow, Bone Spirit): a missile
// that steers onto the aimed unit.
//
// Read again in wave 23 (0x5da2f0, VERIFIED): a non-zero calc1 ("+ dmg%", Guided Arrow: ln34) goes to the
// missile's stat 0x19 through the create callback 0x5da2c0 (stat += calc1), which MISSILE_BuildDamageDescriptor
// reads into the percent pool of the arrow; the unit flag 0x40 is set; a crossbow picks srvmissileb.
func doHomingFn(c *cast) {
	name := c.missileName()
	if UsesMissileB(c.u, c.sk) {
		name = c.sk.SrvMissileB
	}

	if c.castM(name, castOpts{home: c.tgt.Unit, pctAdd: c.calc(1)}) == nil {
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

// doStrafeFn is SRVDO_012_Strafe (0x5da6c0, VERIFIED): one arrow per attack animation event at the stored
// target, then the next enemy within aurarangecalc other than the one just shot; the burst length comes from
// StrafeBurst (SRVST_008) and every event reschedules the animation by Param6 percent (see Burst, actionevents.go).
// A lone enemy ends the burst after one arrow (the rescan excluding it finds nobody). With Options.EventBursts
// only the first arrow flies here and DoResult.Burst carries the rest.
func doStrafeFn(c *cast) {
	rng := strafeReach(c)
	ux, uy := c.u.Pos()
	foes := c.p.foes(ux, uy, rng)
	n := StrafeBurst(c.calc(1), c.calc(3), len(foes))

	if n < 1 {
		n = 1
	}

	tg := c.tgt
	if tg.Unit == nil && len(foes) > 0 {
		tg.Unit, tg.UX, tg.UY = foes[0].Target, foes[0].X, foes[0].Y
	}

	b := &Burst{kind: BurstStrafe, u: c.u, sk: c.sk, lvl: c.lvl, env: c.env, left: n, target: tg, rng: rng}
	if clk := c.p.clockFor(c.u, c.sk); clk != nil {
		b.attachClock(clk, c.sk.Params[6])
	}

	c.p.BurstEvent(b, c.res)

	if len(c.res.Missiles) == 0 {
		c.fail(ReasonMissile)

		return
	}

	if c.p.Opt.EventBursts {
		if !b.Done() {
			c.res.Burst = b
		}

		return
	}

	c.p.RunBurst(b, c.res)
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

// InfernoFlameLife is the lifetime in frames SRVDO_InfernoSpawnFlameMissile (0x5c6d40, VERIFIED wave 23) gives
// each flame: calc1 of the skill (ln12/2 for Inferno), at least 1; the create kind 0x8020 takes the life from the
// params instead of the missile's Range column. The old port used aurarangecalc (ln12/4-2) converted to frames.
func InfernoFlameLife(calc1 int) int {
	if calc1 < 1 {
		return 1
	}

	return calc1
}

// doStreamFn is SRVDO_019_Inferno (Inferno, Arctic Blast): in the exe every run of the do function
// creates ONE flame missile at the caster toward the target position (0x5c6d40, VERIFIED) and the channel
// start function (SRVST_011 0x5c7050, mana gate: startmana unless state 12 is on) repeats the do while the
// button is held. The engine has no channel, so one cast is still a burst of six flames three frames apart (U);
// each flame lives InfernoFlameLife frames.
func doStreamFn(c *cast) {
	name := c.missileName()

	if c.p.Missiles.ByName(name) == nil {
		c.fail(ReasonMissile)
		return
	}

	rng := InfernoFlameLife(c.calc(1))

	// the held-button channel (channel.go): one flame per run after the wind-up
	if IsChannelStart(c.sk) && c.u.IsPlayer() {
		if c.p.channelDo(c.u) && c.castM(name, castOpts{rangeLife: rng}) == nil {
			c.fail(ReasonMissile)
		}

		return
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

// FireWallMakerDestinations are the two destinations SRVDO_024_FireWall (0x5c7fc0, VERIFIED wave 23) gives its
// two srvmissilea makers: both start on the target (tx, ty), and fly along the perpendicular to the line from the
// target to the caster (hx, hy), one each way: (tx - (hy - ty), (hx - tx) + ty) and (tx + (hy - ty), ty - (hx - tx)).
// A destination that equals the start (caster on the target) is moved by one subtile in both axes, the exe's fix for
// a missile with no direction (MISSILE_CreateServerMissile 0x59d5d0).
func FireWallMakerDestinations(tx, ty, hx, hy int) [2][2]int {
	out := [2][2]int{{tx - (hy - ty), (hx - tx) + ty}, {tx + (hy - ty), ty - (hx - tx)}}

	for i := range out {
		if out[i][0] == tx && out[i][1] == ty {
			out[i][0]++
			out[i][1]++
		}
	}

	return out
}

// doFireWallFn is SRVDO_024_FireWall (0x5c7fc0, VERIFIED wave 23). The cast needs a valid srvmissilea
// (firewallmaker), a walkable target cell for that missile (SKILL_IsSkillTargetCellWalkable 0x56c210), a target
// that is not on 0 and not in a town room. Then two firewallmaker missiles start on the target and fly
// perpendicular to the line from the caster, one each way (FireWallMakerDestinations); each drops a firewall
// (its SubMissile1) at every cell it passes (SrvDoFunc 6, 0x5ac1b0) for Range + LevRange * level frames. A third
// missile, srvmissileb (firewall), stands on the target itself. The old port laid a fixed line of stationary
// missiles. Wave 24 VERIFIED the hit cadence of the standing fire: one hit per frame per unit in the cell, no scale.
func doFireWallFn(c *cast) {
	nameA, nameB := c.sk.SrvMissileA, c.sk.SrvMissileB
	if nameA == "" {
		c.fail(ReasonMissile)
		return
	}

	ax, ay := c.aim()
	if ax == 0 || ay == 0 || (c.p.Walkable != nil && !c.p.Walkable(ax, ay)) {
		c.fail(ReasonMissile)
		return
	}

	if c.u.InTown() && !c.p.Opt.IgnoreTown {
		c.fail(ReasonTown)
		return
	}

	hx, hy := c.u.Pos()
	sx, sy := float64(ax)+0.5, float64(ay)+0.5
	// The standing fire hits every frame (w24, VERIFIED 0x5abd00 / 0x5aba10: firewall has neither LastCollide nor
	// NextHit, so nothing dedupes a target standing in it), each hit with the rolled damage << HitShift; the
	// missile column DamageRate (41, about 1024 / 25) scales the target's flat damage reduction per hit (0x579ef0).
	hit := castOpts{}

	for _, d := range FireWallMakerDestinations(ax, ay, hx, hy) {
		o := hit
		o.hasStart, o.startX, o.startY = true, sx, sy
		o.hasDest, o.destX, o.destY = true, float64(d[0])+0.5, float64(d[1])+0.5

		if m := c.p.castMissile(c.u, c.sk, c.lvl, c.env, nameA, Target{X: ax, Y: ay}, o); m != nil {
			c.res.Missiles = append(c.res.Missiles, m)
		}
	}

	if nameB != "" {
		o := hit
		o.hasStart, o.startX, o.startY, o.stationary = true, sx, sy, true

		if m := c.p.castMissile(c.u, c.sk, c.lvl, c.env, nameB, Target{X: ax, Y: ay}, o); m != nil {
			c.res.Missiles = append(c.res.Missiles, m)
		}
	}

	if len(c.res.Missiles) == 0 {
		c.fail(ReasonMissile)
	}
}

// doTwisterFn is SRVDO_118_Twister (Twister, Tornado; 0x5c52c0 / SRVDO_TwisterSpawnChain 0x5c5000, VERIFIED,
// batch 9): calc1 missiles (none when calc1 is below one) are created at the caster toward the target position, ALL
// of them through the shared per-missile hook (0x5c7340): the seed is the destination x plus the index and the path
// becomes type 10, the scattering stepper, with the lifetime clamped to 77 frames (not modelled here). There is no
// straight first missile (the chain's flag is 0) and no fan. The wobbling path is d2missile.PathType10Nodes (0x67d160, VERIFIED wave 20).
func doTwisterFn(c *cast) {
	n := c.calc(1)
	if n < 1 {
		c.fail(ReasonMissile)
		return
	}

	ax, _ := c.aim()

	for i := 0; i < n; i++ {
		c.castM(c.missileName(), castOpts{pathType10: true, pathSeed: ChargedBoltSeed(ax, i)})
	}

	if len(c.res.Missiles) == 0 {
		c.fail(ReasonMissile)
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
// here. Batch 4: the spiral (path type 0xe, d2missile.SpiralNodes) is modelled;
// the Concentration scaling of stats 0x34 / 0x35 (0x647550, VERIFIED: param1 times the
// damage percent of the Concentration state over 8, added to 100 percent) is
// applied through ConcentrationHolder.
func doBlessedHammerFn(c *cast) {
	o := castOpts{}
	if h, ok := c.u.(ConcentrationHolder); ok {
		if s := HammerConcentrationScale(c.sk.Params[1], h.ConcentrationDamagePct()); s != 0 {
			o.scalePct = 100 + s
		}
	}

	m := c.castM(c.missileName(), o)
	if m == nil {
		c.fail(ReasonMissile)
		return
	}

	m.Spiral = true
}

// doMineFn is SRVDO_043_ShockField (Shock Web, VERIFIED in w23, see shockfield.go): count caltrops thrown from the
// caster to random points around the target position, not one stationary missile. The earlier port (a stationary
// missile at the aim point zapping every 12 frames) was an invention.
// The throws use creation kind 0x420 (VERIFIED w29, 0x5d4540): the lob flag 0x400, so each caltrop ends AT its point
// (clamp) instead of flying its whole range of 50 frames past it (the w27 scenario saw the fields 12 tiles beyond the aim).
func doMineFn(c *cast) {
	count := c.calcOrState()
	if count < 1 {
		c.fail(ReasonMissile)
		return
	}

	ax, ay := c.aim()
	hx, hy := c.u.Pos()

	for _, pt := range ShockFieldPoints(ax, ay, count, c.env.eval(c.sk.AuraRangeCalc), hx, hy) {
		tg := Target{X: pt[0], Y: pt[1]}

		if m := c.p.castMissile(c.u, c.sk, c.lvl, c.env, c.sk.SrvMissileA, tg, castOpts{hasDest: true, clamp: true,
			destX: float64(pt[0]) + 0.5, destY: float64(pt[1]) + 0.5}); m != nil {
			c.res.Missiles = append(c.res.Missiles, m)
		}
	}

	if len(c.res.Missiles) == 0 {
		c.fail(ReasonMissile)
	}
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

	// VERIFIED (w37, SRVDO_025_Enchant 0x5c8180): the state goes to the targeted unit when the exe's alliance test
	// holds for it (a pet, mercenary or converted monster), else to the caster. Only Enchant has that rule.
	if c.sk.SrvDoFunc == 25 && c.tgt.Ally != nil && c.tgt.Ally.Alive() && c.tgt.Ally.ID() != c.u.ID() {
		e.Kind, e.Target = "unit_state", c.tgt.Ally
	}

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
// srvmissilea on each new subtile the caster walks onto, see Engine.trail and
// BlazeTrailStep; Interval only marks the trail). Energy Shield: a timed state whose synthetic
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
		Interval: AuraPeriod(c.env, sk), Mode: "friendly"}

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

	// the passive stats that ride on the caster's aura state (SRVDO_065 / 066 / 081, VERIFIED)
	if sk.SrvDoFunc == 65 {
		e.PassiveStats = auraPassiveStats(c.env, sk)
	} else {
		e.Stats = append(e.Stats, auraPassiveStats(c.env, sk)...)
	}

	if sk.SrvDoFunc == 65 { // friendly aura: the upkeep is the aura parameter (VERIFIED 0x5cd4b0 + 0x645ed0)
		e.Cost = int(sk.calcMana(c.lvl, false))
	}

	if sk.SrvDoFunc == 66 { // SRVDO_066 0x5cd880 (VERIFIED wave 23): the same upkeep gates its passive stats (mana >= upkeep); it never pays
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

	if hx, hy := c.u.Pos(); !CreateAtTargetInRange(hx, hy, ax, ay) { // SKILL_ServerCreateMissileAtTarget, distance < 0x65
		c.fail(ReasonMissile)
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

// doFirestormFn is SRVDO_117_Firestorm (0x5c5130, VERIFIED, batch 9): calc1 emitters (the skill's srvmissilea,
// firestormmaker, whose SrvDoFunc 23 drops the firestorm fire along its path) are cast: the first straight at the
// target, the other calc1 - 1 from the caster toward the same destination, each re-seeded by the shared hook
// (path type 10 through d2missile.PathType10Nodes, VERIFIED wave 20). A calc1 below one casts nothing.
func doFirestormFn(c *cast) {
	n := c.calc(1)
	if n < 1 {
		c.fail(ReasonMissile)
		return
	}

	ax, _ := c.aim()

	if c.castM(c.missileName(), castOpts{}) == nil {
		c.fail(ReasonMissile)
		return
	}

	for i := 0; i < n-1; i++ {
		c.castM(c.missileName(), castOpts{pathType10: true, pathSeed: ChargedBoltSeed(ax, i)})
	}
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
		e.Interval, e.Radius = StormInterval(sk.Params[4]), c.env.eval(sk.AuraRangeCalc) // 0x5c65b0 / 0x5c63f0: Param4 frames, the radius as is
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

		stats := []StatMod{{Stat: c.sk.AuraState, Value: 1}}

		// VERIFIED (SRVDO_TigerStrike_ApplyChargeStatList 0x5d19e0): each new charge (up to 3) adds aurastatcalc2 to
		// aurastat2 (progressive_tohit); the engine multiplies a stacking state's progressive_tohit by the count.
		if st := c.sk.AuraStat[2]; st != "" {
			stats = append(stats, StatMod{Stat: st, Value: c.env.eval(c.sk.AuraStatCalc[2])})
		}

		c.effect(Effect{Kind: "self_state", State: c.sk.AuraState, Frames: frames, Stack: 3, Stats: stats})
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
	lastHit := false

	switch c.sk.SrvDoFunc {
	case 42:
		hits = maxInt(c.calc(1), 1)
	case 46:
		hits = 2
	}

	// w23: the charges are in the damage of every blow up to and including the first that hit (chargedamage.go)
	held := c.heldCharges()

	for i := 0; i < hits; i++ {
		o := c.meleeOpt()
		o.pct = c.env.eval(c.sk.Calc[1])

		if c.sk.SrvDoFunc == 42 {
			o.pct = 0
		}

		o.skillElem = false
		lastHit = false
		o.pct += c.chargePct(held)

		m := c.p.strike(c.u, c.sk, c.lvl, c.tgt.Unit, c.env, o)
		lastHit = m.Hit
		c.applyChargeDamage(m, held)

		if m.Hit {
			held = nil
		}

		c.addMelee(m)
	}

	// VERIFIED 0x5d4190 / 0x5d4370: the knockback bits (0xc) are added to the hit result only for the last
	// kick AND only when that kick's attack roll hit.
	if c.sk.SrvDoFunc == 42 && lastHit && c.knockRoll(c.tgt.Unit, 100) {
		c.knock(c.tgt.Unit)
	}

	if c.sk.SrvDoFunc == 50 {
		c.effect(Effect{Kind: "area_hit", Origin: "aim", X: c.tgt.UX, Y: c.tgt.UY,
			Radius: maxInt(c.env.eval(c.sk.AuraRangeCalc), 4), Desc: c.desc()})
	}

	c.releaseCharges()
}

// builderOf finds the charge-up skill whose aurastate is the given progressive state.
func (p *Pipeline) builderOf(state string) *Skill {
	for _, id := range p.Skills.IDs() {
		if sk := p.Skills.ByID(id); sk.AuraState == state && (sk.SrvDoFunc == 34 || sk.SrvDoFunc == 35) {
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

	wmin, wmax := c.u.WeaponDamage()
	o.SkillName = sk.Name
	o.SkillDmgMin = int(sk.PhysMin(c.env, c.lvl, wmin, true)) >> 8
	o.SkillDmgMax = int(sk.PhysMax(c.env, c.lvl, wmax, true)) >> 8
	o.SkillToHit = sk.ToHitBonus(c.env, c.lvl)

	// VERIFIED (SRVDO_114 / 115 / 119): the monster's level is skills.txt calc2
	// (at least 1) through SKILL_ComputeSummonLevel; a blank calc2 keeps the
	// owner's level here.
	switch sk.SrvDoFunc {
	case 114, 115, 119:
		if !sk.Calc[2].Empty() {
			o.Level = maxInt(c.calc(2), 1)
		}

		if sk.SrvDoFunc == 114 && c.lvl > 0 {
			o.Pecks = sk.Params[5] + (c.lvl-1)*sk.Params[6]
		}
	case 31, 56, 57:
		// VERIFIED (SKILL_ComputeSummonLevel 0x5c2850, called by SRVDO_031 / 056 / 057 with no fixed level)
		o.Level = SummonLevel(c.u.Level(), c.lvl)
	case 15, 16:
		// VERIFIED (SRVDO_015 0x5dad10, SRVDO_016 0x5daf10, wave 31): Dopplezon and Valkyrie call SKILL_ComputeSummonLevel
		// with no fixed level, so the monster level is the SummonLevel of the owner and the skill (calc2 only reaches
		// SpawnSummonedMinion as the item level for the template equipment; it is never the unit's level and the
		// wiki's 99 cap is not a monster level cap: only the monlvl.txt row used for AC and AR is capped, LevelBonus).
		o.Level = SummonLevel(c.u.Level(), c.lvl)
	case 45:
		// VERIFIED (SKILL_SummonSentryUnit 0x5d4790, via SRVDO_045 0x5d4b20): the
		// sentry's level comes from SKILL_ComputeSummonLevel with no fixed level.
		o.Level = SummonLevel(c.u.Level(), c.lvl)
	case 44:
		// VERIFIED (SRVDO_044 0x5d49c0): the same, then the caster's own level
		// (stat 0xc) is written over it.
		o.Level = c.u.Level()
	}

	if sk.SrvDoFunc == 49 {
		// VERIFIED (shadow.go): bonuses only above level 1, every slot at aurastatcalc2, life (lvl-1)*Param1 percent
		o.Stats, _ = shadowBonuses(c.env, sk, c.lvl)
		o.HPPct = shadowHPPct(sk, c.lvl)
	}

	if sk.SrvDoFunc == 56 {
		o.AuraSkill, o.AuraLevel = c.minionAura()

		// Fire Golem: its Holy Fire is a SRVDO_066 aura held by a monster, which puts the five passivestat pairs
		// (firemindam / firemaxdam = enms / exms * par5 / 256) on the golem's own state unconditionally (0x5cd880,
		// VERIFIED, see auraPassiveStats), so every blow of the golem carries them on top of the skill's own fire
		// damage. The wiki "Fire damage" column of the golem is exactly skill fire + this + the A1 hit (w31).
		// UNVERIFIED: synergies (Resist Fire / Salvation) read the golem's skills, which it does not have.
		if o.AuraSkill != "" {
			if as := c.p.Skills.ByName(o.AuraSkill); as != nil {
				o.Stats = append(o.Stats, MonsterAuraStats(c.p.Skills, as, o.AuraLevel, o.Level)...)
			}
		}
	}

	// The Skeletal Mage's monster skill (sumskill1) gets the level sumsk1calc (SKILL_SpawnSummonedMinion 0x5c22c0,
	// VERIFIED); the shots take their damage from the missile record at that level.
	if sk.SrvDoFunc == 31 && sk.SumSkill[1] != "" && sk.SumSkCalc[1] != nil {
		o.TrapLevel = c.env.eval(sk.SumSkCalc[1]) // <= 0: the exe adds no skill, the mage then only melees
	}

	if sk.SrvDoFunc == 56 && sk.Name == "BloodGolem" {
		o.OwnerDamagePct, o.HealSharePct = sk.Params[5], sk.Params[6]
	}

	switch {
	case sk.SrvDoFunc == 114:
		// VERIFIED (wave 23): SRVDO_114 0x5c48a0 spawns ONE raven per cast
		// (SKILL_ServerSpawnSummonedMonster once; the do function is run once
		// per cast by SKILL_ServerRunSkillFunc 0x56d6a0). The pet limit
		// (petmax) is enforced when it registers: at the limit the oldest
		// pet of the type is released (PETS_PreparePetSlotForNewPet 0x573a30).
		o.Count = 1
	case sk.SrvDoFunc == 44 || sk.SrvDoFunc == 45 || sk.PetType == "assassintrap":
		o.Kind = "trap"
		o.Frames = 25 * 60
		ax, ay := c.aim()
		o.X, o.Y = ax, ay

		// the trap fires its own skill with the damage of the trap skill
		o.Desc = c.desc()
		o.TrapLevel, o.TrapLife = c.sentryLife()
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

		if !c.corpseOK() {
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
	if !c.tgt.Corpse || !c.corpseOK() {
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
