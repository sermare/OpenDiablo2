package d2monreg

import "github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"

// Placement is the search of FUN_005b0600: a monster is put near a wanted
// subtile by walking the ring of a square around it from a random start,
// every step tested against the room rectangle, the cell, the open-ground
// test and the collision test. The room seed is the random source.
//
// Verified by reading the code and compared with the emulator through the
// creation sequence (class, x, y) and the seeds of natural populations.

// World is what the placement needs from the map.
type World interface {
	// Blocked is FUN_0064ec70: whether a monster of the given radius cannot
	// stand on subtile (x, y) with the collision mask.
	Blocked(room *Room, x, y, radius, mask int) bool
	// Open is FUN_005fc580(x, y, 1).
	Open(room *Room, x, y int) bool
	// CellAt is FUN_0061ae80: the flag value of the cell that holds the point.
	CellAt(room *Room, x, y int) int
	// NearExit is FUN_0054b970: whether the point is too close to a warp.
	NearExit(room *Room, x, y int) bool
}

// Cell is one element of the neighbour block of a room (a rectangle in tiles).
type Cell struct {
	X0, Y0, X1, Y1 int  // +0x10: left, top, right, bottom in tiles
	Skip           bool // +0x20 != 0
	Flag           int  // +0x28
}

// Room is a populated server room (Room1).
type Room struct {
	// Level is the level id of the room. NoPopulate marks the rooms the level
	// does not populate (LvlPrest Populate=0, room flag 0x800000): they are
	// still counted as asking for a population but never get one.
	Level      int
	NoPopulate bool
	Seed       d2rand.Seed // Room1 +0x6c
	// X, Y, W, H is the room rectangle in subtiles (Room1 +0x4c).
	X, Y, W, H int
	Cells      []Cell
}

// Placement is the argument block of FUN_005b0600.
type Placement struct {
	Room       *Room
	Cell       *Cell // non-nil: restricts to the cell and its flag
	Class      int
	X, Y       int
	Ring       int    // P[8]: -1 tests the exact spot only, n searches rings of radius 3..3n
	Flags      uint16 // bit 0: only test, bit 0x80: ignore collision
	Param4     int    // P[4]
	Param5     int    // P[5]
	TestOnly   bool
	IgnoreColl bool
}

const (
	flagTestOnly   = 1
	flagIgnoreColl = 0x80
)

// scaleTiles is DRLG_ScaleCoordsToSubtiles.
func scaleTiles(v int) int { return v * 5 }

func (c *Cell) subRect() (l, t, r, b int) {
	return scaleTiles(c.X0), scaleTiles(c.Y0), scaleTiles(c.X1), scaleTiles(c.Y1)
}

func (t *Tables) collisionMask(class int) (radius, mask int, special bool) {
	if class < 0 || class >= len(t.Mons) {
		return 0, 0x3c01, false
	}

	ex := t.Mons[class].Ex
	if ex < 0 || ex >= len(t.Mon2s) {
		return 0, 0x3c01, false
	}

	m2 := &t.Mon2s[ex]

	switch m2.SpawnCol {
	case 1:
		mask = 0x1c0
	case 2:
		mask = 0x3f11
	case 3:
		mask = 0
	default:
		mask = 0x3c01
	}

	return m2.SizeX, mask, m2.SpawnCol == 1 && (class < 0x102 || class > 0x107) && class != 0x99
}

// Place finds the position FUN_005b0600 would use. ok is false when no
// position was found (the unit is then not created).
func (t *Tables) Place(w World, p Placement) (x, y int, ok bool) {
	radius, mask, special := t.collisionMask(p.Class)
	if special {
		// FUN_005b02f0 picks the spot of these big monsters: not modelled
		return 0, 0, false
	}

	var left, top, right, bottom int

	cellMode := p.Cell != nil
	cellFlag := 0

	if cellMode {
		left, top, right, bottom = p.Cell.subRect()
		cellFlag = p.Cell.Flag
	} else {
		left, top, right, bottom = p.Room.X, p.Room.Y, p.Room.X+p.Room.W, p.Room.Y+p.Room.H
	}

	step, cur, limit := 0, 0, 0

	if p.Ring >= 0 {
		step, cur, limit = 3, 3, p.Ring*3
	}

	if limit < step {
		return 0, 0, false
	}

	s := &p.Room.Seed

	for {
		coin := s.Step()&1 == 0
		var offX, offY int

		if coin {
			offX, offY = int(s.Roll(int32(cur))), cur
		} else {
			offX, offY = cur, int(s.Roll(int32(cur)))
		}

		if s.Step()&1 != 0 {
			offX = -offX
		}

		if s.Step()&1 != 0 {
			offY = -offY
		}

		px, py := p.X+offX, p.Y+offY
		rl, rt := p.X-cur, p.Y-cur
		rr, rb := rl+cur*2, rt+cur*2

		dx, dy := 0, 1
		if coin {
			dx, dy = 1, 0
		}

		n := cur * 8
		first := true

		if n == 0 {
			n = 1
		}

		for ; n > 0; n-- {
			_ = first

			if px == rl && py == rt {
				dx, dy = 1, 0
			}

			if px == rr {
				if py == rt {
					dx, dy = 0, 1
				}

				if py == rb {
					dx, dy = -1, 0
				}
			}

			if px == rl {
				if py == rb {
					dx, dy = 0, -1
				}

				if py == rt && px == rr && py == rb {
					dx, dy = 0, 0
				}
			}

			py += dy
			px += dx

			if px < left || px >= right || py < top || py >= bottom {
				continue
			}

			if cellMode && w.CellAt(p.Room, px, py) != cellFlag {
				continue
			}

			if !w.Open(p.Room, px, py) {
				continue
			}

			if w.Blocked(p.Room, px, py, radius, mask) && p.Flags&flagIgnoreColl == 0 {
				continue
			}

			return px, py, px != 0 && py != 0
		}

		cur += 3
		if cur > limit {
			break
		}
	}

	return 0, 0, false
}
