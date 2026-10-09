package d2inventory

// The belt of the original game (inventory-trade.md, Inventory.cpp): 16 cells
// laid out row-major, cell = row*4 + column, 4 columns by up to 4 rows. Row 0
// is the row shown in the control panel; the equipped belt decides how many
// rows are usable (belts.txt numboxes: none 4, Sash/Light Belt 8, Belt/Heavy
// Belt 12, Girdle and the exceptional/elite belts 16).

// Belt dimensions.
const (
	BeltColumns = 4
	BeltCells   = 16
	// BeltDefaultBoxes is the size without a belt (belts.txt row "default").
	BeltDefaultBoxes = 4
)

// BeltKinds is the potion kind of every belt cell ("" = empty). The kind is
// the item type of the potion (hpot, mpot, rpot ...): a column holds one kind.
type BeltKinds [BeltCells]string

// BeltRows returns the number of rows of a belt with n boxes.
func BeltRows(boxes int) int {
	rows := (boxes + BeltColumns - 1) / BeltColumns
	if rows < 1 {
		return 1
	}

	if rows > BeltCells/BeltColumns {
		return BeltCells / BeltColumns
	}

	return rows
}

// BeltCell returns the row and column of a cell.
func BeltCell(cell int) (row, col int) { return cell / BeltColumns, cell % BeltColumns }

// FindBeltSlot picks the cell a potion of the given kind goes to
// (INV_FindBeltSlotForItem, 0x63d700): a column that already starts with the
// same kind takes it in its first free cell further up; otherwise the first
// empty cell of the front row. The order of the columns (left to right) and
// the fall-through when a matching column is full are UNVERIFIED. ok is false
// when no cell is free within boxes.
func FindBeltSlot(kinds *BeltKinds, boxes int, kind string) (cell int, ok bool) {
	rows := BeltRows(boxes)

	for col := 0; col < BeltColumns; col++ {
		if kinds[col] != kind {
			continue
		}

		for row := 1; row < rows; row++ {
			if c := row*BeltColumns + col; c < boxes && kinds[c] == "" {
				return c, true
			}
		}
	}

	for col := 0; col < BeltColumns; col++ {
		if col < boxes && kinds[col] == "" {
			return col, true
		}
	}

	return 0, false
}

// FrontOfColumn returns the cell a hotkey (column 0..3) uses: the lowest row
// of the column that holds something.
func FrontOfColumn(kinds *BeltKinds, boxes, col int) (cell int, ok bool) {
	if col < 0 || col >= BeltColumns {
		return 0, false
	}

	for row := 0; row < BeltRows(boxes); row++ {
		if c := row*BeltColumns + col; c < boxes && kinds[c] != "" {
			return c, true
		}
	}

	return 0, false
}

// CompactColumn moves every item of a column down to fill gaps, keeping their
// order, and returns the moves as (from, to) cell pairs so the caller can move
// the items the same way. The original shows the next potion in front after
// one is used (UNVERIFIED how it is stored; the cells of the saved belt are
// kept as they are until a potion is used).
func CompactColumn(kinds *BeltKinds, boxes, col int) (moves [][2]int) {
	to := 0

	for row := 0; row < BeltRows(boxes); row++ {
		from := row*BeltColumns + col
		if from >= boxes || kinds[from] == "" {
			continue
		}

		dest := to*BeltColumns + col
		if dest != from {
			kinds[dest], kinds[from] = kinds[from], ""
			moves = append(moves, [2]int{from, dest})
		}

		to++
	}

	return moves
}
