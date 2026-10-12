package d2mapgen

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2drlg/drlgoutdoor"
)

// OBJRGN_ShouldSpawnObjectsForRoom (0x5505e0) refuses a room by its DRLG flags.
func TestRoomHasNoObjects(t *testing.T) {
	for _, c := range []struct {
		name string
		room drlgoutdoor.Room
		want bool
	}{
		{"plain room", drlgoutdoor.Room{Type: 1}, false},
		{"waypoint site bit 0x10000", drlgoutdoor.Room{Type: 1, Flags: 0x10000}, true},
		{"waypoint chunk 0x30000", drlgoutdoor.Room{Type: 2, Flags: 0x30000}, true},
		{"populate=0 room", drlgoutdoor.Room{Type: 2, Flags: roomNoPopulate}, true},
		{"plain room on a road cell", drlgoutdoor.Room{Type: 1, Info54: 0x80}, true},
		{"preset room, road bit of the cell flag is not read", drlgoutdoor.Room{Type: 2, Info54: 0x80}, false},
		{"unrelated flag bits", drlgoutdoor.Room{Type: 1, Flags: 0x80000 | 0x4000, Info54: 0x7f}, false},
	} {
		r := c.room
		if got := roomHasNoObjects(&r); got != c.want {
			t.Errorf("%s: %v, want %v", c.name, got, c.want)
		}
	}
}
