package d2player

import "fmt"

// layoutRects returns the 16 box rectangles of the belt (Belts.txt: the row of the 800x600 mode) and the pictures
// behind the rows above the front row, as the belt draws them.
func (b *BeltPanel) layoutRects() []UIRect {
	var rs []UIRect

	for cell := 0; cell < len(b.items); cell++ {
		x, y, w, h := b.cellRect(cell)
		rs = append(rs, UIRect{"belt", fmt.Sprintf("cell%d", cell), x, y, w, h})
	}

	if b.background == nil {
		return rs
	}

	_ = b.background.SetCurrentFrame(beltPopFrame)
	w, h := b.background.GetCurrentFrameSize()

	for row := 1; row < b.Rows(); row++ {
		x, bottom := beltPopPosition(row)
		rs = append(rs, UIRect{"belt", fmt.Sprintf("pop_row%d", row), x, bottom - h, w, h})
	}

	return rs
}
