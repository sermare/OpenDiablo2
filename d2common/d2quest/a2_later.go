package d2quest

// The later quests of Act 2: The Horadric Staff (A2Q2, Cain), The Tainted Sun
// (A2Q3, Drognan), The Arcane Sanctuary (A2Q4, Drognan and Jerhyn), The Summoner
// (A2Q5) and the speech side of The Seven Tombs (A2Q6, Jerhyn; the Duriel kill is
// in boss.go). The state machines follow quests-2.md sections for A2Q2..A2Q6
// (clean-room D2MOO sources, marked [SRC] there); the speech tables are the ones
// of data/messages.csv. UNVERIFIED against the 1.14b binary: every handler apart
// from the speech tables and the Duriel bit 5; the delays of the timers; the
// object id of Horazon's journal (357, "Tome" of Objects.txt); the sound ids.
//
// Simplifications (one player per game): the party loops collapse to the hero.

// Objects.txt rows used by the Act 2 quests.
const (
	ObjTaintedSunAltar = 149
	ObjStaffOrifice    = 152
	ObjHorazonJournal  = 357 // UNVERIFIED: the sanctuary "Tome"
)

var a2cainLines = [...]int{335, 336, 337, 338, 339}

// ---- A2Q2 The Horadric Staff ----

func newHoradricStaff() *Quest {
	q := &Quest{ID: QuestStaff, Slot: 10, Act: 1, Name: "The Horadric Staff", Label: "A2Q2", LogIndex: 2,
		Active: true, NotIntro: true, NoSetState: true, SeqID: -1, tables: speechTables("A2Q2")}

	q.activate = func(g *Game, q *Quest, npc int) []Speech {
		if npc != NPCCain2 || g.get(q, FlagRewardGranted) {
			return nil
		}

		var out []Speech

		switch {
		case g.hasItem(ItemHoradricStaff) && !g.get(q, FlagCustom6):
			out = q.pick(npc, 4)
		case g.hasItem(ItemHoradricScroll):
			out = q.pick(npc, 0)
		case g.hasItem(ItemAmuletOfTheViper) && !g.get(q, FlagEnterArea):
			out = q.pick(npc, 1)
		case g.hasItem(ItemStaffOfKingsShaft) && !g.get(q, FlagCustom1):
			out = q.pick(npc, 2)
		case g.hasItem(ItemHoradricCube) && !g.get(q, FlagCustom2):
			out = q.pick(npc, 3)
		}

		if len(out) == 0 {
			// topics for what was reported before
			if g.get(q, FlagLeaveTown) {
				out = append(out, q.pick(npc, 6)...)
			}

			for i, bit := range []int{FlagEnterArea, FlagCustom1, FlagCustom2} {
				if g.get(q, bit) {
					out = append(out, q.pick(npc, 7+i)...)
				}
			}
		}

		return out
	}

	q.active = func(g *Game, q *Quest, npc int) bool {
		return npc == NPCCain2 && !g.get(q, FlagRewardGranted) && len(q.activate(g, q, npc)) > 0 &&
			q.activate(g, q, npc)[0].Mode == ModeSpoken
	}

	q.on[EvMessageHeard] = func(g *Game, q *Quest, e *Event) {
		if e.NPC != NPCCain2 {
			return
		}

		switch e.Msg {
		case 335:
			g.consume(ItemHoradricScroll)
			g.set(q, FlagLeaveTown, "Cain read the Horadric Scroll")
		case 336:
			g.set(q, FlagLeaveTown, "Cain heard of the amulet")
			g.set(q, FlagEnterArea, "Cain heard of the amulet")
		case 337:
			g.set(q, FlagLeaveTown, "Cain heard of the staff")
			g.set(q, FlagCustom1, "Cain heard of the staff")
		case 338:
			g.set(q, FlagLeaveTown, "Cain heard of the cube")
			g.set(q, FlagCustom2, "Cain heard of the cube")
		case 339:
			g.clear(q, FlagRewardPending, "Cain explained the assembled staff")

			for _, bit := range []int{FlagLeaveTown, FlagCustom6, FlagEnterArea, FlagCustom2, FlagCustom1} {
				g.set(q, bit, "Cain explained the assembled staff")
			}
		}
	}

	return q
}

// consume removes one quest item from the hero (and asks the engine to).
func (g *Game) consume(code string) {
	if g.Items[code] > 0 {
		g.Items[code]--
	}

	g.emit(Effect{Kind: EffectDeleteItem, Code: code})
}

// CubeHoradricStaff is the cube recipe Staff of Kings + Amulet of the Viper =
// Horadric Staff (cubemain; engine hook for the Cube): both parts are consumed,
// the staff is created, CUSTOM7 of the Horadric Staff slot is set and the Arcane
// Sanctuary quest is offered.
func (g *Game) CubeHoradricStaff() []Effect {
	if !g.hasItem(ItemStaffOfKingsShaft) || !g.hasItem(ItemAmuletOfTheViper) {
		return nil
	}

	q := g.byID[QuestStaff]
	g.consume(ItemStaffOfKingsShaft)
	g.consume(ItemAmuletOfTheViper)
	g.Items[ItemHoradricStaff]++
	g.emit(Effect{Kind: EffectGiveItem, Code: ItemHoradricStaff, Quality: 0, Note: "cube: Staff of Kings + Amulet of the Viper"})
	g.set(q, 11, "Horadric Staff cubed")
	g.offerArcane()

	return g.TakeEffects()
}

// staffPlaced is the Horadric Staff put into Tal Rasha's orifice: the quest is
// complete and the staff parts are deleted (ACT2Q6_DeleteAllHoradricItemsAndOpenTomb).
func (g *Game) finishStaff() {
	q := g.byID[QuestStaff]
	if q == nil || g.get(q, FlagRewardGranted) {
		return
	}

	g.set(q, FlagRewardGranted, "the staff was placed")
	g.set(q, FlagPrimaryGoal, "the staff was placed")
	g.globalDone(q)

	for _, c := range []string{ItemHoradricStaff, ItemStaffOfKingsShaft, ItemAmuletOfTheViper} {
		for g.hasItem(c) {
			g.consume(c)
		}
	}
}

// ---- A2Q3 The Tainted Sun ----

type sunData struct{ rewarded, dark bool }

func newTaintedSun() *Quest {
	d := &sunData{}
	q := &Quest{ID: QuestTaintedSun, Slot: 11, Act: 1, Name: "The Tainted Sun", Label: "A2Q3", LogIndex: 3,
		Active: true, NotIntro: true, SeqID: QuestArcane, tables: speechTables("A2Q3"), data: d}

	q.activate = func(g *Game, q *Quest, npc int) []Speech {
		switch {
		case g.get(q, FlagRewardPending):
			return q.pick(npc, 3)
		case d.rewarded:
			return q.pick(npc, 4)
		case g.get(q, FlagRewardGranted):
			return nil
		case q.State >= 1 && q.State <= 3:
			return q.pick(npc, q.State-1)
		}

		return nil
	}

	q.active = func(g *Game, q *Quest, npc int) bool {
		return !g.get(q, FlagRewardGranted) && ((q.State == 1 && npc == NPCDrognan) ||
			(g.get(q, FlagRewardPending) && len(q.pick(npc, 3)) > 0))
	}

	darken := func(g *Game, q *Quest) {
		if q.State != 0 || g.get(q, FlagRewardGranted) || g.get(q, FlagCompletedEarly) {
			return
		}

		d.dark = true

		g.after(15, func() {
			g.setState(q, 1)
			g.set(q, FlagStarted, "the sun is tainted")
			g.logPage(q, 1)
		})
	}

	q.on[EvAreaChanged] = func(g *Game, q *Quest, e *Event) {
		switch {
		case e.NewLevel == LevelLostCity || e.NewLevel == LevelValleyOfSnakes:
			darken(g, q)

			if q.State == 2 {
				// the altar level is entered before talking to Drognan or after: the original jumps to 3
				g.setState(q, 3)
				g.set(q, FlagEnterArea, "entered the far desert")
			}
		case e.OldLevel == LevelLutGholein && q.State == 2 && !g.grantedOrPending(q):
			g.set(q, FlagLeaveTown, "left Lut Gholein")
			g.setState(q, 3)
		}
	}

	q.on[EvMessageHeard] = func(g *Game, q *Quest, e *Event) {
		switch {
		case e.NPC == NPCDrognan && e.Msg == 348 && q.State == 1:
			g.setState(q, 2)
			g.set(q, FlagLeaveTown, "Drognan explained the sun")
		case e.Msg >= 362 && e.Msg <= 372 && g.get(q, FlagRewardPending):
			g.set(q, FlagRewardGranted, "the sun's reward heard")
			g.clear(q, FlagRewardPending, "the sun's reward heard")
			g.setState(q, 5)

			d.rewarded = true

			g.offerArcane()
		}
	}

	q.on[EvObjectOperated] = func(g *Game, q *Quest, e *Event) {
		if e.Object != ObjTaintedSunAltar || q.State >= 4 || g.get(q, FlagRewardGranted) {
			return
		}

		g.setState(q, 4)
		g.set(q, FlagPrimaryGoal, "the altar was destroyed")
		g.set(q, FlagRewardPending, "the altar was destroyed")
		g.globalDone(q)

		if !g.hasItem(ItemAmuletOfTheViper) && !g.hasItem(ItemHoradricStaff) && !g.get(g.byID[QuestStaff], FlagRewardGranted) {
			g.emit(Effect{Kind: EffectSpawn, Quest: q.ID, Code: ItemAmuletOfTheViper, Note: "the altar drops the Amulet of the Viper"})
		}

		g.emit(Effect{Kind: EffectSound, Quest: q.ID, Value: 52, Note: "sound id unverified"})
		g.logPage(q, 4)
		d.dark = false
	}

	q.on[EvGameStarted] = func(g *Game, q *Quest, _ *Event) {
		switch {
		case g.get(q, FlagRewardGranted):
			q.State, d.rewarded = 0, true
		case g.get(q, FlagEnterArea):
			q.State = 3
		case g.get(q, FlagLeaveTown):
			q.State = 2
		case g.get(q, FlagStarted):
			q.State = 1
		}
	}

	return q
}

// offerArcane makes the Arcane Sanctuary quest available (SeqCallback of A2Q3 and the
// cube recipe).
func (g *Game) offerArcane() {
	if q := g.byID[QuestArcane]; q != nil && q.State == 0 && !g.get(q, FlagRewardGranted) {
		g.setState(q, 1)
	}
}

// ---- A2Q4 The Arcane Sanctuary ----

type arcaneData struct{ rewarded bool }

func newArcaneSanctuary() *Quest {
	d := &arcaneData{}
	q := &Quest{ID: QuestArcane, Slot: 12, Act: 1, Name: "The Arcane Sanctuary", Label: "A2Q4", LogIndex: 4,
		Active: true, NotIntro: true, SeqID: QuestSevenTombs, tables: speechTables("A2Q4"), data: d}

	q.activate = func(g *Game, q *Quest, npc int) []Speech {
		if npc == NPCPalaceGuard {
			if q.State >= 2 {
				return q.pick(npc, 8+g.Rand.Intn(3))
			}

			return q.pick(npc, 7)
		}

		switch {
		case g.get(q, FlagRewardPending):
			return q.pick(npc, 4)
		case d.rewarded:
			return q.pick(npc, 5)
		case q.State >= 1 && q.State <= 5 && !g.get(q, FlagRewardGranted):
			return q.pick(npc, q.State-1)
		}

		return nil
	}

	q.active = func(g *Game, q *Quest, npc int) bool {
		return !g.get(q, FlagRewardGranted) && ((q.State == 1 && npc == NPCDrognan) || (q.State == 2 && npc == NPCJerhyn))
	}

	q.on[EvMessageHeard] = func(g *Game, q *Quest, e *Event) {
		switch {
		case e.NPC == NPCDrognan && e.Msg == 373 && q.State == 1:
			g.setState(q, 2)
			g.set(q, FlagStarted, "Drognan asked about the palace")
			g.emit(Effect{Kind: EffectSpawn, Quest: q.ID, Note: "the harem is opened (blocker object 318)"})
		case e.NPC == NPCJerhyn && e.Msg == 377 && q.State == 2:
			g.setState(q, 3)
		case e.NPC == NPCPalaceGuard && e.Msg == 186:
			g.set(q, FlagCustom3, "a palace guard turned the hero away")
		case e.NPC == NPCPalaceGuard && e.Msg >= 187 && e.Msg <= 189:
			g.set(q, FlagCustom4, "a palace guard let the hero pass")
		case e.Msg >= 397 && e.Msg <= 407 && g.get(q, FlagRewardPending):
			g.clear(q, FlagRewardPending, "the sanctuary's reward heard")

			d.rewarded = true
		}
	}

	q.on[EvAreaChanged] = func(g *Game, q *Quest, e *Event) {
		switch {
		case e.NewLevel == LevelHarem1:
			g.set(q, FlagCustom3, "entered the harem")
			g.set(q, FlagCustom4, "entered the harem")
		case e.NewLevel == LevelArcaneSanctuary && q.State >= 2 && q.State < 5:
			g.setState(q, 4)
			g.set(q, FlagEnterArea, "entered the Arcane Sanctuary")
			g.set(q, FlagCustom1, "entered the Arcane Sanctuary")
		case e.OldLevel == LevelLutGholein && q.State == 3:
			g.setState(q, 4)
			g.set(q, FlagLeaveTown, "left Lut Gholein for the sanctuary")
		}
	}

	q.on[EvObjectOperated] = func(g *Game, q *Quest, e *Event) {
		if e.Object == ObjHorazonJournal && g.Level == LevelArcaneSanctuary {
			g.finishArcane("Horazon's journal was read")
		}
	}

	q.on[EvGameStarted] = func(g *Game, q *Quest, _ *Event) {
		switch {
		case g.get(q, FlagRewardGranted):
			q.State, d.rewarded = 0, true
		case g.get(q, FlagEnterArea):
			q.State = 4
		case g.get(q, FlagLeaveTown):
			q.State = 4
		case g.get(q, FlagStarted):
			q.State = 2
		}
	}

	return q
}

// finishArcane completes the Arcane Sanctuary quest (the journal was read, or the Summoner died
// with the hero in the sanctuary): RG, RP and the goal are set at once, then the progress bits
// are reset and CUSTOM3/4 stay set (the oddity of quests-2.md: the end speech has no marker).
func (g *Game) finishArcane(why string) {
	q := g.byID[QuestArcane]
	if q == nil || g.get(q, FlagRewardGranted) {
		return
	}

	g.setState(q, 5)
	g.set(q, FlagPrimaryGoal, why)
	g.set(q, FlagRewardPending, why)
	g.set(q, FlagRewardGranted, why)
	g.globalDone(q)
	g.resetProgress(q, why)
	g.set(q, FlagRewardGranted, why)
	g.set(q, FlagRewardPending, why)
	g.set(q, FlagCustom3, why)
	g.set(q, FlagCustom4, why)
	g.emit(Effect{Kind: EffectPortal, Quest: q.ID, Note: "permanent town portal to the Canyon of the Magi"})
	g.logPage(q, 5)

	if t := g.byID[QuestSevenTombs]; t != nil && t.State == 0 {
		g.setState(t, 1)
	}
}

// ---- A2Q5 The Summoner ----

type summonerData struct{ rewarded bool }

func newSummoner() *Quest {
	d := &summonerData{}
	q := &Quest{ID: QuestSummoner, Slot: 13, Act: 1, Name: "The Summoner", Label: "A2Q5", LogIndex: 5,
		Active: true, NotIntro: true, SeqID: -1, tables: speechTables("A2Q5"), data: d}

	q.activate = func(g *Game, q *Quest, npc int) []Speech {
		switch {
		case g.get(q, FlagRewardPending):
			return q.pick(npc, 1)
		case d.rewarded:
			return q.pick(npc, 2)
		case q.State == 1 && !g.get(q, FlagRewardGranted):
			return q.pick(npc, 0)
		}

		return nil
	}

	q.active = func(g *Game, q *Quest, npc int) bool {
		return g.get(q, FlagRewardPending) && npc != NPCGreiz && len(q.pick(npc, 1)) > 0
	}

	q.on[EvAreaChanged] = func(g *Game, q *Quest, e *Event) {
		// the original starts the quest when the Summoner's AI first ticks; entering his
		// level is the nearest event (UNVERIFIED)
		if e.NewLevel == LevelArcaneSanctuary && q.State == 0 && !g.grantedOrPending(q) {
			g.setState(q, 1)
			g.set(q, FlagStarted, "the Summoner's lair was entered")
		}
	}

	q.on[EvMonsterKilled] = func(g *Game, q *Quest, e *Event) {
		if e.Monster != NPCSummoner && !nameIs(e, "the summoner") && !nameIs(e, "summoner") {
			return
		}

		if g.get(q, FlagRewardGranted) || g.get(q, FlagRewardPending) {
			return
		}

		g.setState(q, 2)
		g.set(q, FlagPrimaryGoal, "the Summoner died")
		g.set(q, FlagRewardPending, "the Summoner died")
		g.globalDone(q)
		g.emit(Effect{Kind: EffectSound, Quest: q.ID, Value: 51, Note: "sound id unverified"})
		g.logPage(q, 2)

		if g.Level == LevelArcaneSanctuary || e.Level == LevelArcaneSanctuary {
			g.finishArcane("the Summoner died in the sanctuary")
		}
	}

	q.on[EvMessageHeard] = func(g *Game, q *Quest, e *Event) {
		if e.Msg >= 419 && e.Msg <= 429 && g.get(q, FlagRewardPending) {
			g.set(q, FlagRewardGranted, "the Summoner's reward heard")
			g.clear(q, FlagRewardPending, "the Summoner's reward heard")
			g.setState(q, 3)
			g.globalDone(q)

			d.rewarded = true
		}
	}

	q.on[EvGameStarted] = func(g *Game, q *Quest, _ *Event) {
		switch {
		case g.get(q, FlagRewardGranted):
			q.State, d.rewarded = 0, true
		case g.get(q, FlagStarted):
			q.State = 1
		}
	}

	return q
}

// ---- A2Q6 The Seven Tombs: speech and the Jerhyn/Meshif chain on top of the Duriel kill ----

// custom bits of the NPCs that comment on Duriel's death (quests-2.md A2Q6)
var tombsNPCBits = map[int]int{NPCAtma: 6, NPCWarriv2: 7, NPCDrognan: 8, NPCLysander: 9, NPCCain2: 10, NPCFara: 11}

type tombsData struct{ tyraelHeard bool }

func newSevenTombs() *Quest {
	d := &tombsData{}
	q := newBossQuest(bossQuests[0])
	q.Name, q.LogIndex, q.tables, q.data, q.SeqID = "The Seven Tombs", 6, speechTables("A2Q6"), d, -1
	q.InitNo = 0
	kill := q.on[EvMonsterKilled]

	q.on[EvMonsterKilled] = func(g *Game, q *Quest, e *Event) {
		was := q.State
		kill(g, q, e)

		if e.Monster == NPCDuriel || nameIs(e, "duriel") {
			if was < 3 {
				g.setState(q, 3)
			}

			g.after(8, func() { g.logPage(q, 3) })
		}
	}

	q.activate = func(g *Game, q *Quest, npc int) []Speech {
		if g.get(q, FlagRewardGranted) {
			return nil
		}

		switch {
		case npc == NPCTyrael1 && g.Level == LevelDurielLair && q.State >= 3 && !d.tyraelHeard:
			return q.pick(npc, 2)
		case d.tyraelHeard && tombsNPCBits[npc] != 0 && !g.get(q, tombsNPCBits[npc]) && len(q.pick(npc, 6)) > 0:
			return q.pick(npc, 6)
		case g.get(q, FlagLeaveTown) && d.tyraelHeard:
			return q.pick(npc, 3)
		case g.get(q, FlagEnterArea) && npc == NPCMeshif1:
			return q.pick(npc, 5)
		case q.State >= 1 && q.State <= 5 && q.State != 3:
			return q.pick(npc, q.State-1)
		}

		return nil
	}

	q.active = func(g *Game, q *Quest, npc int) bool {
		if g.get(q, FlagRewardGranted) {
			return false
		}

		for _, s := range q.activate(g, q, npc) {
			if s.Mode == ModeSpoken {
				return true
			}
		}

		return false
	}

	q.on[EvMessageHeard] = func(g *Game, q *Quest, e *Event) {
		switch {
		case e.NPC == NPCJerhyn && e.Msg == 430 && q.State == 1:
			g.setState(q, 2)
			g.set(q, FlagStarted, "Jerhyn sent the hero to the tombs")
		case e.NPC == NPCTyrael1 && e.Msg == 302:
			d.tyraelHeard = true

			g.setState(q, 4)
			g.set(q, FlagPrimaryGoal, "Tyrael spoke")
			g.set(q, FlagLeaveTown, "Tyrael spoke")
			g.emit(Effect{Kind: EffectPortal, Quest: q.ID, Note: "Tyrael's portal to Lut Gholein"})
		case e.NPC == NPCJerhyn && e.Msg == 442 && g.get(q, FlagLeaveTown):
			g.setState(q, 5)
			g.set(q, FlagEnterArea, "Jerhyn heard the tale")
			g.clear(q, FlagLeaveTown, "Jerhyn heard the tale")
		case e.NPC == NPCMeshif1 && e.Msg == 450 && g.get(q, FlagEnterArea):
			g.set(q, FlagRewardGranted, "Meshif offered the trip east")
			g.clear(q, FlagEnterArea, "Meshif offered the trip east")
			g.clear(q, FlagRewardPending, "Meshif offered the trip east")
			g.globalDone(q)
			g.finishStaff()
		case tombsNPCBits[e.NPC] != 0 && d.tyraelHeard:
			if line := map[int]int{NPCAtma: 445, NPCWarriv2: 446, NPCDrognan: 449, NPCLysander: 444, NPCCain2: 452, NPCFara: 447}[e.NPC]; e.Msg == line {
				g.set(q, tombsNPCBits[e.NPC], "an NPC commented on Duriel's death")
			}
		}
	}

	q.on[EvAreaChanged] = func(g *Game, q *Quest, e *Event) {
		switch {
		case q.State == 1 && e.NewLevel == LevelCanyon:
			// the original also starts the quest when the Canyon (or the real tomb or Duriel's lair) is entered
			g.setState(q, 2)
			g.set(q, FlagStarted, "the Canyon of the Magi was entered")
		case q.State == 2 && e.NewLevel == LevelDurielLair:
			g.set(q, FlagStarted, "Duriel's lair was entered")
		}
	}

	q.on[EvObjectOperated] = func(g *Game, q *Quest, e *Event) {
		if e.Object == ObjStaffOrifice && g.hasItem(ItemHoradricStaff) {
			g.finishStaff()
			g.emit(Effect{Kind: EffectPortal, Quest: q.ID, Note: "the orifice opens the portal to Duriel's lair"})
		}
	}

	q.on[EvGameStarted] = func(g *Game, q *Quest, _ *Event) {
		switch {
		case g.get(q, FlagRewardGranted):
			q.State = 0
		case g.get(q, FlagEnterArea):
			q.State = 5
		case g.get(q, FlagLeaveTown):
			q.State, d.tyraelHeard = 5, true
		case g.get(q, FlagCustom1):
			q.State = 3
		case g.get(q, FlagStarted):
			q.State = 2
		}
	}

	return q
}

// TravelToAct3 is Meshif taking the hero to Act 3 (QUESTS_OnActChangeNpcTravel, slot 15): the
// Horadric Staff quest is completed if it was not and the Act 2 finished word is set.
func (g *Game) TravelToAct3() []Effect {
	g.finishStaff()

	g.Rec.Set(15, FlagRewardGranted)
	g.Rec.Set(15, FlagPrimaryGoal)
	g.tracef("QUEST act2 finished slot=15")
	g.emit(Effect{Kind: EffectUnlockAct, Value: 3})

	return g.TakeEffects()
}

// logPage stores the quest log page and pushes it to the client.
func (g *Game) logPage(q *Quest, page int) {
	q.LastState = page
	g.emit(Effect{Kind: EffectLogUpdate, Quest: q.ID, Value: page})
}

// Offer makes a quest available (state 1) when it is dormant; scripted runs use it
// to start a quest without its predecessors.
func (g *Game) Offer(id int) {
	if q := g.byID[id]; q != nil && q.State == 0 && !g.get(q, FlagRewardGranted) {
		g.setState(q, 1)
	}
}
