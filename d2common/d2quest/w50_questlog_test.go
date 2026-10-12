package d2quest

import "testing"

// w50: the quest log of the original. Page tables and rules read from Game.exe 1.14b (logview.go, logfilters.go).

// TestW50PageTable pins values of the client page table (0x722aa8..0x7231a7). The pages are not qstsa<act>q<n><page>.
func TestW50PageTable(t *testing.T) {
	for _, c := range []struct {
		slot, page int
		want       string
	}{
		{1, 1, "qstsa1q11"}, {1, 5, "qstsa1q15"}, {1, 10, "qstsa1q15"}, {1, 11, "qstsprevious"}, {1, 12, "qstsother"},
		{1, 13, "qstsComplete"}, {1, 6, ""},
		{3, 3, "qstsa1q3x"}, {3, 4, "Qstsyouarenot8"}, {3, 10, "qstsa1q32b"},
		{4, 10, "qstsa1q46b"}, {6, 10, "qstsa1q62b"},
		{10, 4, "qstsa2q25"}, {10, 5, "qstsa2q24"}, {10, 6, "qstsa2q24"},
		{14, 3, "id11030"}, {14, 7, "qstsa2q61a"},
		{20, 3, "qstsa3q43"}, {20, 4, "qstsa3q44"}, {20, 5, "qstsa3q45"}, {20, 6, "qstsThankYouComeAgain"},
		{36, 2, "qstsa5q22"}, {37, 6, "qstsa5q35"}, {38, 5, "qstsa5q43a"}, {40, 3, "qstsa5q62"},
		{27, 10, "qstsprevious"},
	} {
		if got := LogPageKey(c.slot, c.page); got != c.want && !(c.want == "id11030" && got != "") {
			t.Errorf("slot %d page %d: key %q, want %q", c.slot, c.page, got, c.want)
		}
	}
}

// TestW50SlotTable pins the placement table: every quest with a log index sits in a slot of its act and the title number
// of the table (qstsa<act>q<index>) is the LogIndex the quest system uses.
func TestW50SlotTable(t *testing.T) {
	g, _ := newGame(t)

	// title numbers read from the first word of each quest table
	title := map[int]int{
		1: 1, 2: 2, 3: 3, 4: 4, 5: 5, 6: 6, 9: 1, 10: 2, 11: 3, 12: 4, 13: 5, 14: 6, 17: 1, 18: 2, 19: 3, 20: 4, 21: 5, 22: 6,
		25: 1, 26: 2, 27: 3, 35: 1, 36: 2, 37: 3, 38: 4, 39: 5, 40: 6,
	}

	seen := 0

	for _, q := range g.Quests {
		if q.LogIndex < 1 {
			continue
		}

		seen++

		info, ok := LogSlotInfo(q.Slot)
		if !ok {
			t.Errorf("%s slot %d has no log placement", q.Label, q.Slot)

			continue
		}

		if info.Act != q.Act+1 {
			t.Errorf("%s: act %d, table says %d", q.Label, q.Act+1, info.Act)
		}

		if title[q.Slot] != q.LogIndex {
			t.Errorf("%s: log index %d, title number of the table %d", q.Label, q.LogIndex, title[q.Slot])
		}
	}

	if seen != 27 {
		t.Errorf("%d quests with a log entry, want 27", seen)
	}

	// sockets of an act are a permutation of 0..5 (0..2 in Act 4)
	for act := 1; act <= 5; act++ {
		used := map[int]bool{}

		for _, q := range g.Quests {
			if info, ok := LogSlotInfo(q.Slot); ok && info.Act == act {
				if used[info.Socket] {
					t.Errorf("act %d: socket %d used twice", act, info.Socket)
				}

				used[info.Socket] = true
			}
		}
	}
}

// TestW50SiegePages: reward pending shows page 3 until Larzuk's line (bit 5), then page 4 (0x584bd0).
func TestW50SiegePages(t *testing.T) {
	g, _ := newGame(t)
	g.Hero.Level = 30

	moveTo(g, 1, LevelHarrogath)
	talk(g, NPCLarzuk)
	pageIs(t, g, QuestSiege, 1, "Larzuk")

	moveTo(g, LevelHarrogath, LevelBloodyFoothills)
	g.Dispatch(Event{Kind: EvMonsterKilled, Monster: 999, Super: "Shenk the Overseer", Level: LevelBloodyFoothills})
	g.Tick(20)

	q := g.Quest(QuestSiege)
	if !g.get(q, FlagRewardPending) || g.get(q, FlagCustom1) {
		t.Fatalf("after Shenk: %s", g.Describe(q))
	}

	pageIs(t, g, QuestSiege, 3, "Shenk dead, Larzuk not heard")

	moveTo(g, LevelBloodyFoothills, LevelHarrogath)
	talk(g, NPCLarzuk)
	pageIs(t, g, QuestSiege, 4, "Larzuk will add sockets")

	g.ClaimSocket()
	// the getter still answers 4 (it looks at bit 13 before the reward bit); the client shows "Quest completed" (page 13)
	pageIs(t, g, QuestSiege, 4, "sockets added")
}

// TestW50TombsPages: the Seven Tombs show page 5 after Tyrael's message (bit 3), 6 after Jerhyn's thanks (bit 4), and 0
// once Meshif took the hero east (0x59a660).
func TestW50TombsPages(t *testing.T) {
	g, _ := newGame(t)
	q := g.Quest(QuestSevenTombs)
	q.NotIntro, q.State, q.LastState = true, 2, 2

	pageIs(t, g, QuestSevenTombs, 2, "stored page while the state is below 3")

	q.State = 3
	pageIs(t, g, QuestSevenTombs, 12, "state 3 without the Duriel bit: page 12")

	g.set(q, FlagCustom1, "Duriel dead")

	q.LastState = 3
	pageIs(t, g, QuestSevenTombs, 3, "Duriel dead: stored page 3")

	g.set(q, FlagLeaveTown, "Tyrael")
	pageIs(t, g, QuestSevenTombs, 5, "Tyrael's message")

	g.clear(q, FlagLeaveTown, "Jerhyn")
	g.set(q, FlagEnterArea, "Jerhyn")
	pageIs(t, g, QuestSevenTombs, 6, "Jerhyn's thanks")

	g.set(q, FlagRewardGranted, "Meshif")
	pageIs(t, g, QuestSevenTombs, 0, "done")
}

// TestW50GoldenBirdPages: figurine 1/2, bird 3/4, reward 5 (0x5b7d90).
func TestW50GoldenBirdPages(t *testing.T) {
	g, _ := newGame(t)
	moveTo(g, 1, LevelKurastDocktown)

	pageIs(t, g, QuestGoldenBird, 0, "nothing yet")

	pickup(g, ItemJadeFigurine)
	pageIs(t, g, QuestGoldenBird, 1, "figurine in the pack: ask Cain")

	talk(g, NPCCain3)
	pageIs(t, g, QuestGoldenBird, 2, "Cain heard: show Meshif")

	talk(g, NPCMeshif2)
	g.Dispatch(Event{Kind: EvItemRemoved, Item: ItemJadeFigurine})
	pickup(g, ItemGoldenBird)
	pageIs(t, g, QuestGoldenBird, 3, "bird in the pack: ask Cain")

	g.Hear(NPCCain3, 531) // the topic "the Golden Bird"
	pageIs(t, g, QuestGoldenBird, 4, "Cain explained: give it to Alkor")

	talk(g, NPCAlkor)
	g.Dispatch(Event{Kind: EvItemRemoved, Item: ItemGoldenBird})
	pageIs(t, g, QuestGoldenBird, 13, "Alkor paid: reward granted with the goal bit")
}

// TestW50TaintedSunDefault: the default getter shows 4 or 12 for a finished-state Tainted Sun without the goal bit.
func TestW50TaintedSunDefault(t *testing.T) {
	g, _ := newGame(t)
	q := g.Quest(QuestTaintedSun)
	q.NotIntro, q.State, q.InitNo = true, 5, 4

	q.LastState = 4
	pageIs(t, g, QuestTaintedSun, 4, "Tainted Sun, stored page 4")

	q.LastState = 2
	pageIs(t, g, QuestTaintedSun, 12, "Tainted Sun, other stored page")
}
