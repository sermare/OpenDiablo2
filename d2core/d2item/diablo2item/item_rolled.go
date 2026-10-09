package diablo2item

import (
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2statlist"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/d2drop"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2stats"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2ui"
)

// ItemStatCost ids the creation writes on the unit (base values of the item).
const (
	statIDBlock     = 0x14
	statIDMinDam    = 0x15
	statIDMaxDam    = 0x16
	statIDSecMinDam = 0x17
	statIDSecMaxDam = 0x18
	statIDDefense   = 0x1f
	statIDQuantity  = 0x46
	statIDDurCur    = 0x48
	statIDDurMax    = 0x49
	statIDNoDurab   = 0x98
	statIDThrowMin  = 0x9f
	statIDThrowMax  = 0xa0
	statIDSockets   = 0xc2
)

// The selectors of the set bonus lists an item carries (d2drop.StatWrite.List).
const (
	setListFirst = 0xa5
	setListLast  = 0xa9
)

const (
	flagIdentifiedBit uint32 = 0x10
	flagSocketedBit   uint32 = 0x800
	flagEtherealBit   uint32 = 0x400000
)

// Rolled returns what the creator rolled for the item, nil for an item that
// was not made with it.
func (i *Item) Rolled() *d2drop.Rolled {
	return i.rolled
}

// statByID returns the ItemStatCost record name of a stat id.
func (f *ItemFactory) statByID(id int) string {
	if f.statNames == nil {
		f.statNames = map[int]string{}

		for name, rec := range f.asset.Records.Item.Stats {
			f.statNames[rec.Index] = name
		}
	}

	return f.statNames[id]
}

// statArgs orders a (value, param) pair of the creation the way the stat
// factory wants the values of a stat with the given description function.
func statArgs(descFn, value, param int) []float64 {
	v, p := float64(value), float64(param)

	const (
		skillShift = 6
		skillMask  = 0x3f
		tabBits    = 3
		tabMask    = 7
	)

	switch descFn {
	case 13: // +n to <class> skill levels
		return []float64{v, p}
	case 14: // +n to <tab> skills (<class> only)
		return []float64{v, float64(param >> tabBits), float64(param & tabMask)}
	case 15: // chance to cast level L skill S on event: value = chance
		return []float64{v, float64(param & skillMask), float64(param >> skillShift)}
	case 16: // level L aura: value = level, param = skill
		return []float64{v, p}
	case 22, 23: // n% against / reanimate as <monster>
		return []float64{v, p}
	case 24: // charges: param = skill and level, value = current and max charges
		return []float64{float64(param & skillMask), float64(param >> skillShift), float64(value & 0xff), float64(value >> 8 & 0xff)}
	case 27: // +n to <skill> (<class> only)
		return []float64{v, p, -1}
	case 28: // +n to <skill>
		return []float64{v, p}
	}

	return []float64{v}
}

// shifted undoes the fixed point shift of a stat written by a property.
func (f *ItemFactory) valShift(c *d2drop.Creator, id int) int {
	if c.Props != nil && id >= 0 && id < len(c.Props.ValShift) {
		return c.Props.ValShift[id]
	}

	return 0
}

// statOfWrite makes the stat of a property write; nil when the stat has no
// description in the tables.
func (f *ItemFactory) statOfWrite(c *d2drop.Creator, w d2drop.StatWrite) d2stats.Stat {
	name := f.statByID(w.Stat)
	if name == "" || f.stat == nil {
		return nil
	}

	rec := f.asset.Records.Item.Stats[name]
	value := w.Value >> uint(f.valShift(c, w.Stat))

	return f.stat.NewStat(name, statArgs(rec.DescFnID, value, w.Param)...)
}

// poolOfList says which pool the stats of a list belong to for the item.
func (i *Item) poolOfList(list int) PropertyPool {
	if list >= setListFirst && list <= setListLast {
		return PropertyPoolSet
	}

	switch i.quality {
	case d2drop.QualityUnique:
		return PropertyPoolUnique
	case d2drop.QualitySet:
		return PropertyPoolSetItem
	default:
		return PropertyPoolPrefix
	}
}

// rolledProperties turns the stat writes of the creation into the item's
// properties (the stats of its tooltip).
func (i *Item) rolledProperties(c *d2drop.Creator) {
	i.properties = make(map[PropertyPool][]*Property)

	for _, w := range i.rolled.Writes {
		if w.Kind == 'S' {
			continue
		}

		st := i.factory.statOfWrite(c, w)
		if st == nil {
			continue
		}

		pool := i.poolOfList(w.List)
		i.properties[pool] = append(i.properties[pool], &Property{
			factory: i.factory, stats: []d2stats.Stat{st},
		})
	}
}

// applyRolledBase replaces the base attributes the record gives with the
// values the creation rolled: defense, durability, damage, quantity, sockets
// and the item flags.
func (i *Item) applyRolledBase(c *d2drop.Creator) {
	r := i.rolled
	a := i.attributes

	last := map[int]int{}
	seen := map[int]bool{}

	for _, w := range r.Writes {
		if w.Kind == 'S' || w.Stat == statIDSockets {
			last[w.Stat], seen[w.Stat] = w.Value, true
		}
	}

	if seen[statIDDefense] {
		a.defense = last[statIDDefense]
	}

	if seen[statIDMinDam] || seen[statIDMaxDam] {
		a.damageOneHand.min, a.damageOneHand.max = last[statIDMinDam], last[statIDMaxDam]
	}

	if seen[statIDSecMinDam] || seen[statIDSecMaxDam] {
		a.damageTwoHand.min, a.damageTwoHand.max = last[statIDSecMinDam], last[statIDSecMaxDam]
	}

	if seen[statIDThrowMin] || seen[statIDThrowMax] {
		a.damageMissile.min, a.damageMissile.max = last[statIDThrowMin], last[statIDThrowMax]
	}

	if seen[statIDQuantity] {
		a.currentStackSize = last[statIDQuantity]
	}

	if seen[statIDDurMax] {
		a.durability.max = last[statIDDurMax]
		a.durability.min = last[statIDDurMax]
		a.currentDurability = last[statIDDurMax]
	}

	if seen[statIDDurCur] {
		a.currentDurability = last[statIDDurCur]
	}

	if seen[statIDNoDurab] && last[statIDNoDurab] != 0 {
		a.indestructable = true
	}

	a.durable = !a.indestructable && a.durability.max > 0

	if seen[statIDSockets] {
		a.numSockets = last[statIDSockets]
	}

	a.identitified = r.Flags&flagIdentifiedBit != 0
	a.ethereal = r.Flags&flagEtherealBit != 0
	a.quality = int(r.Quality)
	a.baseItemLevel = r.ILvl
	a.crafted = r.Quality == d2drop.QualityCrafted
}

// NumSockets returns the number of sockets of the item.
func (i *Item) NumSockets() int {
	if i.attributes == nil {
		return 0
	}

	return i.attributes.numSockets
}

// IsEthereal reports whether the item is ethereal.
func (i *Item) IsEthereal() bool {
	return i.attributes != nil && i.attributes.ethereal
}

// SetBonusTiers is the number of set bonus lists of a set item.
const SetBonusTiers = setListLast - setListFirst + 1

// StatItem converts the item to what the hero's stat list computes with:
// base defense, the item's own properties (shifted back to stat units) and
// the set bonus lists with the set the item belongs to (SetID is the 1-based
// row of Sets.txt). The set bonuses are active only while enough pieces of
// the set are worn (d2statlist.Compute counts them).
func (i *Item) StatItem() d2statlist.Item {
	it := d2statlist.Item{Code: strings.TrimSpace(i.CommonCode), Ethereal: i.IsEthereal(), SetID: i.setRow}

	if i.attributes != nil {
		it.Defense = i.attributes.defense
	}

	if i.rolled == nil {
		return it
	}

	lists := make([][]d2statlist.Prop, SetBonusTiers)
	used := 0

	for _, w := range i.rolled.Writes {
		if w.Kind == 'S' {
			continue
		}

		p := d2statlist.Prop{ID: w.Stat, Param: w.Param, Value: int64(w.Value >> uint(i.factory.valShift(i.factory.creator, w.Stat)))}

		if w.List >= setListFirst && w.List <= setListLast {
			tier := w.List - setListFirst
			lists[tier] = append(lists[tier], p)

			if tier+1 > used {
				used = tier + 1
			}

			continue
		}

		if w.List == d2drop.ListRuneword {
			it.RunewordProps = append(it.RunewordProps, p)

			continue
		}

		it.Props = append(it.Props, p)
	}

	it.Runeword = i.runeword != ""
	it.Sockets = i.socketedStatItems()

	if used > 0 {
		it.SetLists = lists[:used]
	}

	return it
}

// rolledName is the name of an item made with the creator: the unique or set
// name, the rare name pair, or the magic affixes around the base name.
func (i *Item) rolledName() string {
	f := i.factory

	if i.ear != nil {
		return i.ear.Label()
	}

	if i.runeword != "" {
		return f.runewordName(i.runeword)
	}

	switch {
	case i.SetItemRecord() != nil:
		return f.asset.TranslateString(i.SetItemRecord().SetItemKey)
	case i.UniqueRecord() != nil:
		return f.asset.TranslateString(i.UniqueRecord().Name)
	}

	base := f.asset.TranslateString(i.CommonRecord().NameString)

	switch i.quality {
	case d2drop.QualityRare, d2drop.QualityCrafted:
		if i.rareName != "" {
			return strings.Title(i.rareName) + "\n" + base
		}
	case d2drop.QualityMagic:
		name := base

		if len(i.PrefixCodes) > 0 {
			name = f.translateName(i.PrefixCodes[0]) + " " + name
		}

		if len(i.SuffixCodes) > 0 {
			name += " " + f.translateName(i.SuffixCodes[0])
		}

		return name
	case d2drop.QualitySuperior:
		return f.translateName("Superior") + " " + base
	}

	return base
}

// rolledColor is the color of the item's name by quality.
func (i *Item) rolledColor() d2ui.ColorToken {
	switch {
	case i.runeword != "":
		return d2ui.ColorTokenUniqueItem
	}

	switch i.quality {
	case d2drop.QualityMagic:
		return d2ui.ColorTokenMagicItem
	case d2drop.QualityRare:
		return d2ui.ColorTokenRareItem
	case d2drop.QualitySet:
		return d2ui.ColorTokenSetItem
	case d2drop.QualityUnique:
		return d2ui.ColorTokenUniqueItem
	case d2drop.QualityCrafted:
		return d2ui.ColorTokenCraftedItem
	}

	if i.NumSockets() > 0 {
		return d2ui.ColorTokenSocketedItem
	}

	return d2ui.ColorTokenNormalItem
}
