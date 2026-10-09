package d2s

// The bit layout of items and the overall decoding order follow the
// community documentation of the .d2s format and github.com/nokka/d2s
// (MIT, Copyright (c) 2020 nokka), which was used as a reference oracle.
// This implementation is independent: magical properties are decoded from
// ItemStatCost (see ItemTables) instead of a hard-coded stat list.

import (
	"errors"
	"fmt"
	"math/bits"
)

// Item qualities.
const (
	QualityLow     uint8 = 1
	QualityNormal  uint8 = 2
	QualityHigh    uint8 = 3
	QualityMagic   uint8 = 4
	QualitySet     uint8 = 5
	QualityRare    uint8 = 6
	QualityUnique  uint8 = 7
	QualityCrafted uint8 = 8
)

// Item storage locations (Item.Location).
const (
	LocationStored   uint8 = 0 // inventory, stash or cube, see Item.Page
	LocationEquipped uint8 = 1
	LocationBelt     uint8 = 2
	LocationCursor   uint8 = 4
	LocationSocketed uint8 = 6
)

// Stat ids the item format itself depends on.
const (
	statArmorClass    = 31
	statDurability    = 72
	statMaxDurability = 73

	quantityBits  = 9
	propertyEndID = 0x1FF
)

// Errors returned by ParseItemList.
var (
	ErrNoItemList  = errors.New("d2s: item list does not start with JM")
	ErrNoItemTag   = errors.New("d2s: item does not start with JM")
	ErrNoTables    = errors.New("d2s: item tables are required")
	ErrUnknownItem = errors.New("d2s: unknown item code")
	ErrUnknownStat = errors.New("d2s: unknown or unsaved stat id")
)

// Property is one decoded magical property. Each stat id is reported
// separately; stats the game stores back to back (such as the minimum and
// maximum of a damage range) appear as consecutive entries.
type Property struct {
	ID    int
	Name  string // only set when the tables were loaded from the .txt form
	Param uint32 // value of the Save Param Bits field, 0 when absent
	Value int64  // stored value minus Save Add
}

// EarInfo is the payload of a player ear item.
type EarInfo struct {
	Class uint8
	Level uint8
	Name  string
}

// Item is one decoded item. Fields after Code are only meaningful for
// non-simple (extended) items.
type Item struct {
	Flags uint32 // the raw 32 flag bits

	Identified   bool
	Socketed     bool
	New          bool
	Ear          bool
	Starter      bool
	Simple       bool
	Ethereal     bool
	Personalized bool
	Runeword     bool

	Version  uint16 // 10 bit version field
	Location uint8
	Equipped uint8 // equipment slot when Location is LocationEquipped
	X, Y     uint8
	Page     uint8 // 1 inventory, 4 cube, 5 stash when Location is stored

	Code        string // 3 character item code, or "ear"
	EarInfo     *EarInfo
	SocketCount uint8 // number of items socketed into this one (3 bits)

	ID           uint32
	Level        uint8
	Quality      uint8
	HasPicture   bool
	Picture      uint8
	HasClassData bool
	ClassData    uint16 // 11 bits, auto-affix data

	LowQualityID uint8
	HighQuality  uint8 // 3 unexplained bits of high quality items
	MagicPrefix  uint16
	MagicSuffix  uint16
	SetID        uint16
	UniqueID     uint16
	RareName1    uint8
	RareName2    uint8
	RareAffixes  [6]uint16 // prefix/suffix ids, alternating, see RareMask
	RareMask     uint8     // bit i set when RareAffixes[i] was present

	RunewordID      uint16
	RunewordUnknown uint8 // 4 bits following the runeword id
	PersonalName    string
	TomeBits        uint8 // 5 bit spell field of tomes
	Timestamp       bool

	Defense       int
	MaxDurability uint16
	Durability    uint16
	Quantity      uint16
	TotalSockets  uint8
	SetListMask   uint8 // which set bonus lists follow

	Properties         []Property
	SetProperties      [][]Property
	RunewordProperties []Property

	Children []Item // items socketed into this one
}

// groupFollowers lists stats that the game stores immediately after the
// stat that leads the group, without a separate id: the min/max enhanced
// damage, and the min/max (and length) of fire, lightning, magic, cold and
// poison damage.
var groupFollowers = map[int][]int{
	17: {18},
	48: {49},
	50: {51},
	52: {53},
	54: {55, 56},
	57: {58, 59},
}

// ParseItemList parses a '(JM)' item list: the 'JM' tag, a u16 item count
// and that many top-level items, each of which is itself prefixed by 'JM'
// and followed by its socketed children. data must start at the list tag;
// consumed is the number of bytes the list occupies.
func ParseItemList(data []byte, tables *ItemTables) (items []Item, consumed int, err error) {
	if tables == nil {
		return nil, 0, ErrNoTables
	}

	if len(data) < 4 {
		return nil, 0, ErrUnexpectedEOF
	}

	if data[0] != 'J' || data[1] != 'M' {
		return nil, 0, ErrNoItemList
	}

	count := int(data[2]) | int(data[3])<<8
	off := 4

	for i := 0; i < count; i++ {
		item, n, perr := parseItem(data[off:], tables)
		if perr != nil {
			return items, off, fmt.Errorf("item %d at offset %d: %w", i, off, perr)
		}

		off += n

		if !item.Simple {
			for c := 0; c < int(item.SocketCount); c++ {
				child, cn, cerr := parseItem(data[off:], tables)
				if cerr != nil {
					return items, off, fmt.Errorf("item %d socketed item %d: %w", i, c, cerr)
				}

				off += cn
				item.Children = append(item.Children, child)
			}
		}

		items = append(items, item)
	}

	return items, off, nil
}

// parseItem decodes one item (without its socketed children) and returns
// the number of bytes it occupies.
func parseItem(data []byte, t *ItemTables) (Item, int, error) {
	r := newBitReader(data)
	it := Item{}

	if err := it.readHeader(r); err != nil {
		return it, 0, err
	}

	if it.Ear {
		if err := it.readEar(r); err != nil {
			return it, 0, err
		}

		r.align()

		return it, r.bytesUsed(), nil
	}

	if err := it.readCode(r); err != nil {
		return it, 0, err
	}

	if !it.Simple {
		if err := it.readExtended(r, t); err != nil {
			return it, 0, err
		}
	}

	r.align()

	return it, r.bytesUsed(), nil
}

func (it *Item) readHeader(r *bitReader) error {
	tag, err := r.read(16)
	if err != nil {
		return err
	}

	if tag != 'J'|'M'<<8 {
		return ErrNoItemTag
	}

	f, err := r.read(32)
	if err != nil {
		return err
	}

	it.Flags = uint32(f)
	it.Identified = f>>4&1 == 1
	it.Socketed = f>>11&1 == 1
	it.New = f>>13&1 == 1
	it.Ear = f>>16&1 == 1
	it.Starter = f>>17&1 == 1
	it.Simple = f>>21&1 == 1
	it.Ethereal = f>>22&1 == 1
	it.Personalized = f>>24&1 == 1
	it.Runeword = f>>26&1 == 1

	fields := []struct {
		bits int
		set  func(uint64)
	}{
		{10, func(v uint64) { it.Version = uint16(v) }},
		{3, func(v uint64) { it.Location = uint8(v) }},
		{4, func(v uint64) { it.Equipped = uint8(v) }},
		{4, func(v uint64) { it.X = uint8(v) }},
		{4, func(v uint64) { it.Y = uint8(v) }},
		{3, func(v uint64) { it.Page = uint8(v) }},
	}

	for _, fd := range fields {
		v, rerr := r.read(fd.bits)
		if rerr != nil {
			return rerr
		}

		fd.set(v)
	}

	return nil
}

func (it *Item) readEar(r *bitReader) error {
	class, err := r.read(3)
	if err != nil {
		return err
	}

	level, err := r.read(7)
	if err != nil {
		return err
	}

	name, err := readCString7(r)
	if err != nil {
		return err
	}

	it.Code = "ear"
	it.EarInfo = &EarInfo{Class: uint8(class), Level: uint8(level), Name: name}

	return nil
}

func (it *Item) readCode(r *bitReader) error {
	code := make([]byte, 4)

	for i := range code {
		v, err := r.read(8)
		if err != nil {
			return err
		}

		code[i] = byte(v)
	}

	it.Code = string(trimRight(code))

	n, err := r.read(3)
	if err != nil {
		return err
	}

	it.SocketCount = uint8(n)

	return nil
}

func trimRight(b []byte) []byte {
	for len(b) > 0 && (b[len(b)-1] == ' ' || b[len(b)-1] == 0) {
		b = b[:len(b)-1]
	}

	return b
}

// readCString7 reads a NUL terminated string of 7 bit characters.
func readCString7(r *bitReader) (string, error) {
	var s []byte

	for {
		c, err := r.read(7)
		if err != nil {
			return "", err
		}

		if c == 0 {
			return string(s), nil
		}

		s = append(s, byte(c))
	}
}

//nolint:gocyclo // the item format is a long fixed sequence of optional fields
func (it *Item) readExtended(r *bitReader, t *ItemTables) error {
	kind := t.ItemKindOf(it.Code)
	if kind == 0 {
		return fmt.Errorf("%w: %q", ErrUnknownItem, it.Code)
	}

	var err error

	rd := func(n int) uint64 {
		var v uint64

		if err == nil {
			v, err = r.read(n)
		}

		return v
	}

	it.ID = uint32(rd(32))
	it.Level = uint8(rd(7))
	it.Quality = uint8(rd(4))

	if it.HasPicture = rd(1) == 1; it.HasPicture {
		it.Picture = uint8(rd(3))
	}

	if it.HasClassData = rd(1) == 1; it.HasClassData {
		it.ClassData = uint16(rd(11))
	}

	switch it.Quality {
	case QualityLow:
		it.LowQualityID = uint8(rd(3))
	case QualityHigh:
		it.HighQuality = uint8(rd(3))
	case QualityMagic:
		it.MagicPrefix = uint16(rd(11))
		it.MagicSuffix = uint16(rd(11))
	case QualitySet:
		it.SetID = uint16(rd(12))
	case QualityUnique:
		it.UniqueID = uint16(rd(12))
	case QualityRare, QualityCrafted:
		it.RareName1 = uint8(rd(8))
		it.RareName2 = uint8(rd(8))

		for i := range it.RareAffixes {
			if rd(1) == 1 {
				it.RareMask |= 1 << uint(i)
				it.RareAffixes[i] = uint16(rd(11))
			}
		}
	}

	if it.Runeword {
		it.RunewordID = uint16(rd(12))
		it.RunewordUnknown = uint8(rd(4))
	}

	if err != nil {
		return err
	}

	if it.Personalized {
		if it.PersonalName, err = readCString7(r); err != nil {
			return err
		}
	}

	if t.IsTome(it.Code) {
		it.TomeBits = uint8(rd(5))
	}

	it.Timestamp = rd(1) == 1

	if err != nil {
		return err
	}

	if err = it.readBaseStats(r, t, kind); err != nil {
		return err
	}

	if it.Socketed {
		it.TotalSockets = uint8(rd(4))
	}

	if it.Quality == QualitySet {
		it.SetListMask = uint8(rd(5))
	}

	if err != nil {
		return err
	}

	if it.Properties, err = readProperties(r, t); err != nil {
		return err
	}

	for i := 0; i < bits.OnesCount8(it.SetListMask); i++ {
		list, lerr := readProperties(r, t)
		if lerr != nil {
			return lerr
		}

		it.SetProperties = append(it.SetProperties, list)
	}

	if it.Runeword {
		if it.RunewordProperties, err = readProperties(r, t); err != nil {
			return err
		}
	}

	return nil
}

// readBaseStats reads defense, durability and quantity, whose widths come
// from the armorclass, maxdurability and durability rows of ItemStatCost.
func (it *Item) readBaseStats(r *bitReader, t *ItemTables, kind ItemKind) error {
	if kind == KindArmor {
		b, _, add, _, ok := t.StatSaveInfo(statArmorClass)
		if !ok {
			return fmt.Errorf("%w: armorclass", ErrUnknownStat)
		}

		v, err := r.read(b)
		if err != nil {
			return err
		}

		it.Defense = int(v) - add
	}

	if kind == KindArmor || kind == KindWeapon {
		mb, _, _, _, ok := t.StatSaveInfo(statMaxDurability)
		if !ok {
			return fmt.Errorf("%w: maxdurability", ErrUnknownStat)
		}

		v, err := r.read(mb)
		if err != nil {
			return err
		}

		it.MaxDurability = uint16(v)

		// Items without durability (e.g. phase blades) store a max of 0
		// and no current value.
		if v > 0 {
			cb, _, _, _, ok := t.StatSaveInfo(statDurability)
			if !ok {
				return fmt.Errorf("%w: durability", ErrUnknownStat)
			}

			if v, err = r.read(cb); err != nil {
				return err
			}

			it.Durability = uint16(v)
		}
	}

	if t.IsStackable(it.Code) {
		v, err := r.read(quantityBits)
		if err != nil {
			return err
		}

		it.Quantity = uint16(v)
	}

	return nil
}

// readProperties reads 9 bit stat ids and their values until the 0x1FF
// terminator.
func readProperties(r *bitReader, t *ItemTables) ([]Property, error) {
	var props []Property

	for {
		id, err := r.read(9)
		if err != nil {
			return props, err
		}

		if id == propertyEndID {
			return props, nil
		}

		p, err := readProperty(r, t, int(id), true)
		if err != nil {
			return props, err
		}

		props = append(props, p)

		for _, f := range groupFollowers[int(id)] {
			fp, err := readProperty(r, t, f, false)
			if err != nil {
				return props, err
			}

			props = append(props, fp)
		}
	}
}

func readProperty(r *bitReader, t *ItemTables, id int, withParam bool) (Property, error) {
	valBits, paramBits, add, _, ok := t.StatSaveInfo(id)
	if !ok {
		return Property{}, fmt.Errorf("%w: %d", ErrUnknownStat, id)
	}

	p := Property{ID: id, Name: t.stats[id].Name}

	if withParam && paramBits > 0 {
		v, err := r.read(paramBits)
		if err != nil {
			return p, err
		}

		p.Param = uint32(v)
	}

	v, err := r.read(valBits)
	if err != nil {
		return p, err
	}

	p.Value = int64(v) - int64(add)

	return p, nil
}
