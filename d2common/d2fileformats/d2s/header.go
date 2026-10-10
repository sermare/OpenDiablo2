package d2s

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math/bits"
)

const (
	// Magic is the signature every .d2s file starts with.
	Magic uint32 = 0xAA55AA55

	// HeaderSize is the smallest valid save: the fixed header with no body.
	HeaderSize = 0x14F

	// MinVersion and MaxVersion bound the file versions the game accepts
	// (0x60 is 1.14b; older versions use a legacy layout).
	MinVersion uint32 = 0x5C
	MaxVersion uint32 = 0x60

	nameOffset       = 0x14
	nameLength       = 16
	checksumOffset   = 0x0C
	checksumLength   = 4
	statusOffset     = 0x24
	classOffset      = 0x28
	skillCountPos    = 0x2A
	levelOffset      = 0x2B
	difficultyOffset = 0xA8 // three bytes: normal, nightmare, hell
	mapSeedOffset    = 0xAB
	mercDeadOffset   = 0xB1
	mercIDOffset     = 0xB3
	mercNameOffset   = 0xB7
	mercTypeOffset   = 0xB9
	mercExpOffset    = 0xBB
	appearanceStart  = 0x88
	appearanceLen    = 16
	colorsStart      = 0x98
	colorsLen        = 16
)

// Status flags stored at offset 0x24.
const (
	StatusNewCharacter uint32 = 0x01
	StatusHardcore     uint32 = 0x04
	StatusDied         uint32 = 0x08
	StatusExpansion    uint32 = 0x20
	StatusLadder       uint32 = 0x40
)

// Class is a character class as stored in the save file.
type Class uint8

// The eight classes the file format can hold.
const (
	Amazon Class = iota
	Sorceress
	Necromancer
	Paladin
	Barbarian
	Druid
	Assassin
)

// Validation errors, in the order the original game checks them.
var (
	ErrTooShort      = errors.New("d2s: file is shorter than the header")
	ErrBadMagic      = errors.New("d2s: bad magic number")
	ErrBadChecksum   = errors.New("d2s: checksum mismatch")
	ErrBadFileSize   = errors.New("d2s: file size field does not match the data")
	ErrBadVersion    = errors.New("d2s: unsupported file version")
	ErrInvalidClass  = errors.New("d2s: invalid character class")
	errNameTerminate = errors.New("d2s: character name is not terminated")
)

// Header holds the fixed-size fields at the start of a save file.
type Header struct {
	Version    uint32
	FileSize   uint32
	Checksum   uint32
	Name       string
	Status     uint32
	Class      Class
	SkillCount uint8
	Level      uint8
	Mercenary  Mercenary
	// Difficulty has one byte per difficulty; the active one has bit 0x80 set and
	// its low bits hold the act the character is in (0 = Act I).
	Difficulty [3]byte
	// MapSeed is the seed the game's level generator is started from. In single
	// player the game reads it from the save, so a character's maps are fixed.
	MapSeed    uint32
	Appearance [appearanceLen]byte
	Colors     [colorsLen]byte

	// Raw is the header exactly as read. Write starts from it so bytes this
	// package does not interpret survive a parse/write round trip.
	Raw [HeaderSize]byte
}

// Mercenary is the hired mercenary stored in the header. Dead is bit 0 of the
// byte at 0xB1 (VERIFIED, MERC_LoadFromD2sHeader 0x568730 tests bit 16 of the
// dword at 0xAF), not "any non-zero u16". A merc exists when the id, the
// experience or the name id is non-zero (Exists); the type alone does not
// count.
type Mercenary struct {
	Dead       bool
	ID         uint32
	NameID     uint16
	Type       uint16
	Experience uint32
}

// Exists reports whether the header holds a mercenary: the loader returns
// "none" only when id, experience and name id are all zero (VERIFIED, 0x568730).
func (m Mercenary) Exists() bool { return m.ID != 0 || m.Experience != 0 || m.NameID != 0 }

// Checksum computes the save checksum: the file is summed byte by byte with
// the checksum field treated as zero, rotating the running sum left by one
// bit before each addition.
func Checksum(data []byte) uint32 {
	var sum uint32

	for i, b := range data {
		if i >= checksumOffset && i < checksumOffset+checksumLength {
			b = 0
		}

		sum = bits.RotateLeft32(sum, 1) + uint32(b)
	}

	return sum
}

// ParseHeader validates data as a .d2s file and decodes its header.
func ParseHeader(data []byte) (*Header, error) {
	if len(data) < HeaderSize {
		return nil, ErrTooShort
	}

	le := binary.LittleEndian

	if le.Uint32(data[0:]) != Magic {
		return nil, ErrBadMagic
	}

	h := &Header{
		Version:  le.Uint32(data[4:]),
		FileSize: le.Uint32(data[8:]),
		Checksum: le.Uint32(data[checksumOffset:]),
		Status:   le.Uint32(data[statusOffset:]),
		Class:    Class(data[classOffset]),
	}

	if Checksum(data) != h.Checksum {
		return nil, ErrBadChecksum
	}

	if int(h.FileSize) != len(data) {
		return nil, ErrBadFileSize
	}

	if h.Version < MinVersion || h.Version > MaxVersion {
		return nil, fmt.Errorf("%w: 0x%X (%s)", ErrBadVersion, h.Version, VersionName(h.Version))
	}

	if h.Class > Assassin {
		return nil, ErrInvalidClass
	}

	name, err := readName(data[nameOffset : nameOffset+nameLength])
	if err != nil {
		return nil, err
	}

	h.Name = name
	h.SkillCount = data[skillCountPos]
	h.Level = data[levelOffset]

	copy(h.Difficulty[:], data[difficultyOffset:difficultyOffset+3])
	h.MapSeed = le.Uint32(data[mapSeedOffset:])

	h.Mercenary = Mercenary{
		Dead:       data[mercDeadOffset]&1 != 0,
		ID:         le.Uint32(data[mercIDOffset:]),
		NameID:     le.Uint16(data[mercNameOffset:]),
		Type:       le.Uint16(data[mercTypeOffset:]),
		Experience: le.Uint32(data[mercExpOffset:]),
	}

	copy(h.Raw[:], data[:HeaderSize])
	copy(h.Appearance[:], data[appearanceStart:appearanceStart+appearanceLen])
	copy(h.Colors[:], data[colorsStart:colorsStart+colorsLen])

	return h, nil
}

func readName(raw []byte) (string, error) {
	for i, b := range raw {
		if b == 0 {
			return string(raw[:i]), nil
		}
	}

	return "", errNameTerminate
}

// IsNewCharacter reports whether the character has never been played; such a
// file contains only the header.
func (h *Header) IsNewCharacter() bool { return h.Status&StatusNewCharacter != 0 }

// IsHardcore reports whether the character is hardcore.
func (h *Header) IsHardcore() bool { return h.Status&StatusHardcore != 0 }

// IsDead reports whether a hardcore character has died.
func (h *Header) IsDead() bool { return h.Status&StatusDied != 0 }

// IsExpansion reports whether the character belongs to Lord of Destruction.
func (h *Header) IsExpansion() bool { return h.Status&StatusExpansion != 0 }

// IsLadder reports whether the character is a ladder character.
func (h *Header) IsLadder() bool { return h.Status&StatusLadder != 0 }

// HasBody reports whether the file has data after the fixed header.
func (h *Header) HasBody() bool { return !h.IsNewCharacter() }

// String returns the class name.
func (c Class) String() string {
	names := [...]string{"Amazon", "Sorceress", "Necromancer", "Paladin", "Barbarian", "Druid", "Assassin"}
	if int(c) < len(names) {
		return names[c]
	}

	return "Unknown"
}

// activeFlag marks the active difficulty in Header.Difficulty.
const activeFlag = 0x80

// ActiveDifficulty returns the active difficulty (0 normal, 1 nightmare,
// 2 hell) and the act (0 = Act I) the character is in. ok is false for a
// character that has never entered the game.
func (h *Header) ActiveDifficulty() (difficulty, act int, ok bool) {
	for i, b := range h.Difficulty {
		if b&activeFlag != 0 {
			return i, int(b &^ activeFlag), true
		}
	}

	return 0, 0, false
}
