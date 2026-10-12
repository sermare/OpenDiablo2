package d2gs

// Server -> client attribute, skill and world packets, layouts read from the
// Game.exe 1.14b client handlers and (where noted) the server senders.
// Anything not confirmed is marked UNVERIFIED.

// Ids.
const (
	S2CAttrAddGoldByte  byte = 0x19 // size 2
	S2CAttrAddExpByte   byte = 0x1a // size 2
	S2CAttrAddExpWord   byte = 0x1b // size 3
	S2CAttrSetExpDword  byte = 0x1c // size 5
	S2CAttrSetByte      byte = 0x1d // size 3
	S2CAttrSetWord      byte = 0x1e // size 4
	S2CAttrSetDword     byte = 0x1f // size 6
	S2CSetPlayerStat    byte = 0x20 // size 10, handler 0x458f90, sender 0x539e20
	S2CUpdateSkillLevel byte = 0x21 // size 12, handler 0x459400
	S2CSetActiveSkill   byte = 0x23 // size 13, handler 0x459540
	S2CWaypointMenu     byte = 0x63 // size 21, handler 0x459db0, sender 0x53b720
	S2CMissileCreate    byte = 0x73 // size 32, handler 0x459e10, sender 0x53b7d0
	S2CSkillSlot        byte = 0x7b // size 8, handler 0x45a020

	statGold = 14 // unit stat 0x0e
	statExp  = 13 // unit stat 0x0d
)

// AttrUpdate is the attribute family 0x19..0x1f, all served by the handler at
// 0x458ea0 (jump table at 0x458f70). Verified from the handler:
//
//	0x19 [1]=byte  adds to unit stat 14 (gold)       0x1d [1]=stat, [2]=byte   sets stat
//	0x1a [1]=byte  adds to unit stat 13 (experience) 0x1e [1]=stat, [2..3]=u16 sets stat
//	0x1b [1..2]=u16 adds to stat 13                  0x1f [1]=stat, [2..5]=u32 sets stat
//	0x1c [1..4]=u32 SETS stat 13
//
// The sender 0x539a70 picks 0x1d when the value is below 0xff, 0x1e below 0xffff
// and 0x1f otherwise (stat id at most 0xfe). The "gold" / "experience" names for
// stats 14 / 13 follow the standard stat table (UNVERIFIED in this binary).
// Value holds the number as it travels; for 0x19-0x1b it is the amount to add.
type AttrUpdate struct {
	ID    byte
	Stat  uint8 // only for 0x1d-0x1f; the other ids imply it, see StatID
	Value uint32
}

// SetStat builds the shortest set packet for a stat the way sender 0x539a70 does.
func SetStat(stat uint8, value uint32) AttrUpdate {
	switch {
	case value < 0xff:
		return AttrUpdate{ID: S2CAttrSetByte, Stat: stat, Value: value}
	case value < 0xffff:
		return AttrUpdate{ID: S2CAttrSetWord, Stat: stat, Value: value}
	}

	return AttrUpdate{ID: S2CAttrSetDword, Stat: stat, Value: value}
}

// Add reports whether the packet adds to the stat instead of setting it.
func (m AttrUpdate) Add() bool { return m.ID >= S2CAttrAddGoldByte && m.ID <= S2CAttrAddExpWord }

// StatID is the unit stat the packet changes.
func (m AttrUpdate) StatID() uint8 {
	switch m.ID {
	case S2CAttrAddGoldByte:
		return statGold
	case S2CAttrAddExpByte, S2CAttrAddExpWord, S2CAttrSetExpDword:
		return statExp
	}

	return m.Stat
}

// Marshal returns the packet. A Value wider than the id allows is truncated;
// an id outside 0x19..0x1f yields nil.
func (m AttrUpdate) Marshal() []byte {
	var b wbuf

	switch m.ID {
	case S2CAttrAddGoldByte, S2CAttrAddExpByte:
		b = newW(m.ID, 2)
		b[1] = byte(m.Value)
	case S2CAttrAddExpWord:
		b = newW(m.ID, 3)
		b.u16(1, uint16(m.Value))
	case S2CAttrSetExpDword:
		b = newW(m.ID, 5)
		b.u32(1, m.Value)
	case S2CAttrSetByte:
		b = newW(m.ID, 3)
		b[1], b[2] = m.Stat, byte(m.Value)
	case S2CAttrSetWord:
		b = newW(m.ID, 4)
		b[1] = m.Stat
		b.u16(2, uint16(m.Value))
	case S2CAttrSetDword:
		b = newW(m.ID, 6)
		b[1] = m.Stat
		b.u32(2, m.Value)
	default:
		return nil
	}

	return b
}

// ParseAttrUpdate decodes a 0x19..0x1f packet.
func ParseAttrUpdate(b []byte) (AttrUpdate, error) {
	if len(b) == 0 {
		return AttrUpdate{}, ErrShort
	}

	n, kind := ExpectedSize(b[0], ServerToClient)
	if b[0] < S2CAttrAddGoldByte || b[0] > S2CAttrSetDword || kind != SizeFixed || n != len(b) {
		return AttrUpdate{}, ErrWrongSize
	}

	r := rbuf(b)
	m := AttrUpdate{ID: b[0]}

	switch b[0] {
	case S2CAttrAddGoldByte, S2CAttrAddExpByte:
		m.Value = uint32(b[1])
	case S2CAttrAddExpWord:
		m.Value = uint32(r.u16(1))
	case S2CAttrSetExpDword:
		m.Value = r.u32(1)
	case S2CAttrSetByte:
		m.Stat, m.Value = b[1], uint32(b[2])
	case S2CAttrSetWord:
		m.Stat, m.Value = b[1], uint32(r.u16(2))
	case S2CAttrSetDword:
		m.Stat, m.Value = b[1], r.u32(2)
	}

	return m, nil
}

// SetPlayerStat (0x20, 10 bytes): handler 0x458f90 finds the player unit by the
// dword id @1, sets the stat given by byte @5 to dword @6 (raw value, no
// scaling) and refreshes the UI. Sender 0x539e20 agrees; stat id at most 0xfe.
type SetPlayerStat struct {
	UnitID uint32
	Stat   uint8
	Value  uint32
}

// Marshal returns the packet.
func (m SetPlayerStat) Marshal() []byte {
	b := newW(S2CSetPlayerStat, 10)
	b.u32(1, m.UnitID)
	b[5] = m.Stat
	b.u32(6, m.Value)

	return b
}

// ParseSetPlayerStat decodes a 0x20 packet.
func ParseSetPlayerStat(b []byte) (SetPlayerStat, error) {
	if err := checkS2C(b, S2CSetPlayerStat, 10); err != nil {
		return SetPlayerStat{}, err
	}

	r := rbuf(b)

	return SetPlayerStat{UnitID: r.u32(1), Stat: b[5], Value: r.u32(6)}, nil
}

// UpdateSkillLevel (0x21, 12 bytes): handler 0x459400 finds the unit by type
// byte @1 and id dword @3, then sets skill u16 @7 to level byte @9 (the 0x648640
// call). Byte @2 is passed along as well (UNVERIFIED: meaning). Bytes @10, @11
// are not read.
type UpdateSkillLevel struct {
	UnitType uint8
	Flag     uint8
	UnitID   uint32
	Skill    uint16
	Level    uint8
}

// Marshal returns the packet.
func (m UpdateSkillLevel) Marshal() []byte {
	b := newW(S2CUpdateSkillLevel, 12)
	b[1], b[2] = m.UnitType, m.Flag
	b.u32(3, m.UnitID)
	b.u16(7, m.Skill)
	b[9] = m.Level

	return b
}

// ParseUpdateSkillLevel decodes a 0x21 packet.
func ParseUpdateSkillLevel(b []byte) (UpdateSkillLevel, error) {
	if err := checkS2C(b, S2CUpdateSkillLevel, 12); err != nil {
		return UpdateSkillLevel{}, err
	}

	r := rbuf(b)

	return UpdateSkillLevel{UnitType: b[1], Flag: b[2], UnitID: r.u32(3), Skill: r.u16(7), Level: b[9]}, nil
}

// SetActiveSkill (0x23, 13 bytes): handler 0x459540 looks up the player by the
// dword id @2 (byte @1 is the unit type, not read), then assigns skill u16 @7
// with item id dword @9 to the hand chosen by byte @6 (non-zero calls 0x644fc0,
// zero calls 0x645050). UNVERIFIED: that non-zero means the right hand; the
// server sender for this packet was not read. 0x41 respawn handling calls the
// server routine with 1 for the left and 0 for the right skill, which suggests
// Hand 1 = left.
type SetActiveSkill struct {
	UnitType uint8
	UnitID   uint32
	Hand     uint8
	Skill    uint16
	ItemID   uint32
}

// Marshal returns the packet.
func (m SetActiveSkill) Marshal() []byte {
	b := newW(S2CSetActiveSkill, 13)
	b[1] = m.UnitType
	b.u32(2, m.UnitID)
	b[6] = m.Hand
	b.u16(7, m.Skill)
	b.u32(9, m.ItemID)

	return b
}

// ParseSetActiveSkill decodes a 0x23 packet.
func ParseSetActiveSkill(b []byte) (SetActiveSkill, error) {
	if err := checkS2C(b, S2CSetActiveSkill, 13); err != nil {
		return SetActiveSkill{}, err
	}

	r := rbuf(b)

	return SetActiveSkill{UnitType: b[1], UnitID: r.u32(2), Hand: b[6], Skill: r.u16(7), ItemID: r.u32(9)}, nil
}

// SkillSlot (0x7b, 8 bytes): handler 0x45a020 reads byte @1 (slot), u16 @2 whose
// low 12 bits are the skill id (an id above the skills table size becomes -1)
// and bit 15 a flag, and dword @4 (passed on to 0x4a67c0). UNVERIFIED: what the
// flag and the dword are.
type SkillSlot struct {
	Slot  uint8
	Skill uint16 // 12 bits
	Flag  bool   // bit 15 of the word at @2
	Param uint32
}

// Marshal returns the packet.
func (m SkillSlot) Marshal() []byte {
	b := newW(S2CSkillSlot, 8)
	b[1] = m.Slot

	w := m.Skill & 0xfff
	if m.Flag {
		w |= 0x8000
	}

	b.u16(2, w)
	b.u32(4, m.Param)

	return b
}

// ParseSkillSlot decodes a 0x7b packet.
func ParseSkillSlot(b []byte) (SkillSlot, error) {
	if err := checkS2C(b, S2CSkillSlot, 8); err != nil {
		return SkillSlot{}, err
	}

	r := rbuf(b)
	w := r.u16(2)

	return SkillSlot{Slot: b[1], Skill: w & 0xfff, Flag: w&0x8000 != 0, Param: r.u32(4)}, nil
}

// WaypointMenu (0x63, 21 bytes): the server opens the waypoint panel.
// OBJECT_OperateActivateWaypoint (0x582d00) builds it as [0]=0x63, dword @1 =
// the waypoint object's unit id, then 16 bytes @5..20 filled by 0x663780 from
// the player's waypoint record; the client handler 0x459db0 copies all of it as
// is into the panel state. UNVERIFIED: the layout inside Flags (it is the
// player's waypoint-known bit set; the port keeps it opaque).
type WaypointMenu struct {
	ObjectID uint32
	Flags    [16]byte
}

// Marshal returns the packet.
func (m WaypointMenu) Marshal() []byte {
	b := newW(S2CWaypointMenu, 21)
	b.u32(1, m.ObjectID)
	copy(b[5:], m.Flags[:])

	return b
}

// ParseWaypointMenu decodes a 0x63 packet.
func ParseWaypointMenu(b []byte) (WaypointMenu, error) {
	if err := checkS2C(b, S2CWaypointMenu, 21); err != nil {
		return WaypointMenu{}, err
	}

	m := WaypointMenu{ObjectID: rbuf(b).u32(1)}
	copy(m.Flags[:], b[5:21])

	return m, nil
}

// MissileCreate (0x73, 32 bytes). The sender MISSILE_SendMissileCreatePacket
// (0x59daa0 -> 0x53b7d0) fills it field by field and the handler 0x459e10 reads
// the same offsets (verified both sides): u16 @5 = missile class (UNVERIFIED:
// the missiles.txt row), dwords @7 and @11 = X and Y from the missile unit
// (0x456530 / 0x456560, UNVERIFIED units: the handler stores them as the start
// position), dwords @15 and @19 = target X and Y from the missile's path (sent
// only when the sender's flag argument is set, else 0), u16 @23 (0x64b620,
// UNVERIFIED), byte @25 = owner unit type, dword @26 = owner unit id, byte @30
// (0x64b500, UNVERIFIED) and byte @31 = the missile's stat 0x148 (UNVERIFIED
// name). Bytes @1..4 are never written by the sender and not read by the handler.
// The handler creates the missile with flags 0x40000021 when both target
// coordinates are non-zero, else 0x40000001.
type MissileCreate struct {
	Class            uint16
	X, Y             uint32
	TargetX, TargetY uint32
	W23              uint16
	OwnerType        uint8
	OwnerID          uint32
	B30              uint8
	Stat148          uint8
}

// Marshal returns the packet.
func (m MissileCreate) Marshal() []byte {
	b := newW(S2CMissileCreate, 32)
	b.u16(5, m.Class)
	b.u32(7, m.X)
	b.u32(11, m.Y)
	b.u32(15, m.TargetX)
	b.u32(19, m.TargetY)
	b.u16(23, m.W23)
	b[25] = m.OwnerType
	b.u32(26, m.OwnerID)
	b[30], b[31] = m.B30, m.Stat148

	return b
}

// ParseMissileCreate decodes a 0x73 packet.
func ParseMissileCreate(b []byte) (MissileCreate, error) {
	if err := checkS2C(b, S2CMissileCreate, 32); err != nil {
		return MissileCreate{}, err
	}

	r := rbuf(b)

	return MissileCreate{Class: r.u16(5), X: r.u32(7), Y: r.u32(11), TargetX: r.u32(15), TargetY: r.u32(19),
		W23: r.u16(23), OwnerType: b[25], OwnerID: r.u32(26), B30: b[30], Stat148: b[31]}, nil
}
