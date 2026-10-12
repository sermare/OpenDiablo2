package d2gamescreen

import "math"

// dropHeroGold puts the gold of the drop gold dialog on the ground next to the hero (SERVER_DropPlayerGold 0x532f20
// makes the piles at the hero's feet). With no free cell the gold goes back to the purse.
func (v *Game) dropHeroGold(amount int) {
	if v.localPlayer == nil || amount <= 0 {
		return
	}

	px, py := v.localPlayer.GetPositionF()

	cells := v.freeDropCells(int(math.Floor(px)), int(math.Floor(py)), 1, true)
	if len(cells) == 0 {
		v.Warningf("no free ground cell for %d gold; keeping it", amount)
		v.gameControls.AddGold(amount)

		return
	}

	if _, err := v.spawnGoldPile(amount, cells[0]); err != nil {
		v.Errorf("could not drop %d gold: %v", amount, err)
		v.gameControls.AddGold(amount)

		return
	}

	v.Infof("AUTOGROUND drop gold amount=%d pos=(%d,%d)", amount, cells[0].X, cells[0].Y)
}
