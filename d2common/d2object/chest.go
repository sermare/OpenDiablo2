package d2object

import "errors"

// ChestHooks are the points where the engine plugs in the loot tables, so the
// chest rules stay testable without MPQs.
type ChestHooks struct {
	// TreasureClass names the class for a chest (e.g. ground.ChestTreasureClass).
	TreasureClass func(def Def, act, diff, areaLevel int) string
	// Drop rolls a treasure class and returns how many entries dropped.
	Drop func(tc string, areaLevel int, seed uint32) int
}

// ChestState is the runtime state of one container.
type ChestState struct {
	Opened bool
	Locked bool
	Trap   bool
}

// ErrAlreadyOpened is returned when a used container is operated again.
var ErrAlreadyOpened = errors.New("object: already opened")

// NewChest builds the state of a freshly spawned container. Lockable rows are
// locked (UNVERIFIED: no key item, a click unlocks) and TrapProb is the percent
// chance of a trap (UNVERIFIED reading of the column).
func NewChest(def Def, r *Roller) ChestState {
	return ChestState{
		Locked: def.Lockable,
		Trap:   def.TrapProb > 0 && r.Roll(100) < def.TrapProb,
	}
}

// OpenResult reports what operating a container did.
type OpenResult struct {
	TreasureClass string
	Dropped       int
	Trapped       bool // a trap fired (damage = Def.Damage)
}

// Open operates a container once: the trap (if any) fires, then the hook rolls
// the treasure class. One drop and the object is used up (VERIFIED pattern for
// racks at 0x582050: mode 0 only).
func Open(def Def, st *ChestState, h ChestHooks, act, diff, areaLevel int, seed uint32) (OpenResult, error) {
	if st.Opened {
		return OpenResult{}, ErrAlreadyOpened
	}

	st.Opened, st.Locked = true, false
	res := OpenResult{Trapped: st.Trap}
	st.Trap = false

	if h.TreasureClass != nil {
		res.TreasureClass = h.TreasureClass(def, act, diff, areaLevel)
	}

	if h.Drop != nil && res.TreasureClass != "" {
		res.Dropped = h.Drop(res.TreasureClass, areaLevel, seed)
	}

	return res, nil
}
