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

func TestHotkeyID(t *testing.T) {
	tests := []struct {
		in   uint32
		want int
	}{{0, 0}, {59, 59}, {d2s.NoSkill, -1}, {0xFFFFFFFF, -1}}

	for _, tc := range tests {
		if got := hotkeyID(tc.in); got != tc.want {
			t.Errorf("hotkeyID(%#x) = %d, want %d", tc.in, got, tc.want)
		}
	}
}

func TestImportedInfo(t *testing.T) {
	h := &d2s.Header{
		Status:         d2s.StatusHardcore | d2s.StatusExpansion | d2s.StatusDied,
		LastPlayed:     1505317599,
		LeftSwapSkill:  0,
		RightSwapSkill: 47,
	}
	h.Hotkeys[0], h.Hotkeys[1] = 59, d2s.NoSkill

	info := importedInfo(h)

	if !info.Hardcore || !info.Expansion || !info.Dead || info.Ladder {
		t.Fatalf("flags: %+v", info)
	}

	if info.Hotkeys[0] != 59 || info.Hotkeys[1] != -1 || len(info.Hotkeys) != d2s.HotkeyCount || info.SwapRightSkill != 47 {
		t.Fatalf("skills: %+v", info)
	}
}

func TestMaxIntAndClampLevel(t *testing.T) {
	if maxInt(869, 1241) != 1241 || maxInt(5, 3) != 5 {
		t.Fatal("maxInt")
	}

	if clampLevel(0) != 1 || clampLevel(94) != 94 {
		t.Fatal("clampLevel")
	}
}
