package d2s

import (
	"encoding/binary"
	"testing"
)

// TestMercDeadBitAndExists pins MERC_LoadFromD2sHeader (0x568730, VERIFIED):
// dead is bit 0 of the byte at 0xB1 only, and a merc exists when the id, the
// experience or the name id is non-zero (the type alone does not count).
func TestMercDeadBitAndExists(t *testing.T) {
	le := binary.LittleEndian

	tests := []struct {
		name   string
		edit   func(d []byte)
		dead   bool
		exists bool
	}{
		{"empty", func(d []byte) {}, false, false},
		{"type only", func(d []byte) { le.PutUint16(d[mercTypeOffset:], 11) }, false, false},
		{"exp only", func(d []byte) { le.PutUint32(d[mercExpOffset:], 5) }, false, true},
		{"name only", func(d []byte) { le.PutUint16(d[mercNameOffset:], 3) }, false, true},
		{"id only", func(d []byte) { le.PutUint32(d[mercIDOffset:], 9) }, false, true},
		{"dead bit 0", func(d []byte) { d[mercDeadOffset] = 1; le.PutUint32(d[mercIDOffset:], 9) }, true, true},
		{"other bits are not dead", func(d []byte) { d[mercDeadOffset] = 2; d[mercDeadOffset+1] = 1 }, false, false},
	}

	for _, tc := range tests {
		data := buildSave("Merc", Paladin, 12, StatusNewCharacter|StatusExpansion)
		tc.edit(data)
		le.PutUint32(data[checksumOffset:], 0)
		le.PutUint32(data[checksumOffset:], Checksum(data))

		h, err := ParseHeader(data)
		if err != nil {
			t.Fatal(err)
		}

		if h.Mercenary.Dead != tc.dead || h.Mercenary.Exists() != tc.exists {
			t.Errorf("%s: dead=%v exists=%v, want %v %v", tc.name, h.Mercenary.Dead, h.Mercenary.Exists(), tc.dead, tc.exists)
		}
	}
}
