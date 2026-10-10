package d2mapgen

import "testing"

func TestPickEntry(t *testing.T) {
	wp := [2]int{13, 27}
	rooms := []entryCand{
		{x: 0, y: 0, w: 8, h: 8},
		{x: 8, y: 0, w: 8, h: 8, flags: 0x10},                   // exit slot
		{x: 16, y: 0, w: 8, h: 8, flags: 0x30000},               // waypoint chunk, no object
		{x: 8, y: 8, w: 8, h: 8, flags: 0x30010, waypoint: &wp}, // later waypoint room
	}

	tests := []struct {
		name         string
		cands        []entryCand
		cx, cy       int
		wantX, wantY int
		wantOK       bool
	}{
		{"first waypoint room wins over the exit room (centre, no object)", rooms, 4, 4, 20, 4, true},
		{"waypoint object position", rooms[3:], 4, 4, 13, 27, true},
		{"exit room when there is no waypoint", rooms[:2], 4, 4, 12, 4, true},
		{"room at the level centre", []entryCand{{x: 0, y: 0, w: 8, h: 8}, {x: 8, y: 0, w: 8, h: 8}}, 9, 3, 12, 4, true},
		{"first room when nothing matches", []entryCand{{x: 0, y: 0, w: 8, h: 8}}, 50, 50, 4, 4, true},
		{"no rooms", nil, 0, 0, 0, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			x, y, _, ok := pickEntry(tt.cands, tt.cx, tt.cy)
			if ok != tt.wantOK || x != tt.wantX || y != tt.wantY {
				t.Fatalf("got (%d,%d,%v), want (%d,%d,%v)", x, y, ok, tt.wantX, tt.wantY, tt.wantOK)
			}
		})
	}
}
