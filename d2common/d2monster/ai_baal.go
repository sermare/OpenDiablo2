package d2monster

// The Baal encounter AIs (Throne of Destruction): the throne statue that sends
// the five waves (BaalThrone), the taunting voice (BaalTaunt), the Baal who
// walks to the Worldstone Chamber portal (BaalToStairs), Baal himself and his
// clones (BaalCrab, BaalCrabClone) and the tentacles (BaalTentacle).
//
// Sources: Game.exe 1.14b read-only Ghidra, MONAI_Think_BaalThrone 0x5ee400,
// BaalTaunt 0x5ee810, BaalToStairs 0x5ee720, BaalTentacle 0x5ee920, BaalCrab
// 0x5fc200, BaalCrabClone 0x5fc440 and the AiBaal.cpp helpers 0x5faf40,
// 0x5fb380, 0x5fb4d0, 0x5fb630, 0x5fb830, 0x5fba40, 0x5fbd50. Each comment says
// what is VERIFIED (control flow, constants, roll order) and what is not.
//
// Class numbering: the exe's monster class is the monstats.txt row minus one
// for rows after the "Expansion" separator (VERIFIED by the three ancients,
// exe classes 0x21c..0x21e = rows 541..543 "Ancient Barbarian 1..3"). So the
// exe class of BaalToStairs (0x22f = 559) is the row 560 "Baal Crab to
// Stairs"; the Worldstone Chamber portal is the OBJECT 563 (objects.txt Id).

func init() {
	register("BaalThrone", TargetOnly, thinkBaalThrone)
	register("BaalTaunt", TargetOnly, thinkBaalTaunt)
	register("BaalToStairs", TargetNone, thinkBaalToStairs)
	register("BaalTentacle", TargetOnly, thinkBaalTentacle)
	register("BaalCrab", TargetOnly, thinkBaalCrab)
	register("BaalCrabClone", TargetOnly, thinkBaalCrabClone)
}

// Skill ids the Baal AIs name directly (skills.txt, VERIFIED against the d2exp
// table: 284 Baal Taunt, 285 Baal Corpse Explode, 286 Baal Monster Spawn).
const (
	SkillBaalTaunt        = 284
	SkillBaalCorpseExpode = 285
	SkillBaalMonsterSpawn = 286
)

// Objects the Baal AIs know (objects.txt Id, VERIFIED).
const (
	ObjectWorldstonePortal = 563 // "The Worldstone Chamber" portal; BaalToStairs walks to it
	ThroneSightRadius      = 0x40
	toStairsScan           = 0x19 // scan radius of BaalToStairs for the portal
	throneWaveCount        = 5    // waves 0..4; a counter above 4 ends the fight
	throneWaveGap          = 250  // frames between the announcement and the wave
	throneAfterSpawn       = 100  // frames after a wave was spawned
)

// Optional World extensions of the Baal AIs.
//
// RawCaster casts a skill by id (not by monstats slot) at a target or a
// ground point; used for the skills the throne casts that are not in its slots.
type RawCaster interface {
	CastSkillID(b *Brain, skill int, mode Mode, t *Target, at *Point) bool
}

// ThroneStep is what the throne asks the encounter to do.
type ThroneStep int

// The throne steps.
const (
	// ThroneAnnounce: the wave is announced (a message to the players, the
	// exe sends packet 0xa4 with a wave-dependent id) and Baal Corpse Explode
	// (285) is cast on the throne itself.
	ThroneAnnounce ThroneStep = iota
	// ThroneSpawnWave: the wave is spawned (the exe casts Baal Monster Spawn,
	// 286, at the throne + 13 subtiles south).
	ThroneSpawnWave
	// ThroneMorph: after the last wave the throne turns into Baal on the way to
	// the Worldstone Chamber (exe class 0x22f) and receives state 0x8e.
	ThroneMorph
)

// ThroneHost is implemented by the world to run the wave side of BaalThrone.
type ThroneHost interface {
	// Throne performs the step for the given wave (0-based). It returns false
	// when it could not (the AI then waits).
	Throne(b *Brain, step ThroneStep, wave int) bool
}

// WaveGate is optional: WaveCleared reports that the monsters of the last wave
// are dead. UNVERIFIED: the exe throne itself has no such gate (it is purely
// timed in its think function); the gating of the waves on the previous wave
// being killed, if it exists, would live in the skill functions 285/286 which
// were not read. The engine uses it so a wave is never piled on the previous.
type WaveGate interface {
	WaveCleared(b *Brain) bool
}

// ObjectFinder finds the nearest object of a class around the monster.
type ObjectFinder interface {
	NearestObject(b *Brain, class, radius int) (p Point, dist int, ok bool)
}

// Leaver removes the monster from the level (BaalToStairs entering the portal).
type Leaver interface{ LeaveLevel(b *Brain) }

// Dismisser kills/removes the monster silently (FUN_0057ac20(0,1), VERIFIED
// callers: BaalTentacle and BaalCrabClone when their time or target is gone).
type Dismisser interface{ Dismiss(b *Brain) }

// TargetModer reports the unit mode of a target (BaalTaunt reacts to a hero
// standing still).
type TargetModer interface {
	TargetMode(b *Brain, t Target) Mode
}

// Puller moves a target to the monster's position (SERVER_MoveUnitToLevelPosition
// in BaalTaunt; UNVERIFIED which position, see thinkBaalTaunt).
type Puller interface {
	PullTarget(b *Brain, t Target) bool
}

// Cloner spawns a Baal clone (FUN_005fba40): a copy with a third of Baal's
// life and mana, bound to him as a minion.
type Cloner interface{ SpawnBaalClone(b *Brain) bool }

// ---------------------------------------------------------------- BaalThrone

// thinkBaalThrone is MONAI_Think_BaalThrone 0x5ee400 (VERIFIED control flow).
//
// Scratch[0] is the wave counter (general +0x14), Scratch[1] the flags (+0x18:
// bit 0 "announced, wave due", bit 1 "wave sent"), Scratch[2] the frame of the
// next step (+0x1c). aip1 is the chance to attack an intruder with Skill1.
//
// A hero within 64 subtiles makes the throne attack (Skill1, aip1%) or wait 10
// instead of advancing the waves (VERIFIED order; two line-of-sight / state
// checks on the hero before the roll are not ported). Otherwise the cycle is:
// announce wave n + cast Baal Corpse Explode (flag bit 0, next step in 250
// frames); then spawn the wave, counter+1, next step in 100 frames, flag bit 0
// cleared; after the fifth wave the next step morphs the throne into Baal.
//
// The announcement message (id from a game table by the counter, plus an extra
// message for two special entries 0x3e/0x69) is UNVERIFIED and left to the host.
func thinkBaalThrone(c *Ctx) {
	b := c.B
	now := c.W.Frame()

	if b.Scratch[2] > now {
		c.Sleep(b.Scratch[2] - now) // the exe passes frame-next (negative); UNVERIFIED which way it clamps

		return
	}

	if c.Target != nil && c.Dist <= ThroneSightRadius {
		if b.Profile.Skills[slot1].Used() && b.Roll(100) < b.AIP(1) {
			c.Cast(slot1, *c.Target)

			return
		}

		c.Sleep(10)

		return
	}

	host, _ := c.W.(ThroneHost)

	wave := b.Scratch[0]

	if b.Scratch[1]&1 != 0 {
		if wave > throneWaveCount-1 {
			if host != nil {
				host.Throne(b, ThroneMorph, wave)
			}

			c.Sleep(5)

			return
		}

		if gate, ok := c.W.(WaveGate); ok && wave > 0 && !gate.WaveCleared(b) {
			c.Sleep(25)

			return
		}

		if host != nil && !host.Throne(b, ThroneSpawnWave, wave) {
			c.Sleep(25)

			return
		}

		b.Scratch[0]++
		b.Scratch[2] = now + throneAfterSpawn
		b.Scratch[1] |= 2
		b.Scratch[1] &^= 1

		c.Sleep(throneAfterSpawn)

		return
	}

	if host != nil {
		host.Throne(b, ThroneAnnounce, wave)
	}

	b.Scratch[1] |= 1
	b.Scratch[2] = now + throneWaveGap

	c.Sleep(throneWaveGap)
}

// ---------------------------------------------------------------- BaalTaunt

// thinkBaalTaunt is MONAI_Think_BaalTaunt 0x5ee810 (VERIFIED control flow). The
// invisible taunter follows the hero and mocks him: aip1 the distance beyond
// which it walks up to the hero, aip2 the idle count after which it taunts,
// aip3 the distance beyond which the hero is pulled back.
//
// Scratch[0] counts consecutive thinks in which the hero stands still (mode
// neutral); when the count exceeds aip2 it is reset and Baal Taunt (284, mode
// 4) is cast at the hero. A moving hero resets the count. If the hero is
// farther than aip3 the exe calls SERVER_MoveUnitToLevelPosition on him with a
// position from the taunter's own accessors (UNVERIFIED reading: the hero is
// pulled to the taunter); when that fails and he is farther than aip1 the
// taunter walks to him. Always sleeps 25.
func thinkBaalTaunt(c *Ctx) {
	b := c.B

	if c.Target == nil {
		c.Sleep(25)

		return
	}

	t := *c.Target

	idle := false
	if m, ok := c.W.(TargetModer); ok {
		idle = m.TargetMode(b, t) == ModeNeutral
	}

	if idle {
		b.Scratch[0]++
		if b.Scratch[0] > b.AIP(2) {
			b.Scratch[0] = 0

			if r, ok := c.W.(RawCaster); ok {
				if r.CastSkillID(b, SkillBaalTaunt, ModeAttack1, &t, nil) {
					c.busy()

					return
				}
			}
		}
	} else {
		b.Scratch[0] = 0
	}

	if b.AIP(3) < c.Dist {
		if p, ok := c.W.(Puller); ok && p.PullTarget(b, t) {
			c.Sleep(25)

			return
		}
	}

	if b.AIP(1) < c.Dist {
		c.WalkTo(t, 0)

		return
	}

	c.Sleep(25)
}

// ---------------------------------------------------------------- BaalToStairs

// thinkBaalToStairs is MONAI_Think_BaalToStairs 0x5ee720 (VERIFIED control
// flow). Baal, on his way to the Worldstone Chamber, scans 25 subtiles for the
// portal object 563; none: sleep 25. When he is within aip1 of it he enters:
// state 0x92 is set, he is taken off the level (the exe also sends a packet and
// removes the unit); otherwise he walks to the portal at normal speed.
func thinkBaalToStairs(c *Ctx) {
	b := c.B

	of, ok := c.W.(ObjectFinder)
	if !ok {
		c.Sleep(25)

		return
	}

	p, dist, found := of.NearestObject(b, ObjectWorldstonePortal, toStairsScan)
	if !found {
		c.Sleep(25)

		return
	}

	if dist < b.AIP(1) {
		if l, ok := c.W.(Leaver); ok {
			l.LeaveLevel(b)
		}

		c.acted = true
		b.Wake = waitForever

		return
	}

	c.SetSpeed(0)
	c.moveTo(p)
}

// ---------------------------------------------------------------- BaalTentacle

// thinkBaalTentacle is MONAI_Think_BaalTentacle 0x5ee920 (VERIFIED control
// flow). aip1 attack%, aip2 stall frames, aip3 lifetime base in seconds. The
// first think sets the expiry (aip3 + roll) * 25 frames after now (the bound of
// the roll is UNVERIFIED: 10 is used); past it, or without a living target, the
// tentacle is dismissed. Otherwise attack mode A2 with aip1%, else stall.
func thinkBaalTentacle(c *Ctx) {
	b := c.B
	now := c.W.Frame()

	if c.Target == nil {
		dismiss(c)

		return
	}

	if b.Scratch[2] == 0 {
		b.Scratch[2] = now + (b.AIP(3)+b.Roll(10))*25
	}

	if b.Scratch[2] < now {
		dismiss(c)

		return
	}

	if b.Roll(100) < b.AIP(1) {
		c.Attack(ModeAttack2, *c.Target)

		return
	}

	c.Sleep(b.AIP(2))
}

func dismiss(c *Ctx) {
	if d, ok := c.W.(Dismisser); ok {
		d.Dismiss(c.B)
	}

	c.acted = true
	c.B.Wake = waitForever
}

// ---------------------------------------------------------------- Baal himself

// Baal action codes of AiBaal.cpp (the index of the weight tables).
const (
	baalNone      = 0
	baalWait      = 1
	baalWalk      = 2 // walk to within 12 of the target
	baalCircle    = 3
	baalWander    = 4
	baalHome      = 5 // move to the anchor point
	baalWalk2     = 6 // same as 2
	baalUnused    = 7 // not in the action switch: wait 5
	baalBuff      = 8 // skill slots 6/7 on the target
	baalTentacles = 9
	baalMelee     = 10
	baalNova      = 11
	baalInferno   = 12
	baalCold      = 13
	baalTeleport  = 14
	baalClone     = 15
	baalActions   = 16
)

// BaalSituation are the inputs of the three decision tables. VERIFIED: the
// constants and which input changes which weight. UNVERIFIED: the exact
// meaning of the booleans whose helpers were not read (see each field).
type BaalSituation struct {
	InReach bool // FUN_00622e40 (the same melee-reach test State12 uses)
	Engaged bool // FUN_005dbfd0 (read as Brain.Aggressive)

	SelfHP, TargetHP int // life percent of Baal and of the target
	TargetBlocked    bool
	// TargetBlocked is state 0xb on the target (UNVERIFIED which state it is).

	Targets    int  // number of candidate targets in range (FUN_005faf40's count)
	Score      int  // threat score of the best target (FUN_005fac60), > 60 raises Buff
	Far        bool // anchor farther than 0x4b (75)
	VeryFar    bool // anchor farther than 100
	Worldstone bool // a hero is in the Worldstone Chamber (level 0x84) near the anchor
	Clones     int  // clones alive (general +0x18)
	Nearby     int  // FUN_00571760's out value (UNVERIFIED), 2 or more disables nothing
	Flag1      bool // FUN_005fb280 (UNVERIFIED predicate)
	Dist       int  // edge distance to the target
}

// BaalWeights returns the 16-entry weight table for the situation: the table
// of 0x5fb380 when the target is in reach, of 0x5fb630 when engaged but out of
// reach and of 0x5fb4d0 otherwise.
func BaalWeights(d Difficulty, s BaalSituation) [baalActions]int {
	switch {
	case s.InReach:
		return baalReachTable(d, s)
	case s.Engaged:
		return baalEngagedTable(d, s)
	}

	return baalIdleTable(d, s)
}

func baalReachTable(d Difficulty, s BaalSituation) [baalActions]int {
	w := [baalActions]int{}
	w[baalBuff], w[baalTentacles], w[baalMelee] = 20, 30, 150
	w[baalNova], w[baalInferno], w[baalCold], w[baalTeleport] = 10, 10, 70, 40

	waitBase := 50
	if d == Normal {
		waitBase = 75
		w[baalCold] = 45
	}

	if s.TargetHP < 33 {
		w[baalMelee] = 200
	}

	w[baalWait] = waitBase + (100 - s.SelfHP)

	if s.TargetBlocked {
		w[baalBuff], w[baalCold], w[baalNova] = 0, 0, 0
	}

	if !s.Engaged {
		w[baalNova], w[baalCold], w[baalInferno] = 0, 0, 0
	}

	return w
}

func baalEngagedTable(d Difficulty, s BaalSituation) [baalActions]int {
	w := [baalActions]int{}
	w[2], w[3], w[4], w[6] = 5, 5, 5, 5
	w[baalBuff], w[baalTentacles] = 40, 40
	w[baalNova], w[baalInferno], w[baalCold], w[baalTeleport] = 70, 80, 60, 20

	base := 100
	if d == Normal {
		base = 125
		w[baalCold] = 40
	}

	w[baalClone] = (2 - s.Clones) * 10
	if w[baalClone] < 0 {
		w[baalClone] = 0
	}

	if s.Dist > 0x23 {
		w[6] = 15
		w[baalCold] = 0
	}

	w[baalWait] = base + (100 - s.SelfHP)

	if s.Dist > 0x19 {
		w[baalTeleport] = 30
		w[baalNova] = 0
	}

	if s.Targets < 2 {
		w[baalNova] -= 10
	}

	if s.Targets > 3 {
		w[baalCold] += 25
	}

	if s.Score > 0x3c {
		w[baalBuff] = 70
	}

	if s.Targets < 2 && s.Nearby < 2 {
		w[baalBuff] = 0
	}

	switch {
	case !s.Flag1 && !s.Far:
	case s.Flag1 && !s.Far:
		w[6] += 0x1e
		w[baalNova] += 10
		w[baalTeleport] += 0xf
	default:
		w[6] += 0x19
		w[baalTeleport] += 0x23
		w[2] = 0
		w[3] = 0x19
	}

	if s.VeryFar {
		w[baalHome] = 0x3c
	}

	if s.Worldstone {
		w[baalUnused] = 0
	}

	return w
}

func baalIdleTable(d Difficulty, s BaalSituation) [baalActions]int {
	w := [baalActions]int{}
	w[2], w[3], w[4], w[6] = 0x14, 0x14, 0x14, 0x14
	w[baalBuff], w[baalTentacles] = 0x50, 0x46

	base := 100
	if d == Normal {
		base = 125
	}

	if s.Targets < 2 {
		w[2] = 0x2d
		w[baalMelee] = 0x19
	}

	w[baalWait] = base + (100 - s.SelfHP)

	switch {
	case s.Flag1 && !s.Far:
		w[baalCold] = 0x32
		w[baalTeleport] = 0x32
	case s.Far:
		w[baalCold] = 0x19
		w[2], w[6] = 0, 0
		w[3] = 0x23
		w[baalNova] = 0x19
	}

	if s.VeryFar {
		w[baalHome] = 0x3c
	}

	return w
}

// baalPick: the exe sums the table in groups of four and rolls once;
// the result is the first index whose running total exceeds the roll, 1
// (wait) when none does.
func baalPick(b *Brain, w [baalActions]int) int {
	sum := 0
	for _, v := range w {
		sum += v
	}

	roll := 0
	if sum > 0 {
		roll = b.Roll(sum)
	}

	acc := 0

	for i, v := range w {
		acc += v
		if roll < acc {
			return i
		}
	}

	return baalWait
}

// BaalSituationHook is an optional World extension that fills the inputs the
// engine knows better than the AI.
type BaalSituationHook interface {
	BaalSituation(b *Brain, s *BaalSituation)
}

// baalDecide is FUN_005fb830: the continuing action in Scratch[0], the
// no-target action and the three tables.
func baalDecide(c *Ctx) int {
	b := c.B
	if b.Scratch[0] != 0 {
		return b.Scratch[0]
	}

	if c.Target == nil {
		// blocked footprint: wander (4); else a tentacle burst with 8%
		if c.W != nil {
			if f, ok := c.W.(interface{ Blocked(b *Brain) bool }); ok && f.Blocked(b) {
				return baalWander
			}
		}

		if b.Roll(100) < 8 {
			return baalTentacles
		}

		return baalWait
	}

	s := BaalSituation{
		InReach: c.InRange, Engaged: b.Aggressive, SelfHP: b.HPPercent, TargetHP: 100,
		Targets: 1, Dist: c.Dist, Nearby: 99,
	}

	if h, ok := c.W.(BaalSituationHook); ok {
		h.BaalSituation(b, &s)
	}

	return baalPick(b, BaalWeights(b.Diff, s))
}

// thinkBaalCrab is MONAI_Think_BaalCrab 0x5fc200 (VERIFIED structure): an anchor
// command at the spawn point, the decision, the action, then a 25-frame
// sleep that replaces the action's own wake-up (VERIFIED, FUN_005dcee0(0x19)).
// The exact threat scoring and the room-population test are UNVERIFIED (see
// BaalSituation); the weights are exact.
func thinkBaalCrab(c *Ctx) { baalThink(c, false) }

// thinkBaalCrabClone is MONAI_Think_BaalCrabClone 0x5fc440: dismissed when the
// leader is dead or it has no live target; actions 7, 9, 14 and 15 (wait,
// tentacles, teleport, clone) are replaced by 2 (walk); with no target it
// sleeps 15.
func thinkBaalCrabClone(c *Ctx) { baalThink(c, true) }

func baalThink(c *Ctx, clone bool) {
	b := c.B

	if clone {
		if b.Leader != nil && !b.Leader.Mode.IsAlive() || c.Target == nil && b.Leader == nil {
			dismiss(c)

			return
		}
	}

	if b.FindCommand(CmdAnchor) == nil {
		b.AppendCommand(Command{Type: CmdAnchor, X: b.X, Y: b.Y})
	}

	act := baalDecide(c)

	if clone && (act == baalUnused || act == baalTentacles || act == baalTeleport || act == baalClone) {
		act = baalWalk
	}

	if c.Target == nil && act != baalWander && act != baalHome && act != baalTentacles && act != baalWait {
		c.Sleep(15)

		return
	}

	baalAct(c, act)
	c.keepWake(25)
}

// keepWake is FUN_005dcee0(n): schedule the next think n frames from now
// whatever the action requested.
func (c *Ctx) keepWake(n int) {
	c.acted = true
	c.B.Wake = c.W.Frame() + n
}

// baalAct is FUN_005fbd50, the action switch (VERIFIED actions and skill ids).
// The skill of each action is taken from the monster's slot that holds it in
// monstats: Skill1 Baal Nova (316, mode 10), Skill2 Baal Inferno (317, mode 8),
// Skill3 Baal Tentacle (315, mode 9) and Skill4 Baal Cold Missiles (318, mode
// 4). Case 0xe (teleport towards the target) uses slot 5 and case 8 slots 6
// and 7; Baal has no skills there in the 1.14b table, so those do nothing.
func baalAct(c *Ctx, act int) {
	b := c.B
	p := b.Profile
	t := c.Target

	cast := func(slot int) {
		if t != nil && p.Skills[slot].Used() {
			c.Cast(slot, *t)

			return
		}

		c.Sleep(5)
	}

	switch act {
	case baalWait:
		w := 0x23
		switch b.Diff {
		case Nightmare:
			w = 0xf
		case Hell:
			w = 5
		}

		c.Sleep(w)
	case baalWalk, baalWalk2:
		if t != nil {
			c.WalkToRange(*t, 0xc, 0)
		}
	case baalCircle:
		if t != nil {
			c.Circle(*t, 6)
		}
	case baalWander:
		c.Wander(0x10)
	case baalHome:
		if a := b.FindCommand(CmdAnchor); a != nil {
			c.moveTo(Point{a.X, a.Y})
		}
	case baalTentacles:
		cast(slot3)
	case baalMelee:
		if t != nil {
			c.Attack(ModeAttack2, *t)
		}
	case baalNova:
		cast(slot1)
	case baalInferno:
		cast(slot2)
	case baalCold:
		cast(slot4)
	case baalBuff:
		cast(slot6)
	case baalTeleport:
		cast(slot5)
	case baalClone:
		if cl, ok := c.W.(Cloner); ok && cl.SpawnBaalClone(b) {
			c.Sleep(5)

			return
		}

		c.Sleep(5)
	default:
		c.Sleep(5)
	}
}
