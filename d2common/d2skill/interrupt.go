package d2skill

// Player mode interruption (PLRMODE_CanInterruptCurrentMode, 0x57ceb0; notes in gaps-slice-G.md).

// Player modes (original numbering, equal to d2enum.PlayerAnimationMode).
const (
	modeDeath   = 0
	modeNeutral = 1
	modeDead    = 0x11
)

// Bits4Hold is dwBits4 bit 31 of a skills row (meaning unverified): the action holds against interruption.
const Bits4Hold = 0x80000000

// srvdo functions that repeat freely when the same skill is cast again.
const (
	doRepeatA = 0x43
	doRepeatB = 0x4c
)

// InterruptInput is everything the decision reads.
type InterruptInput struct {
	StateBlocked  bool // state 0x36 on the unit
	CurrentMode   int
	NewMode       int  // requested mode; 0 means "no new mode"
	Flag          bool // the exe's fourth argument; false allows
	HasUsedSkill  bool
	SameSkill     bool // the new skill is the one in use
	SrvDoFunc     int  // srvdofunc of the skill in use
	SkillBits4    uint32
	State2a       bool // state 0x2a on
	State2aChance int  // its stat 0xa4, percent chance to hold
	Roll          int  // a 0..99 roll, used only with State2a
	State0f       bool
	ActionDue     bool // PLRMODE_IsActionEventDueSoon
}

// CanInterrupt reports whether the current mode may be broken for the new one. neutralRestart is true when
// the exe also forces a restart of neutral mode (mode 1 reached through the paths that need it).
func CanInterrupt(in InterruptInput) (ok, neutralRestart bool) {
	if in.StateBlocked || in.CurrentMode == modeDeath || in.CurrentMode == modeDead {
		return false, false
	}

	if in.NewMode == 0 || !in.Flag || !in.HasUsedSkill {
		return true, false
	}

	if in.SameSkill && (in.SrvDoFunc == doRepeatA || in.SrvDoFunc == doRepeatB) {
		return true, false
	}

	if in.SkillBits4&Bits4Hold == 0 {
		if in.CurrentMode == modeNeutral {
			return true, true
		}

		if !InterruptibleMode(in.NewMode) {
			return false, false
		}

		return in.ActionDue, false
	}

	hold := (in.State2a && in.State2aChance > in.Roll) || in.State0f
	if !hold {
		return true, false
	}

	if in.CurrentMode != modeNeutral {
		return false, false
	}

	return true, true
}

// InterruptibleMode lists the requested modes that may break a running action: attack 1 and 2 (7, 8), cast
// (10), throw (0xb), skill 1 (0xd) and sequence (0x12).
func InterruptibleMode(m int) bool {
	switch m {
	case 7, 8, 10, 0xb, 0xd, 0x12:
		return true
	}

	return false
}

// ActionDueSlack is the frames past the action frame during which an interruption is still allowed.
const ActionDueSlack = 5

// ActionDue is PLRMODE_IsActionEventDueSoon (0x57cd60): the unit's frame counter is at most the soonest
// action event frame plus 5.
func ActionDue(curFrame, soonestActionFrame int) bool {
	return curFrame <= soonestActionFrame+ActionDueSlack
}
