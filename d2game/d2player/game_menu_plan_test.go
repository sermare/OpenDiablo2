package d2player

import "testing"

// Golden of GameMenuPlan: row top, left pentagram x, right pentagram x, pentagram y, for the two descriptors of the
// original (main menu 3 entries, options 5 entries) in both modes; skullW 54 (pentSize).
func TestGameMenuPlanGolden(t *testing.T) {
	for _, c := range []struct {
		name  string
		mode  Mode
		total int
		want  []GameMenuPlacement
	}{
		{"800 main", Mode800, 3, []GameMenuPlacement{
			{185, 97, 649, 236}, {235, 97, 649, 286}, {285, 97, 649, 336}}},
		{"800 options", Mode800, 5, []GameMenuPlacement{
			{135, 97, 649, 186}, {185, 97, 649, 236}, {235, 97, 649, 286}, {285, 97, 649, 336}, {335, 97, 649, 386}}},
		{"640 main", Mode640, 3, []GameMenuPlacement{
			{125, 17, 569, 176}, {175, 17, 569, 226}, {225, 17, 569, 276}}},
	} {
		got := c.mode.GameMenuPlan(c.total, c.total, 54)
		if len(got) != len(c.want) {
			t.Fatalf("%s: %d rows", c.name, len(got))
		}

		for i := range got {
			if got[i] != c.want[i] {
				t.Errorf("%s row %d: got %+v want %+v", c.name, i, got[i], c.want[i])
			}
		}
	}

	// hidden rows count for the centring but are not placed
	if p := Mode800.GameMenuPlan(5, 2, 54); len(p) != 2 || p[0].RowTop != 135 {
		t.Errorf("hidden rows: %+v", p)
	}

	// the behaviour without the exe layout is unchanged: the row offset plus the spacer
	if escapeMenuPentY(100) != 110 {
		t.Error("legacy pentagram y changed")
	}

	if escapeMenuModeFor(640, 480) != Mode640 || escapeMenuModeFor(800, 600) != Mode800 || escapeMenuModeFor(0, 0) != Mode800 {
		t.Error("mode pick")
	}
}
