package d2combat

// Timed damage states of Game.exe 1.14b, verified against the real appliers
// (stun 0x578830, poison 0x578990, burn 0x578b00, chill 0x578ca0, freeze
// 0x578f50) run in the emulator (golden dm_state). The exe keeps ONE list per
// unit and state id; the rules below say how a new application treats the
// existing list.

// Game state ids used by the appliers.
const (
	StateFreeze = 0x01
	StatePoison = 0x02
	StateChill  = 0x0b
	StateStun   = 0x15
	StateBurn   = 0x73
	// StateChillFlag is the state toggled by the 20 percent roll after a chill.
	StateChillFlag = 0x6b
)

// Caps and stat ids.
const (
	StunLengthCap     = 250 // frames
	StunSpecialLength = 13  // frames, for "special" monster classes (0x63fed0) when longer
	DefaultColdEffect = -50 // slow value for targets without a monstats ColdEffect
	StatHPRegen       = 0x4a
	StatVelocity      = 0x43 // chill sets 0x43, 0x44 and 0x45 to the cold effect
	StatAttackRate    = 0x44
	StatOtherAnimRate = 0x45
)

// StateList is one timed statlist: its end frame and the stats it holds.
type StateList struct {
	End   int
	Stats map[int]int
}

// StateTarget holds what the appliers read from the target unit.
type StateTarget struct {
	Monster bool // unit type 1 (type 0 is a player)
	// Record says the monster class has a monstats record in range.
	Record bool
	// ColdEffect is monstats +0x168 for the current difficulty (signed byte).
	ColdEffect int
	// StunFlag is the word at monstats +0x32 (non-zero allows stuns).
	StunFlag bool
	// Boss is helper 0x63fa40, Special is 0x63fed0, DataFlag8 is the monster
	// data flag tested with index 8 (0x59dd60), ChillImmune is 0x45f0b0(class,
	// 0x13), Uninterruptable is state 0x36 on the target.
	Boss, Special, DataFlag8, ChillImmune, Uninterruptable bool
	// ChillDivisor and FreezeDivisor are DifficultyLevels +0x18 and +0x14.
	ChillDivisor, FreezeDivisor int
}

// StateUnit is the set of lists of one unit plus the state bits toggled.
type StateUnit struct {
	Lists map[int]*StateList
	Bits  map[int]bool
}

// NewStateUnit returns an empty StateUnit.
func NewStateUnit() *StateUnit {
	return &StateUnit{Lists: map[int]*StateList{}, Bits: map[int]bool{}}
}

func (u *StateUnit) create(state, end int) *StateList {
	l := &StateList{End: end, Stats: map[int]int{}}
	u.Lists[state] = l
	u.Bits[state] = true

	return l
}

// ApplyStun applies a stun of length frames at frame. The roll source is the
// ATTACKER's generator and is only used for monsters with data flag 8:
// Roll(100) < 90 returns without stunning (so only 10 percent get through).
// Monsters: boss -> none; stun flag 0 -> none; special classes are cut to 13
// frames when longer; everything else is capped at 250. An existing stun list
// has its end OVERWRITTEN (a shorter stun replaces a longer one).
func (u *StateUnit) ApplyStun(frame, length int, t StateTarget, attackerRng Roller) {
	if length <= 0 {
		return
	}

	if t.Monster {
		if t.DataFlag8 && attackerRng.Roll(100) < 90 {
			return
		}

		if t.Boss || !t.StunFlag {
			return
		}

		if t.Special && length >= StunSpecialLength {
			length = StunSpecialLength
		} else if length > StunLengthCap {
			length = StunLengthCap
		}
	} else if length > StunLengthCap {
		length = StunLengthCap
	}

	end := frame + length

	if l := u.Lists[StateStun]; l != nil {
		l.End = end

		return
	}

	u.create(StateStun, end)
}

// ApplyDot applies poison (StatePoison) or burn (StateBurn): value is the
// per-tick damage (stat 0x4a is set to -value), length in frames. Both must be
// positive. One list per kind: an existing list is only replaced (end and
// value) when the new value is >= the old one, otherwise nothing changes.
// Poison and burn are separate lists, so they add up.
func (u *StateUnit) ApplyDot(state, frame, value, length int) {
	if value <= 0 || length <= 0 {
		return
	}

	end := frame + length

	if l := u.Lists[state]; l != nil {
		if -l.Stats[StatHPRegen] <= value {
			l.End = end
			l.Stats[StatHPRegen] = -value
		}

		return
	}

	u.create(state, end).Stats[StatHPRegen] = -value
}

// ApplyChill applies a chill. targetRng is the TARGET's generator: one roll
// is always consumed after a successful application, and Roll(100) < 20 on a
// non-immune monster sets state 0x6b, otherwise clears it. Slow value: the
// monstats ColdEffect for monsters with a record (0 -> nothing happens, no
// roll), -50 for everyone else. For a monster with a negative effect the
// length is divided by the difficulty divisor (if non-zero); the length is at
// least 1. An existing list only has its end extended; its value is kept.
func (u *StateUnit) ApplyChill(frame, length int, t StateTarget, targetRng Roller) {
	if length <= 0 {
		return
	}

	v := DefaultColdEffect
	if t.Monster && t.Record {
		v = t.ColdEffect
		if v == 0 {
			return
		}
	}

	if t.Monster && v < 0 && t.ChillDivisor != 0 {
		length /= t.ChillDivisor
	}

	if length <= 1 {
		length = 1
	}

	end := frame + length

	if l := u.Lists[StateChill]; l == nil {
		nl := u.create(StateChill, end)
		nl.Stats[StatVelocity], nl.Stats[StatAttackRate], nl.Stats[StatOtherAnimRate] = v, v, v
	} else if l.End < end {
		l.End = end
	}

	if targetRng.Roll(100) < 20 && t.Monster && !t.ChillImmune {
		u.Bits[StateChillFlag] = true
	} else {
		u.Bits[StateChillFlag] = false
	}
}

// ApplyFreeze applies a freeze. Players, bosses, data flag 8 and special
// monsters chill instead (monsters with state 0x36 are untouched). A freeze
// needs a negative cold effect (default -50); the length is divided by the
// difficulty freeze divisor for monsters (no minimum of 1), and an existing
// list is extended, never shortened.
func (u *StateUnit) ApplyFreeze(frame, length int, t StateTarget, targetRng Roller) {
	if length <= 0 {
		return
	}

	switch {
	case !t.Monster:
		u.ApplyChill(frame, length, t, targetRng)

		return
	case t.Uninterruptable:
		return
	case t.Boss || t.DataFlag8 || t.Special:
		u.ApplyChill(frame, length, t, targetRng)

		return
	}

	ce := DefaultColdEffect
	if t.Record {
		ce = t.ColdEffect
	}

	if ce >= 0 {
		return
	}

	if t.FreezeDivisor != 0 {
		length /= t.FreezeDivisor
	}

	end := frame + length

	if l := u.Lists[StateFreeze]; l == nil {
		u.create(StateFreeze, end)
	} else if l.End < end {
		l.End = end
	}
}
