package d2gamescreen

import (
	"fmt"
	"math"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
)

// commandPickItem is "pickitem <code>": the hero walks to the nearest ground item with that base code and picks it
// up (the scripted counterpart of clicking one particular item, for the Flail among a Council's drops).
func (v *Game) commandPickItem(args []string) error {
	code := strings.TrimSpace(args[0])
	hx, hy := v.heroTilePos()

	var (
		best *d2mapentity.Item
		bd   = math.MaxFloat64
	)

	for _, e := range v.gameClient.MapEngine.Entities() {
		it, ok := e.(*d2mapentity.Item)
		if !ok || it.Item == nil || strings.TrimSpace(it.Item.GetItemCode()) != code {
			continue
		}

		x, y := it.GetPositionF()
		if d := math.Hypot(x-hx, y-hy); d < bd {
			best, bd = it, d
		}
	}

	if best == nil {
		return fmt.Errorf("no %s on the ground", code)
	}

	v.walkToItem(best)

	return nil
}
