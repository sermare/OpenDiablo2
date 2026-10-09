package d2quest

import "github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"

// ---- helpers shared by the Act 1 quests ----

// grantedOrPending is the "skip" test of the UpdateQuestStateFlags handlers.
func (g *Game) grantedOrPending(q *Quest) bool {
	return g.get(q, FlagRewardGranted) || g.get(q, FlagRewardPending)
}

// updateStateFlags mirrors the quest state into the record bits
// (ACT1Qn_UnitIterate_UpdateQuestStateFlags): state 2 -> STARTED, state 3 ->
// LEAVETOWN when the last log page is 1, else ENTERAREA.
func (g *Game) updateStateFlags(q *Quest) {
	if g.grantedOrPending(q) {
		return
	}

	switch q.State {
	case 2:
		g.set(q, FlagStarted, "started")
	case 3:
		if q.LastState == 1 {
			g.set(q, FlagLeaveTown, "left town")
		} else {
			g.set(q, FlagEnterArea, "entered area")
		}
	}
}

// restoreStd maps the progress bits back to the quest state
// (ACT1Qn_Callback13_PlayerStartedGame).
func restoreStd(g *Game, q *Quest, _ *Event) {
	if g.get(q, FlagRewardGranted) || g.get(q, FlagCompletedEarly) {
		return
	}

	switch {
	case g.get(q, FlagEnterArea):
		q.LastState, q.State = 2, 3
	case g.get(q, FlagLeaveTown):
		q.State, q.LastState = 3, 1
	case g.get(q, FlagStarted):
		q.State, q.LastState = 2, 1
	}
}

// pick returns table idx of q when idx is valid.
func (q *Quest) pick(npc, idx int) []Speech {
	var out []Speech

	for _, s := range q.Table(idx) {
		if s.NPC == npc {
			out = append(out, s)
		}
	}

	return out
}

// stateIndex is the nIndices[] lookup shared by the Act 1 quests: state k
// selects table k-1.
func stateIndex(state int) int {
	if state < 1 || state > 5 {
		return -1
	}

	return state - 1
}

// chainSeq builds the SeqCallback of a quest: first=true quests become
// available (state 1) when the chain reaches them in state 0.
func chainSeq(startAtZero bool, final int) func(g *Game, q *Quest) bool {
	return func(g *Game, q *Quest) bool {
		if startAtZero && q.State == 0 && q.NotIntro {
			g.setState(q, 1)
			return true
		}

		if q.State != final && q.NotIntro {
			return true
		}

		next := g.byID[q.SeqID]
		if next == nil || next.seq == nil {
			return false
		}

		return next.seq(g, next)
	}
}

// ---- Act 1 intro (first meeting lines) ----

func newA1Intro() *Quest {
	q := &Quest{ID: QuestA1Intro, Slot: -1, Act: 0, Name: "Act 1 intro", Label: "A1Intro", Active: true,
		NoSetState: true, SeqID: -1}
	q.tables = [][]Speech{
		{{NPC: NPCAkara, Msg: 11}, {NPC: NPCGheed, Msg: 45}, {NPC: NPCCharsi, Msg: 36}, {NPC: NPCKashya, Msg: 24}},
		{{NPC: NPCAkara, Msg: 12}, {NPC: NPCGheed, Msg: 46}, {NPC: NPCCharsi, Msg: 37}, {NPC: NPCKashya, Msg: 25}},
	}

	special := map[int]int{NPCKashya: ClassAmazon, NPCAkara: ClassSorceress, NPCGheed: ClassNecromancer, NPCCharsi: ClassBarbarian}

	q.activate = func(g *Game, q *Quest, npc int) []Speech {
		cls, ok := special[npc]
		if !ok || g.QuestIntroDone(npc) {
			return nil
		}

		if g.Hero.Class == cls {
			return q.pick(npc, 1)
		}

		return q.pick(npc, 0)
	}

	q.on[EvMessageHeard] = func(g *Game, q *Quest, e *Event) {
		for i := 0; i < 2; i++ {
			for _, s := range q.Table(i) {
				if s.NPC == e.NPC && s.Msg == e.Msg {
					g.setQuestIntro(e.NPC)
					return
				}
			}
		}
	}

	q.active = func(g *Game, q *Quest, npc int) bool {
		a1q1 := g.byID[QuestDenOfEvil]

		return !g.get(a1q1, FlagRewardGranted) && npc == NPCAkara && !g.QuestIntroDone(NPCAkara)
	}

	return q
}

// ---- A1Q0 Warriv's welcome ----

func newA1Prologue() *Quest {
	q := &Quest{ID: QuestA1Prologue, Slot: 0, Act: 0, Name: "Act 1 prologue", Label: "A1Q0", Active: true,
		NoSetState: true, SeqID: -1, tables: speechTables("A1Q0")}

	q.activate = func(g *Game, q *Quest, npc int) []Speech {
		if npc != NPCWarriv1 || g.get(q, FlagRewardGranted) {
			return nil
		}

		if g.Hero.Class == ClassPaladin {
			return q.pick(npc, 1)
		}

		return q.pick(npc, 0)
	}

	q.on[EvMessageHeard] = func(g *Game, q *Quest, e *Event) {
		if e.NPC == NPCWarriv1 && (e.Msg == 0 || e.Msg == 1) {
			g.set(q, FlagRewardGranted, "Warriv welcome heard")
		}
	}

	q.active = func(g *Game, q *Quest, npc int) bool {
		return npc == NPCWarriv1 && !g.get(q, FlagRewardGranted)
	}

	return q
}

// ---- A1Q7 Navi, the rogue scout ----

func newNavi() *Quest {
	q := &Quest{ID: QuestNavi, Slot: 29, Act: 0, Name: "Navi", Label: "A1Q7", Active: true,
		NoSetState: true, SeqID: -1, tables: speechTables("A1Q7")}

	denOpen := func(g *Game) bool {
		den := g.byID[QuestDenOfEvil]

		return g.GlobalDone(den) || !den.NotIntro || g.get(den, FlagRewardGranted) ||
			g.get(den, FlagPrimaryGoal) || g.get(den, FlagRewardPending)
	}

	q.activate = func(g *Game, q *Quest, npc int) []Speech {
		if npc != NPCNavi {
			return nil
		}

		if denOpen(g) {
			return q.pick(npc, 2+g.Rand.Intn(3))
		}

		return q.pick(npc, g.Rand.Intn(2))
	}

	q.active = func(g *Game, q *Quest, npc int) bool {
		den := g.byID[QuestDenOfEvil]

		return npc == NPCNavi && den.NotIntro && !g.GlobalDone(den) && !g.get(den, FlagRewardGranted) &&
			!g.get(den, FlagPrimaryGoal) && !g.get(den, FlagRewardPending)
	}

	return q
}

// ---- A1Q1 Den of Evil ----

type denData struct {
	akaraActivated bool
	rewarded       bool // tPlayerGUIDs: the hero was rewarded this game
}

// SetDenMonsters tells the quest system how many monsters the Den of Evil
// holds (the original reads the level's spawn count); kills are counted from
// EvMonsterKilled events in that level.
func (g *Game) SetDenMonsters(total int) {
	g.denSpawned, g.denKnown = total, total > 0
	g.denKilled = 0
}

// SyncDenMonsters is called just before a Den of Evil kill event with the
// number of monsters still alive in the level (after the kill): the original
// reads the same figure from the level's spawn and kill counters. It makes the
// quest independent of how many monsters the level generator spawned.
func (g *Game) SyncDenMonsters(aliveAfterKill int) {
	g.denKnown = true
	g.denSpawned = g.denKilled + 1 + aliveAfterKill
}

// DenMonstersLeft returns the number of Den monsters still alive (-1 unknown).
func (g *Game) DenMonstersLeft() int {
	if !g.denKnown {
		return -1
	}

	return g.denSpawned - g.denKilled
}

func newDenOfEvil() *Quest {
	d := &denData{}
	q := &Quest{ID: QuestDenOfEvil, Slot: 1, Act: 0, Name: "Den of Evil", Label: "A1Q1", LogIndex: 1,
		Active: true, NotIntro: true, State: 1, InitNo: 4, SeqID: QuestBurial, tables: speechTables("A1Q1"), data: d}

	q.seq = chainSeq(false, 5)

	q.activate = func(g *Game, q *Quest, npc int) []Speech {
		switch {
		case g.get(q, FlagRewardPending):
			return q.pick(npc, 3)
		case d.rewarded:
			return q.pick(npc, 4)
		case !g.get(q, FlagRewardGranted) && (q.State < 4 || g.get(q, FlagPrimaryGoal)):
			if !q.NotIntro {
				return nil
			}

			return q.pick(npc, stateIndex(q.State))
		}

		return nil
	}

	q.active = func(g *Game, q *Quest, npc int) bool {
		if npc != NPCAkara || g.get(q, FlagRewardGranted) {
			return false
		}

		return (q.NotIntro && q.State == 1 && !g.get(q, FlagRewardPending)) || g.get(q, FlagRewardPending)
	}

	q.on[EvMessageHeard] = func(g *Game, q *Quest, e *Event) {
		if e.NPC != NPCAkara {
			return
		}

		switch e.Msg {
		case 64: // Akara hands out the quest
			d.akaraActivated = true
			g.setState(q, 2)
			g.updateStateFlags(q)
		case 76: // Akara's thanks: claim
			if !g.get(q, FlagRewardPending) {
				return
			}

			if g.get(q, FlagPrimaryGoal) && q.State != 5 {
				g.setState(q, 5)
				q.LastState = 13
				q.seq(g, q)
			}

			g.set(q, FlagRewardGranted, "Akara reward claimed")
			g.clear(q, FlagRewardPending, "reward taken")
			g.resetProgress(q, "quest complete")
			g.emit(Effect{Kind: EffectSkillPoint, Quest: q.ID, Value: 1})
			// 1.14b only (found in the binary, not in D2MOO): the free respec flags in slot 41
			before := g.Rec.Slot(d2s.QuestSlotAkaraRespec)
			g.Rec.Set(d2s.QuestSlotAkaraRespec, FlagRewardPending)
			g.Rec.Set(d2s.QuestSlotAkaraRespec, FlagPrimaryGoal)
			g.tracef("QUEST A1Q1 respec slot=41 bits 0x%04x->0x%04x", before, g.Rec.Slot(d2s.QuestSlotAkaraRespec))
			g.emit(Effect{Kind: EffectRespec, Quest: q.ID})
			g.emit(Effect{Kind: EffectLogUpdate, Quest: q.ID, Value: 13})

			d.rewarded = true
		}
	}

	q.on[EvNpcDeactivate] = func(g *Game, q *Quest, e *Event) {
		if e.NPC != NPCAkara || !d.akaraActivated {
			return
		}

		g.cycle(q, 1, true)
		d.akaraActivated = false
		q.on[EvNpcDeactivate] = nil
	}

	q.on[EvAreaChanged] = func(g *Game, q *Quest, e *Event) {
		switch {
		case e.NewLevel == LevelDenOfEvil:
			if !q.NotIntro {
				return
			}

			if q.State == 1 || q.State == 2 {
				g.setState(q, 3)
			}

			if q.LastState >= 2 {
				if q.State != 1 && q.State != 2 {
					return // the original returns here without touching the bits
				}
			} else {
				g.cycle(q, 2, true)
				q.on[EvNpcDeactivate] = nil
			}

			g.updateStateFlags(q)
		case e.OldLevel == LevelRogueEncampment:
			d.rewarded = false

			if q.State != 2 || g.grantedOrPending(q) {
				return
			}

			g.setState(q, 3)
			g.updateStateFlags(q)

			if q.LastState == 1 {
				return
			}

			g.cycle(q, 1, true)
			q.on[EvNpcDeactivate] = nil
		}
	}

	q.on[EvMonsterKilled] = func(g *Game, q *Quest, e *Event) {
		if !q.NotIntro || e.Level != LevelDenOfEvil || !g.denKnown {
			return
		}

		g.denKilled++

		left := g.denSpawned - g.denKilled
		if left > 0 {
			if left <= 5 {
				g.cycle(q, 4, true) // "only a few monsters left" page
				q.on[EvNpcDeactivate] = nil
			}

			return
		}

		// the whole cave is clear
		q.on[EvNpcDeactivate] = nil
		g.setState(q, 4)
		q.on[EvMonsterKilled] = nil
		g.globalDone(q)

		if !g.grantedOrPending(q) {
			g.set(q, FlagPrimaryGoal, "Den of Evil cleared")
			g.set(q, FlagRewardPending, "Den of Evil cleared")
		}

		g.emit(Effect{Kind: EffectSound, Quest: q.ID, Value: 35})

		g.after(8, func() {
			if q.State == 4 {
				g.cycle(q, 5, true)
			}
		})
	}

	q.on[EvGameStarted] = restoreStd

	return q
}

// ---- A1Q2 Sisters' Burial Grounds (Blood Raven) ----

type burialData struct {
	kashyaActivated bool
	rewarded        bool
}

func newBurialGrounds() *Quest {
	d := &burialData{}
	q := &Quest{ID: QuestBurial, Slot: 2, Act: 0, Name: "Sisters' Burial Grounds", Label: "A1Q2", LogIndex: 2,
		Active: true, NotIntro: true, InitNo: 4, SeqID: QuestCain, tables: speechTables("A1Q2"), data: d}

	q.seq = chainSeq(true, 5)

	q.activate = func(g *Game, q *Quest, npc int) []Speech {
		switch {
		case g.get(q, FlagRewardPending):
			return q.pick(npc, 3)
		case d.rewarded:
			return q.pick(npc, 4)
		case q.State != 0 && !g.get(q, FlagRewardGranted) && q.State < 4:
			return q.pick(npc, stateIndex(q.State))
		}

		return nil
	}

	q.active = func(g *Game, q *Quest, npc int) bool {
		if npc != NPCKashya || g.get(q, FlagRewardGranted) {
			return false
		}

		return (q.State == 1 && !g.get(q, FlagRewardPending)) || g.get(q, FlagRewardPending)
	}

	q.on[EvMessageHeard] = func(g *Game, q *Quest, e *Event) {
		if e.NPC != NPCKashya {
			return
		}

		switch e.Msg {
		case 81:
			d.kashyaActivated = true
			g.setState(q, 2)
			g.updateStateFlags(q)
		case 92:
			if !g.get(q, FlagRewardPending) {
				return
			}

			if g.get(q, FlagPrimaryGoal) && q.State != 5 {
				q.LastState = 13
				g.setState(q, 5)
				q.seq(g, q)
			}

			g.set(q, FlagRewardGranted, "Kashya reward claimed")
			g.clear(q, FlagRewardPending, "reward taken")
			d.rewarded = true
			g.emit(Effect{Kind: EffectHireRogues, Quest: q.ID, Note: "Kashya's rogue mercenaries become hirable"})
			g.emit(Effect{Kind: EffectLogUpdate, Quest: q.ID, Value: 13})
		}
	}

	q.on[EvNpcDeactivate] = func(g *Game, q *Quest, e *Event) {
		if e.NPC != NPCKashya || !d.kashyaActivated {
			return
		}

		g.cycle(q, 1, true)
		d.kashyaActivated = false
		q.on[EvNpcDeactivate] = nil

		g.updateStateFlags(q)
	}

	q.on[EvAreaChanged] = func(g *Game, q *Quest, e *Event) {
		if e.NewLevel == LevelBurialGrounds && q.NotIntro {
			// the original keeps the result of the first state check in a boolean
			updated := q.State < 3
			if updated {
				g.setState(q, 3)
			}

			if q.LastState == 1 || q.LastState == 0 {
				g.cycle(q, 2, true)
				g.updateStateFlags(q)

				return
			}

			if updated {
				g.updateStateFlags(q)
				return
			}
		}

		if e.OldLevel == LevelRogueEncampment {
			d.rewarded = false

			if q.State != 2 || g.grantedOrPending(q) {
				return
			}

			g.setState(q, 3)
			g.updateStateFlags(q)
		}
	}

	q.on[EvMonsterKilled] = func(g *Game, q *Quest, e *Event) {
		// the handler never checks the monster id: Blood Raven carries the quest chain record
		if !q.NotIntro || e.Monster != NPCBloodRaven {
			return
		}

		g.setState(q, 4)

		if !g.grantedOrPending(q) {
			g.set(q, FlagPrimaryGoal, "Blood Raven killed")
			g.set(q, FlagRewardPending, "Blood Raven killed")
		}

		g.emit(Effect{Kind: EffectSound, Quest: q.ID, Value: 34})
		g.after(15, func() { g.cycle(q, 3, true) })

		q.on[EvNpcDeactivate] = nil
		g.globalDone(q)
	}

	q.on[EvGameStarted] = restoreStd

	return q
}
