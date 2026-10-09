package diablo2item

import (
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/d2drop"
)

// Loot is everything one treasure class roll puts on the ground: items and
// gold piles, in drop order.
type Loot struct {
	Entries []LootEntry
}

// LootEntry is one dropped thing; either Item or (Gold > 0) a gold pile.
type LootEntry struct {
	Item *Item
	Gold int
}

// Items returns the item entries only.
func (l *Loot) Items() []*Item {
	var out []*Item

	for _, e := range l.Entries {
		if e.Item != nil {
			out = append(out, e.Item)
		}
	}

	return out
}

// DropLoot rolls a treasure class like DropAll and returns items and gold in
// drop order (gold amounts: see DropAll). goldFindPercent is used when the
// options carry no gold find.
func (f *ItemFactory) DropLoot(tcName string, opts DropOptions, goldFindPercent int) (*Loot, error) {
	if opts.GoldFind == 0 {
		opts.GoldFind = goldFindPercent
	}

	entries, err := f.rollEntries(tcName, opts)
	if err != nil {
		return nil, err
	}

	return &Loot{Entries: entries}, nil
}

// TreasureClassLevel returns the Level column of a treasure class (used to
// pick the chest class) and whether the class exists.
func (f *ItemFactory) TreasureClassLevel(name string) (int, bool) {
	tc, ok := f.dropTables().tcs.TreasureClass(name)
	if !ok {
		return 0, false
	}

	return tc.Level, true
}

// QualityName names the quality the item was generated with, as far as the
// item model records it (unique and set rows, affix counts; low/superior are
// not modelled by Item yet and read as "normal").
func (i *Item) QualityName() string {
	if i.rolled != nil {
		return qualityNames[i.quality]
	}

	switch {
	case i.attributes != nil && i.attributes.crafted:
		return "crafted"
	case i.SetItemRecord() != nil:
		return "set"
	case i.UniqueRecord() != nil:
		return "unique"
	}

	switch n := len(i.PrefixRecords()) + len(i.SuffixRecords()); {
	case n > maxAffixesOnMagicItem:
		return "rare"
	case n > 0:
		return "magic"
	}

	return "normal"
}

var qualityNames = map[d2drop.Quality]string{
	d2drop.QualityNone: "normal", d2drop.QualityLow: "low", d2drop.QualityNormal: "normal",
	d2drop.QualitySuperior: "superior", d2drop.QualityMagic: "magic", d2drop.QualitySet: "set",
	d2drop.QualityRare: "rare", d2drop.QualityUnique: "unique", d2drop.QualityCrafted: "crafted",
}

// WorldFlippyFile returns the DC6 (without extension) that animates the item
// falling to the ground: the unique/set row's own file if it has one, else the
// base item's flippyfile.
func (i *Item) WorldFlippyFile() string {
	if u := i.UniqueRecord(); u != nil && u.FlippyFile != "" {
		return u.FlippyFile
	}

	if s := i.SetItemRecord(); s != nil && s.FlippyFile != "" {
		return s.FlippyFile
	}

	return i.CommonRecord().FlippyFile
}

// DropSoundHandle returns the Sounds.txt handle played when the item lands
// (unique/set overrides first, then the base item's dropsound column).
func (i *Item) DropSoundHandle() string {
	if u := i.UniqueRecord(); u != nil && u.DropSound != "" {
		return u.DropSound
	}

	if s := i.SetItemRecord(); s != nil && s.DropSound != "" {
		return s.DropSound
	}

	return i.CommonRecord().DropSound
}
