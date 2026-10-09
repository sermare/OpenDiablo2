package d2s

import (
	"encoding/binary"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

// The bytes of a brand new character's file. What the game writes was checked
// against a real file (an expansion Druid, 335 bytes) and against the header
// writer SAVE_WriteD2sHeader (0x566c70 in Game.exe 1.14b), which is the code
// that fills the same fields for every later save:
//
//	0x00 magic, 0x04 version 0x60, 0x08 size, 0x0C checksum   (verified)
//	0x10 active weapon set (0 on a new character)             (verified, sample)
//	0x14 name, NUL padded                                     (verified)
//	0x24 status: new 0x01 | hardcore 0x04 | expansion 0x20 | ladder 0x40
//	     (the writer ORs 0x20 and 0x40 into the client flags; verified)
//	0x28 class, 0x29 constant 0x10 (the writer stores the literal 0x10),
//	0x2A skill count 30 (a data byte; 30 skills per class), 0x2B level 1
//	0x2C creation time, 0x30 last-played time (unix seconds; equal in a new file)
//	0x34.. hotkeys, skills, ...: zero in the sample
//	0x88 appearance (16), 0x98 colours (16), 0xA8.. difficulty/act, map seed: see below
//
// Which code produces the header-only file is not located in the binary: the
// serialiser reached from the save paths always writes the full body, and
// writes 0xFFFFFFFF at 0x34 where the sample has zero. Treat the creation path
// as UNVERIFIED; the output below matches the real sample byte for byte.

// Errors returned by NewCharacter.
var (
	ErrNameTooShort   = errors.New("d2s: character name needs at least 2 characters")
	ErrNameTooLong    = errors.New("d2s: character name is longer than 15 characters")
	ErrNameCharacters = errors.New("d2s: character name may only hold letters and one '-' or '_' inside")
)

const (
	// MaxCharacterName is the longest name a save can hold (16 bytes, NUL terminated).
	MaxCharacterName = 15
	// MinCharacterName is the shortest name the creation screen accepts
	// (UNVERIFIED: the common rule of the game; the binary only shows the
	// "Name doesn't have alpha characters" and "bad characters" errors).
	MinCharacterName = 2

	defaultUnknown29 = 0x10
	defaultSkills    = 30
	createdOffset    = 0x2C
	lastPlayedOffset = 0x30
	weaponSetOffset  = 0x10
)

// NewCharacterFlags are the choices made on the creation screen.
type NewCharacterFlags struct {
	Expansion bool
	Hardcore  bool
	Ladder    bool
	// Created is the creation time; the zero value means time.Now().
	Created time.Time
}

// Appearance is the composite look stored at 0x88 (graphics) and 0x98 (colours).
// 0xFF means "none" in both.
type Appearance struct {
	Graphics [appearanceLen]byte
	Colors   [colorsLen]byte
}

// DefaultAppearance returns the look of a new character of the class. The
// Druid values are those of a real new character; the other classes use the
// same bytes and are UNVERIFIED (only one real new-character file was
// available).
func DefaultAppearance(Class) Appearance {
	a := Appearance{}

	for i := range a.Graphics {
		a.Graphics[i] = 0xFF
	}

	for i := range a.Colors {
		a.Colors[i] = 0xFF
	}

	// head, torso, legs, right arm, left arm = 1; weapons and shield none;
	// the next two entries = 1 (seen in the sample)
	copy(a.Graphics[:5], []byte{1, 1, 1, 1, 1})
	a.Graphics[8], a.Graphics[9] = 1, 1

	return a
}

// ValidateName checks a character name: 2 to 15 characters, letters, with at
// most one '-' or '_' that is neither first nor last (the char-select list
// also skips names ending in '-' or starting with '_', menus-libs.md).
func ValidateName(name string) error {
	switch {
	case len(name) < MinCharacterName:
		return ErrNameTooShort
	case len(name) > MaxCharacterName:
		return ErrNameTooLong
	}

	separators := 0

	for i := 0; i < len(name); i++ {
		c := name[i]

		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z':
		case c == '-' || c == '_':
			separators++

			if i == 0 || i == len(name)-1 {
				return ErrNameCharacters
			}
		default:
			return ErrNameCharacters
		}
	}

	if separators > 1 {
		return ErrNameCharacters
	}

	return nil
}

// NewCharacter returns the file the game writes for a brand new character: the
// 335 byte header with the "new" status bit and no body. The checksum and size
// are final. Fields that are not derived from the arguments (time stamps) come
// from flags.Created.
func NewCharacter(name string, class Class, flags NewCharacterFlags, look Appearance) ([]byte, error) {
	if err := ValidateName(name); err != nil {
		return nil, err
	}

	if class > Assassin {
		return nil, ErrInvalidClass
	}

	created := flags.Created
	if created.IsZero() {
		created = time.Now()
	}

	le := binary.LittleEndian
	out := make([]byte, HeaderSize)

	le.PutUint32(out[0:], Magic)
	le.PutUint32(out[4:], MaxVersion)
	le.PutUint32(out[8:], HeaderSize)
	copy(out[nameOffset:nameOffset+nameLength], name)

	status := StatusNewCharacter
	if flags.Hardcore {
		status |= StatusHardcore
	}

	if flags.Expansion {
		status |= StatusExpansion
	}

	if flags.Ladder {
		status |= StatusLadder
	}

	le.PutUint32(out[statusOffset:], status)
	out[classOffset] = byte(class)
	out[0x29] = defaultUnknown29
	out[skillCountPos] = defaultSkills
	out[levelOffset] = 1
	le.PutUint32(out[createdOffset:], uint32(created.Unix()))
	le.PutUint32(out[lastPlayedOffset:], uint32(created.Unix()))
	copy(out[appearanceStart:], look.Graphics[:])
	copy(out[colorsStart:], look.Colors[:])

	le.PutUint32(out[checksumOffset:], Checksum(out))

	return out, nil
}

// FieldDiff is one differing header field between two files.
type FieldDiff struct {
	Field string
	Start int
	End   int // exclusive
	Got   []byte
	Want  []byte
	// Expected is true for fields that legitimately differ between two
	// creations of the same character (time stamps and the checksum they feed).
	Expected bool
}

func (d FieldDiff) String() string {
	kind := "DIFF"
	if d.Expected {
		kind = "expected"
	}

	return fmt.Sprintf("%s %s [0x%X,0x%X) got=% X want=% X", kind, d.Field, d.Start, d.End, d.Got, d.Want)
}

type headerField struct {
	name       string
	start, end int
	volatile   bool
}

var headerFields = []headerField{
	{"magic", 0, 4, false}, {"version", 4, 8, false}, {"size", 8, 12, false},
	{"checksum", 12, 16, true}, {"weaponSet", 0x10, 0x14, false}, {"name", 0x14, 0x24, false},
	{"status", 0x24, 0x28, false}, {"class", 0x28, 0x29, false}, {"unknown29", 0x29, 0x2A, false},
	{"skillCount", 0x2A, 0x2B, false}, {"level", 0x2B, 0x2C, false},
	{"created", 0x2C, 0x30, true}, {"lastPlayed", 0x30, 0x34, true},
	{"hotkeys+skills", 0x34, 0x88, false}, {"appearance", 0x88, 0x98, false},
	{"colors", 0x98, 0xA8, false}, {"difficulty+seed+merc", 0xA8, HeaderSize, false},
}

// DiffHeaders compares two files field by field over the header and returns
// the fields that differ, in file order. Time stamps and the checksum are
// marked Expected. A file shorter than the header reports a "length" entry.
func DiffHeaders(got, want []byte) []FieldDiff {
	var out []FieldDiff

	if len(got) != len(want) {
		out = append(out, FieldDiff{Field: "length", Got: []byte{byte(len(got)), byte(len(got) >> 8)},
			Want: []byte{byte(len(want)), byte(len(want) >> 8)}})
	}

	if len(got) < HeaderSize || len(want) < HeaderSize {
		return out
	}

	for _, f := range headerFields {
		g, w := got[f.start:f.end], want[f.start:f.end]
		if string(g) == string(w) {
			continue
		}

		out = append(out, FieldDiff{Field: f.name, Start: f.start, End: f.end,
			Got: append([]byte(nil), g...), Want: append([]byte(nil), w...), Expected: f.volatile})
	}

	sort.SliceStable(out, func(i, j int) bool { return out[i].Start < out[j].Start })

	return out
}

// DiffReport renders DiffHeaders for a log: one line per difference or a
// single "identical" line. unexpected counts the differences that are not time
// related.
func DiffReport(got, want []byte) (report string, unexpected int) {
	diffs := DiffHeaders(got, want)
	if len(diffs) == 0 {
		return "identical", 0
	}

	var b strings.Builder

	for i, d := range diffs {
		if i > 0 {
			b.WriteString("; ")
		}

		b.WriteString(d.String())

		if !d.Expected {
			unexpected++
		}
	}

	return b.String(), unexpected
}

// Promote turns a character parsed from a header-only new-character file into
// one with an empty body, which is what Write needs to save a character that
// has been played. The sections are laid out as a real 1.14b save has them
// (VERIFIED against the NokkaSorc sample: "Woo!" 6 0x12A, "WS" 1 0x50 with
// records starting 02 01, NPC tag 01 77); the quest, waypoint and NPC content
// starts empty except that the first waypoint of normal difficulty (Rogue
// Encampment) is active (UNVERIFIED: the first-save content of the real game
// was not captured). The new-character bit is cleared, normal difficulty Act I
// is marked active, and the item lists are empty: starting items are not
// fabricated.
func (c *Character) Promote() {
	if c == nil || c.Header == nil || c.Body != nil {
		return
	}

	c.Header.Status &^= StatusNewCharacter

	if _, _, ok := c.Header.ActiveDifficulty(); !ok {
		c.Header.Difficulty = [3]byte{activeFlag, 0, 0}
	}

	b := &Body{}
	copy(b.QuestsRaw[:], questsTag[:])
	b.QuestsRaw[4] = 6
	b.QuestsRaw[8], b.QuestsRaw[9] = 0x2A, 0x01 // the 0x12A section size

	copy(b.WaypointsRaw[:], waypointsTag[:])
	b.WaypointsRaw[2] = 1
	b.WaypointsRaw[6] = waypointsSize

	for d := 0; d < numDifficulties; d++ {
		rec := waypointsHeaderLen + d*waypointRecordLen
		b.WaypointsRaw[rec], b.WaypointsRaw[rec+1] = 2, 1
	}

	b.Waypoints[0] = 1 << uint(WPRogueEncampment)
	c.Body = b
}
