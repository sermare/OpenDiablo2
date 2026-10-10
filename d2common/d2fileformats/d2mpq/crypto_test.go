package d2mpq

import (
	"bytes"
	"encoding/binary"
	"io"
	"testing"
)

// naiveDecryptTable is the original word-at-a-time implementation, kept as the reference.
func naiveDecryptTable(r io.Reader, size uint32, name string) ([]uint32, error) {
	cryptoLookup(0)

	seed := hashString(name, 3)
	seed2 := uint32(0xEEEEEEEE)
	size *= 4

	table := make([]uint32, size)
	buf := make([]byte, 4)

	for i := uint32(0); i < size; i++ {
		seed2 += cryptoBuffer[0x400+(seed&0xff)]

		if _, err := r.Read(buf); err != nil {
			return table, err
		}

		result := binary.LittleEndian.Uint32(buf)
		result ^= seed + seed2

		seed = ((^seed << 21) + 0x11111111) | (seed >> 11)
		seed2 = result + seed2 + (seed2 << 5) + 3
		table[i] = result
	}

	return table, nil
}

func TestDecryptTableMatchesWordAtATime(t *testing.T) {
	tests := []struct {
		name    string
		entries uint32
		extra   int // bytes missing from the end (truncated archive)
	}{
		{"(hash table)", 1, 0},
		{"(hash table)", 512, 0},
		{"(block table)", 37, 0},
		{"(block table)", 4, 5},
	}

	for _, tt := range tests {
		data := make([]byte, int(tt.entries)*16-tt.extra)
		for i := range data {
			data[i] = byte(i*131 + 7)
		}

		want, wantErr := naiveDecryptTable(bytes.NewReader(data), tt.entries, tt.name)
		got, gotErr := decryptTable(bytes.NewReader(data), tt.entries, tt.name)

		if (wantErr == nil) != (gotErr == nil) {
			t.Fatalf("%s/%d: error %v, want %v", tt.name, tt.entries, gotErr, wantErr)
		}

		if tt.extra == 0 {
			for i := range want {
				if want[i] != got[i] {
					t.Fatalf("%s/%d: word %d = %#x, want %#x", tt.name, tt.entries, i, got[i], want[i])
				}
			}
		}
	}
}

func BenchmarkDecryptTable(b *testing.B) {
	data := make([]byte, 65536*16)
	b.SetBytes(int64(len(data)))

	for i := 0; i < b.N; i++ {
		if _, err := decryptTable(bytes.NewReader(data), 65536, "(hash table)"); err != nil {
			b.Fatal(err)
		}
	}
}
