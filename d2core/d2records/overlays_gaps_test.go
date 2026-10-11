package d2records

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2txt"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2util"
)

// overlay.txt: Height1..4 are four columns, and LoopWaitTime, NumDirections and
// LocalBlood are loaded (exe 0x666b10 copies all of them).
func TestOverlayColumns(t *testing.T) {
	data := []byte("Overlay\tHeight1\tHeight2\tHeight3\tHeight4\tLoopWaitTime\tNumDirections\tLocalBlood\n" +
		"a\t10\t20\t30\t40\t5\t8\t2\n")
	r := &RecordManager{}
	r.Logger = d2util.NewLogger()

	if err := overlaysLoader(r, d2txt.LoadDataDictionary(data)); err != nil {
		t.Fatal(err)
	}

	got := r.Layout.Overlays["a"]
	want := OverlayRecord{Height1: 10, Height2: 20, Height3: 30, Height4: 40, LoopWaitTime: 5, NumDirections: 8, LocalBlood: 2}

	if got == nil || got.Height1 != want.Height1 || got.Height2 != want.Height2 || got.Height3 != want.Height3 ||
		got.Height4 != want.Height4 || got.LoopWaitTime != want.LoopWaitTime ||
		got.NumDirections != want.NumDirections || got.LocalBlood != want.LocalBlood {
		t.Fatalf("overlay = %+v, want %+v", got, want)
	}
}

func TestClampCollideType(t *testing.T) {
	tests := []struct{ in, want int }{
		{0, 0}, {1, 1}, {7, 7}, {8, 8}, {9, 8}, {255, 8}, {-3, 0},
	}

	for _, tc := range tests {
		if got := ClampCollideType(tc.in); got != tc.want {
			t.Errorf("ClampCollideType(%d) = %d, want %d", tc.in, got, tc.want)
		}
	}
}
