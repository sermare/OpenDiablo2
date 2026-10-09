package d2quest

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2txt"
)

// The monster class constants the boss and unique quest nodes match kills on
// must be the exe class ids of the real 1.14b monstats.txt (patch_d2): its hcIdx column
// (0-based, expansion divider row not counted; docs/boss-encounters.md).
// Skipped unless D2_TABLES is set.
func TestBossClassConstantsMatchMonstats(t *testing.T) {
	dir := os.Getenv("D2_TABLES")
	if dir == "" {
		t.Skip("D2_TABLES not set")
	}

	buf, err := os.ReadFile(filepath.Join(dir, "monsters", "patch_d2", "monstats.txt"))
	if err != nil {
		t.Skip(err)
	}

	want := map[string]int{ // monstats Id -> constant the quests use
		"andariel": NPCAndariel, "duriel": NPCDuriel, "mephisto": NPCMephisto, "diablo": NPCDiablo,
		"radament": NPCRadament, "summoner": NPCSummoner, "izual": NPCIzual, "bloodraven": NPCBloodRaven,
		"baalcrab": NPCBaalCrab,
	}

	d := d2txt.LoadDataDictionary(buf)
	seen := 0

	for d.Next() {
		id := d.String("Id")

		c, ok := want[id]
		if !ok {
			continue
		}

		seen++

		// the hcIdx column already carries the exe numbering (the divider row is not counted)
		hc := d.Number("hcIdx")
		if hc != c {
			t.Errorf("%s: exe class %d, constant %d", id, hc, c)
		}
	}

	if seen != len(want) {
		t.Errorf("found %d of %d rows", seen, len(want))
	}
}

// The kill hooks of the five act bosses: what the quest layer raises for each
// kill. Andariel (A1Q6) sets primary goal + reward pending and opens the town
// portal (act 2 travel itself is Warriv's, a separate trip); Duriel sets only
// custom1 + primary goal (Tyrael's talk gives the rest); the others set
// primary goal + reward pending. Which of these bits the exe sets for Diablo
// and Baal is UNVERIFIED (see boss.go).
func TestBossKillHookMatrix(t *testing.T) {
	type row struct {
		name  string
		ev    Event
		id    int
		slot  int
		flags []int
	}

	for _, c := range []row{
		{"andariel", Event{Kind: EvMonsterKilled, Monster: NPCAndariel, Name: "Andariel"}, QuestAndariel, 6,
			[]int{FlagPrimaryGoal, FlagRewardPending}},
		{"duriel", Event{Kind: EvMonsterKilled, Monster: NPCDuriel, Name: "Duriel"}, QuestSevenTombs, SlotSevenTombs,
			[]int{FlagPrimaryGoal, FlagCustom1}},
	} {
		g, _ := newGame(t)

		if q := g.Quest(c.id); q == nil || q.Slot != c.slot {
			t.Fatalf("%s: quest %d missing or wrong slot", c.name, c.id)
		}

		g.Dispatch(c.ev)

		for _, f := range c.flags {
			if g.Rec.Slot(c.slot)&(1<<f) == 0 {
				t.Errorf("%s: flag %d not set after the kill (slot 0x%04x)", c.name, f, g.Rec.Slot(c.slot))
			}
		}
	}
}
