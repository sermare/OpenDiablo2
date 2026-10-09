package d2quest

// ---- A2Q0 Jerhyn's welcome ----

func newA2Prologue() *Quest {
	q := &Quest{ID: QuestA2Prologue, Slot: 8, Act: 1, Name: "Act 2 prologue", Label: "A2Q0", Active: true,
		NoSetState: true, SeqID: -1, tables: speechTables("A2Q0")}
	cainHeard := false

	q.activate = func(g *Game, q *Quest, npc int) []Speech {
		switch npc {
		case NPCJerhyn:
			if !g.get(q, FlagRewardGranted) {
				return q.pick(npc, 0)
			}
		case NPCCain2:
			// Cain's remark when the Search for Cain was auto-completed on the trip east
			if g.get(g.byID[QuestCain], FlagCompletedNow) && !cainHeard {
				return q.pick(npc, 1)
			}
		}

		return nil
	}

	q.on[EvMessageHeard] = func(g *Game, q *Quest, e *Event) {
		switch {
		case e.NPC == NPCJerhyn && e.Msg == 253:
			g.set(q, FlagRewardGranted, "Jerhyn's welcome heard")
			g.globalDone(q)
		case e.NPC == NPCCain2 && e.Msg == 125:
			cainHeard = true
		}
	}

	q.active = func(g *Game, q *Quest, npc int) bool {
		return npc == NPCJerhyn && !g.get(q, FlagRewardGranted)
	}

	q.on[EvGameStarted] = func(g *Game, q *Quest, e *Event) {
		if g.get(q, FlagRewardGranted) {
			g.globalDone(q)
		}
	}

	return q
}

// ---- A2Q1 Radament's Lair ----

type radamentData struct {
	atmaActivated bool
	rewarded      bool
}

// ReadBookOfSkill is the hero reading the Book of Skill that Radament drops:
// the reward bit is spent and one skill point is granted.
func (g *Game) ReadBookOfSkill() []Effect {
	q := g.byID[QuestRadament]
	if !g.get(q, FlagCustom1) {
		return nil
	}

	g.clear(q, FlagCustom1, "Book of Skill read")
	g.emit(Effect{Kind: EffectSkillPoint, Quest: q.ID, Value: 1})

	return g.TakeEffects()
}

func newRadament() *Quest {
	d := &radamentData{}
	q := &Quest{ID: QuestRadament, Slot: 9, Act: 1, Name: "Radament's Lair", Label: "A2Q1", LogIndex: 1,
		Active: true, NotIntro: true, State: 1, InitNo: 4, SeqID: -1, tables: speechTables("A2Q1"), data: d}

	q.seq = chainSeq(false, 5)

	q.activate = func(g *Game, q *Quest, npc int) []Speech {
		switch {
		case g.get(q, FlagRewardPending):
			return q.pick(npc, 3)
		case d.rewarded:
			return q.pick(npc, 4)
		case !g.get(q, FlagRewardGranted):
			if q.State == 0 || (q.State >= 4 && !g.get(q, FlagPrimaryGoal)) {
				return nil
			}

			return q.pick(npc, stateIndex(q.State))
		}

		return nil
	}

	q.active = func(g *Game, q *Quest, npc int) bool {
		return npc == NPCAtma && !g.get(q, FlagRewardGranted) &&
			((q.State == 1 && !g.get(q, FlagRewardPending)) || g.get(q, FlagRewardPending))
	}

	q.on[EvMessageHeard] = func(g *Game, q *Quest, e *Event) {
		if e.NPC != NPCAtma {
			return
		}

		switch e.Msg {
		case 304:
			d.atmaActivated = true
			g.setState(q, 2)
			g.updateStateFlags(q)
		case 334:
			if !g.get(q, FlagRewardPending) {
				return
			}

			if g.get(q, FlagPrimaryGoal) && q.State != 5 {
				g.setState(q, 5)
				q.LastState = 13
			}

			g.set(q, FlagRewardGranted, "Atma's thanks")
			g.clear(q, FlagRewardPending, "reward taken")

			d.rewarded = true

			g.chainFrom(q)
			g.emit(Effect{Kind: EffectLogUpdate, Quest: q.ID, Value: 13})
		}
	}

	q.on[EvNpcDeactivate] = func(g *Game, q *Quest, e *Event) {
		if e.NPC != NPCAtma || !d.atmaActivated {
			return
		}

		g.cycle(q, 1, true)
		d.atmaActivated = false
		q.on[EvNpcDeactivate] = nil
	}

	q.on[EvAreaChanged] = func(g *Game, q *Quest, e *Event) {
		// the destination is not checked (an oddity of the original)
		if e.OldLevel != LevelLutGholein || q.State != 2 || g.grantedOrPending(q) {
			return
		}

		g.setState(q, 3)
		g.updateStateFlags(q)
	}

	q.on[EvMonsterKilled] = func(g *Game, q *Quest, e *Event) {
		if !q.NotIntro || e.Monster != NPCRadament {
			return
		}

		g.setState(q, 4)
		g.globalDone(q)
		q.on[EvNpcDeactivate] = nil

		if !g.grantedOrPending(q) {
			g.set(q, FlagPrimaryGoal, "Radament killed")
			g.set(q, FlagRewardPending, "Radament killed")
			g.set(q, FlagCustom1, "Book of Skill earned")
			g.emit(Effect{Kind: EffectSpawn, Quest: q.ID, Code: ItemBookOfSkill, Note: "Radament drops the Book of Skill"})
		}

		g.emit(Effect{Kind: EffectSound, Quest: q.ID, Value: 52, Note: "sound id unverified"})
		g.after(8, func() { g.cycle(q, 3, true) })
	}

	q.on[EvGameStarted] = func(g *Game, q *Quest, e *Event) {
		switch {
		case g.get(q, FlagRewardGranted):
			q.State = 0
			g.globalDone(q)
		case g.get(q, FlagCompletedEarly):
			q.State = 0
		case g.get(q, FlagEnterArea):
			q.State, q.LastState = 3, 2
		case g.get(q, FlagLeaveTown):
			q.State, q.LastState = 3, 1
		case g.get(q, FlagStarted):
			q.State, q.LastState = 2, 1
		}
	}

	return q
}
