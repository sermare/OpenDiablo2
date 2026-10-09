package d2monsters

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2statlist"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2records"
)

// realMonster builds a monster from a real monstats.txt row (Id, hcIdx, boss).
func realMonster(t *testing.T, path, id string) *d2mapentity.Monster {
	t.Helper()

	f, err := os.Open(path)
	if err != nil {
		t.Skipf("monstats.txt not readable: %v", err)
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)

	var col map[string]int

	for sc.Scan() {
		cells := strings.Split(sc.Text(), "\t")
		if col == nil {
			col = map[string]int{}
			for i, c := range cells {
				col[c] = i
			}

			continue
		}

		if cells[col["Id"]] != id {
			continue
		}

		idx, _ := strconv.Atoi(cells[col["hcIdx"]])
		boss, _ := strconv.Atoi(cells[col["boss"]])

		return &d2mapentity.Monster{Stat: &d2records.MonStatRecord{Key: id, ID: idx, IsSpecialBoss: boss > 0}}
	}

	t.Fatalf("row %s not found", id)

	return nil
}

// TestHeroAROperandsRealRows covers the plain (0x73) and halved (0x74)
// conditions of 0x57b8b0 with real monstats rows.
func TestHeroAROperandsRealRows(t *testing.T) {
	dir := os.Getenv("D2_TABLES")
	if dir == "" {
		t.Skip("D2_TABLES not set")
	}

	path := filepath.Join(dir, "monsters", "patch_d2", "monstats.txt")

	list := func(id int, v int64) *d2statlist.Totals {
		l := d2statlist.NewList()
		l.Add(id, 0, v)

		return &d2statlist.Totals{Stats: l}
	}

	tests := []struct {
		name  string
		id    string
		flags uint16
		stat  int
		val   int64
		want  int // resulting defense from 200
	}{
		{"plain zombie ignore def", "zombie1", 0, d2statlist.StatIgnoreDef, 1, 0},
		{"plain zombie target ac 40", "zombie1", 0, d2statlist.StatTargetACPct, 40, 120},
		{"unique zombie keeps def", "zombie1", d2mapentity.MonTypeUnique, d2statlist.StatIgnoreDef, 1, 200},
		{"unique zombie not halved", "zombie1", d2mapentity.MonTypeUnique, d2statlist.StatTargetACPct, 40, 120},
		{"super unique keeps def", "zombie1", d2mapentity.MonTypeSuperUnique, d2statlist.StatIgnoreDef, 1, 200},
		{"super unique halved", "zombie1", d2mapentity.MonTypeSuperUnique, d2statlist.StatTargetACPct, 40, 160},
		{"andariel keeps def", "andariel", 0, d2statlist.StatIgnoreDef, 1, 200},
		{"andariel halved", "andariel", 0, d2statlist.StatTargetACPct, 40, 160},
	}

	for _, tt := range tests {
		m := realMonster(t, path, tt.id)
		m.TypeFlags = tt.flags

		if _, def := HeroAROperands(list(tt.stat, tt.val), m, 100, 200); def != tt.want {
			t.Errorf("%s: def %d want %d", tt.name, def, tt.want)
		}
	}
}
