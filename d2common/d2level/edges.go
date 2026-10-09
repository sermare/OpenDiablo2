package d2level

import "math"

// edges.go: walking across the seamless border between two neighbouring Act 1
// outdoor levels. In the original the levels sit next to each other in one
// world (drlgworld places their rectangles) and the hero just walks from one
// into the other; the engine here builds one level at a time, so crossing the
// shared border is a level change whose arrival is the same world position.

// Rect is a level rectangle in world tiles (the drlgworld placement).
type Rect struct{ X, Y, W, H int }

// Side is the side of a rectangle.
type Side int

// Sides, in the order of the DRLG adjacency direction (west, north, east, south).
const (
	West Side = iota
	North
	East
	South
)

func (s Side) String() string { return [...]string{"west", "north", "east", "south"}[s] }

// EdgeNeighbors lists the levels that border level on a seamless edge
// (the SourceDRLG links).
func EdgeNeighbors(level int) []int {
	var out []int

	for _, l := range allLinks {
		if l.From == level && l.Kind == KindEdge {
			out = append(out, l.To)
		}
	}

	return out
}

// Border describes the shared border of two rectangles.
type Border struct {
	Side     Side // of the first rectangle that touches the second
	From, To int  // the range along the border (world tiles, From < To) both share
	Pos      int  // world coordinate of the border line (x for west/east, y for north/south)
}

// SharedBorder returns where a touches b (b lies beyond the returned side of
// a), or false if the rectangles do not touch along a segment.
func SharedBorder(a, b Rect) (Border, bool) {
	overlap := func(a0, a1, b0, b1 int) (int, int, bool) {
		lo, hi := a0, a1
		if b0 > lo {
			lo = b0
		}

		if b1 < hi {
			hi = b1
		}

		return lo, hi, lo < hi
	}

	if lo, hi, ok := overlap(a.Y, a.Y+a.H, b.Y, b.Y+b.H); ok {
		if a.X == b.X+b.W {
			return Border{West, lo, hi, a.X}, true
		}

		if b.X == a.X+a.W {
			return Border{East, lo, hi, b.X}, true
		}
	}

	if lo, hi, ok := overlap(a.X, a.X+a.W, b.X, b.X+b.W); ok {
		if a.Y == b.Y+b.H {
			return Border{North, lo, hi, a.Y}, true
		}

		if b.Y == a.Y+a.H {
			return Border{South, lo, hi, b.Y}, true
		}
	}

	return Border{}, false
}

// EdgeExit reports the neighbour level the hero at world position (wx, wy) is
// about to walk into: the hero is within margin tiles of a border shared with
// a neighbouring level's rectangle and inside the shared range.
func EdgeExit(rects map[int]Rect, from int, wx, wy, margin float64) (to int, ok bool) {
	a, have := rects[from]
	if !have {
		return 0, false
	}

	for _, n := range EdgeNeighbors(from) {
		b, have := rects[n]
		if !have {
			continue
		}

		bd, touch := SharedBorder(a, b)
		if !touch {
			continue
		}

		along, across := wy, wx
		if bd.Side == North || bd.Side == South {
			along, across = wx, wy
		}

		if along < float64(bd.From) || along >= float64(bd.To) {
			continue
		}

		var dist float64

		switch bd.Side {
		case West, North:
			dist = across - float64(bd.Pos)
		default:
			dist = float64(bd.Pos) - across
		}

		if dist <= margin {
			return n, true
		}
	}

	return 0, false
}

// EdgeArrival returns the world position of a hero who leaves from at (wx, wy)
// across the border into to: the same position along the border, inset tiles
// inside to.
func EdgeArrival(rects map[int]Rect, from, to int, wx, wy, inset float64) (x, y float64, ok bool) {
	a, okA := rects[from]
	b, okB := rects[to]

	if !okA || !okB {
		return 0, 0, false
	}

	bd, touch := SharedBorder(a, b)
	if !touch {
		return 0, 0, false
	}

	switch bd.Side {
	case West:
		return float64(bd.Pos) - inset, clampRange(wy, bd), true
	case East:
		return float64(bd.Pos) + inset, clampRange(wy, bd), true
	case North:
		return clampRange(wx, bd), float64(bd.Pos) - inset, true
	default:
		return clampRange(wx, bd), float64(bd.Pos) + inset, true
	}
}

func clampRange(v float64, bd Border) float64 {
	return math.Min(math.Max(v, float64(bd.From)+0.5), float64(bd.To)-0.5)
}

// EdgeTarget is the point to walk to for leaving from into to: the middle of
// the shared border, inset tiles inside from. Callers move it to the nearest
// walkable cell.
func EdgeTarget(rects map[int]Rect, from, to int, inset float64) (x, y float64, ok bool) {
	a, okA := rects[from]
	b, okB := rects[to]

	if !okA || !okB {
		return 0, 0, false
	}

	bd, touch := SharedBorder(a, b)
	if !touch {
		return 0, 0, false
	}

	mid := float64(bd.From+bd.To) / 2

	switch bd.Side {
	case West:
		return float64(bd.Pos) + inset, mid, true
	case East:
		return float64(bd.Pos) - inset, mid, true
	case North:
		return mid, float64(bd.Pos) + inset, true
	default:
		return mid, float64(bd.Pos) - inset, true
	}
}
