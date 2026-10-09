package d2quest

import "github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"

// The "act finished" words of the record (quests.md section 2). Slot 15 is
// set by boarding Meshif and slot 28 by Tyrael (both verified in the binary,
// 0x5446e0); slot 23 (the end of Act 3) is only known from the community
// layout and UNVERIFIED. d2s.QuestSlotAct3Finished (28) is the Act 4 word.
const (
	slotAct2Finished = d2s.QuestSlotAct2Finished
	slotAct3Finished = 23
	slotAct4Finished = 28
)

// Welcome-back lists of the later acts (the same NPC-bit table as Act 1,
// quests-2.md section 5.3).
//
//nolint:gochecknoglobals // static lookup data
var (
	act2Return = []int{NPCAtma, NPCFara, NPCDrognan, NPCLysander, NPCGreiz, NPCElzix, NPCGeglash, NPCMeshif1,
		NPCJerhyn, NPCWarriv2, NPCCain2}
	act3Return = []int{NPCAlkor, NPCAsheara, NPCHratli, NPCOrmus, NPCNatalya, NPCMeshif2, NPCCain3}
	act4Return = []int{NPCTyrael2, NPCCain4}
)

// finishAct sets an "act finished" word (RewardGranted + PrimaryGoal) and, the
// first time, the welcome-back bits of the act's NPCs.
func (g *Game) finishAct(slot int, returns []int) {
	first := !g.Rec.Get(slot, FlagRewardGranted)

	before := g.Rec.Slot(slot)
	g.Rec.Set(slot, FlagRewardGranted)
	g.Rec.Set(slot, FlagPrimaryGoal)
	g.tracef("QUEST act finished slot=%d bits 0x%04x->0x%04x", slot, before, g.Rec.Slot(slot))

	if first {
		for _, c := range returns {
			if NPCBit(c) != 0 {
				g.npcBlock().SetReturnBit(g.Difficulty, NPCBit(c), true)
			}
		}
	}
}

// TravelToAct3 is Meshif taking the hero west (QUESTS_OnActChangeNpcTravel,
// slot 15): an unfinished Horadric Staff quest completes itself (its items are
// deleted), the Act 2 finished word is set and Act 3 opens. The engine decides
// whether to offer the trip (the client only shows it after Duriel).
func (g *Game) TravelToAct3() []Effect {
	if staff := g.byID[QuestHoradricStaff]; staff != nil {
		g.completeStaff(staff)
	}

	g.finishAct(slotAct2Finished, act2Return)
	g.emit(Effect{Kind: EffectUnlockAct, Value: 3})

	return g.TakeEffects()
}

// TravelToAct4 is the hero stepping through Mephisto's portal to the
// Pandemonium Fortress after The Guardian (UNVERIFIED word slot 23).
func (g *Game) TravelToAct4() []Effect {
	g.finishAct(slotAct3Finished, act3Return)
	g.emit(Effect{Kind: EffectUnlockAct, Value: 4})

	return g.TakeEffects()
}

// TravelToAct5 is Tyrael sending the hero to Harrogath after Terror's End
// (slot 28, expansion).
func (g *Game) TravelToAct5() []Effect {
	g.finishAct(slotAct4Finished, act4Return)
	g.emit(Effect{Kind: EffectUnlockAct, Value: 5})

	return g.TakeEffects()
}
