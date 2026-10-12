package d2gamescreen

import (
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2quest"
)

// The Act 5 stages of OD2_AUTOQUEST after the Wave 20 audit of the reward flows (d2quest/a5_w20.go): Larzuk's sockets
// and Anya's personalisation complete their quests, Rescue pays runes, Prison of Ice needs the potion, the scroll
// and Anya's item, and Rite of Passage pays experience at the level gate.

func (a *autoQuest) stageSiege() {
	const id = d2quest.QuestSiege

	a.once("siege: Larzuk and Shenk", func(h autoHost, a *autoQuest) {
		a.move(h, d2quest.LevelHarrogath)
		a.offer(h, id, d2quest.NPCLarzuk, 20077)
		a.move(h, d2quest.LevelBloodyFoothills)
		a.expect(h, "ENTERAREA bit, state 3", a.bit(h, id, d2quest.FlagEnterArea) && a.quest(h, id).State == 3)
		a.kill(h, d2quest.Event{Monster: 999, Super: "Shenk the Overseer"})
		a.expect(h, "reward pending + primary goal; Larzuk owes sockets", a.bit(h, id, d2quest.FlagRewardPending) &&
			a.bit(h, id, d2quest.FlagPrimaryGoal) && h.Q().SocketOwed())
		a.move(h, d2quest.LevelHarrogath)
	})
	a.once("siege: Larzuk's thanks and the sockets", func(h autoHost, a *autoQuest) {
		a.expect(h, "Larzuk shows the quest marker", h.Q().Marker(d2quest.NPCLarzuk))
		a.says(h, "Larzuk's thanks", d2quest.NPCLarzuk, 20090)
		a.state(h, id, "thanks heard")
		// VERIFIED 0x584d60: the thanks keep the reward pending (bit 5 set); VERIFIED 0x5856a0: the item action completes it
		a.expect(h, "thanks heard: bit 5 set, reward still pending", a.bit(h, id, d2quest.FlagCustom1) &&
			a.bit(h, id, d2quest.FlagRewardPending) && !a.bit(h, id, d2quest.FlagRewardGranted))
		h.Apply(h.Q().ClaimSocket())
		a.expect(h, "sockets added: reward granted, pending cleared", a.bit(h, id, d2quest.FlagRewardGranted) &&
			!a.bit(h, id, d2quest.FlagRewardPending))
	})
}

func (a *autoQuest) stageRescue() {
	const id = d2quest.QuestRescue

	a.once("rescue: Qual-Kehk and the cages", func(h autoHost, a *autoQuest) {
		a.require(h, d2quest.QuestSiege)
		a.move(h, d2quest.LevelHarrogath)
		a.offer(h, id, d2quest.NPCQualKehk, 20096)
		a.move(h, d2quest.LevelFrigidHighlands)
		a.expect(h, "ENTERAREA bit, state 3", a.bit(h, id, d2quest.FlagEnterArea) && a.quest(h, id).State == 3)

		for i := 0; i < 3; i++ {
			a.kill(h, d2quest.Event{Monster: d2quest.NPCPrisonDoor})
			a.expect(h, "door destroyed", (i < 2) != a.bit(h, id, d2quest.FlagRewardPending))
		}

		a.state(h, id, "fifteen freed")
		a.expect(h, "all fifteen freed: bit 5 and primary goal", a.bit(h, id, d2quest.FlagCustom1) && a.bit(h, id, d2quest.FlagPrimaryGoal))
		a.move(h, d2quest.LevelHarrogath)
	})
	a.once("rescue: Qual-Kehk's runes", func(h autoHost, a *autoQuest) {
		a.expect(h, "Qual-Kehk shows the quest marker", h.Q().Marker(d2quest.NPCQualKehk))
		a.says(h, "Qual-Kehk's reward line", d2quest.NPCQualKehk, 20110)
		a.state(h, id, "reward claimed")
		a.expect(h, "reward granted, pending cleared", a.bit(h, id, d2quest.FlagRewardGranted) && !a.bit(h, id, d2quest.FlagRewardPending))
	})
}

func (a *autoQuest) stagePrison() {
	const id = d2quest.QuestPrison

	a.once("prison: Malah, the frozen Anya and the potion", func(h autoHost, a *autoQuest) {
		a.requireChain(h, d2quest.QuestRescue)
		a.move(h, d2quest.LevelHarrogath)
		a.says(h, "Malah offers the quest", d2quest.NPCMalah, 20116)
		a.expect(h, "STARTED bit, state 2", a.bit(h, id, d2quest.FlagStarted) && a.quest(h, id).State == 2)
		a.move(h, d2quest.LevelFrozenRiver)
		a.expect(h, "left town: state 3", a.quest(h, id).State == 3)
		a.says(h, "the frozen Anya speaks", d2quest.NPCAnyaFrozen, 20131)
		a.expect(h, "state 4", a.quest(h, id).State == 4)
		a.move(h, d2quest.LevelHarrogath)
		a.says(h, "Malah gives the potion", d2quest.NPCMalah, 20127)
		a.pickup(h, d2quest.ItemDefrostPotion)
		a.move(h, d2quest.LevelFrozenRiver)
		a.object(h, d2quest.ObjectFrozenAnya)
		a.state(h, id, "Anya thawed")
		// the potion is handed out once per game, also after it was used on Anya (VERIFIED 0x587460, counter in the node)
		again := h.Q().Hear(d2quest.NPCMalah, 20127)
		a.expect(h, "Malah gives the potion only once per game", !a.hasEffect(h, again, d2quest.EffectGiveItem, d2quest.ItemDefrostPotion))
		a.expect(h, "reward pending + primary goal, state 5", a.bit(h, id, d2quest.FlagRewardPending) &&
			a.bit(h, id, d2quest.FlagPrimaryGoal) && a.quest(h, id).State == 5)
		a.move(h, d2quest.LevelHarrogath)
	})
	a.once("prison: the scroll and Anya's item", func(h autoHost, a *autoQuest) {
		a.says(h, "Malah's scroll", d2quest.NPCMalah, 20132)
		a.expect(h, "scroll given (bit 8), quest still pending", a.bit(h, id, d2quest.FlagCustom4) && !a.bit(h, id, d2quest.FlagRewardGranted))
		a.says(h, "Anya's item", d2quest.NPCDrehya, 20136)
		a.state(h, id, "both given")
		a.expect(h, "reward granted, pending cleared", a.bit(h, id, d2quest.FlagRewardGranted) && !a.bit(h, id, d2quest.FlagRewardPending))
		h.Apply(h.Q().ReadScrollOfResistance())
		a.expect(h, "reading the scroll sets bit 7 and pays the resistances", a.bit(h, id, d2quest.FlagCustom3) && h.Q().ResistBonus() == 10)
	})
}

func (a *autoQuest) stageBetrayal() {
	const id = d2quest.QuestBetrayal

	a.once("betrayal: Anya, the portal and Nihlathak", func(h autoHost, a *autoQuest) {
		a.requireChain(h, d2quest.QuestPrison)
		a.move(h, d2quest.LevelHarrogath)
		a.offer(h, id, d2quest.NPCDrehya, 20137)
		a.move(h, d2quest.LevelNihlathakTemple)
		// VERIFIED 0x589900: the area only moves the state; bit 4 is set by Anya's later line (20148)
		a.expect(h, "state 3, no ENTERAREA bit yet", !a.bit(h, id, d2quest.FlagEnterArea) && a.quest(h, id).State == 3)
		// D2MOO ACT5Q4_UnitIterate_SetRewardPending: Nihlathak counts once Prison of Ice is done or pending
		if !a.bit(h, d2quest.QuestPrison, d2quest.FlagRewardGranted) && !a.bit(h, d2quest.QuestPrison, d2quest.FlagRewardPending) {
			a.kill(h, d2quest.Event{Monster: d2quest.NPCNihlathakBoss})
			a.expect(h, "no reward without the Prison of Ice", !a.bit(h, id, d2quest.FlagRewardPending))
			h.Q().Rec.Set(h.Q().Quest(d2quest.QuestPrison).Slot, d2quest.FlagRewardGranted)
		}

		a.kill(h, d2quest.Event{Monster: d2quest.NPCNihlathakBoss})
		a.state(h, id, "Nihlathak dead")
		a.expect(h, "reward pending + primary goal; Anya owes a personalisation", a.bit(h, id, d2quest.FlagRewardPending) &&
			a.bit(h, id, d2quest.FlagPrimaryGoal) && h.Q().PersonalizeOwed())
		a.move(h, d2quest.LevelHarrogath)
	})
	a.once("betrayal: Anya's thanks and the personalisation", func(h autoHost, a *autoQuest) {
		a.expect(h, "Anya shows the quest marker", h.Q().Marker(d2quest.NPCDrehya))
		a.says(h, "Anya's line", d2quest.NPCDrehya, 20148)
		a.expect(h, "her line pays nothing (VERIFIED 0x589040)", !a.bit(h, id, d2quest.FlagRewardGranted) && a.bit(h, id, d2quest.FlagRewardPending))
		h.Apply(h.Q().ClaimPersonalize())
		a.expect(h, "personalised: reward granted, pending cleared", a.bit(h, id, d2quest.FlagRewardGranted) &&
			!a.bit(h, id, d2quest.FlagRewardPending))
	})
}

func (a *autoQuest) stageRite() {
	const id = d2quest.QuestRite

	a.once("rite: Qual-Kehk and the Ancients", func(h autoHost, a *autoQuest) {
		a.requireChain(h, d2quest.QuestBetrayal)
		a.move(h, d2quest.LevelHarrogath)
		a.offer(h, id, d2quest.NPCQualKehk, 20153)
		a.move(h, d2quest.LevelArreatSummit)
		a.expect(h, "ENTERAREA bit, state 3", a.bit(h, id, d2quest.FlagEnterArea) && a.quest(h, id).State == 3)

		for i, c := range []int{d2quest.NPCAncient1, d2quest.NPCAncient2, d2quest.NPCAncient3} {
			a.kill(h, d2quest.Event{Monster: c})
			a.expect(h, "ancient killed", (i < 2) != a.bit(h, id, d2quest.FlagRewardGranted))
		}

		// VERIFIED 0x58a7b0: the last Ancient completes the quest at once (a hero of the level gate), no reward pending
		a.state(h, id, "Ancients dead")
		a.expect(h, "done + primary goal at once, no reward pending", a.bit(h, id, d2quest.FlagRewardGranted) &&
			a.bit(h, id, d2quest.FlagPrimaryGoal) && !a.bit(h, id, d2quest.FlagRewardPending))
		a.expect(h, "Eve of Destruction opened", a.quest(h, d2quest.QuestEve).State >= 1)
	})
}

// require marks a quest earlier in the Act 5 chain (Siege, Rescue, Prison, Betrayal, Rite) as done when the script
// plays a later one alone; the chain then opens the quest, as it does for a hero who finished the first ones.
func (a *autoQuest) require(h autoHost, id int) {
	h.Q().ForceDone(id)
	h.Infof("AUTOQUEST requires %s done", h.Q().Quest(id).Label)
}

// requireChain marks the quests of the Act 5 chain up to and including id as done.
func (a *autoQuest) requireChain(h autoHost, id int) {
	for _, q := range []int{d2quest.QuestSiege, d2quest.QuestRescue, d2quest.QuestPrison, d2quest.QuestBetrayal} {
		a.require(h, q)

		if q == id {
			return
		}
	}
}
