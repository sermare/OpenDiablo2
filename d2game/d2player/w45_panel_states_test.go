package d2player

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2state"
)

// The state columns of states.txt turn the defense, resistance, damage and attack rating numbers blue or red
// (UI_DrawCharacterStatsPanel 0x4a43f0, masks 0x118..0x140). Red wins; at the resistance cap blue stays blue, a red
// state at the cap still shows gold.
func TestResistLineStates(t *testing.T) {
	tests := []struct {
		name         string
		shown, total int
		raised, red  bool
		text         string
		col          int
	}{
		{"blue below cap", 30, 75, true, false, "30", StatFontBlue},
		{"red below cap", 30, 75, false, true, "30", StatFontRed},
		{"red beats blue", 30, 75, true, true, "30", StatFontRed},
		{"blue at cap stays blue", 80, 75, true, false, "75", StatFontBlue},
		{"red at cap is gold", 80, 75, false, true, "75", StatFontGold},
		{"negative blue is red", -5, 75, true, false, "-5", StatFontRed},
	}

	for _, tc := range tests {
		txt, col := resistLineStates(tc.shown, tc.total, tc.raised, tc.red)
		if txt != tc.text || col != tc.col {
			t.Errorf("%s: %q/%d, want %q/%d", tc.name, txt, col, tc.text, tc.col)
		}
	}
}

func TestPanelFlagsColours(t *testing.T) {
	var cols [14]bool
	cols[0], cols[1], cols[10], cols[13] = true, true, true, true // armblue armred dmgblue attred

	f := d2state.PanelFlagsFromColumns(cols)

	if DefenseColour(f.DefenseBlue(), f.DefenseRed()) != StatFontRed {
		t.Error("armblue + armred must show red")
	}

	if DefenseColour(f.DamageBlue(), f.DamageRed()) != StatFontBlue {
		t.Error("dmgblue must show blue")
	}

	if DefenseColour(f.AttackBlue(), f.AttackRed()) != StatFontRed {
		t.Error("attred must show red")
	}

	// Holy Shield is blue only with a shield in the hand
	cases := []struct{ armBlue, shield, holy, want bool }{
		{false, false, true, false},
		{false, true, true, true},
		{false, true, false, false},
		{true, false, false, true},
	}

	for _, c := range cases {
		if got := DefenseRaised(c.armBlue, c.shield, c.holy); got != c.want {
			t.Errorf("%+v: %v", c, got)
		}
	}
}
