package d2hero

import (
	"errors"
	"fmt"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
)

// HotkeyCount is the number of skill hotkey slots of a hero: the 16 entries of
// the client record (+0x3dc, 8 bytes each) the original keeps, which a .d2s
// stores at header 0x38. The default key bindings reach the first eight
// (F1-F8, commands "UseSkill1".."UseSkill8").
const HotkeyCount = d2s.HotkeyCount

// NoSkill marks an empty slot.
const NoSkill = -1

const (
	wordNone    = 0xFFFF
	wordLeftBit = 0x8000
)

// SkillSlot is one entry of the skill bar: a skill, the hand it was assigned to
// and the item it comes from (always 0 for the class skills the engine knows;
// item-granted skills are not modelled, the item id is only carried through so
// a save keeps what it found).
type SkillSlot struct {
	Skill int    `json:"skill"`
	Left  bool   `json:"left,omitempty"`
	Item  uint16 `json:"item,omitempty"`
}

// Empty reports whether the slot holds no skill.
func (s SkillSlot) Empty() bool { return s.Skill < 0 }

func slotFromWord(w d2s.SkillWord) SkillSlot {
	if w.Skill == wordNone {
		return SkillSlot{Skill: NoSkill, Item: w.Item}
	}

	return SkillSlot{Skill: int(w.Skill &^ wordLeftBit), Left: w.Skill&wordLeftBit != 0, Item: w.Item}
}

func (s SkillSlot) word() d2s.SkillWord {
	if s.Skill < 0 {
		return d2s.SkillWord{Skill: wordNone, Item: s.Item}
	}

	w := uint16(s.Skill) &^ wordLeftBit
	if s.Left {
		w |= wordLeftBit
	}

	return d2s.SkillWord{Skill: w, Item: s.Item}
}

// SkillBar is everything the player has chosen about skills besides the points:
// the 16 hotkeys, the active left and right skill and the same pair of the
// weapon swap set. It is pure data; the game layers (popup, hotkeys, casting)
// work on it and it round-trips through the .d2s header.
type SkillBar struct {
	Hotkeys   [HotkeyCount]SkillSlot `json:"hotkeys"`
	Left      SkillSlot              `json:"left"`
	Right     SkillSlot              `json:"right"`
	LeftSwap  SkillSlot              `json:"leftSwap"`
	RightSwap SkillSlot              `json:"rightSwap"`
}

// NewSkillBar returns the bar of a new character: no hotkeys, Attack (id 0) on
// both buttons and in the swap set.
func NewSkillBar() *SkillBar {
	b := &SkillBar{}
	for i := range b.Hotkeys {
		b.Hotkeys[i] = SkillSlot{Skill: NoSkill}
	}

	return b
}

// SkillBarFromBlock converts the header words.
func SkillBarFromBlock(blk d2s.SkillBlock) *SkillBar {
	b := &SkillBar{
		Left: slotFromWord(blk.Left), Right: slotFromWord(blk.Right),
		LeftSwap: slotFromWord(blk.LeftSwap), RightSwap: slotFromWord(blk.RightSwap),
	}

	for i, w := range blk.Hotkeys {
		b.Hotkeys[i] = slotFromWord(w)
	}

	return b
}

// Block converts the bar to the header words.
func (b *SkillBar) Block() d2s.SkillBlock {
	blk := d2s.SkillBlock{Left: b.Left.word(), Right: b.Right.word(), LeftSwap: b.LeftSwap.word(), RightSwap: b.RightSwap.word()}

	for i, s := range b.Hotkeys {
		blk.Hotkeys[i] = s.word()
	}

	return blk
}

// ErrBadSlot is returned for a hotkey slot outside 0..15.
var ErrBadSlot = errors.New("skillbar: hotkey slot out of range")

// Assign puts a skill on a hotkey. The hand says which button the skill goes to
// when the key is pressed. A skill sits on one hotkey only: it moves from its
// old key. UNVERIFIED: the server side was read (0x54a690 packet 0x51 accepts a
// slot 0..15 and a skill the unit owns and stores it at client record +0x3dc +
// 8*slot with NO duplicate removal, 0x5330d0/0x536af0); whether the client
// sends a clearing packet for the old key was not found, so this keeps the
// one-key-per-skill behaviour as the design choice.
func (b *SkillBar) Assign(slot, skill int, left bool) error {
	if slot < 0 || slot >= HotkeyCount {
		return fmt.Errorf("%w: %d", ErrBadSlot, slot)
	}

	if skill < 0 {
		return b.Clear(slot)
	}

	for i := range b.Hotkeys {
		if b.Hotkeys[i].Skill == skill {
			b.Hotkeys[i] = SkillSlot{Skill: NoSkill}
		}
	}

	b.Hotkeys[slot] = SkillSlot{Skill: skill, Left: left}

	return nil
}

// Clear empties a hotkey.
func (b *SkillBar) Clear(slot int) error {
	if slot < 0 || slot >= HotkeyCount {
		return fmt.Errorf("%w: %d", ErrBadSlot, slot)
	}

	b.Hotkeys[slot] = SkillSlot{Skill: NoSkill}

	return nil
}

// HotkeyOf returns the hotkey a skill sits on, or -1.
func (b *SkillBar) HotkeyOf(skill int) int {
	if b == nil {
		return -1
	}

	for i, s := range b.Hotkeys {
		if s.Skill == skill && skill >= 0 {
			return i
		}
	}

	return -1
}

// Press returns what a hotkey selects. ok is false for an empty or bad slot.
func (b *SkillBar) Press(slot int) (s SkillSlot, ok bool) {
	if slot < 0 || slot >= HotkeyCount || b.Hotkeys[slot].Empty() {
		return SkillSlot{}, false
	}

	return b.Hotkeys[slot], true
}

// Select makes a skill the active one of a button.
func (b *SkillBar) Select(left bool, skill int) {
	if left {
		b.Left = SkillSlot{Skill: skill}
	} else {
		b.Right = SkillSlot{Skill: skill}
	}
}

// Active returns the active skill id of a button.
func (b *SkillBar) Active(left bool) int {
	if left {
		return b.Left.Skill
	}

	return b.Right.Skill
}

// SwapSets exchanges the active skills with the weapon swap set (the weapon
// switch key).
func (b *SkillBar) SwapSets() {
	b.Left, b.LeftSwap = b.LeftSwap, b.Left
	b.Right, b.RightSwap = b.RightSwap, b.Right
}

// Selectable reports whether a skill may be put on a button: the hero must have
// learned it (one point, or a base skill such as Attack) and it must not be
// passive; the left button also needs skills.txt "leftskill". (The original's
// server checks the same on C2S 0x3C; single player calls this directly.)
func Selectable(s *HeroSkill, left bool) bool {
	if s == nil || s.SkillRecord == nil || s.SkillPoints < 1 || s.Passive {
		return false
	}

	return !left || s.Leftskill
}

// FixActive resets the active buttons to Attack when they name a skill the hero
// cannot use (a save from another build, a respec). Hotkeys are left alone:
// they may name item skills.
func (b *SkillBar) FixActive(skills map[int]*HeroSkill) {
	if !Selectable(skills[b.Left.Skill], true) {
		b.Left = SkillSlot{}
	}

	if !Selectable(skills[b.Right.Skill], false) {
		b.Right = SkillSlot{}
	}
}
