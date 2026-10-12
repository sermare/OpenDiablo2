package d2gamescreen

import "github.com/OpenDiablo2/OpenDiablo2/d2common/d2level"

// advanceVendorRestock runs the town-leaving vendor restock check. In the
// original the check (0x534c20) is tail-called from the town transition
// (SERVER_HandleTownTransition 0x534d40) with the act of the town being left:
// it flags the vendors of that act (every opened one when no other player is
// left in the town, else those whose stock is older than four minutes), and
// the flag is consumed the next time the vendor window opens. Entering a town
// or staying in it never restocks (VERIFIED, d2-re-notes/w44-vendors.md).
func (v *Game) advanceVendorRestock() {
	cur := v.currentLevel()
	prev := v.restockLevel
	v.restockLevel = cur

	if prev == 0 || prev == cur || !d2level.IsTown(prev) || v.gameControls == nil || v.gameControls.Trade == nil {
		return
	}

	// Nobody else stands in the town in this engine's one hero game (the callback 0x534200 counts the players
	// left in the town), so the vendors are reset whatever the age of their stock.
	v.gameControls.Trade.LeaveTown(d2level.ActOfLevel(prev)-1, false)
}
