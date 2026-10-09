package diablo2item

import (
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2statlist"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/d2drop"
)

// SetTable builds the set bonuses of Sets.txt for the hero's stat list: the
// partial bonuses (2 to 5 pieces worn) and the full bonus, keyed by the
// 1-based row of the set (Item.StatItem's SetID).
func (f *ItemFactory) SetTable() (d2statlist.SetDefs, error) {
	c, err := f.Creator()
	if err != nil {
		return nil, err
	}

	props := func(ws []d2drop.StatWrite) []d2statlist.Prop {
		var out []d2statlist.Prop

		for _, w := range ws {
			if w.Kind == 'S' {
				continue
			}

			out = append(out, d2statlist.Prop{
				ID: w.Stat, Param: w.Param, Value: int64(w.Value >> uint(f.valShift(c, w.Stat))),
			})
		}

		return out
	}

	defs := d2statlist.SetDefs{}

	for set := range c.Uniques.Sets {
		b, ok := c.SetBonuses(set)
		if !ok {
			continue
		}

		def := d2statlist.SetDef{Pieces: b.Pieces, Full: props(b.Full)}
		for _, p := range b.Partial {
			def.Partial = append(def.Partial, props(p))
		}

		defs[set+1] = def
	}

	return defs, nil
}

// SetOfSetItem returns the 1-based Sets.txt row of the set a SetItems.txt row
// belongs to, 0 if there is none.
func (f *ItemFactory) SetOfSetItem(row int) int {
	c, err := f.Creator()
	if err != nil || row < 0 || row >= len(c.Uniques.SetItems) {
		return 0
	}

	return c.Uniques.SetItems[row].Set + 1
}
