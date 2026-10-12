package d2gs

import "testing"

// pinVar marks a variable length entry (-1 in the exe).
const pinVar = -1

// exeServerSizes is the server->client length table read back from Game.exe 1.14b
// (dword table at 0x72e958, cross-checked entry by entry with the second dword of
// the 12-byte client handler table at 0x710f70). Ids not listed have size 0 in the
// exe. Two known differences between the two tables are documented in
// docs/NETWORK_PACKET_AUDIT.md: id 0x17 has a handler (variable) but size 0 in the
// length table, and id 0x80 has size 4 in the handler table but a NULL handler.
var exeServerSizes = map[byte]int{
	0x00: 1, 0x01: 8, 0x02: 1, 0x03: 12, 0x04: 1, 0x05: 1, 0x06: 1, 0x07: 6, 0x08: 6,
	0x09: 11, 0x0a: 6, 0x0b: 6, 0x0c: 9, 0x0d: 13, 0x0e: 12, 0x0f: 16, 0x10: 16, 0x11: 8,
	0x12: 26, 0x13: 14, 0x14: 18, 0x15: 11, 0x16: pinVar, 0x18: 15, 0x19: 2, 0x1a: 2, 0x1b: 3,
	0x1c: 5, 0x1d: 3, 0x1e: 4, 0x1f: 6, 0x20: 10, 0x21: 12, 0x22: 12, 0x23: 13, 0x24: 90,
	0x25: 90, 0x26: pinVar, 0x27: 40, 0x28: 103, 0x29: 97, 0x2a: 15, 0x2c: 8, 0x3e: pinVar, 0x3f: 8,
	0x40: 13, 0x42: 6, 0x45: 13, 0x47: 11, 0x48: 11, 0x4c: 16, 0x4d: 17, 0x4e: 7, 0x4f: 1,
	0x50: 15, 0x51: 14, 0x52: 42, 0x53: 10, 0x54: 3, 0x57: 14, 0x58: 7, 0x59: 26, 0x5a: 40,
	0x5b: pinVar, 0x5c: 5, 0x5d: 6, 0x5e: 38, 0x5f: 5, 0x60: 7, 0x61: 2, 0x62: 7, 0x63: 21,
	0x65: 7, 0x66: 7, 0x67: 16, 0x68: 21, 0x69: 12, 0x6a: 12, 0x6b: 16, 0x6c: 16, 0x6d: 10,
	0x6e: 1, 0x6f: 1, 0x70: 1, 0x71: 1, 0x72: 1, 0x73: 32, 0x74: 10, 0x75: 13, 0x76: 6,
	0x77: 2, 0x78: 21, 0x79: 6, 0x7a: 13, 0x7b: 8, 0x7c: 6, 0x7d: 18, 0x7e: 5, 0x7f: 10,
	0x81: 20, 0x82: 29, 0x89: 2, 0x8a: 6, 0x8b: 6, 0x8c: 11, 0x8d: 7, 0x8e: 10, 0x8f: 33,
	0x90: 13, 0x91: 26, 0x92: 6, 0x93: 8, 0x94: pinVar, 0x95: 13, 0x96: 9, 0x97: 1, 0x98: 7,
	0x99: 16, 0x9a: 17, 0x9b: 7, 0x9c: pinVar, 0x9d: pinVar, 0x9e: 7, 0x9f: 8, 0xa0: 10, 0xa1: 7,
	0xa2: 8, 0xa3: 24, 0xa4: 3, 0xa5: 8, 0xa6: pinVar, 0xa7: 7, 0xa8: pinVar, 0xa9: 7,
	0xaa: pinVar, 0xab: 7, 0xac: pinVar, 0xae: pinVar, 0xaf: pinVar, 0xb0: 1, 0xb2: 53,
	0xb3: pinVar, 0xb4: 5,
}

func TestServerSizesPinnedToExe(t *testing.T) {
	for id := 0; id <= 0xff; id++ {
		want := exeServerSizes[byte(id)]
		n, kind := ExpectedSize(byte(id), ServerToClient)
		switch {
		case want == 0:
			if kind != SizeInvalid {
				t.Errorf("S2C 0x%02x: exe size 0, port kind %d", id, kind)
			}
		case want == pinVar:
			if kind != SizeVariable {
				t.Errorf("S2C 0x%02x: exe variable, port kind %d", id, kind)
			}
		default:
			if kind != SizeFixed || n != want {
				t.Errorf("S2C 0x%02x: exe %d, port %d (kind %d)", id, want, n, kind)
			}
		}
	}
}

// exeClientNull lists the C2S ids with a NULL handler in the dispatch table at
// 0x6e2010 (8-byte entries, ids 0..0x66); id 0 is NULL too. Every other id in
// 1..0x66 is handled.
var exeClientNull = []byte{0x2b, 0x4a, 0x4e, 0x55, 0x56, 0x57, 0x5a, 0x5b, 0x5c, 0x64, 0x65}

func TestClientHandledPinnedToExe(t *testing.T) {
	null := map[byte]bool{0: true}
	for _, id := range exeClientNull {
		null[id] = true
	}
	for id := 0; id <= 0xff; id++ {
		_, kind := ExpectedSize(byte(id), ClientToServer)
		wantInvalid := id > 0x66 || null[byte(id)]
		if wantInvalid != (kind == SizeInvalid) {
			t.Errorf("C2S 0x%02x: invalid=%v want %v", id, kind == SizeInvalid, wantInvalid)
		}
	}
}

// TestPinnedLayouts checks field offsets read from the client handlers (0x458020
// game flags, 0x4596a0 skill at location) and the server sender (0x53b1e0).
func TestPinnedLayouts(t *testing.T) {
	g := GameFlags{Difficulty: 2, Hardcore: true, Expansion: true, Ladder: true}.Marshal()
	if g[1] != 2 || g[4] != 1 || g[5] != 0 || g[6] != 1 || g[7] != 1 {
		t.Errorf("GameFlags layout %v", g)
	}
	k := UnitSkillOnLocation{UnitID: 0x01020304, Skill: 0x0a0b, Level: 7, X: 0x1112, Y: 0x2122}.Marshal()
	want := []byte{0x4d, 0, 4, 3, 2, 1, 0x0b, 0x0a, 0, 0, 7, 0x12, 0x11, 0x22, 0x21, 0, 0}
	if string(k) != string(want) {
		t.Errorf("UnitSkillOnLocation %v want %v", k, want)
	}
}
