package d2quest

import "testing"

func inLut(g *Game) { g.Dispatch(Event{Kind: EvAreaChanged, OldLevel: 1, NewLevel: LevelLutGholein}) }

func has(msgs []int, m int) bool {
	for _, x := range msgs {
		if x == m {
			return true
		}
	}

	return false
}

// The Act 2 quest line from the tainted sun to Meshif's boat, through the real speech tables.
func TestAct2QuestChain(t *testing.T) {
	g, _ := newGame(t)
	inLut(g)

	// Tainted Sun: dark after the far desert is entered, Drognan, the altar, the reward
	g.Dispatch(Event{Kind: EvAreaChanged, OldLevel: LevelLutGholein, NewLevel: LevelLostCity})
	g.Tick(20)

	sun := g.Quest(QuestTaintedSun)
	if sun.State != 1 {
		t.Fatalf("sun state %d, want 1", sun.State)
	}

	inLut(g)

	if m, _ := talk(g, NPCDrognan); !has(m, 348) || sun.State != 2 {
		t.Fatalf("Drognan %v state %d", m, sun.State)
	}

	g.Dispatch(Event{Kind: EvAreaChanged, OldLevel: LevelLutGholein, NewLevel: LevelValleyOfSnakes})
	g.Dispatch(Event{Kind: EvObjectOperated, Object: ObjTaintedSunAltar, Level: 61})

	if !g.Rec.Get(sun.Slot, FlagRewardPending) || sun.State != 4 {
		t.Fatalf("altar: bits 0x%x state %d", g.Rec.Slot(sun.Slot), sun.State)
	}

	inLut(g)

	if m, _ := talk(g, NPCJerhyn); !has(m, 362) || !g.Rec.Get(sun.Slot, FlagRewardGranted) || sun.State != 5 {
		t.Fatalf("Jerhyn %v bits 0x%x", m, g.Rec.Slot(sun.Slot))
	}

	// Arcane Sanctuary: Drognan 373, Jerhyn 377, the journal
	arc := g.Quest(QuestArcane)
	if arc.State != 1 {
		t.Fatalf("arcane offered? state %d", arc.State)
	}

	if m, _ := talk(g, NPCDrognan); !has(m, 373) || arc.State != 2 {
		t.Fatalf("Drognan arcane %v %d", m, arc.State)
	}

	if m, _ := talk(g, NPCJerhyn); !has(m, 377) || arc.State != 3 {
		t.Fatalf("Jerhyn arcane %v %d", m, arc.State)
	}

	g.Dispatch(Event{Kind: EvAreaChanged, OldLevel: LevelLutGholein, NewLevel: LevelArcaneSanctuary})
	g.Dispatch(Event{Kind: EvObjectOperated, Object: ObjHorazonJournal, Level: LevelArcaneSanctuary})

	if !g.Rec.Get(arc.Slot, FlagRewardGranted) || !g.Rec.Get(arc.Slot, FlagRewardPending) {
		t.Fatalf("journal: bits 0x%x", g.Rec.Slot(arc.Slot))
	}

	// the Summoner dies in his sanctuary and tells the town
	g.Dispatch(Event{Kind: EvMonsterKilled, Monster: NPCSummoner, Level: LevelArcaneSanctuary})

	sm := g.Quest(QuestSummoner)
	if !g.Rec.Get(sm.Slot, FlagRewardPending) {
		t.Fatalf("summoner bits 0x%x", g.Rec.Slot(sm.Slot))
	}

	inLut(g)

	if m, _ := talk(g, NPCGreiz); !has(m, 419) || !g.Rec.Get(sm.Slot, FlagRewardGranted) {
		t.Fatalf("Greiz %v", m)
	}

	if g.Quest(QuestSevenTombs).State != 1 {
		t.Fatalf("seven tombs not offered, state %d", g.Quest(QuestSevenTombs).State)
	}

	// Seven Tombs: Jerhyn 430, orifice, Duriel, Tyrael, Jerhyn 442, Meshif 450
	tb := g.Quest(QuestSevenTombs)
	talk(g, NPCJerhyn)

	if tb.State != 2 {
		t.Fatalf("Jerhyn 430: state %d", tb.State)
	}

	g.Items[ItemHoradricStaff] = 1
	g.Dispatch(Event{Kind: EvObjectOperated, Object: ObjStaffOrifice, Level: 67})

	if !g.Rec.Get(g.Quest(QuestStaff).Slot, FlagRewardGranted) || g.hasItem(ItemHoradricStaff) {
		t.Fatal("the orifice did not finish the Horadric Staff quest")
	}

	g.Dispatch(Event{Kind: EvMonsterKilled, Monster: NPCDuriel, Level: LevelDurielLair})

	if tb.State != 3 || !g.Rec.Get(tb.Slot, FlagCustom1) {
		t.Fatalf("Duriel: state %d bits 0x%x", tb.State, g.Rec.Slot(tb.Slot))
	}

	g.Dispatch(Event{Kind: EvAreaChanged, OldLevel: 67, NewLevel: LevelDurielLair})

	if m, _ := talk(g, NPCTyrael1); !has(m, 302) || tb.State != 4 {
		t.Fatalf("Tyrael %v state %d", m, tb.State)
	}

	inLut(g)

	var all []int

	for _, npc := range []int{NPCAtma, NPCWarriv2, NPCDrognan, NPCLysander, NPCCain2, NPCFara} {
		m, _ := talk(g, npc)
		all = append(all, m...)
	}

	for _, want := range []int{445, 446, 449, 444, 452, 447} {
		if !has(all, want) {
			t.Errorf("NPC line %d not spoken: %v", want, all)
		}
	}

	if m, _ := talk(g, NPCJerhyn); !has(m, 442) || tb.State != 5 {
		t.Fatalf("Jerhyn 442 %v state %d", m, tb.State)
	}

	if m, _ := talk(g, NPCMeshif1); !has(m, 450) || !g.Rec.Get(tb.Slot, FlagRewardGranted) {
		t.Fatalf("Meshif %v", m)
	}

	if !hasEffect(g.TravelToAct3(), EffectUnlockAct) {
		t.Error("no act 3")
	}
}

// Cain reads the scroll and hears about the three parts, then explains the assembled staff.
func TestHoradricStaffCain(t *testing.T) {
	g, _ := newGame(t)
	inLut(g)

	q := g.Quest(QuestStaff)
	g.Items[ItemHoradricScroll] = 1

	if m, _ := talk(g, NPCCain2); !has(m, 335) || g.hasItem(ItemHoradricScroll) || !g.Rec.Get(q.Slot, FlagLeaveTown) {
		t.Fatalf("scroll: %v", m)
	}

	g.Items[ItemAmuletOfTheViper], g.Items[ItemStaffOfKingsShaft] = 1, 1
	talk(g, NPCCain2)
	talk(g, NPCCain2)

	if !g.Rec.Get(q.Slot, FlagEnterArea) || !g.Rec.Get(q.Slot, FlagCustom1) {
		t.Fatalf("parts not heard: 0x%x", g.Rec.Slot(q.Slot))
	}

	if hasEffect(g.CubeHoradricStaff(), EffectGiveItem) != true || !g.hasItem(ItemHoradricStaff) {
		t.Fatal("cube recipe")
	}

	if m, _ := talk(g, NPCCain2); !has(m, 339) || !g.Rec.Get(q.Slot, FlagCustom6) {
		t.Fatalf("assembled: %v", m)
	}
}
