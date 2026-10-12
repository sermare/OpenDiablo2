package drlgoutdoor

import (
	"reflect"
	"testing"
)

// TestAddSubObjects: the type 2 objects inside the stamped group rectangle become units of the room, moved
// by the offset between the group origin and the stamp position; monsters and objects outside stay out.
func TestAddSubObjects(t *testing.T) {
	d := &Pattern{Path: "Act1/Outdoors/Waypoint.ds1", Objects: []Object{
		{Type: 2, ID: 1, X: 12, Y: 7},  // tile 2,1: inside
		{Type: 1, ID: 2, X: 12, Y: 7},  // a monster: never
		{Type: 2, ID: 3, X: 30, Y: 7},  // tile 6,1: outside the rectangle
		{Type: 2, ID: 4, X: 5, Y: 0},   // the origin sub-tile of the group: out (strict bounds, 0x6726b0)
		{Type: 2, ID: 6, X: 6, Y: 1},   // one sub-tile past the origin: in
		{Type: 2, ID: 5, X: 19, Y: 14}, // tile 3,2: the far corner inside
	}}
	g := Group{X: 1, Y: 0, W: 3, H: 3}
	b := &roomBuilder{r: &Room{X: 40, Y: 80}, g: &RoomGrids{}}

	b.addSubObjects(2, 3, g, d, 0)

	want := []SubStamp{{File: "Act1/Outdoors/Waypoint.ds1", OffX: 40 + 2 - 1, OffY: 80 + 3 - 0, Objects: []int{0, 4, 5}}}
	if !reflect.DeepEqual(b.g.Subs, want) {
		t.Fatalf("got %+v, want %+v", b.g.Subs, want)
	}

	// a pattern without a file name (not loaded through Env.Pattern) or without objects records nothing
	b2 := &roomBuilder{r: &Room{}, g: &RoomGrids{}}
	b2.addSubObjects(0, 0, g, &Pattern{Objects: d.Objects}, 0)
	b2.addSubObjects(0, 0, g, &Pattern{Path: "x"}, 0)

	if len(b2.g.Subs) != 0 {
		t.Fatalf("recorded %+v", b2.g.Subs)
	}
}
