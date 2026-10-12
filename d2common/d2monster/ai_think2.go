package d2monster

import "strings"

// Second think functions (column +0xc of the AI table at 0x739c08) and the curse gate of
// MONAI_GetAiTableEntry 0x5af1b0, wave 38 (read from Game.exe 1.14b, own words).
//
// MONAI_SetAiState 0x5ae9d0 (state number N, nonzero for curses): the table entry is the alternate
// table row N (0x73a548) only for N outside 10..12, or when MONSTATS_IsRevivableType 0x623650 holds for
// the class (walk mode in monstats2, monstats boss flag 0x40 CLEAR, switchai flag 0x10000 SET);
// otherwise the entry is the monster's own class row. Then:
//   - the unit currently runs the class think and the row has a second think: only the state number is
//     stored and the think becomes the SECOND think (scratch fields and queue are kept);
//   - anything else: scratch fields cleared, command queue freed, pre hook of the row run, state and think
//     stored (for a non-revivable class a curse therefore only resets the AI).
// Every second think ends by calling SetAiState(same state) again, which now takes the second branch
// (the unit runs the second think, not the class think): clean reset to the class think.
// Classes with a second think: BatDemon, Diablo, BaalCrab, FetishShaman, FrogDemon, Hireable, SandMaggot.

// think2Fns maps a lower-case AI name to its second think function.
var think2Fns = map[string]Think{}

func registerThink2(name string, t Think) { think2Fns[strings.ToLower(name)] = t }

func init() {
	registerThink2("BatDemon", thinkBatDemon2)
	registerThink2("Diablo", thinkAnchorKeepingThink2)
	registerThink2("BaalCrab", thinkAnchorKeepingThink2)
	// wave 45: the AI table rows of UberDiablo and UberBaal (Game.exe 0x739c08) name the same second think
	// functions (0x5e7370, 0x5fc150) next to their bare-return first think; UberMephisto and UberIzual have none.
	registerThink2("UberDiablo", thinkAnchorKeepingThink2)
	registerThink2("UberBaal", thinkAnchorKeepingThink2)
	registerThink2("FetishShaman", thinkClearInfernoThink2)
	registerThink2("Hireable", thinkClearInfernoThink2)
	registerThink2("SandMaggot", thinkSandMaggot2)
	registerThink2("FrogDemon", thinkFrogDemon2)
}

// HasThink2 reports whether the AI has a second think function (table column +0xc).
func HasThink2(name string) bool { _, ok := think2Fns[strings.ToLower(name)]; return ok }

// StateAIAllowed is MONSTATS_IsRevivableType 0x623650, the gate through which the curse AI states 10, 11
// and 12 (Dim Vision/Cloak, Terror, Taunt) reach the alternate table. A class with the boss flag, without
// the switchai flag or without a walk mode keeps its own AI (VERIFIED 0x5af1b0 + 0x623650; the NoWalk
// wiring from monstats2 is UNVERIFIED in the game host).
func (p *Profile) StateAIAllowed() bool { return !p.Boss && !p.NoSwitchAI && !p.NoWalk }

// applyCurseAIState is the AI side of MONAI_SetAiState(curse state) for a class that is not allowed the
// alternate table row. It reports true when it handled the call.
func (b *Brain) applyCurseAIState() bool {
	if b.Profile == nil || b.Profile.StateAIAllowed() {
		return false
	}

	if t2, ok := think2Fns[strings.ToLower(b.Def.Name)]; ok && b.baseDef == b.Def {
		// state stored, think swapped, scratch and queue kept
		d := *b.Def
		d.Think = t2
		b.Def = &d

		return true
	}

	b.SetAI(b.Def) // reset path: scratch cleared, queue freed, pre hook re-run

	return true
}

// thinkBatDemon2 is MONAI_Think2_BatDemon 0x5f4110 (VERIFIED). In mode S4 (11) the unit goes back to
// phase 1 and waits one tick; in phase 4 it goes to phase 1 and queues S3 (mode 10); otherwise a full
// reset (SetAiState) and one tick of waiting.
func thinkBatDemon2(c *Ctx) {
	b := c.B

	if b.Mode == ModeSkill4 {
		b.Scratch[0] = 1
		c.Sleep(1)

		return
	}

	if b.Scratch[0] == 4 {
		b.Scratch[0] = 1
		c.Attack(ModeSkill3, selfOrTarget(c))

		return
	}

	c.RestoreAI()
	c.Sleep(1)
}

// thinkClearInfernoThink2 is MONAI_Think2_FetishShaman 0x5f8ab0 and MONAI_Think2_Hireable 0x5e41f0
// (VERIFIED, identical): clear unit state 0xc if it is on, reset the AI, wait one tick.
func thinkClearInfernoThink2(c *Ctx) {
	clearInferno(c)
	c.RestoreAI()
	c.Sleep(1)
}

// thinkAnchorKeepingThink2 is MONAI_Think2_Diablo 0x5e7370 and MONAI_Think2_BaalCrab 0x5fc150 (VERIFIED,
// identical): remember the position of the queue node of type 0 (+0xc, +0x10), clear state 0xc, reset the
// AI, and when both remembered values are nonzero and no type 0 node exists after the reset, push an
// anchor command (type 10) with them; wait one tick.
func thinkAnchorKeepingThink2(c *Ctx) {
	b := c.B

	x, y := 0, 0
	if n := b.FindCommand(0); n != nil {
		x, y = n.X, n.Y
	}

	clearInferno(c)
	c.RestoreAI()

	if b.FindCommand(0) == nil && x != 0 && y != 0 {
		b.PushCommand(Command{Type: CmdAnchor, X: x, Y: y})
	}

	c.Sleep(1)
}

// cmdSandMaggotSpit is the queue node type 0xe the maggot's second think finds or creates
// (MONAI_PushCommandIfAbsent); its +0x18 field (Delay) set to 1 asks for the Skill1 cast.
const cmdSandMaggotSpit = 0xe

// thinkSandMaggot2 is MONAI_Think2_SandMaggot 0x5f07c0 (VERIFIED flow). The type 0xe node exists after the
// call. Node field +0x18 == 1 and Skill1 defined: cast Skill1 at the unit's path target, wait 30 keeping an
// earlier pending wake, clear the field. Otherwise reset the AI keeping AiGeneral +0x1c (Scratch[2]) and
// wait one tick (keep earliest).
func thinkSandMaggot2(c *Ctx) {
	b := c.B

	n := b.FindCommand(cmdSandMaggotSpit)
	if n == nil {
		b.PushCommand(Command{Type: cmdSandMaggotSpit})
		n = b.PeekCommand()
	}

	if n.Delay == 1 && b.Profile.Skills[0].Used() {
		c.Cast(0, selfOrTarget(c))
		sleepKeepEarliest(c, 0x1e)

		n.Delay = 0

		return
	}

	keep := b.Scratch[2]

	c.RestoreAI()

	b.Scratch[2] = keep

	sleepKeepEarliest(c, 1)
}

// thinkFrogDemon2 is MONAI_Think2_FrogDemon 0x5f7300: see thinkFrogDemonPhase2B, the state number is the
// curse state that triggered the swap (10, 11 or 12).
func thinkFrogDemon2(c *Ctx) {
	st := 12
	switch c.B.Forced {
	case ForcedBlind:
		st = 10
	case ForcedFear:
		st = 11
	}

	thinkFrogDemonPhase2B(c, st)
}

// sleepKeepEarliest is MONAI_ScheduleWaitTicksKeepEarliest: the new wake is used unless an earlier one is
// already pending in the future.
func sleepKeepEarliest(c *Ctx, n int) {
	now := c.W.Frame()
	w := now + n

	if cur := c.B.Wake; cur > now && cur < w {
		w = cur
	}

	c.acted = true
	c.B.Wake = w
}

func clearInferno(c *Ctx) {
	if ss, ok := c.W.(StateSetter); ok && c.W.HasState(c.B, inferno) {
		ss.SetUnitState(c.B, inferno, false)
	}
}

func selfOrTarget(c *Ctx) Target {
	if c.Target != nil {
		return *c.Target
	}

	return fbxSelf(c.B)
}
