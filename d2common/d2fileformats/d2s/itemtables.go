package d2s

import (
	"bytes"
	"encoding/binary"
	"encoding/csv"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Layout of one row of the compiled itemstatcost.bin that Game.exe 1.14b
// loads (0x144 bytes, indexed by stat id, preceded by a u32 row count).
// The offsets were cross-checked against the full ItemStatCost.txt of
// patch_d2.mpq (all 359 rows match) and against Game.exe's stats parser,
// which reads the character-section width from row+10 and the param width
// from row+11.
const (
	statRowSize = 0x144

	statRowFlags     = 4  // bit 1: Signed
	statRowCharBits  = 10 // CSvBits: width in the character stats section
	statRowCharParam = 11 // CSvParam
	statRowSaveBits  = 25 // Save Bits: width in an item property
	statRowSaveAdd   = 28 // Save Add, int32
	statRowParamBits = 36 // Save Param Bits
	statRowEncode    = 48 // Encode
)

// Errors returned by NewItemTables.
var (
	ErrBadStatTable = errors.New("d2s: malformed ItemStatCost table")
	ErrBadItemTable = errors.New("d2s: malformed item table")
)

// StatInfo describes how one stat is stored in a save file.
type StatInfo struct {
	ID   int
	Name string // only available when loaded from the .txt form

	// Item property storage (Save Bits / Save Add / Save Param Bits).
	SaveBits      int
	SaveAdd       int
	SaveParamBits int
	Signed        bool
	Encode        int

	// Character stats section storage (CSvBits / CSvParam). Zero when the
	// stat is not stored in the character section.
	CharBits  int
	CharParam int
}

// ItemKind says which data file an item code comes from.
type ItemKind uint8

// The three item files.
const (
	KindArmor ItemKind = iota + 1 // armor.txt (includes shields)
	KindWeapon
	KindMisc
)

type itemInfo struct {
	kind         ItemKind
	stackable    bool
	compact      bool // compactsave: saved without extended data (a "simple" item)
	noDurability bool
	itemType     string
}

// ItemTables carries the game data needed to decode items and stats.
type ItemTables struct {
	stats    map[int]StatInfo
	items    map[string]itemInfo
	typeDeps map[string][]string // item type code -> parent type codes
}

// NewItemTables builds the lookup tables from the raw bytes of
// ItemStatCost, armor.txt, weapons.txt, misc.txt and ItemTypes.txt.
//
// itemStatCost may be either the tab separated ItemStatCost.txt (full,
// 1.10+ form with the "Save Bits" columns) or the compiled
// itemstatcost.bin from the game archives; the form is detected from the
// data. The bin form is the one the game itself uses.
func NewItemTables(itemStatCost, armor, weapons, misc, itemTypes []byte) (*ItemTables, error) {
	t := &ItemTables{
		stats:    make(map[int]StatInfo),
		items:    make(map[string]itemInfo),
		typeDeps: make(map[string][]string),
	}

	var err error

	if bytes.HasPrefix(itemStatCost, []byte("Stat")) {
		err = t.loadStatText(itemStatCost)
	} else {
		err = t.loadStatBin(itemStatCost)
	}

	if err != nil {
		return nil, err
	}

	if err = t.loadItemTypes(itemTypes); err != nil {
		return nil, err
	}

	for _, f := range []struct {
		data []byte
		kind ItemKind
	}{{armor, KindArmor}, {weapons, KindWeapon}, {misc, KindMisc}} {
		if err = t.loadItems(f.data, f.kind); err != nil {
			return nil, err
		}
	}

	return t, nil
}

// StatSaveInfo returns how stat id is stored in an item property list:
// the value width in bits, the width of the parameter that precedes it,
// the amount subtracted from the stored value (Save Add) and whether the
// stat is signed. ok is false for unknown stats or stats that are not saved.
func (t *ItemTables) StatSaveInfo(id int) (bits, paramBits, add int, signed, ok bool) {
	s, found := t.stats[id]
	if !found || s.SaveBits == 0 {
		return 0, 0, 0, false, false
	}

	return s.SaveBits, s.SaveParamBits, s.SaveAdd, s.Signed, true
}

// CharStatInfo returns how stat id is stored in the character stats ('gf')
// section of version > 0x5E saves: the width of the value and of the
// parameter that precedes it (CSvBits / CSvParam). ok is false for stats
// that have no character-section width.
func (t *ItemTables) CharStatInfo(id int) (bits, paramBits int, ok bool) {
	s, found := t.stats[id]
	if !found || s.CharBits == 0 {
		return 0, 0, false
	}

	return s.CharBits, s.CharParam, true
}

// Stat returns the full storage description of a stat.
func (t *ItemTables) Stat(id int) (StatInfo, bool) {
	s, ok := t.stats[id]
	return s, ok
}

// ItemKindOf reports which file the item code belongs to, or 0 if unknown.
func (t *ItemTables) ItemKindOf(code string) ItemKind {
	return t.items[code].kind
}

// IsStackable reports whether items of this code carry a quantity field.
func (t *ItemTables) IsStackable(code string) bool {
	return t.items[code].stackable
}

// IsCompact reports whether items of this code are saved without extended
// data (the compactsave column: potions, scrolls, gems, runes, keys, ...).
func (t *ItemTables) IsCompact(code string) bool {
	return t.items[code].compact
}

// IsTome reports whether the item code is of item type "book" (or derived
// from it), which carries an extra 5 bit field.
func (t *ItemTables) IsTome(code string) bool {
	it, ok := t.items[code]
	if !ok {
		return false
	}

	return t.typeIs(it.itemType, "book", 0)
}

func (t *ItemTables) typeIs(have, want string, depth int) bool {
	if have == want {
		return true
	}

	if depth > 16 {
		return false
	}

	for _, p := range t.typeDeps[have] {
		if t.typeIs(p, want, depth+1) {
			return true
		}
	}

	return false
}

func (t *ItemTables) loadStatBin(data []byte) error {
	if len(data) < 4 {
		return fmt.Errorf("%w: too short", ErrBadStatTable)
	}

	n := int(binary.LittleEndian.Uint32(data))
	if n <= 0 || len(data) != 4+n*statRowSize {
		return fmt.Errorf("%w: size %d does not match %d rows", ErrBadStatTable, len(data), n)
	}

	for id := 0; id < n; id++ {
		row := data[4+id*statRowSize : 4+(id+1)*statRowSize]
		t.stats[id] = StatInfo{
			ID:            id,
			SaveBits:      int(row[statRowSaveBits]),
			SaveAdd:       int(int32(binary.LittleEndian.Uint32(row[statRowSaveAdd:]))),
			SaveParamBits: int(row[statRowParamBits]),
			Signed:        row[statRowFlags]&2 != 0,
			Encode:        int(row[statRowEncode]),
			CharBits:      int(row[statRowCharBits]),
			CharParam:     int(row[statRowCharParam]),
		}
	}

	return nil
}

func (t *ItemTables) loadStatText(data []byte) error {
	rows, col, err := readTable(data)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrBadStatTable, err)
	}

	for _, need := range []string{"stat", "id", "save bits", "save add", "save param bits"} {
		if _, ok := col[need]; !ok {
			return fmt.Errorf("%w: missing column %q", ErrBadStatTable, need)
		}
	}

	for _, r := range rows {
		idStr := cell(r, col, "id")
		if idStr == "" {
			continue
		}

		id, err := strconv.Atoi(idStr)
		if err != nil {
			return fmt.Errorf("%w: bad id %q", ErrBadStatTable, idStr)
		}

		t.stats[id] = StatInfo{
			ID:            id,
			Name:          cell(r, col, "stat"),
			SaveBits:      atoi(cell(r, col, "save bits")),
			SaveAdd:       atoi(cell(r, col, "save add")),
			SaveParamBits: atoi(cell(r, col, "save param bits")),
			Signed:        atoi(cell(r, col, "signed")) != 0,
			Encode:        atoi(cell(r, col, "encode")),
			CharBits:      atoi(cell(r, col, "csvbits")),
			CharParam:     atoi(cell(r, col, "csvparam")),
		}
	}

	return nil
}

func (t *ItemTables) loadItemTypes(data []byte) error {
	rows, col, err := readTable(data)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrBadItemTable, err)
	}

	if _, ok := col["code"]; !ok {
		return fmt.Errorf("%w: ItemTypes has no Code column", ErrBadItemTable)
	}

	for _, r := range rows {
		code := cell(r, col, "code")
		if code == "" {
			continue
		}

		for _, k := range []string{"equiv1", "equiv2"} {
			if p := cell(r, col, k); p != "" {
				t.typeDeps[code] = append(t.typeDeps[code], p)
			}
		}
	}

	return nil
}

func (t *ItemTables) loadItems(data []byte, kind ItemKind) error {
	rows, col, err := readTable(data)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrBadItemTable, err)
	}

	if _, ok := col["code"]; !ok {
		return fmt.Errorf("%w: no code column", ErrBadItemTable)
	}

	for _, r := range rows {
		code := cell(r, col, "code")
		if code == "" {
			continue
		}

		t.items[code] = itemInfo{
			kind:         kind,
			stackable:    atoi(cell(r, col, "stackable")) != 0,
			compact:      atoi(cell(r, col, "compactsave")) != 0,
			noDurability: atoi(cell(r, col, "nodurability")) != 0,
			itemType:     cell(r, col, "type"),
		}
	}

	return nil
}

// readTable parses a tab separated game data file. Column names are
// lower-cased; the first occurrence of a duplicated name wins.
func readTable(data []byte) (rows [][]string, col map[string]int, err error) {
	r := csv.NewReader(bytes.NewReader(data))
	r.Comma = '\t'
	r.FieldsPerRecord = -1
	r.LazyQuotes = true

	all, err := r.ReadAll()
	if err != nil {
		return nil, nil, err
	}

	if len(all) == 0 {
		return nil, nil, errors.New("empty table")
	}

	col = make(map[string]int, len(all[0]))

	for i, h := range all[0] {
		h = strings.ToLower(strings.TrimSpace(h))
		if _, dup := col[h]; !dup {
			col[h] = i
		}
	}

	return all[1:], col, nil
}

func cell(row []string, col map[string]int, name string) string {
	i, ok := col[name]
	if !ok || i >= len(row) {
		return ""
	}

	return strings.TrimSpace(row[i])
}

func atoi(s string) int {
	v, _ := strconv.Atoi(s)
	return v
}
