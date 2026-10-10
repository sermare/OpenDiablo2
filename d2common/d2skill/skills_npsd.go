package d2skill

// Do functions of the Necromancer, Paladin, Sorceress and Druid that the first
// class table (class.go) left out: Telekinesis (21), Iron Golem (57),
// Conversion (79), Plague Poppy / Cycle of Life / Vines (115), Hunger (122) and
// Hydra (144). Each cites the exe function it follows; "VERIFIED" marks what
// was read in the decompilation (Game.exe 1.14b, read-only), "U" what was
// inferred from the skills.txt columns.
//
// This file registers its handlers in its own init (it sorts after class.go,
// so doTable exists); the Amazon / Barbarian / Assassin handlers stay in
// class.go.

func init() {
	doTable[21] = doTelekinesisFn
	doTable[57] = doIronGolemFn
	doTable[79] = doConversionFn
	doTable[115] = doVineFn
	doTable[122] = doHungerFn
	doTable[144] = doHydraFn
}

// Convertible is implemented by a Target that can say whether Conversion may
// act on it. VERIFIED (0x5ce820 SRVDO_ConversionRollSuccess): the target must
// be a monster (not a player), not a mercenary or other hireling, and have
// alignment 0 (UNIT_GetStateStatAC_OrDefault2 at 0x625c10: units that are
// already converted, attracted or confused carry another alignment).
type Convertible interface {
	CanConvert() bool
}

// doConversionFn is SRVDO_079_Conversion (0x5ce8c0). VERIFIED flow: it needs a
// target unit; when the target is a valid monster the roll "calc1 > random
// 0..99" decides (calc1 is the chance, Param3..Param4 percent over the skill
// levels). A success puts the conversion state (auratargetstate) on the
// monster for auralencalc frames, makes it leave its leader, clears its
// target and, when the monster is above the caster's level, scales its life
// and level down to the caster's (state 0x6d keeps the originals; the revert
// callback 0x5ce710 restores them). A failed roll resolves the queued melee
// hit like a normal blow (SRVDO_ResolveQueuedHitSequence), so the skill still
// damages. Unverified: the argument of the random roll (taken as 100).
func doConversionFn(c *cast) {
	if !c.needTarget() {
		return
	}

	ok := true
	if cv, has := c.tgt.Unit.(Convertible); has {
		ok = cv.CanConvert()
	}

	if !ok || c.rollN(100) >= c.calc(1) {
		doMeleeFn(c)

		return
	}

	frames := c.env.eval(c.sk.AuraLenCalc)
	if frames < 1 {
		frames = 1
	}

	c.effect(Effect{Kind: "convert", State: c.sk.AuraTargetState, Frames: frames, Target: c.tgt.Unit,
		CasterLevel: c.u.Level()})
}

// doHungerFn is SRVDO_122_Hunger (0x5c5f40, the druid's werewolf bite).
// VERIFIED: a normal melee roll; on a hit the damage struct gets the skill's
// hit flags, its physical damage changes by calc1 percent (Param5 = -75) and
// its life steal and mana steal fields (struct +0x38 and +0x3c) grow by calc2
// and calc3 (dm12 / dm34: percent of the damage dealt).
func doHungerFn(c *cast) {
	if !c.needTarget() {
		return
	}

	o := c.meleeOpt()
	o.pct = c.calc(1)

	m := c.p.strike(c.u, c.sk, c.lvl, c.tgt.Unit, c.env, o)
	if m.Hit {
		m.Damage.LifeLeech += int32(c.calc(2))
		m.Damage.ManaLeech += int32(c.calc(3))
	}

	c.addMelee(m)
}

// doHydraFn is SRVDO_144_Hydra (0x5c8ab0). VERIFIED: three hydras are summoned
// around the aim point at offsets (-1,-1), (0,0) and (1,-1) (tables at
// 0x6e46bc / 0x6e46b0), each gets sumskill1..3 at their calc levels, the cast
// is refused in town. U: the lifetime is Param1 frames, the hydras are
// stationary "trap" shooters that fire the missile of sumskill1 (HydraMissile)
// with the damage of Hydra itself.
func doHydraFn(c *cast) {
	if c.sk.Summon == "" {
		c.fail(ReasonNoSkill)
		return
	}

	if c.u.InTown() && !c.p.Opt.IgnoreTown {
		c.fail(ReasonTown)
		return
	}

	ax, ay := c.aim()

	c.effect(Effect{Kind: "summon", Summon: &SummonOrder{
		Key: c.sk.Summon, PetType: c.sk.PetType, Mode: c.sk.SumMode, Count: 3, Max: maxInt(c.env.eval(c.sk.PetMax), 3),
		Kind: "trap", Frames: c.sk.Params[1], TrapSkill: c.sk.SumSkill[1], Stats: statsOf(c.env, c.sk),
		X: ax, Y: ay, Desc: c.desc(), Cells: [][2]int{{-1, -1}, {0, 0}, {1, -1}},
	}})
}

// doVineFn is SRVDO_115_PlaguePoppy (0x5c4a10; also Cycle of Life and Vines).
// VERIFIED: one vine monster is summoned (pettype "vine", petmax 1, so a new
// one replaces the old), it gets state 0x96, its level is calc2 and its life
// grows by calc1 percent. The poppy shoots the "plague vines" missile of Vine
// Attack (sumskill1) at enemies, with the poison damage of the summoning skill.
// U: the corpse-eating cycler skills of Cycle of Life / Vines (sumskill2) that
// heal the owner are not simulated; those two only attack.
func doVineFn(c *cast) {
	if c.sk.Summon == "" {
		c.fail(ReasonNoSkill)
		return
	}

	ax, ay := c.aim()
	o := &SummonOrder{Key: c.sk.Summon, PetType: c.sk.PetType, Mode: c.sk.SumMode, Count: 1,
		Max: maxInt(c.env.eval(c.sk.PetMax), 1), Kind: "trap", TrapSkill: c.sk.SumSkill[1], HPPct: c.calc(1),
		Level: maxInt(c.calc(2), 1), X: ax, Y: ay, Desc: c.desc(), Stats: statsOf(c.env, c.sk)}

	c.effect(Effect{Kind: "summon", Summon: o})
}

// doIronGolemFn is SRVDO_057_IronGolem (0x5c3100). VERIFIED: the start function
// (SRVST_020) validates the item target; the golem is created next to the
// caster, the item is taken out of the world (it becomes the golem's body)
// and SKILL_SummonGolemCommon scales it. This port raises the golem like the
// other golems (hp from calc1, aurastat thorns / fade, the Golem Mastery
// passives through SummonPassives); consuming the item and the item-derived
// stats are not simulated (U).
func doIronGolemFn(c *cast) {
	doSummonFn(c)
}

// doTelekinesisFn is SRVDO_Telekinesis (0x5c79f0). VERIFIED: the caster must be
// a player and have a target. A hostile monster outside town takes the skill's
// physical and elemental damage (stats flagged by the skill's hit flags; a
// roll under the skill's per-level increment adds hit flag 9: knockback / stun
// class), a ground item of the usable types is picked up, an object is
// operated. Only the damage on monsters is simulated here (U).
func doTelekinesisFn(c *cast) {
	if !c.u.IsPlayer() || !c.needTarget() {
		return
	}

	if c.u.InTown() && !c.p.Opt.IgnoreTown {
		c.fail(ReasonTown)
		return
	}

	c.effect(Effect{Kind: "area_hit", Origin: "aim", X: c.tgt.UX, Y: c.tgt.UY, Radius: 1, Desc: c.desc()})
}
