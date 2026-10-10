package d2player

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The numbers are those of UI_DrawMercenaryPanel 0x48ee50 and its helpers and of the Inventory.txt records
// "Hireling" and "Hireling2" (see uilayout_merc.go).

func TestMercSlotsHireling2IsHirelingShifted(t *testing.T) {
	// the 800x600 record is the 640x480 one moved by the panel offset (80,60)
	for i, s := range mercSlots640 {
		w := mercSlots800[i]
		if w.Name != s.Name || w.X != s.X+80 || w.Y != s.Y+60 || w.W != s.W || w.H != s.H {
			t.Errorf("slot %s: 640 %v vs 800 %v", s.Name, s, w)
		}
	}
}

func TestMercPanelKeyRects(t *testing.T) {
	want := map[Mode]map[string]UIRect{
		Mode800: {
			"art_upper_left": {"merc", "art_upper_left", 80, 60, 256, 256},
			"close":          {"merc", "close", 352, 445, 32, 32},
			"slot_weapon":    {"merc", "slot_weapon", 100, 107, 55, 112},
			"slot_head":      {"merc", "slot_head", 215, 68, 54, 51},
			"name":           {"merc", "name", 95, 274, 160, 0},
			"value_strength": {"merc", "value_strength", 234, 342, 48, 0},
			"value_fire":     {"merc", "value_fire", 389, 342, 48, 0},
			"label_fire":     {"merc", "label_fire", 260, 341, 65, 0},
			"label_level":    {"merc", "label_level", 225, 296, 0, 0},
			"label_life":     {"merc", "label_life", 260, 274, 0, 0},
		},
		Mode640: {
			"art_upper_left": {"merc", "art_upper_left", 0, 0, 256, 256},
			"close":          {"merc", "close", 272, 385, 32, 32},
			"slot_shield":    {"merc", "slot_shield", 251, 47, 55, 112},
			"name":           {"merc", "name", 15, 214, 160, 0},
		},
	}

	for m, names := range want {
		got := map[string]UIRect{}
		for _, r := range m.MercPanelRects() {
			got[r.Name] = r
		}

		for n, w := range names {
			if got[n] != w {
				t.Errorf("%v %s: got %v want %v", m, n, got[n], w)
			}
		}
	}
}

func TestMercSlotAt(t *testing.T) {
	tests := []struct {
		m    Mode
		x, y int
		want string
	}{
		{Mode800, 100, 107, "weapon"},
		{Mode800, 154, 218, "weapon"},
		{Mode800, 155, 218, ""},
		{Mode800, 240, 100, "head"},
		{Mode800, 240, 200, "torso"},
		{Mode800, 360, 150, "shield"},
		{Mode800, 10, 10, ""},
		{Mode640, 160, 30, "head"},
		{Mode640, 30, 100, "weapon"},
	}

	for _, c := range tests {
		if got := c.m.MercSlotAt(c.x, c.y); got != c.want {
			t.Errorf("%v (%d,%d) = %q, want %q", c.m, c.x, c.y, got, c.want)
		}
	}
}

func TestMercTablesStats(t *testing.T) {
	// item stat ids of the value table: ItemStatCost rows strength 0, dexterity 2, life 6, level 12,
	// experience 13, damage min 21 (shown as a range), defense 31, next level 30, resists 39/43/41/45
	want := map[string]int{
		"experience": 13, "level": 12, "next_level": 30, "strength": 0, "dexterity": 2, "damage": 21, "defense": 31,
		"fire": 39, "cold": 43, "lightning": 41, "poison": 45, "life": 6,
	}

	for n, s := range want {
		if got, ok := MercValueStat(n); !ok || got != s {
			t.Errorf("stat of %s = %d (%v), want %d", n, got, ok, s)
		}
	}

	ids := map[string]int{"experience": 4058, "level": 4057, "next_level": 4059, "strength": 4060, "fire": 4071, "poison": 4074, "life": 4068}
	for n, id := range ids {
		if got, ok := MercLabelString(n); !ok || got != id {
			t.Errorf("label of %s = %d, want %d", n, got, id)
		}
	}

	if len(mercLabels) != 12 || len(mercValues) != 12 {
		t.Errorf("tables have %d labels and %d values, want 12 and 12", len(mercLabels), len(mercValues))
	}
}

// TestMercLayoutGolden compares every rectangle of both modes with testdata/merc_panel.golden. Regenerate
// with UPDATE_GOLDEN=1 only after re-reading the numbers from the executable.
func TestMercLayoutGolden(t *testing.T) {
	path := filepath.Join("testdata", "merc_panel.golden")
	got := strings.Join(MercLayoutLines(), "\n") + "\n"

	if os.Getenv("UPDATE_GOLDEN") != "" {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}

		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if string(want) != got {
		wl, gl := strings.Split(string(want), "\n"), strings.Split(got, "\n")
		for i := 0; i < len(wl) && i < len(gl); i++ {
			if wl[i] != gl[i] {
				t.Fatalf("golden differs at line %d: got %q want %q", i+1, gl[i], wl[i])
			}
		}

		t.Fatalf("golden has %d lines, layout %d", len(wl), len(gl))
	}
}
