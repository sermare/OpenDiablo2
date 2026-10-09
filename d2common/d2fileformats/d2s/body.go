package d2s

import (
	"bytes"
	"errors"
	"fmt"
)

// Section sizes and tags in a version 0x60 save, in file order after the
// header. Every section starts with a tag; the stats section is a bit stream.
const (
	questsOffset    = HeaderSize
	questsSize      = 0x12A
	questsPerDiff   = 96
	questsHeaderLen = 10
	numDifficulties = 3

	waypointsSize      = 0x50
	waypointsHeaderLen = 8
	waypointRecordLen  = 24
	waypointFlagsLen   = 5

	npcSize = 0x34

	statIDBits  = 9
	statEndID   = 0x1FF
	numSkills   = 30
	skillsSize  = 2 + numSkills
	statsTagLen = 2
)

var (
	questsTag    = [4]byte{'W', 'o', 'o', '!'}
	waypointsTag = [2]byte{'W', 'S'}
	npcTag       = [2]byte{0x01, 0x77}
	statsTag     = [2]byte{'g', 'f'}
	skillsTag    = [2]byte{'i', 'f'}
)

// Errors returned while reading the body sections.
var (
	ErrTruncated   = errors.New("d2s: file ends inside a section")
	ErrBadSection  = errors.New("d2s: unexpected section tag")
	ErrUnknownStat = errors.New("d2s: unknown character stat id")
)

// Character stat ids stored in the stats section.
const (
	StatStrength     = 0
	StatEnergy       = 1
	StatDexterity    = 2
	StatVitality     = 3
	StatUnusedStats  = 4
	StatUnusedSkills = 5
	StatCurrentHP    = 6
	StatMaxHP        = 7
	StatCurrentMana  = 8
	StatMaxMana      = 9
	StatCurrentStam  = 10
	StatMaxStamina   = 11
	StatLevel        = 12
	StatExperience   = 13
	StatGold         = 14
	StatStashedGold  = 15
)

// StatInfo describes how one stat id is stored in the stats section.
type StatInfo struct {
	Bits      int // width of the value
	ParamBits int // width of the parameter read before the value, 0 for none
}

// StatInfoFunc looks up the storage of a stat id; ok is false for unknown ids.
type StatInfoFunc func(id int) (info StatInfo, ok bool)

// defaultCharacterStats are the widths of the sixteen stats a character
// section normally contains (verified against real 1.14b saves).
var defaultCharacterStats = [...]StatInfo{
	StatStrength: {10, 0}, StatEnergy: {10, 0}, StatDexterity: {10, 0}, StatVitality: {10, 0},
	StatUnusedStats: {10, 0}, StatUnusedSkills: {8, 0},
	StatCurrentHP: {21, 0}, StatMaxHP: {21, 0}, StatCurrentMana: {21, 0}, StatMaxMana: {21, 0},
	StatCurrentStam: {21, 0}, StatMaxStamina: {21, 0},
	StatLevel: {7, 0}, StatExperience: {32, 0}, StatGold: {25, 0}, StatStashedGold: {25, 0},
}

// DefaultStatInfo knows the sixteen core character stats.
func DefaultStatInfo(id int) (StatInfo, bool) {
	if id < 0 || id >= len(defaultCharacterStats) {
		return StatInfo{}, false
	}

	return defaultCharacterStats[id], true
}

// Stat is one entry of the stats section.
type Stat struct {
	ID    int
	Param uint32
	Value uint64
}

// Attributes are the named core stats of a character. Hit points, mana and
// stamina are stored with eight fractional bits and reported as whole points.
type Attributes struct {
	Strength, Energy, Dexterity, Vitality uint64
	UnusedStats, UnusedSkillPoints        uint64
	CurrentHP, MaxHP                      uint64
	CurrentMana, MaxMana                  uint64
	CurrentStamina, MaxStamina            uint64
	Level                                 uint64
	Experience                            uint64
	Gold, StashedGold                     uint64
}

// Waypoints holds, per difficulty, the 40-bit waypoint mask of the save.
type Waypoints [numDifficulties]uint64

// Body is everything between the header and the item list.
type Body struct {
	// Quests holds the raw 96 quest bytes of normal, nightmare and hell.
	Quests [numDifficulties][questsPerDiff]byte
	// Waypoints is the activated-waypoint bit mask for each difficulty.
	Waypoints Waypoints
	// NPC is the raw NPC introduction/return flag block.
	NPC [npcSize - 2]byte
	// Stats lists every stat in file order; Attributes names the core ones.
	Stats      []Stat
	Attributes Attributes
	// SkillPoints is the points spent in each of the class' 30 skills.
	SkillPoints [numSkills]byte
	// ItemsOffset is where the first item section ('JM') starts.
	ItemsOffset int
}

// ParseBody reads the quest, waypoint, NPC, stats and skill sections of a
// save whose header has already been validated. statInfo may be nil to use
// DefaultStatInfo.
func ParseBody(data []byte, statInfo StatInfoFunc) (*Body, error) {
	if statInfo == nil {
		statInfo = DefaultStatInfo
	}

	b := &Body{}
	pos := questsOffset

	if err := b.readQuests(data, &pos); err != nil {
		return nil, err
	}

	if err := b.readWaypoints(data, &pos); err != nil {
		return nil, err
	}

	if err := b.readNPC(data, &pos); err != nil {
		return nil, err
	}

	if err := b.readStats(data, &pos, statInfo); err != nil {
		return nil, err
	}

	if err := b.readSkills(data, &pos); err != nil {
		return nil, err
	}

	b.ItemsOffset = pos

	return b, nil
}

func need(data []byte, pos, n int) error {
	if pos+n > len(data) {
		return fmt.Errorf("%w at 0x%X", ErrTruncated, pos)
	}

	return nil
}

func (b *Body) readQuests(data []byte, pos *int) error {
	if err := need(data, *pos, questsSize); err != nil {
		return err
	}

	section := data[*pos : *pos+questsSize]
	if !bytes.Equal(section[:4], questsTag[:]) {
		return fmt.Errorf("%w: quests at 0x%X", ErrBadSection, *pos)
	}

	for d := 0; d < numDifficulties; d++ {
		start := questsHeaderLen + d*questsPerDiff
		copy(b.Quests[d][:], section[start:start+questsPerDiff])
	}

	*pos += questsSize

	return nil
}

func (b *Body) readWaypoints(data []byte, pos *int) error {
	if err := need(data, *pos, waypointsSize); err != nil {
		return err
	}

	section := data[*pos : *pos+waypointsSize]
	if !bytes.Equal(section[:2], waypointsTag[:]) {
		return fmt.Errorf("%w: waypoints at 0x%X", ErrBadSection, *pos)
	}

	for d := 0; d < numDifficulties; d++ {
		// each record is a 2-byte marker followed by the waypoint bit mask
		start := waypointsHeaderLen + d*waypointRecordLen + 2

		var mask uint64
		for i := 0; i < waypointFlagsLen; i++ {
			mask |= uint64(section[start+i]) << (8 * i)
		}

		b.Waypoints[d] = mask
	}

	*pos += waypointsSize

	return nil
}

func (b *Body) readNPC(data []byte, pos *int) error {
	if err := need(data, *pos, npcSize); err != nil {
		return err
	}

	section := data[*pos : *pos+npcSize]
	if !bytes.Equal(section[:2], npcTag[:]) {
		return fmt.Errorf("%w: NPC block at 0x%X", ErrBadSection, *pos)
	}

	copy(b.NPC[:], section[2:])
	*pos += npcSize

	return nil
}

// bitStream reads little-endian packed bits, least significant bit first.
type bitStream struct {
	data []byte
	pos  int // bit position
}

func (s *bitStream) read(n int) (uint64, error) {
	if s.pos+n > len(s.data)*8 {
		return 0, ErrTruncated
	}

	var v uint64

	for i := 0; i < n; i++ {
		bit := (s.data[s.pos>>3] >> (s.pos & 7)) & 1
		v |= uint64(bit) << i
		s.pos++
	}

	return v, nil
}

// bytePos returns the next whole byte after the bits read so far.
func (s *bitStream) bytePos() int { return (s.pos + 7) / 8 }

func (b *Body) readStats(data []byte, pos *int, statInfo StatInfoFunc) error {
	if err := need(data, *pos, statsTagLen); err != nil {
		return err
	}

	if !bytes.Equal(data[*pos:*pos+statsTagLen], statsTag[:]) {
		return fmt.Errorf("%w: stats at 0x%X", ErrBadSection, *pos)
	}

	bs := &bitStream{data: data, pos: (*pos + statsTagLen) * 8}

	for {
		id, err := bs.read(statIDBits)
		if err != nil {
			return err
		}

		if id == statEndID {
			break
		}

		info, ok := statInfo(int(id))
		if !ok || info.Bits == 0 {
			return fmt.Errorf("%w: %d", ErrUnknownStat, id)
		}

		var param uint64
		if info.ParamBits > 0 {
			if param, err = bs.read(info.ParamBits); err != nil {
				return err
			}
		}

		value, err := bs.read(info.Bits)
		if err != nil {
			return err
		}

		b.Stats = append(b.Stats, Stat{ID: int(id), Param: uint32(param), Value: value})
		b.Attributes.set(int(id), value)
	}

	*pos = bs.bytePos()

	return nil
}

const fractionalBits = 8

func (a *Attributes) set(id int, v uint64) {
	switch id {
	case StatStrength:
		a.Strength = v
	case StatEnergy:
		a.Energy = v
	case StatDexterity:
		a.Dexterity = v
	case StatVitality:
		a.Vitality = v
	case StatUnusedStats:
		a.UnusedStats = v
	case StatUnusedSkills:
		a.UnusedSkillPoints = v
	case StatCurrentHP:
		a.CurrentHP = v >> fractionalBits
	case StatMaxHP:
		a.MaxHP = v >> fractionalBits
	case StatCurrentMana:
		a.CurrentMana = v >> fractionalBits
	case StatMaxMana:
		a.MaxMana = v >> fractionalBits
	case StatCurrentStam:
		a.CurrentStamina = v >> fractionalBits
	case StatMaxStamina:
		a.MaxStamina = v >> fractionalBits
	case StatLevel:
		a.Level = v
	case StatExperience:
		a.Experience = v
	case StatGold:
		a.Gold = v
	case StatStashedGold:
		a.StashedGold = v
	}
}

func (b *Body) readSkills(data []byte, pos *int) error {
	if err := need(data, *pos, skillsSize); err != nil {
		return err
	}

	if !bytes.Equal(data[*pos:*pos+2], skillsTag[:]) {
		return fmt.Errorf("%w: skills at 0x%X", ErrBadSection, *pos)
	}

	copy(b.SkillPoints[:], data[*pos+2:*pos+skillsSize])
	*pos += skillsSize

	return nil
}
