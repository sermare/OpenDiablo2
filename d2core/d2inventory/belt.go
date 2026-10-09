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

// BeltBoxesByType is the numboxes column of Belts.txt, indexed by the belt
// type (the armor.txt "belt" column, read from the base record by
// ITEM_GetBeltType 0x6220b0; VERIFIED). The loader (0x662810) requires 7 rows
// per screen mode, in this order: belt, sash, default, girdle, light belt,
// heavy belt, uber belt. Armor.txt gives Sash 1, Girdle 3, Light Belt 4, Heavy
// Belt 5 and the exceptional/elite belts 6; the plain Belt row has 0.
var BeltBoxesByType = [7]int{12, 8, 4, 16, 8, 12, 16}

// BeltDefaultType is the Belts.txt row used when no belt is equipped
// ("default", 4 boxes; 0x63d700, VERIFIED).
const BeltDefaultType = 2

// BeltBoxes returns the usable cells for a belt type (BeltDefaultType without
// a belt); unknown types fall back to the default row.
func BeltBoxes(beltType int) int {
	if beltType < 0 || beltType >= len(BeltBoxesByType) {
		return BeltBoxesByType[BeltDefaultType]
	}

	return BeltBoxesByType[beltType]
}

// FindBeltSlot is FindBeltSlotOpt with the beltable fallback enabled.
func FindBeltSlot(kinds *BeltKinds, boxes int, kind string) (cell int, ok bool) {
	return FindBeltSlotOpt(kinds, boxes, kind, true)
}

// FindBeltSlotOpt picks the cell a potion of the given kind goes to
// (INV_FindBeltSlotForItem, 0x63d700, VERIFIED). Columns are scanned left to
// right (0..3). A column whose front cell (row 0) holds a compatible item
// (0x628c00: same base item, or both in the same item-type group) and is inside
// numboxes takes the first free cell going down the column in steps of 4 below
// numboxes; when that column is full the scan continues with the next column.
// If no column took it, and the item's base record has the beltable flag
// (+0x131; fallback), the first empty cell among the first four is used. The
// original leaves the output untouched (no slot) otherwise. Only items 1x1
// and of beltable type are accepted by the original; the caller checks that.
func FindBeltSlotOpt(kinds *BeltKinds, boxes int, kind string, beltableFallback bool) (cell int, ok bool) {
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

	if !beltableFallback {
		return 0, false
	}

	for col := 0; col < BeltColumns; col++ {
		if kinds[col] == "" {
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
