package d2object

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2level"
)

func TestRemapShrineCode(t *testing.T) {
	tests := []struct{ in, want int }{
		{ShrineManaExchange, ShrineMana}, {ShrineHealthExchange, ShrineHealth}, {ShrineEnirhs, ShrineGem},
		{ShrineRefill, ShrineRefill}, {ShrineArmor, ShrineArmor}, {ShrinePoison, ShrinePoison}, {ShrineStorm, ShrineStorm},
	}

	for _, tc := range tests {
		if got := RemapShrineCode(tc.in); got != tc.want {
			t.Errorf("RemapShrineCode(%d) = %d, want %d", tc.in, got, tc.want)
		}
	}
}

func TestRollShrineUniform(t *testing.T) {
	counts := map[int]int{}

	for seed := uint32(0); seed < 4000; seed++ {
		s, ok := RollShrineUniform(DefaultShrines, NewRoller(seed), 99)
		if !ok {
			t.Fatal("no shrine at level 99")
		}

		counts[s.Code]++
	}

	for _, c := range []int{ShrineHealthExchange, ShrineManaExchange, ShrineEnirhs, ShrineNone} {
		if counts[c] != 0 {
			t.Errorf("code %d rolled %d times; it is remapped or never rolled", c, counts[c])
		}
	}

	// every other code is reachable at level 99 (uniform: ~4000/22 each)
	for c := 1; c < numShrines; c++ {
		if c == ShrineHealthExchange || c == ShrineManaExchange || c == ShrineEnirhs {
			continue
		}

		if counts[c] < 80 {
			t.Errorf("code %d rolled only %d times of 4000", c, counts[c])
		}
	}

	// the remap of Enirhs doubles the gem upgrade share
	if counts[ShrineGem] < counts[ShrineStorm]*3/2 {
		t.Errorf("gem %d vs storm %d: remap of Enirhs missing", counts[ShrineGem], counts[ShrineStorm])
	}

	// at area level 1 a shrine with a higher LevelMin needs 8 failed tries in a row to survive
	high := 0

	for seed := uint32(0); seed < 4000; seed++ {
		s, _ := RollShrineUniform(DefaultShrines, NewRoller(seed), 1)
		if s.LevelMin > 1 {
			high++
		}
	}

	if high > 600 { // ~9%: Enirhs (code 16, LevelMin 1) is remapped to the level-4 Gem Upgrade after the level check
		t.Errorf("%d of 4000 level-1 rolls kept a shrine above the level", high)
	}
}

func TestWellPulse(t *testing.T) {
	full := WellVitals{Vitals: Vitals{Life: 100, MaxLife: 100, Mana: 50, MaxMana: 50}, Stamina: 80, MaxStamina: 80}
	tests := []struct {
		name         string
		in           WellVitals
		parm1, parm3 int
		want         WellVitals
		changed      bool
	}{
		{"half of max, all three", WellVitals{Vitals{10, 100, 0, 60}, 0, 90}, 128, 3,
			WellVitals{Vitals{60, 100, 30, 60}, 45, 90}, true},
		{"capped at max", WellVitals{Vitals{90, 100, 55, 60}, 85, 90}, 128, 3,
			WellVitals{Vitals{100, 100, 60, 60}, 90, 90}, true},
		{"life flag only (stamina always)", WellVitals{Vitals{10, 100, 0, 60}, 0, 90}, 128, 2,
			WellVitals{Vitals{60, 100, 0, 60}, 45, 90}, true},
		{"mana flag only", WellVitals{Vitals{10, 100, 0, 60}, 90, 90}, 128, 1,
			WellVitals{Vitals{10, 100, 30, 60}, 90, 90}, true},
		{"nothing to heal", full, 128, 3, full, false},
		{"parm1 64 is a quarter", WellVitals{Vitals{0, 100, 0, 0}, 0, 0}, 64, 3,
			WellVitals{Vitals{25, 100, 0, 0}, 0, 0}, true},
	}

	for _, tc := range tests {
		got, ch := WellPulse(tc.in, tc.parm1, tc.parm3)
		if got != tc.want || ch != tc.changed {
			t.Errorf("%s: got %+v changed=%v, want %+v changed=%v", tc.name, got, ch, tc.want, tc.changed)
		}
	}
}

func TestInfoFor(t *testing.T) {
	tests := []struct {
		def   Def
		class Class
	}{
		{Def{OperateFn: 30}, ClassExplode},
		{Def{OperateFn: 30, SubClass: SubTrap}, ClassTrap},
		{Def{OperateFn: 39}, ClassLoot}, {Def{OperateFn: 41}, ClassLoot}, {Def{OperateFn: 57}, ClassLoot},
		{Def{OperateFn: 59}, ClassLoot}, {Def{OperateFn: 51}, ClassLoot}, {Def{OperateFn: 68}, ClassLoot},
		{Def{OperateFn: 52}, ClassQuest}, {Def{OperateFn: 22}, ClassWell}, {Def{OperateFn: 32}, ClassStash},
	}

	for _, tc := range tests {
		if got := InfoFor(tc.def).Class; got != tc.class {
			t.Errorf("fn %d sub %d: %v want %v", tc.def.OperateFn, tc.def.SubClass, got, tc.class)
		}
	}
}

// TestObjectsTableAudit walks every row of the extracted Objects.txt (D2_TABLES) and checks the
// behaviours the engine attaches to it.
func TestObjectsTableAudit(t *testing.T) {
	dir := os.Getenv("D2_TABLES")
	if dir == "" {
		t.Skip("D2_TABLES not set")
	}

	data, err := os.ReadFile(filepath.Join(dir, "objects", "patch_d2", "Objects.txt"))
	if err != nil {
		t.Skip(err)
	}

	defs, err := ParseObjects(data)
	if err != nil {
		t.Fatal(err)
	}

	wp := map[int]int{}
	shrines, wells := 0, 0

	for _, d := range defs {
		info := InfoFor(d)
		if strings.Contains(info.Note, "not mapped") {
			t.Errorf("object %d %q: OperateFn %d is not mapped in operateInfos", d.ID, d.Name, d.OperateFn)
		}

		switch info.Class {
		case ClassShrine:
			shrines++

			// Parm0 selects the shrine list: 0 uniform (exe path), 1..3 class lists
			if d.Parm0 < 0 || d.Parm0 > 3 {
				t.Errorf("shrine object %d: Parm0 %d outside 0..3", d.ID, d.Parm0)
			}

			for lvl := 1; lvl <= 110; lvl += 9 {
				var ok bool
				if d.Parm0 == 0 {
					_, ok = RollShrineUniform(DefaultShrines, NewRoller(uint32(d.ID)), lvl)
				} else {
					_, ok = RollShrine(DefaultShrines, NewRoller(uint32(d.ID)), lvl, d.Parm0, false)
				}

				if !ok {
					t.Errorf("shrine object %d (Parm0 %d) has no shrine at level %d", d.ID, d.Parm0, lvl)
				}
			}
		case ClassWell:
			wells++

			if d.Parm0 != 750 || d.Parm1 != 128 || d.Parm3 != 3 {
				t.Errorf("well %d: parms %d/%d/%d, expected 750/128/3", d.ID, d.Parm0, d.Parm1, d.Parm3)
			}
		case ClassLoot:
			if d.IsChest() && d.TrapProb != 15 {
				t.Errorf("chest %d: trap=%d", d.ID, d.TrapProb)
			}
		}

		if d.IsWaypoint() {
			for lvl := 1; lvl < 140; lvl++ {
				if _, ok := WaypointBitOf(d, lvl, true); ok {
					wp[d2level.ActOfLevel(lvl)]++
				}
			}
		}
	}

	if shrines < 40 || wells < 10 {
		t.Errorf("shrines %d wells %d", shrines, wells)
	}

	for act := 1; act <= 5; act++ {
		if wp[act] == 0 {
			t.Errorf("no waypoint object matches a waypoint level of act %d", act)
		}
	}
}
