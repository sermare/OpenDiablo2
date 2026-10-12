package d2gamescreen

import "testing"

func TestDeathOverlayText(t *testing.T) {
	const gold, exp, hc = "Death takes its toll of %d Gold", "You have lost experience", "Your deeds of valor will be remembered"

	tests := []struct {
		name     string
		hardcore bool
		gold     int
		exp      bool
		want     string
	}{
		{"hardcore", true, 500, true, hc},
		{"softcore gold and exp", false, 120, true, "Death takes its toll of 120 Gold\nYou have lost experience"},
		{"softcore no gold", false, 0, true, exp},
		{"softcore nothing", false, 0, false, ""},
	}

	for _, tc := range tests {
		if got := deathOverlayText(tc.hardcore, tc.gold, tc.exp, gold, exp, hc); got != tc.want {
			t.Errorf("%s: got %q want %q", tc.name, got, tc.want)
		}
	}
}
