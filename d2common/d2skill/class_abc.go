package d2skill

import (
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2missile"
)

// Do functions of the Amazon, Barbarian and Assassin that docs/skills-coverage-abc.md
// listed as missing: Dopplezon (15), Valkyrie (16), Blade Fury (48), Dragon
// Flight (52), Find Potion (69), Find Item (72), Grim Ward (75) and Whirlwind
// (76). The registration lives here (not in the table of class.go) so the
// class files of the other classes stay untouched; Go runs the init functions
// of a package in file name order and "class.go" comes before this file.
//
// Everything marked "verified" was read from Game.exe 1.14b (the addresses are
// in the comments); "U" marks behaviour inferred from skills.txt.
func init() {
	doTable[15] = doDopplezonFn
	doTable[16] = doValkyrieFn
	doTable[48] = doBladeFuryFn
	doTable[52] = doDragonFlightFn
	doTable[69] = doFindPotionFn
	doTable[72] = doFindItemFn
	doTable[75] = doGrimWardFn
	doTable[76] = doWhirlwindFn
	doTable[54] = doBladeShieldFn // a bare state in class.go: the shield now hurts as well
}

// ---- Blade Shield ----

// doBladeShieldFn is SRVDO_054_BladeShield (0x5d6880, verified shape): the do
// function only checks the aurastate and, outside a town room, hands over to
// the periodic effect of the skill (periodic = 1, perdelay = par3 = 25
// frames, the attachment missile). Port: the "bladeshield" state lasts
// auralencalc frames and, every perdelay frames, every enemy within
// aurarangecalc (par4) of the caster takes the skill's damage (U: the exe
// does it with the blade shield attachment missiles; their hit rule was not read).
func doBladeShieldFn(c *cast) {
	frames := c.env.eval(c.sk.AuraLenCalc)

	c.p.doState(c.sk, c.env, c.res, "self_state")
	e := &c.res.Effects[len(c.res.Effects)-1]
	e.Level, e.SkillID, e.SkillName = c.lvl, c.sk.ID, c.sk.Name

	c.effect(Effect{Kind: "shield", State: c.sk.AuraState, Frames: frames, Radius: c.env.eval(c.sk.AuraRangeCalc),
		Interval: maxInt(c.env.eval(c.sk.PerDelay), 1), Desc: c.desc()})
}

// ---- summons: Dopplezon and Valkyrie ----

// doDopplezonFn is SRVDO_015_Dopplezon (0x5dad10, verified): a summon of the
// monstats key in the summon column (the pet cap is petmax), whose life is
// calc3 percent of the caster's maximum life (the exe writes stats 6 and 7 of
// the decoy and copies the caster's level, stat 0xc) and which expires calc2
// frames later (an event scheduled at the current frame + calc2). calc1 is
// not read by the do function.
func doDopplezonFn(c *cast) {
	doSummonFn(c)

	if !c.res.OK && c.res.Reason != "" {
		return
	}

	if n := len(c.res.Effects); n > 0 && c.res.Effects[n-1].Summon != nil {
		o := c.res.Effects[n-1].Summon
		o.HPPct = 0
		o.OwnerHPPct = c.calc(3)
		o.Frames = c.calc(2)
		o.DrawsAggro = true
	}
}

// doValkyrieFn is SRVDO_016_Valkyrie (0x5daf10, verified): only a player can
// cast it; it summons the pet of the summon column (one at a time, petmax),
// the pet level is computed from the owner, calc2 is read for the minion setup
// and calc1 is the life bonus handed to the minion (SKILL_SpawnSummonedMinion);
// the pet gets state 0x5d. The engine uses calc1 as the percent life bonus
// like the other summons (U for the exact use of calc1 inside the spawn).
func doValkyrieFn(c *cast) {
	if !c.u.IsPlayer() {
		c.fail(ReasonNoSkill)
		return
	}

	doSummonFn(c)

	if n := len(c.res.Effects); n > 0 && c.res.Effects[n-1].Summon != nil {
		c.res.Effects[n-1].Summon.DrawsAggro = true
	}
}

// ---- Blade Fury ----

// doBladeFuryFn is SRVDO_048_BladeFury (0x5d5260, verified shape): the skill
// repeats while the button is held; every do creates one missile of the
// skill's progressive missile (srvmissilea, bladefragment1) aimed at the target
// point and blocks the next one for prgcalc1 - 1 frames (prgcalc1 is "par4";
// the exe stores the unlock frame in the skill's target type field, here the
// cooldown of the skill does the same). A value of prgcalc1 that is 1 or less
// makes the do fail.
func doBladeFuryFn(c *cast) {
	interval := c.sk.Params[4]
	if interval-1 <= 0 {
		c.fail(ReasonMissile)
		return
	}

	if c.castM(c.missileName(), castOpts{}) == nil {
		c.fail(ReasonMissile)
		return
	}

	c.u.SetCooldown(c.sk.ID, c.p.frame()+interval-1)
}

// ---- Dragon Flight ----

// doDragonFlightFn is SRVDO_052_DragonFlight (0x5d6290). The exe has two
// branches chosen by a flag of the caster that was not identified: with a
// target the caster delivers a finishing kick (to-hit roll, kick damage), and
// without one he teleports to the clicked point when the level allows it. This
// port does both in one cast, which is how the skill plays (UNVERIFIED which
// condition picks the branch): the caster arrives next to the target and kicks
// it; with no target he only teleports to the aim point.
func doDragonFlightFn(c *cast) {
	if c.tgt.Unit != nil && c.tgt.Unit.Alive() {
		hx, hy := c.u.Pos()
		x, y := c.tgt.UX, c.tgt.UY
		// land on the side of the target the caster comes from
		dx, dy := sign(hx-x), sign(hy-y)
		lx, ly := x+dx, y+dy

		if c.p.Walkable != nil && !c.p.Walkable(lx, ly) {
			lx, ly = hx, hy
		}

		if lx != hx || ly != hy {
			c.effect(Effect{Kind: "move", Mode: "teleport", X: lx, Y: ly})
		}

		o := c.meleeOpt()
		o.autoHit = false
		c.addMelee(c.p.strike(c.u, c.sk, c.lvl, c.tgt.Unit, c.env, o))

		return
	}

	x, y := c.aim()
	if c.p.Walkable != nil && !c.p.Walkable(x, y) {
		c.fail(ReasonLOS)
		return
	}

	c.effect(Effect{Kind: "move", Mode: "teleport", X: x, Y: y})
}

func sign(v int) int {
	switch {
	case v > 0:
		return 1
	case v < 0:
		return -1
	}

	return 0
}

// ---- Find Potion and Find Item ----

// LootOrder asks the engine to drop the loot of a corpse.
type LootOrder struct {
	CorpseID string
	// Kind is "potion" or "item".
	Kind string
	// Chance is calc1 and Roll the 0..99 roll against it; Drop is Roll < Chance.
	Chance, Roll int
	Drop         bool
	// Column picks the potion kind (0 health, 1 mana, 2 rejuvenation) of Find
	// Potion; FindPotionCode turns it into an item code for the act and difficulty.
	Column int
	// TCType is the monstats treasure class column (1 normal, 2 champion, 3
	// unique, 4 quest) Find Item rolls on the corpse's monster.
	TCType int
}

// findPotionTable is the table at 0x73e988 (verified): 12 bytes per entry, the
// item codes of the health, mana and rejuvenation potion, indexed by
// act (0 based) + 5 * difficulty.
var findPotionTable = [15][3]string{
	{"hp2", "mp2", "rvs"}, {"hp3", "mp3", "rvs"}, {"hp3", "mp3", "rvs"}, {"hp4", "mp4", "rvl"}, {"hp4", "mp4", "rvl"},
	{"hp4", "mp4", "rvl"}, {"hp4", "mp5", "rvl"}, {"hp5", "mp5", "rvl"}, {"hp5", "mp5", "rvl"}, {"hp5", "mp5", "rvl"},
	{"hp5", "mp5", "rvl"}, {"hp5", "mp5", "rvl"}, {"hp5", "mp5", "rvl"}, {"hp5", "mp5", "rvl"}, {"hp5", "mp5", "rvl"},
}

// FindPotionCode is SRVDO_FindPotion_RollPotionByAct's table lookup: the item
// code of the potion for a column (0 health, 1 mana, 2 rejuvenation), a 1 based
// act and a difficulty (0 normal .. 2 hell). It returns "" outside the table
// (the exe returns 0 for an act above 5).
func FindPotionCode(act, difficulty, column int) string {
	a := act - 1
	if a < 0 || a > 4 || difficulty < 0 || difficulty > 2 || column < 0 || column > 2 {
		return ""
	}

	return findPotionTable[a+difficulty*5][column]
}

// doFindPotionFn is SRVDO_069_FindPotion (0x5d6c40, verified): the target must
// be a corpse that was not looted (state 0x76); it is marked looted, then
// RAND(100) < calc1 drops a potion. The potion kind is a second roll of 100:
// below Param3 mana, below Param3 + Param4 rejuvenation, else health (0x5d6b80).
// The cast succeeds (plays its sound) for every valid corpse.
func doFindPotionFn(c *cast) {
	if !c.tgt.Corpse || c.tgt.CorpseLooted {
		c.fail(ReasonNoCorpse)
		return
	}

	o := &LootOrder{CorpseID: c.tgt.CorpseID, Kind: "potion", Chance: c.calc(1)}
	o.Roll = c.rollN(100)

	if o.Roll < o.Chance {
		o.Drop = true
		r := c.rollN(100)

		switch {
		case r < c.sk.Params[3]:
			o.Column = 1
		case r < c.sk.Params[3]+c.sk.Params[4]:
			o.Column = 2
		}
	}

	c.effect(Effect{Kind: "loot", Loot: o})
}

// doFindItemFn is SRVDO_FindItem (0x5d7210, verified): same corpse rule, then
// RAND(100) < calc1 rolls the bucket of the treasure class type: [0,P1) is
// type 1, then P2 wide type 2, P3 wide type 3 and P4 wide type 4 (Param1..4,
// 5/60/30/5); the monster's TreasureClass<type> is dropped
// (SRVDO_DropTreasureForFindItem -> ITEMGEN_DropMonsterTreasure).
func doFindItemFn(c *cast) {
	if !c.tgt.Corpse || c.tgt.CorpseLooted {
		c.fail(ReasonNoCorpse)
		return
	}

	p := c.sk.Params
	o := &LootOrder{CorpseID: c.tgt.CorpseID, Kind: "item", Chance: c.calc(1), TCType: 1}
	o.Roll = c.rollN(100)

	if o.Roll < o.Chance {
		o.Drop = true
		r := c.rollN(100)

		switch {
		case r >= p[1] && r < p[1]+p[2]:
			o.TCType = 2
		case r >= p[1]+p[2] && r < p[1]+p[2]+p[3]:
			o.TCType = 3
		case r >= p[1]+p[2]+p[3] && r < p[1]+p[2]+p[3]+p[4]:
			o.TCType = 4
		}
	}

	c.effect(Effect{Kind: "loot", Loot: o})
}

// ---- Grim Ward ----

// WardOrder is a Grim Ward: a stationary field that terrifies enemies.
type WardOrder struct {
	CorpseID string
	X, Y     int
	// Radius is aurarangecalc (ln12), Period the frames between pulses (the
	// missile's Param1, 6), Life the lifetime (missile Range, 200) and Fear the
	// frames of the auratargetstate (auralencalc, Param6).
	Radius, Period, Life, Fear int
	State                      string
}

// Grim Ward numbers of missiles.txt (grimwardsmall..large: pSrvDoFunc 14 with
// Param1 6 "repeat frame"; the exe call 0x5acb00 dispatches the periodic helper
// 30 every Param1 frames). The ward lives the skill's calc1 ("ward duration",
// ln34: 1000 frames), not the table Range 200: the start missile's hit
// function 26 (0x5a93f0) creates it with the lifetime override calc1, minimum
// 5 (VERIFIED from the disassembly); Range 200 is only the fallback.
const (
	wardPeriod  = 6
	wardMinLife = 5
)

// doGrimWardFn is SRVDO_075_GrimWard (0x5d7430, verified shape): the target
// must be a corpse; a free cell is found next to it, a missile of the ward
// family (srvmissilea/b/c by the corpse's size class) is created there with
// the corpse marked consumed (state 0x68) and looted (0x76). The ward then
// repeats the skill's aura (auratargetstate "terror" for auralencalc frames
// on enemies within aurarangecalc) every 6 frames for 200 frames.
func doGrimWardFn(c *cast) {
	if !c.tgt.Corpse {
		c.fail(ReasonNoCorpse)
		return
	}

	state := c.sk.AuraTargetState
	if state == "" {
		state = "terror"
	}

	w := &WardOrder{CorpseID: c.tgt.CorpseID, X: c.tgt.CX, Y: c.tgt.CY, Radius: c.env.eval(c.sk.AuraRangeCalc),
		Period: wardPeriod, Life: maxInt(c.calc(1), wardMinLife), Fear: c.env.eval(c.sk.AuraLenCalc), State: state}

	c.effect(Effect{Kind: "ward", Ward: w})
}

// ---- Whirlwind ----

// WhirlOrder is the whirling run of a Whirlwind.
type WhirlOrder struct {
	// X, Y is the destination, subtile; Radius the reach of one hit (5 subtiles,
	// the 0x5d8010 scan radius); Delay the frames between two hits (10 bare
	// handed, see WhirlDelay) and MaxSteps the longest run in subtiles.
	X, Y, Radius, Delay, MaxSteps int
	// Pct is calc1: the damage percent bonus.
	Pct int
	// State is the aurastate kept while it lasts.
	State string
}

// whirlRadius is the scan radius of SKILL_ScanRadiusForTargets in 0x5d8010
// (verified argument 5, read as subtiles: U).
const whirlRadius = 5

// whirlMaxRun is the longest run of one Whirlwind, subtiles (U: the exe walks
// the path to the clicked point; the length is bounded by the path).
const whirlMaxRun = 40

// WhirlDelay is SKILL_WhirlwindComputeNextStepDelay's (0x5d7de0, verified)
// frames between two hits: 10 without a weapon, else by the weapon's cof frame
// count: <12 -> 4, <15 -> 6, <18 -> 8, <20 -> 10, <23 -> 12, else 14 (16 above 25).
func WhirlDelay(weaponFrames int, armed bool) int {
	if !armed {
		return 10
	}

	switch {
	case weaponFrames < 12:
		return 4
	case weaponFrames < 15:
		return 6
	case weaponFrames < 18:
		return 8
	case weaponFrames < 20:
		return 10
	case weaponFrames < 23:
		return 12
	case weaponFrames > 25:
		return 16
	}

	return 14
}

// doWhirlwindFn is SRVST_038 + SRVDO_076_Whirlwind (0x5d7a00, 0x5d8010): the
// start function sets the skill's aurastate and walks the hero to the target
// point; every step hits one enemy within the scan radius other than the last
// one hit (to-hit roll, damage with calc1 percent) until nothing is left.
func doWhirlwindFn(c *cast) {
	x, y := c.aim()
	if c.p.Walkable != nil && !c.p.Walkable(x, y) {
		c.fail(ReasonLOS)
		return
	}

	// bare handed unless the unit reports a weapon (the weapon's frame count is not exposed: U)
	wmin, wmax := c.u.WeaponDamage()
	armed := wmax > 0 && (wmin > 2 || wmax > 2)

	c.effect(Effect{Kind: "whirl", Whirl: &WhirlOrder{X: x, Y: y, Radius: whirlRadius, MaxSteps: whirlMaxRun,
		Delay: WhirlDelay(14, armed), Pct: c.calc(1), State: c.sk.AuraState}})
}

// WhirlStrike resolves one hit of a running Whirlwind against t.
func (p *Pipeline) WhirlStrike(u Unit, skillID int, t d2missile.Target) *MeleeResult {
	sk := p.Skills.ByID(skillID)
	if sk == nil {
		return nil
	}

	lvl := u.SkillLevel(skillID)
	env := p.env(sk, lvl, u)
	o := strikeOpt{toHitPct: sk.ToHitBonus(env, lvl), skillElem: true, pct: env.eval(sk.Calc[1])}

	return p.strike(u, sk, lvl, t, env, o)
}

// Foes lists the living enemies within a Chebyshev radius of a subtile (the
// engine's Near hook), for the engine-side effects of this file.
func (p *Pipeline) Foes(x, y, radius int) []Foe { return p.foes(x, y, radius) }
