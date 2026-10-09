package d2cube

import (
	"fmt"

	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/d2drop"
)

// Product is one result of a transmute.
type Product struct {
	Item Item

	// Source is the cube item index a "useitem" result was made from, -1 for a
	// new item.
	Source int

	// Roll says the game must roll the item's identity for Item.Quality: the
	// magic/rare/set/unique affixes (or the crafted ones) of Item.Code at
	// Item.ILvl, seeded with Item.Seed. Forced affixes (Item.Prefixes and
	// Suffixes, set from pre=/suf=) replace the rolled ones of their kind.
	Roll bool

	// CraftAffixes is, for crafted items, how many random magic affixes the
	// rolled item keeps besides the recipe's mods.
	CraftAffixes int

	// Portal is set for the named portal results ("Cow Portal", ...): no item
	// is created, the game opens that portal.
	Portal string
}

// Result is the outcome of a transmute.
type Result struct {
	Recipe *Recipe
	// Consumed are the cube item indexes that are used up. A "useitem"
	// product replaces its Source, which is consumed too.
	Consumed []int
	Products []Product
}

// OutputLevel is the item level of a recipe result. UNVERIFIED (table
// semantics): lvl + plvl% of the player level + ilvl% of the main input's item
// level; when all three are blank the main input's level is kept.
func OutputLevel(o *Output, playerLevel, mainILvl int) int {
	lvl := o.Level + o.PLevel*playerLevel/100 + o.ILevel*mainILvl/100
	if o.Level == 0 && o.PLevel == 0 && o.ILevel == 0 {
		lvl = mainILvl
		if lvl <= 0 {
			lvl = playerLevel
		}
	}

	switch {
	case lvl < 1:
		return 1
	case lvl > 99: //nolint:gomnd // the item level cap
		return 99
	}

	return lvl
}

// CraftedAffixCount is the number of random magic affixes of a crafted item
// (ITEMGEN_RollCraftedAffixes, notes itemgen.md 1.7): at least 1, 2 above item
// level 30, 3 above 50, 4 above 70; the roll rand(5) can raise it.
func CraftedAffixCount(rng d2drop.RNG, ilvl int) int {
	n := 1

	switch {
	case ilvl > 70: //nolint:gomnd // thresholds from the binary
		n = 4
	case ilvl > 50: //nolint:gomnd
		n = 3
	case ilvl > 30: //nolint:gomnd
		n = 2
	}

	if r := int(rng.Roll(5)); r > n { //nolint:gomnd
		n = r
	}

	return n
}

// MainItem returns the cube item a "useitem" or "usetype" result is based on:
// the first consumed item that is not an ingredient (gem, rune, jewel, scroll,
// potion), or the first consumed item.
func (cat *Catalog) MainItem(m *Match, items []Item) int {
	first := -1

	for _, list := range m.Assigned {
		for _, i := range list {
			if first < 0 {
				first = i
			}

			if !cat.IsIngredient(&items[i]) {
				return i
			}
		}
	}

	return first
}

// Execute runs a matched recipe.
func (cat *Catalog) Execute(m *Match, ctx *Context, items []Item, rng d2drop.RNG) (*Result, error) {
	res := &Result{Recipe: m.Recipe}

	for _, list := range m.Assigned {
		res.Consumed = append(res.Consumed, list...)
	}

	main := cat.MainItem(m, items)
	if main < 0 {
		return nil, fmt.Errorf("recipe %d has no inputs", m.Recipe.Row)
	}

	for k := range m.Recipe.Outputs {
		p, err := cat.makeProduct(&m.Recipe.Outputs[k], ctx, items, main, rng)
		if err != nil {
			return nil, fmt.Errorf("recipe %d %q output %d: %w", m.Recipe.Row, m.Recipe.Description, k+1, err)
		}

		res.Products = append(res.Products, p)
	}

	return res, nil
}

func newSeed(rng d2drop.RNG) int64 { return int64(rng.Roll(1<<30)) + 1 }

func (cat *Catalog) makeProduct(o *Output, ctx *Context, items []Item, main int, rng d2drop.RNG) (Product, error) {
	src := &items[main]
	p := Product{Source: -1}

	if o.Special != "" {
		p.Portal = o.Special
		return p, nil
	}

	ilvl := OutputLevel(o, ctx.PlayerLevel, src.ILvl)

	switch {
	case o.UseItem:
		p.Item = *src
		p.Source = main

		if o.Level != 0 || o.PLevel != 0 || o.ILevel != 0 {
			p.Item.ILvl = ilvl
		}

		if err := cat.modifyInPlace(o, &p.Item); err != nil {
			return p, err
		}
	case o.UseType:
		p.Item = Item{Code: src.Code, ILvl: ilvl, Seed: newSeed(rng), Quality: o.Quality}
	case o.Code != "":
		p.Item = Item{Code: o.Code, ILvl: ilvl, Seed: newSeed(rng), Quality: o.Quality}
	case o.Type != "":
		code, ok := cat.PickBase(rng, o.Type, ilvl)
		if !ok {
			return p, fmt.Errorf("no base item of type %q", o.Type)
		}

		p.Item = Item{Code: code, ILvl: ilvl, Seed: newSeed(rng), Quality: o.Quality}
	default:
		return p, fmt.Errorf("output %q names nothing", o.Raw)
	}

	if !o.UseItem {
		p.Item.Ethereal = o.Ethereal
		p.Item.Identified = o.Quality == 0 || o.Quality == d2drop.QualityNormal
		p.Roll = o.Quality >= d2drop.QualityMagic

		if o.Qty > 0 {
			p.Item.Quantity = o.Qty
		}

		cat.forceAffixes(o, &p)

		if o.Quality == d2drop.QualityCrafted {
			p.CraftAffixes = CraftedAffixCount(rng, ilvl)
			p.Item.Crafted = true
		}
	}

	cat.applyMods(o, &p.Item, rng)

	if o.Sockets > 0 {
		p.Item.Sockets = minInt(o.Sockets, cat.MaxSockets(p.Item.Code, p.Item.ILvl))
	}

	return p, nil
}

// modifyInPlace applies a "useitem" result's flags to a copy of the input.
func (cat *Catalog) modifyInPlace(o *Output, it *Item) error {
	if o.Tier != TierNone && o.Upgrade {
		code, ok := cat.Upgrade(it.Code, o.Tier)
		if !ok {
			return fmt.Errorf("%s has no tier %d version", it.Code, o.Tier)
		}

		it.Code = code
	}

	if o.Quality != 0 {
		it.Quality = o.Quality
	}

	if o.Repair {
		it.Repaired = true
	}

	if o.Recharge {
		it.Recharged = true
	}

	if o.Unsocket {
		it.Socketed = nil
		it.Runeword = ""
	}

	if o.Ethereal {
		it.Ethereal = true
	}

	return nil
}

// forceAffixes resolves pre=N and suf=N.
func (cat *Catalog) forceAffixes(o *Output, p *Product) {
	for _, id := range o.Prefix {
		if name, ok := cat.AffixName(true, id); ok {
			p.Item.Prefixes = append(p.Item.Prefixes, name)
		}
	}

	for _, id := range o.Suffix {
		if name, ok := cat.AffixName(false, id); ok {
			p.Item.Suffixes = append(p.Item.Suffixes, name)
		}
	}
}

// applyMods rolls the property lines of a result. The pseudo property "sock"
// sets the number of sockets (capped by what the base item can hold); the
// others are kept for the game to turn into item properties.
func (cat *Catalog) applyMods(o *Output, it *Item, rng d2drop.RNG) {
	for _, m := range o.Mods {
		if m.Chance < 100 && int(rng.Roll(100)) >= m.Chance {
			continue
		}

		v := m.Min
		if m.Max > m.Min {
			v = m.Min + int(rng.Roll(int32(m.Max-m.Min+1)))
		}

		if m.Code == "sock" {
			it.Sockets = minInt(v, cat.MaxSockets(it.Code, it.ILvl))
			continue
		}

		it.Mods = append(it.Mods, RolledMod{Code: m.Code, Param: m.Param, Min: m.Min, Max: m.Max, Value: v})
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}

	return b
}
