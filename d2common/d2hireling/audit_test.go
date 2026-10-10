package d2hireling

import (
	"fmt"
	"testing"
)

// Audit tests: rules of hirelings.md pinned against the real hireling.txt
// (skipped without D2_TABLES). V = verified in the notes, U = unverified.

// TestAuditOfferMatrix pins which merc types each hire NPC offers per
// difficulty (the SubType variants of the first band), their act, class,
// base level, base gold and name range. V from the table + HIRE_FindRecord*.
func TestAuditOfferMatrix(t *testing.T) {
	tab := loadReal(t)

	tests := []struct {
		seller, diff     int
		ids              []int
		class, act       int
		level, gold, nms int
	}{
		{150, 1, []int{0, 1}, 271, 1, 3, 100, 41}, // Kashya: rogue archers (fire, cold)
		{150, 2, []int{2, 3}, 271, 1, 36, 6000, 41},
		{150, 3, []int{4, 5}, 271, 1, 67, 12500, 41},
		{198, 1, []int{6, 7, 8}, 338, 2, 9, 350, 21}, // Greiz: desert guards (combat, defensive, offensive)
		{198, 2, []int{9, 10, 11}, 338, 2, 43, 7900, 21},
		{198, 3, []int{12, 13, 14}, 338, 2, 75, 15000, 21},
		{252, 1, []int{15, 16, 17}, 359, 3, 15, 1000, 20}, // Asheara: Iron Wolves (fire, cold, lightning)
		{252, 2, []int{18, 19, 20}, 359, 3, 49, 9500, 20},
		{252, 3, []int{21, 22, 23}, 359, 3, 79, 21000, 20},
		{515, 1, []int{24, 25}, 561, 5, 28, 9000, 67}, // Qual-Kehk: barbarians
		{515, 2, []int{26, 27}, 561, 5, 58, 18000, 67},
		{515, 3, []int{28, 29}, 561, 5, 80, 32000, 67},
	}

	for _, tc := range tests {
		c := tab.Candidates(tc.seller, tc.diff)

		var ids []int
		for _, r := range c {
			ids = append(ids, r.ID)
		}

		if fmt.Sprint(ids) != fmt.Sprint(tc.ids) {
			t.Errorf("seller %d diff %d: ids %v, want %v", tc.seller, tc.diff, ids, tc.ids)
			continue
		}

		r := c[0]
		if r.Class != tc.class || r.Act != tc.act || r.Level != tc.level || r.Gold != tc.gold || NameCount(r) != tc.nms ||
			r.Difficulty != tc.diff {
			t.Errorf("seller %d diff %d: %+v", tc.seller, tc.diff, r)
		}
	}

	if got := tab.Sellers(); fmt.Sprint(got) != "[150 198 252 515]" {
		t.Errorf("sellers = %v", got)
	}
}

// TestAuditExpLevelRoundTrip: for every Id and level 1..99 the start
// experience gives the level back and one point less gives the level below.
func TestAuditExpLevelRoundTrip(t *testing.T) {
	tab := loadReal(t)

	for id := 0; id < 30; id++ {
		for l := 1; l <= MaxLevel; l++ {
			e := tab.StartExp(id, l)
			if got := tab.LevelFromExp(id, e); got != l {
				t.Fatalf("id %d level %d: LevelFromExp(%d) = %d", id, l, e, got)
			}

			if l > 1 {
				if got := tab.LevelFromExp(id, e-1); got != l-1 {
					t.Fatalf("id %d level %d: exp-1 -> %d", id, l, got)
				}
			}
		}
	}
}

// TestAuditStatGrowth: growth is monotonic in level for every Id (also across
// the level bands) and respects the floors (HP 40, Str/Dex 10).
func TestAuditStatGrowth(t *testing.T) {
	tab := loadReal(t)

	for id := 0; id < 30; id++ {
		var prev Stats

		for l := 1; l <= MaxLevel; l++ {
			s, rec := tab.StatsFor(id, l)
			if rec == nil || rec.ID != id {
				t.Fatalf("id %d level %d: no row", id, l)
			}

			if s.MaxHP < minHP || s.Str < minStrDex || s.Dex < minStrDex || s.DmgMax < 1 || s.DmgMin > s.DmgMax {
				t.Fatalf("id %d level %d: floors %+v", id, l, s)
			}

			if l > 1 && l < MaxLevel && s.NextXP < prev.NextXP {
				t.Errorf("id %d level %d: next xp shrank", id, l)
			}

			prev = s
		}
	}
}

// TestAuditBandSelection: the row valid at a level is the one with the highest
// base Level <= level (V), the first row below the first band.
func TestAuditBandSelection(t *testing.T) {
	tab := loadReal(t)

	tests := []struct{ id, level, want int }{
		{0, 1, 3}, {0, 3, 3}, {11, 42, 43}, {11, 43, 43}, {11, 74, 43}, {11, 75, 75}, {11, 99, 75},
		{23, 78, 79}, {23, 79, 79}, {29, 99, 80},
	}

	// expansion rows only change band at Level; the id 0 row set is 3/49/79 (see the d2exp table)
	for _, tc := range tests {
		r := tab.Find(tc.id, tc.level)
		if r == nil || r.ID != tc.id {
			t.Fatalf("Find(%d,%d) = %v", tc.id, tc.level, r)
		}

		if tc.id != 0 && r.Level != tc.want {
			t.Errorf("Find(%d,%d) band %d, want %d", tc.id, tc.level, r.Level, tc.want)
		}
	}
}

// TestAuditCosts pins hire price (max(Gold, (15*delta+100)*Gold/100)), offer
// level (owner level -5..-1, min 2) and revive price (min(50000, lvl^2/2*15)).
func TestAuditCosts(t *testing.T) {
	tab := loadReal(t)
	kashya := tab.Candidates(150, 1)[0] // gold 100, base level 3
	greiz := tab.Candidates(198, 1)[0]  // gold 350, base level 9

	hire := []struct {
		name  string
		r     *Record
		level int
		want  int
	}{
		{"below base is the base price", kashya, 2, 100},
		{"base", kashya, 3, 100},
		{"+1", kashya, 4, 115},
		{"+20", kashya, 23, 400},
		{"desert base", greiz, 9, 350},
		{"desert +10", greiz, 19, 875}, // 350*250/100
		{"desert below base", greiz, 2, 350},
	}

	for _, tc := range hire {
		if got := HireCost(tc.r, tc.level); got != tc.want {
			t.Errorf("%s: HireCost = %d, want %d", tc.name, got, tc.want)
		}
	}

	for _, tc := range []struct{ owner, roll, want int }{
		{1, 0, 2}, {1, 4, 2}, {6, 0, 2}, {7, 0, 2}, {8, 0, 3}, {30, 0, 25}, {30, 4, 29}, {99, 4, 98},
	} {
		if got := OfferLevel(tc.owner, tc.roll); got != tc.want {
			t.Errorf("OfferLevel(%d,%d) = %d, want %d", tc.owner, tc.roll, got, tc.want)
		}
	}

	for _, tc := range []struct{ level, want int }{
		{1, 0}, {2, 30}, {3, 60}, {10, 750}, {40, 12000}, {81, 49200}, {82, 50000}, {99, 50000},
	} {
		if got := ReviveCost(tc.level); got != tc.want {
			t.Errorf("ReviveCost(%d) = %d, want %d", tc.level, got, tc.want)
		}
	}
}

// TestAuditSkills: skill slots are contiguous (the choice stops at the first
// empty slot), modes are the known ones, skill levels stay within 1..32 at
// every merc level.
func TestAuditSkills(t *testing.T) {
	tab := loadReal(t)

	for _, r := range tab.Rows {
		empty := false

		for i, s := range r.Skills {
			if !s.Used() {
				empty = true

				continue
			}

			if empty {
				t.Errorf("id %d level %d: skill %d after an empty slot", r.ID, r.Level, i+1)
			}

			switch s.Mode {
			case 1, 4, 5, 7, 14:
			default:
				t.Errorf("id %d: skill %q mode %d", r.ID, s.Name, s.Mode)
			}

			for l := 1; l <= MaxLevel; l++ {
				if got := r.SkillLevel(i, l); got < 1 || got > 32 {
					t.Fatalf("id %d skill %q level %d -> %d", r.ID, s.Name, l, got)
				}
			}
		}
	}

	// rogue fire (id 0), first band (row level 3): Fire Arrow = Level + (10*d)>>5
	r := tab.Find(0, 3)

	for i, s := range r.Skills {
		if s.Name != "Fire Arrow" {
			continue
		}

		for _, lvl := range []int{3, 10, 24} {
			if got, want := r.SkillLevel(i, lvl), maxInt(1, s.Level+(s.LvlPerLvl*(lvl-r.Level))>>5); got != want {
				t.Errorf("Fire Arrow at %d = %d, want %d", lvl, got, want)
			}
		}
	}
}
