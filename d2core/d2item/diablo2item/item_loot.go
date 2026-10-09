package diablo2item

import (
	"fmt"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/d2drop"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/d2ground"
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

// DropLoot rolls a treasure class like DropItems but keeps the gold entries,
// turning them into gold piles (see d2ground.GoldAmount). The item rolls are
// identical to DropItems for the same seed up to the first gold entry; gold
// consumes one extra generator step each, so later items differ from DropItems.
func (f *ItemFactory) DropLoot(tcName string, opts DropOptions, goldFindPercent int) (*Loot, error) {
	t := f.dropTables()
	rng := d2rand.New(opts.Seed)

	drops, err := t.dropper.Roll(&d2drop.Context{
		RNG: rng, ILvl: opts.ILvl, UpgradeLevel: opts.UpgradeLevel, Players: opts.Players,
		MagicFind: opts.MagicFind, MaxDrops: opts.MaxDrops,
	}, tcName)
	if err != nil {
		return nil, fmt.Errorf("rolling %q: %w", tcName, err)
	}

	loot := &Loot{}

	for i := range drops {
		d := &drops[i]

		if d.Code == goldItemCode {
			base := d2ground.BaseGold(d.ILvl, rng.Roll)
			loot.Entries = append(loot.Entries, LootEntry{Gold: d2ground.GoldAmount(base, d.Mul, goldFindPercent)})

			continue
		}

		if item := f.itemFromDrop(t, rng, d); item != nil {
			if opts.RollExtras {
				f.rollExtras(t, item, opts.Difficulty)
			}

			loot.Entries = append(loot.Entries, LootEntry{Item: item})
		}
	}

	return loot, nil
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
