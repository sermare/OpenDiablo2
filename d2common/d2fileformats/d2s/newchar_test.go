package d2s

import (
	"encoding/binary"
	"errors"
	"os"
	"testing"
	"time"
)

func TestValidateName(t *testing.T) {
	tests := []struct {
		name string
		want error
	}{
		{"Maricon", nil}, {"ab", nil}, {"Ab-cd", nil}, {"Ab_cd", nil}, {"abcdefghijklmno", nil},
		{"a", ErrNameTooShort}, {"", ErrNameTooShort}, {"abcdefghijklmnop", ErrNameTooLong},
		{"-ab", ErrNameCharacters}, {"ab_", ErrNameCharacters}, {"a-b-c", ErrNameCharacters},
		{"ab1", ErrNameCharacters}, {"a b", ErrNameCharacters},
	}

	for _, tc := range tests {
		if got := ValidateName(tc.name); !errors.Is(got, tc.want) {
			t.Errorf("ValidateName(%q) = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestNewCharacterParses(t *testing.T) {
	tests := []struct {
		class Class
		flags NewCharacterFlags
		want  uint32
	}{
		{Amazon, NewCharacterFlags{}, StatusNewCharacter},
		{Druid, NewCharacterFlags{Expansion: true}, StatusNewCharacter | StatusExpansion},
		{Assassin, NewCharacterFlags{Expansion: true, Hardcore: true}, StatusNewCharacter | StatusExpansion | StatusHardcore},
		{Necromancer, NewCharacterFlags{Hardcore: true, Ladder: true}, StatusNewCharacter | StatusHardcore | StatusLadder},
	}

	for _, tc := range tests {
		tc.flags.Created = time.Unix(1700000000, 0)

		data, err := NewCharacter("Tester", tc.class, tc.flags, DefaultAppearance(tc.class))
		if err != nil {
			t.Fatal(err)
		}

		if len(data) != HeaderSize {
			t.Fatalf("size = %d, want %d", len(data), HeaderSize)
		}

		h, err := ParseHeader(data)
		if err != nil {
			t.Fatalf("%v: %v", tc.class, err)
		}

		if h.Status != tc.want || h.Class != tc.class || h.Level != 1 || h.SkillCount != 30 || h.Name != "Tester" {
			t.Errorf("%v: unexpected header %+v", tc.class, h)
		}

		if !h.IsNewCharacter() || h.HasBody() || h.IsDead() {
			t.Errorf("%v: new character flags wrong: %#x", tc.class, h.Status)
		}

		if binary.LittleEndian.Uint32(data[createdOffset:]) != 1700000000 ||
			binary.LittleEndian.Uint32(data[lastPlayedOffset:]) != 1700000000 {
			t.Errorf("%v: time stamps not written", tc.class)
		}
	}
}

func TestNewCharacterRejects(t *testing.T) {
	if _, err := NewCharacter("x", Druid, NewCharacterFlags{}, Appearance{}); !errors.Is(err, ErrNameTooShort) {
		t.Errorf("short name: %v", err)
	}

	if _, err := NewCharacter("Okay", Class(9), NewCharacterFlags{}, Appearance{}); !errors.Is(err, ErrInvalidClass) {
		t.Errorf("bad class: %v", err)
	}
}

func TestDiffHeaders(t *testing.T) {
	a, _ := NewCharacter("Same", Druid, NewCharacterFlags{Created: time.Unix(1, 0)}, DefaultAppearance(Druid))
	b, _ := NewCharacter("Same", Druid, NewCharacterFlags{Created: time.Unix(2, 0)}, DefaultAppearance(Druid))

	diffs := DiffHeaders(a, b)
	if len(diffs) != 3 { // created, lastPlayed, checksum
		t.Fatalf("diffs = %v", diffs)
	}

	for _, d := range diffs {
		if !d.Expected {
			t.Errorf("%v should be expected", d)
		}
	}

	if rep, n := DiffReport(a, a); rep != "identical" || n != 0 {
		t.Errorf("report = %q %d", rep, n)
	}

	c, _ := NewCharacter("Other", Druid, NewCharacterFlags{Created: time.Unix(1, 0)}, DefaultAppearance(Druid))
	if _, n := DiffReport(a, c); n == 0 {
		t.Error("a name change must be unexpected")
	}
}

// TestNewCharacterMatchesRealGame compares against a real new Druid created by
// the game (D2S_SAMPLE_NEW=<Maricon.d2s>): every byte must be equal except the
// time stamps, which are taken from the sample.
func TestNewCharacterMatchesRealGame(t *testing.T) {
	path := os.Getenv("D2S_SAMPLE_NEW")
	if path == "" {
		t.Skip("D2S_SAMPLE_NEW not set")
	}

	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	h, err := ParseHeader(want)
	if err != nil {
		t.Fatal(err)
	}

	created := time.Unix(int64(binary.LittleEndian.Uint32(want[createdOffset:])), 0)

	got, err := NewCharacter(h.Name, h.Class,
		NewCharacterFlags{Expansion: h.IsExpansion(), Hardcore: h.IsHardcore(), Ladder: h.IsLadder(), Created: created},
		DefaultAppearance(h.Class))
	if err != nil {
		t.Fatal(err)
	}

	if string(got) != string(want) {
		rep, _ := DiffReport(got, want)
		t.Fatalf("differs from the real file: %s", rep)
	}

	// the same character created at another time differs only in the time fields
	later, _ := NewCharacter(h.Name, h.Class, NewCharacterFlags{Expansion: true, Created: created.Add(time.Hour)},
		DefaultAppearance(h.Class))
	if rep, n := DiffReport(later, want); n != 0 {
		t.Fatalf("unexpected differences: %s", rep)
	}
}
