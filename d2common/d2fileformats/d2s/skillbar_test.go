package d2s

import "testing"

func TestSkillBlockRoundTrip(t *testing.T) {
	var h Header

	// the bytes of the real NokkaSorc save at 0x38 and 0x78
	copy(h.Raw[0x38:], []byte{
		0x3b, 0, 0, 0, 0x2f, 0, 0, 0, 0x36, 0, 0, 0, 0x2a, 0, 0, 0, 0x37, 0, 0, 0, 0x28, 0, 0, 0, 0x2b, 0, 0, 0,
		0xff, 0xff, 0, 0})
	copy(h.Raw[0x78:], []byte{0, 0, 0, 0, 0x2f, 0, 0, 0, 0, 0, 0, 0, 0x2f, 0, 0, 0})

	b := h.SkillBlock()

	tests := []struct {
		name string
		got  SkillWord
		want SkillWord
	}{
		{"first hotkey", b.Hotkeys[0], SkillWord{Skill: 59}},
		{"second hotkey", b.Hotkeys[1], SkillWord{Skill: 47}},
		{"eighth hotkey (empty)", b.Hotkeys[7], SkillWord{Skill: 0xFFFF}},
		{"left", b.Left, SkillWord{Skill: 0}},
		{"right", b.Right, SkillWord{Skill: 47}},
		{"left swap", b.LeftSwap, SkillWord{Skill: 0}},
		{"right swap", b.RightSwap, SkillWord{Skill: 47}},
	}

	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s = %+v, want %+v", tt.name, tt.got, tt.want)
		}
	}

	// writing a block changes exactly those bytes
	before := h.Raw
	b.Hotkeys[15] = SkillWord{Skill: 0x8000 | 36, Item: 9}
	b.Right = SkillWord{Skill: 54}
	h.SetSkillBlock(b)

	for i := range h.Raw {
		changed := h.Raw[i] != before[i]
		inside := (i >= 0x38+4*15 && i < 0x38+4*16) || (i >= 0x7C && i < 0x80)

		if changed && !inside {
			t.Errorf("byte 0x%X changed outside the edited words", i)
		}
	}

	if got := h.SkillBlock(); got != b {
		t.Errorf("re-read block = %+v, want %+v", got, b)
	}
}
