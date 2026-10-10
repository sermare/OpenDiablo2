package d2s

import (
	"bytes"
	"errors"
	"fmt"
)

const (
	corpseHeaderLen = 12 // unknown bytes before a corpse's item list
	itemTagLen      = 4  // 'JM' plus a 16-bit count
)

var (
	mercTag  = []byte{'j', 'f'}
	golemTag = []byte{'k', 'f'}
	itemTag  = []byte{'J', 'M'}
)

// ErrNoItemSection is returned when an expected item list tag is missing.
var ErrNoItemSection = errors.New("d2s: expected item list tag")

// Character is a fully parsed save file.
type Character struct {
	Header *Header
	// Body is nil for a brand new character, whose file is only the header.
	Body *Body
	// Items are the equipped, inventory, stash, cube and belt items.
	Items []Item
	// Corpse holds the items on the character's corpse, if it has one.
	Corpse    []Item
	HasCorpse bool
	// CorpseHeader is the 12 unexplained bytes before a corpse's item list.
	CorpseHeader [corpseHeaderLen]byte
	// MercItems are the mercenary's equipped items (expansion saves).
	MercItems []Item
	// Golem is the item an iron golem was made from, for Necromancers.
	Golem *Item
	// Trailing is any data after the last section the parser understands,
	// kept so that Write reproduces the file.
	Trailing []byte
}

// Parse decodes a complete .d2s file. tables is required for any character
// with a body, because stats and items are described by game data.
func Parse(data []byte, tables *ItemTables) (c *Character, err error) {
	err = safely(func() error {
		var perr error
		c, perr = parse(data, tables)

		return perr
	})
	if err != nil {
		return nil, err
	}

	return c, nil
}

func parse(data []byte, tables *ItemTables) (*Character, error) {
	header, err := ParseHeader(data)
	if err != nil {
		return nil, err
	}

	c := &Character{Header: header}
	if header.IsNewCharacter() {
		return c, nil
	}

	if tables == nil {
		return nil, ErrNoTables
	}

	storage := func(id int) (StatStorage, bool) {
		bits, paramBits, ok := tables.CharStatInfo(id)

		return StatStorage{Bits: bits, ParamBits: paramBits}, ok
	}

	if c.Body, err = ParseBody(data, storage); err != nil {
		return nil, err
	}

	pos := c.Body.ItemsOffset

	if c.Items, pos, err = readItemSection(data, pos, tables); err != nil {
		return nil, fmt.Errorf("items: %w", err)
	}

	if pos, err = c.readCorpse(data, pos, tables); err != nil {
		return nil, fmt.Errorf("corpse: %w", err)
	}

	if !header.IsExpansion() {
		c.keepTrailing(data, pos)
		return c, nil
	}

	if pos, err = c.readMercenary(data, pos, tables); err != nil {
		return nil, fmt.Errorf("mercenary: %w", err)
	}

	if header.Class == Necromancer {
		if pos, err = c.readGolem(data, pos, tables); err != nil {
			return nil, fmt.Errorf("golem: %w", err)
		}
	}

	c.keepTrailing(data, pos)

	return c, nil
}

// readItemSection parses the item list at pos and returns the position after it.
func readItemSection(data []byte, pos int, tables *ItemTables) ([]Item, int, error) {
	if pos+itemTagLen > len(data) || !bytes.Equal(data[pos:pos+2], itemTag) {
		return nil, pos, fmt.Errorf("%w at 0x%X", ErrNoItemSection, pos)
	}

	items, n, err := ParseItemList(data[pos:], tables)
	if err != nil {
		return nil, pos, err
	}

	return items, pos + n, nil
}

func (c *Character) readCorpse(data []byte, pos int, tables *ItemTables) (int, error) {
	if pos+itemTagLen > len(data) || !bytes.Equal(data[pos:pos+2], itemTag) {
		return pos, fmt.Errorf("%w at 0x%X", ErrNoItemSection, pos)
	}

	count := int(data[pos+2]) | int(data[pos+3])<<8
	pos += itemTagLen

	if count == 0 {
		return pos, nil
	}

	c.HasCorpse = true

	if pos+corpseHeaderLen > len(data) {
		return pos, ErrTruncated
	}

	copy(c.CorpseHeader[:], data[pos:pos+corpseHeaderLen])

	var err error

	c.Corpse, pos, err = readItemSection(data, pos+corpseHeaderLen, tables)

	return pos, err
}

func (c *Character) readMercenary(data []byte, pos int, tables *ItemTables) (int, error) {
	if pos+2 > len(data) || !bytes.Equal(data[pos:pos+2], mercTag) {
		return pos, fmt.Errorf("%w: mercenary tag at 0x%X", ErrBadSection, pos)
	}

	pos += 2

	if c.Header.Mercenary.ID == 0 {
		return pos, nil
	}

	var err error

	c.MercItems, pos, err = readItemSection(data, pos, tables)

	return pos, err
}

func (c *Character) keepTrailing(data []byte, pos int) {
	if pos < len(data) {
		c.Trailing = append([]byte(nil), data[pos:]...)
	}
}

func (c *Character) readGolem(data []byte, pos int, tables *ItemTables) (int, error) {
	if pos+3 > len(data) || !bytes.Equal(data[pos:pos+2], golemTag) {
		return pos, fmt.Errorf("%w: golem tag at 0x%X", ErrBadSection, pos)
	}

	if data[pos+2] == 0 {
		return pos + 3, nil
	}

	item, n, err := parseItem(data[pos+3:], tables)
	if err != nil {
		return pos, err
	}

	c.Golem = &item

	return pos + 3 + n, nil
}
