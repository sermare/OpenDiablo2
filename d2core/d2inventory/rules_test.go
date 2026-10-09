package d2inventory

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// tsv reads a tab separated game table from D2_TABLES; skips when unset.
func tsv(t *testing.T, rel string) (header []string, rows [][]string) {
	t.Helper()

	root := os.Getenv("D2_TABLES")
	if root == "" {
		t.Skip("D2_TABLES not set")
	}

	f, err := os.Open(filepath.Join(root, rel))
	if err != nil {
		t.Skipf("table %s: %v", rel, err)
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)

	for sc.Scan() {
		cells := strings.Split(strings.TrimRight(sc.Text(), "\r"), "\t")
		if header == nil {
			header = cells

			continue
		}

		rows = append(rows, cells)
	}

	return header, rows
}

func col(t *testing.T, header []string, name string) int {
	t.Helper()

	for i, h := range header {
		if strings.EqualFold(h, name) {
			return i
		}
	}

	t.Fatalf("column %q missing", name)

	return -1
}

func cell(r []string, i int) string {
	if i < len(r) {
		return r[i]
	}

	return ""
}

func TestDims(t *testing.T) {
	tests := []struct {
		c    Container
		exp  bool
		w, h int
	}{
		{ContainerInventory, false, 10, 4},
		{ContainerInventory, true, 10, 4},
		{ContainerTrade, true, 10, 4},
		{ContainerCube, true, 3, 4},
		{ContainerStash, false, 6, 4},
		{ContainerStash, true, 6, 8},
		{ContainerVendor, true, 10, 10},
	}

	for _, tt := range tests {
		if w, h := Dims(tt.c, tt.exp); w != tt.w || h != tt.h {
			t.Errorf("Dims(%d,%v)=%dx%d want %dx%d", tt.c, tt.exp, w, h, tt.w, tt.h)
		}
	}
}

// TestDimsMatchInventoryTxt pins Dims to the real Inventory.txt rows.
func TestDimsMatchInventoryTxt(t *testing.T) {
	h, rows := tsv(t, "itemgen/patch_d2/Inventory.txt")
	cls, gx, gy := col(t, h, "class"), col(t, h, "gridX"), col(t, h, "gridY")

	want := map[string]struct {
		c   Container
		exp bool
	}{
		"Amazon":                  {ContainerInventory, false},
		"Druid":                   {ContainerInventory, true},
		"Assassin":                {ContainerInventory, true},
		"Trade Page 1":            {ContainerTrade, false},
		"Trade Page 2":            {ContainerTrade, false},
		"Bank Page 1":             {ContainerStash, false},
		"Big Bank Page 1":         {ContainerStash, true},
		"Transmogrify Box Page 1": {ContainerCube, false},
		"Monster":                 {ContainerVendor, false},
	}

	seen := 0

	for _, r := range rows {
		w, ok := want[cell(r, cls)]
		if !ok {
			continue
		}

		seen++

		x, _ := strconv.Atoi(cell(r, gx))
		y, _ := strconv.Atoi(cell(r, gy))

		if gw, gh := Dims(w.c, w.exp); gw != x || gh != y {
			t.Errorf("%s: table %dx%d, Dims %dx%d", cell(r, cls), x, y, gw, gh)
		}
	}

	if seen != len(want) {
		t.Errorf("found %d of %d rows", seen, len(want))
	}
}

// TestItemSizesInTables checks the invwidth/invheight of well known items and
// that every base item footprint is within 1..4 cells.
func TestItemSizesInTables(t *testing.T) {
	known := map[string][2]int{
		"hp1": {1, 1}, "rin": {1, 1}, "amu": {1, 1}, "box": {2, 2}, "cap": {2, 2},
		"lsd": {2, 3}, "tkf": {1, 2}, "hax": {1, 3},
	}
	found := map[string]bool{}

	for _, f := range []string{"armor", "weapons", "misc"} {
		h, rows := tsv(t, "patch_d2/"+f+".txt")
		code, iw, ih := col(t, h, "code"), col(t, h, "invwidth"), col(t, h, "invheight")

		for _, r := range rows {
			c := strings.TrimSpace(cell(r, code))
			if c == "" {
				continue
			}

			w, _ := strconv.Atoi(cell(r, iw))
			hh, _ := strconv.Atoi(cell(r, ih))

			if w < 1 || w > 4 || hh < 1 || hh > 4 {
				t.Errorf("%s %q: footprint %dx%d outside 1..4", f, c, w, hh)
			}

			if k, ok := known[c]; ok {
				found[c] = true

				if k != [2]int{w, hh} {
					t.Errorf("%s %q: %dx%d want %dx%d", f, c, w, hh, k[0], k[1])
				}
			}
		}
	}

	for c := range known {
		if !found[c] {
			t.Logf("reference item %q not in the tables", c)
		}
	}
}

// TestEveryItemFitsCube: the largest footprint fits the smallest grid (cube).
func TestEveryItemFitsCube(t *testing.T) {
	cw, ch := Dims(ContainerCube, true)

	for _, f := range []string{"armor", "weapons", "misc"} {
		h, rows := tsv(t, "patch_d2/"+f+".txt")
		iw, ih := col(t, h, "invwidth"), col(t, h, "invheight")

		for _, r := range rows {
			w, _ := strconv.Atoi(cell(r, iw))
			hh, _ := strconv.Atoi(cell(r, ih))

			if w > cw || hh > ch {
				t.Errorf("%s: %dx%d does not fit the %dx%d cube", f, w, hh, cw, ch)
			}
		}
	}
}

func TestClassifyDrop(t *testing.T) {
	tests := []struct {
		n    int
		want DropOutcome
	}{{0, DropPlace}, {1, DropSwap}, {2, DropRefuse}, {5, DropRefuse}}

	for _, tt := range tests {
		if got := ClassifyDrop(tt.n); got != tt.want {
			t.Errorf("ClassifyDrop(%d)=%d want %d", tt.n, got, tt.want)
		}
	}
}

func TestMergeStacks(t *testing.T) {
	tests := []struct {
		name                string
		onGrid, held, max   int
		wantStack, wantRest int
	}{
		{"fits", 10, 5, 60, 15, 0},
		{"exact", 55, 5, 60, 60, 0},
		{"overflow", 58, 5, 60, 60, 3},
		{"full already", 60, 5, 60, 60, 5},
		{"no limit known", 5, 5, 0, 5, 5},
	}

	for _, tt := range tests {
		if s, r := MergeStacks(tt.onGrid, tt.held, tt.max); s != tt.wantStack || r != tt.wantRest {
			t.Errorf("%s: got (%d,%d) want (%d,%d)", tt.name, s, r, tt.wantStack, tt.wantRest)
		}
	}
}

func TestSplitStack(t *testing.T) {
	tests := []struct{ qty, n, left, taken int }{
		{10, 3, 7, 3}, {10, 0, 9, 1}, {10, 10, 1, 9}, {2, 5, 1, 1}, {1, 1, 1, 0},
	}

	for _, tt := range tests {
		if l, k := SplitStack(tt.qty, tt.n); l != tt.left || k != tt.taken {
			t.Errorf("SplitStack(%d,%d)=(%d,%d) want (%d,%d)", tt.qty, tt.n, l, k, tt.left, tt.taken)
		}
	}
}

func TestGoldLimits(t *testing.T) {
	if InventoryGoldLimit(1) != 10000 || InventoryGoldLimit(94) != 940000 || InventoryGoldLimit(0) != 10000 {
		t.Error("inventory gold limit is 10000 per level")
	}

	tests := []struct {
		have, add, limit, total, over int
	}{
		{0, 500, 10000, 500, 0},
		{9500, 500, 10000, 10000, 0},
		{9500, 800, 10000, 10000, 300},
		{10000, 1, 10000, 10000, 1},
		{2499999, 5, StashGoldLimit, StashGoldLimit, 4},
		{5, -3, 10000, 5, 0},
	}

	for _, tt := range tests {
		if g, o := AddGold(tt.have, tt.add, tt.limit); g != tt.total || o != tt.over {
			t.Errorf("AddGold(%d,%d,%d)=(%d,%d) want (%d,%d)", tt.have, tt.add, tt.limit, g, o, tt.total, tt.over)
		}
	}
}

// TestVerifiedRulesPinned pins the rules read from Game.exe 1.14b: max stack
// (0x6297b0), merge surplus (0x55c3c0), the empty unstack handler (0x55c7f0),
// the gold caps (0x623050, 0x623640) and the gold setter (0x53dc10).
func TestVerifiedRulesPinned(t *testing.T) {
	if MaxStack(60, 0) != 60 || MaxStack(250, 400) != 511 || MaxStack(100, 20) != 120 {
		t.Error("MaxStack = base + stat 0xfe clamped to 511")
	}

	if s, r := MergeStacks(70, 5, 60); s != 60 || r != 15 {
		t.Errorf("over-full grid stack is clamped and the rest stays on the cursor: %d %d", s, r)
	}

	if MergeDurability(20, 12) != 12 || MergeDurability(10, 12) != 10 {
		t.Error("merge keeps the lower durability")
	}

	if UnstackSupported {
		t.Error("packet 0x22 handler is a stub in 1.14b")
	}

	if StashGoldLimit != 2500000 || InventoryGoldPerLevel != 10000 {
		t.Error("gold caps")
	}

	tests := []struct{ have, delta, limit, want int }{
		{100, 50, 10000, 150}, {100, -101, 10000, 0}, {9990, 11, 10000, 0}, {9990, 10, 10000, 10000},
	}

	for _, tt := range tests {
		if got := SetGold(tt.have, tt.delta, tt.limit); got != tt.want {
			t.Errorf("SetGold(%d,%d,%d)=%d want %d", tt.have, tt.delta, tt.limit, got, tt.want)
		}
	}

	if SanitizeLoadedGold(-1, 100) != 0 || SanitizeLoadedGold(101, 100) != 0 || SanitizeLoadedGold(100, 100) != 100 {
		t.Error("SanitizeLoadedGold")
	}
}

// TestBeltableAreOnlyPotions pins the belt rule: only ItemTypes rows with the
// Beltable flag go on the belt: potion types, elixirs and scrolls.
func TestBeltableAreOnlyPotions(t *testing.T) {
	h, rows := tsv(t, "patch_d2/ItemTypes.txt")
	code, bl := col(t, h, "Code"), col(t, h, "Beltable")

	beltable := map[string]bool{}

	for _, r := range rows {
		if cell(r, bl) == "1" {
			beltable[cell(r, code)] = true
		}
	}

	t.Logf("beltable types: %v", beltable)

	for _, want := range []string{"hpot", "mpot", "rpot", "spot", "wpot", "apot", "poti", "elix", "scro"} {
		if !beltable[want] {
			t.Errorf("%s should be beltable", want)
		}
	}

	for _, no := range []string{"tome", "armo", "weap", "ring", "amul", "gold"} {
		if beltable[no] {
			t.Errorf("%s must not be beltable", no)
		}
	}
}

// TestBeltItemsInArmorTxt: the belts carry a non-zero armor.txt belt column
// (belts.txt row) and no other armor does.
func TestBeltItemsInArmorTxt(t *testing.T) {
	h, rows := tsv(t, "patch_d2/armor.txt")
	name, bc, tc := col(t, h, "name"), col(t, h, "belt"), col(t, h, "type")

	n := 0

	for _, r := range rows {
		isBeltType := cell(r, tc) == "belt"
		hasCol := cell(r, bc) != "" && cell(r, bc) != "0"

		if hasCol && !isBeltType {
			t.Errorf("%q: type belt=%v but belt column=%q", cell(r, name), isBeltType, cell(r, bc))
		}

		if hasCol {
			n++
		}
	}

	if n < 6 {
		t.Errorf("only %d belts found", n)
	}
}
