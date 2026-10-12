package d2maprenderer

import "math"

// sideMargin is how many pixels beside the screen a tile is still drawn: the widest sprites (walls, large
// monsters) reach this far from their tile origin.
const sideMargin = 400

// TileRows narrows the rectangular tile range of TileRange to the tiles that can be seen: per tile row
// (tileY - startY) the half open range [From, To) of tileX whose screen position lies within sideMargin pixels
// of the screen sides and between extraViewTop above and extraViewBottom below it. The rectangle covers the
// whole isometric diamond of the screen plus two empty corners; at 5120x1440 the corners are most of the
// rectangle (docs/DISPLAY.md), so culling per row saves most of the tiles that were looked at and drawn.
// Only tiles that are off screen are removed: every tile whose origin is inside the screen band is kept.
type TileRows struct {
	StartY   int
	From, To []int
}

// Row returns the range of row tileY (empty outside the rows).
func (r *TileRows) Row(tileY int) (from, to int) {
	i := tileY - r.StartY
	if i < 0 || i >= len(r.From) {
		return 0, 0
	}

	return r.From[i], r.To[i]
}

// ComputeTileRows is the pure form: sx0, sy0 is the screen position of tile (0,0), (dxX, dxY) the screen step of
// one tile in x and (dyX, dyY) of one tile in y; bounds are the accepted screen rectangle. The ranges are
// clamped to [startX,endX) x [startY,endY).
func ComputeTileRows(sx0, sy0, dxX, dxY, dyX, dyY float64, minX, minY, maxX, maxY float64,
	startX, startY, endX, endY int) *TileRows {
	rows := &TileRows{StartY: startY}

	n := endY - startY
	if n < 0 {
		n = 0
	}

	rows.From, rows.To = make([]int, n), make([]int, n)

	for i := 0; i < n; i++ {
		ty := float64(startY + i)
		lo, hi := float64(startX), float64(endX-1)
		// screen x = sx0 + tx*dxX + ty*dyX, screen y = sy0 + tx*dxY + ty*dyY: intersect the two bands in tx
		lo, hi = bandInTx(lo, hi, sx0+ty*dyX, dxX, minX, maxX)
		lo, hi = bandInTx(lo, hi, sy0+ty*dyY, dxY, minY, maxY)

		if lo > hi {
			continue // From == To == 0: empty
		}

		rows.From[i], rows.To[i] = int(lo), int(hi)+1
	}

	return rows
}

// bandInTx narrows [lo,hi] to the integer tx with base + tx*step in [minV,maxV].
func bandInTx(lo, hi, base, step, minV, maxV float64) (float64, float64) {
	if step == 0 {
		if base < minV || base > maxV {
			return 1, 0
		}

		return lo, hi
	}

	a, b := (minV-base)/step, (maxV-base)/step
	if a > b {
		a, b = b, a
	}

	return math.Max(lo, math.Ceil(a)), math.Min(hi, math.Floor(b))
}

// VisibleTileRows computes the rows for the viewport's current camera.
func VisibleTileRows(v *Viewport, width, height, startX, startY, endX, endY int) *TileRows {
	x0, y0 := v.WorldToScreenF(0, 0)
	x1, y1 := v.WorldToScreenF(1, 0)
	x2, y2 := v.WorldToScreenF(0, 1)

	return ComputeTileRows(x0, y0, x1-x0, y1-y0, x2-x0, y2-y0,
		-sideMargin, -extraViewTop, float64(width+sideMargin), float64(height+extraViewBottom),
		startX, startY, endX, endY)
}

// TileRef is the position of one visible tile.
type TileRef struct{ X, Y int32 }

// DiagonalOrder lists the visible tiles of rows in depth order: by diagonal x+y (the screen row of an isometric
// tile), then by x. The exe draws its floor and lower wall lists bucket by bucket in depth order (renderer.md b1);
// row-major order (all of y=0 first) put a tile far down the screen before tiles above it. buf is reused when it
// is big enough. The counting sort is linear in the number of tiles.
func (r *TileRows) DiagonalOrder(startY, endY int, buf []TileRef) []TileRef {
	buf = buf[:0]
	minD, maxD, n := math.MaxInt, math.MinInt, 0

	for y := startY; y < endY; y++ {
		from, to := r.Row(y)
		if to <= from {
			continue
		}

		n += to - from

		if d := from + y; d < minD {
			minD = d
		}

		if d := to - 1 + y; d > maxD {
			maxD = d
		}
	}

	if n == 0 {
		return buf
	}

	// start offset of every diagonal; rows are visited by increasing y, so the tiles of one diagonal arrive
	// in decreasing x: fill each diagonal from its end to leave them in increasing x
	ends := make([]int, maxD-minD+2)

	for y := startY; y < endY; y++ {
		from, to := r.Row(y)
		for x := from; x < to; x++ {
			ends[x+y-minD+1]++
		}
	}

	for i := 1; i < len(ends); i++ {
		ends[i] += ends[i-1]
	}

	if cap(buf) < n {
		buf = make([]TileRef, n)
	}

	buf = buf[:n]

	// ends[d+1] is now the end of diagonal d (exclusive): fill backwards
	for y := startY; y < endY; y++ {
		from, to := r.Row(y)
		for x := from; x < to; x++ {
			d := x + y - minD
			ends[d+1]--
			buf[ends[d+1]] = TileRef{int32(x), int32(y)}
		}
	}

	return buf
}
