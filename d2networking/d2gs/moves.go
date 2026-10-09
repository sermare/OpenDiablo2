package d2gs

// Movement and skill packets used by the multiplayer transport.

// MoveToLocation is a walk (0x01) or run (0x03) order, 5 bytes: u16 x @1,
// u16 y @3 in sub-tile coordinates. Verified in the handlers 0x547690 and
// 0x5477a0 (they share the length/position helper 0x5475b0 and differ only in
// the movement mode passed on: 2 = walk, 3 = run).
type MoveToLocation struct {
	Run  bool
	X, Y uint16
}

// PacketID implements Message.
func (m MoveToLocation) PacketID() byte {
	if m.Run {
		return C2SRunToLocation
	}

	return C2SWalkToLocation
}

// MarshalPacket implements Message.
func (m MoveToLocation) MarshalPacket() []byte {
	w := newW(m.PacketID(), 5)
	w.u16(1, m.X)
	w.u16(3, m.Y)

	return w
}

func (m *MoveToLocation) unmarshal(b []byte) { r := rbuf(b); m.X, m.Y = r.u16(1), r.u16(3) }

// CastOnLocation casts the selected left (0x05) or right (0x0c) skill on a
// sub-tile location. Layout 5 bytes like MoveToLocation (verified: 0x547bf0 and
// 0x547ed0 call the same helper 0x5475b0 and then the cast routine 0x5479c0).
type CastOnLocation struct {
	Right bool
	X, Y  uint16
}

// PacketID implements Message.
func (m CastOnLocation) PacketID() byte {
	if m.Right {
		return C2SCastRightLocation
	}

	return C2SCastLeftLocation
}

// MarshalPacket implements Message.
func (m CastOnLocation) MarshalPacket() []byte {
	w := newW(m.PacketID(), 5)
	w.u16(1, m.X)
	w.u16(3, m.Y)

	return w
}

func (m *CastOnLocation) unmarshal(b []byte) { r := rbuf(b); m.X, m.Y = r.u16(1), r.u16(3) }

// SelectSkill (0x3c, 9 bytes) selects the active skill of a hand. Verified
// from SERVER_HandleC2SSelectSkill (0x549ca0): u32 @1 = skill id, bit 31 set
// for the right hand; u32 @5 = id of the item that grants the skill
// (0xFFFFFFFF for none; unverified, only "non-zero value is looked up").
type SelectSkill struct {
	Skill  uint32
	Right  bool
	ItemID uint32
}

// PacketID implements Message.
func (SelectSkill) PacketID() byte { return C2SSelectSkill }

// MarshalPacket implements Message.
func (m SelectSkill) MarshalPacket() []byte {
	w := newW(C2SSelectSkill, 9)
	v := m.Skill & 0x7fffffff
	if m.Right {
		v |= 0x80000000
	}

	w.u32(1, v)
	w.u32(5, m.ItemID)

	return w
}

func (m *SelectSkill) unmarshal(b []byte) {
	r := rbuf(b)
	v := r.u32(1)
	m.Skill, m.Right, m.ItemID = v&0x7fffffff, v&0x80000000 != 0, r.u32(5)
}
