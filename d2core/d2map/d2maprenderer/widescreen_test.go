package d2maprenderer

import (
	"math"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2vector"
)

func testViewport(w, h int, camX, camY float64) *Viewport {
	v := NewViewport(0, 0, w, h)
	pos := d2vector.NewPositionTile(camX, camY)
	v.SetCamera(&Camera{position: &pos})

	return v
}

// At 800x600 the tile range is exactly the one the renderer always used.
func TestTileRangeLegacy800(t *testing.T) {
	v := testViewport(800, 600, 400, 400)
	sx, sy := v.ScreenToWorld(400, -200)
	ex, ey := v.ScreenToWorld(400, 1050)

	gx0, gy0, gx1, gy1 := TileRange(v, 800, 600, 5000, 5000)

	want := [4]int{int(math.Floor(sx)), int(math.Floor(sy)), int(math.Ceil(ex)), int(math.Ceil(ey))}
	if got := [4]int{gx0, gy0, gx1, gy1}; got != want {
		t.Errorf("got %v want %v", got, want)
	}
}

// Every screen corner (and the panel-shifted views) lies inside the tile range of any size.
func TestTileRangeCoversScreen(t *testing.T) {
	sizes := [][2]int{{800, 600}, {1512, 982}, {1920, 1080}, {2560, 1440}, {3440, 1440}, {5120, 1440}, {1980, 1800}}

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

			for _, p := range [][2]int{{0, 0}, {w, 0}, {0, h}, {w, h}, {w / 2, h / 2}} {
				wx, wy := v.ScreenToWorld(p[0], p[1])
				wx, wy = math.Max(wx, 0), math.Max(wy, 0) // the range is clamped to the map
				if wx < float64(x0) || wx > float64(x1) || wy < float64(y0) || wy > float64(y1) {
					t.Errorf("%dx%d align %d: screen %v -> world (%.1f,%.1f) outside [%d,%d]x[%d,%d]",
						w, h, align, p, wx, wy, x0, x1, y0, y1)
				}
			}
		}
	}
}

// The free half beside an open panel stays 400 wide and centred on W/2 +- 200.
func TestViewportAlignWide(t *testing.T) {
	v := testViewport(1920, 1080, 10, 10)
	v.toLeft()

	if got := v.screenRect.Left + v.screenRect.Width/2; got != 960+200 {
		t.Errorf("left align centre %d", got)
	}

	v.resetAlign()
	v.toRight()

	if got := v.screenRect.Left + v.screenRect.Width/2; got != 960-200 {
		t.Errorf("right align centre %d", got)
	}

	v.Resize(2560, 1440)

	if got := v.screenRect.Left + v.screenRect.Width/2; got != 1280-200 {
		t.Errorf("after resize centre %d", got)
	}
}
