package d2s

import (
	"encoding/binary"
	"errors"
	"os"
	"testing"
)

// buildSave returns a minimal valid save with the given name, class, level
// and status flags, with a correct size field and checksum.
func buildSave(name string, class Class, level uint8, status uint32) []byte {
	data := make([]byte, HeaderSize)
	le := binary.LittleEndian

	le.PutUint32(data[0:], Magic)
	le.PutUint32(data[4:], MaxVersion)
	le.PutUint32(data[8:], uint32(len(data)))
	copy(data[nameOffset:], name)
	le.PutUint32(data[statusOffset:], status)
	data[classOffset] = byte(class)
	data[skillCountPos] = 30
	data[levelOffset] = level

	le.PutUint32(data[checksumOffset:], Checksum(data))

	return data
}

func TestParseHeader(t *testing.T) {
	data := buildSave("Maricon", Paladin, 12, StatusNewCharacter|StatusExpansion|StatusLadder)

	h, err := ParseHeader(data)
	if err != nil {
		t.Fatal(err)
	}

	if h.Name != "Maricon" || h.Class != Paladin || h.Level != 12 || h.SkillCount != 30 {
		t.Fatalf("unexpected header: %+v", h)
	}

	if !h.IsNewCharacter() || !h.IsExpansion() || !h.IsLadder() || h.IsHardcore() || h.IsDead() || h.HasBody() {
		t.Fatalf("unexpected flags: %#x", h.Status)
	}

	if h.Class.String() != "Paladin" {
		t.Fatalf("class name = %q", h.Class.String())
	}
}

func TestParseHeaderErrors(t *testing.T) {
	tests := []struct {
		name   string
		mutate func([]byte) []byte
		want   error
	}{
		{"too short", func(d []byte) []byte { return d[:HeaderSize-1] }, ErrTooShort},
		{"bad magic", func(d []byte) []byte { d[0] ^= 0xFF; return d }, ErrBadMagic},
		{"bad checksum", func(d []byte) []byte { d[levelOffset]++; return d }, ErrBadChecksum},
		{"bad size", func(d []byte) []byte {
			binary.LittleEndian.PutUint32(d[8:], 1)
			binary.LittleEndian.PutUint32(d[checksumOffset:], Checksum(d))

			return d
		}, ErrBadFileSize},
		{"old version", func(d []byte) []byte {
			binary.LittleEndian.PutUint32(d[4:], MinVersion-1)
			binary.LittleEndian.PutUint32(d[checksumOffset:], Checksum(d))

			return d
		}, ErrBadVersion},
		{"bad class", func(d []byte) []byte {
			d[classOffset] = 9
			binary.LittleEndian.PutUint32(d[checksumOffset:], Checksum(d))

			return d
		}, ErrInvalidClass},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			data := tc.mutate(buildSave("Test", Amazon, 1, 0))

			if _, err := ParseHeader(data); !errors.Is(err, tc.want) {
				t.Fatalf("got %v, want %v", err, tc.want)
			}
		})
	}
}

// TestRealSave checks a real game save when D2S_SAMPLE points at one.
// The file is only read, never copied into the repository.
func TestRealSave(t *testing.T) {
	path := os.Getenv("D2S_SAMPLE")
	if path == "" {
		t.Skip("set D2S_SAMPLE to a .d2s file to run")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	h, err := ParseHeader(data)
	if err != nil {
		t.Fatalf("%s: %v", path, err)
	}

	t.Logf("%s: name=%q class=%v level=%d expansion=%v ladder=%v new=%v version=0x%X checksum=0x%08X",
		path, h.Name, h.Class, h.Level, h.IsExpansion(), h.IsLadder(), h.IsNewCharacter(), h.Version, h.Checksum)
}
