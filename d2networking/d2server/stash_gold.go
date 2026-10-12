package d2server

import (
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2hero"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2inventory"
)

// stashGoldOf is the hero's stash gold (0 when the state has none).
func stashGoldOf(st *d2hero.HeroState) int {
	if st == nil || st.StashGold == nil {
		return 0
	}

	return *st.StashGold
}

// setStashGold stores the stash gold a client reports on a save; an amount outside 0..stash limit becomes 0 like a
// loaded stat (SanitizeLoadedGold, 0x531a50). A hero that had no stash gold and still has none keeps nil.
func setStashGold(st *d2hero.HeroState, n int) {
	n = d2inventory.SanitizeLoadedGold(n, d2inventory.StashGoldLimit)

	if st.StashGold == nil && n == 0 {
		return
	}

	st.StashGold = &n
}
