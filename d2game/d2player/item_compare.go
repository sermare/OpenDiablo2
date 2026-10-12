package d2player

import (
	"fmt"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2equip"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2statlist"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2hero"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/diablo2item"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2ui"
)

// The comparison lines of an item tooltip (the owner's request): how an item compares with the one the hero
// wears where it would go. The numbers come from the stat list of the two bodies (d2hero.Compare), not from the
// description text. The same lines show on the inventory, the stash, the cube, the vendor window, the held item
// and the item under the pointer on the ground. This is an OD2 extension: INV_BuildItemNameTooltipLines 0x488260 builds the 1.14b tooltip from the item alone, with no
// comparison against worn gear.

// compareCache keeps the lines of the last item: a tooltip asks every frame, a comparison runs a handful of stat
// list passes.
type compareCache struct {
	key   string
	lines []string
}

// compareLines returns the coloured comparison lines of an item (nothing for an item that is worn, one that
// cannot go to the body, an unidentified one, or a body place that is still empty).
func (g *GameControls) compareLines(it InventoryItem) []string {
	item, ok := it.(*diablo2item.Item)
	if !ok || item == nil {
		return nil
	}

	key := g.compareKey(item)
	if g.cmpCache != nil && g.cmpCache.key == key {
		return g.cmpCache.lines
	}

	lines := g.formatComparison(g.compareItem(item))
	g.cmpCache = &compareCache{key: key, lines: lines}

	return lines
}

// updateCompareTooltip shows the comparison of the item held on the cursor, or else of the item on the ground under
// the pointer (its label is up with the show-items key or the hover), as a small tooltip above it. It is hidden
// when there is no comparison to show or the pointer is over an interface panel.
func (g *GameControls) updateCompareTooltip() {
	if g.cmpTooltip == nil {
		return
	}

	var item *diablo2item.Item

	dy := 28 // above the picture of a held item

	mx, my := g.lastMouseX, g.lastMouseY

	if cur, ok := g.inventory.CursorItem().(*diablo2item.Item); ok && cur != nil {
		item = cur

		if sprite := g.inventory.grid.sprites[cur.GetItemCode()]; sprite != nil {
			_, h := sprite.GetCurrentFrameSize()
			dy = h/2 + 4
		}
	} else if g.hud != nil && g.inventory.CursorItem() == nil && !g.isInActiveMenusRect(mx, my) {
		if e, isItem := g.hud.hoveredEntity.(*d2mapentity.Item); isItem && e.Item != nil {
			item, dy = e.Item, 40
		}
	}

	var lines []string

	if item != nil {
		if cmp := g.compareLines(item); len(cmp) > 0 {
			lines = append([]string{firstLineOf(item)}, cmp...)
		}
	}

	if len(lines) == 0 {
		g.cmpTooltip.SetVisible(false)

		return
	}

	g.cmpTooltip.SetTextLines(lines)
	g.cmpTooltip.SetPosition(mx, my-dy)
	g.cmpTooltip.SetVisible(true)
}

func firstLineOf(item *diablo2item.Item) string {
	if lines := item.GetItemDescription(); len(lines) > 0 {
		return lines[0]
	}

	return item.Label()
}

// compareKey says when a cached comparison is still right: the item, everything worn, the weapon set, the hero's
// attributes and the colour option.
func (g *GameControls) compareKey(item *diablo2item.Item) string {
	var b strings.Builder

	fmt.Fprintf(&b, "%p|%d|%t|", item, g.inventory.activeArms, d2ui.ColorBlind())

	if g.hero != nil && g.hero.Stats != nil {
		st := g.hero.Stats
		fmt.Fprintf(&b, "%d|%d|%d|", st.Level, st.Strength, st.Dexterity)

		if t := st.Totals; t != nil {
			fmt.Fprintf(&b, "%d|%d|%d|%d|", t.Str, t.Dex, t.Defense, t.DamageMax)
		}
	}

	for l := d2equip.Loc(1); l < d2equip.NumLocs; l++ {
		fmt.Fprintf(&b, "%p,", g.inventory.WornAt(l))
	}

	cur, mx := item.Durability()
	fmt.Fprintf(&b, "|%d/%d", cur, mx)

	return b.String()
}

// compareItem runs the comparison of an item against the body place(s) it would go to; ok false when there is
// nothing to compare.
func (g *GameControls) compareItem(item *diablo2item.Item) *d2hero.Comparison {
	rules, bases, ok := g.heroState.EquipRules()
	if !ok || g.hero.Stats == nil || g.hero.Stats.Totals == nil {
		return nil
	}

	for l := d2equip.Loc(1); l < d2equip.NumLocs; l++ {
		if g.inventory.WornAt(l) == item {
			return nil
		}
	}

	s, isItem := g.storedOf(item, d2equip.LocNone)
	if !isItem || !s.Identified && !item.IsIdentified() {
		return nil
	}

	eq := d2hero.EquipItemOf(&s, bases)
	itemIsWeapon := eq.Weapon

	base := g.snapshotContainers()
	st0 := g.hero.Stats
	eval := func(body []d2hero.StoredItem, b d2hero.Boost) (d2statlist.Totals, []d2hero.EquipStatus) {
		c := *base
		c.Equipped, c.EquippedSet, c.ActiveArms = body, true, g.inventory.activeArms
		st := *st0
		st.Strength += b.Str
		st.Dexterity += b.Dex
		st.Level += b.Level
		status := g.heroState.RecalcWorn(&st, g.hero.Class, &c)

		if st.Totals == nil {
			return d2statlist.Totals{}, status
		}

		return *st.Totals, status
	}

	// the requirements (the figures say what the item gives once they are met; the red note says they are not)
	t := st0.Totals
	req := d2equip.Check(d2equip.Hero{Class: g.equipClass(), Str: t.Str, Dex: t.Dex, Level: st0.Level}, &eq, rules.Types)

	worn := map[d2equip.Loc]d2hero.StoredItem{}
	wornWeapon := map[d2equip.Loc]bool{}

	for l := d2equip.Loc(1); l < d2equip.NumLocs; l++ {
		if w := g.inventory.WornAt(l); w != nil {
			if ws, okw := g.storedOf(w, l); okw {
				worn[l] = ws
				wornWeapon[l] = d2hero.EquipItemOf(&ws, bases).Weapon
			}
		}
	}

	var all []d2hero.Comparison

	for _, loc := range autoEquipLocs {
		res := g.evaluateEquip(item, loc)
		if !placeable(res.Reason) {
			continue
		}

		w, has := worn[loc]
		if !has && len(res.Displaced) == 0 {
			if itemIsWeapon {
				continue // a weapon is compared with the weapons that are worn
			}

			return nil // the place is free: the item simply goes there
		}

		if itemIsWeapon && !wornWeapon[loc] {
			continue
		}

		if !itemIsWeapon && loc == d2equip.LocRightHand {
			continue // ammunition goes to a hand: nothing to compare with
		}

		in := d2hero.CompareInput{
			Eval: eval, Worn: worn, Item: s, Loc: loc, Displaced: res.Displaced,
			NeedStr: req.NeedStr, NeedDex: req.NeedDex, NeedLevel: req.NeedLevel, HeroLevel: st0.Level,
			FileLoc: func(l d2equip.Loc) int { return int(d2equip.EffectiveLoc(l, g.inventory.activeArms)) },
			Weapons: itemIsWeapon && has && wornWeapon[loc],
			// the sockets of the rolled item and of the worn one, and their maximum durability
			NewSockets: socketsOf(&s, bases), OldSockets: socketsOf(&w, bases),
			NewMaxDur: maxDurabilityOf(item), OldMaxDur: maxDurabilityOf(g.inventory.WornAt(loc)),
		}

		all = append(all, d2hero.Compare(in))
	}

	if i := d2hero.PickLowest(all); i >= 0 {
		return &all[i]
	}

	return nil
}

// placeable says whether a verdict of evaluateEquip leaves the body place a candidate: only the requirements and a
// broken item may still be wrong (the comparison then says what the item would give).
func placeable(r d2equip.Reason) bool {
	switch r {
	case d2equip.ReasonOK, d2equip.ReasonStrength, d2equip.ReasonDexterity, d2equip.ReasonLevel, d2equip.ReasonBroken:
		return true
	}

	return false
}

// socketsOf is the rolled socket count of an item whose base can have sockets, else 0 (arrows and potions are not
// compared by it).
func socketsOf(s *d2hero.StoredItem, bases d2equip.Bases) int {
	if bases[s.Code].GemSockets <= 0 {
		return 0
	}

	return s.Sockets
}

func maxDurabilityOf(it InventoryItem) int {
	if item, ok := it.(*diablo2item.Item); ok {
		_, mx := item.Durability()

		return mx
	}

	return 0
}

// formatComparison turns the rows into tooltip lines: blue for better, red for worse, white for the same; the
// colour-blind option swaps the red for the vermillion and also writes the words.
func (g *GameControls) formatComparison(c *d2hero.Comparison) []string {
	if c == nil || !c.Active {
		return nil
	}

	lines := compareRowLines(c.Rows)

	for _, l := range c.Unequips {
		name := l.String()
		if w := g.inventory.WornAt(l); w != nil {
			name = itemName(w)
		}

		lines = append(lines, d2ui.ColorTokenize(fmt.Sprintf("Also takes off: %s", name), d2ui.CompareEqual()))
	}

	return lines
}

// compareRowLines colours the rows of a comparison.
func compareRowLines(rows []d2hero.CompareRow) []string {
	lines := make([]string, 0, len(rows))

	for _, r := range rows {
		text := fmt.Sprintf("%s: %s", r.Name, r.Text)
		token := d2ui.CompareEqual()

		switch r.Verdict {
		case d2hero.Better:
			token = d2ui.CompareBetter()

			if d2ui.CompareWords() {
				text += " (better)"
			}
		case d2hero.Worse:
			token = d2ui.CompareWorse()

			if d2ui.CompareWords() {
				text += " (worse)"
			}
		}

		lines = append(lines, d2ui.ColorTokenize(text, token))
	}

	return lines
}
