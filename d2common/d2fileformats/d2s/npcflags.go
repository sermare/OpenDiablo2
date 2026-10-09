package d2s

// NPCBlock is the NPC introduction/return section payload (the 50 bytes after
// the 0x01 0x77 tag). Layout (matches the nokka reference parser): byte 0 is
// the section size 0x34, byte 1 padding, then three 8 byte "introduced"
// bitfields (normal, nightmare, hell) and three 8 byte "return" bitfields.
// Which bit means which NPC is NOT verified: the available sample saves have
// the block all zero. The reference parser only reads 5 intro and 4 return
// bytes per difficulty, so higher bits are probably unused. The Go side only
// offers raw bit access; everything else stays untouched for round-trip.
type NPCBlock [npcSize - 2]byte

const (
	npcIntroStart  = 2
	npcReturnStart = npcIntroStart + numDifficulties*8
	npcFieldLen    = 8
)

// NPCFlags returns a view of the NPC block of a body; writes reach Write.
func (b *Body) NPCFlags() *NPCBlock { return (*NPCBlock)(&b.NPC) }

func (n *NPCBlock) get(start, difficulty int) uint64 {
	if difficulty < 0 || difficulty >= numDifficulties {
		return 0
	}

	var v uint64

	for i := npcFieldLen - 1; i >= 0; i-- {
		v = v<<8 | uint64(n[start+difficulty*npcFieldLen+i])
	}

	return v
}

func (n *NPCBlock) set(start, difficulty int, v uint64) {
	if difficulty < 0 || difficulty >= numDifficulties {
		return
	}

	for i := 0; i < npcFieldLen; i++ {
		n[start+difficulty*npcFieldLen+i] = byte(v >> (8 * uint(i)))
	}
}

// Intro returns the 64 "has been introduced" bits of a difficulty.
func (n *NPCBlock) Intro(difficulty int) uint64 { return n.get(npcIntroStart, difficulty) }

// Return returns the 64 "greet with the return line" bits of a difficulty.
func (n *NPCBlock) Return(difficulty int) uint64 { return n.get(npcReturnStart, difficulty) }

// IntroBit tests one introduction bit (0..63).
func (n *NPCBlock) IntroBit(difficulty, bit int) bool {
	return bit >= 0 && bit < 64 && n.Intro(difficulty)>>uint(bit)&1 != 0
}

// ReturnBit tests one return bit (0..63).
func (n *NPCBlock) ReturnBit(difficulty, bit int) bool {
	return bit >= 0 && bit < 64 && n.Return(difficulty)>>uint(bit)&1 != 0
}

// SetIntroBit sets or clears one introduction bit.
func (n *NPCBlock) SetIntroBit(difficulty, bit int, on bool) {
	n.setBit(npcIntroStart, difficulty, bit, on)
}

// SetReturnBit sets or clears one return bit.
func (n *NPCBlock) SetReturnBit(difficulty, bit int, on bool) {
	n.setBit(npcReturnStart, difficulty, bit, on)
}

func (n *NPCBlock) setBit(start, difficulty, bit int, on bool) {
	if bit < 0 || bit >= 64 {
		return
	}

	v := n.get(start, difficulty)
	if on {
		v |= 1 << uint(bit)
	} else {
		v &^= 1 << uint(bit)
	}

	n.set(start, difficulty, v)
}
