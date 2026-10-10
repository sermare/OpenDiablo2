package d2gamescreen

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapengine"
)

// A super unique's minions and extras, and the unlinked party packs of any leader, belong to the pack they were
// created around, so they are neither natural types of the level nor density-roll groups of their own.
func TestClassifyPlanAndGroups(t *testing.T) {
	plan := []d2mapengine.PlannedMonster{
		{Key: "skeleton1", Leader: -1},                        // 0 natural leader
		{Key: "skeleton1", Leader: 0, Origin: 1},              // 1 its follower
		{Key: "zombie1", Leader: -1, Origin: 1},               // 2 unlinked party member of the leader (Leader cleared)
		{Key: "fallen2", Leader: -1, SuperKey: "Rakanishu"},   // 3 super unique
		{Key: "fallen1", Leader: 3, Origin: 4, Minion: true},  // 4 linked super follower
		{Key: "goatman1", Leader: -1, Origin: 4},              // 5 unlinked extra of the super
		{Key: "goatman1", Leader: -1, Unique: true},           // 6 rare leader
		{Key: "goatman1", Leader: 6, Origin: 7, Minion: true}, // 7 rare minion
		{Key: "bighead1", Leader: -1, Origin: 7},              // 8 unlinked party member of the rare
		{Key: "zombie1", Leader: -1},                          // 9 second natural group
	}

	want := []planKind{planNatural, planNatural, planNatural, planSuper, planSuper, planSuper, planPack, planPack, planPack, planNatural}
	kinds := classifyPlan(plan)

	for i := range want {
		if kinds[i] != want[i] {
			t.Errorf("unit %d (%s): kind %d, want %d", i, plan[i].Key, kinds[i], want[i])
		}
	}

	// groups the density rolls made: unit 0, the rare leader 6 and unit 9
	if got := countPlanGroups(plan, kinds); got != 3 {
		t.Errorf("groups = %d, want 3", got)
	}
}
