package d2monster

// Ground-monster archetypes ported in the monster-ai-4 pass: Quill Rat,
// Panther Woman, Sand Leaper, Sand Raider, Sand Maggot, Greater Mummy (Radament
// and the Unravelers), Fetish / Fetish Blowgun / Fetish Shaman, Bat Demon,
// Doom Knight / Abyss Knight, Venom Lord (Megademon), Succubus and Minion.
//
// Sources: notes monster-ai-2.md section 4.8 plus a fresh read of each think
// function in Game.exe 1.14b (read-only Ghidra). aip<n> are the monai.txt
// columns (stored as shorts at +0x56 + 6*(n-1) + 2*difficulty in monstats).
// Helper calls whose meaning is not recorded are marked UNVERIFIED and use the
// documented stand-in.

func init() {
	register("QuillRat", TargetStandard, thinkQuillRat)
	register("PantherWoman", TargetStandard, thinkPantherWoman)
	register("SandLeaper", TargetStandard, thinkSandLeaper)
	register("SandRaider", TargetStandard, thinkSandRaider)
	register("SandMaggot", TargetFindOrWait, thinkSandMaggot) // exe mode 4 (VERIFIED)
	register("GreaterMummy", TargetStandard, thinkGreaterMummy)
	register("Fetish", TargetStandard, thinkFetish)
	register("FetishBlowgun", TargetStandard, thinkFetishBlowgun)
	register("FetishShaman", TargetStandard, thinkFetishShaman)
	register("BatDemon", TargetStandard, thinkBatDemon)
	register("DoomKnight", TargetStandard, thinkDoomKnight)
	register("AbyssKnight", TargetStandard, thinkAbyssKnight)
	register("Megademon", TargetStandard, thinkMegademon)
	register("Succubus", TargetStandard, thinkSuccubus)
	register("Minion", TargetStandard, thinkMinion)
}

// CmdFollow is the AiGeneral command type 0xe: "walk to this unit" broadcast
// by the Fetish Shaman (VERIFIED value).
const CmdFollow = 0xe

// commandedUnit resolves the unit named by a queued command, when the world
// can (SourceFinder). The exe reads the id at +0xc (Fetish) or +0x10 (Quill
// Rat); Command.Target holds it for both.
func (c *Ctx) commandedUnit(cmd *Command) (Target, bool) {
	sf, ok := c.W.(SourceFinder)
	if !ok || cmd == nil {
		return Target{}, false
	}

	t, _, found := sf.UnitTarget(c.B, cmd.Target)

	return t, found
}

// thinkQuillRat is MONAI_Think_QuillRat 0x5f0200 (VERIFIED). aip1 kite
// distance, aip2 shoot%, aip4 walk-away / wander distance. A2 is the spike
// shot. FUN_005dbff0 is the aggressive flag (VERIFIED: pUnitData+0x54 in {3,0x13}; matches
// the notes).
func thinkQuillRat(c *Ctx) {
	b, t := c.B, *c.Target

	if cmd := b.PeekCommand(); cmd != nil {
		_, found := c.commandedUnit(cmd)
		b.PopCommand()

		if found {
			c.Attack(ModeAttack2, t)

			return
		}
	}

	if c.InRange {
		c.Attack(ModeAttack1, t)

		return
	}

	if b.Aggressive {
		c.Attack(ModeAttack2, t)

		return
	}

	if c.Dist < b.AIP(1) {
		if b.Chance(b.AIP(2)) {
			c.Attack(ModeAttack2, t)

			return
		}

		if c.WalkAway(t, b.AIP(4)) {
			return
		}

		if c.Dist < 4 {
			c.Attack(ModeAttack2, t)

			return
		}
	}

	n := b.AIP(4)
	if n < 3 {
		n = 3
	}

	c.Wander(n)
}

// thinkPantherWoman is MONAI_Think_PantherWoman 0x5f1300 (VERIFIED).
// aip1 approach%, aip2 attack%, aip3 regroup distance, aip4 stall.
func thinkPantherWoman(c *Ctx) {
	b, t := c.B, *c.Target

	if c.InRange {
		if b.Chance(b.AIP(2)) {
			c.Attack(ModeAttack1, t)

			return
		}

		c.Sleep(b.AIP(4))

		return
	}

	if b.Chance(b.AIP(1)) {
		c.SetSpeed(75)

		if c.WalkTo(t, meleeReach) {
			return
		}
	} else if f, ok := c.W.(AllyFinder); ok {
		if ally, d, found := f.NearestAlly(b); found && d*d > b.AIP(3)*b.AIP(3) {
			c.SetSpeed(75)

			if c.WalkTo(ally, meleeReach) {
				return
			}
		}
	}

	if b.Chance(25) && c.Circle(t, 3) {
		return
	}

	c.Sleep(b.AIP(4))
}

// thinkSandLeaper is MONAI_Think_SandLeaper 0x5f1100 (VERIFIED from the
// notes). Skill1 is the leap. aip1 leap%, aip2 bite%, aip3 approach%, aip4
// circle%.
func thinkSandLeaper(c *Ctx) {
	b, t := c.B, *c.Target
	p := b.Profile

	if c.Dist < 5 && p.Skills[slot1].Used() && b.Chance(b.AIP(1)) {
		c.Cast(slot1, t)

		return
	}

	if !c.InRange {
		if c.Dist > 10 {
			c.SetSpeed(75)

			if c.WalkNearTarget(&t, 5) {
				return
			}
		}

		if b.Chance(b.AIP(3)) && c.WalkTo(t, meleeReach) {
			return
		}

		if b.Chance(b.AIP(4)) && c.Circle(t, 4) {
			return
		}
	} else if b.Chance(b.AIP(2)) {
		c.Attack(ModeAttack2, t)

		return
	}

	c.Sleep(10)
}

// Sand Raider scratch: charge counter (+0x14), "charged" flag (+0x18), defend
// attempts (+0x1c).
const (
	raiderCount   = 0
	raiderCharged = 1
	raiderDefend  = 2
)

// Unit states the Sand Raider toggles (VERIFIED ids in 0x5ef800 through
// FUN_0063aef0; the colour meaning is UNVERIFIED).
const (
	raiderStateA = 0x5a
	raiderStateB = 0x5b
)

// StateSetter is an optional extension of Actor: FUN_0063aef0(unit, state, on).
type StateSetter interface {
	SetUnitState(b *Brain, state int, on bool)
}

// thinkSandRaider is MONAI_Think_SandRaider 0x5ef800 (VERIFIED, re-read from
// the disassembly in this pass; the roll order below is the exe's). aip1 hurt%
// below which it runs to an ally, aip2 strafe%, aip3 melee gate%, aip4 approach
// %, aip5 charge ticks, aip6 selects which of two states/sounds, aip7 A2%.
//
//  1. counter == 0: clear states 0x5a/0x5b and the charged flag. counter++.
//  2. counter == aip5: (a FUN_00622020 call, UNVERIFIED, not ported) sleep
//     aidel+1, end. counter > aip5: set state 0x5a (aip6 == 1) or 0x5b, flag.
//  3. defend < 7 and hp% < aip1: walk to the nearest ally if any (end), else
//     defend++. (no roll)
//  4. dist > 4 and not charged: roll(100) < aip2 -> Circle, end.
//  5. in range: charged and Skill1 set -> cast it, counter = flag = 0, end;
//     roll(100) < aip3 -> roll(100) < aip7 ? A2 : A1, end; else step 7.
//     not in range: charged -> walk to the target; else roll(100) < aip4 ->
//     walk, else step 7.
//  7. if counter > max(aip5+6, 24): counter = flag = 0. sleep 15.
func thinkSandRaider(c *Ctx) {
	b, t := c.B, *c.Target
	p := b.Profile

	setState := func(s int, on bool) {
		if ss, ok := c.W.(StateSetter); ok {
			ss.SetUnitState(b, s, on)
		}
	}

	if b.Scratch[raiderCount] == 0 {
		setState(raiderStateA, false)
		setState(raiderStateB, false)
		b.Scratch[raiderCharged] = 0
	}

	b.Scratch[raiderCount]++

	switch k := b.Scratch[raiderCount]; {
	case k == b.AIP(5):
		c.Sleep(p.AIDel + 1)

		return
	case k > b.AIP(5):
		if b.AIP(6) == 1 {
			setState(raiderStateA, true)
		} else {
			setState(raiderStateB, true)
		}

		b.Scratch[raiderCharged] = 1
	}

	if b.Scratch[raiderDefend] < 7 && b.HPPercent < b.AIP(1) {
		if f, ok := c.W.(AllyFinder); ok {
			if ally, _, found := f.NearestAlly(b); found && c.WalkTo(ally, 0) {
				return
			}
		}

		b.Scratch[raiderDefend]++
	}

	charged := b.Scratch[raiderCharged] == 1

	if c.Dist > 4 && !charged && b.Roll(100) < b.AIP(2) {
		c.Circle(t, 4)

		return
	}

	if c.InRange {
		if charged && p.Skills[slot1].Used() {
			b.Scratch[raiderCount], b.Scratch[raiderCharged] = 0, 0
			c.Cast(slot1, t)

			return
		}

		if b.Roll(100) < b.AIP(3) {
			if b.Roll(100) < b.AIP(7) {
				c.Attack(ModeAttack2, t)
			} else {
				c.Attack(ModeAttack1, t)
			}

			return
		}
	} else if charged || b.Roll(100) < b.AIP(4) {
		if !c.WalkTo(t, 0) {
			c.Sleep(10)
		}

		return
	}

	extra := 24 - b.AIP(5)
	if extra < 6 {
		extra = 6
	}

	lim := b.AIP(5) + extra
	if b.Scratch[raiderCount] > lim {
		b.Scratch[raiderCount], b.Scratch[raiderCharged] = 0, 0
	}

	c.Sleep(15)
}

// Sand Maggot scratch: phase (0x14), cooldown-until-frame (0x18), spit count
// (0x1c).
const (
	maggotPhase = 0
	maggotCool  = 1
	maggotCount = 2
)

// thinkSandMaggot is MONAI_Think_SandMaggot 0x5f0860 (VERIFIED flow). It runs
// without a target too. aip1 burrow%, aip2 spit%, aip3 spit budget, aip4
// bite%, aip5 skill cooldown frames. Slots: 1 = emerge/surface, 2 = spit
// burst, 3 = lay eggs (names UNVERIFIED). FUN_005dbfc0 < 25 is read as
// hp% < 25 (UNVERIFIED). Waits use ScheduleWait via Sleep.
func thinkSandMaggot(c *Ctx) {
	b, t := c.B, c.Target
	p := b.Profile
	now := c.W.Frame()

	att, d, haveAtt := c.W.AttackTarget(b)
	phase := b.Scratch[maggotPhase]
	near := haveAtt && d < 16

	cast := func(slot int, at Target, nextPhase int) {
		if c.Cast(slot, at) {
			b.Scratch[maggotCool] = b.AIP(5) + now
			b.Scratch[maggotPhase] = nextPhase
		}
	}

	// target-less surfacing (phase < 3 only)
	if phase < 3 && t == nil && (!haveAtt || d > 10) && b.Scratch[maggotCool] < now && p.Skills[slot2].Used() {
		cast(slot2, Target{}, 3)

		return
	}

	if phase == 3 && (t != nil || near) {
		at := att
		if t != nil {
			at = *t
		}

		if b.Scratch[maggotCool] < now && p.Skills[slot1].Used() {
			cast(slot1, at, 1)

			return
		}
	}

	if phase == 3 {
		c.Sleep(20)

		return
	}

	if t != nil && b.HPPercent < 25 && p.Skills[slot2].Used() && c.Dist < 7 && b.Scratch[maggotCool] < now &&
		b.Roll(100) < 20 {
		cast(slot2, *t, 3)

		return
	}

	if t != nil && c.InRange && b.Chance(b.AIP(4)) {
		c.Attack(ModeAttack1, *t)

		return
	}

	if haveAtt && d < 15 && b.Roll(100) < b.AIP(2) {
		c.Attack(ModeAttack2, att)

		return
	}

	if t == nil {
		c.Sleep(12)

		return
	}

	if b.Roll(100) < 20 {
		c.Circle(*t, 6)

		return
	}

	if b.Scratch[maggotCount] < b.AIP(3) && b.Roll(100) < b.AIP(1) {
		if phase != 2 || !p.Skills[slot4].Used() {
			c.Circle(*t, 6)
			b.Scratch[maggotPhase] = 2

			return
		}

		b.Scratch[maggotCount]++
		b.Scratch[maggotPhase] = 1
		c.Cast(slot4, *t)

		return
	}

	c.Sleep(12)
}

// thinkGreaterMummy is MONAI_Think_GreaterMummy 0x5f1b00 (VERIFIED flow;
// Radament and the Unravelers). aip1 attack%, aip2 skill1 (resurrect) %, aip3
// skill2 %, aip4 skill3 %, aip5 corpse search radius. The exe scans for a
// corpse with FUN_005dbe00(filter 0x5f1a00) - here the CorpseFinder with the
// Mummy classes is not available, so the corpse is any corpse of the unit's
// own class family (UNVERIFIED filter). In hell the radius is aip5+10
// (the exe's difficulty-0 / game-id quirk is not reproduced).
func thinkGreaterMummy(c *Ctx) {
	b, t := c.B, *c.Target
	p := b.Profile

	if c.InRange && b.Chance(b.AIP(1)) {
		c.Attack(ModeAttack1, t)

		return
	}

	if c.Dist < 5 && b.Chance(b.AIP(1)) {
		c.Attack(ModeAttack2, t)

		return
	}

	var corpse *Target

	if cf, ok := c.W.(CorpseFinder); ok {
		if ct, found := cf.NearestCorpse(b, []int{b.Class}, b.AIP(5)); found {
			corpse = &ct
		}
	}

	if corpse != nil && p.Skills[slot2].Used() && b.Roll(100) < b.AIP(3) {
		c.Cast(slot2, *corpse)

		return
	}

	if corpse != nil && b.Roll(100) < b.AIP(2) && p.Skills[slot1].Used() {
		c.Cast(slot1, *corpse)

		return
	}

	if p.Skills[slot3].Used() && b.Roll(100) < b.AIP(4) {
		if at, _, ok := c.W.AttackTarget(b); ok {
			c.Cast(slot3, at)

			return
		}
	}

	if corpse == nil {
		c.SetSpeed(50)

		if c.WalkTo(t, 3) {
			return
		}
	}

	if b.Roll(100) > 49 {
		c.Sleep(6)

		return
	}

	c.Circle(t, 3)
}

// Fetish scratch: phase (0x14), counter (0x18).
const (
	fetPhase = 0
	fetCount = 1
)

// fetishCommand handles the group commands shared by Fetish and Blowgun:
// type 1 attacks the commanded unit, type 0xe walks to it (VERIFIED). It
// returns true when it acted.
func fetishCommand(c *Ctx, attackOnCmd1 bool) bool {
	b := c.B

	cmd := b.PeekCommand()
	if cmd == nil {
		return false
	}

	u, found := c.commandedUnit(cmd)

	switch {
	case cmd.Type == CmdAlert && found:
		b.Scratch[fetPhase], b.Scratch[fetCount] = 0, 0

		if attackOnCmd1 {
			c.Attack(ModeAttack1, u)
		} else {
			c.SetSpeed(50)
			c.WalkTo(u, 0)
		}
	case cmd.Type == CmdFollow && found:
		b.Scratch[fetPhase], b.Scratch[fetCount] = 0, 0
		c.SetSpeed(50)
		c.WalkTo(u, 0)
	default:
		b.PopCommand()

		return false
	}

	b.PopCommand()

	return true
}

// thinkFetish is MONAI_Think_Fetish 0x5f4500 (VERIFIED). 3-phase scratch:
// 0 approach/attack, 1 attack loop (aip3 attacks, then flee when the target's
// hit points exceed aip4 - FUN_00622100 read as hp%, UNVERIFIED), 2 flee.
// aip1 attack%, aip2 stall. Type 1 and 0xe commands walk to the commanded
// unit.
func thinkFetish(c *Ctx) {
	b, t := c.B, *c.Target

	if cmd := b.PeekCommand(); cmd != nil {
		u, found := c.commandedUnit(cmd)
		if found && (cmd.Type == CmdAlert || cmd.Type == CmdFollow) {
			b.Scratch[fetPhase], b.Scratch[fetCount] = 0, 0
			c.SetSpeed(50)
			c.WalkTo(u, 0)
			b.PopCommand()

			return
		}

		b.PopCommand()
	}

	switch b.Scratch[fetPhase] {
	case 0:
		if !c.InRange {
			c.SetSpeed(50)
			c.WalkTo(t, meleeReach)

			return
		}

		b.Scratch[fetCount], b.Scratch[fetPhase] = 0, 1

		if b.Roll(100) >= b.AIP(1) {
			c.Sleep(b.AIP(2))

			return
		}

		c.Attack(ModeAttack1, t)
	case 1:
		b.Scratch[fetCount]++

		if b.Scratch[fetCount] > b.AIP(3) && b.AIP(4) < b.HPPercent {
			b.Scratch[fetPhase], b.Scratch[fetCount] = 2, 0
			c.SetSpeed(50)
			c.RunAway(t, 14)

			return
		}

		if !c.InRange {
			c.SetSpeed(50)
			c.WalkTo(t, meleeReach)

			return
		}

		if b.Roll(100) >= b.AIP(1) {
			c.Sleep(b.AIP(2))

			return
		}

		c.Attack(ModeAttack1, t)
	case 2:
		if c.Dist > 12 {
			b.Scratch[fetCount]++
			if b.Scratch[fetCount] > 1 {
				b.Scratch[fetPhase], b.Scratch[fetCount] = 0, 0
			}

			if b.Roll(100) > 19 {
				c.Sleep(10)

				return
			}

			c.Circle(t, 4)

			return
		}

		c.SetSpeed(50)

		if c.WalkAway(t, 14) {
			return
		}

		b.Scratch[fetPhase], b.Scratch[fetCount] = 0, 0
		c.Sleep(10)
	default:
		c.Sleep(10)
	}
}

// thinkFetishBlowgun is MONAI_Think_FetishBlowgun 0x5e00b0 (VERIFIED). Scratch
// phase: 0 shoot-and-count, 1 reposition, 2 kite away. aip1 regroup distance,
// aip2 kite%.
func thinkFetishBlowgun(c *Ctx) {
	b, t := c.B, *c.Target

	if fetishCommand(c, true) {
		return
	}

	if !c.InRange {
		if b.AIP(1) < c.Dist {
			c.SetSpeed(50)
			c.WalkNearTarget(&t, 6)

			return
		}

		if c.Dist < 6 && b.Roll(100) < b.AIP(2) {
			b.Scratch[fetPhase] = 2
		}
	} else if b.Roll(100) < b.AIP(2) {
		b.Scratch[fetPhase] = 2
	}

	at, _, haveAt := c.W.AttackTarget(b)

	switch b.Scratch[fetPhase] {
	case 0:
		b.Scratch[fetCount]++

		if b.Roll(3)+3 < b.Scratch[fetCount] {
			b.Scratch[fetPhase], b.Scratch[fetCount] = 1, 0
		}

		if haveAt {
			c.Attack(ModeAttack1, at)

			return
		}

		c.Circle(t, 4)
	case 1:
		b.Scratch[fetPhase], b.Scratch[fetCount] = 0, 0
		c.SetSpeed(50)

		if !c.Circle(t, 6) {
			c.Sleep(10)
		}
	case 2:
		if c.Dist < 13 {
			c.SetSpeed(50)

			if !c.WalkAway(t, 14) {
				if haveAt {
					c.Attack(ModeAttack1, at)
				} else {
					c.Attack(ModeAttack1, t)
				}

				b.Scratch[fetPhase], b.Scratch[fetCount] = 0, 0
			}

			return
		}

		if b.Roll(100) > 19 || !c.Circle(t, 4) {
			b.Scratch[fetPhase], b.Scratch[fetCount] = 0, 0
			c.Sleep(10)
		}
	default:
		c.Sleep(10)
	}
}

// shamanStateFlag is Scratch[2] standing in for unit state 0xc, which the
// Fetish Shaman's Skill1 sets and its next think clears (UNVERIFIED).
const shamanStateFlag = 2

// thinkFetishShaman is MONAI_Think_FetishShaman 0x5f8be0 (VERIFIED flow).
// aip1 heal-buddy%, aip2 buddy search "same-room" param, aip3 walk-near
// distance, aip4 circle%, aip5 buddy radius. Skill1 is cast on self when the
// state flag is clear and the skill range check passes (range is unknown
// here, the exe default 1 is used, UNVERIFIED); Skill3 targets a buddy found
// with the AllyFinder.
func thinkFetishShaman(c *Ctx) {
	b, t := c.B, *c.Target
	p := b.Profile

	if p.Skills[slot1].Used() && c.Dist < 1 && b.Scratch[shamanStateFlag] == 0 {
		b.Broadcast(Command{Type: CmdAlert, Target: t.ID, Count: 6})
		b.Scratch[shamanStateFlag] = 1
		c.Cast(slot1, t)

		return
	}

	b.Scratch[shamanStateFlag] = 0

	if f, ok := c.W.(AllyFinder); ok {
		if ally, d, found := f.NearestAlly(b); found && d <= b.AIP(5) && b.Roll(100) < b.AIP(1) &&
			p.Skills[slot3].Used() {
			b.Broadcast(Command{Type: CmdFollow, Target: ally.ID, Count: 1})

			if d > b.AIP(3) {
				c.WalkNearTarget(&ally, 10)

				return
			}

			c.Cast(slot3, ally)

			return
		}
	}

	if b.Roll(100) < b.AIP(4) && c.Circle(t, 4) {
		return
	}

	c.Sleep(10)
}

// Bat Demon phases (Scratch[0]): 0 and 4 take off, 1 climb (attacks with
// S3/S4 modes while ascending), 2 retreat, 3 hunt.
//
// thinkBatDemon is MONAI_Think_BatDemon 0x5f4170 (VERIFIED flow; the second
// think function 0x5f4110 is not ported). aip1 flee hp%, aip2 retreat%, aip3
// attack%, aip4 A2%, aip5 stat boost eighths. The flight-stat boost (unit
// stat 0x4a) and the S3/S4/S2 mode requests use ModeSkill modes; the engine
// treats unknown modes as failed attacks, which sleeps 10.
func thinkBatDemon(c *Ctx) {
	b, t := c.B, *c.Target

	switch b.Scratch[0] {
	case 0, 4:
		c.Attack(ModeSkill3, t)

		b.Scratch[0], b.Scratch[1] = 1, 0
		b.Airborne = true
	case 1:
		if b.Scratch[1] > 1 && !(!c.InRange && c.Dist > 6 && (c.Dist > 13 || b.HPPercent < 51)) {
			b.Airborne = false
			c.Attack(ModeSkill2, t)

			b.Scratch[0] = 3

			return
		}

		c.Attack(ModeSkill4, t)

		b.Scratch[1]++
	case 2:
		if b.HPPercent < b.AIP(1) && c.WalkAway(t, 15) {
			b.Scratch[0] = 4

			return
		}

		if b.Roll(100) <= 32 {
			if c.WalkTo(t, 0) {
				b.Scratch[0] = 3

				return
			}
		}

		if b.Roll(100) < 15 && c.Wander(6) {
			return
		}

		c.Sleep(10)
	default: // 3
		if b.HPPercent < b.AIP(1) && c.WalkAway(t, 15) {
			b.Scratch[0] = 4

			return
		}

		if !c.InRange {
			c.WalkTo(t, 0)

			return
		}

		if b.Scratch[1] > 0 {
			c.Attack(ModeAttack2, t)

			b.Scratch[1] = 0

			return
		}

		if b.Aggressive && b.Roll(100) < b.AIP(2) && c.WalkAway(t, 12) {
			b.Scratch[0] = 2

			return
		}

		if b.Roll(100) < b.AIP(3) {
			if b.Roll(100) < b.AIP(4) {
				c.Attack(ModeAttack2, t)
			} else {
				c.Attack(ModeAttack1, t)
			}

			return
		}

		c.Sleep(10)
	}
}

// thinkDoomKnight is MONAI_Think_DoomKnight 0x5f9c20 (VERIFIED). In range
// chance(aip1) A1 else sleep(aip2); out of range chance(aip3) walk else sleep
// (aip4).
func thinkDoomKnight(c *Ctx) {
	b, t := c.B, *c.Target

	if !c.InRange {
		if b.Chance(b.AIP(3)) {
			c.SetSpeed(0)

			if !c.WalkTo(t, meleeReach) {
				c.Sleep(10)
			}

			return
		}

		c.Sleep(b.AIP(4))

		return
	}

	if b.Chance(b.AIP(1)) {
		c.Attack(ModeAttack1, t)

		return
	}

	c.Sleep(b.AIP(2))
}

// thinkAbyssKnight is MONAI_Think_AbyssKnight 0x5f9cf0 (VERIFIED flow).
// Scratch[0] is the skill cooldown. aip1 low-hp% for the self buff (skill 2,
// only when its state is not up: unit states are not modelled so the buff is
// always considered down, UNVERIFIED), aip2 buff%, aip3 attack%, aip4 stall,
// aip5 skill1 distance, aip6 cooldown, aip7 approach%, aip8 circle distance.
func thinkAbyssKnight(c *Ctx) {
	b, t := c.B, *c.Target
	p := b.Profile

	if p.Skills[slot2].Used() && b.HPPercent < b.AIP(1) && b.Roll(100) < b.AIP(2) {
		c.Cast(slot2, *c.Target)

		return
	}

	if c.InRange {
		if b.Chance(b.AIP(3)) {
			c.Attack(ModeAttack1, t)

			return
		}

		c.Sleep(b.AIP(4))

		return
	}

	if c.Dist < b.AIP(5) && b.Scratch[0] < 1 {
		b.Scratch[0] = b.AIP(6)
	}

	if p.Skills[slot1].Used() && b.Scratch[0] == 0 && !t.IsPlayer {
		b.Scratch[0] = b.AIP(6)
		c.Cast(slot1, t)

		return
	}

	if p.Skills[slot1].Used() && b.Scratch[0] == 0 {
		b.Scratch[0] = b.AIP(6)
		c.Cast(slot1, t)

		return
	}

	if b.Scratch[0] > 0 {
		b.Scratch[0]--
	}

	if b.Roll(100) < b.AIP(7) {
		b.Roll(2) // the exe spends a second roll on the walk speed override
		c.SetSpeed(0)
		c.WalkTo(t, meleeReach)

		return
	}

	if c.Dist < b.AIP(8) {
		c.Circle(t, 3)

		return
	}

	c.Sleep(15)
}

// thinkMegademon is MONAI_Think_Megademon 0x5dfad0 (Venom Lord, Fire Lord
// and kin; VERIFIED flow). Skill1 is a charge/leap that needs a cooldown
// (Scratch[0] = frame it is ready) and has a range; the exe reads the range
// from the skill table (FUN_00645680), unavailable here, so range 1 is used
// (UNVERIFIED) which makes the pre-cast approach branch the common one.
// aip1 skill1 %, aip2 skill1 % in melee, aip3 attack%, aip4 approach%, aip5
// circle%, aip6 skill cooldown.
func thinkMegademon(c *Ctx) {
	b, t := c.B, *c.Target
	p := b.Profile
	now := c.W.Frame()

	skill := p.Skills[slot1].Used()

	if skill && !c.InRange && c.Dist > 1 {
		if b.Scratch[0] < now && b.Roll(100) < b.AIP(1) {
			b.Scratch[0] = b.AIP(6) + now
			c.Cast(slot1, t)

			return
		}
	} else if c.InRange || !skill {
		if c.InRange {
			if skill && b.Scratch[0] < now && b.Roll(100) < b.AIP(2) {
				b.Scratch[0] = b.AIP(6) + now
				c.Cast(slot1, t)

				return
			}

			if b.Roll(100) < b.AIP(3) {
				c.Attack(ModeAttack1, t)

				return
			}

			if b.Roll(100) < b.AIP(5) && c.Circle(t, 3) {
				return
			}

			c.Sleep(5)

			return
		}
	}

	if b.Roll(100) < b.AIP(4) {
		c.WalkTo(t, meleeReach)

		return
	}

	c.Sleep(10)
}

// thinkSuccubus is MONAI_Think_Succubus 0x5e0c70 (VERIFIED flow; the helper
// FUN_006259a0(target, 0x20) is a state test and FUN_00622100 a hit point
// percent; both are read as such, UNVERIFIED). aip1 attack%, aip2 approach%,
// aip3 cast%, aip4 cast distance, aip5/aip6 stall, aip7 skill1 target hp%
// floor, aip8 skill2 self hp% ceiling / skill5 chance. Slots: 1 and 2 hero
// cast, 3 on non-players (or wounded targets), 4 on players, 5 random cast.
func thinkSuccubus(c *Ctx) {
	b, t := c.B, *c.Target
	p := b.Profile

	if c.Dist < b.AIP(4) && b.Chance(b.AIP(3)) {
		switch {
		case p.Skills[slot1].Used() && 100 >= b.AIP(7):
			c.Cast(slot1, t)

			return
		case p.Skills[slot2].Used() && b.HPPercent <= b.AIP(8):
			c.Cast(slot2, t)

			return
		case p.Skills[slot3].Used() && !t.IsPlayer:
			c.Cast(slot3, t)

			return
		case p.Skills[slot4].Used() && t.IsPlayer:
			c.Cast(slot4, t)

			return
		}
	}

	if c.InRange {
		if b.Roll(100) >= b.AIP(1) {
			c.Sleep(b.AIP(5))

			return
		}

		c.Attack(ModeAttack1, t)

		return
	}

	if p.Skills[slot5].Used() && b.AIP(8) > 0 {
		if at, _, ok := c.W.AttackTarget(b); ok && b.Roll(100) < b.AIP(8) {
			c.Cast(slot5, at)

			return
		}
	}

	if b.Roll(100) >= b.AIP(2) {
		c.Sleep(b.AIP(6))

		return
	}

	if !c.WalkTo(t, 0) {
		c.Sleep(10)
	}
}

// thinkMinion is MONAI_Think_Minion 0x5e09c0 (VERIFIED, re-read in the exe: a command is live while count > frame; section
// 5): a type 1 command with an unexpired count (frame) and a living commanded
// unit overrides the target. aip1 attack gate, aip2 A2%, aip3 approach%, aip4
// stall.
func thinkMinion(c *Ctx) {
	b := c.B
	t := *c.Target

	commanded := false

	if cmd := b.PeekCommand(); cmd != nil && cmd.Type == CmdAlert && c.W.Frame() < cmd.Count {
		if u, found := c.commandedUnit(cmd); found {
			t, commanded = u, true
		} else {
			b.PopCommand()
		}
	} else if cmd != nil {
		b.PopCommand()
	}

	// The in-range flag is the tick's for the tick target; for a different
	// commanded unit the exe re-tests reach to it (FUN_00622e40, VERIFIED).
	inRange := c.InRange
	if commanded && (c.Target == nil || t.ID != c.Target.ID) {
		inRange = c.W.InRange(b, t, b.DistanceTo(t.X, t.Y))
	}

	// Out of reach: a commanded minion walks at once (no roll); a free one
	// rolls aip3 to approach, else stalls aip4 (VERIFIED order).
	if !inRange {
		if !commanded && !b.Chance(b.AIP(3)) {
			c.Sleep(b.AIP(4))

			return
		}

		c.WalkTo(t, 0)

		return
	}

	if !commanded && b.Roll(100) >= b.AIP(1) {
		c.Sleep(b.AIP(2))

		return
	}

	if b.Chance(b.AIP(2)) {
		c.Attack(ModeAttack2, t)
	} else {
		c.Attack(ModeAttack1, t)
	}
}
