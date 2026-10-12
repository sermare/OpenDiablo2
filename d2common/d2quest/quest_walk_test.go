package d2quest

import "testing"

// Found by playing the Horadric Staff: once the staff is assembled the log must show page 4 ("Take the Staff
// into Tal Rasha's Tomb"), also after Cain confirmed it (it showed page 5, "Take the artifacts to Cain").
func TestHoradricStaffLogPageAfterAssembly(t *testing.T) {
	g, _ := newGame(t)
	q := g.Quest(QuestHoradricStaff)

	moveTo(g, 1, LevelLutGholein)
	pickup(g, ItemHoradricScroll)
	talk(g, NPCCain2)

	if p := g.LogPage(q); p != 2 {
		t.Errorf("page after the scroll = %d, want 2 (search the Halls and the Maggot Lair)", p)
	}

	for _, c := range []string{ItemViperAmulet, ItemStaffOfKingsShaft, ItemHoradricCube} {
		pickup(g, c)
		talk(g, NPCCain2)
	}

	if p := g.LogPage(q); p != 3 {
		t.Errorf("page after every report = %d, want 3 (restore the staff in the cube)", p)
	}

	// the cube recipe consumes the three parts
	for _, c := range []string{ItemViperAmulet, ItemStaffOfKingsShaft, ItemHoradricCube} {
		g.Dispatch(Event{Kind: EvItemRemoved, Item: c})
	}

	pickup(g, ItemHoradricStaff)

	// VERIFIED 0x59c1f0: the staff in the pack is page 6 until Cain's line 339 sets bit 10, then 5; both pages show
	// "Take the Staff into Tal Rasha's Tomb" (qstsa2q24 in the page table of the client)
	if p := g.LogPage(q); p != 6 {
		t.Errorf("page after the cube made the staff = %d, want 6", p)
	}

	talk(g, NPCCain2) // msg 339

	if p := g.LogPage(q); p != 5 {
		t.Errorf("page after Cain's confirmation = %d, want 5", p)
	}

	for _, p := range []int{5, 6} {
		if k := LogPageKey(10, p); k != "qstsa2q24" {
			t.Errorf("page %d of the staff shows %q, want qstsa2q24", p, k)
		}
	}
}

// The quest objects of Acts 2-4 are walked to as quest objects (the engine only did so for the Act 1 ones, so the
// chests, the altar and the tome never reached the quest system in the real world).
func TestLaterQuestObjectsAreQuestObjects(t *testing.T) {
	for _, id := range []int{ObjectCubeChest, ObjectScrollChest, ObjectStaffChest, ObjectTaintedSunAltar,
		ObjectOrifice, ObjectHorazonJournal, ObjectLamEsenTome, ObjectCompellingOrb, ObjectHellforge} {
		if !IsQuestObject(id) {
			t.Errorf("object %d is not a quest object", id)
		}
	}

	if IsQuestObject(1) {
		t.Error("object 1 must not be a quest object")
	}
}

// Lam Esen's Tome: operating the tome object on the altar of the Ruined Temple gives the item; picking it up moves the
// quest to state 4 (the hand-over to Alkor). VERIFIED 0x5b53e0: the goal and the reward come only with Alkor's line.
func TestLamEsenTomeObjectGivesTheItem(t *testing.T) {
	g, _ := newGame(t)
	q := g.Quest(QuestLamEsen)

	eff := obj(g, ObjectLamEsenTome, LevelRuinedTemple)
	if !effectCode(eff, EffectSpawn, ItemLamEsenTome) {
		t.Fatalf("tome object: %+v", eff)
	}

	pickup(g, ItemLamEsenTome)

	if q.State != 4 || g.get(q, FlagPrimaryGoal) || g.get(q, FlagRewardPending) || g.get(q, FlagRewardGranted) {
		t.Errorf("taking the tome must only reach state 4: %s", g.Describe(q))
	}
}
