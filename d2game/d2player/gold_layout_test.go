package d2player

import "testing"

func TestGoldLayout(t *testing.T) {
	tests := []struct {
		m    Mode
		k    GoldKind
		open bool
		want GoldLayout
	}{
		{Mode800, GoldStash, false, GoldLayout{TextY: 296, ButtonX: 155, ButtonY: 296, TooltipY: 278, HitW: 20, HitH: 18, TooltipText: 0x101c}},
		{Mode800, GoldStash, true, GoldLayout{TextY: 600 - 60 - 0x1b8, ButtonX: 155, ButtonY: 600 - 60 - 0x1b6 - 2, TooltipY: 600 - 60 - 0x1ca, HitW: 20, HitH: 18, TooltipText: 0x101c}},
		{Mode800, GoldCarried, false, GoldLayout{TextY: 468, ButtonX: 484, ButtonY: 469}},
		{Mode640, GoldCarried, false, GoldLayout{TextY: 480 - 0x48, ButtonX: 640 - 0xed + 1, ButtonY: 480 - 0x45 - 2}},
		{Mode800, GoldTrade, false, GoldLayout{TextY: 600 - 60 - 0x6a}},
	}

	for _, c := range tests {
		if got := c.m.Gold(c.k, c.open); got != c.want {
			t.Errorf("%v kind %d open %v: got %+v want %+v", c.m, c.k, c.open, got, c.want)
		}
	}
}
