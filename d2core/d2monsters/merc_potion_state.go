package d2monsters

import (
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2inventory"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
)

// An antidote or thawing potion given to the mercenary runs the same item effect as for the hero: the exe's
// give-to-merc handler (NPCSRV_ValidateMercenaryItemEquip 0x54b040) calls ITEM_ServerUseItemEffect with the
// mercenary as the user, and that accepts unit types 0 and 1 (VERIFIED), so the potion's timed state with its
// resist stats lands on the merc, extended by the length when one already works (0x5bc990). Here the stat
// lists of a merc are modelled only for the resists, so the state is a per-name end frame plus the resist
// mods that mercResistInput adds. The cure half (cstate1/2 on a poisoned or frozen merc) is not modelled:
// the engine puts no poison or freeze state on a merc (U).

type mercPotionState struct {
	until int
	mods  []d2inventory.StateMod
}

// resistStatIndex maps the resist stat names (and their max variants) to d2hireling.Gear.Resist indices:
// fire, cold, lightning, poison.
var resistStatIndex = map[string]int{"fireresist": 0, "coldresist": 1, "lightresist": 2, "poisonresist": 3}

var maxResistStatIndex = map[string]int{"maxfireresist": 0, "maxcoldresist": 1, "maxlightresist": 2, "maxpoisonresist": 3}

// DrinkMercState gives the owner's merc a timed-state potion. A dead merc takes nothing. It reports whether
// the state started.
func (d *Director) DrinkMercState(owner *d2mapentity.Player, p d2inventory.StatePotion) bool {
	u := d.mercs[owner]
	if u == nil || !u.m.Alive() || p.State == "" || p.Frames < 1 {
		return false
	}

	mu := u.merc
	if mu.potions == nil {
		mu.potions = map[string]*mercPotionState{}
	}

	left := 0
	if ex := mu.potions[p.State]; ex != nil && ex.until > d.frame {
		left = ex.until - d.frame
	}

	mu.potions[p.State] = &mercPotionState{until: d.frame + left + p.Frames, mods: p.Mods}

	d.emit("merc", "MERC state potion name=%s state=%s frames=%d until=%d", u.m.Label(), p.State, p.Frames, d.frame+left+p.Frames)

	return true
}

// potionResists returns the resist and max-resist additions of the merc's running potion states.
func (mu *mercUnit) potionResists(frame int) (resist, maxBonus [4]int) {
	for _, st := range mu.potions {
		if st == nil || st.until <= frame {
			continue
		}

		for _, m := range st.mods {
			name := strings.ToLower(m.Stat)

			if i, ok := resistStatIndex[name]; ok {
				resist[i] += m.Value
			}

			if i, ok := maxResistStatIndex[name]; ok {
				maxBonus[i] += m.Value
			}
		}
	}

	return resist, maxBonus
}
