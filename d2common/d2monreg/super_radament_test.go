package d2monreg

import "testing"

// Radament (hcIdx 10) is a DS1 super unique of Sewers Level 3 that the Radament's Lair quest waits for. On the open
// ground of a room it must be created on every difficulty with its followers (real superuniques.txt, skipped when
// D2_TABLES is unset).
func TestSuperUniqueRadament(t *testing.T) {
	tb := realUModTables(t)
	r := superRows(t, tb)["Radament"]

	if r.HcIdx != 10 || r.Class < 0 {
		t.Fatalf("radament row %+v", r)
	}

	for diff := 0; diff < 3; diff++ {
		g := NewGame(tb, 99, diff, true)

		var pop Population

		if u := g.SuperUnique(openWorld{}, superRoom(77), r, 520, 520, &pop); u == nil {
			t.Fatalf("diff %d: Radament was not created", diff)
		}
	}
}
