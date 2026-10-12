package d2skills

import (
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2state"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2inventory"
)

// DrinkStatePotion gives a unit the state of an antidote, thawing or stamina potion (d2inventory.StatePotionOf):
// the states named in Cure end first (poison also ends its damage stream), then the potion's state starts, or
// is extended by the potion's length when it already runs (d2state.Set.ApplyPotionState). The state's stats
// reach the unit's totals through StateStats. It returns the names of the states that ended and whether the
// new state started.
func (e *Engine) DrinkStatePotion(unitID string, p d2inventory.StatePotion) (cured []string, started bool) {
	set := e.setOf(unitID)

	for _, c := range p.Cure {
		if set.EndState(e.frame, c) {
			cured = append(cured, c)
		}
	}

	mods := make([]d2state.StatMod, 0, len(p.Mods))
	for _, m := range p.Mods {
		mods = append(mods, d2state.StatMod{Stat: m.Stat, Value: m.Value})
	}

	_, started = set.ApplyPotionState(e.frame, p.State, p.Frames, mods)

	return cured, started
}
