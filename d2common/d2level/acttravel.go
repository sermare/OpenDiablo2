package d2level

import (
	"errors"
	"fmt"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
)

// Travel between acts (SERVER_TravelFromNpcMenu 0x5496d0 ends in
// SERVER_ChangePlayerLevel with the start level of the destination act, so
// the LoadAct machinery of PlanTransition does the rest).
//
// What the notes establish (VERIFIED unless marked):
//   - Warriv(2) (class 175, Lut Gholein) has the row "go west" and Meshif(2)
//     (class 264, Kurast Docks) "sail west" in the static menu table
//     (ui-npc.md, table 0x725d60). They are always offered.
//   - Warriv(1) (155) and Meshif(1) (210) are "Talk only" in that table; the
//     east-bound rows ("go east" 0xd36 / "sail east" 0xd38, string keys
//     WarrivMenu1b / MeshifMenuEast) are added by the client only once the
//     act's last quest is done (quests-2.md: "the client only offers the sail
//     menu after Andariel's quest", "[BIN] checks A2Q6 RG in the client
//     menu"). The exact client condition bits are UNVERIFIED; GoalMet accepts
//     done, reward pending, party-done and goal-reached.
//   - The server side (QUESTS_OnActChangeNpcTravel 0x5446e0) sets the "act
//     finished" slots 7 (Warriv), 15 (Meshif) and 28 (Tyrael) to done.
//   - Act 3 -> 4 is not an NPC menu in the original: Mephisto's red portal in
//     the Durance of Hate (a portal object whose destination is level 103).
//     That it is gated only by the portal existing (Mephisto dead) is
//     UNVERIFIED.
//   - Act 4 -> 5 is Tyrael(2) (class 367, 0x16f) after Terror's End (expansion
//     only); the static table gives him Talk only and there is no travel
//     string for him, so the trip is modelled as the effect of talking to
//     him once the quest is done. UNVERIFIED.
//   - There is no NPC that travels back from Act 4 or Act 5 in the original
//     (no "go west" row exists for those towns' NPCs). UNVERIFIED as a
//     negative; Rule lookups therefore only know the four forward trips and
//     the two back trips, and the debug "free" mode allows any act.

// Via names how an act change is triggered.
type Via int

// Act change triggers.
const (
	ViaNPC Via = iota
	ViaPortal
	ViaTalk // the effect of talking to the NPC
)

// ActRule is one act-to-act trip the original allows.
type ActRule struct {
	From, To int
	Via      Via
	// NPCClass is the monstats class of the NPC (0 for a portal).
	NPCClass int
	// Row is the string.tbl key of the menu row ("" if none).
	Row string
	// QuestSlot is the quest slot that must have its goal met (-1: none).
	QuestSlot int
	// NeedExpansion: only Lord of Destruction characters.
	NeedExpansion bool
	// Verified says whether the rule is VERIFIED in the notes.
	Verified bool
}

// Quest slots gating the trips.
var (
	slotAndariel = mustSlot(1, 6)
	slotDuriel   = mustSlot(2, 6)
	slotMephisto = mustSlot(3, 6)
	slotDiablo   = mustSlot(4, 2)
)

func mustSlot(act, q int) int {
	s, ok := d2s.QuestSlot(act, q)
	if !ok {
		panic("bad quest slot")
	}

	return s
}

// Class ids of the travel NPCs.
const (
	ClassWarriv1 = 155
	ClassWarriv2 = 175
	ClassMeshif1 = 210
	ClassMeshif2 = 264
	ClassTyrael2 = 367
)

// ActRules lists every trip, forward first.
//
//nolint:gochecknoglobals // static rule data
var ActRules = []ActRule{
	{From: 1, To: 2, Via: ViaNPC, NPCClass: ClassWarriv1, Row: "WarrivMenu1b", QuestSlot: slotAndariel},
	{From: 2, To: 3, Via: ViaNPC, NPCClass: ClassMeshif1, Row: "MeshifMenuEast", QuestSlot: slotDuriel},
	{From: 3, To: 4, Via: ViaPortal, QuestSlot: slotMephisto},
	{From: 4, To: 5, Via: ViaTalk, NPCClass: ClassTyrael2, QuestSlot: slotDiablo, NeedExpansion: true},
	{From: 2, To: 1, Via: ViaNPC, NPCClass: ClassWarriv2, Row: "WarrivMenu1c", QuestSlot: -1, Verified: true},
	{From: 3, To: 2, Via: ViaNPC, NPCClass: ClassMeshif2, Row: "MeshifMenuWest", QuestSlot: -1, Verified: true},
}

// Errors of CheckActTravel.
var (
	ErrNoRoute      = errors.New("act travel: no way leads from that act to that act")
	ErrQuestNotDone = errors.New("act travel: the act's last quest is not done")
	ErrNeedLoD      = errors.New("act travel: Act V needs Lord of Destruction")
)

// RuleFor finds the rule for a trip.
func RuleFor(from, to int) (ActRule, bool) {
	for _, r := range ActRules {
		if r.From == from && r.To == to {
			return r, true
		}
	}

	return ActRule{}, false
}

// RuleForNPC finds the rule an NPC class triggers.
func RuleForNPC(class int) (ActRule, bool) {
	for _, r := range ActRules {
		if r.NPCClass == class && class != 0 {
			return r, true
		}
	}

	return ActRule{}, false
}

// GoalMet reports whether a quest slot shows the goal reached: done, reward
// pending, party-done (bit 13) or goal-reached (bit 14). UNVERIFIED mapping,
// see the package comment above.
func GoalMet(q *d2s.QuestRecord, slot int) bool {
	if slot < 0 {
		return true
	}

	if q == nil {
		return false
	}

	for _, b := range []int{d2s.QuestBitDone, d2s.QuestBitRewardPending, d2s.QuestBitUpdated, d2s.QuestBitGoalReached} {
		if q.Get(slot, b) {
			return true
		}
	}

	return false
}

// CheckActTravel applies the rule of the trip to the hero's quest record
// (current difficulty). free skips every check (debug mode, and the only way
// to take trips the original has no NPC for).
func CheckActTravel(from, to int, q *d2s.QuestRecord, expansion, free bool) (ActRule, error) {
	if from == to || from < 1 || to < 1 || from > NumActs || to > NumActs {
		return ActRule{}, fmt.Errorf("%w (%d -> %d)", ErrNoRoute, from, to)
	}

	r, ok := RuleFor(from, to)
	if free {
		if !ok {
			r = ActRule{From: from, To: to, QuestSlot: -1}
		}

		return r, nil
	}

	if !ok {
		return ActRule{}, fmt.Errorf("%w (%d -> %d)", ErrNoRoute, from, to)
	}

	if r.NeedExpansion && !expansion {
		return r, ErrNeedLoD
	}

	if !GoalMet(q, r.QuestSlot) {
		return r, fmt.Errorf("%w (slot %d)", ErrQuestNotDone, r.QuestSlot)
	}

	return r, nil
}

// actFinishedSlot is the "act finished" slot set when the trip starts
// (QUESTS_OnActChangeNpcTravel): Warriv 7, Meshif 15, Tyrael 28.
var actFinishedSlot = map[int]int{1: d2s.QuestSlotAct1Finished, 2: d2s.QuestSlotAct2Finished, 4: d2s.QuestSlotAct3Finished}

// MarkActFinished records the trip's server-side quest effect on the record:
// the act-finished slot gets RG (bit 0) and PGD (bit 13). It returns the slot
// or -1.
func MarkActFinished(q *d2s.QuestRecord, from int) int {
	slot, ok := actFinishedSlot[from]
	if !ok || q == nil {
		return -1
	}

	q.Set(slot, d2s.QuestBitDone)
	q.Set(slot, d2s.QuestBitUpdated)

	return slot
}
