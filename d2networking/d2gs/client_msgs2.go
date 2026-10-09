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
// Verified (0x547c70, 0x547d00, 0x547df0, 0x547e50, 0x547f50, 0x547fe0, 0x5480d0):
// no skill id travels in these packets, the server uses the unit's selected
// left (0x06/0x07/0x09/0x0a) or right (0x0d/0x0e/0x10/0x11) skill. 0x06 and 0x0d
// store the target with flag 1, 0x07 and 0x0e with flag 0 (UNVERIFIED: that the
// flag is "shift held"); 0x09/0x0a and 0x10 are thin wrappers that check the
// target and then run the 0x06/0x07/0x0d body (0x11 -> 0x0e by symmetry,
// UNVERIFIED: 0x548130 not read). So "hold" and "ex" differ only in that flag
// or in being the wrapper path, never in the wire layout.
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

// AllocateStat (0x3a, 3 bytes, handler 0x549b40). Verified: the u16 at +1 is
// split into a low byte (<= 0x0f, else return code 3) which is passed as the
// stat argument to the allocate-one-point helper 0x56ec40, and a high byte
// (<= 99) = Extra; the handler calls the helper Extra+1 times and stops with
// return code 2 at the first failure. UNVERIFIED: the mapping of the stat byte
// values to strength/energy/dexterity/vitality (0..3 in the public docs).
type AllocateStat struct{ Stat, Extra uint8 }

// Count is the number of points the packet asks for (Extra+1).
func (m AllocateStat) Count() int { return int(m.Extra) + 1 }

// Valid mirrors the handler's range checks (stat <= 0x0f, count <= 100).
func (m AllocateStat) Valid() bool { return m.Stat <= 0x0f && m.Extra <= 99 }

// PacketID implements Message.
func (AllocateStat) PacketID() byte { return C2SAllocateStat }

// MarshalPacket implements Message.
func (m AllocateStat) MarshalPacket() []byte { return []byte{C2SAllocateStat, m.Stat, m.Extra} }

func (m *AllocateStat) unmarshal(b []byte) { m.Stat, m.Extra = b[1], b[2] }

// AddSkillPoint (0x3b, 3 bytes, handler 0x549bc0). Verified: u16 skill id @1
// (it is passed to the can-add check 0x547370, the level lookup 0x644dc0 and
// the prerequisite check 0x56df60). The server may refuse with code 2.
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

// NpcTrade (0x38, 13 bytes, handler 0x549ad0): three u32 handed to
// 0x577b70(game, client, Action, NpcID, Param). Verified: NpcID @5 is checked
// as a unit of type 1 (monster) within range 0x32 by 0x546e70 before the call;
// Action @1 and Param @9 are passed through unchanged, their meaning is
// UNVERIFIED (public docs: menu action, extra).
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
