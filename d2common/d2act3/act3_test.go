package d2act3

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2quest"
)

func TestQuestObjectsGiveTheirItems(t *testing.T) {
	cases := []struct {
		id   int
		item string
		lvl  int
	}{
		{ObjChestEye, d2quest.ItemKhalimEye, 85},
		{ObjChestBrain, d2quest.ItemKhalimBrain, 91},
		{ObjChestHeart, d2quest.ItemKhalimHeart, 93},
		{ObjLamEsenTome, d2quest.ItemLamEsenTome, 94},
		{ObjGidbinn, d2quest.ItemGidbinn, 0},
		{ObjCompellingOrb, "", 83},
		{ObjSewerLever, "", 92},
	}

	for _, c := range cases {
		o, ok := Object(c.id)
		if !ok {
			t.Errorf("object %d is not an Act 3 quest object", c.id)

			continue
		}

		if o.Item != c.item || o.Level != c.lvl {
			t.Errorf("object %d (%s): item %q level %d, want %q level %d", c.id, o.Name, o.Item, o.Level, c.item, c.lvl)
		}
	}

	for _, id := range []int{0, 1, 17, ObjHellgate, ObjMephistoBridge, ObjDuranceStairs} {
		if IsQuestObject(id) {
			t.Errorf("object %d is operated by the quest code but must not be", id)
		}
	}
}

func TestOrbNeedsTheWill(t *testing.T) {
	none := func(string) bool { return false }
	will := func(c string) bool { return c == d2quest.ItemKhalimWill }

	if CanSmashOrb(none) || !CanSmashOrb(will) {
		t.Fatal("the orb is smashed with Khalim's Will and nothing else")
	}
}

func TestCouncilKillsCountOnlyTheMembers(t *testing.T) {
	for _, n := range []string{"Ismail Vilehand", "Geleb Flamefinger", "Toorc Icefist", "toorc icefist"} {
		if !IsTravincalCouncil(n) {
			t.Errorf("%q should be a Council member of Travincal", n)
		}
	}

	for _, n := range []string{"Bremm Sparkfist", "Council Member", "", "Mephisto"} {
		if IsTravincalCouncil(n) {
			t.Errorf("%q is not a Council member of Travincal", n)
		}
	}

	if got := QuestKillClass(d2quest.NPCCouncilA, true); got != d2quest.NPCCouncilA {
		t.Errorf("a Council member's kill is class %d, want %d", got, d2quest.NPCCouncilA)
	}

	if got := QuestKillClass(d2quest.NPCCouncilB, false); got != 0 {
		t.Errorf("a follower of a Council member counted as class %d", got)
	}

	if got := QuestKillClass(242, false); got != 242 {
		t.Errorf("a plain kill changed class: %d", got)
	}
}

func TestFlailDropsOnceFromTheCouncil(t *testing.T) {
	has := func(codes ...string) func(string) bool {
		return func(c string) bool {
			for _, x := range codes {
				if x == c {
					return true
				}
			}

			return false
		}
	}

	if code, ok := FlailDrop("Ismail Vilehand", has()); !ok || code != d2quest.ItemKhalimFlail {
		t.Fatalf("the first Council kill should drop the Flail, got %q %v", code, ok)
	}

	if _, ok := FlailDrop("Ismail Vilehand", has(d2quest.ItemKhalimFlail)); ok {
		t.Error("a second Flail dropped")
	}

	if _, ok := FlailDrop("Geleb Flamefinger", has(d2quest.ItemKhalimWill)); ok {
		t.Error("the Flail dropped after the Will was made")
	}

	if _, ok := FlailDrop("Bremm Sparkfist", has()); ok {
		t.Error("a Council member of the Durance dropped the Flail")
	}
}

func TestSewerStairsWaitForTheLever(t *testing.T) {
	if err := CheckSewerStairs(92, 93, false); err != ErrSewerStairsHidden {
		t.Errorf("stairs without the lever: %v", err)
	}

	for _, c := range []struct {
		from, to int
		lever    bool
	}{{92, 93, true}, {93, 92, false}, {92, 80, false}, {80, 92, false}, {100, 101, false}} {
		if err := CheckSewerStairs(c.from, c.to, c.lever); err != nil {
			t.Errorf("%d -> %d (lever %v): %v", c.from, c.to, c.lever, err)
		}
	}
}
