package d2quest

import "testing"

func TestBossKillsSetQuestBits(t *testing.T) {
	for _, c := range []struct {
		name  string
		ev    Event
		id    int
		slot  int
		extra uint16 // bits beyond primary goal + reward pending
	}{
		{"duriel", Event{Kind: EvMonsterKilled, Monster: NPCDuriel, Name: "Duriel"}, QuestSevenTombs, SlotSevenTombs, 1 << FlagCustom1},
		{"mephisto", Event{Kind: EvMonsterKilled, Monster: NPCMephisto, Name: "Mephisto"}, QuestGuardian, SlotGuardian, 0},
		{"diablo", Event{Kind: EvMonsterKilled, Monster: NPCDiablo, Name: "Diablo"}, QuestTerrorsEnd, SlotTerrorsEnd, 0},
		{"baal by name", Event{Kind: EvMonsterKilled, Monster: 12345, Name: "Baal"}, QuestEveOfDestruction, SlotEveOfDestruction, 0},
		{"baal by class", Event{Kind: EvMonsterKilled, Monster: NPCBaalCrab}, QuestEveOfDestruction, SlotEveOfDestruction, 0},
	} {
		g, _ := newGame(t)

		if g.Quest(c.id) == nil || g.Quest(c.id).Slot != c.slot {
			t.Fatalf("%s: no node for quest %d", c.name, c.id)
		}

		if v := g.Rec.Slot(c.slot); v != 0 {
			t.Fatalf("%s: slot starts with 0x%x", c.name, v)
		}

		g.Dispatch(c.ev)

		want := uint16(1<<FlagPrimaryGoal|1<<FlagRewardPending) | c.extra
		if got := g.Rec.Slot(c.slot); got != want {
			t.Errorf("%s: slot %d = 0x%04x want 0x%04x", c.name, c.slot, got, want)
		}

		// a second kill changes nothing
		g.Dispatch(c.ev)

		if got := g.Rec.Slot(c.slot); got != want {
			t.Errorf("%s: second kill changed the slot to 0x%04x", c.name, got)
		}
	}
}

func TestBossKillsIgnoreOthers(t *testing.T) {
	g, _ := newGame(t)

	for _, e := range []Event{
		{Kind: EvMonsterKilled, Monster: NPCAndariel, Name: "Andariel"},
		{Kind: EvMonsterKilled, Monster: 1, Name: "Zombie"},
		{Kind: EvMonsterKilled, Monster: 543, Name: "Baal Throne"},
	} {
		g.Dispatch(e)
	}

	for _, s := range []int{SlotSevenTombs, SlotGuardian, SlotTerrorsEnd, SlotEveOfDestruction} {
		if v := g.Rec.Slot(s); v != 0 {
			t.Errorf("slot %d changed to 0x%x", s, v)
		}
	}
}

// Duriel's bit 5 needs bits 0, 3, 4, 5 clear (VERIFIED condition of 0x59ac40).
func TestDurielBit5Condition(t *testing.T) {
	g, _ := newGame(t)
	g.Rec.Set(SlotSevenTombs, FlagLeaveTown)
	g.Dispatch(Event{Kind: EvMonsterKilled, Monster: NPCDuriel})

	if g.Rec.Get(SlotSevenTombs, FlagCustom1) {
		t.Error("bit 5 set although bit 3 was")
	}
}
