package d2hero

import (
	"errors"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2records"
)

func TestSkillWordRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		word d2s.SkillWord
		want SkillSlot
	}{
		{"none", d2s.SkillWord{Skill: 0xFFFF}, SkillSlot{Skill: NoSkill}},
		{"none with item", d2s.SkillWord{Skill: 0xFFFF, Item: 7}, SkillSlot{Skill: NoSkill, Item: 7}},
		{"attack", d2s.SkillWord{Skill: 0}, SkillSlot{Skill: 0}},
		{"fire ball", d2s.SkillWord{Skill: 47}, SkillSlot{Skill: 47}},
		{"left hand", d2s.SkillWord{Skill: 0x8000 | 36}, SkillSlot{Skill: 36, Left: true}},
		{"item skill", d2s.SkillWord{Skill: 54, Item: 0x1234}, SkillSlot{Skill: 54, Item: 0x1234}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := slotFromWord(tt.word)
			if got != tt.want {
				t.Fatalf("decode = %+v, want %+v", got, tt.want)
			}

			if back := got.word(); back != tt.word {
				t.Fatalf("encode = %+v, want %+v", back, tt.word)
			}
		})
	}
}

func TestSkillBarAssign(t *testing.T) {
	b := NewSkillBar()

	if got := b.HotkeyOf(47); got != -1 {
		t.Fatalf("empty bar: HotkeyOf = %d", got)
	}

	if err := b.Assign(0, 47, false); err != nil {
		t.Fatal(err)
	}

	if err := b.Assign(3, 36, true); err != nil {
		t.Fatal(err)
	}

	// moving a skill to another key empties its old key
	if err := b.Assign(8, 47, false); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		slot   int
		want   SkillSlot
		wantOK bool
	}{
		{0, SkillSlot{}, false},
		{3, SkillSlot{Skill: 36, Left: true}, true},
		{8, SkillSlot{Skill: 47}, true}, // the second set of hotkeys (slots 8-15) works too
		{15, SkillSlot{}, false},
		{16, SkillSlot{}, false},
		{-1, SkillSlot{}, false},
	}

	for _, tt := range tests {
		got, ok := b.Press(tt.slot)
		if ok != tt.wantOK || (ok && got != tt.want) {
			t.Errorf("Press(%d) = %+v,%v want %+v,%v", tt.slot, got, ok, tt.want, tt.wantOK)
		}
	}

	if got := b.HotkeyOf(47); got != 8 {
		t.Errorf("HotkeyOf(47) = %d, want 8", got)
	}

	if err := b.Assign(16, 1, false); !errors.Is(err, ErrBadSlot) {
		t.Errorf("slot 16: err = %v", err)
	}

	if err := b.Assign(3, -1, false); err != nil || !b.Hotkeys[3].Empty() {
		t.Errorf("assigning -1 must clear the slot: err=%v slot=%+v", err, b.Hotkeys[3])
	}
}

func TestSkillBarSelectAndSwap(t *testing.T) {
	b := NewSkillBar()
	b.Select(false, 47)
	b.Select(true, 36)
	b.LeftSwap, b.RightSwap = SkillSlot{Skill: 0}, SkillSlot{Skill: 54}

	if b.Active(true) != 36 || b.Active(false) != 47 {
		t.Fatalf("active = %d/%d", b.Active(true), b.Active(false))
	}

	b.SwapSets()

	if b.Active(true) != 0 || b.Active(false) != 54 || b.LeftSwap.Skill != 36 || b.RightSwap.Skill != 47 {
		t.Fatalf("after swap: %+v", b)
	}

	b.SwapSets()

	if b.Active(true) != 36 || b.Active(false) != 47 {
		t.Fatalf("swap twice must restore: %+v", b)
	}
}

func heroSkill(id int, points int, left, passive bool) *HeroSkill {
	return &HeroSkill{SkillRecord: &d2records.SkillRecord{ID: id, Leftskill: left, Passive: passive}, SkillPoints: points}
}

func TestSelectable(t *testing.T) {
	tests := []struct {
		name  string
		skill *HeroSkill
		left  bool
		want  bool
	}{
		{"nil", nil, false, false},
		{"unlearned", heroSkill(47, 0, true, false), false, false},
		{"learned right", heroSkill(47, 1, true, false), false, true},
		{"learned left", heroSkill(47, 1, true, false), true, true},
		{"right-only skill on the left", heroSkill(54, 3, false, false), true, false},
		{"right-only skill on the right", heroSkill(54, 3, false, false), false, true},
		{"passive", heroSkill(61, 5, true, true), false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Selectable(tt.skill, tt.left); got != tt.want {
				t.Fatalf("Selectable = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFixActive(t *testing.T) {
	skills := map[int]*HeroSkill{0: heroSkill(0, 1, true, false), 47: heroSkill(47, 2, true, false), 54: heroSkill(54, 0, false, false)}
	b := NewSkillBar()
	b.Left, b.Right = SkillSlot{Skill: 54}, SkillSlot{Skill: 47}
	b.FixActive(skills)

	if b.Left.Skill != 0 || b.Right.Skill != 47 {
		t.Fatalf("FixActive: left=%d right=%d", b.Left.Skill, b.Right.Skill)
	}
}

// The real NokkaSorc save: Fire Ball (47) on the right button, Attack on the
// left, the same in the swap set, seven hotkeys; and the bar survives an export.
func TestSkillBarRealSave(t *testing.T) {
	data, tables, state := realSave(t)

	hdr, err := d2s.ParseHeader(data)
	if err != nil {
		t.Fatal(err)
	}

	bar := SkillBarFromBlock(hdr.SkillBlock())

	if hdr.Name == "NokkaSorc" {
		if bar.Right.Skill != 47 || bar.Left.Skill != 0 || bar.RightSwap.Skill != 47 || bar.LeftSwap.Skill != 0 {
			t.Fatalf("NokkaSorc skills left=%d right=%d swap=%d/%d, want 0/47 0/47",
				bar.Left.Skill, bar.Right.Skill, bar.LeftSwap.Skill, bar.RightSwap.Skill)
		}

		want := []int{59, 47, 54, 42, 55, 40, 43}
		for i, id := range want {
			if bar.Hotkeys[i].Skill != id {
				t.Errorf("hotkey %d = %d, want %d", i, bar.Hotkeys[i].Skill, id)
			}
		}

		for i := len(want); i < HotkeyCount; i++ {
			if !bar.Hotkeys[i].Empty() {
				t.Errorf("hotkey %d = %d, want none", i, bar.Hotkeys[i].Skill)
			}
		}
	}

	if blk := bar.Block(); blk != hdr.SkillBlock() {
		t.Fatalf("decode/encode changed the words:\n got %+v\nwant %+v", blk, hdr.SkillBlock())
	}

	// an imported, untouched bar leaves the file byte-identical
	state.SkillBar = bar

	out, _, err := ExportD2SWithOptions(state, data, tables, ExportOptions{})
	if err != nil {
		t.Fatal(err)
	}

	if string(out) != string(data) {
		t.Fatal("unchanged skill bar changed the exported file")
	}

	// edits are written to the header and survive a re-parse
	edit := *bar
	if err := edit.Assign(10, 36, true); err != nil {
		t.Fatal(err)
	}

	edit.Select(false, 54)
	edit.LeftSwap = SkillSlot{Skill: 36}
	edit.RightSwap = SkillSlot{Skill: 47}
	state.SkillBar = &edit

	out, _, err = ExportD2SWithOptions(state, data, tables, ExportOptions{})
	if err != nil {
		t.Fatal(err)
	}

	hdr2, err := d2s.ParseHeader(out)
	if err != nil {
		t.Fatal(err)
	}

	got := SkillBarFromBlock(hdr2.SkillBlock())
	if *got != edit {
		t.Fatalf("re-parsed bar differs:\n got %+v\nwant %+v", *got, edit)
	}

	if got.Hotkeys[10] != (SkillSlot{Skill: 36, Left: true}) || got.Right.Skill != 54 || got.LeftSwap.Skill != 36 {
		t.Fatalf("edits lost: %+v", got)
	}
}
