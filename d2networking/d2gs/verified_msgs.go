package d2gs

// Field layouts verified from the handler bodies in Game.exe (read-only
// Ghidra, see d2-re-notes/verify-packets.md). Anything not confirmed there is
// marked UNVERIFIED.

// bitReader reads an LSB-first bit stream, like the exe's reader at 0x40ca50:
// bits fill the low end of each byte first and the first field read lands in
// the low bits of the result. Reading past the end yields zero bits.
type bitReader struct {
	b   []byte
	pos int // bit position
}

func (r *bitReader) read(n int) uint32 {
	var v uint32

	for i := 0; i < n; i++ {
		p := r.pos + i
		if p>>3 < len(r.b) && r.b[p>>3]>>(uint(p)&7)&1 != 0 {
			v |= 1 << uint(i)
		}
	}

	r.pos += n

	return v
}

type bitWriter struct {
	b   []byte
	pos int
}

func (w *bitWriter) write(n int, v uint32) {
	for i := 0; i < n; i++ {
		if v>>uint(i)&1 != 0 {
			p := w.pos + i
			w.b[p>>3] |= 1 << (uint(p) & 7)
		}
	}

	w.pos += n
}

// signedDelta mirrors the exe's byte to delta conversion in the 0x18 / 0x95
// handlers: values above 0x80 become value-0x100, 0x80 itself stays +128.
func signedDelta(b uint32) int16 {
	if b > 0x80 {
		return int16(b) - 0x100
	}

	return int16(b)
}

func deltaByte(d int16) uint32 {
	switch {
	case d > 0x80:
		d = 0x80
	case d < -0x7f:
		d = -0x7f
	}

	return uint32(d) & 0xff
}

const (
	lifeManaSize   = 13 // 0x95
	vitalsSize     = 15 // 0x18
	assignPlayerSz = 26 // 0x59
	vitalBits      = 15
	vitalMask      = 1<<vitalBits - 1
)

// LifeMana is S2C 0x95 (13 bytes), handler 0x459240, bit packed LSB first by
// 0x4591b0 after the id byte: life:15, mana:15, stamina:15, x:16, y:16,
// dx:8, dy:8 (101 bits, 3 spare). Verified. The handler stores life, mana and
// stamina into unit stats 6, 8, 10 shifted left by 8 (fixed point) and moves
// the player towards (x+dx, y+dy). UNVERIFIED: the exact meaning of the x/y
// pair (current position, with dx/dy the offset to the target).
type LifeMana struct {
	Life, Mana, Stamina uint16 // 15 bits each
	X, Y                uint16
	DX, DY              int16 // range -127..128, see signedDelta
}

// Marshal returns the 13 byte packet.
func (m LifeMana) Marshal() []byte {
	b := newW(S2CPlayerLifeMana, lifeManaSize)
	w := bitWriter{b: b, pos: 8}
	w.write(vitalBits, uint32(m.Life))
	w.write(vitalBits, uint32(m.Mana))
	w.write(vitalBits, uint32(m.Stamina))
	w.write(16, uint32(m.X))
	w.write(16, uint32(m.Y))
	w.write(8, deltaByte(m.DX))
	w.write(8, deltaByte(m.DY))

	return b
}

// ParseLifeMana decodes a 0x95 packet.
func ParseLifeMana(b []byte) (LifeMana, error) {
	if len(b) != lifeManaSize || b[0] != S2CPlayerLifeMana {
		return LifeMana{}, ErrWrongSize
	}

	r := bitReader{b: b, pos: 8}
	m := LifeMana{Life: uint16(r.read(vitalBits)), Mana: uint16(r.read(vitalBits)), Stamina: uint16(r.read(vitalBits))}
	m.X, m.Y = uint16(r.read(16)), uint16(r.read(16))
	m.DX, m.DY = signedDelta(r.read(8)), signedDelta(r.read(8))

	return m, nil
}

// PlayerVitals is S2C 0x18 (15 bytes), handler 0x4590c0, unpacked by 0x459010:
// life:15, mana:15, stamina:15, A:7, B:7, x:16, y:16, dx:8, dy:8 (115 bits).
// Verified. A is stored as unit stat 0x4a and B as stat 0x1a without scaling;
// UNVERIFIED: what those two stats are called.
type PlayerVitals struct {
	Life, Mana, Stamina uint16 // 15 bits each
	StatA, StatB        uint8  // 7 bits each
	X, Y                uint16
	DX, DY              int16
}

// Marshal returns the 15 byte packet.
func (m PlayerVitals) Marshal() []byte {
	b := newW(S2CPlayerHPMana, vitalsSize)
	w := bitWriter{b: b, pos: 8}
	w.write(vitalBits, uint32(m.Life))
	w.write(vitalBits, uint32(m.Mana))
	w.write(vitalBits, uint32(m.Stamina))
	w.write(7, uint32(m.StatA))
	w.write(7, uint32(m.StatB))
	w.write(16, uint32(m.X))
	w.write(16, uint32(m.Y))
	w.write(8, deltaByte(m.DX))
	w.write(8, deltaByte(m.DY))

	return b
}

// ParsePlayerVitals decodes a 0x18 packet.
func ParsePlayerVitals(b []byte) (PlayerVitals, error) {
	if len(b) != vitalsSize || b[0] != S2CPlayerHPMana {
		return PlayerVitals{}, ErrWrongSize
	}

	r := bitReader{b: b, pos: 8}
	m := PlayerVitals{Life: uint16(r.read(vitalBits)), Mana: uint16(r.read(vitalBits)), Stamina: uint16(r.read(vitalBits))}
	m.StatA, m.StatB = uint8(r.read(7)), uint8(r.read(7))
	m.X, m.Y = uint16(r.read(16)), uint16(r.read(16))
	m.DX, m.DY = signedDelta(r.read(8)), signedDelta(r.read(8))

	return m, nil
}

// AssignPlayer is S2C 0x59 (26 bytes), handler 0x459c00 -> UNIT_CreatePlayerUnit
// (0x4619e0). Verified: u32 unit id @1, u8 class @5, 16 byte name @6, u16 x @22,
// u16 y @24 (the last two are used to find the room, so they are a position).
// This is the only 0x59 layout in the engine: the former PlayerInGame put a
// level and a party id at 22/24, which contradicts the real packet. Level and
// party travel in the real 0x5b RosterEntry and 0x75 / 0x8d (roster_msgs.go,
// verified); the tunnelled AddPlayer and roster stay as the OD2 extension for
// the rest of the hero state, hostility and invitations.
type AssignPlayer struct {
	UnitID uint32
	Class  uint8
	Name   string
	X, Y   uint16
}

// Marshal returns the packet.
func (m AssignPlayer) Marshal() []byte {
	b := newW(S2CPlayerInGame, assignPlayerSz)
	b.u32(1, m.UnitID)
	b[5] = m.Class
	putName(b[6:], m.Name)
	b.u16(22, m.X)
	b.u16(24, m.Y)

	return b
}

// ParseAssignPlayer decodes a 0x59 packet.
func ParseAssignPlayer(b []byte) (AssignPlayer, error) {
	if len(b) != assignPlayerSz || b[0] != S2CPlayerInGame {
		return AssignPlayer{}, ErrWrongSize
	}

	r := rbuf(b)

	return AssignPlayer{UnitID: r.u32(1), Class: b[5], Name: getName(b[6:]), X: r.u16(22), Y: r.u16(24)}, nil
}

// C2S ids added by the verification pass.
const (
	C2SSetQuestFlag   byte = 0x58 // 0x54a7d0, len 3
	C2SNpcInteractPar byte = 0x59 // 0x54a820, len 17
	C2SMoveChecked    byte = 0x5f // 0x54ab60, len 5
)

// SetQuestFlag (0x58, 3 bytes, verified): u16 quest record index @1 (< 0x2a,
// else return code 2). The handler sets bit 0xc of that entry in the quest
// record of the current act. UNVERIFIED: what bit 0xc means.
type SetQuestFlag struct{ Index uint16 }

// PacketID implements Message.
func (SetQuestFlag) PacketID() byte { return C2SSetQuestFlag }

// MarshalPacket implements Message.
func (m SetQuestFlag) MarshalPacket() []byte {
	w := newW(C2SSetQuestFlag, 3)
	w.u16(1, m.Index)

	return w
}

func (m *SetQuestFlag) unmarshal(b []byte) { m.Index = rbuf(b).u16(1) }

// NpcInteractParams (0x59, 17 bytes, verified from 0x54a820): u32 unit type @1
// (< 6), u32 unit id @5 (must be within range 0x32 of the player), u32 @9 and
// u32 @0xd stored as unit values 2 and 3 (value 1 is set to 0x28 by the
// handler). UNVERIFIED: the meaning of the two parameters.
type NpcInteractParams struct{ UnitType, UnitID, Param1, Param2 uint32 }

// PacketID implements Message.
func (NpcInteractParams) PacketID() byte { return C2SNpcInteractPar }

// MarshalPacket implements Message.
func (m NpcInteractParams) MarshalPacket() []byte {
	w := newW(C2SNpcInteractPar, 17)
	w.u32(1, m.UnitType)
	w.u32(5, m.UnitID)
	w.u32(9, m.Param1)
	w.u32(13, m.Param2)

	return w
}

func (m *NpcInteractParams) unmarshal(b []byte) {
	r := rbuf(b)
	m.UnitType, m.UnitID, m.Param1, m.Param2 = r.u32(1), r.u32(5), r.u32(9), r.u32(13)
}

// MoveChecked (0x5f, 5 bytes, verified from 0x54ab60): u16 x @1, u16 y @3, a
// checked move to a location (server validates distance and path).
type MoveChecked struct{ X, Y uint16 }

// PacketID implements Message.
func (MoveChecked) PacketID() byte { return C2SMoveChecked }

// MarshalPacket implements Message.
func (m MoveChecked) MarshalPacket() []byte {
	w := newW(C2SMoveChecked, 5)
	w.u16(1, m.X)
	w.u16(3, m.Y)

	return w
}

func (m *MoveChecked) unmarshal(b []byte) { r := rbuf(b); m.X, m.Y = r.u16(1), r.u16(3) }
