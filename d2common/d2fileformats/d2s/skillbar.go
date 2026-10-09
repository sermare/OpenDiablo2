package d2s

import "encoding/binary"

// The skill assignment block of the header: 16 hotkey words at 0x38 (4 bytes
// each), then the left and right skill (0x78, 0x7C) and the left and right
// skill of the weapon swap set (0x80, 0x84). Every word is a u16 skill id
// followed by a u16 item id; a hotkey word's skill carries bit 0x8000 when the
// skill was assigned for the left hand, and 0xFFFF means "no skill".
// (Layout confirmed by the real NokkaSorc save and the nokka-d2s reference;
// the item id of the real saves seen so far is always 0.)
const (
	// HotkeyCount is the number of skill hotkey slots a character has.
	HotkeyCount = 16

	hotkeysOffset   = 0x38
	leftSkillOff    = 0x78
	rightSkillOff   = 0x7C
	leftSwapSkillOf = 0x80
	rightSwapSkillO = 0x84
)

// SkillWord is one 4-byte entry of the skill block: the raw skill u16 (hand
// flag included) and the item id u16.
type SkillWord struct {
	Skill uint16
	Item  uint16
}

// SkillBlock is the skill assignment part of the header.
type SkillBlock struct {
	Hotkeys   [HotkeyCount]SkillWord
	Left      SkillWord
	Right     SkillWord
	LeftSwap  SkillWord
	RightSwap SkillWord
}

func readWord(raw []byte) SkillWord {
	return SkillWord{Skill: binary.LittleEndian.Uint16(raw), Item: binary.LittleEndian.Uint16(raw[2:])}
}

func (w SkillWord) put(raw []byte) {
	binary.LittleEndian.PutUint16(raw, w.Skill)
	binary.LittleEndian.PutUint16(raw[2:], w.Item)
}

// SkillBlock decodes the assigned, left, right and swap skills.
func (h *Header) SkillBlock() SkillBlock {
	var b SkillBlock

	for i := range b.Hotkeys {
		b.Hotkeys[i] = readWord(h.Raw[hotkeysOffset+4*i:])
	}

	b.Left, b.Right = readWord(h.Raw[leftSkillOff:]), readWord(h.Raw[rightSkillOff:])
	b.LeftSwap, b.RightSwap = readWord(h.Raw[leftSwapSkillOf:]), readWord(h.Raw[rightSwapSkillO:])

	return b
}

// SetSkillBlock writes the skill assignments into the header (Raw, which
// Write starts from).
func (h *Header) SetSkillBlock(b SkillBlock) {
	for i, w := range b.Hotkeys {
		w.put(h.Raw[hotkeysOffset+4*i:])
	}

	b.Left.put(h.Raw[leftSkillOff:])
	b.Right.put(h.Raw[rightSkillOff:])
	b.LeftSwap.put(h.Raw[leftSwapSkillOf:])
	b.RightSwap.put(h.Raw[rightSwapSkillO:])
}
