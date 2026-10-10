package d2gs

// Roster / party / event-message packets, layouts read from the client
// handlers in Game.exe 1.14b (read-only Ghidra, d2-re-notes/verify-roster-packets.md).
// "Verified" means the offsets below were read from the handler or the roster
// helpers it calls. Field names come from the roster getters
// (ROSTER_GetLevel 0x475880 reads entry+0x20, ROSTER_GetPartyIdForUnit 0x4758f0
// reads entry+0x22, ROSTER_GetKillCount 0x475870 reads entry+0x18); the
// meaning of individual flag bits is UNVERIFIED where said so.

// S2C ids of the roster family.
const (
	S2CEventMessage byte = 0x5a // size 40, handler 0x4597b0 -> UI_ShowEventMessage
	S2CRosterEntry  byte = 0x5b // variable, handler 0x459c20 -> 0x4763d0
	S2CRosterKills  byte = 0x65 // size 7, handler 0x459df0
	S2CRosterUpdate byte = 0x75 // size 13, handler 0x459f30 -> 0x476540
	S2CRosterRelate byte = 0x8c // size 11, handler 0x45a1c0 -> 0x476060
	S2CRosterParty  byte = 0x8d // size 7, handler 0x45a1e0 -> 0x476020
	S2CRosterLink   byte = 0x8e // size 10, handler 0x45a1f0 -> 0x4760b0 / 0x476170

	rosterEntryFixed = 0x22 // bytes of 0x5b before the account name
	rosterStrMax     = 16
)

// NoParty is the party id the exe returns for an unknown unit
// (ROSTER_GetPartyIdForUnit returns 0xffff); also used here for "no party".
const NoParty uint16 = 0xffff

// RosterEntry is S2C 0x5b "player in game", variable length. Verified layout:
// u16 total length @1 (the framing rule), u32 unit id @3, u8 class @7, name[16]
// @8, u16 level @0x18, u16 party id @0x1a, u16 @0x1c (not stored by the
// roster), u16 @0x1e (stored at roster entry+0x30, which ROSTER_GetPartyFlags
// returns; bit meanings UNVERIFIED), u16 @0x20 (entry+0x44, meaning
// UNVERIFIED), then two NUL terminated strings from 0x22: the account name and
// a third string stored at entry+0x4a (meaning UNVERIFIED). The roster
// displays "name*account" when the account string is non-empty.
type RosterEntry struct {
	UnitID         uint32
	Class          uint8
	Name           string
	Level          uint16
	PartyID        uint16
	Unk1C          uint16
	Flags          uint16
	Unk20          uint16
	Account, Extra string
}

func clipStr(s string) string {
	for i := 0; i < len(s) && i < rosterStrMax; i++ {
		if s[i] == 0 {
			return s[:i]
		}
	}

	if len(s) > rosterStrMax {
		return s[:rosterStrMax]
	}

	return s
}

// cstr returns the NUL terminated string at the start of b and the bytes it
// consumed including the NUL, or -1 when b holds no NUL.
func cstr(b []byte) (string, int) {
	for i, c := range b {
		if c == 0 {
			return string(b[:i]), i + 1
		}
	}

	return "", -1
}

// Marshal returns the packet (both strings are always written).
func (m RosterEntry) Marshal() []byte {
	acct, extra := clipStr(m.Account), clipStr(m.Extra)
	b := newW(S2CRosterEntry, rosterEntryFixed+len(acct)+1+len(extra)+1)
	b.u16(1, uint16(len(b)))
	b.u32(3, m.UnitID)
	b[7] = m.Class
	putName(b[8:24], m.Name)
	b.u16(0x18, m.Level)
	b.u16(0x1a, m.PartyID)
	b.u16(0x1c, m.Unk1C)
	b.u16(0x1e, m.Flags)
	b.u16(0x20, m.Unk20)
	copy(b[rosterEntryFixed:], acct)
	copy(b[rosterEntryFixed+len(acct)+1:], extra)

	return b
}

// ParseRosterEntry decodes a 0x5b packet. The length field must equal len(b).
// The account string must be terminated inside the packet; a missing third
// string reads as empty.
func ParseRosterEntry(b []byte) (RosterEntry, error) {
	if len(b) < rosterEntryFixed+1 || b[0] != S2CRosterEntry || len(b) > MaxPacketSize {
		return RosterEntry{}, ErrWrongSize
	}

	r := rbuf(b)
	if int(r.u16(1)) != len(b) {
		return RosterEntry{}, ErrBadLength
	}

	// Our encoder keeps names NUL terminated within 16 bytes (15 characters).
	// UNVERIFIED: the exe copies the name up to its NUL without a bound.
	name := getName(b[8:24])
	if len(name) > 15 {
		name = name[:15]
	}

	m := RosterEntry{UnitID: r.u32(3), Class: b[7], Name: name, Level: r.u16(0x18), PartyID: r.u16(0x1a),
		Unk1C: r.u16(0x1c), Flags: r.u16(0x1e), Unk20: r.u16(0x20)}

	acct, n := cstr(b[rosterEntryFixed:])
	if n < 0 {
		return RosterEntry{}, ErrShort
	}

	m.Account = clipStr(acct)
	if rest := b[rosterEntryFixed+n:]; len(rest) > 0 {
		extra, _ := cstr(rest)
		m.Extra = clipStr(extra)
	}

	return m, nil
}

// RosterKills is S2C 0x65 (7 bytes): u32 unit id @1, signed 16 bit value @5
// (sign extended like the exe) stored in roster entry+0x18, which
// ROSTER_GetKillCount returns; the party list is regrouped and rebuilt
// afterwards. Verified offsets; "kills" is the getter's name.
type RosterKills struct {
	UnitID uint32
	Kills  int16
}

// Marshal returns the packet.
func (m RosterKills) Marshal() []byte {
	b := newW(S2CRosterKills, 7)
	b.u32(1, m.UnitID)
	b.u16(5, uint16(m.Kills))

	return b
}

// ParseRosterKills decodes a 0x65 packet.
func ParseRosterKills(b []byte) (RosterKills, error) {
	if len(b) != 7 || b[0] != S2CRosterKills {
		return RosterKills{}, ErrWrongSize
	}

	r := rbuf(b)

	return RosterKills{UnitID: r.u32(1), Kills: int16(r.u16(5))}, nil
}

// RosterUpdate is S2C 0x75 (13 bytes): u32 unit id @1, u16 party id @5
// (entry+0x22), u16 level @7 (entry+0x20), u16 @9 (unused by the handler),
// u16 @0xb (entry+0x30, the same field as 0x5b @0x1e). Verified offsets. This
// is the packet that changes an existing roster entry's level and party.
type RosterUpdate struct {
	UnitID         uint32
	PartyID, Level uint16
	Unk9, Flags    uint16
}

// Marshal returns the packet.
func (m RosterUpdate) Marshal() []byte {
	b := newW(S2CRosterUpdate, 13)
	b.u32(1, m.UnitID)
	b.u16(5, m.PartyID)
	b.u16(7, m.Level)
	b.u16(9, m.Unk9)
	b.u16(11, m.Flags)

	return b
}

// ParseRosterUpdate decodes a 0x75 packet.
func ParseRosterUpdate(b []byte) (RosterUpdate, error) {
	if len(b) != 13 || b[0] != S2CRosterUpdate {
		return RosterUpdate{}, ErrWrongSize
	}

	r := rbuf(b)

	return RosterUpdate{UnitID: r.u32(1), PartyID: r.u16(5), Level: r.u16(7), Unk9: r.u16(9), Flags: r.u16(11)}, nil
}

// RosterRelation is S2C 0x8c (11 bytes): u32 unit A @1, u32 unit B @5, u16
// relation flags @9. The handler records the flags for the pair (0x4d90d0 and
// 0x4d9270) and, when B is the local player, calls 0x474f60 for A. Verified
// offsets; the meaning of the flag bits is UNVERIFIED.
type RosterRelation struct {
	A, B  uint32
	Flags uint16
}

// Marshal returns the packet.
func (m RosterRelation) Marshal() []byte {
	b := newW(S2CRosterRelate, 11)
	b.u32(1, m.A)
	b.u32(5, m.B)
	b.u16(9, m.Flags)

	return b
}

// ParseRosterRelation decodes a 0x8c packet.
func ParseRosterRelation(b []byte) (RosterRelation, error) {
	if len(b) != 11 || b[0] != S2CRosterRelate {
		return RosterRelation{}, ErrWrongSize
	}

	r := rbuf(b)

	return RosterRelation{A: r.u32(1), B: r.u32(5), Flags: r.u16(9)}, nil
}

// RosterParty is S2C 0x8d (7 bytes): u32 unit id @1, u16 party id @5, written
// to roster entry+0x22 (0xffff means none). Verified.
type RosterParty struct {
	UnitID  uint32
	PartyID uint16
}

// Marshal returns the packet.
func (m RosterParty) Marshal() []byte {
	b := newW(S2CRosterParty, 7)
	b.u32(1, m.UnitID)
	b.u16(5, m.PartyID)

	return b
}

// ParseRosterParty decodes a 0x8d packet.
func ParseRosterParty(b []byte) (RosterParty, error) {
	if len(b) != 7 || b[0] != S2CRosterParty {
		return RosterParty{}, ErrWrongSize
	}

	r := rbuf(b)

	return RosterParty{UnitID: r.u32(1), PartyID: r.u16(5)}, nil
}

// RosterLink is S2C 0x8e (10 bytes): u8 add flag @1 (non-zero adds, zero
// removes), u32 unit A @2, u32 unit B @6. The handler keeps a per-entry list
// of unit ids on A (entry+0x38) and adds or removes B. Verified offsets; what
// the list means is UNVERIFIED.
type RosterLink struct {
	Add  bool
	A, B uint32
}

// Marshal returns the packet.
func (m RosterLink) Marshal() []byte {
	b := newW(S2CRosterLink, 10)
	b[1] = boolByte(m.Add)
	b.u32(2, m.A)
	b.u32(6, m.B)

	return b
}

// ParseRosterLink decodes a 0x8e packet.
func ParseRosterLink(b []byte) (RosterLink, error) {
	if len(b) != 10 || b[0] != S2CRosterLink {
		return RosterLink{}, ErrWrongSize
	}

	r := rbuf(b)

	return RosterLink{Add: b[1] != 0, A: r.u32(2), B: r.u32(6)}, nil
}

// EventMessage is S2C 0x5a (40 bytes), handler 0x4597b0 copies the packet and
// calls UI_ShowEventMessage. Verified offsets: u8 type @1 (switch 0..0x12),
// u32 @3 (string id or object id, depending on type), u8 @7 (sub kind; type 6:
// 0 player, 1 monster, 2 object), subject name[16] @8 (byte 0x17 forced NUL),
// 16 bytes @0x18: another name or a u16 super-unique id (type 6, kind 1).
// Byte @2 is not read by the paths examined (UNVERIFIED, perhaps a colour).
// Which type is which message (2 joined, 3 left, 4/5 dropped, 7 party text...)
// is UNVERIFIED beyond the string-table use; types 2 and 3 compare the subject
// with the local player's name.
type EventMessage struct {
	Type, B2 uint8
	Param    uint32
	Kind     uint8
	Name     string
	Name2    [16]byte
}

// Marshal returns the packet.
func (m EventMessage) Marshal() []byte {
	b := newW(S2CEventMessage, 40)
	b[1], b[2] = m.Type, m.B2
	b.u32(3, m.Param)
	b[7] = m.Kind
	putName(b[8:24], m.Name)
	copy(b[24:], m.Name2[:])

	return b
}

// ParseEventMessage decodes a 0x5a packet.
func ParseEventMessage(b []byte) (EventMessage, error) {
	if len(b) != 40 || b[0] != S2CEventMessage {
		return EventMessage{}, ErrWrongSize
	}

	m := EventMessage{Type: b[1], B2: b[2], Param: rbuf(b).u32(3), Kind: b[7], Name: getName(b[8:24])}
	if len(m.Name) > 15 { // the exe forces byte 0x17 to NUL
		m.Name = m.Name[:15]
	}

	copy(m.Name2[:], b[24:])

	return m, nil
}
