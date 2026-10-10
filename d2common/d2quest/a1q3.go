package d2quest

import "strings"

// Objects (objects.txt rows) the Act 1 quests react to.
const (
	ObjectHoradricMalus = 108 // the chest the Horadric Malus sits in (A1Q3)
	ObjectCairnStone1   = 17  // cairn stones alpha..lambda are 17..21
	ObjectCairnStone5   = 21
	ObjectCainGibbet    = 26
	ObjectInifussTree   = 30
	ObjectTowerTome     = 8
)

// IsQuestObject reports whether an objects.txt row is operated by a quest
// (the Malus chest, cairn stones, Cain's gibbet, the Inifuss tree, the tower tome).
func IsQuestObject(id int) bool {
	return id == ObjectHoradricMalus || id == ObjectCainGibbet || id == ObjectInifussTree || id == ObjectTowerTome ||
		id == ObjectTaintedSunAltar || id == ObjectOrifice || id == ObjectHorazonJournal ||
		(id >= ObjectCairnStone1 && id <= ObjectCairnStone5)
}

// superIs reports whether the killed monster is the named super unique.
func superIs(e *Event, name string) bool {
	return strings.Contains(strings.ToLower(e.Super), name)
}

type toolsData struct {
	charsiIntro bool
	charsiEnd   bool
	opened      bool // the Malus chest was opened and the item dropped
}

// ClaimImbue is the end of Tools of the Trade: Charsi imbued an item. The
// engine calls it after it applied EffectImbue; the quest then becomes
// inert (ACT1Q3_SetRewardGranted).
func (g *Game) ClaimImbue() []Effect {
	q := g.byID[QuestTools]
	if !g.get(q, FlagRewardPending) {
		return nil
	}

	g.set(q, FlagRewardGranted, "Charsi imbued an item")
	g.clear(q, FlagRewardPending, "reward taken")

	if !g.get(q, FlagCompletedEarly) {
		q.Active = false
	}

	return g.TakeEffects()
}

func newToolsOfTheTrade() *Quest {
	d := &toolsData{}
	q := &Quest{ID: QuestTools, Slot: 3, Act: 0, Name: "Tools of the Trade", Label: "A1Q3", LogIndex: 3,
		Active: true, NotIntro: true, InitNo: 5, SeqID: QuestAndariel, tables: speechTables("A1Q3"), data: d}

	q.seq = chainSeq(true, 5)

	flags := func(g *Game, q *Quest) {
		if g.grantedOrPending(q) {
			return
		}

		switch q.State {
		case 2:
			g.set(q, FlagStarted, "started")
		case 3, 4:
			g.set(q, FlagLeaveTown, "left town")
		}
	}

	q.status = func(g *Game, q *Quest) int {
		switch {
		case g.get(q, FlagRewardPending):
			return 10
		case g.hasItem(ItemHoradricMalus):
			if !g.get(q, FlagRewardGranted) {
				return 2
			}

			return 0
		case !q.NotIntro:
			return 0
		case g.get(q, FlagPrimaryGoal):
			return 13
		case g.get(q, FlagCompletedNow):
			return 12
		case q.State < 5:
			return q.LastState
		case g.GlobalDone(q):
			if g.Hero.Level >= 8 {
				return 12
			}

			return 4
		}

		return 0
	}

	q.activate = func(g *Game, q *Quest, npc int) []Speech {
		if g.get(q, FlagRewardGranted) && !g.get(q, FlagPrimaryGoal) {
			return nil
		}

		if g.hasItem(ItemHoradricMalus) {
			if g.Hero.Level >= 8 && !g.get(q, FlagRewardGranted) {
				return q.pick(npc, 3)
			}

			return nil
		}

		if g.get(q, FlagRewardGranted) || q.State == 0 || q.State == 4 {
			return nil
		}

		return q.pick(npc, stateIndex(q.State))
	}

	q.active = func(g *Game, q *Quest, npc int) bool {
		if npc != NPCCharsi || g.get(q, FlagRewardGranted) {
			return false
		}

		if q.State == 1 && !g.get(q, FlagRewardPending) {
			return true
		}

		return g.Hero.Level >= 8 && g.hasItem(ItemHoradricMalus)
	}

	q.on[EvMessageHeard] = func(g *Game, q *Quest, e *Event) {
		if e.NPC != NPCCharsi {
			return
		}

		switch e.Msg {
		case 146:
			g.setState(q, 2)
			d.charsiIntro = true
		case 163:
			if g.get(q, FlagRewardGranted) {
				return
			}

			if g.hasItem(ItemHoradricMalus) {
				g.set(q, FlagPrimaryGoal, "Horadric Malus handed in")
				g.set(q, FlagRewardPending, "Horadric Malus handed in")
				g.Items[ItemHoradricMalus]--
				g.emit(Effect{Kind: EffectDeleteItem, Quest: q.ID, Code: ItemHoradricMalus})
				g.emit(Effect{Kind: EffectImbue, Quest: q.ID, Note: "Charsi imbues one item (rare, ilvl by clvl)"})

				if q.NotIntro && q.State == 4 {
					g.setState(q, 5)
					d.charsiEnd = true
					g.globalDone(q)
					q.seq(g, q)
				}
			}
		}
	}

	q.on[EvNpcDeactivate] = func(g *Game, q *Quest, e *Event) {
		if e.NPC != NPCCharsi {
			return
		}

		switch {
		case d.charsiIntro:
			flags(g, q)
			g.cycle(q, 1, true)

			d.charsiIntro = false
		case d.charsiEnd:
			g.cycle(q, 13, true)

			d.charsiEnd = false
		}
	}

	q.on[EvAreaChanged] = func(g *Game, q *Quest, e *Event) {
		if !q.NotIntro {
			q.on[EvAreaChanged] = nil
			return
		}

		if e.OldLevel != LevelRogueEncampment || q.State != 2 || g.grantedOrPending(q) {
			return
		}

		if q.LastState != 1 {
			g.cycle(q, 1, true)
		}

		g.setState(q, 3)
		flags(g, q)

		q.on[EvAreaChanged] = nil
	}

	q.on[EvItemPickedUp] = func(g *Game, q *Quest, e *Event) {
		if e.Item != ItemHoradricMalus {
			return
		}

		if !g.get(q, FlagCustom2) {
			g.set(q, FlagCustom2, "Horadric Malus picked up")
			g.emit(Effect{Kind: EffectSound, Quest: q.ID, Value: 36})
		}

		g.cycle(q, 2, true)
	}

	q.on[EvItemDropped] = func(g *Game, q *Quest, e *Event) {
		if e.Item != ItemHoradricMalus || !q.NotIntro || q.State == 5 {
			return
		}

		if g.Items[ItemHoradricMalus] == 0 && d.opened {
			g.cycle(q, 3, true)
		}
	}

	q.on[EvItemRemoved] = func(g *Game, q *Quest, e *Event) {
		// the Malus left the game (sold, destroyed): the chest can be opened again
		if e.Item != ItemHoradricMalus || !q.NotIntro || g.Items[ItemHoradricMalus] != 0 || !d.opened {
			return
		}

		d.opened = false
		g.setState(q, 3)
		g.cycle(q, 1, true)
	}

	q.on[EvObjectOperated] = func(g *Game, q *Quest, e *Event) {
		if e.Object != ObjectHoradricMalus {
			return
		}

		switch {
		case !q.NotIntro:
			g.emit(Effect{Kind: EffectSound, Quest: q.ID, Value: 19})
		case d.opened || g.grantedOrPending(q):
		case g.Hero.Level < 8:
			g.emit(Effect{Kind: EffectSound, Quest: q.ID, Value: 19, Note: "the chest refuses to open below clvl 8"})
		default:
			d.opened = true
			g.emit(Effect{Kind: EffectSpawn, Quest: q.ID, Code: ItemHoradricMalus, Note: "Horadric Malus drops from the chest"})

			if q.State != 4 {
				g.setState(q, 4)
				flags(g, q)
			}

			if q.LastState != 1 {
				q.LastState = 1
			}
		}
	}

	q.on[EvGameStarted] = func(g *Game, q *Quest, e *Event) {
		d.opened = g.hasItem(ItemHoradricMalus)

		if g.get(q, FlagRewardGranted) || g.get(q, FlagCompletedEarly) {
			return
		}

		switch {
		case g.get(q, FlagStarted):
			q.State, q.LastState = 2, 1
		case g.get(q, FlagLeaveTown):
			q.State, q.LastState = 3, 1
		}
	}

	return q
}
