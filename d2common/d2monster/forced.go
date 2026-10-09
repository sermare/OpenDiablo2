package d2monster

import "strings"

// Forced AI states: the alternate AI table at 0x73a548 (18 entries) plus the
// target-bucket mechanics that Confuse, Attract and Conversion use.
//
// What the binary shows (read-only Ghidra, Game.exe 1.14b):
//
//   - MONAI_SetAiState(state != 0) swaps the monster's think function for the
//     entry state of the alternate table; MONAI_SetAiState(0) puts the class's
//     own AI back (VERIFIED: FUN_005c1140, FUN_0056e430, every state think's
//     exit path calls it). Both clear the AiGeneral scratch fields and the
//     command queue (VERIFIED, notes), so a forced state always starts clean
//     and the restored AI does not remember its old scratch values.
//   - Curses pick the state from the id of the unit state they apply
//     (FUN_005c1270, VERIFIED): STATE_DIMVISION (23) -> AI state 10,
//     STATE_TERROR (56) -> AI state 11, STATE_TAUNT (27) -> AI state 12, any
//     other curse state -> AI state 0. Cloak of Shadows (SRVDO_047) sets state
//     10 directly. Overseer whip sets 15, Imp teleport 16 (not ported).
//   - Confuse (SRVDO_061 + FUN_005c1c00) and Attract (SRVDO_059) do NOT call
//     SetAiState: they set the unit's alignment stat (stat 0xac) to 1, insert
//     the monster into target list 9 of the game ("Targets.cpp": lists 0..7
//     are the players' parties, 8 and 9 are extra lists that
//     MONAI_FindNearestPlayerTarget scans after the players' lists), and
//     schedule unit event 10 at frame+duration to undo it. Confuse also sets
//     pUnitData+0x34/+0x38 to "mode 3", which makes the monster's own target
//     search (FUN_005dc370) take the nearest unit of ANY kind, players and
//     monsters alike.
//   - Conversion (SRVDO_079) adds STATE_CONVERSION (53) with the duration,
//     inserts the monster into its caster's party list (FUN_005af4e0 with the
//     caster's slot) and sets alignment 1, so hostile monsters target it and
//     it targets hostile monsters; the state's removal callback (FUN_005ce710)
//     restores the monster's level and life.
//
// Everything not read from those functions is marked UNVERIFIED below.

// ForcedKind is a forced condition a skill (or the debug command) puts on a
// monster.
type ForcedKind int

// Forced kinds.
const (
	ForcedNone    ForcedKind = iota
	ForcedFear               // Terror / Howl / Battle Cry flee: AI state 11 (VERIFIED mapping)
	ForcedBlind              // Dim Vision / Cloak of Shadows: AI state 10 (VERIFIED mapping)
	ForcedTaunt              // Taunt: AI state 12 (VERIFIED mapping)
	ForcedConfuse            // Confuse: list 9 + "mode 3" targeting (VERIFIED)
	ForcedAttract            // Attract: list 9 + alignment 1 (VERIFIED)
	ForcedCharm              // Conversion: owner's list + alignment 1 (VERIFIED)
)

var forcedNames = map[string]ForcedKind{
	"fear": ForcedFear, "terror": ForcedFear, "flee": ForcedFear, "howl": ForcedFear,
	"blind": ForcedBlind, "dimvision": ForcedBlind, "cloak": ForcedBlind,
	"taunt":   ForcedTaunt,
	"confuse": ForcedConfuse,
	"attract": ForcedAttract,
	"charm":   ForcedCharm, "convert": ForcedCharm, "conversion": ForcedCharm,
}

// ParseForced converts a debug command word ("fear", "confuse", "attract",
// "charm", "blind", "taunt") to a ForcedKind.
func ParseForced(s string) (ForcedKind, bool) {
	k, ok := forcedNames[strings.ToLower(strings.TrimSpace(s))]

	return k, ok
}

// String is the canonical word of a kind.
func (k ForcedKind) String() string {
	switch k {
	case ForcedFear:
		return "fear"
	case ForcedBlind:
		return "blind"
	case ForcedTaunt:
		return "taunt"
	case ForcedConfuse:
		return "confuse"
	case ForcedAttract:
		return "attract"
	case ForcedCharm:
		return "charm"
	}

	return "none"
}

// AIState is the index of the alternate AI table the kind installs, or 0 when
// the kind does not change the think function.
func (k ForcedKind) AIState() int {
	switch k {
	case ForcedFear:
		return StateFear
	case ForcedBlind:
		return StateBlind
	case ForcedTaunt:
		return StateTaunted
	}

	return 0
}

// UnitStateID is the states.txt id a skill applies for the kind (VERIFIED
// against states.txt: 56 terror, 23 dimvision, 27 taunt, 59 confuse, 57
// attract, 53 conversion). The World's HasState answers true for it while the
// kind is active.
func (k ForcedKind) UnitStateID() int {
	switch k {
	case ForcedFear:
		return StateTerror
	case ForcedBlind:
		return StateDimVision
	case ForcedTaunt:
		return StateTaunt
	case ForcedConfuse:
		return StateConfuse
	case ForcedAttract:
		return StateAttract
	case ForcedCharm:
		return StateConversion
	}

	return 0
}

// Indices of the alternate AI table (0x73a548). Only the entries marked
// "ported" have a think function here.
const (
	StateIdle      = 1  // Idle (5ae8a0)
	StateWander    = 2  // MONAI_State2_Think, ported
	StateFormation = 3  // Pre 5e46c0 / Think 5e4810, ported (group march)
	StateHireable  = 4  // Hireable (ported in ai_hireable.go)
	StateGoodRange = 5  // GoodNpcRanged, not ported
	StateTownGuard = 6  // MONAI_State6_Think, ported
	StateNecroPet  = 7  // NecroPet, not ported
	StateAttack25  = 8  // MONAI_State8_Think, ported
	StateRelease   = 9  // MONAI_State9_Think, ported (group release)
	StateBlind     = 10 // MONAI_State10_Think, ported (also 17, target mode 2)
	StateFear      = 11 // Pre 5e6fe0 / Think 5e7040, ported
	StateTaunted   = 12 // MONAI_State12_Think, ported
	StateLeash     = 13 // MONAI_State13_Think (anchor + bucket), not ported
	StateCharge    = 14 // MONAI_State14_Think, ported
	StateSuicide   = 15 // SuicideMinion, ported as an archetype
	StateImp       = 16 // Pre 5e1bc0 / Think 5e1c60 (imp after teleport), not ported
	StateBlind2    = 17 // = State10_Think with target mode 2
)

// Unit state ids (states.txt) the forced kinds correspond to.
const (
	StateStunned    = 21
	StateDimVision  = 23
	StateTaunt      = 27
	StateConversion = 53
	StateTerror     = 56
	StateAttract    = 57
	StateConfuse    = 59
)

// CmdRelease is the group command (type 8) that states 3 and 9 broadcast to
// release the followers (VERIFIED value; the receivers pop it and return to
// their own AI).
const CmdRelease = 8

var stateTable [18]*AIDef

func regState(id int, mode int, t Think) {
	stateTable[id] = &AIDef{Name: "State" + itoa(id), TargetMode: mode, Think: t, Implemented: true}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}

	var b [8]byte

	i := len(b)
	for ; n > 0; n /= 10 {
		i--
		b[i] = byte('0' + n%10)
	}

	return string(b[i:])
}

func init() {
	regState(StateWander, TargetStandard, thinkState2)
	regState(StateFormation, TargetStandard, thinkState3)
	regState(StateTownGuard, TargetNone, thinkState6)
	regState(StateAttack25, TargetStandard, thinkState8)
	regState(StateRelease, TargetStandard, thinkState9)
	regState(StateBlind, TargetStandard, thinkState10)
	regState(StateFear, TargetStandard, thinkState11)
	regState(StateTaunted, TargetStandard, thinkState12)
	regState(StateCharge, TargetStandard, thinkState14)
	regState(StateBlind2, TargetOnly, thinkState10)
}

// StateDef returns the alternate-table entry of an AI state, or nil when the
// state is not ported.
func StateDef(id int) *AIDef {
	if id < 0 || id >= len(stateTable) {
		return nil
	}

	return stateTable[id]
}

// Optional extensions of World the forced states use. Each has a default when
// the World does not implement it.

// TownChecker reports whether the monster is in a town level (FUN_0061a850).
type TownChecker interface{ InTown(b *Brain) bool }

// ModeChecker reports whether the monster's class has an animation mode
// (monstats2 mode bits, FUN_00467af0). Default: true except RN, which needs a
// run velocity.
type ModeChecker interface{ HasMode(b *Brain, m Mode) bool }

// SourceFinder resolves a unit id to a target (the unit that cursed the
// monster).
type SourceFinder interface {
	UnitTarget(b *Brain, id uint32) (t Target, dist int, ok bool)
}

// ConfusedSenses answers a confused monster's target search: the nearest unit
// of any kind, monsters included (FUN_005dc370 case 3).
type ConfusedSenses interface {
	NearestAny(b *Brain) (t Target, dist int, ok bool)
}

// DecoyFinder finds the nearest monster under Attract (target list 9).
type DecoyFinder interface {
	NearestDecoy(b *Brain) (t Target, dist int, ok bool)
}

func hasMode(c *Ctx, m Mode) bool {
	if mc, ok := c.W.(ModeChecker); ok {
		return mc.HasMode(c.B, m)
	}

	if m == ModeRun {
		return c.B.Profile.Run > 0
	}

	return true
}

func inTown(c *Ctx) bool {
	tc, ok := c.W.(TownChecker)

	return ok && tc.InTown(c.B)
}

// ApplyForced puts a forced condition on the monster for frames game frames.
// source is the unit that caused it (the curser; the owner for a conversion).
// A condition already active is ended first, as a second curse replaces the
// first one.
//
// The think function is swapped for states 10/11/12 (and the scratch fields
// and command queue cleared, as MONAI_SetAiState does); Confuse, Attract and
// Charm set flags the target search reads.
func (b *Brain) ApplyForced(kind ForcedKind, now, frames int, source uint32) {
	if kind == ForcedNone {
		return
	}

	if b.Forced != ForcedNone {
		b.endForced()
	}

	if frames < 1 {
		frames = 1
	}

	b.Forced, b.ForcedUntil, b.ForcedSource = kind, now+frames, source
	b.baseDef = b.Def

	switch kind {
	case ForcedAttract:
		b.Attracting = true
	case ForcedCharm:
		b.Allied, b.OwnerID = true, source
	case ForcedConfuse:
		// mode 3 targeting is read from b.Forced by pickTarget
	default:
		if def := StateDef(kind.AIState()); def != nil {
			b.SetAI(def)
		}
	}

	b.Wake = now // MONAI_SetAiState is followed by a new AI event at once
}

// ClearForced ends the forced condition now (the unit event 10 / state removal
// in the exe) and restores the class AI.
func (b *Brain) ClearForced(now int) {
	if b.Forced == ForcedNone {
		return
	}

	b.endForced()
	b.Wake = now
}

func (b *Brain) expireForced(now int) {
	if b.Forced != ForcedNone && now >= b.ForcedUntil {
		b.ClearForced(now)
	}
}

// endForced is MONAI_SetAiState(0): the class AI comes back with clean
// scratch fields and an empty command queue, flags are cleared.
func (b *Brain) endForced() {
	def := b.baseDef
	if def == nil {
		def = b.classDef()
	}

	b.SetAI(def)
	b.baseDef = nil
	b.Forced, b.ForcedUntil, b.ForcedSource = ForcedNone, 0, 0
	b.Attracting = false
	b.Allied, b.OwnerID = false, 0
}

// classDef is the AI definition of the monster's own class.
func (b *Brain) classDef() *AIDef {
	if d, ok := Lookup(b.Profile.AI); ok {
		return d
	}

	return unimplementedDef(b.Profile.AI)
}

// RestoreAI is MONAI_SetAiState(unit, 0) as called by the state think
// functions when their condition is over.
func (c *Ctx) RestoreAI() { c.B.endForced() }

// StateLabel describes what drives the monster right now, for traces: the AI
// name plus the forced condition.
func (b *Brain) StateLabel() string {
	s := b.Def.Name
	if b.Forced != ForcedNone {
		s += "+" + b.Forced.String()
	}

	return s
}

// pickTarget is the target search of the tick: the nearest hero (or whatever
// the World's rules give), changed by the forced conditions.
func pickTarget(c *Ctx) (Target, int, bool) {
	b := c.B

	// a feared or taunted monster reacts to the unit that cursed it
	if b.ForcedSource != 0 && (b.Forced == ForcedFear || b.Forced == ForcedTaunt) {
		if sf, ok := c.W.(SourceFinder); ok {
			if t, d, found := sf.UnitTarget(b, b.ForcedSource); found {
				return t, d, true
			}
		}
	}

	// Confuse: mode 3 of FUN_005dc370, the nearest unit of any kind
	if b.Forced == ForcedConfuse {
		if cs, ok := c.W.(ConfusedSenses); ok {
			return cs.NearestAny(b)
		}
	}

	t, d, ok := c.W.Nearest(b)

	// Attract: list 9 is scanned after the players' lists and FUN_005dc270
	// decides whether its best candidate replaces the player target. That
	// function's exact rule (path based) is UNVERIFIED; here the decoy wins when
	// it is inside the aggro radius and not farther than the hero.
	if df, isD := c.W.(DecoyFinder); isD {
		if dt, dd, dok := df.NearestDecoy(b); dok && dd <= b.Profile.Aggro() && (!ok || dd <= d) {
			return dt, dd, true
		}
	}

	return t, d, ok
}

// ---- think functions of the alternate table ----

// thinkState2 is MONAI_State2_Think 0x5af0d0 (VERIFIED): while the monster is
// not "aggressive" and the target is more than 3 away it drifts toward it
// (one 12-subtile approach per state entry when more than 10 away) and
// otherwise wanders; with a close target (or when aggressive) it returns to
// its own AI.
func thinkState2(c *Ctx) {
	b := c.B

	if !b.Aggressive && c.Dist > 3 && c.Target != nil {
		if c.Dist > 10 && b.Scratch[0] == 0 {
			b.Scratch[0] = 1
			c.WalkNearTarget(c.Target, 12)

			return
		}

		if b.Seed.Step()&0xff < 0x4d { // 77/256
			c.Wander(4)

			return
		}

		if b.Seed.Step()&0xff < 0x1a { // 26/256
			b.Scratch[0] = 0
		}
	} else {
		c.RestoreAI()
	}

	c.Sleep(10)
}

// thinkState3 is the group march (Pre 5e46c0 + Think 5e4810, VERIFIED shape):
// followers walk to the leader's last order plus the offset they had when the
// state started; the leader picks a point around itself every few ticks and
// broadcasts it (command type 2). A type-8 command or an aggressive monster
// ends the march. Offsets: Scratch[0], Scratch[1]; the leader's quadrant
// counter: Scratch[2]. The roll that sizes the leader's step (a hidden
// argument of RAND_RollSeedBounded) is taken as 5, UNVERIFIED.
func thinkState3(c *Ctx) {
	b := c.B
	lead := b.Leader
	isLeader := lead == nil && len(b.Minions) > 0

	if !b.formationInit {
		b.formationInit = true

		if lead != nil {
			b.Scratch[0], b.Scratch[1] = b.X-lead.X, b.Y-lead.Y
		}

		if isLeader {
			b.Scratch[2] = b.Roll(100)
		}
	}

	cmd := b.PeekCommand()
	if cmd != nil && cmd.Type == CmdRelease {
		if lead != nil {
			lead.RemoveMinion(b)
		}

		c.RestoreAI()
		c.Sleep(10)

		return
	}

	if b.Aggressive {
		b.Broadcast(Command{Type: CmdRelease, Count: 1})

		if lead != nil {
			lead.RemoveMinion(b)
		}

		c.RestoreAI()
		c.Sleep(10)

		return
	}

	if !isLeader {
		if cmd == nil {
			c.Sleep(15)

			return
		}

		x, y := cmd.X+b.Scratch[0], cmd.Y+b.Scratch[1]
		b.PopCommand()
		c.moveTo(Point{x, y})

		return
	}

	if b.Roll(100) < 30 {
		c.Sleep(10)

		return
	}

	n := b.Scratch[2] + 1
	if n > 99 {
		n = 0
	}

	off := b.Roll(5)
	x, y := b.X, b.Y

	sgn := func(v int) int {
		if b.Aux.Chance() {
			return -v
		}

		return v
	}

	switch {
	case n < 25:
		x, y = x+sgn(off), y-5
	case n < 50:
		x, y = x+5, y+sgn(off)
	case n < 75:
		x, y = x+sgn(off), y+5
	default:
		x, y = x-5, y+sgn(off)
	}

	if c.moveTo(Point{x, y}) {
		b.Broadcast(Command{Type: 2, X: x, Y: y})
		b.Scratch[2] = n

		return
	}

	b.Scratch[2] = n + 25
}

// moveTo walks to a point (FUN_005ddb70).
func (c *Ctx) moveTo(p Point) bool { return c.move(p, nil, 0, false) }

// thinkState6 is MONAI_State6_Think 0x5e6b30 (VERIFIED): the stand-in AI of a
// follower without an owner (a mercenary whose owner left). Outside town it
// hunts the nearest hero (30% to walk up to melee, 80% to strike when in
// reach); otherwise it shuffles (20%) or waits.
func thinkState6(c *Ctx) {
	b := c.B

	if b.Mode != ModeNeutral {
		c.Sleep(5)

		return
	}

	if !inTown(c) {
		if t, d, ok := c.W.Nearest(b); ok {
			r := b.Roll(100)

			if !c.W.InRange(b, t, d) {
				if r < 30 && c.WalkTo(t, meleeReach) {
					return
				}
			} else if r < 80 && c.Attack(ModeAttack1, t) {
				return
			}

			c.Sleep(10)

			return
		}
	}

	if b.Seed.Step()%100 < 20 {
		c.Wander(5)

		return
	}

	c.Sleep(10)
}

// thinkState8 is MONAI_State8_Think 0x5e6cb0 (VERIFIED): strike the nearest
// attackable unit within 25, else wait 50.
func thinkState8(c *Ctx) {
	if t, d, ok := c.W.AttackTarget(c.B); ok && d < 25 {
		c.Attack(ModeAttack1, t)

		return
	}

	c.Sleep(50)
}

// thinkState9 is MONAI_State9_Think 0x5e6e70 (VERIFIED): the followers'
// release. While the monster is not aggressive and the target is farther
// than 19 it stays; otherwise it tells its own followers to stand down
// (command 8), leaves its leader and returns to its AI. The queue clean-up
// FUN_0058d160 that runs first is not ported (UNVERIFIED effect).
func thinkState9(c *Ctx) {
	b := c.B

	cmd := b.PeekCommand()
	if cmd == nil || cmd.Type != CmdRelease {
		if !b.Aggressive && c.Dist > 0x13 {
			c.Sleep(10)

			return
		}

		b.Broadcast(Command{Type: CmdRelease, Count: 1})
	}

	if b.Leader != nil {
		b.Leader.RemoveMinion(b)
	}

	c.RestoreAI()
	c.Sleep(10)
}

// thinkState10 is MONAI_State10_Think 0x5e6f20 (VERIFIED), the state that
// Dim Vision and Cloak of Shadows give: the monster no longer pursues. If a
// target is in reach and the class has an attack animation it strikes
// (A1), otherwise it shuffles 3 subtiles (20%) or waits 10. The monstats
// "interact" flag exclusion of the first test is not modelled.
func thinkState10(c *Ctx) {
	b := c.B

	if c.Target != nil && c.InRange && hasMode(c, ModeAttack1) {
		c.Attack(ModeAttack1, *c.Target)

		return
	}

	if b.Seed.Step()%100 > 0x13 {
		c.Sleep(10)

		return
	}

	c.Wander(3)
}

// fleeBoost is the speed override of the fear state: how much faster the run
// velocity is than the walk velocity, in percent over 100, capped at 120
// (VERIFIED arithmetic; what the override does to the speed is UNVERIFIED).
func fleeBoost(p *Profile) int {
	if p.Walk < 1 {
		return 0
	}

	pct := p.Run * 100 / p.Walk
	if pct < 100 {
		return 0
	}

	return clamp(pct-100, 0, 0x78)
}

// fleeDistance is how far each flee step goes (VERIFIED, 30 subtiles).
const fleeDistance = 30

// thinkState11 is MONAI_State11_Think 0x5e7040 (VERIFIED), the Terror / Howl /
// Battle Cry flight:
//   - the terror unit state must still be on, else the monster returns to its
//     AI;
//   - a source farther than the trigger distance (Scratch[0], default 30) is
//     ignored (wait 10);
//   - the first tick starts the flight (run when the class can run, else
//     walk, 30 subtiles directly away from the source);
//   - afterwards a cornered monster (the source is in reach) strikes back, and
//     otherwise keeps fleeing; if no flee path exists it shuffles 6.
//
// FUN_00452b20, a monstats flag test in the cornered condition, is not
// modelled (UNVERIFIED which flag; treated as clear).
func thinkState11(c *Ctx) {
	b, src := c.B, c.Target

	if !c.W.HasState(b, StateTerror) {
		c.RestoreAI()
		c.Sleep(1)

		return
	}

	if src == nil {
		c.Sleep(10)

		return
	}

	th := b.Scratch[0]
	if th == 0 {
		th = 30
	}

	if th < c.Dist {
		c.Sleep(10)

		return
	}

	canRun := hasMode(c, ModeRun)
	boost := fleeBoost(b.Profile)

	if b.Scratch[2] == 0 {
		b.Scratch[2] = 1

		c.SetSpeed(boost)

		if canRun {
			c.RunAway(*src, fleeDistance)
		} else {
			c.WalkAway(*src, fleeDistance)
		}

		return
	}

	if c.InRange && hasMode(c, ModeAttack1) {
		c.Attack(ModeAttack1, *src)

		return
	}

	c.SetSpeed(boost)

	ok := false
	if canRun {
		ok = c.RunAway(*src, fleeDistance)
	} else {
		ok = c.WalkAway(*src, fleeDistance)
	}

	if !ok {
		c.Wander(6)
	}
}

// thinkState12 is MONAI_State12_Think 0x5e7240 (VERIFIED), the Taunt state:
// the monster goes for the taunting unit. Out of town, with the taunter
// alive: the first tick only walks; later ticks strike the taunter when in
// reach, strike an ordinary target if one is in reach (and forget the
// taunter's lock for one tick), else keep walking to the taunter. Without a
// taunter the monster returns to its AI.
func thinkState12(c *Ctx) {
	b := c.B

	var (
		src Target
		ok  bool
	)

	if sf, isSF := c.W.(SourceFinder); isSF {
		src, _, ok = sf.UnitTarget(b, b.ForcedSource)
	}

	if !ok || inTown(c) {
		c.RestoreAI()
		c.Sleep(1)

		return
	}

	if b.Scratch[0] == 0 {
		b.Scratch[0] = 1
	} else {
		if c.W.InRange(b, src, b.DistanceTo(src.X, src.Y)) {
			c.Attack(ModeAttack1, src)

			return
		}

		if c.Target != nil && c.InRange {
			b.Scratch[0] = 0
			c.Attack(ModeAttack1, *c.Target)

			return
		}

		b.Scratch[0] = 1
	}

	if !c.WalkTo(src, meleeReach) {
		c.Sleep(10)
	}
}

// thinkState14 is MONAI_State14_Think 0x5e14b0 (VERIFIED): out of reach 89%
// walk to the target (tile exact), in reach 95% an A2 strike (two LCG steps
// are consumed, VERIFIED), else wait 10.
func thinkState14(c *Ctx) {
	b := c.B

	if c.Target == nil {
		c.Sleep(10)

		return
	}

	if !c.InRange {
		if b.Seed.Step()%100 < 0x59 {
			c.WalkTo(*c.Target, 0)

			return
		}
	} else if b.Seed.Step()%100 < 0x5f {
		b.Seed.Step()
		c.Attack(ModeAttack2, *c.Target)

		return
	}

	c.Sleep(10)
}
