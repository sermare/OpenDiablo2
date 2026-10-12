package d2automap

import "testing"

// The distance steps compare the halved isometric metric with 50 and 100 (0x454cf5, 0x454cfe), VERIFIED in the
// disassembly; straight below the hero the metric equals the vertical distance.
func TestCellTransparencySteps(t *testing.T) {
	const w, h = 800, 600

	for dy, want := range map[int]Transparency{0: TransVeryLight, 49: TransVeryLight, 50: TransLight, 99: TransLight,
		100: TransMedium, 140: TransMedium} {
		if got := CellTransparency(true, SizeFull, w/2, h/2-10+dy, w, h, 0); got != want {
			t.Errorf("dy %d: got %d want %d", dy, got, want)
		}
	}
}
