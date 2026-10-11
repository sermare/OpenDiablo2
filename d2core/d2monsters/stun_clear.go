package d2monsters

import "github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"

// ClearStun ends the stun hold of a monster at once (Leap Attack removes the
// victim's state 21 after its blow, SKILL_LeapAttackBuildHit 0x5d9170).
func (d *Director) ClearStun(m *d2mapentity.Monster) {
	if u := d.byEntity[m.ID()]; u != nil {
		u.stunUntil = 0
	}
}
