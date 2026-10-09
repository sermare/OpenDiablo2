package d2quest

// ---- A1Q5 The Forgotten Tower (the Countess) ----

type towerData struct {
	deathList  bool // nUnitGUIDs2: the hero was in the Tower Cellar 5 when the Countess died
	heardList  bool // nUnitGUIDs1: the hero heard the congratulations
	triggerSeq bool
}

// the Countess' death line speakers (msgs 140..145): the six town NPCs
func isTowerSuccess(msg int) bool { return msg >= 140 && msg <= 145 }

func newForgottenTower() *Quest {
	d := &towerData{}
	q := &Quest{ID: QuestTower, Slot: 5, Act: 0, Name: "The Forgotten Tower", Label: "A1Q5", LogIndex: 5,
		Active: true, NotIntro: true, InitNo: 4, SeqID: QuestTools, tables: speechTables("A1Q5"), data: d}

	q.seq = chainSeq(false, 5)

	flags := func(g *Game, q *Quest) {
		if g.grantedOrPending(q) {
			return
		}

		switch q.State {
		case 2:
			g.set(q, FlagStarted, "started")
		case 3:
			switch q.LastState {
			case 1:
				g.set(q, FlagLeaveTown, "left town")
			case 2:
				g.set(q, FlagEnterArea, "entered the Tower Cellar")
			case 3:
				g.set(q, FlagCustom1, "tower stage 3")
			case 4:
				g.set(q, FlagCustom2, "tower stage 4")
			}
		}
	}

	// nIndices = {-1,-1,0,1,2,3}
	idx := func(state int) int {
		if state < 2 || state > 5 {
			return -1
		}

		return state - 2
	}

	q.activate = func(g *Game, q *Quest, npc int) []Speech {
		if !q.NotIntro || (g.get(q, FlagRewardGranted) && !g.get(q, FlagPrimaryGoal)) {
			return nil
		}

		if q.State >= 4 && !d.deathList && !d.heardList {
			return nil
		}

		switch {
		case d.deathList:
			return q.pick(npc, 2)
		case !(g.get(q, FlagRewardGranted) && g.get(q, FlagPrimaryGoal)):
			return q.pick(npc, idx(q.State))
		case d.heardList:
			return q.pick(npc, 3)
		}

		return nil
	}

	q.active = func(g *Game, q *Quest, npc int) bool {
		return d.deathList && npc != NPCWarriv1 && npc != NPCGheed
	}

	q.on[EvMessageHeard] = func(g *Game, q *Quest, e *Event) {
		if e.Msg == 127 && q.NotIntro && q.State < 2 { // the Tower Tome was read
			g.setState(q, 2)
			flags(g, q)
		}

		if !isTowerSuccess(e.Msg) {
			return
		}

		if g.get(q, FlagPrimaryGoal) && d.triggerSeq {
			d.triggerSeq = false
			g.setState(q, 5)
			q.seq(g, q)
		}

		if d.deathList {
			d.deathList, d.heardList = false, true
		}
	}

	q.on[EvAreaChanged] = func(g *Game, q *Quest, e *Event) {
		if !q.NotIntro {
			return
		}

		switch {
		case e.NewLevel == LevelForgottenTower:
			if q.State == 0 {
				g.setState(q, 2)
				g.cycle(q, 3, true)
				flags(g, q)
			} else if q.State <= 3 && q.LastState == 1 {
				g.cycle(q, 4, true)
				flags(g, q)
			}
		case e.NewLevel == LevelTowerCellar5:
			if q.State < 4 && q.LastState != 2 {
				if q.State != 3 {
					g.setState(q, 3)
				}

				g.cycle(q, 2, true)
				flags(g, q)
			}
		case e.OldLevel == LevelRogueEncampment:
			if q.State == 2 && !g.get(q, FlagRewardGranted) {
				g.setState(q, 3)
			}
		}
	}

	q.on[EvMonsterKilled] = func(g *Game, q *Quest, e *Event) {
		if !superIs(e, "countess") || !q.NotIntro {
			return
		}

		q.on[EvNpcDeactivate] = nil
		g.setState(q, 5)
		d.triggerSeq = true
		g.globalDone(q)
		q.on[EvMonsterKilled] = nil

		if g.get(q, FlagRewardGranted) {
			return
		}

		if g.Level == LevelTowerCellar5 {
			g.set(q, FlagPrimaryGoal, "Countess killed")
			g.set(q, FlagRewardGranted, "Countess killed")
			g.clear(q, FlagRewardPending, "no reward phase")
			g.emit(Effect{Kind: EffectSound, Quest: q.ID, Value: 37})

			d.deathList = true
		} else {
			g.set(q, FlagCompletedNow, "Countess killed elsewhere")
			g.emit(Effect{Kind: EffectLogUpdate, Quest: q.ID, Value: 0})
		}

		g.emit(Effect{Kind: EffectSpawn, Quest: q.ID, Note: "the Countess' chest erupts"})
	}

	q.on[EvGameStarted] = func(g *Game, q *Quest, e *Event) {
		if g.get(q, FlagRewardGranted) || g.get(q, FlagCompletedEarly) {
			return
		}

		switch {
		case g.get(q, FlagEnterArea):
			q.State, q.LastState = 3, 1
		case g.get(q, FlagCustom2):
			q.LastState, q.State = 4, 3
		case g.get(q, FlagCustom1):
			q.LastState, q.State = 3, 2
		case g.get(q, FlagLeaveTown):
			q.State, q.LastState = 3, 1
		case g.get(q, FlagStarted):
			q.State, q.LastState = 2, 1
		}
	}

	return q
}

// ---- A1Q6 Sisters to the Slaughter (Andariel) ----

type andarielData struct {
	cainActivated bool
	cain, akara   bool // congratulation lists (pGUID1/2/3): the speaker has a line to say
	kashya        bool
	rewarded      bool
}

var chippedGems = []string{"gcv", "gcr", "gcb", "gcy", "gcg", "gcw", "skc"}
var normalGems = []string{"gsv", "gsr", "gsb", "gsy", "gsg", "gsw", "sku"}

func newSistersToTheSlaughter() *Quest {
	d := &andarielData{}
	q := &Quest{ID: QuestAndariel, Slot: 6, Act: 0, Name: "Sisters to the Slaughter", Label: "A1Q6", LogIndex: 6,
		Active: true, NotIntro: true, InitNo: 4, SeqID: QuestA1Intro, tables: speechTables("A1Q6"), data: d}

	q.seq = func(g *Game, q *Quest) bool {
		if q.State == 0 && q.NotIntro {
			g.after(20, func() {
				if q.State == 0 {
					g.setState(q, 1)
				}
			})
		}

		return true
	}

	q.activate = func(g *Game, q *Quest, npc int) []Speech {
		switch {
		case npc == NPCAkara && d.akara, npc == NPCKashya && d.kashya, npc == NPCCain5 && d.cain:
			return q.pick(npc, 3)
		case g.get(q, FlagRewardPending):
			if npc == NPCCain5 || npc == NPCAkara || npc == NPCKashya {
				return q.pick(npc, 4)
			}

			return q.pick(npc, 3)
		case d.rewarded:
			return q.pick(npc, 4)
		case q.State == 1 && npc == NPCCain5 && !g.get(q, FlagRewardGranted):
			return q.pick(NPCCain5, 0)
		}

		pgd := g.get(q, FlagPrimaryGoal)
		rg := g.get(q, FlagRewardGranted)

		if q.State == 0 || (rg && !pgd) || (q.State >= 4 && !pgd) || (rg && pgd) {
			return nil
		}

		return q.pick(npc, stateIndex(q.State))
	}

	q.active = func(g *Game, q *Quest, npc int) bool {
		rg, rp := g.get(q, FlagRewardGranted), g.get(q, FlagRewardPending)

		switch npc {
		case NPCCain5:
			return d.cain || (!rg && q.State == 1 && !rp)
		case NPCWarriv1:
			return !rg && rp
		case NPCAkara:
			return d.akara
		case NPCKashya:
			return d.kashya
		}

		return false
	}

	q.on[EvMessageHeard] = func(g *Game, q *Quest, e *Event) {
		switch {
		case e.NPC == NPCCain5 && e.Msg == 166:
			g.setState(q, 2)
			d.cainActivated = true
			g.updateStateFlags(q)
		case e.NPC == NPCCain5 && e.Msg == 184:
			d.cain = false
		case e.NPC == NPCAkara && e.Msg == 179:
			d.akara = false
		case e.NPC == NPCKashya && e.Msg == 181:
			d.kashya = false
		case e.NPC == NPCWarriv1 && e.Msg == 183:
			if !g.get(q, FlagRewardPending) {
				return
			}

			if g.get(q, FlagPrimaryGoal) {
				g.cycle(q, 13, false)
				g.setState(q, 5)
				g.globalDone(q)
			}

			g.clear(q, FlagRewardPending, "Warriv's thanks")
			g.set(q, FlagRewardGranted, "Warriv's thanks")

			d.rewarded = true

			g.emit(Effect{Kind: EffectLogUpdate, Quest: q.ID, Value: 13})
		}
	}

	q.on[EvNpcDeactivate] = func(g *Game, q *Quest, e *Event) {
		if e.NPC != NPCCain5 || !d.cainActivated {
			return
		}

		g.cycle(q, 1, true)
		d.cainActivated = false
		q.on[EvNpcDeactivate] = nil
	}

	q.on[EvAreaChanged] = func(g *Game, q *Quest, e *Event) {
		if e.NewLevel >= LevelCatacombs1 && e.NewLevel <= LevelCatacombs4 && q.NotIntro {
			if q.State < 3 {
				g.setState(q, 3)
			}

			if e.NewLevel == LevelCatacombs4 {
				if q.LastState < 2 {
					g.cycle(q, 2, true)
					g.updateStateFlags(q)

					return
				}
			} else if q.LastState == 0 {
				g.cycle(q, 1, false)
				g.updateStateFlags(q)

				return
			}

			if q.State < 3 {
				g.updateStateFlags(q)
			}

			return
		}

		if q.State == 4 && e.NewLevel == LevelLutGholein {
			g.setState(q, 5)
			return
		}

		if e.OldLevel != LevelRogueEncampment {
			return
		}

		d.rewarded = false

		if q.State != 2 || g.grantedOrPending(q) {
			return
		}

		g.setState(q, 3)
		g.updateStateFlags(q)
	}

	q.on[EvMonsterKilled] = func(g *Game, q *Quest, e *Event) {
		if e.Monster != NPCAndariel {
			return
		}

		q.on[EvNpcDeactivate] = nil

		if q.NotIntro {
			if !g.get(q, FlagRewardGranted) && !g.get(q, FlagRewardPending) {
				g.set(q, FlagPrimaryGoal, "Andariel killed")
				g.set(q, FlagRewardPending, "Andariel killed")

				// two chipped gems and one standard gem drop (random kinds)
				for i := 0; i < 2; i++ {
					g.emit(Effect{Kind: EffectGiveItem, Quest: q.ID, Code: chippedGems[g.Rand.Intn(len(chippedGems))],
						Note: "Andariel drops a chipped gem"})
				}

				g.emit(Effect{Kind: EffectGiveItem, Quest: q.ID, Code: normalGems[g.Rand.Intn(len(normalGems))],
					Note: "Andariel drops a gem"})
			}

			if g.Level == LevelCatacombs4 && !g.get(q, FlagRewardGranted) {
				d.cain, d.akara, d.kashya = true, true, true
			}

			if !g.get(q, FlagRewardGranted) && !g.get(q, FlagRewardPending) {
				g.set(q, FlagCompletedNow, "Andariel killed")
			}

			g.after(10, func() {
				g.emit(Effect{Kind: EffectPortal, Quest: q.ID, Value: LevelRogueEncampment, Note: "town portal at Andariel's corpse"})
			})
			g.after(12, func() {
				if q.LastState != 3 && q.LastState != 13 {
					g.cycle(q, 3, true)
				}
			})
			g.emit(Effect{Kind: EffectSound, Quest: q.ID, Value: 38, Note: "sound id unverified"})
		}

		g.setState(q, 4)
		q.on[EvMonsterKilled] = nil
	}

	q.on[EvGameStarted] = restoreStd

	return q
}
