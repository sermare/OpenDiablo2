package d2object

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2level"
)

const objHdr = "Name\tSpawnMax\tTrapProb\tAct\tSubClass\tOperateFn\tPopulateFn\tInitFn\tOperateRange\tLockable\tRestoreVirgins\tParm0\tDamage\tSelectable0\n"

func testDefs(t *testing.T) []Def {
	t.Helper()

	data := objHdr +
		"Dummy\t0\t0\t0\t0\t0\t0\t0\t0\t0\t0\t0\t0\t0\n" +
		"chest\t2\t15\t1\t8\t4\t3\t3\t2\t1\t1\t18\t0\t1\n" +
		"Waypoint\t0\t0\t8\t64\t23\t0\t17\t2\t0\t0\t0\t0\t1\n" +
		"x5chest\t1\t15\t16\t8\t4\t3\t3\t2\t1\t1\t20\t0\t1\n" +
		"well\t0\t0\t15\t32\t22\t8\t16\t2\t0\t0\t750\t0\t1\n"

	defs, err := ParseObjects([]byte(data))
	if err != nil {
		t.Fatal(err)
	}

	return defs
}

func TestParseAndAllowed(t *testing.T) {
	defs := testDefs(t)
	if len(defs) != 5 || defs[1].Name != "chest" || !defs[1].Lockable || defs[1].TrapProb != 15 {
		t.Fatalf("bad parse %+v", defs)
	}

	tests := []struct {
		id        int
		act       int
		expansion bool
		want      bool
	}{
		{1, 1, false, true}, {1, 2, false, false}, {3, 5, true, true}, {3, 5, false, false},
		{4, 3, false, true}, {2, 4, false, true}, {2, 1, false, false}, {0, 1, true, false},
	}

	for _, tc := range tests {
		if got := defs[tc.id].AllowedIn(tc.act, tc.expansion); got != tc.want {
			t.Errorf("def %d act %d exp %v: %v want %v", tc.id, tc.act, tc.expansion, got, tc.want)
		}
	}

	if _, err := ParseObjects([]byte("Name\n")); err == nil {
		t.Error("expected error for missing columns")
	}
}

func TestRollRoom(t *testing.T) {
	defs := testDefs(t)
	g := Group{Name: "g", Offset: 1}
	g.Members[0] = GroupMember{1, 5000, 100} // chest, capped by SpawnMax 2
	g.Members[1] = GroupMember{3, 100, 100}  // act 5 chest: filtered in act 1
	g.Members[2] = GroupMember{1, 1, 0}      // prob 0 never spawns

	got := RollRoom(g, defs, 1, true, 100, NewRoller(1))
	if len(got) != 2 {
		t.Fatalf("placements %v want 2 chests", got)
	}

	// deterministic
	a := RollRoom(g, defs, 1, true, 100, NewRoller(7))
	b := RollRoom(g, defs, 1, true, 100, NewRoller(7))

	if len(a) != len(b) {
		t.Error("not deterministic")
	}

	// well group: exactly one member
	w := Group{Wells: true}
	w.Members[0] = GroupMember{4, 0, 50}
	w.Members[1] = GroupMember{4, 0, 50}

	if got := RollRoom(w, defs, 2, false, 50, NewRoller(3)); len(got) != 1 || got[0].ObjectID != 4 {
		t.Errorf("well room %v", got)
	}

	if SpawnCount(1, 10, 0) != 1 || SpawnCount(0, 10, 0) != 0 || SpawnCount(500, 100, 3) != 3 {
		t.Error("SpawnCount")
	}
}

func TestPickGroup(t *testing.T) {
	groups := [8]int{1, 2, 3}
	probs := [8]int{50, 50}
	seen := map[int]int{}

	r := NewRoller(9)
	for i := 0; i < 1000; i++ {
		seen[PickGroup(groups, probs, r)]++
	}

	if seen[1] < 400 || seen[2] < 400 || seen[0]+seen[3] != 0 {
		t.Errorf("distribution %v", seen)
	}

	if PickGroup(groups, [8]int{}, r) != 0 {
		t.Error("no probability must give none")
	}
}

func TestChest(t *testing.T) {
	defs := testDefs(t)
	r := NewRoller(5)

	st := NewChest(defs[1], r)
	if !st.Locked {
		t.Error("lockable chest should start locked")
	}

	var gotTC string

	h := ChestHooks{
		TreasureClass: func(d Def, act, diff, lvl int) string { return "Act 1 Chest A" },
		Drop:          func(tc string, lvl int, seed uint32) int { gotTC = tc; return int(seed) },
	}

	st.Trap = true

	res, err := Open(defs[1], &st, h, 1, 0, 5, 3)
	if err != nil || res.Dropped != 3 || gotTC != "Act 1 Chest A" || !res.Trapped || st.Locked || st.Trap {
		t.Fatalf("open: %+v %v %+v", res, err, st)
	}

	if _, err := Open(defs[1], &st, h, 1, 0, 5, 3); err != ErrAlreadyOpened {
		t.Errorf("second open: %v", err)
	}

	// no hooks: safe
	st2 := ChestState{}
	if _, err := Open(defs[1], &st2, ChestHooks{}, 1, 0, 1, 1); err != nil {
		t.Error(err)
	}

	// trap rate over many chests tracks TrapProb
	traps := 0

	for i := 0; i < 2000; i++ {
		if NewChest(defs[1], r).Trap {
			traps++
		}
	}

	if traps < 200 || traps > 400 {
		t.Errorf("trap rate %d/2000 for 15%%", traps)
	}
}

func TestWaypointBitOf(t *testing.T) {
	defs := testDefs(t)

	tests := []struct {
		def   int
		level int
		ok    bool
	}{
		{2, 103, true},  // Pandemonium Fortress, act 4 (mask 8)
		{2, 1, false},   // act 1 town: object not allowed in act 1
		{1, 103, false}, // not a waypoint
		{2, 104, false}, // act 4 level without waypoint
	}

	for _, tc := range tests {
		bit, ok := WaypointBitOf(defs[tc.def], tc.level, false)
		if ok != tc.ok {
			t.Errorf("def %d level %d: ok=%v want %v", tc.def, tc.level, ok, tc.ok)
		}

		if ok && d2level.WaypointLevel(bit) != tc.level {
			t.Errorf("bit %d does not map back to %d", bit, tc.level)
		}
	}
}

// TestRealObjects checks the extracted 1.14b tables when D2_TABLES is set.
func TestRealObjects(t *testing.T) {
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

	if len(defs) != 574 {
		t.Fatalf("rows %d want 574", len(defs))
	}

	// every waypoint object maps to a real waypoint level in its own act
	n := 0

	for _, d := range defs {
		if !d.IsWaypoint() {
			continue
		}

		n++

		found := false

		for bit := 0; bit < 39; bit++ {
			lvl := d2level.WaypointLevel(bit)
			if _, ok := WaypointBitOf(d, lvl, true); ok {
				found = true
			}
		}

		if !found && d.Act != 0 {
			t.Errorf("waypoint object %d (act mask %d) matches no waypoint level", d.ID, d.Act)
		}
	}

	if n < 10 {
		t.Errorf("only %d waypoint objects", n)
	}

	// chests are lockable containers with a trap chance; wells use OperateFn 22
	chests := 0

	for _, d := range defs {
		if d.IsChest() && d.Lockable {
			chests++

			if d.TrapProb != 15 {
				t.Errorf("chest %d trapprob %d", d.ID, d.TrapProb)
			}
		}
	}

	if chests < 20 {
		t.Errorf("only %d lockable chests", chests)
	}

	if g, err := os.ReadFile(filepath.Join(dir, "objects", "objgroup.txt")); err == nil {
		groups, err := ParseGroups(g)
		if err != nil || len(groups) < 100 {
			t.Fatalf("groups %d %v", len(groups), err)
		}

		for _, gr := range groups {
			for _, m := range gr.Members {
				if m.ID >= len(defs) {
					t.Errorf("group %q references object %d", gr.Name, m.ID)
				}
			}
		}
	}
}
