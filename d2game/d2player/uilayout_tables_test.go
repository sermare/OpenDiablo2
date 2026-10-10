package d2player

import "testing"

// The numbers below were read from Game.exe 1.14b (UI_DrawSkillTreeIcon 0x4a86b0, UI_GetSkillTreeTierY 0x4a7100,
// UI_DrawPartyScreen 0x496540, UI_CreateDialog layout 0x4b4d40, UI_DrawBeltPanel 0x495530).

func TestSkillIconPosition(t *testing.T) {
	cases := []struct{ row, col, x, y int }{
		{1, 1, 415, 122}, {1, 2, 484, 122}, {1, 3, 553, 122},
		{2, 1, 415, 190}, {3, 2, 484, 258}, {4, 3, 553, 326},
		{5, 1, 415, 395}, {6, 3, 553, 463},
	}

	for _, c := range cases {
		if x, y := skillIconPosition(c.row, c.col); x != c.x || y != c.y {
			t.Errorf("row %d col %d: got (%d,%d), want (%d,%d)", c.row, c.col, x, y, c.x, c.y)
		}
	}
}

func TestPartyRowPitch(t *testing.T) {
	if indexOffset != 38 {
		t.Errorf("party row pitch %d, want 38", indexOffset)
	}
}

func TestBeltPopPosition(t *testing.T) {
	want := [][2]int{{1, 559}, {2, 527}, {3, 495}}
	for _, w := range want {
		if x, y := beltPopPosition(w[0]); x != 421 || y != w[1] {
			t.Errorf("row %d: (%d,%d)", w[0], x, y)
		}
	}
}
