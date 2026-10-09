package d2player

import (
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2interface"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
)

type inputCallbackListener interface {
	OnPlayerMove(x, y float64)
	OnPlayerCast(skillID int, x, y float64)
	OnPlayerInteract(entity d2interface.MapEntity)
	// OnPlayerDropItem is called when the hero clicks the world while holding
	// an item on the cursor; the item is already off the cursor.
	OnPlayerDropItem(item InventoryItem)
	OnPlayerSave() error
	OnPlayerAttack(monster *d2mapentity.Monster)
}
