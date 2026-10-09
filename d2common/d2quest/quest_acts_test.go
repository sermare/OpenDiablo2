package d2quest

import "testing"

// heard feeds a message the way the client acknowledges it, in the act of the
// current level.
func heard(g *Game, npc, msg int) { g.Hear(npc, msg) }

func kill(g *Game, monster, level int) []Effect {
	return g.Dispatch(Event{Kind: EvMonsterKilled, Monster: monster, Level: level})
}

func obj(g *Game, id, level int) []Effect {
	return g.Dispatch(Event{Kind: EvObjectOperated, Object: id, Level: level})
}

func pickup(g *Game, code string) []Effect {
	return g.Dispatch(Event{Kind: EvItemPickedUp, Item: code})
}

func effectCode(effects []Effect, kind EffectKind, code string) bool {
	for _, e := range effects {
		if e.Kind == kind && e.Code == code {
			return true
		}
	}

	return false
}

func bit(g *Game, id, b int) bool { return g.Rec.Get(g.Quest(id).Slot, b) }

func TestAct2To5MessagesHaveNPCsAndSounds(t *testing.T) {
	prefixes := []string{"A2Q2", "A2Q3", "A2Q4", "A2Q5", "A2Q6", "A2Q7", "A3Q0", "A3Q1", "A3Q2", "A3Q3", "A3Q4",
		"A3Q5", "A3Q6", "A4Q0", "A4Q1", "A4Q2", "A4Q3", "A4Q4", "A5Q1", "A5Q2", "A5Q3", "A5Q4", "A5Q5", "A5Q6"}

	for _, q := range prefixes {
		if len(speechTables(q)) == 0 {
			t.Errorf("%s has no speech tables", q)
		}

		for i, tbl := range speechTables(q) {
			for _, s := range tbl {
				if s.NPC < 0 {
					t.Errorf("%s table %d: unknown NPC in %+v", q, i, s)
				}

				if snd, ok := SoundForMessage(s.Msg); !ok || snd.Index == 0 {
					t.Errorf("%s table %d msg %d has no Sounds.txt row", q, i, s.Msg)
				}
			}
		}
	}
}

func TestEveryQuestHasItsSlot(t *testing.T) {
	want := map[int]int{
		QuestHoradricStaff: 10, QuestTaintedSun: 11, QuestArcane: 12, QuestSummoner: 13, QuestSevenTombs: 14,
		QuestA3Prologue: 16, QuestLamEsen: 17, QuestKhalim: 18, QuestBlade: 19, QuestGoldenBird: 20,
		QuestBlackenedTemple: 21, QuestGuardian: 22, QuestA4Prologue: 24, QuestFallenAngel: 25, QuestTerrorsEnd: 26,
		QuestHellforge: 27, QuestMalachai: 33, QuestSiege: 35, QuestRescue: 36, QuestPrison: 37, QuestBetrayal: 38,
		QuestRite: 39, QuestEve: 40,
	}

	g, _ := newGame(t)

	for id, slot := range want {
		q := g.Quest(id)
		if q == nil || q.Slot != slot {
			t.Errorf("quest %d: %+v want slot %d", id, q, slot)
		}
	}
}

func TestHoradricStaffQuest(t *testing.T) {
	g, _ := newGame(t)
	q := g.Quest(QuestHoradricStaff)

	moveTo(g, 1, LevelLutGholein)

	// nothing to report: Cain has nothing to say
	if lines := g.Activate(NPCCain2).Lines; len(lines) != 0 {
		t.Fatalf("lines %+v", lines)
	}

	// the scroll chest gives the Horadric Scroll, Cain deciphers it
	eff := obj(g, ObjectScrollChest, 60)
	if !effectCode(eff, EffectSpawn, ItemHoradricScroll) {
		t.Fatalf("scroll chest: %+v", eff)
	}

	pickup(g, ItemHoradricScroll)

	if !g.Marker(NPCCain2) {
		t.Error("Cain shows no marker for the scroll")
	}

	msgs, eff := talk(g, NPCCain2)
	if len(msgs) != 1 || msgs[0] != 335 || !effectCode(eff, EffectDeleteItem, ItemHoradricScroll) {
		t.Fatalf("scroll: %v %+v", msgs, eff)
	}

	if !g.get(q, FlagLeaveTown) || g.hasItem(ItemHoradricScroll) {
		t.Errorf("%s", g.Describe(q))
	}

	// amulet, staff shaft and cube are reported one by one
	for _, c := range []struct {
		item string
		msg  int
		bit  int
	}{
		{ItemViperAmulet, 336, FlagEnterArea}, {ItemStaffOfKingsShaft, 337, FlagCustom1},
		{ItemHoradricCube, 338, FlagCustom2},
	} {
		pickup(g, c.item)

		msgs, _ = talk(g, NPCCain2)
		if len(msgs) != 1 || msgs[0] != c.msg || !g.get(q, c.bit) {
			t.Fatalf("%s: %v %s", c.item, msgs, g.Describe(q))
		}
	}

	// the cube made the staff: Cain's final line, and the palace quest opens
	pickup(g, ItemHoradricStaff)

	if !g.get(q, flagAssembled) {
		t.Fatalf("not assembled %s", g.Describe(q))
	}

	msgs, _ = talk(g, NPCCain2)
	if len(msgs) != 1 || msgs[0] != 339 || !g.get(q, FlagCustom6) {
		t.Fatalf("final: %v %s", msgs, g.Describe(q))
	}

	// the staff in Tal Rasha's orifice completes the quest and deletes the items
	eff = obj(g, ObjectOrifice, 68)
	if !g.get(q, FlagRewardGranted) || !g.get(q, FlagPrimaryGoal) || !effectCode(eff, EffectDeleteItem, ItemHoradricStaff) ||
		!effectCode(eff, EffectPortal, "") {
		t.Fatalf("orifice: %s %+v", g.Describe(q), eff)
	}
}

func TestTaintedSunAndArcaneSanctuary(t *testing.T) {
	g, _ := newGame(t)
	sun, arc := g.Quest(QuestTaintedSun), g.Quest(QuestArcane)

	moveTo(g, 1, LevelLutGholein)

	if sun.State != 0 || arc.State != 0 {
		t.Fatalf("dormant states %d %d", sun.State, arc.State)
	}

	// entering the Lost City darkens the sun
	moveTo(g, LevelLutGholein, LevelLostCity)
	moveTo(g, LevelLostCity, LevelLutGholein)

	if sun.State != 1 {
		t.Fatalf("sun %s", g.Describe(sun))
	}

	msgs, _ := talk(g, NPCDrognan)
	if len(msgs) == 0 || msgs[0] != 348 || sun.State != 2 || !g.get(sun, FlagStarted) {
		t.Fatalf("Drognan: %v %s", msgs, g.Describe(sun))
	}

	moveTo(g, LevelLutGholein, LevelLostCity)

	if sun.State != 3 || !g.get(sun, FlagLeaveTown) {
		t.Fatalf("left town: %s", g.Describe(sun))
	}

	eff := obj(g, ObjectTaintedSunAltar, 61)
	if !g.get(sun, FlagRewardPending) || !g.get(sun, FlagPrimaryGoal) || !effectCode(eff, EffectSpawn, ItemViperAmulet) {
		t.Fatalf("altar: %s %+v", g.Describe(sun), eff)
	}

	moveTo(g, LevelLostCity, LevelLutGholein)

	if !g.Marker(NPCDrognan) || g.Marker(NPCFara) {
		t.Error("only the NPCs with an m1 line show the marker (Fara has none)")
	}

	msgs, _ = talk(g, NPCDrognan)
	// Drognan pays out (371) and, since the sun quest opens the Arcane
	// Sanctuary quest, goes on with its first line (373) in the same talk
	if len(msgs) != 2 || msgs[0] != 371 || msgs[1] != 373 || !g.get(sun, FlagRewardGranted) ||
		g.get(sun, FlagRewardPending) || arc.State != 2 {
		t.Fatalf("reward: %v %s %s", msgs, g.Describe(sun), g.Describe(arc))
	}

	msgs, _ = talk(g, NPCJerhyn)
	if !contains2(msgs, 377) || arc.State != 3 {
		t.Fatalf("Jerhyn 377: %v %s", msgs, g.Describe(arc))
	}

	// the palace guard speaks one of the open lines
	if l := g.Activate(NPCGuard2).Lines; len(l) != 1 || l[0].Msg < 187 || l[0].Msg > 189 {
		t.Errorf("guard %+v", l)
	}

	moveTo(g, LevelLutGholein, LevelArcane)

	if arc.State != 4 || !g.get(arc, FlagCustom1) {
		t.Fatalf("sanctuary: %s", g.Describe(arc))
	}

	obj(g, ObjectHorazonJournal, LevelArcane)

	if !g.get(arc, FlagRewardPending) || !g.get(arc, FlagPrimaryGoal) {
		t.Fatalf("journal: %s", g.Describe(arc))
	}

	moveTo(g, LevelArcane, LevelLutGholein)
	talk(g, NPCJerhyn)

	if !g.get(arc, FlagRewardGranted) {
		t.Fatalf("claim: %s", g.Describe(arc))
	}
}

func TestSummonerAndSevenTombs(t *testing.T) {
	g, _ := newGame(t)
	sum, arc, tombs, rad := g.Quest(QuestSummoner), g.Quest(QuestArcane), g.Quest(QuestSevenTombs), g.Quest(QuestRadament)

	moveTo(g, 1, LevelArcane)

	if sum.State != 1 || !g.get(sum, FlagStarted) {
		t.Fatalf("summoner %s", g.Describe(sum))
	}

	kill(g, NPCSummoner, LevelArcane)

	if !g.get(sum, FlagRewardPending) || !g.get(sum, FlagPrimaryGoal) {
		t.Fatalf("summoner dead %s", g.Describe(sum))
	}

	// the Summoner's death also completes the Arcane Sanctuary quest
	if !g.get(arc, FlagPrimaryGoal) {
		t.Errorf("arcane %s", g.Describe(arc))
	}

	moveTo(g, LevelArcane, LevelLutGholein)

	msgs, _ := talk(g, NPCAtma)
	if !contains2(msgs, 427) || !g.get(sum, FlagRewardGranted) {
		t.Fatalf("Atma: %v %s", msgs, g.Describe(sum))
	}

	// the Arcane Sanctuary quest (claimed by Atma's 406) opened the Duriel quest
	if tombs.State != 1 {
		t.Fatalf("tombs %s", g.Describe(tombs))
	}

	moveTo(g, LevelLutGholein, 41)
	g.Dispatch(Event{Kind: EvMonsterKilled, Monster: NPCRadament, Level: LevelSewers3})
	moveTo(g, 41, LevelLutGholein)
	talk(g, NPCAtma)

	if !g.get(rad, FlagRewardGranted) || tombs.State != 1 {
		t.Fatalf("radament done, tombs %s", g.Describe(tombs))
	}

	msgs, _ = talk(g, NPCJerhyn)
	if !contains2(msgs, 430) || tombs.State != 2 || !g.get(tombs, FlagStarted) {
		t.Fatalf("Jerhyn 430: %v %s", msgs, g.Describe(tombs))
	}

	moveTo(g, LevelLutGholein, LevelDurielLair)
	kill(g, NPCDuriel, LevelDurielLair)

	if tombs.State != 3 || !g.get(tombs, FlagCustom1) {
		t.Fatalf("Duriel dead %s", g.Describe(tombs))
	}

	// Tyrael in the lair
	g.Level = LevelDurielLair
	g.Hear(NPCTyrael1, 302)

	if tombs.State != 4 || !g.get(tombs, FlagPrimaryGoal) || !g.get(tombs, FlagLeaveTown) {
		t.Fatalf("Tyrael %s", g.Describe(tombs))
	}

	moveTo(g, LevelDurielLair, LevelLutGholein)

	// the townspeople comment once
	msgs, _ = talk(g, NPCAtma)
	if !contains2(msgs, 445) || !g.get(tombs, FlagCustom2) {
		t.Fatalf("Atma 445: %v %s", msgs, g.Describe(tombs))
	}

	msgs, _ = talk(g, NPCJerhyn)
	if !contains2(msgs, 442) || tombs.State != 5 || !g.get(tombs, FlagEnterArea) || g.get(tombs, FlagLeaveTown) {
		t.Fatalf("Jerhyn 442: %v %s", msgs, g.Describe(tombs))
	}

	msgs, eff := talk(g, NPCMeshif1)
	if !contains2(msgs, 450) || !g.get(tombs, FlagRewardGranted) || g.get(tombs, FlagEnterArea) {
		t.Fatalf("Meshif: %v %s", msgs, g.Describe(tombs))
	}

	if !hasEffect(eff, EffectUnlockAct) {
		t.Errorf("no act unlock: %+v", eff)
	}
}

func contains2(l []int, n int) bool {
	for _, x := range l {
		if x == n {
			return true
		}
	}

	return false
}

func TestTravelToAct3SetsActWord(t *testing.T) {
	g, _ := newGame(t)

	eff := g.TravelToAct3()
	if !g.Rec.Get(slotAct2Finished, FlagRewardGranted) || !hasEffect(eff, EffectUnlockAct) {
		t.Fatalf("%+v", eff)
	}

	if !g.ReturnGreetingPending(NPCAtma) || !g.get(g.Quest(QuestHoradricStaff), FlagRewardGranted) {
		t.Error("welcome-back bit or the staff completion missing")
	}

	g.TravelToAct4()
	g.TravelToAct5()

	if !g.Rec.Get(slotAct3Finished, FlagRewardGranted) || !g.Rec.Get(slotAct4Finished, FlagRewardGranted) {
		t.Error("act words")
	}
}

func TestAct3Prologue(t *testing.T) {
	g, _ := newGame(t)

	moveTo(g, 1, LevelKurastDocktown)

	msgs, _ := talk(g, NPCHratli)
	if !contains2(msgs, 466) || contains2(msgs, 465) { // the test hero is a Sorceress
		t.Fatalf("Hratli: %v", msgs)
	}

	if !g.Rec.Get(16, FlagRewardGranted) {
		t.Error("prologue slot not set")
	}

	if msgs, _ = talk(g, NPCHratli); len(msgs) != 0 {
		t.Errorf("Hratli repeats: %v", msgs)
	}
}

func TestAct3QuestLine(t *testing.T) {
	g, _ := newGame(t)
	lam, khalim, blade := g.Quest(QuestLamEsen), g.Quest(QuestKhalim), g.Quest(QuestBlade)
	bird, temple, guardian := g.Quest(QuestGoldenBird), g.Quest(QuestBlackenedTemple), g.Quest(QuestGuardian)

	moveTo(g, 1, LevelKurastDocktown)

	// Lam Esen's Tome: 5 stat points
	msgs, _ := talk(g, NPCAlkor)
	if len(msgs) == 0 || msgs[0] != 549 || lam.State != 2 {
		t.Fatalf("Alkor: %v %s", msgs, g.Describe(lam))
	}

	moveTo(g, LevelKurastDocktown, LevelRuinedTemple)

	if lam.State != 3 || !g.get(lam, FlagEnterArea) {
		t.Fatalf("temple %s", g.Describe(lam))
	}

	pickup(g, ItemLamEsenTome)

	if !g.get(lam, FlagRewardPending) {
		t.Fatalf("tome %s", g.Describe(lam))
	}

	moveTo(g, LevelRuinedTemple, LevelKurastDocktown)

	msgs, eff := talk(g, NPCAlkor)
	if len(msgs) != 1 || msgs[0] != 564 || !g.get(lam, FlagRewardGranted) {
		t.Fatalf("Alkor reward: %v %s", msgs, g.Describe(lam))
	}

	if !effectCode(eff, EffectReward, "stat-points") || !effectCode(eff, EffectDeleteItem, ItemLamEsenTome) {
		t.Errorf("effects %+v", eff)
	}

	// Khalim's Will
	msgs, _ = talk(g, NPCCain3)
	if len(msgs) != 1 || msgs[0] != 543 || !g.get(khalim, FlagStarted) {
		t.Fatalf("Cain: %v %s", msgs, g.Describe(khalim))
	}

	for _, p := range []struct {
		item string
		msg  int
	}{{ItemKhalimEye, 545}, {ItemKhalimHeart, 544}, {ItemKhalimBrain, 546}, {ItemKhalimWill, 547}} {
		pickup(g, p.item)

		if msgs, _ = talk(g, NPCCain3); len(msgs) != 1 || msgs[0] != p.msg {
			t.Fatalf("%s: %v", p.item, msgs)
		}
	}

	obj(g, ObjectCompellingOrb, LevelTravincal)

	if !g.get(khalim, FlagRewardPending) {
		t.Fatalf("orb %s", g.Describe(khalim))
	}

	if msgs, _ = talk(g, NPCCain3); len(msgs) != 1 || msgs[0] != 548 || !g.get(khalim, FlagRewardGranted) {
		t.Fatalf("Cain 548: %v %s", msgs, g.Describe(khalim))
	}

	// The Blade of the Old Religion
	talk(g, NPCHratli)

	if blade.State != 2 {
		t.Fatalf("blade %s", g.Describe(blade))
	}

	moveTo(g, LevelKurastDocktown, LevelFlayerDungeon1)
	pickup(g, ItemGidbinn)

	if blade.State != 4 {
		t.Fatalf("gidbinn %s", g.Describe(blade))
	}

	moveTo(g, LevelFlayerDungeon1, LevelKurastDocktown)
	talk(g, NPCOrmus)

	if blade.State != 5 || g.Items[ItemGidbinn] != 0 {
		t.Fatalf("Ormus %s", g.Describe(blade))
	}

	talk(g, NPCAsheara)

	if !g.get(blade, FlagRewardPending) {
		t.Fatalf("Asheara %s", g.Describe(blade))
	}

	_, eff = talk(g, NPCOrmus)
	if !g.get(blade, FlagRewardGranted) || !effectCode(eff, EffectReward, "hire-ironwolves") {
		t.Fatalf("Ormus reward %s %+v", g.Describe(blade), eff)
	}

	// The Golden Bird starts with the figurine
	if bird.State != 0 {
		t.Fatalf("bird early %s", g.Describe(bird))
	}

	pickup(g, ItemJadeFigurine)
	talk(g, NPCCain3)
	_, eff = talk(g, NPCMeshif2)

	if bird.State != 3 || !effectCode(eff, EffectSpawn, ItemGoldenBird) {
		t.Fatalf("Meshif %s %+v", g.Describe(bird), eff)
	}

	// Alkor takes the bird (534) and pays out (538) in one talk
	msgs, eff = talk(g, NPCAlkor)
	if !contains2(msgs, 534) || !contains2(msgs, 538) || !g.get(bird, FlagRewardGranted) || !effectCode(eff, EffectReward, "life-boost") {
		t.Fatalf("bird reward %s %+v", g.Describe(bird), eff)
	}

	// Blackened Temple (opened by the Blade) and the Guardian
	if temple.State < 1 || guardian.State != 0 {
		t.Fatalf("chain %d %d", temple.State, guardian.State)
	}

	talk(g, NPCOrmus)
	moveTo(g, LevelKurastDocktown, LevelTravincal)

	for _, c := range []int{NPCCouncilA, NPCCouncilB} {
		kill(g, c, LevelTravincal)
	}

	if g.get(temple, FlagRewardPending) {
		t.Fatal("goal before the third council member")
	}

	kill(g, NPCCouncilC, LevelTravincal)

	if !g.get(temple, FlagRewardPending) {
		t.Fatalf("council %s", g.Describe(temple))
	}

	moveTo(g, LevelTravincal, LevelKurastDocktown)
	talk(g, NPCCain3)

	if !g.get(temple, FlagRewardGranted) || guardian.State != 1 {
		t.Fatalf("temple done %s; guardian %d", g.Describe(temple), guardian.State)
	}

	talk(g, NPCOrmus)
	moveTo(g, LevelKurastDocktown, LevelDurance1)
	moveTo(g, LevelDurance1, LevelDurance3)
	kill(g, NPCMephisto, LevelDurance3)

	if !g.get(guardian, FlagRewardPending) || guardian.State != 5 {
		t.Fatalf("mephisto %s", g.Describe(guardian))
	}

	moveTo(g, LevelDurance3, LevelKurastDocktown)
	talk(g, NPCAlkor)

	if !g.get(guardian, FlagRewardGranted) {
		t.Fatalf("guardian reward %s", g.Describe(guardian))
	}
}

func TestAct4QuestLine(t *testing.T) {
	g, _ := newGame(t)
	fa, te, hf := g.Quest(QuestFallenAngel), g.Quest(QuestTerrorsEnd), g.Quest(QuestHellforge)

	moveTo(g, 1, LevelPandemonium)

	// Tyrael welcomes the hero (664) and offers both quests in one talk
	if msgs, _ := talk(g, NPCTyrael2); !contains2(msgs, 664) || !contains2(msgs, 670) || !contains2(msgs, 681) {
		t.Fatalf("prologue %v", msgs)
	}

	if fa.State != 2 || !g.get(fa, FlagStarted) {
		t.Fatalf("Tyrael %s", g.Describe(fa))
	}

	moveTo(g, LevelPandemonium, LevelPlainsDespair)
	kill(g, NPCIzual, LevelPlainsDespair)

	if !g.get(fa, FlagRewardPending) {
		t.Fatalf("Izual %s", g.Describe(fa))
	}

	moveTo(g, LevelPlainsDespair, LevelPandemonium)

	_, eff := talk(g, NPCTyrael2)
	if !g.get(fa, FlagRewardGranted) {
		t.Fatalf("reward %s", g.Describe(fa))
	}

	skill := 0

	for _, e := range eff {
		if e.Kind == EffectSkillPoint {
			skill += e.Value
		}
	}

	if skill != 2 {
		t.Errorf("skill points %d", skill)
	}

	// Terror's End: the expansion lines
	if te.State < 2 {
		t.Fatalf("terror %s", g.Describe(te))
	}

	moveTo(g, LevelPandemonium, LevelChaosSanctum)
	kill(g, NPCDiablo, LevelChaosSanctum)
	moveTo(g, LevelChaosSanctum, LevelPandemonium)

	if msgs, _ := talk(g, NPCTyrael2); !contains2(msgs, 20000) || !g.get(te, FlagRewardGranted) {
		t.Fatalf("Tyrael exp: %v %s", msgs, g.Describe(te))
	}

	// the Hellforge: Cain's first line depends on the soulstone
	if l := g.Activate(NPCCain4).Lines; len(l) == 0 || l[0].Msg != 679 {
		t.Fatalf("Cain without stone: %+v", l)
	}

	pickup(g, ItemMephistoSoulstone)

	if l := g.Activate(NPCCain4).Lines; len(l) == 0 || l[0].Msg != 678 {
		t.Fatalf("Cain with stone: %+v", l)
	}

	if l := g.Activate(NPCMalachai).Lines; len(l) != 1 || l[0].Msg != 668 {
		t.Errorf("Malachai %+v", l)
	}

	talk(g, NPCCain4)

	// without the hammer the forge does nothing
	obj(g, ObjectHellforge, LevelRiverOfFlame)

	if g.get(hf, FlagRewardPending) {
		t.Fatal("forge worked without the hammer")
	}

	pickup(g, ItemHellforgeHammer)

	eff = obj(g, ObjectHellforge, LevelRiverOfFlame)
	if !g.get(hf, FlagRewardPending) || !effectCode(eff, EffectDeleteItem, ItemMephistoSoulstone) {
		t.Fatalf("forge %s %+v", g.Describe(hf), eff)
	}

	talk(g, NPCCain4)

	if !g.get(hf, FlagRewardGranted) {
		t.Fatalf("Cain reward %s", g.Describe(hf))
	}

	if l := g.Activate(NPCMalachai).Lines; len(l) != 1 || l[0].Msg != 669 {
		t.Errorf("Malachai after %+v", l)
	}
}

func TestAct5QuestLine(t *testing.T) {
	g, _ := newGame(t)

	moveTo(g, 1, LevelHarrogath)

	type stage struct {
		id      int
		giver   int
		area    int
		kills   []int
		claimBy int
		reward  string
	}

	stages := []stage{
		{QuestSiege, NPCLarzuk, LevelBloodyFoothills, nil, NPCLarzuk, "socket-quest"},
		{QuestRescue, NPCQualKehk, LevelFrigidHighlands, []int{NPCPrisonDoor, NPCPrisonDoor, NPCPrisonDoor}, NPCQualKehk,
			"hire-barbarians"},
		{QuestBetrayal, NPCDrehya, LevelNihlathakTemple, []int{NPCNihlathakBoss}, NPCDrehya, "personalize"},
		{QuestRite, NPCQualKehk, LevelArreatSummit, []int{NPCAncient1, NPCAncient2, NPCAncient3}, NPCMalah, ""},
	}

	for _, s := range stages {
		q := g.Quest(s.id)

		// the giver may already have been talked to by an earlier claim
		msgs, _ := talk(g, s.giver)
		if q.State < 2 || (q.State == 2 && len(msgs) == 0 && !g.get(q, FlagStarted)) {
			t.Fatalf("%s giver: %v %s", q.Label, msgs, g.Describe(q))
		}

		if !g.get(q, FlagStarted) {
			t.Fatalf("%s: %s", q.Label, g.Describe(q))
		}

		moveTo(g, LevelHarrogath, s.area)

		if q.State != 3 || !g.get(q, FlagEnterArea) {
			t.Fatalf("%s area: %s", q.Label, g.Describe(q))
		}

		if s.id == QuestSiege {
			g.Dispatch(Event{Kind: EvMonsterKilled, Monster: 999, Super: "Shenk the Overseer", Level: s.area})
		}

		for i, k := range s.kills {
			if i < len(s.kills)-1 && g.get(q, FlagRewardPending) {
				t.Fatalf("%s: goal after %d kills", q.Label, i)
			}

			kill(g, k, s.area)
		}

		if !g.get(q, FlagRewardPending) || !g.get(q, FlagPrimaryGoal) {
			t.Fatalf("%s goal: %s", q.Label, g.Describe(q))
		}

		moveTo(g, s.area, LevelHarrogath)

		if !g.Marker(s.claimBy) {
			t.Errorf("%s: no marker on NPC %d", q.Label, s.claimBy)
		}

		_, eff := talk(g, s.claimBy)
		if !g.get(q, FlagRewardGranted) || g.get(q, FlagRewardPending) {
			t.Fatalf("%s claim: %s", q.Label, g.Describe(q))
		}

		if s.reward != "" && !effectCode(eff, EffectReward, s.reward) {
			t.Errorf("%s: no reward %s in %+v", q.Label, s.reward, eff)
		}
	}

	// Eve of Destruction opened with the Rite of Passage
	eve := g.Quest(QuestEve)
	if eve.State != 1 {
		t.Fatalf("eve %s", g.Describe(eve))
	}

	moveTo(g, LevelHarrogath, LevelWorldstone1)
	moveTo(g, LevelWorldstone1, LevelThrone)
	kill(g, NPCBaal, LevelThrone)

	if !g.get(eve, FlagRewardPending) {
		t.Fatalf("baal %s", g.Describe(eve))
	}

	moveTo(g, LevelThrone, LevelHarrogath)

	_, eff := talk(g, NPCTyrael3)
	if !g.get(eve, FlagRewardGranted) || !effectCode(eff, EffectReward, "unlock-difficulty") {
		t.Fatalf("eve reward %s %+v", g.Describe(eve), eff)
	}
}

func TestPrisonOfIce(t *testing.T) {
	g, _ := newGame(t)
	q := g.Quest(QuestPrison)

	moveTo(g, 1, LevelHarrogath)
	talk(g, NPCMalah)

	if q.State != 2 {
		t.Fatalf("%s", g.Describe(q))
	}

	moveTo(g, LevelHarrogath, LevelFrozenRiver)

	if q.State != 4 {
		t.Fatalf("river %s", g.Describe(q))
	}

	// the frozen Anya and Malah both speak at this stage
	if l := g.Activate(NPCAnyaFrozen).Lines; len(l) != 1 || l[0].Msg != 20131 {
		t.Fatalf("Anya %+v", l)
	}

	_, eff := talk(g, NPCAnyaFrozen)
	if q.State != 5 || !effectCode(eff, EffectSpawn, ItemMalahScroll) {
		t.Fatalf("scroll %s %+v", g.Describe(q), eff)
	}

	pickup(g, ItemMalahScroll)
	g.Dispatch(Event{Kind: EvItemRemoved, Item: ItemMalahScroll})

	if !g.get(q, FlagRewardPending) {
		t.Fatalf("read %s", g.Describe(q))
	}

	moveTo(g, LevelFrozenRiver, LevelHarrogath)

	_, eff = talk(g, NPCMalah)
	if !g.get(q, FlagRewardGranted) || !effectCode(eff, EffectReward, "resist-bonus") {
		t.Fatalf("claim %s %+v", g.Describe(q), eff)
	}
}

func TestLaterActQuestsRestoreFromBits(t *testing.T) {
	g, body := newGame(t)

	moveTo(g, 1, LevelHarrogath)
	talk(g, NPCLarzuk)
	moveTo(g, LevelHarrogath, LevelBloodyFoothills)

	// a new game from the same record restores the state from the bits
	g2 := New(body.QuestRecord(Normal), body.NPCFlags(), Normal)
	g2.Hero = Hero{Class: ClassSorceress, Level: 10}
	g2.Start()

	q := g2.Quest(QuestSiege)
	if q.State != 3 || q.LastState != 2 {
		t.Fatalf("restored %s", g2.Describe(q))
	}

	// a completed quest is inert and opens its successor
	g2.Rec.Set(g2.Quest(QuestRite).Slot, FlagRewardGranted)

	g3 := New(body.QuestRecord(Normal), body.NPCFlags(), Normal)
	g3.Start()

	if g3.Quest(QuestRite).NotIntro || g3.Quest(QuestEve).State != 1 {
		t.Errorf("rite %v eve %d", g3.Quest(QuestRite).NotIntro, g3.Quest(QuestEve).State)
	}
}

func TestLogCoversAllActs(t *testing.T) {
	g, _ := newGame(t)

	perAct := map[int]int{}

	for _, l := range g.Log() {
		perAct[l.Act]++
	}

	for act, want := range map[int]int{1: 6, 2: 6, 3: 6, 4: 3, 5: 6} {
		if perAct[act] != want {
			t.Errorf("act %d has %d log entries, want %d", act, perAct[act], want)
		}
	}
}
