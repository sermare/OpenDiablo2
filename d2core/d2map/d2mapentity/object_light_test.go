package d2mapentity

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2records"
)

func TestObjectLightRadius(t *testing.T) {
	rec := &d2records.ObjectDetailRecord{LightDiameter: [8]int{0, 19, 17, 0, 0, 0, 0, 0}}
	ob := &Object{objectRecord: rec}

	for _, tc := range []struct {
		mode d2enum.ObjectAnimationMode
		want int
	}{
		{d2enum.ObjectAnimationModeNeutral, 0},
		{d2enum.ObjectAnimationModeOperating, 19},
		{d2enum.ObjectAnimationModeOpened, 17},
		{d2enum.ObjectAnimationModeSpecial1, 0},
	} {
		ob.mode = tc.mode
		if got := ob.LightRadius(); got != tc.want {
			t.Errorf("mode %v: radius %d want %d", tc.mode, got, tc.want)
		}
	}

	if (&Object{}).LightRadius() != 0 {
		t.Error("object without a record must not light")
	}
}
