package d2skills

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2records"
)

func TestStateOverlayNames(t *testing.T) {
	cases := []struct {
		name string
		rec  *d2records.StateRecord
		want [4]string
	}{
		{"nil", nil, [4]string{}},
		{"single", &d2records.StateRecord{Overlay1: "stun"}, [4]string{"stun"}},
		{"front back", &d2records.StateRecord{Overlay1: "aura_might_front", Overlay2: "aura_might_back"},
			[4]string{"aura_might_front", "aura_might_back"}},
		{"null skipped", &d2records.StateRecord{Overlay1: "null", Overlay3: "x"}, [4]string{"", "", "x", ""}},
		{"nooverlays", &d2records.StateRecord{Overlay1: "stun", NoOverlays: true}, [4]string{}},
	}

	for _, c := range cases {
		if got := stateOverlayNames(c.rec); got != c.want {
			t.Errorf("%s: %v want %v", c.name, got, c.want)
		}
	}
}

func TestStateOverlayOrder(t *testing.T) {
	cases := []struct {
		slot    int
		preDraw bool
		want    int
	}{
		{0, false, d2mapentity.DrawFront},
		{1, false, d2mapentity.DrawBehind},
		{0, true, d2mapentity.DrawBehind},
		{2, false, d2mapentity.DrawFront},
	}

	for _, c := range cases {
		if got := d2mapentity.StateOverlayOrder(c.slot, c.preDraw); got != c.want {
			t.Errorf("slot %d predraw %v: %d want %d", c.slot, c.preDraw, got, c.want)
		}
	}
}
