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

	pickup(g, ItemHoradricStaff)

	if p := g.LogPage(q); p != 4 {
		t.Errorf("page after the cube made the staff = %d, want 4", p)
	}

	talk(g, NPCCain2) // msg 339

	if p := g.LogPage(q); p != 4 {
		t.Errorf("page after Cain's confirmation = %d, want 4", p)
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
