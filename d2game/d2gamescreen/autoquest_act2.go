package d2gamescreen

import "github.com/OpenDiablo2/OpenDiablo2/d2common/d2quest"

// stageAct2 adds the scripted stages of the later Act 2 quests (OD2_AUTOQUEST=sun|staff|arcane|summoner|tombs|act2).
// Item pick-ups, the cube recipe and the object clicks are injected the way the engine reports them.
func (a *autoQuest) stageAct2(name string) {
	switch name {
	case "sun":
		a.stageSun()
	case "staff":
		a.stageStaff()
	case "arcane":
		a.stageArcane()
	case "summoner":
		a.stageSummoner()
	case "tombs":
		a.stageTombs()
	}
}

func (a *autoQuest) stageSun() {
	const id = d2quest.QuestTaintedSun

	a.once("sun: the far desert darkens the sun", func(h autoHost, a *autoQuest) {
		a.move(h, d2quest.LevelLutGholein)
		a.expect(h, "the Tainted Sun is dormant", a.quest(h, id).State == 0)
		a.move(h, d2quest.LevelLostCity)
		a.tick(h, 20)
		a.state(h, id, "sun dark")
		a.expect(h, "the sun is dark (state 1, started)", a.quest(h, id).State == 1 && a.bit(h, id, d2quest.FlagStarted))
		a.move(h, d2quest.LevelLutGholein)
	})
	a.once("sun: Drognan", func(h autoHost, a *autoQuest) {
		a.expect(h, "Drognan shows the quest marker", h.Q().Marker(d2quest.NPCDrognan))
		msgs := a.talk(h, d2quest.NPCDrognan)
		a.state(h, id, "after Drognan")
		a.expect(h, "Drognan says message 348 (state 2)", contains(msgs, 348) && a.quest(h, id).State == 2)
		a.expect(h, "the other NPCs offer their topics", len(a.topics(h, d2quest.NPCWarriv2)) > 0)
		a.move(h, d2quest.LevelValleyOfSnakes)
		a.expect(h, "state 3 after leaving town", a.quest(h, id).State == 3)
	})
	a.once("sun: the altar in the Claw Viper Temple", func(h autoHost, a *autoQuest) {
		a.move(h, d2quest.LevelClawViper2)
		a.object(h, d2quest.ObjTaintedSunAltar)
		a.state(h, id, "altar destroyed")
		a.expect(h, "reward pending + goal, state 4", a.bit(h, id, d2quest.FlagRewardPending) &&
			a.bit(h, id, d2quest.FlagPrimaryGoal) && a.quest(h, id).State == 4)
		a.move(h, d2quest.LevelLutGholein)
	})
	a.once("sun: the reward speech", func(h autoHost, a *autoQuest) {
		msgs := a.talk(h, d2quest.NPCJerhyn)
		a.state(h, id, "reward claimed")
		a.expect(h, "Jerhyn says message 362", contains(msgs, 362))
		a.expect(h, "reward granted, state 5", a.bit(h, id, d2quest.FlagRewardGranted) && a.quest(h, id).State == 5)
		a.expect(h, "the Arcane Sanctuary quest is offered", a.quest(h, d2quest.QuestArcane).State == 1)
	})
}

func (a *autoQuest) stageStaff() {
	const id = d2quest.QuestStaff

	a.once("staff: the scroll, the amulet, the staff and the cube", func(h autoHost, a *autoQuest) {
		a.move(h, d2quest.LevelLutGholein)
		a.pickup(h, "tr1")
		msgs := a.talk(h, d2quest.NPCCain2)
		a.expect(h, "Cain reads the scroll (335)", contains(msgs, 335) && a.bit(h, id, d2quest.FlagLeaveTown))
		a.pickup(h, "vip")
		msgs = a.talk(h, d2quest.NPCCain2)
		a.expect(h, "Cain hears of the amulet (336)", contains(msgs, 336) && a.bit(h, id, d2quest.FlagEnterArea))
		a.pickup(h, "msf")
		msgs = a.talk(h, d2quest.NPCCain2)
		a.expect(h, "Cain hears of the staff (337)", contains(msgs, 337) && a.bit(h, id, d2quest.FlagCustom1))
		a.pickup(h, "box")
		msgs = a.talk(h, d2quest.NPCCain2)
		a.expect(h, "Cain hears of the cube (338)", contains(msgs, 338) && a.bit(h, id, d2quest.FlagCustom2))
	})
	a.once("staff: the cube makes the Horadric Staff", func(h autoHost, a *autoQuest) {
		h.Apply(h.Q().CubeHoradricStaff())
		a.expect(h, "the cube recipe consumed the parts", h.Q().Items["hst"] == 1 && h.Q().Items["msf"] == 0)
		msgs := a.talk(h, d2quest.NPCCain2)
		a.state(h, id, "staff assembled")
		a.expect(h, "Cain explains the assembled staff (339)", contains(msgs, 339) && a.bit(h, id, d2quest.FlagCustom6))
	})
}

func (a *autoQuest) stageArcane() {
	const id = d2quest.QuestArcane

	a.once("arcane: Drognan and Jerhyn", func(h autoHost, a *autoQuest) {
		a.move(h, d2quest.LevelLutGholein)
		h.Q().Offer(id) // offered by the Tainted Sun reward or the cube; offered here when the script runs alone
		a.expect(h, "a palace guard turns the hero away before Drognan", contains(a.talk(h, d2quest.NPCPalaceGuard), 186))
		msgs := a.talk(h, d2quest.NPCDrognan)
		a.expect(h, "Drognan says message 373", contains(msgs, 373) && a.quest(h, id).State == 2)
		msgs = a.talk(h, d2quest.NPCJerhyn)
		a.expect(h, "Jerhyn says message 377", contains(msgs, 377) && a.quest(h, id).State == 3)
	})
	a.once("arcane: the harem, the sanctuary and the journal", func(h autoHost, a *autoQuest) {
		a.move(h, d2quest.LevelHarem1)
		a.move(h, d2quest.LevelPalaceCellar3)
		a.move(h, d2quest.LevelArcaneSanctuary)
		a.state(h, id, "in the sanctuary")
		a.expect(h, "state 4 with the enter-area bit", a.quest(h, id).State == 4 && a.bit(h, id, d2quest.FlagEnterArea))
		a.object(h, d2quest.ObjHorazonJournal)
		a.state(h, id, "journal read")
		a.expect(h, "completed with the reward pending", a.bit(h, id, d2quest.FlagRewardGranted) &&
			a.bit(h, id, d2quest.FlagRewardPending) && a.quest(h, id).State == 5)
		a.expect(h, "the Seven Tombs are offered", a.quest(h, d2quest.QuestSevenTombs).State == 1)
		a.move(h, d2quest.LevelLutGholein)
	})
}

func (a *autoQuest) stageSummoner() {
	const id = d2quest.QuestSummoner

	a.once("summoner: the kill and the town's reaction", func(h autoHost, a *autoQuest) {
		a.move(h, d2quest.LevelArcaneSanctuary)
		a.expect(h, "the Summoner quest starts in his sanctuary", a.quest(h, id).State == 1)
		a.kill(h, d2quest.Event{Monster: d2quest.NPCSummoner})
		a.state(h, id, "the Summoner died")
		a.expect(h, "reward pending + goal, state 2", a.bit(h, id, d2quest.FlagRewardPending) &&
			a.bit(h, id, d2quest.FlagPrimaryGoal) && a.quest(h, id).State == 2)
		a.move(h, d2quest.LevelLutGholein)
		a.expect(h, "Atma shows the quest marker", h.Q().Marker(d2quest.NPCAtma))
		msgs := a.talk(h, d2quest.NPCGreiz)
		a.state(h, id, "reward claimed")
		a.expect(h, "Greiz says message 419", contains(msgs, 419))
		a.expect(h, "reward granted, state 3", a.bit(h, id, d2quest.FlagRewardGranted) && a.quest(h, id).State == 3)
	})
}

func (a *autoQuest) stageTombs() {
	const id = d2quest.QuestSevenTombs

	a.once("tombs: Jerhyn sends the hero to the tombs, the staff goes into the orifice", func(h autoHost, a *autoQuest) {
		a.move(h, d2quest.LevelLutGholein)

		h.Q().Offer(id) // the sanctuary quests offer it; a script that runs alone starts it by hand

		msgs := a.talk(h, d2quest.NPCJerhyn)
		a.expect(h, "Jerhyn says message 430 (state 2)", contains(msgs, 430) && a.quest(h, id).State == 2)
		h.Q().Items["hst"] = 1
		a.move(h, d2quest.LevelFirstTomb+1)
		a.object(h, d2quest.ObjStaffOrifice)
		a.expect(h, "the Horadric Staff quest finished with the orifice", a.bit(h, d2quest.QuestStaff, d2quest.FlagRewardGranted))
	})
	a.once("tombs: Duriel, Tyrael and the way east", func(h autoHost, a *autoQuest) {
		a.move(h, d2quest.LevelDurielLair)
		a.kill(h, d2quest.Event{Monster: d2quest.NPCDuriel, Name: "Duriel"})
		a.tick(h, 12)
		a.state(h, id, "Duriel dead")
		a.expect(h, "Duriel's kill sets bit 5, state 3", a.bit(h, id, d2quest.FlagCustom1) && a.quest(h, id).State == 3)
		msgs := a.talk(h, d2quest.NPCTyrael1)
		a.expect(h, "Tyrael says message 302 (state 4)", contains(msgs, 302) && a.quest(h, id).State == 4)
		a.move(h, d2quest.LevelLutGholein)

		var all []int
		for _, npc := range []int{d2quest.NPCAtma, d2quest.NPCWarriv2, d2quest.NPCDrognan, d2quest.NPCLysander, d2quest.NPCCain2, d2quest.NPCFara} {
			all = append(all, a.talk(h, npc)...)
		}

		a.expect(h, "Atma, Warriv, Drognan, Lysander, Cain and Fara comment (445 446 449 444 452 447)",
			contains(all, 445) && contains(all, 446) && contains(all, 449) && contains(all, 444) && contains(all, 452) && contains(all, 447))
		a.expect(h, "Jerhyn says message 442 (state 5)", contains(a.talk(h, d2quest.NPCJerhyn), 442) && a.quest(h, id).State == 5)
		a.expect(h, "Meshif says message 450", contains(a.talk(h, d2quest.NPCMeshif1), 450))
		a.state(h, id, "Meshif")
		a.expect(h, "The Seven Tombs completed", a.bit(h, id, d2quest.FlagRewardGranted))
		h.Apply(h.Q().TravelToAct3())
		a.expect(h, "Act 2 finished word (slot 15) set", h.Q().Rec.ActFinished(2))
	})
}
