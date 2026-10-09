package d2hero

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
)

func TestStarterSlot(t *testing.T) {
	tests := []struct {
		loc  string
		want int
	}{
		{"rarm", 4}, {"larm", 5}, {"RARM", 4}, {" belt ", 8},
		{"4", 4}, {"5", 5}, {"0", 0}, {"", 0}, {"inv", 0}, {"99", 0}, {"-1", 0},
	}

	for _, tc := range tests {
		if got := starterSlot(tc.loc); got != tc.want {
			t.Errorf("starterSlot(%q) = %d, want %d", tc.loc, got, tc.want)
		}
	}
}

func TestActiveWeaponSetSlot(t *testing.T) {
	tests := []struct {
		slot  uint8
		setII bool
		want  uint8
	}{
		{4, false, 4}, {5, false, 5}, {11, false, 0}, {12, false, 0},
		{4, true, 0}, {5, true, 0}, {11, true, 4}, {12, true, 5},
		{1, false, 1}, {1, true, 1}, {9, true, 9},
	}

	for _, tc := range tests {
		if got := ActiveWeaponSetSlot(tc.slot, tc.setII); got != tc.want {
			t.Errorf("ActiveWeaponSetSlot(%d, %v) = %d, want %d", tc.slot, tc.setII, got, tc.want)
		}
	}
}

func TestNameAt(t *testing.T) {
	list := []string{"", "Sturdy", "of Vita"}

	tests := []struct {
		id   int
		want string
	}{{0, ""}, {1, "Sturdy"}, {2, "of Vita"}, {3, ""}, {-1, ""}}

	for _, tc := range tests {
		if got := nameAt(list, tc.id); got != tc.want {
			t.Errorf("nameAt(%d) = %q, want %q", tc.id, got, tc.want)
		}
	}
}

func TestImportedInfo(t *testing.T) {
	h := &d2s.Header{Status: d2s.StatusHardcore | d2s.StatusExpansion | d2s.StatusDied}
	h.Raw[0x10] = 1

	info := importedInfo(h)

	if !info.Hardcore || !info.Expansion || !info.Dead || info.Ladder || !info.WeaponSetII {
		t.Fatalf("flags: %+v", info)
	}
}

func TestClampLevel(t *testing.T) {
	if clampLevel(0) != 1 || clampLevel(94) != 94 {
		t.Fatal("clampLevel")
	}
}
