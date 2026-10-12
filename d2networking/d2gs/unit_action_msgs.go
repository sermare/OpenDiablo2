package d2gs

import "fmt"

// Server -> client unit-action packets. Layouts read read-only from the
// Game.exe 1.14b client handlers (d2-re-notes/w46-names2.md has the address
// table) and cross-checked against the server senders where one was read.
// Every handler first lets the dispatcher resolve the target unit, so the first
// bytes are the unit reference: 0x0c-0x10 and 0x4c use [1]=unit type,
// [2..5]=unit id; the monster movement family 0x67-0x6d has no type byte and
// carries only the dword id at [1..4]. The meaning of the byte-sized "extra"
// fields is not decoded; they are carried verbatim (marked UNVERIFIED) so that
// decode -> encode is byte-identical for every field the handler reads.

// S2C ids of the unit-action family (0x4d is S2CUnitSkillOnLoc).
const (
	S2CUnitActionBasic   byte = 0x0c // size 9, handler 0x4583e0
	S2CUnitStop          byte = 0x0d // size 13, handler 0x458430
	S2CObjectState       byte = 0x0e // size 12, handler 0x458480
	S2CMoveToEntity      byte = 0x10 // size 16, handler 0x458500
	S2CUnitSkillOnTarget byte = 0x4c // size 16, handler 0x459630
	S2CUnitSkillLocal    byte = 0x99 // size 16, copy of 0x4c for the local unit (UNVERIFIED)
	S2CNpcMoveToLoc      byte = 0x67 // size 16, handler 0x458550
	S2CNpcMoveToTarget   byte = 0x68 // size 21, handler 0x4585b0
	S2CNpcStateAtLoc     byte = 0x69 // size 12, handler 0x458620
	S2CNpcStateOnTarget  byte = 0x6a // size 12, handler 0x458680
	S2CNpcAttackLoc      byte = 0x6b // size 16, handler 0x4586d0
	S2CNpcAttackTarget   byte = 0x6c // size 16, handler 0x458700
	S2CNpcStop           byte = 0x6d // size 10, handler 0x458770
)

func checkS2C(b []byte, id byte, size int) error {
	if len(b) != size || b[0] != id {
		return fmt.Errorf("%w: want id %#x size %d, got id %#x size %d", ErrWrongSize, id, size, firstByte(b), len(b))
	}

	return nil
}

func firstByte(b []byte) byte {
	if len(b) == 0 {
		return 0
	}

	return b[0]
}

// UnitRef is the [1]=type, [2..5]=id header of the unit-action packets.
type UnitRef struct {
	Type uint8
	ID   uint32
}

func putRef(b wbuf, r UnitRef) { b[1] = r.Type; b.u32(2, r.ID) }
func getRef(b []byte) UnitRef  { return UnitRef{Type: b[1], ID: rbuf(b).u32(2)} }

// UnitActionBasic (0x0c, 9 bytes): handler 0x4583e0 reads mode @6 and two
// bytes @7, @8 that go into the action record. UNVERIFIED: what A and B mean.
type UnitActionBasic struct {
	Unit UnitRef
	Mode uint8
	A, B uint8
}

// Marshal returns the packet.
func (m UnitActionBasic) Marshal() []byte {
	b := newW(S2CUnitActionBasic, 9)
	putRef(b, m.Unit)
	b[6], b[7], b[8] = m.Mode, m.A, m.B

	return b
}

// ParseUnitActionBasic decodes a 0x0c packet.
func ParseUnitActionBasic(b []byte) (UnitActionBasic, error) {
	if err := checkS2C(b, S2CUnitActionBasic, 9); err != nil {
		return UnitActionBasic{}, err
	}

	return UnitActionBasic{Unit: getRef(b), Mode: b[6], A: b[7], B: b[8]}, nil
}

// UnitStop (0x0d, 13 bytes): sender 0x538ff0 (type must be <= 5) and handler
// 0x458430 agree on mode @6, u16 x @7, u16 y @9, byte @11, byte @12. For a
// player (type 0) the handler passes byte @12 to 0x476370 (UNVERIFIED: it is
// the life percentage used by the party health bar). Byte @11 goes into the
// action record (UNVERIFIED).
type UnitStop struct {
	Unit     UnitRef
	Mode     uint8
	X, Y     uint16
	B11, B12 uint8
}

// Marshal returns the packet.
func (m UnitStop) Marshal() []byte {
	b := newW(S2CUnitStop, 13)
	putRef(b, m.Unit)
	b[6] = m.Mode
	b.u16(7, m.X)
	b.u16(9, m.Y)
	b[11], b[12] = m.B11, m.B12

	return b
}

// ParseUnitStop decodes a 0x0d packet.
func ParseUnitStop(b []byte) (UnitStop, error) {
	if err := checkS2C(b, S2CUnitStop, 13); err != nil {
		return UnitStop{}, err
	}

	r := rbuf(b)

	return UnitStop{Unit: getRef(b), Mode: b[6], X: r.u16(7), Y: r.u16(9), B11: b[11], B12: b[12]}, nil
}

// ObjectState (0x0e, 12 bytes): handler 0x458480 reads mode @6, byte @7 and
// dword @8 into the action record (sender 0x538fa0 is the object state
// change). UNVERIFIED: meaning of State and Param.
type ObjectState struct {
	Unit  UnitRef
	Mode  uint8
	State uint8
	Param uint32
}

// Marshal returns the packet.
func (m ObjectState) Marshal() []byte {
	b := newW(S2CObjectState, 12)
	putRef(b, m.Unit)
	b[6], b[7] = m.Mode, m.State
	b.u32(8, m.Param)

	return b
}

// ParseObjectState decodes a 0x0e packet.
func ParseObjectState(b []byte) (ObjectState, error) {
	if err := checkS2C(b, S2CObjectState, 12); err != nil {
		return ObjectState{}, err
	}

	return ObjectState{Unit: getRef(b), Mode: b[6], State: b[7], Param: rbuf(b).u32(8)}, nil
}

// MoveToEntity (0x10, 16 bytes): handler 0x458500 first moves the unit to the
// position (u16 x @12, u16 y @14) with 0x47c320, then starts the action with
// mode @6, target type byte @7 and target id dword @8.
type MoveToEntity struct {
	Unit       UnitRef
	Mode       uint8
	TargetType uint8
	TargetID   uint32
	X, Y       uint16
}

// Marshal returns the packet.
func (m MoveToEntity) Marshal() []byte {
	b := newW(S2CMoveToEntity, 16)
	putRef(b, m.Unit)
	b[6], b[7] = m.Mode, m.TargetType
	b.u32(8, m.TargetID)
	b.u16(12, m.X)
	b.u16(14, m.Y)

	return b
}

// ParseMoveToEntity decodes a 0x10 packet.
func ParseMoveToEntity(b []byte) (MoveToEntity, error) {
	if err := checkS2C(b, S2CMoveToEntity, 16); err != nil {
		return MoveToEntity{}, err
	}

	r := rbuf(b)

	return MoveToEntity{Unit: getRef(b), Mode: b[6], TargetType: b[7], TargetID: r.u32(8), X: r.u16(12), Y: r.u16(14)}, nil
}

// UnitSkillOnTarget (0x4c, 16 bytes): a unit casts a skill at another unit.
// Sender 0x53b250 and handler 0x459630 agree: u16 skill @6, byte @8 (the same
// parameter slot that is the skill level in 0x4d, so called Level, meaning
// inferred), target type @9, target id dword @10, u16 @14 (written by the
// sender, not read by the handler). The sender emits 0x99 instead when its
// last argument is non-zero; Local selects that id (UNVERIFIED: that the client
// treats it like 0x4c).
type UnitSkillOnTarget struct {
	Local      bool
	Unit       UnitRef
	Skill      uint16
	Level      uint8
	TargetType uint8
	TargetID   uint32
	Extra      uint16
}

// Marshal returns the packet.
func (m UnitSkillOnTarget) Marshal() []byte {
	id := S2CUnitSkillOnTarget
	if m.Local {
		id = S2CUnitSkillLocal
	}

	b := newW(id, 16)
	putRef(b, m.Unit)
	b.u16(6, m.Skill)
	b[8], b[9] = m.Level, m.TargetType
	b.u32(10, m.TargetID)
	b.u16(14, m.Extra)

	return b
}

// ParseUnitSkillOnTarget decodes a 0x4c (or 0x99) packet.
func ParseUnitSkillOnTarget(b []byte) (UnitSkillOnTarget, error) {
	if len(b) != 16 || (b[0] != S2CUnitSkillOnTarget && b[0] != S2CUnitSkillLocal) {
		return UnitSkillOnTarget{}, ErrWrongSize
	}

	r := rbuf(b)

	return UnitSkillOnTarget{Local: b[0] == S2CUnitSkillLocal, Unit: getRef(b), Skill: r.u16(6), Level: b[8],
		TargetType: b[9], TargetID: r.u32(10), Extra: r.u16(14)}, nil
}

// NpcMoveToLocation (0x67, 16 bytes), handler 0x458550: id @1, mode @5, u16 x
// @6, u16 y @8, bytes @10, @11, @12, int16 @13, byte @15 (all stored into the
// action record; the id 0x67 itself is inferred, see w46-names2.md).
// UNVERIFIED: meaning of B10, B11, B12, W13, B15.
type NpcMoveToLocation struct {
	ID            uint32
	Mode          uint8
	X, Y          uint16
	B10, B11, B12 uint8
	W13           int16
	B15           uint8
}

// Marshal returns the packet.
func (m NpcMoveToLocation) Marshal() []byte {
	b := newW(S2CNpcMoveToLoc, 16)
	b.u32(1, m.ID)
	b[5] = m.Mode
	b.u16(6, m.X)
	b.u16(8, m.Y)
	b[10], b[11], b[12] = m.B10, m.B11, m.B12
	b.u16(13, uint16(m.W13))
	b[15] = m.B15

	return b
}

// ParseNpcMoveToLocation decodes a 0x67 packet.
func ParseNpcMoveToLocation(b []byte) (NpcMoveToLocation, error) {
	if err := checkS2C(b, S2CNpcMoveToLoc, 16); err != nil {
		return NpcMoveToLocation{}, err
	}

	r := rbuf(b)

	return NpcMoveToLocation{ID: r.u32(1), Mode: b[5], X: r.u16(6), Y: r.u16(8), B10: b[10], B11: b[11], B12: b[12],
		W13: int16(r.u16(13)), B15: b[15]}, nil
}

// NpcMoveToTarget (0x68, 21 bytes), handler 0x4585b0: id @1, mode @5. The
// handler first corrects the position to (u16 x @6, u16 y @8) with 0x47c320,
// then starts the action with byte @10 (target type, by analogy with 0x6a),
// dword @11 (target id), bytes @15, @16, @17, int16 @18, byte @20.
// UNVERIFIED: B15, B16, B17, W18, B20.
type NpcMoveToTarget struct {
	ID            uint32
	Mode          uint8
	X, Y          uint16
	TargetType    uint8
	TargetID      uint32
	B15, B16, B17 uint8
	W18           int16
	B20           uint8
}

// Marshal returns the packet.
func (m NpcMoveToTarget) Marshal() []byte {
	b := newW(S2CNpcMoveToTarget, 21)
	b.u32(1, m.ID)
	b[5] = m.Mode
	b.u16(6, m.X)
	b.u16(8, m.Y)
	b[10] = m.TargetType
	b.u32(11, m.TargetID)
	b[15], b[16], b[17] = m.B15, m.B16, m.B17
	b.u16(18, uint16(m.W18))
	b[20] = m.B20

	return b
}

// ParseNpcMoveToTarget decodes a 0x68 packet.
func ParseNpcMoveToTarget(b []byte) (NpcMoveToTarget, error) {
	if err := checkS2C(b, S2CNpcMoveToTarget, 21); err != nil {
		return NpcMoveToTarget{}, err
	}

	r := rbuf(b)

	return NpcMoveToTarget{ID: r.u32(1), Mode: b[5], X: r.u16(6), Y: r.u16(8), TargetType: b[10], TargetID: r.u32(11),
		B15: b[15], B16: b[16], B17: b[17], W18: int16(r.u16(18)), B20: b[20]}, nil
}

// NpcStateAtLocation (0x69, 12 bytes), handler 0x458620: id @1, mode @5, u16 x
// @6, u16 y @8, bytes @10, @11 (the handler adds the constants 2, 0, 4 to the
// action record itself). UNVERIFIED: B10, B11.
type NpcStateAtLocation struct {
	ID       uint32
	Mode     uint8
	X, Y     uint16
	B10, B11 uint8
}

// Marshal returns the packet.
func (m NpcStateAtLocation) Marshal() []byte {
	b := newW(S2CNpcStateAtLoc, 12)
	m.fill(b)

	return b
}

func (m NpcStateAtLocation) fill(b wbuf) {
	b.u32(1, m.ID)
	b[5] = m.Mode
	b.u16(6, m.X)
	b.u16(8, m.Y)
	b[10], b[11] = m.B10, m.B11
}

func readStateAtLocation(b []byte) NpcStateAtLocation {
	r := rbuf(b)

	return NpcStateAtLocation{ID: r.u32(1), Mode: b[5], X: r.u16(6), Y: r.u16(8), B10: b[10], B11: b[11]}
}

// ParseNpcStateAtLocation decodes a 0x69 packet.
func ParseNpcStateAtLocation(b []byte) (NpcStateAtLocation, error) {
	if err := checkS2C(b, S2CNpcStateAtLoc, 12); err != nil {
		return NpcStateAtLocation{}, err
	}

	return readStateAtLocation(b), nil
}

// NpcStateOnTarget (0x6a, 12 bytes), handler 0x458680: id @1, mode @5, target
// type @6, target id dword @7, byte @11 (UNVERIFIED: B11).
type NpcStateOnTarget struct {
	ID         uint32
	Mode       uint8
	TargetType uint8
	TargetID   uint32
	B11        uint8
}

// Marshal returns the packet.
func (m NpcStateOnTarget) Marshal() []byte {
	b := newW(S2CNpcStateOnTarget, 12)
	m.fill(b)

	return b
}

func (m NpcStateOnTarget) fill(b wbuf) {
	b.u32(1, m.ID)
	b[5], b[6] = m.Mode, m.TargetType
	b.u32(7, m.TargetID)
	b[11] = m.B11
}

func readStateOnTarget(b []byte) NpcStateOnTarget {
	r := rbuf(b)

	return NpcStateOnTarget{ID: r.u32(1), Mode: b[5], TargetType: b[6], TargetID: r.u32(7), B11: b[11]}
}

// ParseNpcStateOnTarget decodes a 0x6a packet.
func ParseNpcStateOnTarget(b []byte) (NpcStateOnTarget, error) {
	if err := checkS2C(b, S2CNpcStateOnTarget, 12); err != nil {
		return NpcStateOnTarget{}, err
	}

	return readStateOnTarget(b), nil
}

// NpcAttackLocation (0x6b, 16 bytes), handler 0x4586d0: the 0x69 layout plus a
// position correction (u16 x @12, u16 y @14) applied first. The 0x69 fields keep
// their offsets (x @6, y @8, bytes @10, @11).
type NpcAttackLocation struct {
	NpcStateAtLocation
	FixX, FixY uint16
}

// Marshal returns the packet.
func (m NpcAttackLocation) Marshal() []byte {
	b := newW(S2CNpcAttackLoc, 16)
	m.fill(b)
	b.u16(12, m.FixX)
	b.u16(14, m.FixY)

	return b
}

// ParseNpcAttackLocation decodes a 0x6b packet.
func ParseNpcAttackLocation(b []byte) (NpcAttackLocation, error) {
	if err := checkS2C(b, S2CNpcAttackLoc, 16); err != nil {
		return NpcAttackLocation{}, err
	}

	r := rbuf(b)

	return NpcAttackLocation{NpcStateAtLocation: readStateAtLocation(b), FixX: r.u16(12), FixY: r.u16(14)}, nil
}

// NpcAttackTarget (0x6c, 16 bytes), handler 0x458700: position correction (u16
// x @12, u16 y @14) first, then id @1, mode @5, target type @6, target id
// dword @7, byte @11 (the 0x6a layout).
type NpcAttackTarget struct {
	NpcStateOnTarget
	FixX, FixY uint16
}

// Marshal returns the packet.
func (m NpcAttackTarget) Marshal() []byte {
	b := newW(S2CNpcAttackTarget, 16)
	m.fill(b)
	b.u16(12, m.FixX)
	b.u16(14, m.FixY)

	return b
}

// ParseNpcAttackTarget decodes a 0x6c packet.
func ParseNpcAttackTarget(b []byte) (NpcAttackTarget, error) {
	if err := checkS2C(b, S2CNpcAttackTarget, 16); err != nil {
		return NpcAttackTarget{}, err
	}

	r := rbuf(b)

	return NpcAttackTarget{NpcStateOnTarget: readStateOnTarget(b), FixX: r.u16(12), FixY: r.u16(14)}, nil
}

// NpcStop (0x6d, 10 bytes), handler 0x458770: id @1, u16 x @5, u16 y @7, byte
// @9. The handler moves the unit to (x, y), resets stat 0x148 and starts
// animation mode 7. UNVERIFIED: B9.
type NpcStop struct {
	ID   uint32
	X, Y uint16
	B9   uint8
}

// Marshal returns the packet.
func (m NpcStop) Marshal() []byte {
	b := newW(S2CNpcStop, 10)
	b.u32(1, m.ID)
	b.u16(5, m.X)
	b.u16(7, m.Y)
	b[9] = m.B9

	return b
}

// ParseNpcStop decodes a 0x6d packet.
func ParseNpcStop(b []byte) (NpcStop, error) {
	if err := checkS2C(b, S2CNpcStop, 10); err != nil {
		return NpcStop{}, err
	}

	r := rbuf(b)

	return NpcStop{ID: r.u32(1), X: r.u16(5), Y: r.u16(7), B9: b[9]}, nil
}
