package d2mapgen

import (
	"sort"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
)

// superKeys lists the super unique placements the engine holds after a level build.
func superKeys(g *MapGenerator) []string {
	var out []string

	for _, e := range g.engine.Entities() {
		if npc, ok := e.(*d2mapentity.NPC); ok && npc.SuperKey != "" {
			out = append(out, npc.SuperKey)
		}
	}

	sort.Strings(out)

	return out
}

// Playtest bug 44: the Council of Travincal and the guards of Mephisto stood in their levels as plain
// "Council Member" monsters without followers, names or treasure class. The DS1 names a super unique, the
// population keeps its SuperUniques.txt key on the placement and the monster director builds the unique
// (real data: D2_GAME_DIR).
func TestAct3CouncilPlacementsCarryTheirSuperUniques(t *testing.T) {
	a := perfAssets(t)
	g := perfGenerator(t, a)

	for _, c := range []struct {
		level  int
		want   []string
		preset bool
	}{
		{83, []string{"Geleb Flamefinger", "Ismail Vilehand", "Toorc Icefist"}, false},
		{102, []string{"Bremm Sparkfist", "Maffer Dragonhand", "Wyand Voidfinger"}, true},
	} {
		var err error
		if c.preset {
			err = g.GenerateRealPreset(c.level, perfSeed, 0)
		} else {
			err = g.GenerateRealOutdoor(c.level, perfSeed, 0)
		}

		if err != nil {
			t.Fatalf("level %d: %v", c.level, err)
		}

		got := superKeys(g)
		if len(got) != len(c.want) {
			t.Fatalf("level %d: super unique placements %v, want %v", c.level, got, c.want)
		}

		for i := range got {
			if got[i] != c.want[i] {
				t.Errorf("level %d: super unique placements %v, want %v", c.level, got, c.want)

				break
			}
		}
	}
}
