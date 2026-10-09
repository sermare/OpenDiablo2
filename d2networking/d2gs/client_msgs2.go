package d2gs

// Client -> server packets added by the packet oracle audit. Sizes and the
// fields marked "verified" come from the handler bodies in Game.exe (read-only
// Ghidra, handler addresses in each comment); everything else says UNVERIFIED.

// Ids of the packets below.
const (
	C2SToggleState12 byte = 0x12 // 0x548190, len 1
	C2SAllocateStat  byte = 0x3a // 0x549b40, len 3
	C2SAddSkillPoint byte = 0x3b // 0x549bc0, len 3
	C2SNpcTrade      byte = 0x38 // 0x549ad0, len 13
	C2SSetHotkey     byte = 0x51 // 0x54a690, len 9
	C2SPartyRelation byte = 0x5d // 0x54a0c0, len 7
	C2SPartyRequest  byte = 0x5e // 0x54a120, len 6
	// C2SOverheadChat (0x14, 0x5481b0) is variable: 4..0x113 bytes, text at +3.
	// Like 0x15 it cannot be framed from a byte stream without a transport length.
	C2SOverheadChat byte = 0x14
)

// UnitOrder is a walk (0x02), run (0x04) or left/right skill order (0x06, 0x07,
// 0x09, 0x0a / 0x0d, 0x0e, 0x10, 0x11) aimed at a unit: u32 unit type @1 (the
// handlers answer return code 2 above 5), u32 unit id @5, length 9. Verified:
// 0x547740 (02), 0x5477f0 (04), 0x547c70 (06), 0x547d00 (07), 0x547df0 (09),
// 0x547e50 (0a), 0x547f50 (0d), 0x547fe0 (0e), 0x5480d0 (10), 0x548130 (11).
// Which skill id is "hold" or "ex" comes from handler names only (UNVERIFIED).
type UnitOrder struct {
	ID               byte
	UnitType, UnitID uint32
}

// PacketID implements Message.
func (m UnitOrder) PacketID() byte { return m.ID }

// MarshalPacket implements Message.
func (m UnitOrder) MarshalPacket() []byte { return two(m.ID, m.UnitType, m.UnitID) }

func (m *UnitOrder) unmarshal(b []byte) { r := rbuf(b); m.UnitType, m.UnitID = r.u32(1), r.u32(5) }

// LocationOrder is the held cast on a location: left (0x08) or right (0x0f).
// Verified: 0x547d90 and 0x548070 parse the packet with the same helper as
// 0x05/0x0c (length 5, u16 x @1, u16 y @3) and then call those handlers.
type LocationOrder struct {
	ID   byte
	X, Y uint16
}

// PacketID implements Message.
func (m LocationOrder) PacketID() byte { return m.ID }

// MarshalPacket implements Message.
func (m LocationOrder) MarshalPacket() []byte {
	w := newW(m.ID, 5)
	w.u16(1, m.X)
	w.u16(3, m.Y)

	return w
}

func (m *LocationOrder) unmarshal(b []byte) { r := rbuf(b); m.X, m.Y = r.u16(1), r.u16(3) }

// ToggleState12 (0x12, 1 byte, verified): toggles unit state 0x0c.
type ToggleState12 struct{}

// PacketID implements Message.
func (ToggleState12) PacketID() byte { return C2SToggleState12 }

// MarshalPacket implements Message.
func (ToggleState12) MarshalPacket() []byte { return []byte{C2SToggleState12} }

func (*ToggleState12) unmarshal([]byte) {}

// AllocateStat (0x3a, 3 bytes). Verified: the u16 at +1 is split into a low byte
// (< 0x10) and a high byte (< 100); the handler then allocates high+1 points.
// UNVERIFIED: that the low byte is the stat index (the callee takes it from a
// register that was not traced).
type AllocateStat struct{ Stat, Extra uint8 }

// PacketID implements Message.
func (AllocateStat) PacketID() byte { return C2SAllocateStat }

// MarshalPacket implements Message.
func (m AllocateStat) MarshalPacket() []byte { return []byte{C2SAllocateStat, m.Stat, m.Extra} }

func (m *AllocateStat) unmarshal(b []byte) { m.Stat, m.Extra = b[1], b[2] }

// AddSkillPoint (0x3b, 3 bytes, verified length). UNVERIFIED: u16 skill id @1
// (the callee reads it from a register that was not traced).
type AddSkillPoint struct{ Skill uint16 }

// PacketID implements Message.
func (AddSkillPoint) PacketID() byte { return C2SAddSkillPoint }

// MarshalPacket implements Message.
func (m AddSkillPoint) MarshalPacket() []byte {
	w := newW(C2SAddSkillPoint, 3)
	w.u16(1, m.Skill)

	return w
}

func (m *AddSkillPoint) unmarshal(b []byte) { m.Skill = rbuf(b).u16(1) }

// SetHotkey (0x51, 9 bytes, verified from 0x54a690): u32 @1 = slot<<16 |
// right<<15 | skill (15 bits; slot must be < 16, else return code 3), u32 @5 =
// id of the item that grants the skill (passed to SKILL_FindByIdAndItem).
type SetHotkey struct {
	Slot   uint16
	Skill  uint16
	Right  bool
	ItemID uint32
}

// PacketID implements Message.
func (SetHotkey) PacketID() byte { return C2SSetHotkey }

// MarshalPacket implements Message.
func (m SetHotkey) MarshalPacket() []byte {
	v := uint32(m.Slot)<<16 | uint32(m.Skill&0x7fff)
	if m.Right {
		v |= 0x8000
	}

	return two(C2SSetHotkey, v, m.ItemID)
}

func (m *SetHotkey) unmarshal(b []byte) {
	r := rbuf(b)
	v := r.u32(1)
	m.Slot, m.Skill, m.Right, m.ItemID = uint16(v>>16), uint16(v&0x7fff), v&0x8000 != 0, r.u32(5)
}

// NpcTrade (0x38, 13 bytes, verified length): three u32 handed to
// TRADE_ServerHandleNpcMenuAction. The field meanings are UNVERIFIED (public
// docs: menu action, npc id, extra).
type NpcTrade struct{ Action, NpcID, Param uint32 }

// PacketID implements Message.
func (NpcTrade) PacketID() byte { return C2SNpcTrade }

// MarshalPacket implements Message.
func (m NpcTrade) MarshalPacket() []byte {
	w := newW(C2SNpcTrade, 13)
	w.u32(1, m.Action)
	w.u32(5, m.NpcID)
	w.u32(9, m.Param)

	return w
}

func (m *NpcTrade) unmarshal(b []byte) {
	r := rbuf(b)
	m.Action, m.NpcID, m.Param = r.u32(1), r.u32(5), r.u32(9)
}

// PartyRequest (0x5e, 6 bytes, verified): byte action @1 (<= 10), u32 player id @2.
type PartyRequest struct {
	Action   uint8
	PlayerID uint32
}

// PacketID implements Message.
func (PartyRequest) PacketID() byte { return C2SPartyRequest }

// MarshalPacket implements Message.
func (m PartyRequest) MarshalPacket() []byte {
	w := newW(C2SPartyRequest, 6)
	w[1] = m.Action
	w.u32(2, m.PlayerID)

	return w
}

func (m *PartyRequest) unmarshal(b []byte) { m.Action, m.PlayerID = b[1], rbuf(b).u32(2) }

// PartyRelation (0x5d, 7 bytes, verified): byte action @1 (<= 10), byte flag @2,
// u32 player id @3.
type PartyRelation struct {
	Action, Flag uint8
	PlayerID     uint32
}

// PacketID implements Message.
func (PartyRelation) PacketID() byte { return C2SPartyRelation }

// MarshalPacket implements Message.
func (m PartyRelation) MarshalPacket() []byte {
	w := newW(C2SPartyRelation, 7)
	w[1], w[2] = m.Action, m.Flag
	w.u32(3, m.PlayerID)

	return w
}

func (m *PartyRelation) unmarshal(b []byte) {
	m.Action, m.Flag, m.PlayerID = b[1], b[2], rbuf(b).u32(3)
}
