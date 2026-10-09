package d2hero

import (
	"fmt"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2equip"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2hireling"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2statlist"
)

// This file carries the mercenary's equipment ('jf' section of a .d2s): import into
// MercState.Items, conversion to stat list items (for d2hireling.ApplyGear), the give
// and take operations built on the pure rules of d2equip (Rules.MercGive), and the
// export back into the save. A merc item is a StoredItem with Page PageEquipped and X the
// body location (1 head, 3 torso, 4 right hand, 5 left hand).

// mercSlotOK says whether a StoredItem.X is a body location a merc has.
func mercSlotOK(x int) bool { return d2equip.MercHasLoc(d2equip.Loc(x)) }

// importMercItems records the items of the 'jf' section in the merc state. Items without
// an OpenDiablo2 record, ears and slots a merc does not have are left out (the export
// keeps the file's list until the player changes the gear, see MercState.ItemsDirty).
func importMercItems(m *MercState, items []d2s.Item, known func(string) bool) {
	if m == nil {
		return
	}

	m.Items = nil

	for i := range items {
		it := &items[i]
		if it.Location != d2s.LocationEquipped || it.Ear || !known(trimCode(it.Code)) {
			continue
		}

		s, skip := storedFromD2S(it, PageEquipped, known)
		if skip != "" {
			continue
		}

		s.X, s.Y = int(it.Equipped), 0
		m.Items = append(m.Items, s)
	}
}

// MercStatItems converts the merc's gear into stat list items for d2hireling.ApplyGear.
// An item from the save carries its exact properties; one made in the game only its base
// defense and weapon damage (as for the hero, see statItemOfStored).
func (f *HeroStateFactory) MercStatItems(m *MercState) []d2statlist.Item {
	if m == nil || len(m.Items) == 0 {
		return nil
	}

	_, ebases, ok := f.EquipRules()
	if !ok {
		ebases = d2equip.Bases{}
	}

	sbases := f.loadStatBases()
	out := make([]d2statlist.Item, 0, len(m.Items))

	for i := range m.Items {
		s := &m.Items[i]
		if !mercSlotOK(s.X) {
			continue
		}

		out = append(out, f.statItemOfStored(s, d2equip.Loc(s.X), sbases, EquipItemOf(s, ebases)))
	}

	return out
}

// MercGiveResult is the outcome of GiveMercItem.
type MercGiveResult struct {
	d2equip.MercResult
	// Returned is the stored item that was in the slot; it goes back to the player.
	Returned *StoredItem
}

// MercRef is what the rules need to know about the merc: its class (monstats id) and its
// stats from the hireling table at its level (without gear).
type MercRef struct {
	Class int
	Base  d2hireling.Stats
}

func (r MercRef) merc(gear []d2statlist.Item) d2equip.Merc {
	g := d2hireling.ApplyGear(r.Base, gear)

	return d2equip.Merc{Class: r.Class, Level: r.Base.Level, Str: g.Str, Dex: g.Dex}
}

// GiveMercItem tries to give an item to the merc following d2equip.Rules.MercGive
// (type rule of the class, identified and not broken, requirements against the merc's own
// strength, dexterity and level; a potion is consumed). On success the item takes its slot
// in state.Merc.Items and the item it replaced is returned for the player's cursor. A
// refusal changes nothing. The caller removes the item from the cursor or inventory only
// when the result is OK.
func (f *HeroStateFactory) GiveMercItem(m *MercState, ref MercRef, give StoredItem) MercGiveResult {
	if m == nil {
		return MercGiveResult{MercResult: d2equip.MercResult{Decision: d2equip.Decision{
			Reason: d2equip.ReasonBodyLoc, Detail: "no mercenary"}}}
	}

	rules, bases, ok := f.EquipRules()
	if !ok {
		return MercGiveResult{MercResult: d2equip.MercResult{Decision: d2equip.Decision{
			Reason: d2equip.ReasonBodyLoc, Detail: "equip rules unavailable"}}}
	}

	body := map[d2equip.Loc]*d2equip.Item{}
	stored := map[d2equip.Loc]*StoredItem{}

	for i := range m.Items {
		s := &m.Items[i]
		eq := EquipItemOf(s, bases)
		body[d2equip.Loc(s.X)] = &eq
		stored[d2equip.Loc(s.X)] = s
	}

	newItem := EquipItemOf(&give, bases)

	// the merc's stats as they are with all its gear (requirements are checked against
	// those, hirelings.md), and without the item that would be replaced
	withAll := ref.merc(f.MercStatItems(m))

	without := func(removed *d2equip.Item) d2equip.Merc {
		var rest []StoredItem

		for i := range m.Items {
			if s := &m.Items[i]; body[d2equip.Loc(s.X)] != removed {
				rest = append(rest, *s)
			}
		}

		return ref.merc(f.MercStatItems(&MercState{Items: rest}))
	}

	res := rules.MercGive(withAll, body, &newItem, without)
	out := MercGiveResult{MercResult: res}

	if !res.OK || res.Consumed {
		return out
	}

	if old := stored[res.Loc]; old != nil {
		cp := *old
		out.Returned = &cp
	}

	give.Page, give.X, give.Y = PageEquipped, int(res.Loc), 0

	m.setItem(give)
	m.ItemsDirty = true

	return out
}

// TakeMercItem removes the item at a body location from the merc and returns it (nil
// when the slot is empty).
func (m *MercState) TakeMercItem(loc d2equip.Loc) *StoredItem {
	if m == nil {
		return nil
	}

	for i := range m.Items {
		if m.Items[i].X == int(loc) {
			it := m.Items[i]
			m.Items = append(m.Items[:i], m.Items[i+1:]...)
			m.ItemsDirty = true

			return &it
		}
	}

	return nil
}

func (m *MercState) setItem(s StoredItem) {
	for i := range m.Items {
		if m.Items[i].X == s.X {
			m.Items[i] = s
			return
		}
	}

	m.Items = append(m.Items, s)
}

// MercItemAt returns the merc's item at a body location, or nil.
func (m *MercState) MercItemAt(loc d2equip.Loc) *StoredItem {
	if m == nil {
		return nil
	}

	for i := range m.Items {
		if m.Items[i].X == int(loc) {
			return &m.Items[i]
		}
	}

	return nil
}

// exportMercItems writes the merc's gear into the 'jf' list. The list of the file is left
// alone until the gear was changed in the game (ItemsDirty), so a save the player did not
// touch stays byte-exact even if some items could not be imported. Items made in the game
// have no bit-exact form (the writer needs the property list, which the item model does
// not keep) and are reported, not written, like the container items.
func exportMercItems(c *d2s.Character, state *HeroState, warn func(string, ...interface{})) {
	m := state.Merc
	if m == nil || !m.ItemsDirty || c.Header.Mercenary.ID == 0 {
		return
	}

	var out []d2s.Item

	for i := range m.Items {
		s := &m.Items[i]
		if s.D2S == nil {
			warn("merc item %q (slot %d) was made in the game, not written", s.Code, s.X)
			continue
		}

		it := *s.D2S

		// an item that was already worn (the merc's own or the hero's) keeps its position
		// bits; one that came out of a container loses the grid position
		if it.Location != d2s.LocationEquipped {
			it.X, it.Y = 0, 0
		}

		it.Location, it.Page, it.Equipped = d2s.LocationEquipped, 0, uint8(s.X)

		if s.Identified {
			it.Identified = true
		}

		// only a changed durability is written (a real save may hold more than the maximum)
		if s.Durability != nil && it.MaxDurability > 0 && *s.Durability != int(it.Durability) {
			it.Durability = uint16(clampInt(*s.Durability, 0, int(it.MaxDurability)))
		}

		out = append(out, it)
	}

	c.MercItems = out
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}

	if v > hi {
		return hi
	}

	return v
}

// MercItemsSummary is a log line of the gear.
func (m *MercState) MercItemsSummary() string {
	if m == nil || len(m.Items) == 0 {
		return "none"
	}

	s := ""

	for i := range m.Items {
		s += fmt.Sprintf("%s@%d ", m.Items[i].Code, m.Items[i].X)
	}

	return s
}
