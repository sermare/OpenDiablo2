package diablo2item

import (
	"fmt"
	"math/rand"

	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/d2drop"
)

// Spec is everything needed to build the same item again: the base item, the
// rolled unique/set/affix identity, the item level and the seed the property
// values were rolled with. It is the persistent form of an item (a hero keeps
// its stash and inventory as specs, see d2hero.StoredItem). Building an item
// from a spec is deterministic.
type Spec struct {
	Code     string
	Quality  int // d2drop.Quality the item was created with, 0 if unknown
	ILvl     int
	Seed     int64
	Unique   string
	SetItem  string
	Set      string
	Prefixes []string
	Suffixes []string

	// Rolled is what the item creator rolled; with it the item is rebuilt
	// exactly (affixes, properties, base values, sockets) without rolling.
	Rolled *d2drop.Rolled `json:"rolled,omitempty"`
	// Socketed are the items in the sockets of a creator-made item and Ear the
	// player of an ear. (Runeword below is the Runes.txt name of the runeword
	// of a creator-made item, and the display name for a cube-made one.)
	Socketed []Spec   `json:"socketed,omitempty"`
	Ear      *EarInfo `json:"ear,omitempty"`

	Identified bool
	Ethereal   bool
	Quantity   int // 0 = leave the default
	Durability int // current durability, -1 = leave the default
	// Sockets is the rolled socket count (0 = none), see Item.NumSockets.
	Sockets int
	// MaxDurability is the maximum durability when it differs from the base
	// record's (ethereal items have base/2+1); 0 = leave the default.
	MaxDurability int

	// Horadric Cube and socketing state (items without Rolled). A product of
	// the cube never carries Rolled: it is rebuilt from these fields.
	SocketCodes []string
	Runeword    string
	CubeMods    []ExtraMod
	Crafted     bool

	// Personal is the name Anya's reward personalized the item with.
	Personal string
}

// intn rolls a property value. While an item with a seed is being built the
// factory has a generator seeded from the item (see Item.init), so the values
// of a saved item come out the same every time; otherwise the global generator
// is used, as before.
func (f *ItemFactory) intn(n int) int {
	if f != nil && f.propRand != nil {
		return f.propRand.Intn(n)
	}

	// nolint:gosec // not concerned with crypto-strong randomness
	return rand.Intn(n)
}

// seedProperties makes the property rolls of the next init calls depend on
// the seed only; it returns the function that undoes it.
func (f *ItemFactory) seedProperties(seed int64) func() {
	prev := f.propRand
	// nolint:gosec // not concerned with crypto-strong randomness
	f.propRand = rand.New(rand.NewSource(seed))

	return func() { f.propRand = prev }
}

// Spec returns the item's persistent description.
func (i *Item) Spec() Spec {
	s := Spec{
		Code:       i.CommonCode,
		Quality:    int(i.quality),
		ILvl:       i.itemLevel,
		Seed:       i.Seed,
		Unique:     i.UniqueCode,
		SetItem:    i.SetItemCode,
		Set:        i.SetCode,
		Prefixes:   append([]string(nil), i.PrefixCodes...),
		Suffixes:   append([]string(nil), i.SuffixCodes...),
		Durability: -1,

		Sockets:     i.Sockets,
		SocketCodes: append([]string(nil), i.SocketCodes...),
		Runeword:    i.Runeword,
		CubeMods:    append([]ExtraMod(nil), i.CubeMods...),
		Crafted:     i.Crafted,
		Rolled:      i.rolled,
		Ear:         i.ear,
	}

	if i.rolled != nil {
		s.Runeword = i.runeword
	}

	for _, c := range i.socketed {
		s.Socketed = append(s.Socketed, c.Spec())
	}

	if i.attributes != nil {
		s.Identified = i.attributes.identitified
		s.Ethereal = i.attributes.ethereal
		s.Quantity = i.attributes.currentStackSize
		s.Sockets = i.attributes.numSockets
		s.Personal = i.attributes.personalization

		if i.attributes.numSockets > 0 {
			s.Sockets = i.attributes.numSockets
		}

		if i.attributes.durable {
			s.Durability = i.attributes.currentDurability

			if rec := i.CommonRecord(); rec != nil && i.attributes.durability.max != rec.Durability {
				s.MaxDurability = i.attributes.durability.max
			}
		}
	}

	return s
}

// ItemFromSpec rebuilds an item from its spec.
func (f *ItemFactory) ItemFromSpec(s Spec) (*Item, error) {
	rec := f.asset.Records.Item.All[s.Code]
	if rec == nil {
		return nil, fmt.Errorf("%w: %q", errUnknownItemCode, s.Code)
	}

	if s.Rolled != nil {
		if c, err := f.Creator(); err == nil {
			item := f.itemFromRolled(c, s.Code, rec.Type, s.Rolled)
			item.itemLevel = s.ILvl
			item.runeword, item.ear = s.Runeword, s.Ear

			for _, cs := range s.Socketed {
				if child, err := f.ItemFromSpec(cs); err == nil {
					item.socketed = append(item.socketed, child)
				}
			}

			if item.runeword != "" || item.ear != nil {
				item.rolledProperties(c)
				item.name = item.rolledName()
			}

			f.restoreState(item, s)

			return item, nil
		}
	}

	item := &Item{
		factory:     f,
		CommonCode:  s.Code,
		TypeCode:    rec.Type,
		itemLevel:   s.ILvl,
		Seed:        s.Seed,
		UniqueCode:  s.Unique,
		SetItemCode: s.SetItem,
		SetCode:     s.Set,
		PrefixCodes: append([]string(nil), s.Prefixes...),
		SuffixCodes: append([]string(nil), s.Suffixes...),
		quality:     d2drop.Quality(s.Quality),

		Sockets:     s.Sockets,
		SocketCodes: append([]string(nil), s.SocketCodes...),
		Runeword:    s.Runeword,
		CubeMods:    append([]ExtraMod(nil), s.CubeMods...),
		Crafted:     s.Crafted,
		ear:         s.Ear,
	}
	// nolint:gosec // not concerned with crypto-strong randomness
	item.rand = rand.New(rand.NewSource(s.Seed))
	item.init()

	f.restoreState(item, s)

	return item, nil
}

// restoreState puts the state the spec keeps (identified, ethereal, stack,
// durability) on a rebuilt item.
func (f *ItemFactory) restoreState(item *Item, s Spec) {
	if s.Identified {
		item.Identify()
	}

	item.attributes.ethereal = s.Ethereal
	item.attributes.personalization = s.Personal

	if s.Ethereal {
		item.attributes.applyEtherialBonus() // the bonus is not stored: re-apply on the fresh base values
	}

	if s.Sockets > 0 {
		item.attributes.numSockets = s.Sockets
	}

	if s.MaxDurability > 0 && item.attributes.durable {
		item.attributes.durability.max = s.MaxDurability
	}

	if s.Quantity > 0 {
		item.SetQuantity(s.Quantity)
	}

	if s.Durability >= 0 && item.attributes.durable {
		item.SetDurability(s.Durability)
	}

}
