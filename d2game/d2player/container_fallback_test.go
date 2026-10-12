package d2player

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// The built in fallback of the stash is the classic 6 x 4 grid of the "Bank Page2" record (the 800x600 layout) of
// Inventory.txt. D2_TABLES points at extracted tables (itemgen/patch_d2/Inventory.txt); skipped without them.
func TestStashFallbackMatchesInventoryRecord(t *testing.T) {
	dir := os.Getenv("D2_TABLES")
	if dir == "" {
		t.Skip("D2_TABLES not set")
	}

	data, err := os.ReadFile(filepath.Join(dir, "itemgen", "patch_d2", "Inventory.txt"))
	if err != nil {
		t.Skip("Inventory.txt not available")
	}

	lines := strings.FieldsFunc(string(data), func(r rune) bool { return r == '\n' || r == '\r' })
	col := map[string]int{}

	for i, h := range strings.Split(lines[0], "\t") {
		col[h] = i
	}

	for _, ln := range lines[1:] {
		f := strings.Split(ln, "\t")
		if f[0] != "Bank Page2" {
			continue
		}

		num := func(name string) int {
			v, err := strconv.Atoi(f[col[name]])
			if err != nil {
				t.Fatalf("%s: %v", name, err)
			}

			return v
		}

		// gridX is the width in cells, gridY the height
		if num("gridX") != stashKind.cols || num("gridY") != stashKind.rows ||
			num("gridLeft") != stashKind.left || num("gridTop") != stashKind.top {
			t.Errorf("fallback %dx%d at (%d,%d) differs from the record", stashKind.cols, stashKind.rows,
				stashKind.left, stashKind.top)
		}

		return
	}

	t.Fatal("no Bank Page2 record")
}
