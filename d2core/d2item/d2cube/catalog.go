package d2cube

import (
	"sort"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/d2drop"
)

// Tier is the version of a base item: basic (normal), exceptional or elite.
type Tier int

// Item tiers.
const (
	TierNone Tier = iota // items that have no tiers (rings, gems, potions)
	TierBasic
	TierExceptional
	TierElite
)

// Base is what the cube needs to know about a base item (a row of weapons.txt,
// armor.txt or misc.txt).
type Base struct {
	Code         string
	Name         string
	Type, Type2  string
	Types        []string // Type, Type2 and every ancestor type
	Norm         string   // normcode, ubercode and ultracode of the item family
	Uber, Ultra  string
	Tier         Tier
	Stackable    bool
	MaxStack     int
	Weapon       bool
	Armor        bool
	GemSockets   int // most sockets the base item can have
	Level        int
	Spawnable    bool
	Quest        bool
	NoDurability bool
	Durability   int
}

// HasType reports whether the base item is of the type (or a descendant of it).
func (b *Base) HasType(t string) bool {
	for _, x := range b.Types {
		if x == t {
			return true
		}
	}

	return false
}

type typeInfo struct {
	code                           string
	equiv                          []string
	maxSock1, maxSock25, maxSock40 int
}

// Catalog is the item data the cube works with, built from the raw tables.
type Catalog struct {
	bases    map[string]*Base
	order    []string
	types    map[string]*typeInfo
	prefixes []string // magic prefix names by row number (the id of pre=N)
	suffixes []string
}

// NewCatalog reads ItemTypes.txt, weapons.txt, armor.txt and misc.txt.
func NewCatalog(itemTypes, weapons, armor, misc []byte) *Catalog {
	c := &Catalog{bases: map[string]*Base{}, types: map[string]*typeInfo{}}

	it := parseTable(itemTypes)
	for _, r := range it.rows {
		code := it.s(r, "code")
		if code == "" || it.s(r, "itemtype") == "Expansion" {
			continue
		}

		ti := &typeInfo{code: code,
			maxSock1: it.n(r, "maxsock1"), maxSock25: it.n(r, "maxsock25"), maxSock40: it.n(r, "maxsock40")}

		for _, e := range []string{"equiv1", "equiv2"} {
			if v := it.s(r, e); v != "" {
				ti.equiv = append(ti.equiv, v)
			}
		}

		c.types[code] = ti
	}

	c.loadBases(weapons, true)
	c.loadBases(armor, false)
	c.loadMisc(misc)

	for _, b := range c.bases {
		b.Types = c.ancestors(b.Type, b.Type2)
	}

	for _, b := range c.bases {
		c.setTier(b)
	}

	return c
}

func (c *Catalog) loadBases(raw []byte, weapon bool) {
	t := parseTable(raw)

	for _, r := range t.rows {
		code := t.s(r, "code")
		if code == "" || t.s(r, "name") == "Expansion" {
			continue
		}

		b := &Base{
			Code: code, Name: t.s(r, "name"), Type: t.s(r, "type"), Type2: t.s(r, "type2"),
			Norm: t.s(r, "normcode"), Uber: t.s(r, "ubercode"), Ultra: t.s(r, "ultracode"),
			Stackable: t.b(r, "stackable"), MaxStack: t.n(r, "maxstack"),
			Weapon: weapon, Armor: !weapon,
			GemSockets: t.n(r, "gemsockets"), Level: t.n(r, "level"),
			Spawnable: t.b(r, "spawnable"), Quest: t.n(r, "quest") != 0,
			NoDurability: t.b(r, "nodurability"), Durability: t.n(r, "durability"),
		}
		c.add(b)
	}
}

func (c *Catalog) loadMisc(raw []byte) {
	t := parseTable(raw)

	for _, r := range t.rows {
		code := t.s(r, "code")
		if code == "" || t.s(r, "name") == "Expansion" {
			continue
		}

		b := &Base{
			Code: code, Name: t.s(r, "name"), Type: t.s(r, "type"), Type2: t.s(r, "type2"),
			Stackable: t.b(r, "stackable"), MaxStack: t.n(r, "maxstack"),
			GemSockets: t.n(r, "gemsockets"), Level: t.n(r, "level"),
			Spawnable: t.b(r, "spawnable"), Quest: t.n(r, "quest") != 0,
			NoDurability: true,
		}
		c.add(b)
	}
}

func (c *Catalog) add(b *Base) {
	if _, dup := c.bases[b.Code]; !dup {
		c.order = append(c.order, b.Code)
	}

	c.bases[b.Code] = b
}

// ancestors returns the given types and everything they derive from.
func (c *Catalog) ancestors(roots ...string) []string {
	seen := map[string]bool{}

	var out []string

	var walk func(string)

	walk = func(code string) {
		if code == "" || seen[code] {
			return
		}

		seen[code] = true
		out = append(out, code)

		if ti := c.types[code]; ti != nil {
			for _, e := range ti.equiv {
				walk(e)
			}
		}
	}

	for _, r := range roots {
		walk(r)
	}

	return out
}

func (c *Catalog) setTier(b *Base) {
	switch {
	case b.Norm == "" && b.Uber == "" && b.Ultra == "":
		b.Tier = TierNone
	case b.Code == b.Norm:
		b.Tier = TierBasic
	case b.Code == b.Uber:
		b.Tier = TierExceptional
	case b.Code == b.Ultra:
		b.Tier = TierElite
	}
}

// Base returns a base item by code.
func (c *Catalog) Base(code string) (*Base, bool) {
	b, ok := c.bases[code]
	return b, ok
}

// IsType reports whether code is an item type.
func (c *Catalog) IsType(code string) bool {
	_, ok := c.types[code]
	return ok
}

// Upgrade returns the code of the exceptional or elite version of an item.
func (c *Catalog) Upgrade(code string, to Tier) (string, bool) {
	b, ok := c.bases[code]
	if !ok {
		return "", false
	}

	switch to {
	case TierExceptional:
		return b.Uber, b.Uber != "" && c.bases[b.Uber] != nil
	case TierElite:
		return b.Ultra, b.Ultra != "" && c.bases[b.Ultra] != nil
	case TierBasic:
		return b.Norm, b.Norm != "" && c.bases[b.Norm] != nil
	}

	return "", false
}

// MaxSockets is the most sockets a base item can hold at an item level: the
// base item's gemsockets limited by the ItemTypes MaxSock1/25/40 columns
// (item level up to 25, up to 40, above).
func (c *Catalog) MaxSockets(code string, ilvl int) int {
	b, ok := c.bases[code]
	if !ok || b.GemSockets <= 0 {
		return 0
	}

	limit := b.GemSockets

	if ti := c.types[b.Type]; ti != nil {
		cap := ti.maxSock40

		switch {
		case ilvl <= 25:
			cap = ti.maxSock1
		case ilvl <= 40:
			cap = ti.maxSock25
		}

		if cap > 0 && cap < limit {
			limit = cap
		}
	}

	return limit
}

// PickBase picks a random spawnable base item of a type whose level does not
// exceed ilvl (any spawnable one of the type when none is low enough). Used
// for recipe outputs that name an item type instead of an item.
func (c *Catalog) PickBase(rng d2drop.RNG, typeCode string, ilvl int) (string, bool) {
	var low, all []string

	for _, code := range c.order {
		b := c.bases[code]
		if !b.Spawnable || b.Quest || !b.HasType(typeCode) {
			continue
		}

		all = append(all, code)

		if b.Level <= ilvl {
			low = append(low, code)
		}
	}

	pool := low
	if len(pool) == 0 {
		pool = all
	}

	if len(pool) == 0 {
		return "", false
	}

	sort.Strings(pool)

	return pool[int(rng.Roll(int32(len(pool))))], true
}

// LoadAffixes reads MagicPrefix.txt and MagicSuffix.txt so that pre=N and suf=N
// can be resolved to affix names. N is the 0-based row number counting every
// data row (checked against the 1.14b table: Prismatic is pre=331).
func (c *Catalog) LoadAffixes(prefix, suffix []byte) {
	names := func(raw []byte) []string {
		t := parseTable(raw)
		out := make([]string, len(t.rows))

		for i, r := range t.rows {
			out[i] = t.s(r, "name")
		}

		return out
	}

	c.prefixes, c.suffixes = names(prefix), names(suffix)
}

// AffixName returns the name of the prefix or suffix with the given id.
func (c *Catalog) AffixName(prefix bool, id int) (string, bool) {
	list := c.suffixes
	if prefix {
		list = c.prefixes
	}

	if id < 0 || id >= len(list) || list[id] == "" || strings.EqualFold(list[id], "Expansion") {
		return "", false
	}

	return list[id], true
}
