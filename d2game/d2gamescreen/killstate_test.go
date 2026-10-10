package d2gamescreen

import "testing"

// TestNewKillStateMakesItsMaps: the defensive fight a walk starts (level_edges.go) casts the left skill too, and
// panicked on a nil map in the Act 2 playthrough of the Barbarian.
func TestNewKillStateMakesItsMaps(t *testing.T) {
	for _, defend := range []bool{false, true} {
		k := newKillState(30, 45, defend)
		k.casts["Frenzy"]++
		k.refusals["los"]++
		k.dropped[3] = true
		k.skip[nil] = 1

		if k.defend != defend || k.radius != 30 || k.deadline != 45 {
			t.Errorf("fight %+v", k)
		}
	}
}
