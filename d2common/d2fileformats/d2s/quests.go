package d2s

// Quest records. Each difficulty owns 96 bytes: 48 little-endian u16 "slots"
// with 16 flag bits each (verified against Game.exe QUESTREC_GetFlag, which
// addresses bit slot*16+bit LSB-first; see d2-re-notes/quests.md). A slot is
// NOT a quest id; the slot layout below comes from the static quest table.

// Flag bits of a quest slot.
const (
	// QuestBitDone is set once the quest is completed and its reward taken.
	QuestBitDone = 0
	// QuestBitRewardPending is set when the goal is reached but the reward
	// has not been claimed yet.
	QuestBitRewardPending = 1
	// QuestBitProgressFirst..QuestBitProgressLast are quest-specific
	// progress bits, all cleared by the engine when the quest completes.
	QuestBitProgressFirst = 2
	QuestBitProgressLast  = 11
	// QuestBitClosed is the "closed/seen" log bit (semantics unverified).
	QuestBitClosed = 12
	// QuestBitUpdated and QuestBitGoalReached are transient: the engine
	// clears both when it loads a save (QUESTREC_LoadFromBuffer).
	QuestBitUpdated     = 13
	QuestBitGoalReached = 14
	// QuestBitCompleteOnEntry is set by the engine for quests that were
	// already complete when the player entered the game.
	QuestBitCompleteOnEntry = 15
)

// Record layout constants.
const (
	// QuestSlots is the number of u16 slots of one difficulty.
	QuestSlots = questsPerDiff / 2
	// QuestSlotAkaraRespec is the slot whose bits 1 and 13 are set by the
	// Den of Evil reward and gate Akara's "Reset Stat/Skill Points".
	QuestSlotAkaraRespec = 41
	// NumActs is the number of acts.
	NumActs = 5
)

// Slots of the "act finished" words that have no quest of their own; they
// are set by NPC talk side effects (Warriv, Meshif, Tyrael; quests.md,
// 0x5446e0). Slots 23 and 34 are listed by the community layout as travel
// words but were not confirmed in the binary.
const (
	QuestSlotAct1Finished = 7
	QuestSlotAct2Finished = 15
	QuestSlotAct3Finished = 28
)

// actFirstSlot is the first slot of each act: for acts 1-4 it is the act's
// prologue (the introduction by Warriv, Jerhyn, Hratli, Tyrael), the real
// quests follow. Act 5 has no prologue slot; its six quests start at 35.
var actFirstSlot = [NumActs]int{0, 8, 16, 24, 35}

// actQuestCount is the number of real quests of each act.
var actQuestCount = [NumActs]int{6, 6, 6, 3, 6}

// QuestSlot maps (act, quest) to a record slot. act is 1..5; quest is
// 1-based within the act, and 0 selects the act's prologue slot (acts 1-4).
// ok is false for combinations that have no slot.
func QuestSlot(act, quest int) (slot int, ok bool) {
	if act < 1 || act > NumActs || quest < 0 || quest > actQuestCount[act-1] {
		return 0, false
	}

	if act == NumActs {
		if quest == 0 {
			return 0, false
		}

		return actFirstSlot[act-1] + quest - 1, true
	}

	return actFirstSlot[act-1] + quest, true
}

// QuestCount returns the number of real quests of an act (0 if invalid).
func QuestCount(act int) int {
	if act < 1 || act > NumActs {
		return 0
	}

	return actQuestCount[act-1]
}

// QuestRecord is the 96 byte quest block of one difficulty.
type QuestRecord [questsPerDiff]byte

// QuestRecord returns a view of a difficulty's quest block (0 normal,
// 1 nightmare, 2 hell). Writes through the view reach Body.Quests and are
// therefore written by Write. It returns nil for an invalid difficulty.
func (b *Body) QuestRecord(difficulty int) *QuestRecord {
	if difficulty < 0 || difficulty >= numDifficulties {
		return nil
	}

	return (*QuestRecord)(&b.Quests[difficulty])
}

// Slot returns the 16 bits of a slot (0 for an invalid slot).
func (q *QuestRecord) Slot(slot int) uint16 {
	if slot < 0 || slot >= QuestSlots {
		return 0
	}

	return uint16(q[2*slot]) | uint16(q[2*slot+1])<<8
}

// SetSlot replaces the 16 bits of a slot.
func (q *QuestRecord) SetSlot(slot int, v uint16) {
	if slot < 0 || slot >= QuestSlots {
		return
	}

	q[2*slot] = byte(v)
	q[2*slot+1] = byte(v >> 8)
}

// Get tests one flag bit of a slot.
func (q *QuestRecord) Get(slot, bit int) bool {
	if bit < 0 || bit > 15 {
		return false
	}

	return q.Slot(slot)>>uint(bit)&1 != 0
}

// Set sets one flag bit of a slot.
func (q *QuestRecord) Set(slot, bit int) {
	if bit < 0 || bit > 15 {
		return
	}

	q.SetSlot(slot, q.Slot(slot)|1<<uint(bit))
}

// Clear clears one flag bit of a slot.
func (q *QuestRecord) Clear(slot, bit int) {
	if bit < 0 || bit > 15 {
		return
	}

	q.SetSlot(slot, q.Slot(slot)&^(1<<uint(bit)))
}

// ClearProgressBits clears the quest-specific bits 2..11, as the engine does
// when a quest completes (QUESTREC_ClearProgressBits).
func (q *QuestRecord) ClearProgressBits(slot int) {
	const mask = ((1 << (QuestBitProgressLast + 1)) - 1) &^ ((1 << QuestBitProgressFirst) - 1)

	q.SetSlot(slot, q.Slot(slot)&^mask)
}

// Progress returns the quest-specific bits 2..11 shifted down to bit 0.
func (q *QuestRecord) Progress(slot int) uint16 {
	return q.Slot(slot) >> QuestBitProgressFirst & (1<<(QuestBitProgressLast-QuestBitProgressFirst+1) - 1)
}

// Completed reports bit 0 ("done") of the quest at (act, quest); see QuestSlot.
func (q *QuestRecord) Completed(act, quest int) bool {
	return q.bit(act, quest, QuestBitDone)
}

// RewardPending reports bit 1 of the quest at (act, quest).
func (q *QuestRecord) RewardPending(act, quest int) bool {
	return q.bit(act, quest, QuestBitRewardPending)
}

// CompleteOnEntry reports bit 15 of the quest at (act, quest).
func (q *QuestRecord) CompleteOnEntry(act, quest int) bool {
	return q.bit(act, quest, QuestBitCompleteOnEntry)
}

// SetCompleted marks a quest done the way the engine does: progress bits
// and the reward-pending bit are cleared, bit 0 is set.
func (q *QuestRecord) SetCompleted(act, quest int) {
	slot, ok := QuestSlot(act, quest)
	if !ok {
		return
	}

	q.ClearProgressBits(slot)
	q.Clear(slot, QuestBitRewardPending)
	q.Set(slot, QuestBitDone)
}

// ActFinished reports the "act finished" word of acts 1-3 (slots 7, 15, 28).
// Acts 4 and 5 have no verified word and report false.
func (q *QuestRecord) ActFinished(act int) bool {
	switch act {
	case 1:
		return q.Get(QuestSlotAct1Finished, QuestBitDone)
	case 2:
		return q.Get(QuestSlotAct2Finished, QuestBitDone)
	case 3:
		return q.Get(QuestSlotAct3Finished, QuestBitDone)
	}

	return false
}

// AkaraRespecAvailable reports whether the Den of Evil respec reward
// (slot 41, bit 1) is still pending.
func (q *QuestRecord) AkaraRespecAvailable() bool {
	return q.Get(QuestSlotAkaraRespec, QuestBitRewardPending)
}

// CompletedCount returns how many real quests of an act are done.
func (q *QuestRecord) CompletedCount(act int) int {
	n := 0

	for i := 1; i <= QuestCount(act); i++ {
		if q.Completed(act, i) {
			n++
		}
	}

	return n
}

func (q *QuestRecord) bit(act, quest, bit int) bool {
	slot, ok := QuestSlot(act, quest)

	return ok && q.Get(slot, bit)
}
