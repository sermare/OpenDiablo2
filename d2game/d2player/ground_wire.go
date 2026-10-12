package d2player

import (
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2hero"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/diablo2item"
)

// GroundItemData describes an item for the wire (a ground drop sent to the server), the same description the
// hero file keeps.
func GroundItemData(it *diablo2item.Item) d2hero.StoredItem {
	return storedFromItem(it, 0, 0, 0, nil)
}

// GroundItemFromData builds the item a ground drop describes.
func GroundItemFromData(f *diablo2item.ItemFactory, s *d2hero.StoredItem) (*diablo2item.Item, error) {
	return realiseStored(f, s)
}
