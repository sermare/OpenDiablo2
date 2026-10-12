package d2hero

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2statlist"
)

// A weapon at 0 durability has no broken flag (ITEM_RollDurabilityLoss 0x557d90 sets it on armor only), so its
// stat item stays on; armor at 0 is off. VERIFIED in the exe by the w46 durability audit, applied to the stat list
// in w49.
func TestStatItemWeaponAtZeroDurabilityStaysOn(t *testing.T) {
	bases := d2statlist.Bases{
		"ssd": {Weapon: &d2statlist.WeaponBase{}},
		"cap": {},
	}

	for _, tc := range []struct {
		code   string
		broken bool
	}{{"ssd", false}, {"cap", true}} {
		it := &d2s.Item{Code: tc.code, MaxDurability: 20, Durability: 0}
		if got := statItem(it, bases, false, nil).Broken; got != tc.broken {
			t.Errorf("%s at 0 durability: Broken = %v, want %v", tc.code, got, tc.broken)
		}
	}
}
