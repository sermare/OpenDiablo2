package diablo2item

import (
	"fmt"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2statlist"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/d2drop"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2ui"
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

// SetInfo describes a set for the tooltip of its pieces: the piece names, the
// bonus texts of the partial tiers (from 2 to 5 pieces) and of the full set.
type SetInfo struct {
	Name    string
	Pieces  []string
	Partial [4][]string // Partial[i] is the bonus from i+2 pieces worn
	Full    []string
}

// SetInfo returns the set with the 1-based Sets.txt row (Item.SetRow).
func (f *ItemFactory) SetInfo(row int) (SetInfo, bool) {
	c, err := f.Creator()
	if err != nil || c.Uniques == nil || row < 1 || row > len(c.Uniques.Sets) {
		return SetInfo{}, false
	}

	def := &c.Uniques.Sets[row-1]
	info := SetInfo{Name: def.Name}

	for _, si := range def.Items {
		if si >= 0 && si < len(c.Uniques.SetItems) {
			info.Pieces = append(info.Pieces, c.Uniques.SetItems[si].Name)
		}
	}

	b, ok := c.SetBonuses(row - 1)
	if !ok {
		return info, true
	}

	text := func(ws []d2drop.StatWrite) []string {
		var out []string

		for _, w := range ws {
			if w.Kind == 'S' {
				continue
			}

			if st := f.statOfWrite(c, w); st != nil && st.String() != "" {
				out = append(out, st.String())
			} else {
				// no description engine (or no text for the stat): the stat's table name or id
				name := f.statByID(w.Stat)
				if name == "" {
					name = fmt.Sprintf("stat %d", w.Stat)
				}

				out = append(out, fmt.Sprintf("%+d %s", w.Value>>uint(f.valShift(c, w.Stat)), name))
			}
		}

		return out
	}

	for i := range b.Partial {
		info.Partial[i] = text(b.Partial[i])
	}

	info.Full = text(b.Full)

	return info, true
}

// SetRow is the 1-based Sets.txt row of the set a set item belongs to, 0 for
// any other item.
func (i *Item) SetRow() int { return i.setRow }

// SetPieceName is the SetItems.txt name of a set item ("" for other items).
func (i *Item) SetPieceName() string {
	if rec := i.SetItemRecord(); rec != nil {
		return rec.SetItemKey
	}

	return ""
}

// SetTooltipLines is the set part of a set item's tooltip: the set name, its
// pieces (green when worn, red when not), the partial bonuses (green while
// active, grey otherwise) and the full set bonus. worn tells which piece names
// the hero wears.
func SetTooltipLines(info SetInfo, worn map[string]bool) []string {
	n := 0

	for _, p := range info.Pieces {
		if worn[p] {
			n++
		}
	}

	lines := []string{d2ui.ColorTokenize(info.Name, d2ui.ColorTokenGold)}

	for _, p := range info.Pieces {
		c := d2ui.ColorTokenRed
		if worn[p] {
			c = d2ui.ColorTokenGreen
		}

		lines = append(lines, d2ui.ColorTokenize(p, c))
	}

	for i, tier := range info.Partial {
		c := d2ui.ColorTokenGrey
		if n >= i+2 {
			c = d2ui.ColorTokenGreen
		}

		for _, t := range tier {
			lines = append(lines, d2ui.ColorTokenize(t, c))
		}
	}

	c := d2ui.ColorTokenGrey
	if len(info.Pieces) > 0 && n >= len(info.Pieces) {
		c = d2ui.ColorTokenGreen
	}

	for _, t := range info.Full {
		lines = append(lines, d2ui.ColorTokenize(t, c))
	}

	return lines
}
