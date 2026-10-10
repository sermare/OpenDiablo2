package d2mapgen

import (
	"reflect"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2drlg/drlgoutdoor"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2monreg"
)

// useLogic hands the logic regions of the built tiles to the population rooms in map tiles, keeps the clipped-away
// (all zero) nodes zero, and takes over the no-population flag the build set.
func TestUseLogic(t *testing.T) {
	// the game room is at (3000,960); on the map it is at (40,8)
	rt := &drlgoutdoor.RoomTiles{
		Room: &drlgoutdoor.Room{X: 3000, Y: 960, W: 8, H: 8},
		Logic: []drlgoutdoor.LogicRegion{
			{X0: 3004, Y0: 960, X1: 3008, Y1: 968, ID: 7},
			{X0: 3000, Y0: 960, X1: 3004, Y1: 968, Skip: true, ID: 2},
			{ID: 0}, // clipped away
		},
	}

	p := &popLevel{}
	r := p.addRoom(40, 8, 8, 8, d2rand.Seed{}, false)

	p.useLogic([]*drlgoutdoor.RoomTiles{rt}, nil)

	want := []d2monreg.Cell{
		{X0: 44, Y0: 8, X1: 48, Y1: 16, Flag: 7},
		{X0: 40, Y0: 8, X1: 44, Y1: 16, Skip: true, Flag: 2},
		{},
	}

	if !reflect.DeepEqual(r.cells, want) {
		t.Errorf("cells = %+v, want %+v", r.cells, want)
	}

	if r.noPop {
		t.Error("room became no-populate without the flag")
	}

	rt.Room.Flags = 0x800000 // a hidden warp floor / a town next to the room sets this during the build
	p.useLogic([]*drlgoutdoor.RoomTiles{rt}, nil)

	if !r.noPop {
		t.Error("the no-populate flag of the build was not taken over")
	}
}
