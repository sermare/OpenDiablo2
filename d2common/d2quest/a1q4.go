package d2quest

type cainData struct {
	akaraIntro   bool // bAkaraIntroActivated
	akaraScroll  bool // bAkaraActivatedForScroll
	treeUsed     bool // the Inifuss tree gave its scroll
	lastStone    bool // bOperatedWithLastCairnStone
	rescued      bool // Cain was freed at the gibbet (unk0x50)
	cainInTown   bool
	rewarded     bool // base player list: Akara's reward (msg 118) was taken this game
	cainPending  bool // pQuestGUID: PGD but has not talked to Cain
	cainHeard    bool // tPlayerGUIDs of the Ex block: heard Cain's rescue line
	stoneOrder   [5]int
	stoneOrdered bool
	curStone     int
	stoneTouches int
}

// cainIdx is the nIndices[] of Search for Cain: state k selects table k-1,
// state 4 (scroll exists) shares table 2.
func cainIdx(state int) int {
	if state == 4 {
		return 2
	}

	return stateIndex(state)
}

func newSearchForCain() *Quest {
	d := &cainData{}
	q := &Quest{ID: QuestCain, Slot: 4, Act: 0, Name: "The Search for Cain", Label: "A1Q4", LogIndex: 4,
		Active: true, NotIntro: true, InitNo: 6, SeqID: QuestTools, tables: speechTables("A1Q4"), data: d}

	q.seq = chainSeq(true, 6)

	flags := func(g *Game, q *Quest) {
		if g.grantedOrPending(q) {
			return
		}

		switch {
		case q.State == 2:
			g.set(q, FlagStarted, "started")
		case q.State >= 3 && q.State <= 5:
			g.set(q, FlagLeaveTown, "left town")
		}
	}

	resetTree := func(g *Game, q *Quest) {
		if d.treeUsed {
			g.emit(Effect{Kind: EffectSpawn, Quest: q.ID, Note: "Inifuss tree resets"})
		}

		g.cycle(q, 1, true)
		g.setState(q, 3)

		if d.treeUsed {
			d.treeUsed = false
			flags(g, q)
		}
	}

	q.activate = func(g *Game, q *Quest, npc int) []Speech {
		var out []Speech

		if npc == NPCCain1 {
			out = append(out, q.pick(NPCCain1, 9)...)
		}

		if d.lastStone && g.hasItem(ItemDecipheredScroll) {
			g.Items[ItemDecipheredScroll]--
			g.emit(Effect{Kind: EffectDeleteItem, Quest: q.ID, Code: ItemDecipheredScroll})
		}

		switch {
		case npc == NPCCain5 && !d.cainHeard && g.get(q, FlagPrimaryGoal):
			return append(out, q.pick(NPCCain5, 5)...)
		case g.get(q, FlagRewardPending):
			if npc != NPCCain5 {
				return append(out, q.pick(npc, 5)...)
			}

			if d.cainHeard {
				return append(out, q.pick(NPCCain5, 7)...)
			}

			return append(out, q.pick(NPCCain5, 5)...)
		case d.cainPending:
			return append(out, q.pick(npc, 6)...)
		case d.rewarded && npc == NPCCain5 && !d.cainHeard:
			return append(out, q.pick(NPCCain5, 5)...)
		case d.rewarded:
			switch {
			case g.get(q, FlagCompletedNow):
				return append(out, q.pick(npc, 8)...)
			case g.get(q, FlagRewardGranted):
				return append(out, q.pick(npc, 7)...)
			}

			return out
		case g.get(q, FlagCompletedNow) || q.State == 0 || g.get(q, FlagRewardGranted) || g.get(q, FlagCompletedEarly):
			return out
		case g.hasItem(ItemScrollOfInifuss):
			return append(out, q.pick(npc, 3)...)
		case q.State == 4:
			return append(out, q.pick(npc, 2)...)
		case q.State < 6:
			return append(out, q.pick(npc, cainIdx(q.State))...)
		}

		return out
	}

	q.active = func(g *Game, q *Quest, npc int) bool {
		switch npc {
		case NPCAkara:
			rg, rp := g.get(q, FlagRewardGranted), g.get(q, FlagRewardPending)

			switch {
			case q.State == 4 && !rg && !rp && g.hasItem(ItemScrollOfInifuss):
				return true
			case q.State == 1 && !rg && !rp:
				return true
			case q.State == 6 && g.get(q, FlagPrimaryGoal) && !rg:
				return true
			}

			return rp
		case NPCCain5:
			return !d.cainPending && !d.cainHeard && g.get(q, FlagPrimaryGoal)
		}

		return false
	}

	q.on[EvMessageHeard] = func(g *Game, q *Quest, e *Event) {
		switch e.NPC {
		case NPCAkara:
			switch e.Msg {
			case 97:
				d.akaraIntro = true
				g.setState(q, 2)
			case 112:
				if g.hasItem(ItemScrollOfInifuss) {
					g.Items[ItemScrollOfInifuss]--
					g.Items[ItemDecipheredScroll]++
					g.emit(Effect{Kind: EffectDeleteItem, Quest: q.ID, Code: ItemScrollOfInifuss})
					g.emit(Effect{Kind: EffectGiveItem, Quest: q.ID, Code: ItemDecipheredScroll, Quality: 0, Value: 1})
					d.akaraScroll = true
					g.setState(q, 5)
					g.cycle(q, 3, false)
				}
			case 118:
				if !g.get(q, FlagRewardPending) {
					return
				}

				g.set(q, FlagRewardGranted, "Akara reward claimed")
				g.clear(q, FlagRewardPending, "reward taken")

				d.rewarded = true

				quality, ilvl := 1, 7 // magic ring
				switch g.Difficulty {
				case Nightmare:
					quality, ilvl = 2, 30
				case Hell:
					quality, ilvl = 2, 60
				}

				g.emit(Effect{Kind: EffectGiveItem, Quest: q.ID, Code: ItemReward, Quality: quality, Value: ilvl,
					Note: "ring from Akara"})
				g.emit(Effect{Kind: EffectLogUpdate, Quest: q.ID, Value: 13})

				if g.get(q, FlagPrimaryGoal) {
					q.LastState = 13

					if q.State != 6 {
						g.setState(q, 6)
					}

					if !g.GlobalDone(q) {
						g.globalDone(q)
						q.seq(g, q)
					}
				}
			}
		case NPCCain5:
			switch e.Msg {
			case 125:
				d.rewarded = true
				d.cainPending = false
			case 126, 123:
				d.cainHeard = true
			}
		}
	}

	q.on[EvNpcDeactivate] = func(g *Game, q *Quest, e *Event) {
		if e.NPC != NPCAkara {
			return
		}

		if d.akaraIntro {
			g.cycle(q, 1, true)

			d.akaraIntro = false

			flags(g, q)
		}

		if d.akaraScroll {
			g.cycle(q, 3, true)
			flags(g, q)

			d.akaraScroll = false
		}
	}

	q.on[EvAreaChanged] = func(g *Game, q *Quest, e *Event) {
		if e.NewLevel == LevelTristram && !d.cainInTown && !d.rescued && q.State >= 6 {
			g.setState(q, 5)
			g.cycle(q, 4, false)
			flags(g, q)
		}

		if e.OldLevel == LevelRogueEncampment {
			d.rewarded = false
			d.cainPending = false

			if !g.grantedOrPending(q) && q.State == 2 {
				g.setState(q, 3)
			}
		}

		switch e.NewLevel {
		case LevelRogueEncampment:
			if d.rescued && !d.cainInTown {
				d.cainInTown = true
				g.emit(Effect{Kind: EffectSpawn, Quest: q.ID, Note: "Cain appears in the Rogue Encampment"})
			}
		case LevelLutGholein:
			if !g.grantedOrPending(q) && !d.rescued && q.State < 6 {
				g.setState(q, 7)
				g.cycle(q, 5, true)
				g.globalDone(q)

				d.cainPending = true
			}
		}
	}

	q.on[EvItemPickedUp] = func(g *Game, q *Quest, e *Event) {
		if q.NotIntro && (e.Item == ItemScrollOfInifuss || e.Item == ItemDecipheredScroll) {
			flags(g, q)
		}
	}

	q.on[EvItemRemoved] = func(g *Game, q *Quest, e *Event) {
		if !q.NotIntro || q.State == 6 || (e.Item != ItemScrollOfInifuss && e.Item != ItemDecipheredScroll) {
			return
		}

		if g.Items[ItemScrollOfInifuss]+g.Items[ItemDecipheredScroll] == 0 && q.State <= 5 && !d.lastStone && d.treeUsed {
			resetTree(g, q)
		}
	}

	q.on[EvMonsterKilled] = func(g *Game, q *Quest, e *Event) {
		// the Cow King: eight stamina potions, once per hero, if the Diablo (classic) /
		// Baal (expansion) quest was done
		if e.Super != "The Cow King" || g.get(q, FlagCustom6) {
			return
		}

		slot := 26 // A4Q2 Terror's End
		if g.Expansion {
			slot = 40 // A5Q6 Eve of Destruction
		}

		if g.Rec.Get(slot, FlagRewardGranted) {
			g.set(q, FlagCustom6, "Cow King killed")
			g.emit(Effect{Kind: EffectGiveItem, Quest: q.ID, Code: "vps", Value: 8, Note: "8 stamina potions drop"})
		}
	}

	q.on[EvObjectOperated] = func(g *Game, q *Quest, e *Event) {
		switch {
		case e.Object == ObjectCainGibbet:
			if !q.NotIntro || d.rescued || q.State >= 6 {
				return
			}

			if g.get(q, FlagRewardPending) || g.get(q, FlagRewardGranted) {
				g.emit(Effect{Kind: EffectSound, Quest: q.ID, Value: 19})
				return
			}

			d.rescued = true

			g.set(q, FlagPrimaryGoal, "Cain freed")
			g.set(q, FlagRewardPending, "Cain freed")
			g.emit(Effect{Kind: EffectSpawn, Quest: q.ID, Note: "Cain steps out of the gibbet (17 frames later)"})
			g.emit(Effect{Kind: EffectLogUpdate, Quest: q.ID, Value: g.LogPage(q)})
		case e.Object == ObjectInifussTree:
			g.treeOperated(q, d, flags)
		case e.Object >= ObjectCairnStone1 && e.Object <= ObjectCairnStone5:
			g.stoneOperated(q, d, e.Object, flags)
		}
	}

	q.on[EvGameStarted] = func(g *Game, q *Quest, e *Event) {
		switch {
		case g.get(q, FlagRewardGranted):
			d.rescued = true

			g.globalDone(q)
		case g.get(q, FlagCompletedEarly):
			d.rescued = true
		case g.get(q, FlagEnterArea):
			q.LastState, q.State = 4, 5
			d.lastStone = true
		case g.get(q, FlagLeaveTown):
			q.State, q.LastState = 3, 1
		case g.get(q, FlagStarted):
			q.State, q.LastState = 2, 1
		}

		switch {
		case g.hasItem(ItemDecipheredScroll):
			q.State, q.LastState = 5, 3
			d.treeUsed = true
		case g.hasItem(ItemScrollOfInifuss):
			d.treeUsed = true
			q.State, q.LastState = 4, 2
		}
	}

	return q
}

// treeOperated is OBJECTS_OperateFunction12_InifussTree.
func (g *Game) treeOperated(q *Quest, d *cainData, flags func(*Game, *Quest)) {
	if !q.NotIntro || q.State >= 6 || d.treeUsed || g.grantedOrPending(q) {
		return
	}

	if g.hasItem(ItemDecipheredScroll) || g.hasItem(ItemScrollOfInifuss) {
		g.emit(Effect{Kind: EffectSound, Quest: q.ID, Value: 19})
		return
	}

	g.emit(Effect{Kind: EffectSound, Quest: q.ID, Value: 45})
	g.setState(q, 4)
	g.emit(Effect{Kind: EffectSpawn, Quest: q.ID, Code: ItemScrollOfInifuss, Note: "the Inifuss tree drops the Scroll of Inifuss"})

	d.treeUsed = true

	g.cycle(q, 2, true)
}

// stoneOrderSetup picks the order the five cairn stones have to be clicked in
// (ACT1Q4_SetMonolithOrder: a random permutation of the classes 17..21).
func (g *Game) stoneOrderSetup(d *cainData) {
	perm := g.Rand.Perm(5)
	for i, p := range perm {
		d.stoneOrder[i] = ObjectCairnStone1 + p
	}

	d.stoneOrdered = true
	g.tracef("QUEST A1Q4 stone order %v", d.stoneOrder)
}

// StoneOrder returns the order the cairn stones must be clicked in (object
// classes 17..21), generating it on first use.
func (g *Game) StoneOrder() [5]int {
	d := g.byID[QuestCain].data.(*cainData)
	if !d.stoneOrdered {
		g.stoneOrderSetup(d)
	}

	return d.stoneOrder
}

// stoneOperated is OBJECTS_OperateFunction09_Monolith.
func (g *Game) stoneOperated(q *Quest, d *cainData, class int, flags func(*Game, *Quest)) {
	if !d.stoneOrdered {
		g.stoneOrderSetup(d)
	}

	if g.get(q, FlagRewardGranted) || g.get(q, FlagRewardPending) {
		g.emit(Effect{Kind: EffectSound, Quest: q.ID, Value: 19})
		return
	}

	if !g.hasItem(ItemDecipheredScroll) {
		if d.stoneTouches%64 == 0 && !g.get(q, FlagLeaveTown) && !g.get(q, FlagEnterArea) {
			g.emit(Effect{Kind: EffectSound, Quest: q.ID, Value: 39})
		}

		d.stoneTouches++

		return
	}

	if !q.NotIntro || q.State >= 6 || d.lastStone {
		return
	}

	if q.State != 5 {
		if q.State == 0 {
			g.setState(q, 1)
		}

		g.setState(q, 5)
	}

	if class != d.stoneOrder[d.curStone] {
		return // a wrong stone is ignored
	}

	d.curStone++
	g.tracef("QUEST A1Q4 cairn stone %d/5 (class %d)", d.curStone, class)

	if d.curStone < 5 {
		return
	}

	d.lastStone = true
	g.Items[ItemDecipheredScroll]--
	g.emit(Effect{Kind: EffectDeleteItem, Quest: q.ID, Code: ItemDecipheredScroll})
	g.emit(Effect{Kind: EffectPortal, Quest: q.ID, Note: "the cairn stones open the portal to Tristram"})

	if q.LastState < 4 {
		g.cycle(q, 4, true)
		g.set(q, FlagEnterArea, "cairn stones solved")
	}
}
