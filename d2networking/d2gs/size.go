package d2gs

import "errors"

// SizeKind classifies what is known about a packet id's length.
type SizeKind int

// Size kinds.
const (
	// SizeInvalid: the id is rejected by the real game (zero table entry,
	// NULL handler, or out of range).
	SizeInvalid SizeKind = iota
	// SizeFixed: n is the exact packet length.
	SizeFixed
	// SizeVariable: length depends on packet contents (server->client only).
	SizeVariable
	// SizeUnknown: the id is valid but its size has not been verified yet.
	SizeUnknown
)

// ExpectedSize reports the length rule for id in direction dir.
func ExpectedSize(id byte, dir Direction) (n int, kind SizeKind) {
	if dir == ServerToClient {
		if id > MaxServerID {
			return 0, SizeInvalid
		}
		s, ok := serverSizes[id]
		switch {
		case !ok:
			return 0, SizeInvalid
		case s == variable:
			return 0, SizeVariable
		}
		return s, SizeFixed
	}
	if id == 0 || id > MaxClientID || !clientHandled[id] {
		return 0, SizeInvalid
	}
	if s, ok := clientSizes[id]; ok {
		return s, SizeFixed
	}
	return 0, SizeUnknown
}

// Errors returned by the decoder.
var (
	ErrInvalidID   = errors.New("d2gs: invalid packet id")
	ErrUnknownSize = errors.New("d2gs: packet size not verified, cannot frame")
	ErrBadLength   = errors.New("d2gs: invalid variable packet length")
	ErrShort       = errors.New("d2gs: packet shorter than its layout")
	ErrWrongSize   = errors.New("d2gs: packet has wrong size for its id")
	ErrTooLarge    = errors.New("d2gs: packet exceeds 0x204 bytes")
)

// strlen returns the NUL-terminated length starting at b[off], or -1 if there
// is no NUL within b.
func strlen(b []byte, off int) int {
	for i := off; i < len(b); i++ {
		if b[i] == 0 {
			return i - off
		}
	}
	return -1
}

// serverLength mirrors SCMD_GetServerPacketLength (0x00529300), verified
// against its decompiled switch. buf must hold at least one byte.
// It returns the total packet length (including the id byte). need > 0 means
// that many more bytes (at least) must be buffered before the length is known.
func serverLength(buf []byte) (n int, need int, err error) {
	id := buf[0]
	size, kind := ExpectedSize(id, ServerToClient)
	switch kind {
	case SizeInvalid:
		return 0, 0, ErrInvalidID
	case SizeFixed:
		return size, 0, nil
	}
	avail := len(buf)
	// wait returns the shortfall when fewer than min bytes are buffered.
	wait := func(min int) int {
		if avail < min {
			return min - avail
		}
		return 0
	}
	switch id {
	case 0x16: // u16@1; original requires more than 0xc bytes buffered
		if w := wait(0xd); w > 0 {
			return 0, w, nil
		}
		n = int(buf[1]) | int(buf[2])<<8
	case 0x26: // two NUL-terminated strings from offset 10; total = len1+len2+12
		if w := wait(10); w > 0 {
			return 0, w, nil
		}
		l1 := strlen(buf, 10)
		if l1 < 0 {
			if avail > MaxPacketSize {
				return 0, 0, ErrBadLength
			}
			return 0, 1, nil
		}
		l2 := strlen(buf, 10+l1+1)
		if l2 < 0 {
			if avail > MaxPacketSize {
				return 0, 0, ErrBadLength
			}
			return 0, 1, nil
		}
		n = l1 + l2 + 12
	case 0x3e: // byte@1
		if w := wait(2); w > 0 {
			return 0, w, nil
		}
		n = int(buf[1])
	case 0x5b: // u16@1; original requires more than 0x21 bytes buffered
		if w := wait(0x22); w > 0 {
			return 0, w, nil
		}
		n = int(buf[1]) | int(buf[2])<<8
	case 0x94: // (byte@1 + 2) * 3; original requires at least 9 bytes buffered
		if w := wait(9); w > 0 {
			return 0, w, nil
		}
		n = (int(buf[1]) + 2) * 3
	case 0x9c, 0x9d: // byte@2
		if w := wait(3); w > 0 {
			return 0, w, nil
		}
		n = int(buf[2])
	case 0xa6: // u16@2
		if w := wait(4); w > 0 {
			return 0, w, nil
		}
		n = int(buf[2]) | int(buf[3])<<8
	case 0xa8, 0xaa: // byte@6
		if w := wait(7); w > 0 {
			return 0, w, nil
		}
		n = int(buf[6])
	case 0xac: // byte@0xc
		if w := wait(0xd); w > 0 {
			return 0, w, nil
		}
		n = int(buf[0xc])
	case 0xae: // u16@1 + 3, payload clamped: values above 0x1fd give 0 (-> 3)
		if w := wait(3); w > 0 {
			return 0, w, nil
		}
		p := int(buf[1]) | int(buf[2])<<8
		if p > 0x1fd {
			p = 0
		}
		n = p + 3
	case 0xaf: // byte@1 == 0 -> 2, else byte@1 + 1
		if w := wait(2); w > 0 {
			return 0, w, nil
		}
		if buf[1] != 0 {
			n = int(buf[1]) + 1
		} else {
			n = 2
		}
	case 0xb3: // byte@1 + 7
		if w := wait(8); w > 0 {
			return 0, w, nil
		}
		n = int(buf[1]) + 7
	default:
		return 0, 0, ErrInvalidID
	}
	if n < 1 || n > MaxPacketSize {
		return 0, 0, ErrBadLength
	}
	// A packet cannot be shorter than the header bytes its own rule read.
	if n < variableHeader[id] {
		return 0, 0, ErrBadLength
	}
	return n, 0, nil
}

// variableHeader is the number of leading bytes each variable-length rule
// reads; a packet claiming to be shorter than that is malformed.
var variableHeader = map[byte]int{
	0x16: 3, 0x26: 12, 0x3e: 2, 0x5b: 3, 0x94: 2, 0x9c: 3, 0x9d: 3, 0xa6: 4,
	0xa8: 7, 0xaa: 7, 0xac: 13, 0xae: 3, 0xaf: 2, 0xb3: 8,
}
