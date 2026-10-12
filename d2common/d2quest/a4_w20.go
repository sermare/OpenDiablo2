package d2quest

// Wave 20 rewrite of The Hellforge (A4Q3).
//
// Evidence. VERIFIED (Game.exe 1.14b, OperateFn 49 = 0x5b3820, read as raw bytes because Ghidra has no function
// there): the forge in its neutral state reacts to the Mephisto Soulstone ("mss"), which is consumed and opens the
// forge; an opened forge needs the Hellforge Hammer ("hfh") in the pack AND wielded; the third operation sets the
// mode of the forge to 3, clears the progress bits of slot 27, sets bits 0xd and 1 (reward pending), sets the
// shower tier to 4 and the quest state to 4. The hammer is destroyed with the smash (0x5b3710 removes "hfh "
// and "mss "). The hammer drop of Hephasto is in a345.go's history (0x5b4190).
// VERIFIED (0x5b3d00, wave 44): Cain's message 679 (0x2a7) creates the soulstone ("mss ") with
// QUEST_GiveQuestItemToPlayer, which does not look for an existing one; the quest data byte +2 and +0x1c are set
// and the state goes to 2 when it was lower, only if the item was created. Message 678 (0x2a6) sets +2 and state 2;
// message 680 (0x2a8) needs bits 1 and 0xd of slot 27. Whether the hero already holds a stone is decided by the
// speech (0x5b3ec0), not by the handler.
// UNVERIFIED in the exe (D2MOO ACT4Q3): the leave-town and start bits, the log pages.

type forgeData struct {
	hits           int
	opened         bool
	stoneGiven     bool
	cainActivated  bool
	rewardedInGame bool
}

func newHellforge() *Quest {
	d := &forgeData{}
	q := &Quest{ID: QuestHellforge, Slot: 27, Act: 3, Name: "The Hellforge", Label: "A4Q3", LogIndex: 3,
		Active: true, NotIntro: true, State: 1, InitNo: 4, SeqID: QuestTerrorsEnd, tables: speechTables("A4Q3")}

	q.seq = func(g *Game, q *Quest) bool {
		if q.State == 0 && q.NotIntro {
			g.setState(q, 1)
		}

		return true
	}

	flags := func(g *Game, q *Quest) {
		if g.grantedOrPending(q) {
			return
		}

		switch q.State {
		case 2:
			if d.stoneGiven {
				g.set(q, FlagCustom1, "Cain has the soulstone")
			} else {
				g.set(q, FlagStarted, "started")
			}
		case 3:
			g.set(q, FlagLeaveTown, "left town")
		}
	}

	q.activate = func(g *Game, q *Quest, npc int) []Speech {
		if npc != NPCCain4 {
			return nil
		}

		switch {
		case g.get(q, FlagRewardPending):
			return q.pick(npc, 2)
		case d.rewardedInGame:
			return q.pick(npc, 3)
		case q.State == 0 || g.get(q, FlagRewardGranted) || !q.NotIntro || g.get(q, FlagRewardPending):
			return nil
		case q.State >= 4 && !g.get(q, FlagPrimaryGoal):
			return nil
		case q.State == 1:
			if g.hasItem(ItemMephistoSoulstone) {
				return q.pick(npc, 0)
			}

			if !d.stoneGiven && q.LastState < 3 {
				return q.pick(npc, 1)
			}
		default:
			if !g.hasItem(ItemMephistoSoulstone) && q.LastState < 3 && !d.stoneGiven {
				return q.pick(npc, 1)
			}
		}

		return nil
	}

	q.active = func(g *Game, q *Quest, npc int) bool {
		return npc == NPCCain4 && !g.grantedOrPending(q) && q.NotIntro && q.State == 1 && !d.stoneGiven
	}

	q.on[EvMessageHeard] = func(g *Game, q *Quest, e *Event) {
		if e.NPC != NPCCain4 {
			return
		}

		switch e.Msg {
		case 678:
			d.cainActivated = true

			g.setState(q, 2)
		case 679:
			// VERIFIED (0x5b3d00): the handler always creates a soulstone; the speech offers the line only to a
			// hero without one
			g.emit(Effect{Kind: EffectGiveItem, Quest: q.ID, Code: ItemMephistoSoulstone,
				Note: "Cain gives a soulstone (0x5b3d00 message 0x2a7)"})

			d.cainActivated, d.stoneGiven = true, true

			if q.State < 2 {
				g.setState(q, 2)
			}
		case 680:
			if !g.get(q, FlagRewardPending) {
				return
			}

			if g.get(q, FlagPrimaryGoal) && q.State != 5 {
				q.LastState = 13
				g.setState(q, 5)
			}

			g.set(q, FlagRewardGranted, "Cain's thanks")
			g.clear(q, FlagRewardPending, "Cain's thanks")
			g.globalDone(q)

			d.rewardedInGame = true

			g.emit(Effect{Kind: EffectLogUpdate, Quest: q.ID, Value: 13})
			g.chainFrom(q)
		}
	}

	q.on[EvNpcDeactivate] = func(g *Game, q *Quest, e *Event) {
		if e.NPC != NPCCain4 || !d.cainActivated {
			return
		}

		if d.stoneGiven {
			q.LastState = 4
		} else {
			q.LastState = 1
		}

		d.cainActivated = false
		flags(g, q)
	}

	q.on[EvAreaChanged] = func(g *Game, q *Quest, e *Event) {
		if e.OldLevel == LevelPandemonium && q.State == 2 && !g.grantedOrPending(q) {
			g.setState(q, 3)
			flags(g, q)
		}
	}

	q.on[EvItemPickedUp] = func(g *Game, q *Quest, e *Event) {
		if q.NotIntro && q.State == 0 && !g.grantedOrPending(q) && q.LastState != 13 {
			g.setState(q, 1)
		}
	}

	// Hephasto's death leaves the hammer (VERIFIED 0x5b4190, attached to class 0x199 by 0x5af8c0): while the node is
	// active the dying unit's item code is stamped "hfh " and dropped; no check of the quest state.
	q.on[EvMonsterKilled] = func(g *Game, q *Quest, e *Event) {
		if e.Monster != NPCHephasto && !nameIs(e, "hephasto") && !nameIs(e, "Hephasto the Armorer") {
			return
		}

		if !q.NotIntro || g.LegacyBossBits {
			return
		}

		g.emit(Effect{Kind: EffectGiveItem, Quest: q.ID, Code: ItemHellforgeHammer,
			Note: "Hephasto drops the hammer (exe: unit code override \"hfh \", 0x5b4190)"})
	}

	// OperateFn 49
	q.on[EvObjectOperated] = func(g *Game, q *Quest, e *Event) {
		if e.Object != ObjectHellforge || g.grantedOrPending(q) {
			return
		}

		if !d.opened {
			if g.hasItem(ItemMephistoSoulstone) {
				d.opened = true

				g.dropItem(ItemMephistoSoulstone)
				g.emit(Effect{Kind: EffectDeleteItem, Quest: q.ID, Code: ItemMephistoSoulstone, Note: "the soulstone opens the Hellforge"})
				q.LastState = 3
			} else if q.State == 0 && q.NotIntro {
				g.setState(q, 1)
			}

			return
		}

		if !g.hasItem(ItemHellforgeHammer) || e.Wielded != ItemHellforgeHammer {
			return
		}

		d.hits++
		if d.hits <= 2 {
			return
		}

		g.dropItem(ItemHellforgeHammer)
		g.emit(Effect{Kind: EffectDeleteItem, Quest: q.ID, Code: ItemHellforgeHammer,
			Note: "the hammer smashes the soulstone on the Hellforge (VERIFIED 0x5b3820)"})
		g.set(q, FlagPrimaryGoal, "the Hellforge is smashed")
		g.set(q, FlagRewardPending, "the Hellforge is smashed")
		g.resetProgress(q, "the Hellforge is smashed")
		g.globalDone(q)
		g.setState(q, 4)
		q.LastState = 13
		g.emit(Effect{Kind: EffectLogUpdate, Quest: q.ID, Value: 13})
	}

	q.on[EvGameStarted] = func(g *Game, q *Quest, _ *Event) {
		switch {
		case g.get(q, FlagRewardGranted), g.get(q, FlagCompletedEarly):
			return
		case !g.hasItem(ItemMephistoSoulstone):
			q.State = 1
		case g.get(q, FlagCustom1):
			d.stoneGiven = true
			q.LastState, q.State = 4, 2
		case g.get(q, FlagLeaveTown):
			q.LastState, q.State = 1, 3
		case g.get(q, FlagStarted):
			q.LastState, q.State = 1, 2
		}
	}

	return q
}
