package d2path

// Collision cell flags (one uint16 per subtile). Only bits whose meaning the
// notes verified are named; see the comment on each.
const (
	// FlagWalk blocks walking: wall, hole or black (VERIFIED: in every block
	// mask; set from tile flag 0x40).
	FlagWalk uint16 = 0x0001
	// FlagWall blocks missiles and line of sight (VERIFIED, from tile flag 0x80).
	FlagWall uint16 = 0x0004
	// FlagPlayerOnly blocks players but not monsters (VERIFIED by comparing
	// the player and monster masks).
	FlagPlayerOnly uint16 = 0x0008
	// FlagMissile marks a missile in the cell (VERIFIED).
	FlagMissile uint16 = 0x0040
	// FlagPlayer marks a player unit footprint (VERIFIED).
	FlagPlayer uint16 = 0x0080
	// FlagMonster marks a monster/NPC unit footprint (VERIFIED).
	FlagMonster uint16 = 0x0100
	// FlagDoor marks a closed door (VERIFIED through the LOS masks 0x804/0x805).
	FlagDoor uint16 = 0x0800
	// FlagCorpse marks a dead monster footprint (VERIFIED).
	FlagCorpse uint16 = 0x8000

	// OutOfGrid is what a read outside the grid returns: treated as fully
	// blocked (VERIFIED, 0x27 = 0x1|0x2|0x4|0x20).
	OutOfGrid uint16 = 0x27
)

// Block masks: the cell flags a unit may not enter (VERIFIED values from
// PATH_AllocUnitPath; the meaning of the bits not named above is UNVERIFIED).
// Note the unit footprint bits 0x80, 0x100, 0x8000 and the missile bit 0x40 are
// in none of these masks.
const (
	MaskPlayer  uint16 = 0x1C09
	MaskMonster uint16 = 0x3C01
	// MaskMonsterOpensDoors is the monster mask when monstats opendoors is set.
	MaskMonsterOpensDoors uint16 = 0x3401
	// MaskFlyer is used for flying/ghost monsters (monstats byte +0xD bit 6;
	// which monsters those are is UNVERIFIED).
	MaskFlyer uint16 = 0x1804
)

// Grid is read-only access to collision flags.
type Grid interface {
	// Flags returns the flags of a subtile; outside the grid it must return
	// OutOfGrid.
	Flags(x, y int) uint16
}

// CellGrid is a rectangular Grid of flags covering [X0,X0+W) x [Y0,Y0+H).
type CellGrid struct {
	X0, Y0, W, H int
	Cells        []uint16
}

// NewCellGrid creates an empty (all walkable) grid.
func NewCellGrid(x0, y0, w, h int) *CellGrid {
	return &CellGrid{X0: x0, Y0: y0, W: w, H: h, Cells: make([]uint16, w*h)}
}

func (g *CellGrid) index(x, y int) (int, bool) {
	x -= g.X0
	y -= g.Y0

	if x < 0 || y < 0 || x >= g.W || y >= g.H {
		return 0, false
	}

	return y*g.W + x, true
}

// Flags implements Grid.
func (g *CellGrid) Flags(x, y int) uint16 {
	i, ok := g.index(x, y)
	if !ok {
		return OutOfGrid
	}

	return g.Cells[i]
}

// Set ORs bits into a cell (COLLISION_SetCellFlags); outside cells are ignored.
func (g *CellGrid) Set(x, y int, bits uint16) {
	if i, ok := g.index(x, y); ok {
		g.Cells[i] |= bits
	}
}

// Clear removes bits from a cell.
func (g *CellGrid) Clear(x, y int, bits uint16) {
	if i, ok := g.index(x, y); ok {
		g.Cells[i] &^= bits
	}
}

// Blocked reports whether a unit with the given block mask cannot stand on
// the subtile (COLLISION_TestFootprintMask for shape 0, a single cell; the
// 3x3 and 5x5 footprint shapes are not ported).
func Blocked(g Grid, x, y int, mask uint16) bool { return g.Flags(x, y)&mask != 0 }
