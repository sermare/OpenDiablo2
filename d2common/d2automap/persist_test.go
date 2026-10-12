package d2automap

import (
	"os"
	"path/filepath"
	"testing"
)

func sample() *Model {
	m := NewModel()
	m.Add(LayerFloor, 16, -8, 100)
	m.Add(LayerFloor, 24, 4, 101)
	m.Add(LayerWall, -40, 12, 7)
	m.Add(LayerObject, 5, 6, 309)

	return m
}

func equalCells(a, b *Model) bool {
	x, y := a.Cells(), b.Cells()
	if len(x) != len(y) {
		return false
	}

	for i := range x {
		if x[i] != y[i] {
			return false
		}
	}

	return true
}

func TestSidecarRoundTrip(t *testing.T) {
	sc := Sidecar{Dir: t.TempDir(), Name: "Hero", GameID: 77}
	if err := sc.SaveLevels(map[int]*Model{1: sample(), 109: sample()}); err != nil {
		t.Fatal(err)
	}

	for _, lvl := range []int{1, 109} {
		got := sc.LoadLevel(lvl)
		if got == nil || !equalCells(got, sample()) {
			t.Fatalf("level %d did not round trip: %v", lvl, got)
		}
	}

	if sc.LoadLevel(2) != nil {
		t.Fatal("level 2 was never stored")
	}

	// saving another level keeps the first ones
	if err := sc.SaveLevels(map[int]*Model{2: sample()}); err != nil {
		t.Fatal(err)
	}

	if sc.LoadLevel(1) == nil || sc.LoadLevel(2) == nil {
		t.Fatal("merge lost a level")
	}
}

func TestSidecarGameIDEviction(t *testing.T) {
	dir := t.TempDir()
	a := Sidecar{Dir: dir, Name: "H", GameID: 1}
	_ = a.SaveLevels(map[int]*Model{1: sample()})

	b := Sidecar{Dir: dir, Name: "H", GameID: 2}
	if b.LoadLevel(1) != nil {
		t.Fatal("another game must not see the map")
	}

	_ = b.SaveLevels(map[int]*Model{1: NewModel()})

	// both games live in the ring until four more distinct ids evict the oldest
	if a.LoadLevel(1) == nil {
		t.Fatal("game 1 was evicted too early")
	}

	for id := uint32(3); id <= 5; id++ {
		_ = Sidecar{Dir: dir, Name: "H", GameID: id}.SaveLevels(map[int]*Model{1: sample()})
	}

	if a.LoadLevel(1) != nil {
		t.Fatal("game 1 should be evicted by the 5th game")
	}
}

func TestSidecarCorruptAndMissing(t *testing.T) {
	dir := t.TempDir()
	sc := Sidecar{Dir: dir, Name: "H", GameID: 9}

	if sc.LoadLevel(1) != nil {
		t.Fatal("missing files")
	}

	if err := sc.SaveLevels(map[int]*Model{1: sample(), 2: sample()}); err != nil {
		t.Fatal(err)
	}

	slot := sc.slotPath(0)

	good, err := os.ReadFile(slot)
	if err != nil {
		t.Fatal(err)
	}

	garbage := make([]byte, len(good))
	for i := range garbage {
		garbage[i] = 0xff
	}

	for name, data := range map[string][]byte{"empty": nil, "short": good[:50], "garbage": garbage, "truncated": good[:len(good)-5]} {
		if err := os.WriteFile(slot, data, 0o644); err != nil {
			t.Fatal(err)
		}

		l1, l2 := sc.LoadLevel(1), sc.LoadLevel(2) // must not panic

		if name != "truncated" && (l1 != nil || l2 != nil) {
			t.Errorf("%s: loaded cells from a corrupt file", name)
		}

		if name == "truncated" && l2 != nil {
			t.Errorf("truncated: the cut level must be dropped")
		}
	}

	// a corrupt header resets the ring; saving works again
	if err := os.WriteFile(sc.mapPath(), []byte("xx"), 0o644); err != nil {
		t.Fatal(err)
	}

	if sc.LoadLevel(1) != nil {
		t.Fatal("corrupt header")
	}

	if err := sc.SaveLevels(map[int]*Model{1: sample()}); err != nil || sc.LoadLevel(1) == nil {
		t.Fatalf("recovery failed: %v", err)
	}

	left, _ := filepath.Glob(filepath.Join(dir, ".*.tmp*"))
	if len(left) != 0 {
		t.Fatalf("temp files left: %v", left)
	}
}

func TestSidecarInvalid(t *testing.T) {
	if (Sidecar{}).SaveLevels(nil) == nil || (Sidecar{}).LoadLevel(1) != nil {
		t.Fatal("empty sidecar must refuse")
	}
}
