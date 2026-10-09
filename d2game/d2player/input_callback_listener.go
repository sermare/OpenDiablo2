package d2player

import "github.com/OpenDiablo2/OpenDiablo2/d2common/d2interface"

type inputCallbackListener interface {
	OnPlayerMove(x, y float64)
	OnPlayerCast(skillID int, x, y float64)
	OnPlayerInteract(entity d2interface.MapEntity)
	OnPlayerSave() error
}
