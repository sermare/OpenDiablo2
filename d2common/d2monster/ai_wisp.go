package d2monster

// Will-o-the-Wisp, MONAI_Think_WillOWisp 0x5f2970 (VERIFIED from the full
// disassembly: the stack offset tables of the function were read word by word,
// the decompiler output is scrambled by them). The wisp has two layers.
//
// States 0..3 are an ordinary melee/ranged brain (aip1 ranged %, aip2 melee %,
// aip3 approach %). With a target within squared distance 0x400 (the
// UNIT_DistanceSquared helper), the dance cooldown (S2) over, state < 4 and a
// roll(1000) <= difficulty+2, the wisp enters state 4: the "dance".
//
// State 4 searches (UNITFIND, radius 0x20, filter 0x5f2920 = alive monsters of
// base class 0x76, i.e. wisps) around the wisp and, with exactly 4 or exactly
// 5 found, writes a formation state into EVERY found unit through
// MONAI_SetAiParam (S0 = 6 for four wisps, 5 for five; S1 = 0; S2 = 1-based
// slot) and makes each wait until the next multiple of 0x151 frames. The caller
// waits likewise. Any other count: state 2, wait 8, no cooldown. The searching
// wisp itself is expected among the found units (the filter does not exclude
// it; whether the host list contains it is the host's business).
//
// Dance bodies (per wisp, every think):
//   - S2 outside the formation (>5 for state 5, >4 for 6..12): state 2, wait 8.
//   - target point = the TARGET's position + an offset from the slot table.
//     More than 3 away (UNIT_IsWithinTileDistance): 34% wander(4), else walk to
//     the exact point (mode 2); done for this think.
//   - on place, wait for frame % 0x43 == 0; on that beat, while S1 < 3: take the
//     act offset from the second table; (0,0) just counts S1; otherwise queue
//     mode 7 (SC) at target + offset and count S1.
//   - S1 >= 3: state 5 ends (state 2, S2 = frame + 0x708, wait to the next
//     0x151 multiple). State 6 advances S0 by one (phases 6..12, 7 in all),
//     S1 = 0; after the last phase it ends like state 5; EVERY advance waits to
//     the next 0x151 multiple and, for a living player target, attaches a stat
//     list to it: stat 0x50 (magic find, item_magicbonus) = (difficulty+1)*50,
//     expiring after 0x1a5e00 frames (the same 0x5f3259 block as the end).
const (
	wispDanceDist2  = 0x400
	wispFellowClass = 0x76
	wispBeat        = 0x43
	wispSync        = 0x151
	wispCooldown    = 0x708
	wispBlessStat   = 0x50
	wispBlessStep   = 50
	wispBlessFrames = 0x1a5e00
	wispPhaseFirst  = 6
	wispPhaseLast   = 12
)

// wispSlotsFive is the first position table of state 5 (slot 1..5) and
// wispActFive the second (the act points).
var wispSlotsFive = [6]Point{{}, {-5, -5}, {3, -5}, {6, 3}, {3, 6}, {-5, 3}}

var wispActFive = [6]Point{{}, {6, 3}, {3, 6}, {-5, 3}, {-5, -5}, {3, -5}}

// wispSlotsSix[p] is the position table of states 6..12, p = slot + 4*state - 25;
// wispActSix[p] the act table with the same index.
var wispSlotsSix = [28]Point{
	{-10, -3}, {1, -5}, {6, 0}, {4, 10},
	{-9, -2}, {-6, -11}, {-2, -5}, {6, 1},
	{0, 13}, {-12, 0}, {7, 5}, {-3, -11},
	{-15, -3}, {-6, -3}, {-4, -5}, {-6, -13},
	{-12, -2}, {-6, 2}, {-5, 5}, {1, 11},
	{-13, -6}, {-8, -11}, {-7, 0}, {1, 8},
	{-5, -8}, {-8, 1}, {1, -6}, {6, 9},
}

var wispActSix = [28]Point{
	{1, -5}, {6, 0}, {4, 10}, {-10, -3},
	{1, 8}, {-9, -2}, {-5, 4}, {1, 8},
	{-12, 0}, {7, 5}, {-3, -11}, {11, -2},
	{7, 9}, {-15, -3}, {-6, -13}, {9, 7},
	{-5, -7}, {-5, -7}, {9, 0}, {-12, -2},
	{-8, -11}, {-2, -6}, {-2, -6}, {-13, -6},
	{-8, 1}, {1, -6}, {6, 9}, {0, 0},
}

// WispMate is one unit the formation search found: its brain and the way to
// make it wait (MONAI_ScheduleWaitTicks on another unit).
type WispMate struct {
	B    *Brain
	Wait func(frames int)
}

// WispFormer is the host side of the dance trigger: the wisps (class 0x76,
// alive) within 0x20 subtiles of b, b itself included when it matches.
type WispFormer interface {
	WispFormation(b *Brain) []WispMate
}

// WispBlesser applies the stat list the finished dance phases give the player:
// stat id, value, frames until it expires.
type WispBlesser interface {
	WispBless(b *Brain, target Target, stat, value, frames int)
}

// WispDanceTrigger reports the gate that starts a dance (the roll is made by
// the caller after the first three tests, in the exe's order).
func wispDanceReady(c *Ctx) bool {
	b := c.B

	return c.Target != nil && b.Scratch[2] < c.W.Frame() &&
		sqDist(b.X, b.Y, c.Target.X, c.Target.Y) < wispDanceDist2 && b.Scratch[0] < 4 &&
		b.Seed.Roll(1000) <= uint32(int(b.Diff)+2)
}

// wispSyncWait is the wait until the next multiple of 0x151 frames.
func wispSyncWait(frame int) int { return wispSync - frame%wispSync }

func thinkWillOWisp(c *Ctx) {
	b := c.B

	if wispDanceReady(c) {
		b.Scratch[0] = 4
	}

	switch s := b.Scratch[0]; {
	case s == 4:
		wispFormUp(c)

		return
	case s == 5 || s >= 6:
		wispDance(c)

		return
	case s == 1:
		wispChase(c)

		return
	}

	wispFight(c)
}

// wispFormUp is state 4.
func wispFormUp(c *Ctx) {
	b := c.B
	frame := c.W.Frame()

	var mates []WispMate
	if f, ok := c.W.(WispFormer); ok {
		mates = f.WispFormation(b)
	}

	state := 0

	switch len(mates) {
	case 4:
		state = 6
	case 5:
		state = 5
	default:
		b.Scratch[0] = 2

		c.Sleep(8)

		return
	}

	for i, m := range mates {
		m.B.Scratch = [3]int{state, 0, i + 1}

		if m.Wait != nil {
			m.Wait(wispSyncWait(frame))
		}
	}

	c.Sleep(wispSyncWait(frame))
}

// wispDance is states 5 and 6..12.
func wispDance(c *Ctx) {
	b, t := c.B, c.Target
	frame := c.W.Frame()
	state, step, slot := b.Scratch[0], b.Scratch[1], b.Scratch[2]

	var pos, act Point

	if state == 5 {
		if slot > 5 || slot < 0 {
			b.Scratch[0] = 2
			c.Sleep(8)

			return
		}

		pos, act = wispSlotsFive[slot], wispActFive[slot]
	} else {
		p := slot + 4*state - 25
		if slot > 4 || p < 0 || p >= len(wispSlotsSix) {
			b.Scratch[0] = 2
			c.Sleep(8)

			return
		}

		pos, act = wispSlotsSix[p], wispActSix[p]
	}

	if t == nil {
		c.Sleep(25)

		return
	}

	spot := Point{t.X + pos.X, t.Y + pos.Y}

	if unitTileDist(b, spot.X, spot.Y) > 3 {
		if b.Roll(100) < 0x22 {
			c.Wander(4)

			return
		}

		c.move(spot, nil, 0, false)

		return
	}

	if r := frame % wispBeat; r != 0 {
		c.Sleep(wispBeat - r)

		return
	}

	if step < 3 {
		b.Scratch[1]++

		if act == (Point{}) {
			c.Sleep(wispBeat - frame%wispBeat)

			return
		}

		c.Attack(ModeSpecialCast, Target{X: t.X + act.X, Y: t.Y + act.Y})

		return
	}

	// the phase is finished
	b.Scratch[1] = 0

	if state == 5 {
		b.Scratch[0] = 2
		b.Scratch[2] = frame + wispCooldown
		c.Sleep(wispSyncWait(frame))

		return
	}

	b.Scratch[0]++
	if b.Scratch[0] > wispPhaseLast {
		b.Scratch[0] = 2
		b.Scratch[2] = frame + wispCooldown
	}

	c.Sleep(wispSyncWait(frame))

	if t.IsPlayer {
		if bl, ok := c.W.(WispBlesser); ok {
			bl.WispBless(b, *t, wispBlessStat, (int(b.Diff)+1)*wispBlessStep, wispBlessFrames)
		}
	}
}

// wispFight is states 0, 2 and 3 (VERIFIED modes: out of reach the cast mode 7
// at the target on state 2 or an aip1 roll; in reach mode 4 on state 3 or an
// aip2 roll; then the approach roll aip3 with S1 = 3).
func wispFight(c *Ctx) {
	b := c.B

	if c.Target == nil {
		c.Sleep(25)

		return
	}

	tgt := *c.Target

	if !c.InRange {
		if b.Scratch[0] == 2 || b.Roll(100) < b.AIP(1) {
			c.Attack(ModeSpecialCast, tgt)
			b.Scratch[0] = 0

			return
		}
	} else if b.Scratch[0] == 3 || b.Roll(100) < b.AIP(2) {
		c.Attack(ModeAttack1, tgt)
		b.Scratch[0] = 0

		return
	}

	r := b.Roll(100)
	b.Scratch[0] = 1
	b.Scratch[1] = 3

	if r >= b.AIP(3) {
		c.Wander(4)

		return
	}

	c.WalkTo(tgt, 0)
}

// wispChase is state 1: while the counter S1 is spent, in reach the S1 mode
// (8, no target) and state 3, else an aip1 roll gives the same mode and state
// 2; otherwise count down and walk toward the target (aip3) or wander(6).
func wispChase(c *Ctx) {
	b := c.B

	if c.Target == nil {
		c.Sleep(25)

		return
	}

	tgt := *c.Target

	if b.Scratch[1] < 1 {
		if c.InRange {
			c.Attack(ModeSkill1, tgt)
			b.Scratch[0] = 3

			return
		}

		if b.Roll(100) < b.AIP(1) {
			c.Attack(ModeSkill1, tgt)
			b.Scratch[0] = 2

			return
		}
	}

	b.Scratch[1]--
	r := b.Roll(100)
	b.Scratch[0] = 1

	if r < b.AIP(3) {
		c.WalkTo(tgt, 0)

		return
	}

	c.Wander(6)
}
