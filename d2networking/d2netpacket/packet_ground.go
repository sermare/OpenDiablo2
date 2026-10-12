package d2netpacket

import (
	"encoding/json"

	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2hero"
	"github.com/OpenDiablo2/OpenDiablo2/d2networking/d2netpacket/d2netpackettype"
)

// Ground item operations (GroundItemPacket.Op). The original's server owns every item unit: a pickup
// request names the unit id (ITEMMODE_ServerHandlePickupRequest 0x55ace0 looks it up by id and type, and
// does nothing when its mode is no longer 3, "on the ground"), so of two heroes asking for the same item
// the first request wins and the second one finds nothing to take.
const (
	GroundDrop   = "drop"   // client -> server: an item or pile landed on the ground in the sender's level
	GroundPlaced = "placed" // server -> heroes of the level: the item exists now, with its id
	GroundBind   = "bind"   // server -> sender: the drop was known already (same Key), here is its id
	GroundPickup = "pickup" // client -> server: request to take the item named by the entry's ID
	GroundRemove = "remove" // server -> heroes of the level: the item is gone, Picker took it ("" = it expired)
	GroundDeny   = "deny"   // server -> requester: the item was taken or is not there
	GroundExpire = "expire" // client -> server: the item timed out on the ground
	GroundSync   = "sync"   // server -> a hero that entered a level: every item lying there
)

// GroundEntry is one item or gold pile on the ground.
type GroundEntry struct {
	// ID is the server's item id (empty in a drop request).
	ID string `json:"id,omitempty"`
	// Key identifies the drop independent of who reports it: the clients simulate monster loot
	// alike and every one of them reports it, the server keeps the first.
	Key   string `json:"key,omitempty"`
	Level int    `json:"level,omitempty"`
	// X, Y are subtile coordinates.
	X, Y int
	// Item is the item's description (nil for a gold pile).
	Item *d2hero.StoredItem `json:"item,omitempty"`
	Gold int                `json:"gold,omitempty"`
	// Owner names the hero that dropped the pile or item ("" for world loot).
	Owner string `json:"owner,omitempty"`
}

// GroundItemPacket carries one ground item operation.
type GroundItemPacket struct {
	Op string `json:"op"`
	// Ref is the sender's own handle of a drop (the uuid of its local entity), echoed in
	// placed/bind so it can attach the server id to the entity it already shows.
	Ref string `json:"ref,omitempty"`
	// Picker (remove) is the id of the hero that took the item; Sender (placed) the one that dropped it.
	Picker  string        `json:"picker,omitempty"`
	Sender  string        `json:"sender,omitempty"`
	Entries []GroundEntry `json:"entries,omitempty"`
}

// CreateGroundItemPacket builds a GroundItem packet.
func CreateGroundItemPacket(p GroundItemPacket) (NetPacket, error) {
	return social(d2netpackettype.GroundItem, p)
}

// UnmarshalGroundItem decodes a GroundItem packet's data.
func UnmarshalGroundItem(b []byte) (p GroundItemPacket, err error) {
	err = json.Unmarshal(b, &p)
	return
}
