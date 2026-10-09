package d2sfx

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// TestRealSoundsTxt reads the real Sounds.txt (patch_d2 > d2exp > d2data) from
// $D2_TABLES/sound/patch_d2 and checks the invariants the engine relies on.
func TestRealSoundsTxt(t *testing.T) {
	dir := os.Getenv("D2_TABLES")
	if dir == "" {
		t.Skip("D2_TABLES not set")
	}

	data, err := os.ReadFile(filepath.Join(dir, "sound", "patch_d2", "Sounds.txt"))
	if err != nil {
		t.Skip(err)
	}

	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	col := map[string]int{}

	for i, h := range strings.Split(lines[0], "\t") {
		col[h] = i
	}

	num := func(f []string, name string) int {
		n, _ := strconv.Atoi(f[col[name]])
		return n
	}

	var rows []Row

	for _, ln := range lines[1:] {
		if ln == "" {
			continue
		}

		f := strings.Split(ln, "\t")
		rows = append(rows, Row{Handle: f[col["Sound"]], Index: num(f, "Index"), GroupSize: num(f, "Group Size"),
			Priority: num(f, "Priority"), Falloff: num(f, "Falloff")})
	}

	for i, r := range rows {
		if r.Index != i {
			t.Fatalf("row ordinal %d has Index %d", i, r.Index)
		}
	}

	tb := NewTable(rows)
	if tb.Len() != len(rows) {
		t.Fatalf("table len %d rows %d", tb.Len(), len(rows))
	}

	for i := range rows {
		h := tb.Head(i)
		if h > i || (h != i && rows[h].GroupSize <= i-h) {
			t.Fatalf("row %d has bad head %d", i, h)
		}
	}
}
