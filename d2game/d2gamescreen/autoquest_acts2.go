package d2gamescreen

import "github.com/OpenDiablo2/OpenDiablo2/d2common/d2quest"

// Act 2 (continued).

func (a *autoQuest) stageSun() {
	const id = d2quest.QuestTaintedSun

	a.once("sun: the sun darkens", func(h autoHost, a *autoQuest) {
		a.move(h, d2quest.LevelLutGholein)
		a.expect(h, "the Tainted Sun is dormant", a.quest(h, id).State == 0)
		a.move(h, d2quest.LevelLostCity)
		a.move(h, d2quest.LevelLutGholein)
		a.state(h, id, "sun darkened")
		a.expect(h, "the quest is available (state 1)", a.quest(h, id).State == 1)
	})
	a.once("sun: Drognan and the Lost City", func(h autoHost, a *autoQuest) {
		a.says(h, "Drognan starts the quest", d2quest.NPCDrognan, 348)
		a.expect(h, "STARTED bit set, state 2", a.bit(h, id, d2quest.FlagStarted) && a.quest(h, id).State == 2)
		a.move(h, d2quest.LevelLostCity)
		a.expect(h, "LEAVETOWN bit set, state 3", a.bit(h, id, d2quest.FlagLeaveTown) && a.quest(h, id).State == 3)
	})
	a.once("sun: the altar", func(h autoHost, a *autoQuest) {
		h.Dispatch(d2quest.Event{Kind: d2quest.EvObjectOperated, Object: d2quest.ObjectTaintedSunAltar, Level: 61})
		a.state(h, id, "altar destroyed")
		a.expect(h, "reward pending + primary goal", a.bit(h, id, d2quest.FlagRewardPending) &&
			a.bit(h, id, d2quest.FlagPrimaryGoal))
		a.pickup(h, d2quest.ItemViperAmulet)
		a.move(h, d2quest.LevelLutGholein)
	})
	a.once("sun: Drognan's thanks", func(h autoHost, a *autoQuest) {
		a.expect(h, "Drognan shows the quest marker", h.Q().Marker(d2quest.NPCDrognan))
		a.says(h, "Drognan's reward line", d2quest.NPCDrognan, 371)
		a.state(h, id, "reward claimed")
		a.expect(h, "reward granted", a.bit(h, id, d2quest.FlagRewardGranted))
		a.expect(h, "the Arcane Sanctuary quest opened", a.quest(h, d2quest.QuestArcane).State >= 1)
	})
}

func (a *autoQuest) stageArcane() {
	const id = d2quest.QuestArcane

	a.once("arcane: Drognan and Jerhyn", func(h autoHost, a *autoQuest) {
		a.move(h, d2quest.LevelLutGholein)
		a.offer(h, id, d2quest.NPCDrognan, 373)

		if a.quest(h, id).State == 2 {
			a.says(h, "Jerhyn sends the hero in", d2quest.NPCJerhyn, 377)
		}

		a.expect(h, "state 3", a.quest(h, id).State == 3)

		open := false

		for _, s := range h.Q().Activate(d2quest.NPCGuard2).Lines {
			open = open || (s.Msg >= 187 && s.Msg <= 189)
		}

		a.expect(h, "the palace guard speaks an open line (187-189)", open)
	})
	a.once("arcane: the sanctuary and the journal", func(h autoHost, a *autoQuest) {
		a.move(h, d2quest.LevelArcane)
		a.state(h, id, "entered the sanctuary")
		a.expect(h, "CUSTOM1 set, state 4", a.bit(h, id, d2quest.FlagCustom1) && a.quest(h, id).State == 4)
		h.Dispatch(d2quest.Event{Kind: d2quest.EvObjectOperated, Object: d2quest.ObjectHorazonJournal, Level: d2quest.LevelArcane})
		a.expect(h, "reward pending + primary goal", a.bit(h, id, d2quest.FlagRewardPending) &&
			a.bit(h, id, d2quest.FlagPrimaryGoal))
		a.move(h, d2quest.LevelLutGholein)
	})
	a.once("arcane: the town congratulates", func(h autoHost, a *autoQuest) {
		a.says(h, "Jerhyn's end line", d2quest.NPCJerhyn, 398)
		a.state(h, id, "reward claimed")
		a.expect(h, "reward granted", a.bit(h, id, d2quest.FlagRewardGranted))
	})
}

func (a *autoQuest) stageSummoner() {
	const id = d2quest.QuestSummoner

	a.once("summoner: the lair", func(h autoHost, a *autoQuest) {
		a.move(h, d2quest.LevelArcane)
		a.expect(h, "STARTED bit set when the lair is entered", a.bit(h, id, d2quest.FlagStarted) && a.quest(h, id).State == 1)
		a.kill(h, d2quest.Event{Monster: d2quest.NPCSummoner})
		a.state(h, id, "summoner dead")
		a.expect(h, "reward pending + primary goal", a.bit(h, id, d2quest.FlagRewardPending) &&
			a.bit(h, id, d2quest.FlagPrimaryGoal))
		a.expect(h, "the Arcane Sanctuary quest completes too", a.bit(h, d2quest.QuestArcane, d2quest.FlagPrimaryGoal))
		a.move(h, d2quest.LevelLutGholein)
	})
	a.once("summoner: Atma's thanks", func(h autoHost, a *autoQuest) {
		a.says(h, "Atma's reward line", d2quest.NPCAtma, 427)
		a.state(h, id, "reward claimed")
		a.expect(h, "reward granted", a.bit(h, id, d2quest.FlagRewardGranted))
	})
}

func (a *autoQuest) stageTombs() {
	const id = d2quest.QuestSevenTombs

	a.once("tombs: Jerhyn", func(h autoHost, a *autoQuest) {
		a.move(h, d2quest.LevelLutGholein)
		a.offer(h, id, d2quest.NPCJerhyn, 430)
		a.expect(h, "state 2", a.quest(h, id).State == 2)
	})
	a.once("tombs: Duriel", func(h autoHost, a *autoQuest) {
		a.move(h, d2quest.LevelDurielLair)
		a.kill(h, d2quest.Event{Monster: d2quest.NPCDuriel})
		a.state(h, id, "Duriel dead")
		a.expect(h, "CUSTOM1 set, state 3", a.bit(h, id, d2quest.FlagCustom1) && a.quest(h, id).State == 3)
		h.Dispatch(d2quest.Event{Kind: d2quest.EvMessageHeard, NPC: d2quest.NPCTyrael1, Msg: 302})
		a.expect(h, "Tyrael's message: primary goal + LEAVETOWN, state 4", a.bitsOf(h, id, d2quest.FlagPrimaryGoal,
			d2quest.FlagLeaveTown) && a.quest(h, id).State == 4)
		a.move(h, d2quest.LevelLutGholein)
	})
	a.once("tombs: the town and Meshif", func(h autoHost, a *autoQuest) {
		a.says(h, "Atma comments", d2quest.NPCAtma, 445)
		a.says(h, "Jerhyn's thanks", d2quest.NPCJerhyn, 442)
		a.expect(h, "ENTERAREA set, LEAVETOWN cleared, state 5", a.bit(h, id, d2quest.FlagEnterArea) &&
			!a.bit(h, id, d2quest.FlagLeaveTown) && a.quest(h, id).State == 5)
		a.says(h, "Meshif's ship", d2quest.NPCMeshif1, 450)
		a.state(h, id, "boarded")
		a.expect(h, "reward granted", a.bit(h, id, d2quest.FlagRewardGranted))
		h.Apply(h.Q().TravelToAct3())
		a.expect(h, "the Act 2 finished word (slot 15) is set", h.Q().Rec.Get(15, d2quest.FlagRewardGranted))
	})
}

// ---- Act 3 ----

func (a *autoQuest) stageLamEsen() {
	const id = d2quest.QuestLamEsen

	a.once("lamesen: Alkor", func(h autoHost, a *autoQuest) {
		a.move(h, d2quest.LevelKurastDocktown)
		a.says(h, "Alkor offers the quest", d2quest.NPCAlkor, 549)
		a.expect(h, "STARTED bit, state 2", a.bit(h, id, d2quest.FlagStarted) && a.quest(h, id).State == 2)
		a.move(h, d2quest.LevelRuinedTemple)
		a.expect(h, "ENTERAREA bit, state 3", a.bit(h, id, d2quest.FlagEnterArea) && a.quest(h, id).State == 3)
	})
	a.once("lamesen: the tome", func(h autoHost, a *autoQuest) {
		a.pickup(h, d2quest.ItemLamEsenTome)
		a.state(h, id, "tome picked up")
		a.expect(h, "reward pending + primary goal", a.bit(h, id, d2quest.FlagRewardPending) &&
			a.bit(h, id, d2quest.FlagPrimaryGoal))
		a.move(h, d2quest.LevelKurastDocktown)
		a.says(h, "Alkor's thanks", d2quest.NPCAlkor, 564)
		a.state(h, id, "reward claimed")
		a.expect(h, "reward granted, the tome is gone", a.bit(h, id, d2quest.FlagRewardGranted) &&
			h.Q().Items[d2quest.ItemLamEsenTome] == 0)
	})
}

func (a *autoQuest) stageKhalim() {
	const id = d2quest.QuestKhalim

	a.once("khalim: Cain and the parts", func(h autoHost, a *autoQuest) {
		a.move(h, d2quest.LevelKurastDocktown)
		a.says(h, "Cain explains the Will", d2quest.NPCCain3, 543)
		a.expect(h, "STARTED bit set", a.bit(h, id, d2quest.FlagStarted))

		for _, p := range []struct {
			item string
			msg  int
			bit  int
		}{
			{d2quest.ItemKhalimEye, 545, d2quest.FlagCustom1}, {d2quest.ItemKhalimHeart, 544, d2quest.FlagCustom1 + 1},
			{d2quest.ItemKhalimBrain, 546, d2quest.FlagCustom1 + 2}, {d2quest.ItemKhalimWill, 547, d2quest.FlagCustom1 + 3},
		} {
			a.pickup(h, p.item)
			a.says(h, "Cain reports "+p.item, d2quest.NPCCain3, p.msg)
			a.expect(h, "bit for "+p.item, a.bit(h, id, p.bit))
		}
	})
	a.once("khalim: the Compelling Orb", func(h autoHost, a *autoQuest) {
		a.move(h, d2quest.LevelTravincal)
		a.object(h, d2quest.ObjectCompellingOrb)
		a.state(h, id, "orb smashed")
		a.expect(h, "reward pending + primary goal", a.bit(h, id, d2quest.FlagRewardPending) &&
			a.bit(h, id, d2quest.FlagPrimaryGoal))
		a.move(h, d2quest.LevelKurastDocktown)
		a.says(h, "Cain's last line", d2quest.NPCCain3, 548)
		a.expect(h, "reward granted", a.bit(h, id, d2quest.FlagRewardGranted))
	})
}

func (a *autoQuest) stageBlade() {
	const id = d2quest.QuestBlade

	a.once("blade: Hratli and the Flayer Dungeon", func(h autoHost, a *autoQuest) {
		a.move(h, d2quest.LevelKurastDocktown)
		a.says(h, "Hratli offers the quest", d2quest.NPCHratli, 571)
		a.move(h, d2quest.LevelFlayerDungeon1)
		a.expect(h, "ENTERAREA bit, state 3", a.bit(h, id, d2quest.FlagEnterArea) && a.quest(h, id).State == 3)
		a.pickup(h, d2quest.ItemGidbinn)
		a.expect(h, "Gidbinn found, state 4", a.quest(h, id).State == 4)
		a.move(h, d2quest.LevelKurastDocktown)
	})
	a.once("blade: Ormus, Asheara", func(h autoHost, a *autoQuest) {
		a.says(h, "Ormus takes the blade", d2quest.NPCOrmus, 587)
		a.says(h, "Asheara's line", d2quest.NPCAsheara, 589)
		a.state(h, id, "Asheara")
		a.expect(h, "reward pending + primary goal", a.bit(h, id, d2quest.FlagRewardPending) &&
			a.bit(h, id, d2quest.FlagPrimaryGoal))
		a.says(h, "Ormus pays out", d2quest.NPCOrmus, 593)
		a.expect(h, "reward granted", a.bit(h, id, d2quest.FlagRewardGranted))
		a.expect(h, "the Blackened Temple quest opened", a.quest(h, d2quest.QuestBlackenedTemple).State >= 1)
	})
}

func (a *autoQuest) stageBird() {
	const id = d2quest.QuestGoldenBird

	a.once("bird: figurine, Cain, Meshif", func(h autoHost, a *autoQuest) {
		a.move(h, d2quest.LevelKurastDocktown)
		a.expect(h, "the quest is dormant", a.quest(h, id).State == 0)
		a.pickup(h, d2quest.ItemJadeFigurine)
		a.says(h, "Cain looks at the figurine", d2quest.NPCCain3, 527)
		a.says(h, "Meshif trades the bird", d2quest.NPCMeshif2, 529)
		a.state(h, id, "bird in hand")
		a.expect(h, "STARTED bit, state 3", a.bit(h, id, d2quest.FlagStarted) && a.quest(h, id).State == 3)
	})
	a.once("bird: Alkor", func(h autoHost, a *autoQuest) {
		a.says(h, "Alkor takes the bird and pays out", d2quest.NPCAlkor, 534, 538)
		a.state(h, id, "reward claimed")
		a.expect(h, "reward granted", a.bit(h, id, d2quest.FlagRewardGranted))
	})
}

func (a *autoQuest) stageTemple() {
	const id = d2quest.QuestBlackenedTemple

	a.once("temple: Ormus and Travincal", func(h autoHost, a *autoQuest) {
		a.move(h, d2quest.LevelKurastDocktown)
		a.offer(h, id, d2quest.NPCOrmus, 594)
		a.move(h, d2quest.LevelTravincal)
		a.expect(h, "ENTERAREA bit, state 3", a.bit(h, id, d2quest.FlagEnterArea) && a.quest(h, id).State == 3)

		for i, c := range []int{d2quest.NPCCouncilA, d2quest.NPCCouncilB, d2quest.NPCCouncilC} {
			a.kill(h, d2quest.Event{Monster: c})
			a.expect(h, "council member killed", (i < 2) != a.bit(h, id, d2quest.FlagRewardPending))
		}

		a.move(h, d2quest.LevelKurastDocktown)
		a.says(h, "Cain's line", d2quest.NPCCain3, 626)
		a.state(h, id, "reward claimed")
		a.expect(h, "reward granted; the Guardian opened", a.bit(h, id, d2quest.FlagRewardGranted) &&
			a.quest(h, d2quest.QuestGuardian).State >= 1)
	})
}

func (a *autoQuest) stageGuardian() {
	const id = d2quest.QuestGuardian

	a.once("guardian: the Durance of Hate", func(h autoHost, a *autoQuest) {
		a.move(h, d2quest.LevelKurastDocktown)
		a.offer(h, id, d2quest.NPCOrmus, 628)
		a.move(h, d2quest.LevelDurance1)
		a.move(h, d2quest.LevelDurance3)
		a.expect(h, "CUSTOM1 set, state 4", a.bit(h, id, d2quest.FlagCustom1) && a.quest(h, id).State == 4)
		a.kill(h, d2quest.Event{Monster: d2quest.NPCMephisto})
		a.state(h, id, "Mephisto dead")
		a.expect(h, "reward pending + primary goal", a.bit(h, id, d2quest.FlagRewardPending) &&
			a.bit(h, id, d2quest.FlagPrimaryGoal))
		a.move(h, d2quest.LevelKurastDocktown)
		a.says(h, "the town cheers", d2quest.NPCAlkor, 657)
		a.expect(h, "reward granted", a.bit(h, id, d2quest.FlagRewardGranted))
		h.Apply(h.Q().TravelToAct4())
		a.expect(h, "the Act 3 finished word is set", h.Q().Rec.Get(23, d2quest.FlagRewardGranted))
	})
}

// ---- Act 4 ----

func (a *autoQuest) stageFallen() {
	const id = d2quest.QuestFallenAngel

	a.once("fallen: Tyrael, Izual", func(h autoHost, a *autoQuest) {
		a.move(h, d2quest.LevelPandemonium)
		a.says(h, "Tyrael's welcome and the quest", d2quest.NPCTyrael2, 664, 670)
		a.expect(h, "STARTED bit set", a.bit(h, id, d2quest.FlagStarted))
		a.move(h, d2quest.LevelPlainsDespair)
		a.expect(h, "ENTERAREA bit, state 3", a.bit(h, id, d2quest.FlagEnterArea) && a.quest(h, id).State == 3)
		a.kill(h, d2quest.Event{Monster: d2quest.NPCIzual})
		a.expect(h, "reward pending + primary goal", a.bit(h, id, d2quest.FlagRewardPending) &&
			a.bit(h, id, d2quest.FlagPrimaryGoal))
		a.move(h, d2quest.LevelPandemonium)
		before := h.SkillPoints()
		a.says(h, "Tyrael's thanks", d2quest.NPCTyrael2, 676)
		a.state(h, id, "reward claimed")
		a.expect(h, "reward granted, 2 skill points", a.bit(h, id, d2quest.FlagRewardGranted) && h.SkillPoints() == before+2)
	})
}

func (a *autoQuest) stageTerror() {
	const id = d2quest.QuestTerrorsEnd

	a.once("terror: Tyrael, Diablo", func(h autoHost, a *autoQuest) {
		a.move(h, d2quest.LevelPandemonium)
		a.offer(h, id, d2quest.NPCTyrael2, 681)
		a.move(h, d2quest.LevelChaosSanctum)
		a.expect(h, "ENTERAREA bit, state 3", a.bit(h, id, d2quest.FlagEnterArea) && a.quest(h, id).State == 3)
		a.kill(h, d2quest.Event{Monster: d2quest.NPCDiablo})
		a.expect(h, "reward pending + primary goal", a.bit(h, id, d2quest.FlagRewardPending) &&
			a.bit(h, id, d2quest.FlagPrimaryGoal))
		a.move(h, d2quest.LevelPandemonium)
		a.says(h, "Tyrael's expansion line", d2quest.NPCTyrael2, 20000)
		a.expect(h, "reward granted", a.bit(h, id, d2quest.FlagRewardGranted))
		h.Apply(h.Q().TravelToAct5())
		a.expect(h, "the Act 4 finished word (slot 28) is set", h.Q().Rec.Get(28, d2quest.FlagRewardGranted))
	})
}

func (a *autoQuest) stageHellforge() {
	const id = d2quest.QuestHellforge

	a.once("hellforge: Cain and the forge", func(h autoHost, a *autoQuest) {
		a.move(h, d2quest.LevelPandemonium)
		a.pickup(h, d2quest.ItemMephistoSoulstone)
		a.says(h, "Cain talks about the soulstone", d2quest.NPCCain4, 678)
		a.pickup(h, d2quest.ItemHellforgeHammer)
		a.move(h, d2quest.LevelRiverOfFlame)
		a.object(h, d2quest.ObjectHellforge)
		a.state(h, id, "forge used")
		a.expect(h, "reward pending + primary goal", a.bit(h, id, d2quest.FlagRewardPending) &&
			a.bit(h, id, d2quest.FlagPrimaryGoal))
		a.move(h, d2quest.LevelPandemonium)
		a.says(h, "Cain's last line", d2quest.NPCCain4, 680)
		a.expect(h, "reward granted", a.bit(h, id, d2quest.FlagRewardGranted))
	})
}

// ---- Act 5 ----

func (a *autoQuest) stagePrison() {
	const id = d2quest.QuestPrison

	a.once("prison: Malah and the frozen Anya", func(h autoHost, a *autoQuest) {
		a.move(h, d2quest.LevelHarrogath)
		a.says(h, "Malah offers the quest", d2quest.NPCMalah, 20116)
		a.move(h, d2quest.LevelFrozenRiver)
		a.expect(h, "state 4 in the Frozen River", a.quest(h, id).State == 4)
		a.says(h, "the frozen Anya speaks", d2quest.NPCAnyaFrozen, 20131)
		a.expect(h, "Malah's scroll handed out, state 5", a.quest(h, id).State == 5)
	})
	a.once("prison: the scroll", func(h autoHost, a *autoQuest) {
		a.pickup(h, d2quest.ItemMalahScroll)
		h.Dispatch(d2quest.Event{Kind: d2quest.EvItemRemoved, Item: d2quest.ItemMalahScroll})
		a.state(h, id, "scroll read")
		a.expect(h, "reward pending + primary goal", a.bit(h, id, d2quest.FlagRewardPending) &&
			a.bit(h, id, d2quest.FlagPrimaryGoal))
		a.move(h, d2quest.LevelHarrogath)
		a.says(h, "Malah's thanks", d2quest.NPCMalah, 20132)
		a.expect(h, "reward granted", a.bit(h, id, d2quest.FlagRewardGranted))
	})
}

func (a *autoQuest) stageEve() {
	const id = d2quest.QuestEve

	a.once("eve: the Worldstone Keep and Baal", func(h autoHost, a *autoQuest) {
		a.move(h, d2quest.LevelHarrogath)
		a.expect(h, "the quest is available after Rite of Passage", a.quest(h, id).State == 1)
		a.move(h, d2quest.LevelWorldstone1)
		a.move(h, d2quest.LevelThrone)
		a.expect(h, "STARTED bit set, state 2", a.bit(h, id, d2quest.FlagStarted) && a.quest(h, id).State == 2)
		a.kill(h, d2quest.Event{Monster: d2quest.NPCBaalCrab})
		a.state(h, id, "Baal dead")
		a.expect(h, "reward pending + primary goal", a.bit(h, id, d2quest.FlagRewardPending) &&
			a.bit(h, id, d2quest.FlagPrimaryGoal))
		a.move(h, d2quest.LevelHarrogath)
		a.says(h, "Tyrael's end line", d2quest.NPCTyrael3, 20175)
		a.expect(h, "reward granted", a.bit(h, id, d2quest.FlagRewardGranted))
	})
}
