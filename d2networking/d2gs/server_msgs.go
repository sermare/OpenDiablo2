package d2gs

import (
	"encoding/binary"
	"fmt"
)

// Server -> client packets used by the multiplayer transport. The sizes are
// the verified ones of the table at 0x72e958; where a field layout is not
// verified against a handler body it says so ("layout unverified"): we only
// need our own peers to agree, and the public D2GS documentation was used as
// the guess.

// S2C ids used here that are not in ids.go.
const (
	S2CPlayerMove     byte = 0x0f // size 16
	S2CChat           byte = 0x26 // variable: strings from offset 10
	S2CUnitSkillOnLoc byte = 0x4d // size 17
	S2CPlayerInGame   byte = 0x59 // size 26
	S2CPlayerLeave    byte = 0x5c // size 5
	// CtlTunnel is the client -> server tunnel (0x6c SaveFileUpload), same chunk layout as S2CMetaAE.
	CtlTunnel byte = 0x6c
)

const (
	maxTunnelChunk        = 0x1f0
	tunnelHeader          = 2 // [type][flags] inside the payload
	tunnelFlagLast   byte = 1
	joinGameSize          = 21
	nameLen               = 16
	chatTypeNormal   byte = 1
	moveTypeWalk     byte = 1
	moveTypeRun      byte = 2
	unitTypePlayer   byte = 0
	playerMoveSize        = 16
	unitSkillOnLocSz      = 17
	chatStringsOff        = 10
	maxChatText           = 0xff
	gameFlagsSize         = 8
	loadActSize           = 12
	playerLeaveSize       = 5
	tunnelPayloadMax      = 0x1fd
	// maxTunnelMessage bounds the reassembly buffer so a peer that never sends the
	// "last" flag cannot grow memory without limit (our own limit, not the exe's).
	maxTunnelMessage = 32 << 20
)

func putName(dst []byte, name string) {
	copy(dst[:nameLen], name)

	if len(name) >= nameLen { // keep it NUL terminated
		dst[nameLen-1] = 0
	}
}

func getName(src []byte) string {
	n := 0
	for n < nameLen && src[n] != 0 {
		n++
	}

	return string(src[:n])
}

func boolByte(v bool) byte {
	if v {
		return 1
	}

	return 0
}

// GameFlags (0x01, 8 bytes): game difficulty / hardcore / expansion / ladder.
// Layout unverified: [1]=difficulty, [2..3]=0, [4]=hardcore, [5]=expansion, [6]=ladder.
type GameFlags struct {
	Difficulty                  uint8
	Hardcore, Expansion, Ladder bool
}

// Marshal returns the packet.
func (m GameFlags) Marshal() []byte {
	b := newW(S2CGameFlags, gameFlagsSize)
	b[1] = m.Difficulty
	b[4], b[5], b[6] = boolByte(m.Hardcore), boolByte(m.Expansion), boolByte(m.Ladder)

	return b
}

// ParseGameFlags decodes a 0x01 packet.
func ParseGameFlags(b []byte) (GameFlags, error) {
	if len(b) != gameFlagsSize || b[0] != S2CGameFlags {
		return GameFlags{}, ErrWrongSize
	}

	return GameFlags{Difficulty: b[1], Hardcore: b[4] != 0, Expansion: b[5] != 0, Ladder: b[6] != 0}, nil
}

// LoadAct (0x03, 12 bytes): [1]=act, [2..5]=game seed, [6..7]=start level,
// [8..11]=aux dword. Verified (session-core.md: CLIENT_HandleLoadActPacket).
// The engine puts its region type into Aux (the real game sends the second seed).
type LoadAct struct {
	Act        uint8
	Seed       uint32
	StartLevel uint16
	Aux        uint32
}

// Marshal returns the packet.
func (m LoadAct) Marshal() []byte {
	b := newW(S2CLoadAct, loadActSize)
	b[1] = m.Act
	b.u32(2, m.Seed)
	b.u16(6, m.StartLevel)
	b.u32(8, m.Aux)

	return b
}

// ParseLoadAct decodes a 0x03 packet.
func ParseLoadAct(b []byte) (LoadAct, error) {
	if len(b) != loadActSize || b[0] != S2CLoadAct {
		return LoadAct{}, ErrWrongSize
	}

	r := rbuf(b)

	return LoadAct{Act: b[1], Seed: r.u32(2), StartLevel: r.u16(6), Aux: r.u32(8)}, nil
}

// PlayerLeave (0x5c, 5 bytes): [1..4]=unit id of the player who left.
type PlayerLeave struct{ UnitID uint32 }

// Marshal returns the packet.
func (m PlayerLeave) Marshal() []byte {
	b := newW(S2CPlayerLeave, playerLeaveSize)
	b.u32(1, m.UnitID)

	return b
}

// ParsePlayerLeave decodes a 0x5c packet.
func ParsePlayerLeave(b []byte) (PlayerLeave, error) {
	if len(b) != playerLeaveSize || b[0] != S2CPlayerLeave {
		return PlayerLeave{}, ErrWrongSize
	}

	return PlayerLeave{UnitID: rbuf(b).u32(1)}, nil
}

// PlayerMove (0x0f, 16 bytes). Verified from the deferred handler 0x4584b0:
// [6]=move mode byte (passed to 0x47ca50), [7..8]=target x, [9..10]=target y,
// [11]=a byte copied into the move record (always 0 here, meaning UNVERIFIED),
// [12..13]=current x, [14..15]=current y (passed to 0x47c320). UNVERIFIED: [1]
// unit type and [2..5] unit id (consumed by the unit lookup in the
// dispatcher through registers) and that mode 1/2 are walk/run. Coordinates
// are sub-tiles.
type PlayerMove struct {
	UnitID           uint32
	Run              bool
	TargetX, TargetY uint16
	CurX, CurY       uint16
}

// Marshal returns the packet.
func (m PlayerMove) Marshal() []byte {
	b := newW(S2CPlayerMove, playerMoveSize)
	b[1] = unitTypePlayer
	b.u32(2, m.UnitID)
	b[6] = moveTypeWalk

	if m.Run {
		b[6] = moveTypeRun
	}

	b.u16(7, m.TargetX)
	b.u16(9, m.TargetY)
	b.u16(12, m.CurX)
	b.u16(14, m.CurY)

	return b
}

// ParsePlayerMove decodes a 0x0f packet.
func ParsePlayerMove(b []byte) (PlayerMove, error) {
	if len(b) != playerMoveSize || b[0] != S2CPlayerMove {
		return PlayerMove{}, ErrWrongSize
	}

	r := rbuf(b)

	return PlayerMove{UnitID: r.u32(2), Run: b[6] == moveTypeRun,
		TargetX: r.u16(7), TargetY: r.u16(9), CurX: r.u16(12), CurY: r.u16(14)}, nil
}

// UnitSkillOnLocation (0x4d, 17 bytes): a unit casts a skill at a location.
// Layout unverified: [1]=unit type, [2..5]=unit id, [6..7]=skill, [8]=level,
// [9..10]=x, [11..12]=y, [13..16]=0.
type UnitSkillOnLocation struct {
	UnitID uint32
	Skill  uint16
	Level  uint8
	X, Y   uint16
}

// Marshal returns the packet.
func (m UnitSkillOnLocation) Marshal() []byte {
	b := newW(S2CUnitSkillOnLoc, unitSkillOnLocSz)
	b[1] = unitTypePlayer
	b.u32(2, m.UnitID)
	b.u16(6, m.Skill)
	b[8] = m.Level
	b.u16(9, m.X)
	b.u16(11, m.Y)

	return b
}

// ParseUnitSkillOnLocation decodes a 0x4d packet.
func ParseUnitSkillOnLocation(b []byte) (UnitSkillOnLocation, error) {
	if len(b) != unitSkillOnLocSz || b[0] != S2CUnitSkillOnLoc {
		return UnitSkillOnLocation{}, ErrWrongSize
	}

	r := rbuf(b)

	return UnitSkillOnLocation{UnitID: r.u32(2), Skill: r.u16(6), Level: b[8], X: r.u16(9), Y: r.u16(11)}, nil
}

// ChatMessage (0x26, variable). Verified: total length = len(name)+len(text)+12
// with the two NUL-terminated strings starting at offset 10. Unverified: the
// header fields: [1]=chat type, [2]=language, [3..6]=speaker unit id, [7]=colour.
type ChatMessage struct {
	Type   uint8
	UnitID uint32
	Name   string
	Text   string
}

// Marshal returns the packet (text and name are cut to fit 0x204 bytes).
func (m ChatMessage) Marshal() []byte {
	name, text := m.Name, m.Text
	if len(name) > nameLen {
		name = name[:nameLen]
	}

	if len(text) > maxChatText {
		text = text[:maxChatText]
	}

	b := make([]byte, chatStringsOff+len(name)+1+len(text)+1)
	b[0], b[1] = S2CChat, m.Type
	binary.LittleEndian.PutUint32(b[3:], m.UnitID)
	copy(b[chatStringsOff:], name)
	copy(b[chatStringsOff+len(name)+1:], text)

	return b
}

// ParseChatMessage decodes a 0x26 packet.
func ParseChatMessage(b []byte) (ChatMessage, error) {
	if len(b) < chatStringsOff+2 || b[0] != S2CChat {
		return ChatMessage{}, ErrShort
	}

	l1 := strlen(b, chatStringsOff)
	if l1 < 0 {
		return ChatMessage{}, ErrBadLength
	}

	l2 := strlen(b, chatStringsOff+l1+1)
	if l2 < 0 || chatStringsOff+l1+l2+2 != len(b) {
		return ChatMessage{}, ErrBadLength
	}

	return ChatMessage{
		Type:   b[1],
		UnitID: rbuf(b).u32(3),
		Name:   string(b[chatStringsOff : chatStringsOff+l1]),
		Text:   string(b[chatStringsOff+l1+1 : chatStringsOff+l1+1+l2]),
	}, nil
}

// ClientChat builds the client chat packet 0x15: [1]=chat type, [2]=0, then the
// NUL-terminated message at offset 3 and a NUL-terminated recipient name
// (empty = everybody). The handler reads the message with a 0x100 limit
// (verified: 0x5484a0); the recipient field is unverified.
func ClientChat(text string) []byte {
	if len(text) > maxChatText {
		text = text[:maxChatText]
	}

	b := make([]byte, 3+len(text)+1+1)
	b[0], b[1] = C2SChat, chatTypeNormal
	copy(b[3:], text)

	return b
}

// ParseClientChat extracts the message of a 0x15 packet.
func ParseClientChat(b []byte) (string, error) {
	if len(b) < 5 || b[0] != C2SChat {
		return "", ErrShort
	}

	n := strlen(b, 3)
	// handler 0x5484a0 (verified): text shorter than 0x100 and the packet must be
	// longer than strlen+4, i.e. the NUL-terminated recipient field must exist.
	if n < 0 || n > maxChatText || len(b) <= n+4 {
		return "", ErrBadLength
	}

	return string(b[3 : 3+n]), nil
}

// JoinGame is the engine's 0x68 packet. The real packet's layout is not
// decoded in the notes, so this is OUR layout (unverified): [1..16]=name,
// [17]=class, [18]=level, [19]=difficulty, [20]=0. The character itself
// follows in the tunnel (the real game uploads the .d2s with 0x6c).
type JoinGame struct {
	Name       string
	Class      uint8
	Level      uint8
	Difficulty uint8
}

// Marshal returns the packet.
func (m JoinGame) Marshal() []byte {
	b := newW(CtlJoinGame, joinGameSize)
	putName(b[1:], m.Name)
	b[17], b[18], b[19] = m.Class, m.Level, m.Difficulty

	return b
}

// ParseJoinGame decodes a 0x68 packet.
func ParseJoinGame(b []byte) (JoinGame, error) {
	if len(b) != joinGameSize || b[0] != CtlJoinGame {
		return JoinGame{}, ErrWrongSize
	}

	return JoinGame{Name: getName(b[1:]), Class: b[17], Level: b[18], Difficulty: b[19]}, nil
}

// LeaveGame returns the 0x69 packet (1 byte, size unverified).
func LeaveGame() []byte { return []byte{CtlLeaveGame} }

// GameExit returns the S2C 0x06 packet (verified size 1).
func GameExit() []byte { return []byte{S2CGameExit} }

// Tunnel carries engine messages that have no real D2GS counterpart (the
// hero's whole state, saves, waypoints...). Server -> client it rides in the
// variable meta packet 0xAE ([1..2]=payload length, verified rule total =
// payload+3, payload <= 0x1fd), client -> server in 0x6c (SaveFileUpload, same
// chunk layout; the real limit is 0x2000). Payload = [type][flags] + data,
// flags bit 0 marks the last chunk of the message.
func Tunnel(id byte, typ byte, data []byte) [][]byte {
	var out [][]byte

	for first := true; first || len(data) > 0; first = false {
		n := len(data)
		if n > maxTunnelChunk {
			n = maxTunnelChunk
		}

		flags := byte(0)
		if n == len(data) {
			flags = tunnelFlagLast
		}

		b := make([]byte, 3+tunnelHeader+n)
		b[0] = id
		binary.LittleEndian.PutUint16(b[1:], uint16(tunnelHeader+n))
		b[3], b[4] = typ, flags
		copy(b[5:], data[:n])
		out = append(out, b)
		data = data[n:]
	}

	return out
}

// TunnelAssembler reassembles tunnel messages (in order, per connection).
type TunnelAssembler struct {
	typ byte
	buf []byte
}

// Add feeds one 0xAE / 0x6c packet; done is true when a whole message is ready.
func (a *TunnelAssembler) Add(pkt []byte) (typ byte, data []byte, done bool, err error) {
	if len(pkt) < 3+tunnelHeader {
		return 0, nil, false, ErrShort
	}

	n := int(binary.LittleEndian.Uint16(pkt[1:]))
	if n < tunnelHeader || n > tunnelPayloadMax || 3+n != len(pkt) {
		return 0, nil, false, ErrBadLength
	}

	if len(a.buf) == 0 {
		a.typ = pkt[3]
	} else if a.typ != pkt[3] {
		a.buf = nil

		return 0, nil, false, fmt.Errorf("%w: tunnel chunk type %d inside message %d", ErrBadLength, pkt[3], a.typ)
	}

	if len(a.buf)+len(pkt)-5 > maxTunnelMessage {
		a.buf = nil

		return 0, nil, false, fmt.Errorf("%w: tunnel message exceeds %d bytes", ErrTooLarge, maxTunnelMessage)
	}

	a.buf = append(a.buf, pkt[5:]...)

	if pkt[4]&tunnelFlagLast == 0 {
		return 0, nil, false, nil
	}

	typ, data = a.typ, a.buf
	a.buf = nil

	return typ, data, true, nil
}

// Pending returns the number of payload bytes buffered for the message that
// is still being assembled (callers use it to bound uploads).
func (a *TunnelAssembler) Pending() int { return len(a.buf) }
