package d2monster

// FallenShaman (0x5f04d0) and the forced states 13 (0x5e4be0) and 16
// (0x5e1c60). Sources: Game.exe 1.14b, read-only Ghidra; the comments say what
// is VERIFIED and what is not.

func init() {
	register("FallenShaman", TargetStandard, thinkFallenShaman)
	// Both rows are target mode 1 in the exe's state table (VERIFIED, 0x73a548
	// rows 13 and 16): with nobody inside the aggro radius the tick idles.
	regState(StateLeash, TargetStandard, thinkState13)
	regState(StateImp, TargetStandard, thinkState16)
}

// Classes the Shaman resurrects: exe monster classes 19 (Fallen) and 58
// (FallenShaman) by MONSTER_GetBaseClassId (VERIFIED, FUN_005f0420).
var shamanCorpseClasses = []int{19, 58}

// CorpseFinder is an optional extension of Senses: the nearest dead unit of one
// of classes whose corpse can be raised, within maxDist subtiles (the exe
// filter FUN_005f0420 also needs: the corpse flag set, not already being
// resurrected, mode DD).
type CorpseFinder interface {
	NearestCorpse(b *Brain, classes []int, maxDist int) (Target, bool)
}

// thinkFallenShaman is MONAI_Think_FallenShaman 0x5f04d0 (VERIFIED, re-read:
// the roll order below is the decompiled one). aip1 resurrect%, aip2 fire%,
// aip3 melee%, aip4 corpse search radius, aip5 fire range. Slots: Skill1
// Resurrect, Skill2 Shaman Fire (mode of slot 2).
//
// The exe skips the corpse scan under a game-state test (FUN_0059dd60, two
// calls; UNVERIFIED what it tests); the corpse scan is a no-op without the
// World's CorpseFinder.
func thinkFallenShaman(c *Ctx) {
	b := c.B
	p := b.Profile
	t := c.Target

	if c.InRange && t != nil && b.Roll(100) < b.AIP(3) {
		c.Attack(ModeAttack1, *t)

		return
	}

	var corpse *Target

	if cf, ok := c.W.(CorpseFinder); ok {
		if ct, found := cf.NearestCorpse(b, shamanCorpseClasses, b.AIP(4)); found {
			corpse = &ct
		}
	}

	if b.Roll(100) < b.AIP(1) {
		b.Broadcast(Command{Type: CmdAlert, Count: 1})
	}

	if corpse != nil && b.Roll(100) < b.AIP(1) && p.Skills[slot1].Used() {
		c.Cast(slot1, *corpse)

		return
	}

	if t != nil && p.Skills[slot2].Used() && c.Dist < b.AIP(5) && b.Roll(100) < b.AIP(2) {
		c.Cast(slot2, *t)

		return
	}

	if t2, d, ok := c.W.AttackTarget(b); ok && d < b.AIP(5) && p.Skills[slot2].Used() && b.Roll(100) < b.AIP(2) {
		c.Cast(slot2, t2)

		return
	}

	if b.AIP(3) <= b.Roll(100) {
		c.Sleep(10)

		return
	}

	if t != nil {
		c.Circle(*t, 3)

		return
	}

	c.Sleep(10)
}

// RoomChecker is an optional extension of Senses: whether two subtile points
// lie in the same room (FUN_0061ae80 maps a point to its room/region id).
type RoomChecker interface {
	SameRoom(b *Brain, a, z Point) bool
}

// stateLeashRadius is the vertical-distance cutoff of State13 (0x28, VERIFIED
// constant, applied to FUN_005db310 of the anchor's y: UNVERIFIED which axis).
const stateLeashRadius = 0x28

// thinkState13 is MONAI_State13_Think 0x5e4be0, a leashed guard: it keeps to
// the room of its anchor (command type 10, created at its own position on the
// first tick; VERIFIED) and strikes anything that enters it.
//
// VERIFIED: needs a profile; the anchor repair (a zero anchor takes the
// monster's position); when the monster is not in the anchor's room it walks
// back; when the target is not in the anchor's room either, the monster sits on
// the anchor, using its Skill1 ring cast (FUN_005e4b10) if fewer than 25
// casts were made, else sleeps 10; the final rolls: in reach, aip3+10%
// attack mode A2 (mode 4 in the exe's queue = A1 here) else stall aip2; out of
// reach aip1% run (speed 100) else stall aip2.
// UNVERIFIED: the room test (default: within 20 subtiles of the anchor), the
// ring-cast helper (not ported: always "not cast"), the two unknown calls
// FUN_005a3c60/FUN_0053eb80 around the walk back.
func thinkState13(c *Ctx) {
	b := c.B
	p := b.Profile
	t := c.Target

	if p == nil {
		return
	}

	a := b.FindCommand(CmdAnchor)
	if a == nil {
		b.AppendCommand(Command{Type: CmdAnchor, X: b.X, Y: b.Y})
		a = b.FindCommand(CmdAnchor)
	}

	if a.X == 0 || a.Y == 0 {
		a.X, a.Y = b.X, b.Y
	}

	anchor := Point{a.X, a.Y}
	same := func(x, y Point) bool {
		if rc, ok := c.W.(RoomChecker); ok {
			return rc.SameRoom(b, x, y)
		}

		return Distance(x.X-y.X, x.Y-y.Y) <= 20
	}

	if !same(anchor, Point{b.X, b.Y}) {
		if c.moveTo(anchor) {
			return
		}
	}

	if t == nil {
		c.Sleep(10)

		return
	}

	if !same(anchor, Point{t.X, t.Y}) {
		if b.X == anchor.X && b.Y == anchor.Y {
			c.Sleep(10)

			return
		}

		if c.moveTo(anchor) {
			return
		}
	}

	if abs(anchor.Y-b.Y) > stateLeashRadius {
		if c.moveTo(anchor) {
			return
		}
	}

	if c.InRange {
		if b.Roll(100) < b.AIP(3)+10 {
			c.Attack(ModeAttack1, *t)

			return
		}

		c.Sleep(b.AIP(2))

		return
	}

	if b.Roll(100) < b.AIP(1) {
		c.SetSpeed(100)
		c.RunTo(*t, 0)

		return
	}

	c.Sleep(b.AIP(2))
}

func abs(v int) int {
	if v < 0 {
		return -v
	}

	return v
}

// ImpStateID is the unit state that keeps a monster in AI state 16 (0x8f,
// VERIFIED constant: the think returns the class AI as soon as it is gone).
const ImpStateID = 0x8f

// thinkState16 is MONAI_State16_Think 0x5e1c60, the AI of an imp-class monster
// right after its teleport (the skill Imp Teleport puts the state 0x8f on it;
// the Pre hook 0x5e1bc0 rolls a coin and picks one of two (skill, count) pairs
// from a table at 0x6e4a70 into Scratch[1]/Scratch[2], or zeroes both when the
// coin's candidates are not available; the table contents are UNVERIFIED).
//
// VERIFIED: without the state or without a living target the class AI is
// restored (SetAiState(0)) and the think sleeps 1. With both, within 25
// subtiles and while the pre-chosen skill (Scratch[1]) is set, a roll under
// the chosen record's aip2 casts it at the target; otherwise a 50% strike
// (mode 8) or a 20-frame wait. Farther than that, the monster casts its
// first skill and drops the Scratch marker. The record whose profile is read
// is the imp row of the exe (class 0x1ed); here the monster's own profile.
func thinkState16(c *Ctx) {
	b := c.B
	p := b.Profile
	t := c.Target

	if t == nil || !c.W.HasState(b, ImpStateID) {
		b.ClearForced(c.W.Frame())
		c.Sleep(1)

		return
	}

	if c.Dist < 25 && b.Scratch[1] != 0 {
		slot := b.Scratch[1] - 1
		if slot >= 0 && slot < NumSkills && p.Skills[slot].Used() && c.Dist < b.AIP(1) && b.Roll(100) < b.AIP(2) {
			c.Cast(slot, *t)

			return
		}

		if b.Roll(100) < 50 {
			c.Attack(ModeSkill1, *t)

			return
		}

		c.Sleep(20)

		return
	}

	if p.Skills[slot1].Used() {
		b.Scratch[1] = 0
		c.Cast(slot1, *t)

		return
	}

	c.Sleep(20)
}
