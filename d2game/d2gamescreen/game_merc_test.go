package d2gamescreen

import "testing"

func TestMercArrival(t *testing.T) {
	tests := []struct {
		name      string
		carry     mercCarry
		saveDead  bool
		wantSpawn bool
		wantHP    int
	}{
		{"first spawn, alive", mercCarry{}, false, true, 0},
		{"first spawn, dead in the save: a corpse", mercCarry{}, true, true, 0},
		{"travels alive with half life", mercCarry{valid: true, hp: 500}, false, true, 500},
		{"travels alive at full life", mercCarry{valid: true, hp: 0}, false, true, 0},
		{"died in the level left: stays behind", mercCarry{valid: true, hp: 0, dead: true}, true, false, 0},
		{"save says dead: stays behind", mercCarry{valid: true, hp: 10}, true, false, 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			spawn, hp := mercArrival(tc.carry, tc.saveDead)
			if spawn != tc.wantSpawn || hp != tc.wantHP {
				t.Fatalf("mercArrival = (%v, %d), want (%v, %d)", spawn, hp, tc.wantSpawn, tc.wantHP)
			}
		})
	}
}
