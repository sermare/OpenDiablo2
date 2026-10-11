package d2player

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2display"
)

func TestNoticePlacement(t *testing.T) {
	for _, tc := range []struct {
		screen     d2display.Size
		textW      int
		wantX, w   int
		wantScreen bool
	}{
		{d2display.Size{W: 800, H: 600}, 100, 342, 116, true}, // the legacy position
		{d2display.Size{W: 1920, H: 1080}, 100, 902, 116, true},
		{d2display.Size{W: 5120, H: 1440}, 300, 2402, 316, true},
	} {
		x, y, w := NoticePlacement(tc.screen, tc.textW)
		if x != tc.wantX || w != tc.w || y != noticeTop {
			t.Errorf("%v: got x=%d y=%d w=%d, want x=%d y=%d w=%d", tc.screen, x, y, w, tc.wantX, noticeTop, tc.w)
		}

		if left, right := x, tc.screen.W-(x+w); left-right > 1 || right-left > 1 {
			t.Errorf("%v: not centred on the screen (%d left, %d right)", tc.screen, left, right)
		}
	}
}
