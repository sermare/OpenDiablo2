package d2hero

import "testing"

func TestIsPotion(t *testing.T) {
	tests := []struct {
		code string
		want bool
	}{
		{"hp1", true}, {"hp5", true}, {"mp3", true}, {"rvs", true}, {"rvl", true},
		{"hp6", false}, {"hp", false}, {"tsc", false}, {"isc", false}, {"sst", false}, {"", false},
	}

	for _, tc := range tests {
		if got := isPotion(tc.code); got != tc.want {
			t.Errorf("isPotion(%q) = %v, want %v", tc.code, got, tc.want)
		}
	}
}

func TestBodySlotLocation(t *testing.T) {
	tests := []struct {
		loc  string
		want bool
	}{
		{"rarm", true}, {"larm", true}, {"tors", true}, {"RARM", true}, {" belt ", true},
		{"", false}, {"stor", false}, {"inv", false},
	}

	for _, tc := range tests {
		if got := bodySlotLocation(tc.loc); got != tc.want {
			t.Errorf("bodySlotLocation(%q) = %v, want %v", tc.loc, got, tc.want)
		}
	}
}
