package d2monreg

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
)

// superRows reads the superuniques.txt of D2_TABLES (skips when absent).
func superRows(t *testing.T, tb *Tables) map[string]SuperRec {
	t.Helper()

	b, err := os.ReadFile(filepath.Join(os.Getenv("D2_TABLES"), "monsters", "patch_d2", "superuniques.txt"))
	if err != nil {
		t.Skip("superuniques.txt not under D2_TABLES/monsters/patch_d2")
	}

	ts, err := readTSV(b)
	if err != nil {
		t.Fatal(err)
	}

	out := map[string]SuperRec{}

	for _, r := range ts.rows {
		out[ts.str(r, "Superunique")] = SuperRec{HcIdx: ts.num(r, "hcIdx"), Class: tb.MonByKey(ts.str(r, "Class")),
			Mods:   [3]int{ts.num(r, "Mod1"), ts.num(r, "Mod2"), ts.num(r, "Mod3")},
			MinGrp: ts.num(r, "MinGrp"), MaxGrp: ts.num(r, "MaxGrp"), Stacks: ts.num(r, "Stacks") != 0}
	}

	return out
}

func superRoom(seed uint32) *Room {
	return &Room{Level: 25, Seed: *d2rand.New(seed), X: 500, Y: 500, W: 40, H: 40,
		Cells: []Cell{{X0: 100, Y0: 100, X1: 108, Y1: 108, Flag: 1}}}
}

// The Countess (hcIdx 6): the class itself plus MinGrp = MaxGrp = 6 (+ difficulty)
// units of its minion class, which is the class itself (corruptrogue3 has no
// valid minion1); the row's modifiers, + one extra pick per difficulty, then 22.
func TestSuperUniqueCountess(t *testing.T) {
	tb := realUModTables(t)
	rows := superRows(t, tb)
	c := rows["The Countess"]

	if c.HcIdx != 6 || c.Class < 0 {
		t.Fatalf("countess row %+v", c)
	}

	for diff := 0; diff < 3; diff++ {
		g := NewGame(tb, 99, diff, true)
		room := superRoom(77)

		var pop Population

		u := g.SuperUnique(openWorld{}, room, c, 520, 520, &pop)
		if u == nil {
			t.Fatalf("diff %d: not created", diff)
		}

		followers := 0

		for _, f := range pop.Units {
			if f.Leader == u && f.Minion {
				followers++

				if f.Class != c.Class {
					t.Errorf("diff %d: follower class %d, want the countess' own %d", diff, f.Class, c.Class)
				}
			}
		}

		if want := 6 + diff; followers != want {
			t.Errorf("diff %d: %d followers, want %d", diff, followers, want)
		}

		// Mod1..Mod3 of the row (id 24 skipped, reading stops at 0), diff extras, 22 last
		wantMods := 0

		for _, m := range c.Mods {
			if m == 0 {
				break
			}

			if m != 0x18 {
				wantMods++
			}
		}

		wantMods += diff + 1

		if len(u.Mods) != wantMods || u.Mods[len(u.Mods)-1] != 22 {
			t.Errorf("diff %d: mods %v, want %d ending in 22", diff, u.Mods, wantMods)
		}

		if u.Super != 7 {
			t.Errorf("diff %d: super id %d", diff, u.Super)
		}

		// once per game
		if g.SuperUnique(openWorld{}, room, c, 520, 520, &pop) != nil {
			t.Errorf("diff %d: made twice", diff)
		}
	}

	// above Hell nothing is made
	g := NewGame(tb, 1, 0, true)
	g.Difficulty = 3

	if g.SuperUnique(openWorld{}, superRoom(1), c, 520, 520, &Population{}) != nil {
		t.Error("difficulty 3 made a super unique")
	}
}

// Shenk (hcIdx 42) brings twenty Enslaved; Radament (hcIdx 10) the skeleton group.
func TestSuperUniqueExtras(t *testing.T) {
	tb := realUModTables(t)
	rows := superRows(t, tb)

	for _, tc := range []struct {
		key   string
		hc    int
		extra int // minimum extra units around the leader besides MinGrp followers
	}{{"Siege Boss", 42, 20}, {"Radament", 10, 6}} {
		r, ok := rows[tc.key]
		if !ok || r.HcIdx != tc.hc {
			t.Fatalf("%s row %+v", tc.key, r)
		}

		g := NewGame(tb, 5, 0, true)

		var pop Population

		if g.SuperUnique(openWorld{}, superRoom(3), r, 520, 520, &pop) == nil {
			t.Fatalf("%s not created", tc.key)
		}

		if got := len(pop.Units) - 1 - r.MinGrp; got < tc.extra {
			t.Errorf("%s: %d extra units, want at least %d", tc.key, got, tc.extra)
		}
	}
}
