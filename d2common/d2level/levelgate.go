package d2level

import (
	"errors"
	"fmt"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
)

// levelgate.go: the Levels.txt QuestFlag / QuestFlagEx gate on entering a level.
//
// What the exe does (VERIFIED, verify-level-graph.md section 1): the column is a quest record slot read by
// SERVER_UsePortalObject (0x582700) with QUESTREC_GetFlag bit 0; a portal object into a level whose slot is not
// done is refused. The warp-tile handler SERVER_EnterWarpTile (0x553140) does NOT read the column; it has its
// own destination list (73, 100, 118, 128, 132), of which only the 120 -> 128 check (Rite of Passage node
// 0x58ae70) concerns a flagged level that a plain warp reaches. The engine applies the table like this:
//   - Portal (town portal and portal objects): the table, VERIFIED.
//   - Waypoint: the table, UNVERIFIED (the exe was not seen to read it there; a hero cannot have the waypoint
//     of a flagged level without having walked in, so the rule only guards forged states).
//   - Warp: only the Worldstone Keep levels 128..132 (the exe's warp list names 128 from 120 and 132; both
//     wait on the Rite of Passage node, UNVERIFIED for 132 whose exe check reads another byte, +0x86 of the
//     Eve of Destruction node, not decoded). The other flagged levels (the Durance, Duriel's lair, the
//     palace) have their own warp rules (CheckActThreeWarp, gates.go).
//
// Moving between two levels with the same slot is free (inside the Worldstone Keep, the palace, the Durance).
// The original tests bit 0 of the slot; a quest whose reward is pending counts as done too, because the
// engine sets bit 1 first (Rite of Passage ends with the Ancients' kill and the reward is the talk with
// Qual-Kehk; UNVERIFIED which of the two the exe's node byte follows).

// ErrLevelGated is wrapped by every refusal of CheckLevelGate (errors.Is).
var ErrLevelGated = errors.New("level: locked by a quest")

// GateVia is how the level is entered.
type GateVia string

// The ways into a level.
const (
	GateWarp     GateVia = "warp"
	GateWaypoint GateVia = "waypoint"
	GatePortal   GateVia = "portal"
)

// LevelGateError says which level was refused and which record slot it waits for.
type LevelGateError struct {
	From, To, Slot int
	Via            GateVia
}

func (e *LevelGateError) Error() string {
	return fmt.Sprintf("level %d is locked until quest slot %d is done (%s from level %d)", e.To, e.Slot, e.Via, e.From)
}

// Is makes errors.Is(err, ErrLevelGated) true.
func (e *LevelGateError) Is(target error) bool { return target == ErrLevelGated }

// warpGated reports the flagged levels a warp tile refuses on the table's word.
func warpGated(to int) bool { return to >= 128 && to <= 132 }

// QuestSlotDone reports a quest slot as done: bit 0, or the reward-pending bit 1 that precedes it.
func QuestSlotDone(q *d2s.QuestRecord, slot int) bool {
	return q != nil && (q.Get(slot, d2s.QuestBitDone) || q.Get(slot, d2s.QuestBitRewardPending))
}

// CheckLevelGate returns a *LevelGateError (errors.Is ErrLevelGated) when the move from -> to is refused by the
// QuestFlag / QuestFlagEx column of the destination. A nil record counts as nothing done.
func CheckLevelGate(from, to int, via GateVia, expansion bool, q *d2s.QuestRecord) error {
	slot := LevelQuestFlag(to, expansion)
	if slot == 0 {
		return nil
	}

	if via == GateWarp && !warpGated(to) {
		return nil
	}

	if LevelQuestFlag(from, expansion) == slot {
		return nil // inside the gated area
	}

	if QuestSlotDone(q, slot) {
		return nil
	}

	return &LevelGateError{From: from, To: to, Slot: slot, Via: via}
}

// GatedLevels lists the levels with a QuestFlag for the mode, ascending, as {level, slot}.
func GatedLevels(expansion bool) [][2]int {
	var out [][2]int

	for l := 1; l <= 136; l++ {
		if s := LevelQuestFlag(l, expansion); s != 0 {
			out = append(out, [2]int{l, s})
		}
	}

	return out
}
