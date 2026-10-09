package d2drop

import (
	"errors"
	"fmt"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
)

// RequestFlags are the creation request flags of the game (request +0x80).
// A drop sets ForceEthereal / ForceSockets from the "ce"/"cg" modifier rolls
// of the treasure class (Drop.EFlag / Drop.GFlag).
type RequestFlags uint32

// Request flags.
const (
	FlagNoEthereal    RequestFlags = 0x2
	FlagForceEthereal RequestFlags = 0x4
	FlagNoSockets     RequestFlags = 0x8
	FlagForceSockets  RequestFlags = 0x10
)

// Request describes the creation of one item, like the request structure
// ITEMGEN_CreateItemFromRequest (0x556e00) receives.
type Request struct {
	Code string
	ILvl int
	// Quality is the quality to apply; QualityNone lets the game roll it with
	// ITEMGEN_RollExistingItemQuality (shops, gambling).
	Quality    Quality
	Difficulty int // 0 normal, 1 nightmare, 2 hell
	// Version is the game's version word: 100 for Lord of Destruction games,
	// 0 for classic ones. Items created in a game inherit it and it selects
	// the classic or the Lord of Destruction algorithms.
	Version   int
	Expansion bool
	// ForcedID is 1 + the UniqueItems or SetItems row to create (0: pick one).
	ForcedID int
	Flags    RequestFlags
	// GameSeed is the server's generator; the item takes two steps of it, one
	// for its unit generator (base stats) and one for its item generator
	// (affixes, properties, sockets), exactly as ITEMGEN does.
	GameSeed d2rand.Seed
}

// StatWrite is one stat the creation writes on the item, in game order.
// Kind is 'S' for a stat set on the unit (base stats: durability, damage,
// defense...), 'L' / 'M' for stats written to the item's own stat list by the
// properties (value already shifted left by the stat's valshift).
type StatWrite struct {
	Kind  byte
	Stat  int
	Value int
	Param int
}

// Rolled is the result of creating an item: the part of the game's item data
// that is rolled. It mirrors the oracle goldens (testdata/create_*.json).
type Rolled struct {
	Quality Quality
	// Flags are the item flags: 0x10 identified, 0x800 socketed, 0x400000
	// ethereal ...
	Flags uint32
	ILvl  int
	// Prefix, Suffix: ids of up to three magic prefixes and suffixes (1-based
	// rows of the combined affix table, 0 = none). Auto is the automagic row.
	Prefix, Suffix [3]int
	Auto           int
	// RareNames: the two rare / crafted name ids.
	RareNames [2]int
	// Unique is the UniqueItems or SetItems row of a unique or set item.
	Unique int
	// Gfx: the two graphic variant bytes (charm / invfile variants).
	Gfx    [2]int
	Writes []StatWrite
	// UnitSeed and ItemSeed are the generators after creation.
	UnitSeed, ItemSeed d2rand.Seed
}

// Creator creates items. The sub-tables are filled by the loader (tests read
// them from D2_TABLES, the game from d2records).
type Creator struct {
	Items   *ItemTables
	Quality *QualityTables // QualityItems.txt, LowQualityItems.txt (slice A)
	Affixes *AffixTables   // magic / rare / automagic affixes (slice B)
	Uniques *UniqueTables  // UniqueItems.txt, SetItems.txt, Sets.txt (slice C)
	Props   *PropTables    // Properties.txt, ItemStatCost.txt (slice C)
}

// itemState is the item being created.
type itemState struct {
	c    *Creator
	req  Request
	base *BaseItem

	// unit is the unit generator (the game's unit+0x20): base stats.
	// item is the item generator (pItemData+4): affixes, properties,
	// sockets, ethereal.
	unit, item d2rand.Seed

	quality Quality
	flags   uint32
	ilvl    int
	// itemWord is the item's own seed word (pItemData+0x10) that sockets
	// use for their count; the quality fallbacks reset it.
	itemWord uint32

	prefix, suffix [3]int
	auto           int
	rareNames      [2]int
	uniqueRow      int
	gfx            [2]int

	writes []StatWrite
}

func (st *itemState) write(kind byte, stat, value, param int) {
	st.writes = append(st.writes, StatWrite{Kind: kind, Stat: stat, Value: value, Param: param})
}

// ErrNotCreated is returned when the game would refuse to create the item
// (for example a Lord of Destruction item in a classic game).
var ErrNotCreated = errors.New("d2drop: item not created")

// stepInit takes a step of the generator g and returns a new generator
// initialised from the result, like InitSeed(rand(g)) in the game: the new
// generator is {lo, 0x29a}.
func stepInit(g *d2rand.Seed) d2rand.Seed {
	g.Step()

	var s d2rand.Seed

	s.Init(g.Lo)

	return s
}

// Create creates an item. The generators of the item are derived from the
// request's game seed, so creation is reproducible.
func (c *Creator) Create(req Request) (*Rolled, error) {
	base := c.Items.ByCode[req.Code]
	if base == nil {
		return nil, fmt.Errorf("d2drop: unknown item code %q", req.Code)
	}

	if !req.Expansion && base.Version >= 100 {
		return nil, ErrNotCreated
	}

	g := req.GameSeed
	st := &itemState{c: c, req: req, base: base, ilvl: req.ILvl}
	st.unit = stepInit(&g)
	st.item = stepInit(&g)

	if st.ilvl < 1 {
		st.ilvl = 1
	}

	if err := c.createItem(st); err != nil {
		return nil, err
	}

	return &Rolled{
		Quality: st.quality, Flags: st.flags, ILvl: st.ilvl, Prefix: st.prefix, Suffix: st.suffix,
		Auto: st.auto, RareNames: st.rareNames, Unique: st.uniqueRow, Gfx: st.gfx, Writes: st.writes,
		UnitSeed: st.unit, ItemSeed: st.item,
	}, nil
}
