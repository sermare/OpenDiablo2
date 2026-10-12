package d2player

import (
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2equip"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2hero"
)

// The hero's side of the corpse: PLAYER_ServerCreateCorpse 0x57d6f0 moves the cursor item and every worn item to the
// corpse, PLAYER_ServerLootCorpse 0x57db90 / ITEMMODE_ServerAutoEquipFromOtherUnit 0x560d30 give them back (see
// d2hero/corpse_loot.go for the rules).

// TakeBodyForCorpse removes everything worn and the cursor item and returns them as corpse items (worn ones with
// Page PageEquipped and X the .d2s body location, the cursor item on the inventory page). The inventory, belt and
// stash stay with the hero.
func (g *GameControls) TakeBodyForCorpse() []d2hero.StoredItem {
	out := g.snapshotEquipped()

	if cur := g.inventory.CursorItem(); cur != nil {
		if s, ok := g.storedOf(cur, d2equip.LocNone); ok {
			s.Page, s.X, s.Y = d2hero.PageInventory, 0, 0
			out = append(out, s)
		}

		g.inventory.SetCursorItem(nil)
	}

	g.inventory.ClearBody()
	g.equipTouched = true
	g.recalcWorn() // no save here: the caller attaches the items to the corpse first, then saves

	return out
}

// corpseHost is the LootHost of the hero.
type corpseHost struct {
	g *GameControls
	// canTake is the quest/unique gate; nil allows everything.
	canTake func(it *d2hero.StoredItem) bool
}

func (h *corpseHost) logical(slot int) d2equip.Loc {
	if c := h.g.hero.Containers; c != nil {
		return c.LogicalLoc(slot)
	}

	return d2equip.Loc(slot)
}

// put realises the stored item on the cursor (the cursor is empty during a loot) and reports it.
func (h *corpseHost) put(s *d2hero.StoredItem) bool {
	item, err := realiseStored(h.g.inventory.item, s)
	if err != nil {
		return false
	}

	if s.D2S != nil {
		h.g.itemOrigin[item] = s.D2S
	}

	h.g.inventory.SetCursorItem(item)

	return true
}

// drop clears the cursor again after a failed attempt.
func (h *corpseHost) drop() { h.g.inventory.SetCursorItem(nil) }

func (h *corpseHost) MeetsRequirements(s *d2hero.StoredItem) bool {
	if !h.put(s) {
		return false
	}
	defer h.drop()

	if h.logical(s.X).IsSwap() {
		return true // the other weapon set is not checked while it is inactive
	}

	return h.g.evaluateEquip(h.g.inventory.CursorItem(), h.logical(s.X)).OK
}

func (h *corpseHost) CanTake(s *d2hero.StoredItem) bool { return h.canTake == nil || h.canTake(s) }

func (h *corpseHost) TryEquip(slot int, s *d2hero.StoredItem) bool {
	loc := h.logical(slot)
	if h.g.inventory.WornAt(loc) != nil || !h.put(s) {
		return false
	}

	// the inactive weapon set takes its item back where it was (no rules are checked until it is switched in)
	if loc.IsSwap() {
		h.g.inventory.SetWorn(loc, h.g.inventory.CursorItem())
		h.g.inventory.SetCursorItem(nil)

		return true
	}

	if res := h.g.evaluateEquip(h.g.inventory.CursorItem(), loc); !res.OK || len(res.Displaced) > 0 {
		h.g.Infof("CORPSE equip refused item=%s slot=%s ok=%v reason=%q displaced=%v", s.Code, loc, res.OK, res.Reason, res.Displaced)
		h.drop()
		return false
	}

	if res := h.g.TryEquipCursor(loc); res.OK && h.g.inventory.CursorItem() == nil {
		return true
	}

	h.drop()

	return false
}

func (h *corpseHost) TryBelt(s *d2hero.StoredItem) bool {
	if !h.put(s) {
		return false
	}

	if cur := h.g.inventory.CursorItem(); cur != nil {
		if _, onBelt := h.g.belt.AutoBelt(cur); onBelt {
			h.g.inventory.SetCursorItem(nil)
			return true
		}
	}

	h.drop()

	return false
}

func (h *corpseHost) TryInventory(s *d2hero.StoredItem) bool {
	if !h.put(s) {
		return false
	}

	if _, _, ok := h.g.inventory.AutoPlaceCursor(); ok {
		return true
	}

	h.drop()

	return false
}

// LootCorpse gives the hero what the corpse holds (d2hero.LootCorpse) and returns whether nothing is left. A cursor
// item the hero holds is set aside during the transfer and held again afterwards. canTake may be nil.
func (g *GameControls) LootCorpse(c *d2hero.Corpse, canTake func(it *d2hero.StoredItem) bool) (complete bool) {
	held := g.inventory.CursorItem()
	g.inventory.SetCursorItem(nil)

	prevNoSave := g.equipNoSave
	g.equipNoSave = true

	defer func() { g.equipNoSave = prevNoSave }()

	_, complete = d2hero.LootCorpse(c, &corpseHost{g: g, canTake: canTake})

	g.inventory.SetCursorItem(held)
	// no save here: the corpse still lists its items until the caller has updated it, and the caller saves
	g.equipTouched = true
	g.recalcWorn()

	return complete
}
