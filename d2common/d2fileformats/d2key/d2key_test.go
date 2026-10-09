package d2key

import (
	"os"
	"path/filepath"
	"testing"
)

func synthetic() *File {
	f := &File{Records: make([]Record, RecordCount)}
	for i := range f.Records {
		f.Records[i] = Record{Command: uint32(i), Primary: Unbound, Secondary: Unbound, FlagA: 1}
	}

	f.Records[0].Primary = 'A'
	f.Records[0].Secondary = 'C'
	f.Records[7] = Record{Command: 21, Primary: 0x77, FlagA: 1, Secondary: Unbound}

	return f
}

func TestParseSynthetic(t *testing.T) {
	data := synthetic().Marshal()
	if len(data) != FileSize {
		t.Fatalf("size %d", len(data))
	}

	f, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}

	if r, ok := f.Lookup(0); !ok || r.Primary != 'A' || r.Secondary != 'C' {
		t.Errorf("command 0 = %+v %v", r, ok)
	}

	if r, ok := f.Lookup(21); !ok || r.Primary != 0x77 {
		t.Errorf("command 21 (out of order) = %+v %v", r, ok)
	}

	if _, ok := f.Lookup(999); ok {
		t.Error("unexpected record 999")
	}
}

func TestParseErrors(t *testing.T) {
	good := synthetic().Marshal()

	badMagic := append([]byte(nil), good...)
	badMagic[0] = 'X'

	badDup := append([]byte(nil), good...)
	badDup[HeaderSize+10] = 99

	tests := []struct {
		name string
		data []byte
	}{
		{"short", good[:100]},
		{"empty", nil},
		{"magic", badMagic},
		{"dup id", badDup},
	}

	for _, tc := range tests {
		if _, err := Parse(tc.data); err == nil {
			t.Errorf("%s: expected error", tc.name)
		}
	}
}

func TestIsKeyboard(t *testing.T) {
	for code, want := range map[uint16]bool{0x41: true, Unbound: false, 0x103: false, 0xff: true} {
		if got := IsKeyboard(code); got != want {
			t.Errorf("IsKeyboard(%#x)=%v", code, got)
		}
	}
}

// TestRealDefaultKey reads the genuine file when D2_DEFAULT_KEY (a path) is
// set, or D2_INSTALL (the game directory) contains default.key.
func TestRealDefaultKey(t *testing.T) {
	path := os.Getenv("D2_DEFAULT_KEY")
	if path == "" && os.Getenv("D2_INSTALL") != "" {
		path = filepath.Join(os.Getenv("D2_INSTALL"), "default.key")
	}

	if path == "" {
		t.Skip("D2_DEFAULT_KEY / D2_INSTALL not set")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Skip(err)
	}

	f, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}

	want := map[uint32][2]uint16{
		0: {'A', 'C'}, 1: {'B', 'I'}, 2: {'P', Unbound}, 7: {0x09, 0x100},
		21: {0x77, Unbound}, 22: {0xc0, Unbound}, 56: {0x1b, Unbound}, 46: {Unbound, Unbound},
	}

	for id, keys := range want {
		r, ok := f.Lookup(id)
		if !ok || r.Primary != keys[0] || r.Secondary != keys[1] {
			t.Errorf("command %d = %+v, want %#x/%#x", id, r, keys[0], keys[1])
		}
	}

	for id := range Commands {
		if _, ok := f.Lookup(id); !ok {
			t.Errorf("named command %d missing from real file", id)
		}
	}
}
