package d2npc

import "testing"

// TestHellRespecGrant covers the Hell-only grant of Akara's free reset (0x577b70 case 0x94, VERIFIED w47).
func TestHellRespecGrant(t *testing.T) {
	den := [2]int{1, 0}

	cases := []struct {
		name string
		diff int
		c    Ctx
		want bool
	}{
		{"hell, den done, nothing in slot 41", 2, Ctx{Quest: quest(den)}, true},
		{"nightmare", 1, Ctx{Quest: quest(den)}, false},
		{"normal", 0, Ctx{Quest: quest(den)}, false},
		{"hell, den not done", 2, Ctx{Quest: quest()}, false},
		{"hell, reset already used (bit 0)", 2, Ctx{Quest: quest(den, [2]int{0x29, 0})}, false},
		{"hell, reset already pending (bit 1)", 2, Ctx{Quest: quest(den, [2]int{0x29, 1})}, false},
	}

	for _, c := range cases {
		if got := HellRespecGrant(c.diff, c.c); got != c.want {
			t.Errorf("%s: %v want %v", c.name, got, c.want)
		}
	}
}

func TestRespecPanelFlagIsHell(t *testing.T) {
	for d, want := range []bool{false, false, true} {
		if got := RespecPanelFlagFor(d); got != want {
			t.Errorf("difficulty %d: %v want %v", d, got, want)
		}
	}

	// in Hell the row shows without the pending bit, until the reset is used
	if !RespecShown(Ctx{RespecPanelFlag: RespecPanelFlagFor(2)}) {
		t.Error("Hell should show the row")
	}
}
