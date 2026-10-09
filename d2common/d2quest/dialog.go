package d2quest

import "github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"

// Dialog is the NPC message list the server builds when the player clicks an
// NPC (QUEST_DispatchEvent ev0 on every quest of the act, then S2C 0x27).
type Dialog struct {
	NPC   int
	Lines []Speech
}

// Activate builds the speech list of an NPC: every quest of the current act
// appends the lines for its current state. Nodes are visited newest first
// like the original.
func (g *Game) Activate(npc int) *Dialog {
	d := &Dialog{NPC: npc}

	for _, q := range g.reversed() {
		if q.activate == nil || q.Act != actOfLevel(g.Level) {
			continue
		}

		for _, s := range q.activate(g, q, npc) {
			s = s.normalise()
			s.Quest = q.ID
			d.Lines = append(d.Lines, s)
		}
	}

	g.dialog = d

	return d
}

// Heard reports whether the client already played a message (the 9 entry
// heard list of the client).
func (g *Game) Heard(msg int) bool {
	for _, m := range g.heard {
		if m == msg {
			return true
		}
	}

	return false
}

// NextSpoken is the next spoken (mode 0) line that was not heard yet
// (UI_PickNextUnheardNpcMessage).
func (d *Dialog) NextSpoken(g *Game) (Speech, bool) {
	for _, s := range d.Lines {
		if s.Mode == ModeSpoken && !g.Heard(s.Msg) {
			return s, true
		}
	}

	return Speech{}, false
}

// Topics returns the clickable topics (mode 2) of the dialog.
func (d *Dialog) Topics() []Speech {
	var out []Speech

	for _, s := range d.Lines {
		if s.Mode == ModeTopic {
			out = append(out, s)
		}
	}

	return out
}

// Hear records that the client played message msg of npc and runs the
// "message heard" quest event; the returned effects are what the engine has
// to apply. Playing a message that is not in the current dialog is allowed
// (the engine may replay lines).
func (g *Game) Hear(npc, msg int) []Effect {
	if !g.Heard(msg) {
		g.heard = append(g.heard, msg)
		if len(g.heard) > heardCap {
			g.heard = g.heard[1:]
		}
	}

	return g.Dispatch(Event{Kind: EvMessageHeard, NPC: npc, Msg: msg})
}

// ForgetHeard clears the heard list (the client does this when it enters a game).
func (g *Game) ForgetHeard() { g.heard = nil }

// Close is the player leaving the NPC dialog (ev2).
func (g *Game) Close(npc int) []Effect {
	return g.Dispatch(Event{Kind: EvNpcDeactivate, NPC: npc})
}

// Marker reports whether the NPC shows the quest marker ("!" over its head),
// the active filter of the quest nodes of the act.
func (g *Game) Marker(npc int) bool {
	for _, q := range g.Quests {
		if q.active != nil && q.Act == actOfLevel(g.Level) && q.active(g, q, npc) {
			return true
		}
	}

	return false
}

// ---- NPC intro bit fields (PLRINTRO_*) ----

// npcBits is the table at 0x72FE18 in Game.exe 1.14b: monster class -> bit of
// the quest-intro and NPC-intro ("welcome back") fields of the 0x7701 block.
//
//nolint:gochecknoglobals // static lookup data
var npcBits = map[int]int{
	NPCGheed: 1, NPCAkara: 2, NPCKashya: 3, NPCWarriv1: 4, NPCCharsi: 5, NPCCain5: 6, NPCWarriv2: 7,
	NPCAtma: 8, NPCDrognan: 9, NPCFara: 10, NPCLysander: 11, NPCGeglash: 12, NPCMeshif1: 13, NPCJerhyn: 14,
	NPCGreiz: 15, NPCElzix: 16, NPCCain2: 17, NPCCain3: 18, NPCCain4: 19, NPCTyrael1: 20, NPCAsheara: 21,
	NPCHratli: 22, NPCAlkor: 23, NPCOrmus: 24, NPCIzual: 25, 257: 26, NPCMeshif2: 27, NPCNatalya: 28,
	NPCLarzuk: 29, NPCDrehya: 30, NPCMalah: 31, NPCNihlathak: 32, NPCQualKehk: 33, NPCCain6: 34,
}

// NPCBit returns the intro bit of an NPC class; unmapped classes share bit 0
// (the 1.14b quirk, quests-2.md section 5.2).
func NPCBit(class int) int { return npcBits[class] }

// act1Return is the list of Act 1 NPCs of the welcome-back packet (0x72F3A0).
//
//nolint:gochecknoglobals // static lookup data
var act1Return = []int{NPCAkara, NPCCharsi, NPCKashya, NPCGheed, NPCWarriv1, NPCCain5}

func (g *Game) npcBlock() *d2s.NPCBlock {
	if g.NPC == nil {
		g.NPC = &d2s.NPCBlock{}
	}

	return g.NPC
}

// QuestIntroDone reports whether an NPC already gave its first-meeting line.
func (g *Game) QuestIntroDone(class int) bool {
	return g.npcBlock().IntroBit(g.Difficulty, NPCBit(class))
}

func (g *Game) setQuestIntro(class int) {
	g.npcBlock().SetIntroBit(g.Difficulty, NPCBit(class), true)
	g.tracef("QUEST intro flag set npc=%d bit=%d", class, NPCBit(class))
}

// ReturnGreetingPending reports whether the NPC greets with the "welcome
// back" line (the NPC-intro bit).
func (g *Game) ReturnGreetingPending(class int) bool {
	return NPCBit(class) != 0 && g.npcBlock().ReturnBit(g.Difficulty, NPCBit(class))
}

// ClearReturnGreeting is the client packet 0x4d: the greeting was played.
func (g *Game) ClearReturnGreeting(class int) {
	if NPCBit(class) != 0 {
		g.npcBlock().SetReturnBit(g.Difficulty, NPCBit(class), false)
	}
}

// ReturnGreetingNPCs lists the Act 1 NPC classes whose welcome-back bit is
// set (the content of packet 0x91 for the Rogue Encampment).
func (g *Game) ReturnGreetingNPCs() []int {
	var out []int

	for _, c := range act1Return {
		if g.ReturnGreetingPending(c) {
			out = append(out, c)
		}
	}

	return out
}

// TravelToAct2 is Warriv taking the hero to Act 2 (QUESTS_OnActChangeNpcTravel,
// slot 7): the Act 1 finished word gets RewardGranted and PrimaryGoal, an
// unfinished Search for Cain completes itself, and the Act 1 NPCs get their
// welcome-back bit the first time. The engine decides whether to offer the
// trip (the client only shows it after Andariel).
func (g *Game) TravelToAct2() []Effect {
	first := !g.Rec.Get(d2s.QuestSlotAct1Finished, FlagRewardGranted)

	before := g.Rec.Slot(d2s.QuestSlotAct1Finished)
	g.Rec.Set(d2s.QuestSlotAct1Finished, FlagRewardGranted)
	g.Rec.Set(d2s.QuestSlotAct1Finished, FlagPrimaryGoal)
	g.tracef("QUEST act1 finished slot=7 bits 0x%04x->0x%04x", before, g.Rec.Slot(d2s.QuestSlotAct1Finished))

	if first {
		for _, c := range act1Return {
			if NPCBit(c) != 0 {
				g.npcBlock().SetReturnBit(g.Difficulty, NPCBit(c), true)
			}
		}
	}

	cain := g.byID[QuestCain]
	if cain != nil && cain.NotIntro && !g.get(cain, FlagRewardGranted) && !g.get(cain, FlagRewardPending) && cain.State < 6 {
		// ACT1Q4_UpdateQuestStateOnActChange: auto-complete (state 7, completed now)
		g.setState(cain, 7)
		g.cycle(cain, 5, true)
		g.globalDone(cain)
		g.set(cain, FlagCompletedNow, "left for Act 2 without rescuing Cain")
	}

	g.emit(Effect{Kind: EffectUnlockAct, Value: 2})

	return g.TakeEffects()
}
