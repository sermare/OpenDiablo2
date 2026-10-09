package diablo2item

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2asset"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/d2drop"
)

// This file creates items with d2drop.Creator, the port of the game's item
// creation (ITEMGEN_CreateItemFromRequest): quality, affixes, unique and set
// rows, base stats, sockets, ethereal and every property are rolled the way
// Game.exe 1.14b rolls them. Everything that makes an item - drops, vendor
// stock, gambling, the cube, quest rewards and the giveitem command - goes
// through Create.

const (
	// maxItemLevel is the item level of items named explicitly.
	maxItemLevel = 99
	excelDir     = "/data/global/excel/"
	// lodGameVersion and classicGameVersion are Request.Version.
	classicGameVersion = 0
)

var errNoCreator = errors.New("item creator not available")

// assetTables reads the game tables through the asset manager (the first of
// patch_d2, d2exp, d2data that has a file wins, as in the game).
type assetTables struct {
	f *ItemFactory
}

// Table implements d2drop.TableSource.
func (a assetTables) Table(name string) ([]byte, error) {
	return a.f.asset.LoadFile(excelDir + name)
}

// shared is what every factory of a game shares: the creator (its tables are
// large) and the game's item state.
type shared struct {
	once    sync.Once
	creator *d2drop.Creator
	err     error
	game    *d2drop.GameState
}

var (
	sharedMu    sync.Mutex
	sharedByAsm = map[*d2asset.AssetManager]*shared{}
)

func (f *ItemFactory) shared() *shared {
	sharedMu.Lock()
	defer sharedMu.Unlock()

	s := sharedByAsm[f.asset]
	if s == nil {
		s = &shared{game: &d2drop.GameState{}}
		sharedByAsm[f.asset] = s
	}

	return s
}

// Creator returns the item creator, loading its tables on first use. The
// factories of one asset manager share it.
func (f *ItemFactory) Creator() (*d2drop.Creator, error) {
	s := f.shared()

	s.once.Do(func() {
		c, err := d2drop.LoadCreator(assetTables{f})
		if err != nil {
			s.err = fmt.Errorf("%w: %v", errNoCreator, err)

			return
		}

		s.creator = c
	})

	if s.err != nil {
		return nil, s.err
	}

	f.creator = s.creator

	return s.creator, nil
}

// SetCreator installs a creator (tests build one from extracted tables) and a
// new game state.
func (f *ItemFactory) SetCreator(c *d2drop.Creator) {
	s := f.shared()
	s.once.Do(func() {})
	s.creator, s.err = c, nil
	s.game = &d2drop.GameState{}
	f.creator = c
	f.rowByName = nil
}

// Game returns the item state of the game (the one-per-game unique mask and
// the ladder flag the creator reads).
func (f *ItemFactory) Game() *d2drop.GameState {
	return f.shared().game
}

// ResetGame forgets the unique items created so far (a new game).
func (f *ItemFactory) ResetGame() {
	f.shared().game = &d2drop.GameState{}
}

// CreateParams describe one item to create.
type CreateParams struct {
	Code string
	ILvl int
	// Quality is the quality to give; QualityNone lets the game roll it with
	// the item ratio table (vendors, gambling).
	Quality d2drop.Quality
	// Name is the UniqueItems / SetItems name (the "index" column) to create;
	// empty picks a row.
	Name       string
	Difficulty int
	// Classic creates the item as a game without the expansion would.
	Classic bool
	Flags   d2drop.RequestFlags
	// FreshGame creates the item in a game of its own, so the one-per-game
	// rule of unique items does not apply (the giveitem command).
	FreshGame bool
	// Seed seeds the generator the item takes its two generators from.
	Seed uint32
	// Rng, if set, is the shared generator of a drop; the item takes two steps
	// of it (and Seed is ignored).
	Rng *d2rand.Seed
}

// uniqueRowOf finds the row of a unique or set item name.
func (f *ItemFactory) uniqueRowOf(c *d2drop.Creator, name string, set bool) int {
	if c.Uniques == nil || name == "" {
		return -1
	}

	if f.rowByName == nil {
		f.rowByName = map[string]int{}

		for i := range c.Uniques.Uniques {
			f.rowByName["u:"+c.Uniques.Uniques[i].Name] = i
		}

		for i := range c.Uniques.SetItems {
			f.rowByName["s:"+c.Uniques.SetItems[i].Name] = i
		}
	}

	key := "u:"
	if set {
		key = "s:"
	}

	if row, ok := f.rowByName[key+name]; ok {
		return row
	}

	return -1
}

// Create makes an item. Equal parameters (and equal game state) give equal
// items.
func (f *ItemFactory) Create(p CreateParams) (*Item, error) {
	c, err := f.Creator()
	if err != nil {
		return nil, err
	}

	icr := f.asset.Records.Item.All[p.Code]
	if icr == nil {
		return nil, fmt.Errorf("%w: %q", errUnknownItemCode, p.Code)
	}

	req := d2drop.Request{
		Code: p.Code, ILvl: p.ILvl, Quality: p.Quality, Difficulty: p.Difficulty,
		Version: lodVersion, Expansion: !p.Classic, Flags: p.Flags, Game: f.Game(),
	}

	if p.FreshGame {
		req.Game = &d2drop.GameState{Ladder: true}
	}

	if p.Classic {
		req.Version = classicGameVersion
	}

	if p.Name != "" {
		set := p.Quality == d2drop.QualitySet
		if row := f.uniqueRowOf(c, p.Name, set); row >= 0 {
			req.ForcedID = row + 1
		}
	}

	if p.Rng != nil {
		req.GameSeed = *p.Rng
		// the item takes two steps of the shared generator (unit + item generator)
		p.Rng.Step()
		p.Rng.Step()
	} else {
		req.GameSeed = *d2rand.New(p.Seed)
	}

	rolled, err := c.Create(req)
	if err != nil {
		return nil, fmt.Errorf("creating %q: %w", p.Code, err)
	}

	return f.itemFromRolled(c, p.Code, icr.Type, rolled), nil
}

// affixName returns the name of a row of the combined affix table.
func affixName(c *d2drop.Creator, id int) string {
	if c.Affixes == nil || id < 1 || id > len(c.Affixes.Rows) {
		return ""
	}

	return c.Affixes.Rows[id-1].Name
}

// rareName returns the rare name with a 1-based id of the creator's table.
func rareName(c *d2drop.Creator, id int) string {
	if c.Affixes == nil || id < 1 || id > len(c.Affixes.RareNames) {
		return ""
	}

	return c.Affixes.RareNames[id-1].Name
}

func isRareLike(q d2drop.Quality) bool {
	return q == d2drop.QualityRare || q == d2drop.QualityCrafted
}

// itemFromRolled builds the item the creator rolled.
func (f *ItemFactory) itemFromRolled(c *d2drop.Creator, code, typeCode string, r *d2drop.Rolled) *Item {
	item := &Item{
		factory:    f,
		CommonCode: code,
		TypeCode:   typeCode,
		itemLevel:  r.ILvl,
		quality:    r.Quality,
		rolled:     r,
		Seed:       int64(r.ItemSeed.Lo),
	}

	item.fromRolled(c)

	return item
}

// fromRolled fills the item's identity (names of the affixes, unique and set
// rows) and its properties from the rolled data.
func (i *Item) fromRolled(c *d2drop.Creator) {
	r := i.rolled
	f := i.factory

	for _, id := range r.Prefix {
		if n := affixName(c, id); n != "" {
			i.PrefixCodes = append(i.PrefixCodes, n)
		}
	}

	for _, id := range r.Suffix {
		if n := affixName(c, id); n != "" {
			i.SuffixCodes = append(i.SuffixCodes, n)
		}
	}

	if n := affixName(c, r.Auto); n != "" {
		i.AutoCode = n
	}

	switch r.Quality {
	case d2drop.QualityUnique:
		if c.Uniques != nil && r.Unique >= 0 && r.Unique < len(c.Uniques.Uniques) {
			i.UniqueCode = c.Uniques.Uniques[r.Unique].Name
		}
	case d2drop.QualitySet:
		if c.Uniques != nil && r.Unique >= 0 && r.Unique < len(c.Uniques.SetItems) {
			si := c.Uniques.SetItems[r.Unique]
			i.SetItemCode = si.Name
			i.setRow = si.Set + 1

			if si.Set >= 0 && si.Set < len(c.Uniques.Sets) {
				i.SetCode = c.Uniques.Sets[si.Set].Name
			}
		}
	}

	if isRareLike(r.Quality) {
		i.rareName = strings.TrimSpace(f.translateName(rareName(c, r.RareNames[0])) + " " +
			f.translateName(rareName(c, r.RareNames[1])))
	}

	if i.rand == nil {
		i.SetSeed(i.Seed)
	}

	if i.attributes == nil {
		i.attributes = &itemAttributes{}
	}

	i.rolledProperties(c)
	i.updateItemAttributes()
	i.applyRolledBase(c)
}

// translateName returns the string table text of a name, or the key.
func (f *ItemFactory) translateName(key string) string {
	if key == "" {
		return ""
	}

	return f.asset.TranslateString(key)
}

// nextSeed draws the seed of an item that is not part of a drop.
func (f *ItemFactory) nextSeed() uint32 {
	if f.rand == nil {
		f.SetSeed(defaultSeed)
	}

	return uint32(f.rand.Int63())
}
