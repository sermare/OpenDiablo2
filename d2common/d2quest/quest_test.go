package d2quest

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
)

func newGame(t *testing.T) (*Game, *d2s.Body) {
	t.Helper()

	var body d2s.Body

	g := New(body.QuestRecord(Normal), body.NPCFlags(), Normal)
	g.Hero = Hero{Class: ClassSorceress, Level: 10}
	g.Start()

	return g, &body
}

func hasEffect(effects []Effect, kind EffectKind) bool {
	for _, e := range effects {
		if e.Kind == kind {
			return true
		}
	}

	return false
}

func slot(g *Game, id int) uint16 { return g.Rec.Slot(g.Quest(id).Slot) }

// talk plays every spoken line of an NPC the way the client does and returns the messages.
func talk(g *Game, npc int) (msgs []int, effects []Effect) {
	for i := 0; i < 20; i++ {
		d := g.Activate(npc)

		s, ok := d.NextSpoken(g)
		if !ok {
			break
		}

		msgs = append(msgs, s.Msg)
		effects = append(effects, g.Hear(npc, s.Msg)...)
	}

	effects = append(effects, g.Close(npc)...)

	return msgs, effects
}

func moveTo(g *Game, from, to int) []Effect {
	return g.Dispatch(Event{Kind: EvAreaChanged, OldLevel: from, NewLevel: to})
}

func TestDenOfEvilFullLine(t *testing.T) {
	g, _ := newGame(t)
	den := g.Quest(QuestDenOfEvil)

	if den.State != 1 || slot(g, QuestDenOfEvil) != 0 {
		t.Fatalf("fresh game: %s", g.Describe(den))
	}

	// the first meeting: Akara's intro line, then the quest line
	msgs, _ := talk(g, NPCAkara)
	if len(msgs) != 2 || msgs[0] != 64 && msgs[1] != 64 {
		t.Fatalf("Akara lines %v, want the intro and 64", msgs)
	}

	if !g.QuestIntroDone(NPCAkara) {
		t.Error("Akara's intro flag not set")
	}

	if den.State != 2 || slot(g, QuestDenOfEvil) != 1<<FlagStarted {
		t.Fatalf("after Akara: %s", g.Describe(den))
	}

	if den.LastState != 1 {
		t.Errorf("closing the dialog should push log page 1, last=%d", den.LastState)
	}

	// everybody comments while the den is not cleared (topics)
	for _, npc := range []int{NPCKashya, NPCCharsi, NPCGheed, NPCWarriv1} {
		d := g.Activate(npc)
		if len(d.Topics()) != 1 {
			t.Errorf("npc %d: want one topic, got %+v", npc, d.Lines)
		}
	}

	moveTo(g, LevelRogueEncampment, 2) // Blood Moor
	if den.State != 3 || slot(g, QuestDenOfEvil) != 1<<FlagStarted|1<<FlagLeaveTown {
		t.Fatalf("left town: %s", g.Describe(den))
	}

	g.SetDenMonsters(8)
	moveTo(g, 2, LevelDenOfEvil)

	want := uint16(1<<FlagStarted | 1<<FlagLeaveTown | 1<<FlagEnterArea)
	if slot(g, QuestDenOfEvil) != want || den.LastState != 2 {
		t.Fatalf("entered den: %s", g.Describe(den))
	}

	for i := 0; i < 7; i++ {
		g.Dispatch(Event{Kind: EvMonsterKilled, Monster: 1, Level: LevelDenOfEvil})
	}

	if den.LastState != 4 || den.State != 3 {
		t.Fatalf("few monsters left should push page 4: %s", g.Describe(den))
	}

	// monsters elsewhere do not count
	g.Dispatch(Event{Kind: EvMonsterKilled, Monster: 1, Level: 2})

	if g.DenMonstersLeft() != 1 {
		t.Fatalf("monsters left %d", g.DenMonstersLeft())
	}

	g.Dispatch(Event{Kind: EvMonsterKilled, Monster: 1, Level: LevelDenOfEvil})

	if den.State != 4 || !g.get(den, FlagRewardPending) || !g.get(den, FlagPrimaryGoal) {
		t.Fatalf("den cleared: %s", g.Describe(den))
	}

	if g.Quest(QuestBurial).State != 0 {
		t.Error("the next quest must not be available before the reward is claimed")
	}

	g.Tick(8)

	if den.LastState != 5 {
		t.Errorf("page 5 after the timer, got %d", den.LastState)
	}

	moveTo(g, LevelDenOfEvil, 2)
	moveTo(g, 2, LevelRogueEncampment)

	// Akara's thanks
	d := g.Activate(NPCAkara)

	s, ok := d.NextSpoken(g)
	if !ok || s.Msg != 76 {
		t.Fatalf("Akara should say 76, got %+v", d.Lines)
	}

	if !g.Marker(NPCAkara) {
		t.Error("Akara should show the quest marker")
	}

	eff := g.Hear(NPCAkara, 76)

	if !hasEffect(eff, EffectSkillPoint) || !hasEffect(eff, EffectRespec) {
		t.Errorf("reward effects missing: %+v", eff)
	}

	if g.get(den, FlagRewardPending) || !g.get(den, FlagRewardGranted) {
		t.Fatalf("reward claimed: %s", g.Describe(den))
	}

	if slot(g, QuestDenOfEvil)&0x0FFC != 0 {
		t.Errorf("progress bits must be cleared: %#x", slot(g, QuestDenOfEvil))
	}

	if !g.Rec.AkaraRespecAvailable() {
		t.Error("slot 41 respec flag not set")
	}

	if g.Quest(QuestBurial).State != 1 {
		t.Error("Sisters' Burial Grounds should now be available")
	}

	// done: nothing more to say, no marker
	if g.Marker(NPCAkara) {
		t.Error("marker should be gone")
	}
}

func TestDenOfEvilRewardTableAfterReload(t *testing.T) {
	// a save with reward pending (bit 1): bit 15 is derived on load
	var body d2s.Body

	rec := body.QuestRecord(Normal)
	rec.Set(1, FlagRewardPending)
	rec.Set(1, FlagPrimaryGoal) // transient: dropped on load

	g := New(rec, body.NPCFlags(), Normal)
	g.Hero = Hero{Class: ClassPaladin, Level: 12}
	g.Start()

	if rec.Get(1, FlagPrimaryGoal) || !rec.Get(1, FlagCompletedEarly) {
		t.Fatalf("load transform wrong: %#x", rec.Slot(1))
	}

	g.setQuestIntro(NPCAkara)

	d := g.Activate(NPCAkara)
	if s, ok := d.NextSpoken(g); !ok || s.Msg != 76 {
		t.Fatalf("pending reward should replay 76: %+v", d.Lines)
	}

	g.Hear(NPCAkara, 76)

	if !rec.Get(1, FlagRewardGranted) || rec.Get(1, FlagRewardPending) {
		t.Fatal("reward not claimed after reload")
	}
}

func TestRestoreFromFlags(t *testing.T) {
	tests := []struct {
		bits        []int
		state, last int
	}{
		{[]int{FlagStarted}, 2, 1},
		{[]int{FlagStarted, FlagLeaveTown}, 3, 1},
		{[]int{FlagStarted, FlagLeaveTown, FlagEnterArea}, 3, 2},
	}

	for _, id := range []int{QuestDenOfEvil, QuestBurial} {
		for _, tc := range tests {
			var body d2s.Body

			rec := body.QuestRecord(Normal)
			for _, b := range tc.bits {
				rec.Set(id, b) // slot == quest id for Act 1
			}

			g := New(rec, body.NPCFlags(), Normal)
			g.Start()

			q := g.Quest(id)
			if q.State != tc.state || q.LastState != tc.last {
				t.Errorf("quest %d bits %v: state %d/%d, want %d/%d", id, tc.bits, q.State, q.LastState, tc.state, tc.last)
			}
		}
	}
}

func TestCompletedQuestsAreInertAndChainContinues(t *testing.T) {
	var body d2s.Body

	rec := body.QuestRecord(Normal)
	rec.Set(1, FlagRewardGranted)
	rec.Set(2, FlagRewardGranted)

	g := New(rec, body.NPCFlags(), Normal)
	g.Start()

	if g.Quest(QuestDenOfEvil).NotIntro || g.Quest(QuestBurial).NotIntro {
		t.Error("completed quests must be inert")
	}

	// Q1 -> Q2 (inert) -> Q4 becomes available
	if g.Quest(QuestCain).State != 1 {
		t.Errorf("Search for Cain should be available, state %d", g.Quest(QuestCain).State)
	}

	if g.Quest(QuestTools).State != 0 || g.Quest(QuestAndariel).State != 0 {
		t.Error("later quests must wait for their predecessors")
	}
}

func TestBurialGroundsAndKashyaReward(t *testing.T) {
	g, _ := newGame(t)
	g.Quest(QuestBurial).State = 1 // as if Den of Evil were done
	q := g.Quest(QuestBurial)

	msgs, _ := talk(g, NPCKashya)
	if len(msgs) != 2 { // her intro + the quest
		t.Fatalf("Kashya msgs %v", msgs)
	}

	if q.State != 2 || !g.get(q, FlagStarted) {
		t.Fatalf("%s", g.Describe(q))
	}

	moveTo(g, LevelRogueEncampment, 2)
	moveTo(g, 2, LevelBurialGrounds)

	if !g.get(q, FlagLeaveTown) || !g.get(q, FlagEnterArea) || q.State != 3 {
		t.Fatalf("%s", g.Describe(q))
	}

	g.Dispatch(Event{Kind: EvMonsterKilled, Monster: NPCBloodRaven, Level: LevelBurialGrounds})

	if q.State != 4 || !g.get(q, FlagRewardPending) || !g.get(q, FlagPrimaryGoal) {
		t.Fatalf("%s", g.Describe(q))
	}

	g.Tick(15)

	if q.LastState != 3 {
		t.Errorf("log page after the timer %d", q.LastState)
	}

	moveTo(g, LevelBurialGrounds, LevelRogueEncampment)

	_, eff := talk(g, NPCKashya)
	if !hasEffect(eff, EffectHireRogues) {
		t.Fatalf("Kashya's rogues should become hirable: %+v", eff)
	}

	if !g.get(q, FlagRewardGranted) || q.State != 5 {
		t.Fatalf("%s", g.Describe(q))
	}

	if g.Quest(QuestCain).State != 1 {
		t.Error("Search for Cain should follow Sisters' Burial Grounds")
	}
}

func TestToolsOfTheTrade(t *testing.T) {
	g, _ := newGame(t)
	q := g.Quest(QuestTools)
	g.Quest(QuestTools).State = 1

	msgs, _ := talk(g, NPCCharsi)
	if len(msgs) != 2 || q.State != 2 {
		t.Fatalf("Charsi msgs %v: %s", msgs, g.Describe(q))
	}

	moveTo(g, LevelRogueEncampment, 3)

	if q.State != 3 || !g.get(q, FlagLeaveTown) {
		t.Fatalf("%s", g.Describe(q))
	}

	// below clvl 8 the chest refuses
	g.Hero.Level = 5
	eff := g.Dispatch(Event{Kind: EvObjectOperated, Object: ObjectHoradricMalus})

	if hasEffect(eff, EffectSpawn) || q.State != 3 {
		t.Fatalf("the chest must stay shut at clvl 5: %+v", eff)
	}

	g.Hero.Level = 9
	eff = g.Dispatch(Event{Kind: EvObjectOperated, Object: ObjectHoradricMalus})

	if !hasEffect(eff, EffectSpawn) || q.State != 4 {
		t.Fatalf("chest opened: %+v %s", eff, g.Describe(q))
	}

	g.Dispatch(Event{Kind: EvItemPickedUp, Item: ItemHoradricMalus})

	if !g.get(q, FlagCustom2) || q.LastState != 2 {
		t.Fatalf("pickup: %s", g.Describe(q))
	}

	moveTo(g, 3, LevelRogueEncampment)

	// Charsi's turn-in line is offered while the item is held
	d := g.Activate(NPCCharsi)
	if s, ok := d.NextSpoken(g); !ok || s.Msg != 163 {
		t.Fatalf("expected 163, got %+v", d.Lines)
	}

	eff = g.Hear(NPCCharsi, 163)

	if !hasEffect(eff, EffectImbue) || !hasEffect(eff, EffectDeleteItem) {
		t.Fatalf("effects %+v", eff)
	}

	if !g.get(q, FlagRewardPending) || !g.get(q, FlagPrimaryGoal) || q.State != 5 {
		t.Fatalf("%s", g.Describe(q))
	}

	if g.Quest(QuestAndariel).State != 0 {
		t.Error("Sisters to the Slaughter must wait for the log timer")
	}

	g.Tick(20)

	if g.Quest(QuestAndariel).State != 1 {
		t.Error("Sisters to the Slaughter should become available 20 frames later")
	}

	g.ClaimImbue()

	if !g.get(q, FlagRewardGranted) || g.get(q, FlagRewardPending) || q.Active {
		t.Fatalf("%s", g.Describe(q))
	}
}

func TestSearchForCain(t *testing.T) {
	for diff, want := range []struct{ quality, ilvl int }{{1, 7}, {2, 30}, {2, 60}} {
		var body d2s.Body

		g := New(body.QuestRecord(diff), body.NPCFlags(), diff)
		g.Hero = Hero{Class: ClassBarbarian, Level: 12}
		g.Start()

		q := g.Quest(QuestCain)
		q.State = 1

		talk(g, NPCAkara)

		if q.State != 2 || !g.get(q, FlagStarted) {
			t.Fatalf("%s", g.Describe(q))
		}

		moveTo(g, LevelRogueEncampment, 5)

		// the Inifuss tree
		eff := g.Dispatch(Event{Kind: EvObjectOperated, Object: ObjectInifussTree})
		if !hasEffect(eff, EffectSpawn) || q.State != 4 || q.LastState != 2 {
			t.Fatalf("tree: %+v %s", eff, g.Describe(q))
		}

		// a second use does nothing
		if eff = g.Dispatch(Event{Kind: EvObjectOperated, Object: ObjectInifussTree}); hasEffect(eff, EffectSpawn) {
			t.Error("the tree must give one scroll")
		}

		g.Dispatch(Event{Kind: EvItemPickedUp, Item: ItemScrollOfInifuss})

		if !g.get(q, FlagLeaveTown) {
			t.Fatalf("%s", g.Describe(q))
		}

		moveTo(g, 5, LevelRogueEncampment)

		d := g.Activate(NPCAkara)
		if s, ok := d.NextSpoken(g); !ok || s.Msg != 112 {
			t.Fatalf("Akara should decipher (112): %+v", d.Lines)
		}

		eff = g.Hear(NPCAkara, 112)
		if !hasEffect(eff, EffectGiveItem) || q.State != 5 || g.hasItem(ItemScrollOfInifuss) {
			t.Fatalf("decipher: %+v %s items=%v", eff, g.Describe(q), g.Items)
		}

		g.Close(NPCAkara)
		g.Dispatch(Event{Kind: EvItemPickedUp, Item: ItemDecipheredScroll}) // the hero picks up the new item

		// the stones: wrong stones are ignored, the right order solves it
		order := g.StoneOrder()
		wrong := order[1]

		g.Dispatch(Event{Kind: EvObjectOperated, Object: wrong})

		if d := g.byID[QuestCain].data.(*cainData); d.curStone != 0 {
			t.Fatal("a wrong stone advanced the puzzle")
		}

		var last []Effect

		for _, s := range order {
			last = g.Dispatch(Event{Kind: EvObjectOperated, Object: s})
		}

		if !hasEffect(last, EffectPortal) || !g.get(q, FlagEnterArea) || g.hasItem(ItemDecipheredScroll) || q.LastState != 4 {
			t.Fatalf("stones: %+v %s", last, g.Describe(q))
		}

		// Cain in the gibbet
		moveTo(g, LevelRogueEncampment, LevelTristram)
		eff = g.Dispatch(Event{Kind: EvObjectOperated, Object: ObjectCainGibbet})

		if !g.get(q, FlagRewardPending) || !g.get(q, FlagPrimaryGoal) || !hasEffect(eff, EffectSpawn) {
			t.Fatalf("gibbet: %+v %s", eff, g.Describe(q))
		}

		moveTo(g, LevelTristram, LevelRogueEncampment)

		// Akara's thanks gives the ring
		msgs, eff := talk(g, NPCAkara)

		var ring *Effect

		for i := range eff {
			if eff[i].Kind == EffectGiveItem && eff[i].Code == ItemReward {
				ring = &eff[i]
			}
		}

		if ring == nil || ring.Quality != want.quality || ring.Value != want.ilvl {
			t.Fatalf("difficulty %d: ring %+v msgs %v", diff, ring, msgs)
		}

		if !g.get(q, FlagRewardGranted) || q.State != 6 {
			t.Fatalf("%s", g.Describe(q))
		}

		if g.Quest(QuestTools).State != 1 {
			t.Error("Tools of the Trade should follow")
		}
	}
}

func TestSearchForCainAbandoned(t *testing.T) {
	g, _ := newGame(t)
	q := g.Quest(QuestCain)
	q.State = 1

	g.TravelToAct2()

	if q.State != 7 || !g.get(q, FlagCompletedNow) || !g.Rec.ActFinished(1) {
		t.Fatalf("%s act1=%v", g.Describe(q), g.Rec.ActFinished(1))
	}

	if !g.ReturnGreetingPending(NPCAkara) || !g.ReturnGreetingPending(NPCWarriv1) {
		t.Error("the Act 1 NPCs should greet with the welcome-back line")
	}

	g.ClearReturnGreeting(NPCAkara)

	if g.ReturnGreetingPending(NPCAkara) {
		t.Error("welcome-back flag not cleared")
	}
}

func TestForgottenTower(t *testing.T) {
	g, _ := newGame(t)
	q := g.Quest(QuestTower)

	moveTo(g, LevelRogueEncampment, 6)
	moveTo(g, 6, LevelForgottenTower)

	if q.State != 2 || !g.get(q, FlagStarted) || q.LastState != 3 {
		t.Fatalf("%s", g.Describe(q))
	}

	moveTo(g, LevelForgottenTower, LevelTowerCellar5)

	if q.State != 3 || !g.get(q, FlagEnterArea) {
		t.Fatalf("%s", g.Describe(q))
	}

	g.Dispatch(Event{Kind: EvMonsterKilled, Super: "The Countess", Level: LevelTowerCellar5, Monster: 45})

	if q.State != 5 || !g.get(q, FlagRewardGranted) || !g.get(q, FlagPrimaryGoal) || g.get(q, FlagRewardPending) {
		t.Fatalf("%s", g.Describe(q))
	}

	moveTo(g, LevelTowerCellar5, LevelRogueEncampment)

	// every NPC (except Warriv and Gheed) shows the marker and says the success line
	if !g.Marker(NPCAkara) || !g.Marker(NPCCharsi) {
		t.Error("marker rules")
	}

	g.Rec.Set(0, FlagRewardGranted) // Warriv's welcome would otherwise show a marker

	if g.Marker(NPCWarriv1) || g.Marker(NPCGheed) {
		t.Error("Warriv and Gheed have no congratulation marker")
	}

	if g.Quest(QuestTools).State != 0 {
		t.Error("Tools of the Trade must wait for the congratulations")
	}

	msgs, _ := talk(g, NPCKashya)
	if len(msgs) == 0 || msgs[len(msgs)-1] != 140 {
		t.Fatalf("Kashya msgs %v", msgs)
	}

	if g.Quest(QuestTools).State != 1 {
		t.Error("Tools of the Trade should become available after the congratulations")
	}
}

func TestSistersToTheSlaughter(t *testing.T) {
	g, _ := newGame(t)
	q := g.Quest(QuestAndariel)
	q.State = 1

	talk(g, NPCCain5)

	if q.State != 2 || !g.get(q, FlagStarted) {
		t.Fatalf("%s", g.Describe(q))
	}

	moveTo(g, LevelRogueEncampment, 34)
	moveTo(g, 34, 35)
	moveTo(g, 35, 36)
	moveTo(g, 36, LevelCatacombs4)

	if q.State != 3 || !g.get(q, FlagEnterArea) {
		t.Fatalf("%s", g.Describe(q))
	}

	eff := g.Dispatch(Event{Kind: EvMonsterKilled, Monster: NPCAndariel, Level: LevelCatacombs4})

	gems := 0

	for _, e := range eff {
		if e.Kind == EffectGiveItem {
			gems++
		}
	}

	if gems != 3 || q.State != 4 || !g.get(q, FlagRewardPending) || !g.get(q, FlagPrimaryGoal) {
		t.Fatalf("gems=%d %s", gems, g.Describe(q))
	}

	eff = g.Tick(12)

	if !hasEffect(eff, EffectPortal) || q.LastState != 3 {
		t.Fatalf("%+v %s", eff, g.Describe(q))
	}

	moveTo(g, LevelCatacombs4, LevelRogueEncampment)

	// Warriv, Cain, Akara and Kashya all have a congratulation
	for _, npc := range []int{NPCCain5, NPCAkara, NPCKashya, NPCWarriv1} {
		if !g.Marker(npc) {
			t.Errorf("npc %d should show a marker", npc)
		}
	}

	_, eff = talk(g, NPCWarriv1)

	if !g.get(q, FlagRewardGranted) || g.get(q, FlagRewardPending) || q.State != 5 {
		t.Fatalf("%s %+v", g.Describe(q), eff)
	}

	g.TravelToAct2()

	if !g.Rec.ActFinished(1) {
		t.Error("act 1 not finished")
	}
}

func TestToolsLevelGate(t *testing.T) {
	g, _ := newGame(t)
	q := g.Quest(QuestTools)
	q.State = 4
	g.Items[ItemHoradricMalus] = 1
	g.Hero.Level = 6

	if g.Marker(NPCCharsi) {
		t.Error("no marker below clvl 8")
	}

	for _, s := range g.Activate(NPCCharsi).Lines {
		if s.Quest == QuestTools {
			t.Errorf("Charsi must not take the Malus below clvl 8: %+v", s)
		}
	}
}

func TestRadamentAndBook(t *testing.T) {
	g, _ := newGame(t)
	q := g.Quest(QuestRadament)

	moveTo(g, 39, LevelLutGholein)

	msgs, _ := talk(g, NPCAtma)
	if len(msgs) == 0 || msgs[0] != 304 || q.State != 2 {
		t.Fatalf("msgs %v %s", msgs, g.Describe(q))
	}

	moveTo(g, LevelLutGholein, 41)
	g.Dispatch(Event{Kind: EvMonsterKilled, Monster: NPCRadament, Level: LevelSewers3})

	if q.State != 4 || !g.get(q, FlagRewardPending) || !g.get(q, FlagCustom1) {
		t.Fatalf("%s", g.Describe(q))
	}

	moveTo(g, 41, LevelLutGholein)
	talk(g, NPCAtma)

	if !g.get(q, FlagRewardGranted) {
		t.Fatalf("%s", g.Describe(q))
	}

	eff := g.ReadBookOfSkill()
	if !hasEffect(eff, EffectSkillPoint) || g.get(q, FlagCustom1) {
		t.Fatalf("book: %+v", eff)
	}
}

func TestIntroForSpecialClass(t *testing.T) {
	var body d2s.Body

	g := New(body.QuestRecord(Normal), body.NPCFlags(), Normal)
	g.Hero.Class = ClassSorceress
	g.Start()

	d := g.Activate(NPCAkara)
	if s, _ := d.NextSpoken(g); s.Msg != 12 {
		t.Errorf("a sorceress hears the _SOR intro (12), got %+v", d.Lines)
	}

	d = g.Activate(NPCCharsi)
	if s, _ := d.NextSpoken(g); s.Msg != 36 {
		t.Errorf("Charsi plain intro, got %+v", d.Lines)
	}

	g.Hear(NPCCharsi, 36)

	if !g.QuestIntroDone(NPCCharsi) || !body.NPCFlags().IntroBit(Normal, NPCBit(NPCCharsi)) {
		t.Error("intro not recorded in the NPC block")
	}

	// intro lines are said once
	d = g.Activate(NPCCharsi)
	for _, s := range d.Lines {
		if s.Msg == 36 || s.Msg == 37 {
			t.Error("intro repeated")
		}
	}
}

func TestWarrivProloguePaladin(t *testing.T) {
	var body d2s.Body

	g := New(body.QuestRecord(Normal), body.NPCFlags(), Normal)
	g.Hero.Class = ClassPaladin
	g.Start()

	d := g.Activate(NPCWarriv1)
	if s, _ := d.NextSpoken(g); s.Msg != 1 {
		t.Fatalf("paladins hear message 1: %+v", d.Lines)
	}

	g.Hear(NPCWarriv1, 1)

	if !g.Rec.Get(0, FlagRewardGranted) {
		t.Error("prologue slot not granted")
	}

	if _, ok := g.Activate(NPCWarriv1).NextSpoken(g); ok {
		t.Error("prologue must not repeat")
	}
}

func TestNaviBarks(t *testing.T) {
	g, _ := newGame(t)

	d := g.Activate(NPCNavi)
	if len(d.Lines) != 1 || (d.Lines[0].Msg != 59 && d.Lines[0].Msg != 60) {
		t.Fatalf("while the den is open Navi speaks 59 or 60: %+v", d.Lines)
	}

	g.Quest(QuestDenOfEvil).NotIntro = false

	d = g.Activate(NPCNavi)
	if len(d.Lines) != 1 || d.Lines[0].Msg < 61 || d.Lines[0].Msg > 63 {
		t.Fatalf("after the den Navi speaks 61-63: %+v", d.Lines)
	}
}

func TestTraceReportsBitChanges(t *testing.T) {
	var body d2s.Body

	var lines []string

	g := New(body.QuestRecord(Normal), body.NPCFlags(), Normal)
	g.Trace = func(s string) { lines = append(lines, s) }
	g.Start()
	g.Activate(NPCAkara)
	g.Hear(NPCAkara, 64)

	changes := g.TakeChanges()
	if len(changes) != 1 || changes[0].Quest != QuestDenOfEvil || changes[0].Before != 0 || changes[0].After != 1<<FlagStarted {
		t.Fatalf("changes %+v", changes)
	}

	found := false

	for _, l := range lines {
		if l == "QUEST A1Q1 slot=1 bits 0x0000->0x0004 (started) state=2" {
			found = true
		}
	}

	if !found {
		t.Fatalf("trace lines %q", lines)
	}
}

func TestLogStatuses(t *testing.T) {
	g, _ := newGame(t)

	find := func(act, idx int) QuestLog {
		for _, l := range g.Log() {
			if l.Act == act && l.Index == idx {
				return l
			}
		}

		t.Fatalf("no log entry %d/%d", act, idx)

		return QuestLog{}
	}

	if l := find(1, 1); l.Status != LogNotStarted {
		t.Errorf("fresh den: %+v", l)
	}

	talk(g, NPCAkara)
	moveTo(g, 1, 2)

	if l := find(1, 1); l.Status != LogInProgress || l.Page != 1 {
		t.Errorf("den in progress: %+v", l)
	}

	g.SetDenMonsters(1)
	moveTo(g, 2, LevelDenOfEvil)

	if l := find(1, 1); l.Page != 2 {
		t.Errorf("den entered: %+v", l)
	}

	g.Dispatch(Event{Kind: EvMonsterKilled, Level: LevelDenOfEvil})

	if l := find(1, 1); l.Status != LogCompleting {
		t.Errorf("den cleared: %+v", l)
	}

	talk(g, NPCAkara)

	if l := find(1, 1); l.Status != LogCompleted {
		t.Errorf("den claimed: %+v", l)
	}
}
