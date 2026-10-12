package d2level

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestServerTravelCain(t *testing.T) {
	r, ok := ServerTravel(ClassCain6)
	if !ok || r.From != 5 || r.To != 4 || r.QuestSlot != -1 || r.Via != ViaServer {
		t.Fatalf("Cain(6) rule = %+v ok=%v", r, ok)
	}

	if _, ok := ServerTravel(ClassTyrael2); ok {
		t.Error("Tyrael has a client row, not a server-only action")
	}

	// the server-only action is not an entry of the forward/back rule list
	if _, ok := RuleFor(5, 4); ok {
		t.Error("Act 5 -> 4 must stay outside ActRules")
	}
}

func TestTyraelRowAndRule(t *testing.T) {
	r, ok := RuleForNPC(ClassTyrael2)
	if !ok || r.Row != "Travel To Harrogath" || !r.NeedExpansion || r.To != 5 {
		t.Fatalf("Tyrael rule = %+v ok=%v", r, ok)
	}
}

func TestVisSlotDestinationRepeatedWarps(t *testing.T) {
	cases := []struct{ level, slot, want int }{
		{10, 0, 4}, {10, 1, 5}, {10, 4, 14},
		{31, 0, 30}, {31, 1, 32},
		{92, 0, 80}, {92, 2, 81}, {92, 3, 81}, {92, 4, 93},
	}

	for _, c := range cases {
		if got, ok := TileDestination(c.level, c.slot); !ok || got != c.want {
			t.Errorf("TileDestination(%d, %d) = %d,%v want %d", c.level, c.slot, got, ok, c.want)
		}
	}

	if _, ok := VisSlotDestination(92, 5); ok {
		t.Error("an empty slot leads nowhere")
	}
}

// TestRepeatedWarpSlotsMatchLevelsTxt reads the Vis columns of a real Levels.txt (D2_TABLES).
func TestRepeatedWarpSlotsMatchLevelsTxt(t *testing.T) {
	dir := os.Getenv("D2_TABLES")
	if dir == "" {
		t.Skip("D2_TABLES not set")
	}

	raw, err := os.ReadFile(filepath.Join(dir, "drlg", "patch_d2", "Levels.txt"))
	if err != nil {
		t.Skip(err)
	}

	lines := strings.Split(strings.ReplaceAll(string(raw), "\r", ""), "\n")
	idx := map[string]int{}

	for i, c := range strings.Split(lines[0], "\t") {
		idx[c] = i
	}

	num := func(s string) int {
		n, _ := strconv.Atoi(s)

		return n
	}

	found := map[int]bool{}

	for _, line := range lines[1:] {
		f := strings.Split(line, "\t")
		if len(f) < len(idx) {
			continue
		}

		lvl := num(f[idx["Id"]])

		slots, ok := repeatedWarpSlots[lvl]
		if !ok {
			continue
		}

		found[lvl] = true

		for k := 0; k < 8; k++ {
			if want, got := num(f[idx["Vis"+strconv.Itoa(k)]]), slots[k]; got != want {
				t.Errorf("level %d Vis%d = %d, table %d", lvl, k, want, got)
			}
		}
	}

	if len(found) != len(repeatedWarpSlots) {
		t.Errorf("only %d of %d levels found in Levels.txt", len(found), len(repeatedWarpSlots))
	}
}
