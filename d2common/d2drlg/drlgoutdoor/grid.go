package drlgoutdoor

import "fmt"

// Grid ops of DRLG_ApplyGridOp (table 0x745720).
const (
	opOr = iota
	opAnd
	opXor
	opAssign
	opAssignIfZero
	opAndNot
)

// Grid is the original's DrlgGrid: a flat row-major array with stride W whose
// Get/Set have no bounds check on x. A coordinate with x >= W reads the next
// row (the cave-entrance search relies on that); only an index outside the
// allocation panics, where the original would touch foreign heap memory.
type Grid struct {
	W, H int
	C    []uint32
}

// NewGrid allocates exactly W*H cells like DRLG_InitGrid.
func NewGrid(w, h int) *Grid { return &Grid{W: w, H: h, C: make([]uint32, w*h)} }

func (g *Grid) idx(x, y int) int {
	i := y*g.W + x
	if i < 0 || i >= len(g.C) {
		panic(fmt.Sprintf("drlgoutdoor: grid access (%d,%d) outside %dx%d", x, y, g.W, g.H))
	}

	return i
}

// Get reads a cell with the original's aliasing.
func (g *Grid) Get(x, y int) uint32 { return g.C[g.idx(x, y)] }

// Set writes a cell.
func (g *Grid) Set(x, y int, v uint32) { g.C[g.idx(x, y)] = v }

// Op applies DRLG_ApplyGridOp.
func (g *Grid) Op(x, y int, v uint32, op int) {
	i := g.idx(x, y)
	c := g.C[i]

	switch op {
	case opOr:
		c |= v
	case opAnd:
		c &= v
	case opXor:
		c ^= v
	case opAssign:
		c = v
	case opAssignIfZero:
		if c == 0 {
			c = v
		}
	case opAndNot:
		c &^= v
	}

	g.C[i] = c
}

// In reports whether (x, y) is inside the grid proper.
func (g *Grid) In(x, y int) bool { return x >= 0 && y >= 0 && x < g.W && y < g.H }

// Rows returns the grid as rows [y][x] (used by tests and callers).
func (g *Grid) Rows() [][]uint32 {
	out := make([][]uint32, g.H)
	for y := range out {
		out[y] = append([]uint32(nil), g.C[y*g.W:(y+1)*g.W]...)
	}

	return out
}
