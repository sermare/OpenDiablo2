package drlgoutdoor

import (
	"reflect"
	"testing"
)

func TestGridAliasing(t *testing.T) {
	g := NewGrid(3, 2)
	g.Set(0, 1, 7) // flat index 3
	g.Op(0, 1, 8, opOr)

	if g.Get(3, 0) != 15 {
		t.Fatalf("x >= W must read the next row, got %d", g.Get(3, 0))
	}

	defer func() {
		if recover() == nil {
			t.Fatal("an access outside the allocation must panic")
		}
	}()

	g.Get(0, 2)
}

// Blood Moor (56x96) with the town to the west and Cold Plains to the south,
// the example of drlg3.md section 4.
func TestBloodMoorPolygon(t *testing.T) {
	head := buildPolygon(Rect{1120, 920, 56, 96}, []Neighbor{
		{Level: 1, Dir: 0, F8: true, Rect: Rect{1064, 928, 56, 40}},
		{Level: 3, Dir: 3, Rect: Rect{1080, 1016, 80, 80}},
	})

	var got [][4]int

	for p := head; ; {
		got = append(got, [4]int{p.X, p.Y, p.B, p.F})

		if p = p.Next; p == head {
			break
		}
	}

	want := [][4]int{{0, 11, 0, 0}, {0, 5, 0, 3}, {0, 1, 0, 0}, {0, 0, 0, 0}, {6, 0, 0, 0}, {6, 11, 0, 0}, {4, 11, 0, 1}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("polygon %v, want %v", got, want)
	}
}

func TestInsertSortedKeepsHead(t *testing.T) {
	a := Neighbor{Dir: 2, Rect: Rect{Y: 10}}
	b := Neighbor{Dir: 1, Rect: Rect{X: 5}}
	c := Neighbor{Dir: 0, Rect: Rect{Y: 3}}
	l := insertSorted(nil, a)
	l = insertSorted(l, b) // one element: compared, goes first
	l = insertSorted(l, c) // two elements: the head (b) is never compared

	if l[0].Dir != 1 || l[1].Dir != 0 || l[2].Dir != 2 {
		t.Fatalf("unexpected order %+v", l)
	}
}

func TestParsePatternPadsTreesOverRead(t *testing.T) {
	// version 12, 1x1, one wall and one floor layer is not representable in v12
	// (single layers), 2 groups announced but only 1 stored: the second is zero.
	b := []byte{}
	put := func(v int32) { b = append(b, byte(v), byte(v>>8), byte(v>>16), byte(v>>24)) }

	put(12) // version
	put(0)  // width-1
	put(0)  // height-1
	put(0)  // act
	put(1)  // substitution type
	put(0)  // number of file names
	put(0)
	put(0) // 8 skipped bytes (versions 9..13)
	put(1) // wall layers

	for k := 0; k < 4; k++ { // wall, orientation, floor, shadow
		put(0)
	}

	put(0) // substitution layer
	put(0) // objects
	put(2) // group count
	put(1)
	put(2)
	put(3)
	put(4)

	p, err := ParsePattern(b)
	if err != nil {
		t.Fatal(err)
	}

	if len(p.Groups) != 2 || p.Groups[0] != (Group{1, 2, 3, 4, 0}) || p.Groups[1] != (Group{}) {
		t.Fatalf("groups %+v", p.Groups)
	}
}

func TestEdgeAdjacent(t *testing.T) {
	a := Rect{0, 0, 64, 160}

	for _, tt := range []struct {
		b    Rect
		want bool
	}{
		{Rect{64, 0, 160, 64}, true},    // shares an edge segment
		{Rect{64, 160, 64, 64}, false},  // corner touch only
		{Rect{200, 0, 64, 64}, false},   // apart
		{Rect{-64, 100, 64, 100}, true}, // left neighbour
	} {
		if got := edgeAdjacent(a, tt.b); got != tt.want {
			t.Errorf("edgeAdjacent(%v, %v) = %v, want %v", a, tt.b, got, tt.want)
		}
	}
}
