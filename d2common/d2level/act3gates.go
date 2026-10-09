package d2level

import (
	"errors"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
)

// Act 3 progression levels (Levels.txt ids).
const (
	LevelTravincal = 83
	LevelDurance1  = 100
	LevelDurance2  = 101 // has a waypoint
	LevelDurance3  = 102 // Mephisto's lair
)

// ErrDuranceSealed is returned for the Travincal stairs while the Compelling
// Orb is intact.
var ErrDuranceSealed = errors.New("warp: the stairs to the Durance of Hate are sealed until the Compelling Orb is smashed")

// slotKhalim is the quest slot of Khalim's Will (act 3, quest 2 = quest id 16).
var slotKhalim = mustSlot(3, 2)

// OrbSmashed reports whether the Compelling Orb is broken. Smashing it with
// Khalim's Will sets the primary-goal and reward-pending bits of the quest
// (quest engine, quests-2.md); Cain's last line turns them into "done".
// A nil record reports false.
func OrbSmashed(q *d2s.QuestRecord) bool {
	return q != nil && (q.Get(slotKhalim, d2s.QuestBitDone) || q.Get(slotKhalim, d2s.QuestBitRewardPending))
}

// CheckActThreeWarp applies the Act 3 progression rules to a stair or warp
// from level `from` to level `to`. Only the Travincal -> Durance 1 stairs are
// gated; every other link returns nil. VERIFIED against Game.exe: the warp
// handler SERVER_EnterWarpTile (0x553140) calls 0x5b9b60, which refuses the warp
// into level 100 while the Blackened Temple node (quest id 19, slot 21) has its
// private byte +0xc clear and the player is not coming from level 101. That byte
// is restored on join from slot 18 (Khalim's Will) bit 0 = done only, so a record
// with just "reward pending" (OrbSmashed's second test) is a live-session state.
func CheckActThreeWarp(from, to int, q *d2s.QuestRecord) error {
	if from == LevelTravincal && to == LevelDurance1 && !OrbSmashed(q) {
		return ErrDuranceSealed
	}

	return nil
}

// MephistoPortalOpen reports whether the red portal out of the lair may be
// used: Mephisto is dead (Guardian, act 3 quest 6: done or reward pending).
func MephistoPortalOpen(q *d2s.QuestRecord) bool {
	return q != nil && (q.Completed(3, 6) || q.RewardPending(3, 6))
}
