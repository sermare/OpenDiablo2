// Package d2steal is the belt theft of Game.exe's hit callback at 0x5a0be0
// (VERIFIED by decompile; which monster class carries the callback is
// unverified). A monster whose hit connects with a player may take the first
// potion out of the player's belt: the item is cloned to the ground next to
// the player and removed from the belt, then the belt column is compacted.
//
// The package is pure and not yet wired into the combat code
// (d2core/d2monsters/combat.go has no hit callback table); a caller supplies
// the roll and the belt.
package d2steal

// Belt geometry: sixteen cells, four columns; cell = row*4 + column.
const (
	BeltCells   = 16
	BeltColumns = 4

	// ModeBelt is the item mode of an item that sits in the belt.
	ModeBelt = 2

	// PassPercent is the chance that the theft goes ahead: the roll
	// seed % 100 must be 30 or more (VERIFIED, the exe tests "> 0x1d").
	PassPercent = 70
	rollFloor   = 30
)

// Item is what the theft needs to know of a belt entry.
type Item struct {
	Code string
	Mode int // item mode; ModeBelt when it is really in the belt
}

// Belt is the sixteen cells; a nil entry is an empty cell.
type Belt [BeltCells]*Item

// Passes reports whether a step of the monster's generator lets the theft go
// ahead (70 percent).
func Passes(seedLow uint32) bool { return seedLow%100 >= rollFloor }

// Result is what Steal did.
type Result struct {
	Stolen *Item // the item that left the belt, nil when nothing happened
	Cell   int   // the cell it was taken from
}

// Steal tries the theft against a belt. step is one step of the monster's
// generator (it is consumed even when nothing can be stolen, like the exe);
// targetIsPlayer is the exe's unit-type-0 test; cursorEmpty is true when the
// victim holds nothing on the cursor. Only the first occupied cell (0..15) is
// considered: when that item is not in belt mode, or the cursor is busy,
// nothing is stolen and the belt is left alone.
func Steal(b *Belt, step func() uint32, targetIsPlayer, cursorEmpty bool) Result {
	if !Passes(step()) || !targetIsPlayer {
		return Result{}
	}

	for i, it := range b {
		if it == nil {
			continue
		}

		if it.Mode != ModeBelt || !cursorEmpty {
			return Result{}
		}

		b[i] = nil
		Compact(b, i%BeltColumns)

		return Result{Stolen: it, Cell: i}
	}

	return Result{}
}

// Compact packs the items of one column toward the first row, keeping their
// order (VERIFIED 0x5a0be0 callee INV_CompactBeltColumn).
func Compact(b *Belt, col int) {
	if col < 0 {
		col = 0
	}

	if col > BeltColumns-1 {
		col = BeltColumns - 1
	}

	dst := col

	for src := col; src < BeltCells; src += BeltColumns {
		if b[src] == nil {
			continue
		}

		if dst != src {
			b[dst], b[src] = b[src], nil
		}

		dst += BeltColumns
	}
}
