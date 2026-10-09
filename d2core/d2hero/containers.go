package d2hero

import (
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2statlist"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/diablo2item"
)

// Item pages of a .d2s file (d2s.Item.Page for stored items) plus the belt.
// PageBelt is not a .d2s page: belt items have d2s.LocationBelt (2) and their
// X is the belt cell; it is numbered like that location so the two read alike.
const (
	PageInventory = 1
	PageBelt      = 2
	PageCube      = 4
	PageStash     = 5
)

// StoredItem is one item of the hero's inventory, belt, cube or stash in the
// form the hero file keeps. The item itself is rebuilt from it (see
// diablo2item.Spec): either from the rolled identity (Unique, SetItem, Prefixes,
// Suffixes and Seed), or, for an item imported from a .d2s, from Origin, which
// rolls the base item again.
type StoredItem struct {
	Code string `json:"code"`
	Page int    `json:"page"` // PageInventory, PageBelt, PageCube or PageStash
	X    int    `json:"x"`    // grid column, or the belt cell for PageBelt
	Y    int    `json:"y"`

	Quality  int      `json:"quality,omitempty"`
	ILvl     int      `json:"ilvl,omitempty"`
	Seed     int64    `json:"seed,omitempty"`
	Unique   string   `json:"unique,omitempty"`
	SetItem  string   `json:"setItem,omitempty"`
	Set      string   `json:"set,omitempty"`
	Prefixes []string `json:"prefixes,omitempty"`
	Suffixes []string `json:"suffixes,omitempty"`

	Identified bool `json:"identified,omitempty"`
	Ethereal   bool `json:"ethereal,omitempty"`
	Quantity   int  `json:"quantity,omitempty"`
	// Durability is the current durability; nil leaves the item's default.
	Durability *int `json:"durability,omitempty"`

	// Origin is set for an item imported from a .d2s: the item is created
	// from Code, Quality, ILvl and Seed with the item generator, which is an
	// APPROXIMATION of the real item (the save's affix ids are not mapped to
	// OpenDiablo2's records, so a magic or unique item gets random affixes of
	// the same kind). It is cleared as soon as the item has been built once
	// and its rolled identity stored.
	Origin bool `json:"origin,omitempty"`

	// D2S is the item exactly as the save held it, kept for writing the item
	// back to a .d2s (ExportD2SItems). Items made in the game have none.
	D2S *d2s.Item `json:"d2s,omitempty"`

	// Spec is the whole item as the item creator rolled it (affixes,
	// properties, sockets, runeword, ear): the item is rebuilt from it exactly.
	// Stat is what the hero's stat list computes with (see
	// diablo2item.Item.StatItem), so the worn item's properties and set pieces
	// count without the item being rebuilt.
	Spec *diablo2item.Spec `json:"spec,omitempty"`
	Stat *d2statlist.Item  `json:"stat,omitempty"`
}

// HeroContainers is the hero's item storage besides the equipment. A hero file
// without it (older files, new heroes) loads with nil; a non-nil value means
// the containers have been saved at least once, even if they are empty.
type HeroContainers struct {
	Items []StoredItem `json:"items"`
	// BeltCode is the base code of the equipped belt (it sets the number of
	// belt cells). The equipment model has no belt slot yet, so an imported
	// save keeps the code here; empty when unknown.
	BeltCode string `json:"beltCode,omitempty"`

	// Equipped is what the hero wears (Page PageEquipped, X the .d2s body
	// location: weapon set II in 11 and 12). EquippedSet says the list is
	// authoritative even when empty; without it (older files, new heroes) the
	// stat list falls back on the imported save or the base equipment.
	Equipped    []StoredItem `json:"equipped,omitempty"`
	EquippedSet bool         `json:"equippedSet,omitempty"`
	// ActiveArms is the weapon set in the hands: 0 set I (locations 4/5), 1 set
	// II (11/12). It is the .d2s header field at 0x10.
	ActiveArms int `json:"activeArms,omitempty"`
}

// d2sSlotBelt is the .d2s equipment slot of the belt.
const d2sSlotBelt = 8

// Page returns the items of a page, in storage order.
func (c *HeroContainers) Page(page int) []StoredItem {
	if c == nil {
		return nil
	}

	var out []StoredItem

	for i := range c.Items {
		if c.Items[i].Page == page {
			out = append(out, c.Items[i])
		}
	}

	return out
}

// d2sQuality maps a .d2s quality to the item generator's, which numbers
// qualities the same way. Simple items (potions, runes) store 0 for none.
func d2sQuality(q uint8) int {
	if q == 0 {
		return int(d2s.QualityNormal)
	}

	return int(q)
}

// StoredFromD2S converts a stored or belt item of a .d2s save. skip is a
// non-empty reason when the item is not converted: it is not in a container
// (equipped, cursor, socketed), is an ear, or known(code) says OpenDiablo2 has
// no record for its base item.
func StoredFromD2S(it *d2s.Item, known func(code string) bool) (s StoredItem, skip string) {
	return storedFromD2S(it, -1, known)
}

// storedFromD2S is StoredFromD2S; a page >= 0 forces the page (worn items,
// whose location is not a container).
func storedFromD2S(it *d2s.Item, forcePage int, known func(code string) bool) (s StoredItem, skip string) {
	var page int

	switch {
	case forcePage >= 0:
		page = forcePage
	default:
		page, skip = containerPage(it)
		if skip != "" {
			return s, skip
		}
	}

	return finishStored(it, page, known)
}

func containerPage(it *d2s.Item) (page int, skip string) {
	switch it.Location {
	case d2s.LocationStored:
		page = int(it.Page)
		if page != PageInventory && page != PageCube && page != PageStash {
			return 0, "unknown page"
		}
	case d2s.LocationBelt:
		page = PageBelt
	default:
		return 0, "not in a container"
	}

	return page, ""
}

func finishStored(it *d2s.Item, page int, known func(code string) bool) (s StoredItem, skip string) {
	if it.Ear {
		return s, "player ear"
	}

	code := trimCode(it.Code)
	if !known(code) {
		return s, "no OpenDiablo2 record for item code " + code
	}

	s = StoredItem{
		Code: code, Page: page, X: int(it.X), Y: int(it.Y),
		Quality: d2sQuality(it.Quality), ILvl: int(it.Level),
		// the save's item id is stable per item, so the same save imports to the same items
		Seed:       int64(it.ID),
		Identified: it.Identified || it.Simple,
		Ethereal:   it.Ethereal,
		Origin:     true,
	}

	if it.Quantity > 0 {
		s.Quantity = int(it.Quantity)
	}

	if it.MaxDurability > 0 {
		d := int(it.Durability)
		s.Durability = &d
	}

	cp := *it
	s.D2S = &cp

	return s, ""
}

func trimCode(code string) string {
	for len(code) > 0 && (code[len(code)-1] == ' ' || code[len(code)-1] == 0) {
		code = code[:len(code)-1]
	}

	return code
}

// ExportD2SItems returns the container items as .d2s items for the item list
// of a save: the item exactly as imported, with its page, position and (for
// the belt) cell updated from the stored item. Items made in the game have no
// bit-exact form yet and are reported in skipped (the writer needs the item's
// property list, which the item model does not keep).
func ExportD2SItems(c *HeroContainers) (out []d2s.Item, skipped []StoredItem) {
	if c == nil {
		return nil, nil
	}

	for i := range c.Items {
		s := &c.Items[i]
		if s.D2S == nil {
			skipped = append(skipped, *s)
			continue
		}

		it := *s.D2S
		it.X, it.Y = uint8(s.X), uint8(s.Y)

		// identification done in the game (Cain, a scroll, a bought gamble item) is
		// the one change besides the position: an item never becomes
		// unidentified again. The affix data of an unidentified item is stored in
		// the save anyway, so only the flag changes.
		if s.Identified {
			it.Identified = true
		}

		if s.Page == PageBelt {
			it.Location, it.Page = d2s.LocationBelt, 0
		} else {
			it.Location, it.Page = d2s.LocationStored, uint8(s.Page)
		}

		out = append(out, it)
	}

	return out, skipped
}
