package d2monster

// Archetypes added in the third AI pass: Vulture, Summoner, Duriel, Mephisto,
// Diablo, Izual, BaalMinion and SuicideMinion. Sources are the decompiled
// think functions (Game.exe 1.14b, read-only Ghidra) of monster-ai-2.md; the
// per-function comments say what is VERIFIED (control flow, constants and
// roll order read from the decompilation) and what is not.
//
// Skill slots: Skill1..8 of monstats are slots 0..7; the slot names given in
// the comments come from the 1.14b monstats.txt (patch_d2).

func init() {
	register("Vulture", TargetStandard, thinkVulture)
	register("Summoner", TargetStandard, thinkSummoner)
	register("Duriel", TargetStandard, thinkDuriel)
	register("Mephisto", TargetStandard, thinkMephisto)
	register("UberMephisto", TargetStandard, thinkMephisto)
	register("Diablo", TargetStandard, thinkDiablo)
	register("UberDiablo", TargetStandard, thinkDiablo)
	register("Izual", TargetStandard, thinkIzual)
	register("UberIzual", TargetStandard, thinkIzual)
	register("BaalMinion", TargetStandard, thinkBaalMinion)
	register("SuicideMinion", TargetStandard, thinkSuicideMinion)
}

// Skill slots beyond the first two (see ai_boss.go).
const (
	slot3 = 2
	slot4 = 3
	slot5 = 4
	slot6 = 5
	slot7 = 6
	slot8 = 7
)

// Optional World extensions of the boss AIs.

// ResistFinder gives the target's fire and cold resistance (stats 0x27 and
// 0x2b); the Summoner picks his element from them.
type ResistFinder interface {
	Resists(t Target) (fire, cold, lightning int)
}

// AuraStarter makes the monster's aura skill (Duriel's Holy Freeze) active.
type AuraStarter interface {
	StartAura(b *Brain, slot, level int)
}

// PreyFinder is the Vulture's corpse-and-wounded scan (FUN_005f20d0): the
// nearest other unit within radius subtiles whose life is at most pct percent
// of its maximum.
type PreyFinder interface {
	NearestPrey(b *Brain, radius, pct int) (Target, bool)
}

// Flier lets the engine show the Vulture's take-off and landing (collision
// layer 5 and 1 in the exe). Both are optional; the AI keeps its own
// Brain.Airborne flag.
type Flier interface {
	TakeOff(b *Brain)
	Land(b *Brain)
}

// StateClearer removes a unit state from the monster (Diablo drops the state
// his DiabLight breath leaves on him).
type StateClearer interface {
	ClearState(b *Brain, state int)
}

// LineOfSighter answers FUN_00622e40, "can the monster act on the target from
// here" (a straight line without a wall). Default: true.
type LineOfSighter interface {
	LineOfSight(b *Brain, t Target) bool
}

func sqDist(ax, ay, bx, by int) int {
	dx, dy := ax-bx, ay-by

	return dx*dx + dy*dy
}

// ---------------------------------------------------------------- Summoner

// thinkSummoner is MONAI_Think_Summoner 0x5f7720 (VERIFIED, re-read in this
// pass; it corrects the guesses of monster-ai-2.md 5.4).
//
// Slots (names from monstats.txt): Skill1 Glacial Spike, Skill2 Frost Nova,
// Skill3 Fire Ball, Skill4 VampireFirewall, Skill5 Weaken. Parameters: aip1
// act%, aip2 Weaken%, aip3 element-swap threshold, aip4 Frost Nova cooldown
// (frames), aip5 Fire Wall cooldown (frames), aip6 back-off% when the hero is
// within 5, aip7 Frost Nova range, aip8 missile range. Scratch[1] is the frame
// Frost Nova is ready again, Scratch[2] the frame the Fire Wall is.
//
// "Cold" is chosen when the hero's cold resistance is not higher than the fire
// resistance; a roll above aip3 flips the choice. The first-tick call
// FUN_00599f70 (Scratch[0]) is not ported (UNVERIFIED purpose).
func thinkSummoner(c *Ctx) {
	b, t := c.B, c.Target
	p := b.Profile
	now := c.W.Frame()

	if b.Scratch[0] == 0 {
		b.Scratch[0] = 1
	}

	if t == nil {
		c.Wander(4)

		return
	}

	if c.Dist < 5 && b.Chance(b.AIP(6)) {
		c.WalkAway(*t, 6) // no return: the cast below replaces the request
	}

	useCold := true
	if rf, ok := c.W.(ResistFinder); ok {
		fire, cold, _ := rf.Resists(*t)
		useCold = cold <= fire
	}

	if b.Chance(b.AIP(1)) {
		if p.Skills[slot5].Used() && b.Chance(b.AIP(2)) {
			c.Cast(slot5, *t)

			return
		}

		at, _, hasAT := c.W.AttackTarget(b)

		if b.Roll(100) > b.AIP(3) {
			useCold = !useCold
		}

		nova := func() bool {
			if p.Skills[slot2].Used() && now > b.Scratch[1] && c.Dist < b.AIP(7) {
				b.Scratch[1] = now + b.AIP(4)
				c.Cast(slot2, *t)

				return true
			}

			return false
		}

		if useCold {
			if nova() {
				return
			}

			if p.Skills[slot1].Used() && hasAT && c.Dist < b.AIP(8) {
				c.Cast(slot1, at)

				return
			}
		}

		if p.Skills[slot4].Used() && b.Scratch[2] < now {
			b.Scratch[2] = b.AIP(5) + now
			c.Cast(slot4, *t)

			return
		}

		if p.Skills[slot3].Used() && hasAT && c.Dist < b.AIP(8) {
			c.Cast(slot3, at)

			return
		}

		if !useCold && nova() {
			return
		}

		if p.Skills[slot5].Used() {
			c.Cast(slot5, *t)

			return
		}
	}

	c.Wander(4)
}

// ---------------------------------------------------------------- Duriel

// thinkDuriel is MONAI_Think_Duriel 0x5f5880 (VERIFIED). Slots: Skill1
// Charge, Skill2 Jab, Skill3 Smite, Skill4 Holy Freeze (his aura). aip1 is the
// aura's level argument, aip2 Smite%, aip3 Jab%, aip4 A2%, aip5 Charge%
// (empty in the 1.14b table, so he never charges).
//
// On the first tick the Holy Freeze aura is made active (the exe calls
// FUN_0056bc60(aip1,1) and SERVER_SetPlayerActiveSkill(Skill4) while
// FUN_00620480 says no skill is active; UNVERIFIED what that test reads, so
// the aura is started once, tracked in Scratch[0]).
func thinkDuriel(c *Ctx) {
	b, t := c.B, c.Target
	p := b.Profile

	if p.Skills[slot4].Used() && b.Scratch[0] == 0 {
		b.Scratch[0] = 1

		if as, ok := c.W.(AuraStarter); ok {
			as.StartAura(b, slot4, b.AIP(1))
		}
	}

	if t == nil {
		c.Sleep(10)

		return
	}

	if !c.InRange {
		if p.Skills[slot1].Used() && b.Roll(100) < b.AIP(5) {
			c.Cast(slot1, *t)

			return
		}

		c.SetSpeed(0)

		if !c.WalkTo(*t, meleeReach) {
			c.Sleep(10)
		}

		return
	}

	if p.Skills[slot3].Used() && b.Roll(100) < b.AIP(2) {
		c.Cast(slot3, *t)

		return
	}

	if p.Skills[slot2].Used() && b.Roll(100) < b.AIP(3) {
		c.Cast(slot2, *t)

		return
	}

	if b.Roll(100) < b.AIP(4) {
		c.Attack(ModeAttack2, *t)

		return
	}

	c.Attack(ModeAttack1, *t)
}

// ---------------------------------------------------------------- Mephisto

// Mephisto's behaviour phase, kept in Scratch[2] (AiGeneral+0x1c); Scratch[0]
// is the burst counter (+0x14), Scratch[1] the burst step flag (+0x18).
const (
	mephIdle      = 0
	mephBackOff   = 1
	mephBurst     = 2
	mephMelee     = 3
	mephApproach  = 4
	mephBurstMin  = 3
	mephHealthLow = 21
)

// thinkMephisto is MONAI_Think_Mephisto 0x5f6950. Slots (monstats.txt):
// Skill1 PrimeLightning, Skill2 PrimeBolt, Skill3 PrimePoisonNova, Skill4
// MephistoMissile, Skill5 MephFrostNova, Skill6 Blizzard; aip1 = chance of a
// cast burst when in reach.
//
// VERIFIED (decompiled): the phase selector and the roll order: k =
// max(0,(100-hp%)/5); out of reach and nearer than 21: in phase 0, below 21%
// life and within 5, 40% to back off with a Poison Nova; else a burst with
// chance 50+k (3..5 casts, one roll sets the length), otherwise the idle
// choice (roll>64 wait 10; roll>64 and near walk up; else circle); farther
// than 20: approach; in reach: roll>aip1 melee phase else burst.
//
// UNVERIFIED (read from the disassembly of the jump-table targets, which
// Ghidra has no function for; the slot choice and thresholds are exact, the
// glue calls are interpreted): the bodies of the phases. Burst: first step
// retreats 3, then each cast picks among the used slots in equal shares
// (share = 100/used): Blizzard when the hero is far (>30) and not in reach on
// nightmare/hell, Frost Nova when within 15 (n/nm/h only) in the first share,
// Mephisto Missile in the first two, Prime Bolt in the first three, Prime
// Lightning otherwise.
func thinkMephisto(c *Ctx) {
	b, t := c.B, c.Target
	p := b.Profile

	if t == nil {
		c.Sleep(10)

		return
	}

	hp := b.HPPercent
	k := (100 - hp) / 5

	if k < 0 {
		k = 0
	}

	phase := b.Scratch[2]

	if !c.InRange {
		switch {
		case c.Dist < 21:
			switch {
			case phase == mephIdle:
				if hp < mephHealthLow && c.Dist < 5 && b.Roll(100) < 40 {
					phase = mephBackOff

					break
				}

				if b.Roll(100) >= k+50 {
					mephIdleChoice(c, *t)

					return
				}

				phase = mephBurst
				b.Scratch[0] = b.Roll(3) + mephBurstMin
			case phase > mephApproach:
				mephIdleChoice(c, *t)

				return
			}
		default:
			phase = mephApproach
		}
	} else if b.Roll(100) > b.AIP(1) {
		phase = mephMelee
	} else {
		phase = mephBurst
		b.Scratch[0] = b.Roll(3) + mephBurstMin
	}

	switch phase {
	case mephIdle:
		b.Scratch[2] = mephApproach
		c.Sleep(5)
	case mephBackOff:
		b.Scratch[2] = mephIdle
		c.SetSpeed(50)

		if c.WalkAway(*t, 8) {
			return
		}

		if p.Skills[slot3].Used() {
			c.Cast(slot3, *t)
		} else {
			c.Sleep(10)
		}
	case mephBurst:
		mephBurstStep(c, *t, k)
	case mephMelee:
		b.Scratch[2] = mephIdle

		switch {
		case b.Roll(100) >= k+80:
			c.SetSpeed(50)

			if !c.WalkAway(*t, 3) {
				c.Sleep(10)
			}
		case b.Roll(100) >= 80-k && p.Skills[slot1].Used():
			c.Cast(slot1, *t)
		default:
			c.Attack(ModeAttack1, *t)
		}
	case mephApproach:
		b.Scratch[2] = mephIdle
		c.SetSpeed(50)

		if c.WalkTo(*t, 6) {
			return
		}

		if !c.WalkToRange(*t, 12, 6) {
			c.Sleep(10)
		}
	}
}

// mephIdleChoice is the "nothing to cast" tail of the selector (VERIFIED).
func mephIdleChoice(c *Ctx, t Target) {
	b := c.B
	b.Scratch[2] = mephIdle

	if b.Roll(100) > 0x40 {
		c.Sleep(10)

		return
	}

	if b.Roll(100) > 0x40 && c.Dist < 6 {
		c.SetSpeed(4)

		if c.WalkTo(t, meleeReach) {
			return
		}

		c.Sleep(10)

		return
	}

	if !c.Circle(t, 4) {
		c.Sleep(10)
	}
}

func mephBurstStep(c *Ctx, t Target, k int) {
	b := c.B
	p := b.Profile

	b.Scratch[0]--
	if b.Scratch[0] <= 0 {
		b.Scratch[2] = mephIdle
		b.Scratch[1] = 0
	} else {
		b.Scratch[2] = mephBurst
	}

	if b.Scratch[1] == 0 { // the first step of a burst steps back
		b.Scratch[1] = 2
		if b.Scratch[2] == mephBurst && c.WalkAway(t, 3) {
			return
		}
	} else {
		b.Scratch[1]++
	}

	used := 0

	for i := 0; i < 8; i++ {
		if !p.Skills[i].Used() {
			break
		}

		used++
	}

	if used == 0 {
		c.Attack(ModeAttack1, t)

		return
	}

	share := 100 / used
	r := b.Roll(100)
	hard := b.Diff > Normal

	switch {
	case hard && c.Dist > 30 && !c.InRange && p.Skills[slot6].Used():
		c.Cast(slot6, t) // Blizzard from afar
	case hard && c.Dist < 15 && r < share && p.Skills[slot5].Used():
		c.Cast(slot5, t) // Frost Nova up close
	case r < 2*share && p.Skills[slot4].Used():
		c.Cast(slot4, t)
	case r < 3*share && p.Skills[slot2].Used():
		c.Cast(slot2, t)
	case p.Skills[slot1].Used():
		c.Cast(slot1, t)
	default:
		c.Attack(ModeAttack1, t)
	}
}

// ---------------------------------------------------------------- Diablo

// Diablo's action codes: the result of the weighted decision (FUN_005e7710)
// is the index of the table below.
const (
	diaWalkHome  = 1
	diaA1        = 2
	diaA2        = 3
	diaAttack11  = 4  // mode 0xb strike
	diaLight     = 5  // Skill1 DiabLight (fire breath channel)
	diaFire      = 6  // Skill3 DiabFire
	diaCold      = 7  // Skill2 DiabCold
	diaWall      = 8  // Skill4 DiabWall
	diaPrison    = 9  // Skill7 DiabPrison
	diaRun       = 10 // Skill5 DiabRun
	diaWait      = 11 // default: wait
	diaCircle    = 12
	diaFirewall  = 13 // Skill6 PrimeFirewall
	diaRunHome   = 14
	diaPrisonAny = 15
	diaWander    = 16
	diaWeights   = 17
)

// diabloAnchorFar are the distances from his anchor beyond which he is
// "away from home" and "very far" (0x55 and 0x69, VERIFIED constants).
const (
	diabloAway    = 0x55
	diabloVeryFar = 0x69
)

// DiabloWeights returns the weight table of the decision for the current
// situation (VERIFIED constants of FUN_005e7710; the situation flags that
// depend on unread helpers are listed in the comment of thinkDiablo).
func DiabloWeights(inSight, engaged, awayFromHome, farFromHome bool, hp, dist, fireRes, lightRes, others int, hasState0xb bool) [diaWeights]int {
	var w [diaWeights]int

	switch {
	case inSight:
		w[diaA1], w[diaA2], w[diaLight], w[diaFire], w[diaCold], w[diaWall] = 40, 70, 40, 24, 40, 15

		if hp < 20 {
			w[diaA1] = 50
		}

		if hasState0xb {
			w[diaCold], w[diaWall] = 0, 0
		}

		if lightRes < fireRes {
			w[diaFire] -= 10
		}

		if fireRes < lightRes {
			w[diaFire] += 10
		}

		if !engaged {
			w[diaLight], w[diaFire] = 0, 0
		}
	case engaged:
		w[diaLight], w[diaFire], w[diaPrison], w[diaCircle], w[diaWall], w[diaRun] = 25, 25, 20, 20, 15, 10
		if dist > 25 {
			w[diaWall] -= 5
			w[diaRun] = 20
			w[diaLight] = 0
		}

		if lightRes < fireRes {
			w[diaFire] -= 10
			w[diaWall] -= 10
		}

		if fireRes < lightRes {
			w[diaFire] += 10
			w[diaWall] += 10
		}

		if others < 2 {
			w[diaFire] -= 10
		}

		if others > 3 {
			w[diaFire] += 5
		}

		if awayFromHome {
			w[diaWalkHome], w[diaCircle], w[diaRun], w[diaFirewall] = 0, 10, 0, 15
			if w[diaPrison] == 0 {
				w[diaPrison] = 10
			}
		}

		if farFromHome {
			w[diaPrison], w[diaRunHome] = 20, 60
		}
	default:
		w[diaAttack11], w[diaFire], w[diaWall], w[diaPrison], w[diaCircle] = 5, 25, 25, 40, 25

		if others < 2 {
			w[diaWall] -= 5
			w[diaFire] = 0
			w[diaWalkHome] = 25
			w[diaPrison] = 0
		}

		if awayFromHome {
			w[diaWalkHome], w[diaCircle], w[diaFirewall] = 0, 15, 25
			if w[diaPrison] == 0 {
				w[diaPrison] = 20
			}

			if others < 2 {
				w[diaPrison] -= 5
			}
		}

		if farFromHome {
			w[diaRunHome] = 60
		}
	}

	for i := range w {
		if w[i] < 0 {
			w[i] = 0
		}
	}

	return w
}

// pickWeighted rolls over a weight table exactly like the exe: one roll
// bounded by the sum, the first entry whose running total exceeds it wins;
// the fallthrough result is 11 (wait).
func pickWeighted(b *Brain, w []int) int {
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

	return diaWait
}

// thinkDiablo is MONAI_Think_Diablo 0x5e8150 plus the decision function
// FUN_005e7710 (VERIFIED structure and weights).
//
// First tick: an anchor command (type 10) at the spawn point is queued
// (VERIFIED). Scratch[0] holds the continuing action (the decision returns it
// unchanged while it is non-zero: the DiabLight channel, VERIFIED).
//
// Slots (monstats.txt): Skill1 DiabLight, Skill2 DiabCold, Skill3 DiabFire,
// Skill4 DiabWall, Skill5 DiabRun, Skill6 PrimeFirewall, Skill7 DiabPrison,
// Skill8 Diablogeddon (clone only: kept up as a buff, not ported).
//
// UNVERIFIED simplifications: the decision's target is the tick's target (the
// exe scores every hero in range with FUN_005e7420); "engaged" (FUN_005dbfd0)
// is Brain.Aggressive; the flags derived from the target's current skill
// (teleporting away) and the Chaos Sanctuary seal-position check are left out;
// "others" (heroes in range) is 1; fire/lightning resistance come from
// ResistFinder when the World has one. The levelled sleep of the default
// case (12/8/4 frames by difficulty) is VERIFIED.
func thinkDiablo(c *Ctx) {
	b, t := c.B, c.Target
	p := b.Profile

	anchor := b.FindCommand(CmdAnchor)
	if anchor == nil {
		b.AppendCommand(Command{Type: CmdAnchor, X: b.X, Y: b.Y})
		anchor = b.FindCommand(CmdAnchor)
	}

	if t == nil {
		c.Sleep(10)

		return
	}

	act := b.Scratch[0]

	if act == 0 {
		fire, light := 0, 0
		if rf, ok := c.W.(ResistFinder); ok {
			fire, _, light = rf.Resists(*t)
		}

		sight := true
		if ls, ok := c.W.(LineOfSighter); ok {
			sight = ls.LineOfSight(b, *t)
		}

		home := b.DistanceTo(anchor.X, anchor.Y)
		w := DiabloWeights(sight, b.Aggressive, home > diabloAway, home > diabloVeryFar,
			b.HPPercent, c.Dist, fire, light, 1, c.W.HasState(b, 0xb))
		act = pickWeighted(b, w[:])
	}

	cast := func(slot int) bool {
		if !p.Skills[slot].Used() {
			return false
		}

		b.Scratch[0] = 0

		return c.Cast(slot, *t)
	}

	switch act {
	case diaWalkHome:
		c.SetSpeed(0x14)
		c.moveTo(Point{b.X, b.Y})

		b.Scratch[0] = 0

		return
	case diaA1:
		b.Scratch[0] = 0
		c.Attack(ModeAttack1, *t)

		return
	case diaA2:
		b.Scratch[0] = 0
		c.Attack(ModeAttack2, *t)

		return
	case diaAttack11:
		b.Scratch[0] = 0
		c.Attack(Mode(0xb), *t)

		return
	case diaLight:
		if p.Skills[slot1].Used() {
			if !c.W.HasState(b, 0xc) {
				b.Scratch[0] = diaLight
				c.Cast(slot1, *t)

				return
			}
			// state 0xc (left by the breath) is cleared and the channel ends
			if sc, ok := c.W.(StateClearer); ok {
				sc.ClearState(b, 0xc)
			}
		}
	case diaFire:
		if cast(slot3) {
			return
		}
	case diaCold:
		if cast(slot2) {
			return
		}
	case diaWall:
		if cast(slot4) {
			return
		}
	case diaPrison, diaPrisonAny:
		if act == diaPrisonAny && !t.IsPlayer {
			c.Sleep(3)

			b.Scratch[0] = 0

			return
		}

		if cast(slot7) {
			return
		}
	case diaRun:
		if cast(slot5) {
			return
		}
	case diaFirewall:
		if cast(slot6) {
			return
		}
	case diaCircle:
		b.Scratch[0] = 0
		c.Circle(*t, 4)

		return
	case diaRunHome:
		c.SetSpeed(0x32)

		if c.moveTo(Point{anchor.X, anchor.Y}) {
			b.Scratch[0] = 0

			return
		}

		c.Sleep(2)
		b.Scratch[0] = 0

		return
	case diaWander:
		c.SetSpeed(0x14)
		c.Wander(5)

		b.Scratch[0] = 0

		return
	default:
		b.Scratch[0] = 0

		switch b.Diff {
		case Nightmare:
			c.Sleep(8)
		case Hell:
			c.Sleep(4)
		default:
			c.Sleep(12)
		}

		return
	}

	b.Scratch[0] = 0
	c.Sleep(2)
}

// ---------------------------------------------------------------- Izual

// thinkIzual is MONAI_Think_Izual 0x5f7b30 (VERIFIED). Slot: Skill1 Frost
// Nova. aip1 melee%, aip2 hold%, aip3 nova% when closing in, aip4 nova% when
// in reach, aip5 stall frames after a nova, aip6 melee swings that follow a
// nova. Scratch[0] first-tick flag (FUN_005b1fb0, UNVERIFIED), Scratch[1] the
// pending stall, Scratch[2] the swings left. The stall uses aidel (the sleep
// after a failed attack roll).
func thinkIzual(c *Ctx) {
	b, t := c.B, c.Target
	p := b.Profile

	if t == nil {
		c.Sleep(10)

		return
	}

	if b.Scratch[0] == 0 {
		b.Scratch[0] = 1
	}

	if b.Scratch[1] != 0 {
		n := b.Scratch[1]
		b.Scratch[1] = 0
		c.Sleep(n)

		return
	}

	nova := func() {
		c.Cast(slot1, *t)

		b.Scratch[1] = b.AIP(5)
		b.Scratch[2] = b.AIP(6)
	}

	if c.InRange {
		if b.Scratch[2] > 0 || b.Seed.Step()%100 < uint32(b.AIP(1)) {
			if b.Scratch[2] > 0 {
				b.Scratch[2]--
			}

			c.Attack(ModeAttack1, *t)

			return
		}

		b.Scratch[2] = 0

		if !p.Skills[slot1].Used() || b.Roll(100) >= b.AIP(4) {
			c.Sleep(p.AIDel)

			return
		}

		nova()

		return
	}

	if p.Skills[slot1].Used() && c.Dist < 10 {
		if b.Scratch[2] > 0 {
			c.WalkTo(*t, meleeReach)

			return
		}

		if b.Roll(100) < b.AIP(3) {
			nova()

			return
		}
	}

	if b.Scratch[2] < 1 && b.Seed.Step()%100 >= uint32(b.AIP(2)) {
		if c.Dist < 11 {
			c.Sleep(p.AIDel)

			return
		}

		c.WalkToRange(*t, 6, 9)

		return
	}

	c.WalkTo(*t, meleeReach)
}

// ---------------------------------------------------------------- BaalMinion

// thinkBaalMinion is MONAI_Think_BaalMinion 0x5eea30 (VERIFIED), the three
// Baal minions (Smite is Skill1, mode A2). aip1 attack%, aip2 approach%,
// aip3 Smite% / stall, aip4 the delay before the next think, which is
// scheduled after every action (FUN_005dcee0 is a plain sleep that replaces
// the action's own wake-up).
func thinkBaalMinion(c *Ctx) {
	b, t := c.B, c.Target
	p := b.Profile

	if t == nil || !c.InRange {
		if t != nil && b.Roll(100) < b.AIP(2) {
			c.WalkTo(*t, 1)
		}
	} else if b.Roll(100) < b.AIP(1) {
		if p.Skills[slot1].Used() && b.Roll(100) < b.AIP(3) {
			c.Cast(slot1, *t)
		} else {
			c.Attack(ModeAttack1, *t)
		}
	} else {
		c.Sleep(b.AIP(3))
	}

	c.Sleep(b.AIP(4))
}

// ---------------------------------------------------------------- SuicideMinion

// thinkSuicideMinion is MONAI_Think_SuicideMinion 0x5e0ba0 (VERIFIED; also
// alternate-table state 15). aip2 stall, aip3 approach%, aip5 fuse time. Once
// in reach it arms a fuse (Scratch[0] = frame + aip5); when the fuse is over
// it triggers its death mode (mode 0, the explosion).
func thinkSuicideMinion(c *Ctx) {
	b, t := c.B, c.Target
	now := c.W.Frame()

	if t == nil {
		c.Sleep(b.AIP(2))

		return
	}

	if fuse := b.Scratch[0]; fuse != 0 {
		if fuse < now {
			c.Attack(ModeDying, *t)

			return
		}

		c.Sleep(b.AIP(2))

		return
	}

	if !c.InRange {
		if b.Roll(100) < b.AIP(3) {
			c.WalkTo(*t, 0)

			return
		}

		c.Sleep(b.AIP(2))

		return
	}

	b.Scratch[0] = b.AIP(5) + now
	c.Sleep(b.AIP(2))
}

// ---------------------------------------------------------------- Vulture

// Vulture constants (VERIFIED).
const (
	vultureFarSq     = 0x90 // squared distance (12 subtiles) above which it takes off
	vultureTakeOff   = 60   // % chance per tick
	vultureLapsBase  = 0x18 // laps of the flight: 24 or 25
	vultureHoverStep = 12   // sleep between hover moves
	vultureNearSq    = 121  // squared radius of the prey scan (11)
)

// thinkVulture is MONAI_Think_Vulture 0x5f2150, the flying circler. aip1
// attack%, aip2 stall, aip3 prey hp% (wounded units it homes in on), aip4
// circle%, aip5 move%. Scratch[0] is the number of laps left (24..25 after
// take-off, -1 right after landing), Scratch[1]/Scratch[2] the current
// waypoint.
//
// VERIFIED: the lap counter flow, the take-off rule (on the ground, no leader,
// hero farther than 12 and 60%: take off, laps = 24+coin, fly to within 8),
// the waypoint picking (a random point within (laps+8) of the target, pushed
// out to at least clamp(2*laps, 12, 36) from the vulture), landing when the
// laps run out (or it is within 5 and a roll of 15%), and the ground tail
// (strike aip1%, else stall aip2 / circle aip4% / approach).
//
// UNVERIFIED: FUN_0064fb50 (a line test applied to the waypoint) is not
// ported, so a pending waypoint is simply kept until reached; the vulture's
// S1 "flying" move animation is a plain walk; hovering over a wounded unit
// needs the World's PreyFinder, without which no prey is ever found.
func thinkVulture(c *Ctx) {
	b, t := c.B, c.Target
	p := b.Profile
	laps := b.Scratch[0]

	fl, _ := c.W.(Flier)

	takeOff := func() {
		b.Airborne = true

		if fl != nil {
			fl.TakeOff(b)
		}
	}

	land := func() bool {
		if !b.Airborne {
			return false
		}

		b.Airborne = false

		if fl != nil {
			fl.Land(b)
		}

		return true
	}

	if t == nil {
		if laps < 1 {
			c.Sleep(vultureHoverStep)

			return
		}

		if land() {
			b.Scratch[0] = -1
			c.Sleep(vultureHoverStep)

			return
		}
	}

	var sq int
	if t != nil {
		sq = sqDist(b.X, b.Y, t.X, t.Y)
	}

	if laps == 0 && b.Leader == nil && t != nil && sq > vultureFarSq && b.Roll(100) < vultureTakeOff {
		takeOff()

		b.Scratch[0] = vultureLapsBase + int(b.Aux.Roll(2))
		c.RunToRange(*t, 8, 8)
		c.Sleep(vultureHoverStep)

		return
	}

	var prey bool

	if pf, ok := c.W.(PreyFinder); ok && t != nil {
		_, prey = pf.NearestPrey(b, 11, b.AIP(3))
	}

	switch {
	case laps >= 2:
		if t != nil && !prey && (c.Dist > 5 || b.Roll(100) > 14) {
			takeOff()

			wx, wy := b.Scratch[1], b.Scratch[2]
			if wx == 0 || wy == 0 || sqDist(b.X, b.Y, wx, wy) < 4 {
				k := laps + 8
				wx = t.X - k + b.Roll(2*k)
				wy = t.Y - k + b.Roll(2*k)

				min := clamp(laps*2, 12, 36)
				for guard := 0; guard < 64 && EdgeDistance(b.X-wx, b.Y-wy, 0) < min; guard++ {
					wx += sign(wx - b.X)
					wy += sign(wy - b.Y)

					if wx == b.X && wy == b.Y {
						wx++
						wy++
					}
				}

				b.Scratch[1], b.Scratch[2] = wx, wy
			}

			b.Scratch[0] = laps - 1

			c.SetSpeed(0)
			c.moveTo(Point{b.Scratch[1], b.Scratch[2]})
			c.Sleep(vultureHoverStep)

			return
		}

		b.Scratch[0], laps = 1, 1

		fallthrough
	case laps == 1:
		if land() && t != nil {
			c.RunToRange(*t, 9, 2)
			c.Sleep(vultureHoverStep)

			b.Scratch[0] = -1

			return
		}

		b.Scratch[0] = 8

		c.Sleep(vultureHoverStep)
	}

	if t == nil {
		c.Sleep(vultureHoverStep)

		return
	}

	if c.InRange {
		if b.Chance(b.AIP(1)) {
			c.Attack(ModeAttack1, *t)
		} else {
			c.Sleep(b.AIP(2))
		}

		return
	}

	if laps != -1 && b.Roll(100) >= b.AIP(5) {
		c.Sleep(b.AIP(2))

		return
	}

	if b.Roll(100) < b.AIP(4) {
		c.Circle(*t, 6)
	} else {
		c.WalkToRange(*t, 9, 0)
	}

	b.Scratch[0] = 0
	c.Sleep(vultureHoverStep)

	_ = p
}
