package d2mapentity

import (
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2interface"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2vector"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/diablo2item"
)

// static check that item implements map entity interface
var _ d2interface.MapEntity = &Item{}

// Item is a map entity for an item
type Item struct {
	*AnimatedEntity
	// Item is nil for a gold pile.
	Item *diablo2item.Item
	// Gold is the amount of a gold pile (0 for real items).
	Gold      int
	goldLabel string
	// DropOwner names the hero that dropped the pile (gold linked to a player,
	// flag-ex bit 10 of the unit in ITEMMODE_PickupGoldFromGround 0x55a5f0); empty for
	// gold from the world (monsters, chests). A player-owned pile is not shared with the
	// party: it goes to the picker alone.
	DropOwner string
	// NetID is the server's id of the item in a multiplayer game ("" in single player, or before the server
	// answered the drop); NetKey identifies the drop for the server's dedup; NetPending: a pickup was asked.
	NetID, NetKey string
	NetPending    bool
	// DropSound is the Sounds.txt handle played when the item lands.
	DropSound string
}

// IsGold reports whether the entity is a gold pile.
func (i *Item) IsGold() bool { return i.Item == nil && i.Gold > 0 }

// ID returns the item uuid
func (i *Item) ID() string {
	return i.AnimatedEntity.uuid
}

// GetPosition returns the item position vector
func (i *Item) GetPosition() d2vector.Position {
	return i.AnimatedEntity.Position
}

// GetVelocity returns the item velocity vector
func (i *Item) GetVelocity() d2vector.Vector {
	return i.AnimatedEntity.velocity
}

// Selectable always returns true for items
func (i *Item) Selectable() bool {
	return true
}

// Highlight sets the highlight flag for a single render tick
func (i *Item) Highlight() {
	i.AnimatedEntity.highlight = true
}

// Label returns the item label
func (i *Item) Label() string {
	if i.IsGold() {
		return i.goldLabel
	}

	return i.Item.Label()
}

// GroundLabel is the name shown for the Alt ground label: coloured by the label rule of
// INV_DrawGroundItemLabels (gold piles keep their gold label).
func (i *Item) GroundLabel() string {
	if i.IsGold() || i.Item == nil {
		return i.Label()
	}

	return i.Item.GroundLabelText()
}

// GetSize returns the current frame size
func (i *Item) GetSize() (width, height int) {
	w, h := i.animation.GetCurrentFrameSize()

	if w < minHitboxSize {
		w = minHitboxSize
	}

	if h < minHitboxSize {
		h = minHitboxSize
	}

	return w, h
}
