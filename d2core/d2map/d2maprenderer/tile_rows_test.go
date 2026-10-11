package d2maprenderer

import "testing"

// Culling per row removes only tiles that are off screen: every tile whose origin is inside the accepted
// screen band is kept, and the saving at 5120x1440 is large.
func TestVisibleTileRows(t *testing.T) {
	sizes := [][2]int{{800, 600}, {1512, 982}, {1920, 1080}, {3440, 1440}, {5120, 1440}}

	for _, sz := range sizes {
		for _, align := range []int{center, left, right} {
			w, h := sz[0], sz[1]
			v := testViewport(w, h, 400, 400)

			switch align {
			case left:
				v.toLeft()
			case right:
				v.toRight()
			}

			x0, y0, x1, y1 := TileRange(v, w, h, 5000, 5000)
			rows := VisibleTileRows(v, w, h, x0, y0, x1, y1)
			box, kept := (x1-x0)*(y1-y0), 0

			for ty := y0; ty < y1; ty++ {
				from, to := rows.Row(ty)
				if to > from && (from < x0 || to > x1) {
					t.Fatalf("%dx%d row %d [%d,%d) outside the range [%d,%d)", w, h, ty, from, to, x0, x1)
				}

				kept += to - from

				for tx := x0; tx < x1; tx++ {
					sx, sy := v.WorldToScreenF(float64(tx), float64(ty))
					in := sx >= -sideMargin+1 && sx <= float64(w+sideMargin)-1 && sy >= -extraViewTop+1 && sy <= float64(h+extraViewBottom)-1
					has := tx >= from && tx < to

					if in && !has {
						t.Fatalf("%dx%d align %d: visible tile (%d,%d) at screen (%.0f,%.0f) was culled", w, h, align, tx, ty, sx, sy)
					}

					if has && (sx < -sideMargin-1 || sx > float64(w+sideMargin)+1 || sy < -extraViewTop-1 || sy > float64(h+extraViewBottom)+1) {
						t.Fatalf("%dx%d align %d: off-screen tile (%d,%d) at (%.0f,%.0f) kept", w, h, align, tx, ty, sx, sy)
					}
				}
			}

			if align == center {
				t.Logf("%dx%d: %d tiles in the range rectangle, %d after the per-row cull (%.0f%%)", w, h, box, kept, 100*float64(kept)/float64(box))
			}

			if w == 5120 && kept*10 > box*8 {
				t.Errorf("5120x1440: only %d of %d tiles culled", box-kept, box)
			}
		}
	}
}
