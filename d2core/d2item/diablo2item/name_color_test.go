package diablo2item

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2ui"
)

func TestNameColorToken(t *testing.T) {
	tests := []struct {
		name string
		in   nameColorInput
		want d2ui.ColorToken
	}{
		{"normal", nameColorInput{}, d2ui.ColorTokenWhite},
		{"socketed normal", nameColorInput{socketed: true}, d2ui.ColorTokenGrey},
		{"ethereal normal", nameColorInput{ethereal: true}, d2ui.ColorTokenGrey},
		{"magic one affix", nameColorInput{affixes: 1}, d2ui.ColorTokenBlue},
		{"magic two affixes", nameColorInput{affixes: 2}, d2ui.ColorTokenBlue},
		{"magic socketed stays blue", nameColorInput{affixes: 1, socketed: true}, d2ui.ColorTokenBlue},
		{"rare", nameColorInput{affixes: 4}, d2ui.ColorTokenYellow},
		{"set", nameColorInput{set: true, affixes: 3}, d2ui.ColorTokenGreen},
		{"unique ethereal", nameColorInput{unique: true, ethereal: true}, d2ui.ColorTokenGold},
		{"crafted wins", nameColorInput{crafted: true, affixes: 4}, d2ui.ColorTokenOrange},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := nameColorToken(tc.in); got != tc.want {
				t.Fatalf("got %s want %s", got, tc.want)
			}
		})
	}
}
