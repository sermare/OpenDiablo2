package d2skills

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
)

func TestSlowTargetAmount(t *testing.T) {
	cases := []struct {
		name         string
		stat         int
		player, boss bool
		flags        uint16
		want         int
	}{
		{"none", 0, false, false, 0, 0},
		{"normal monster under cap", 40, false, false, 0, 40},
		{"normal monster capped", 120, false, false, 0, 90},
		{"player capped", 80, true, false, 0, 50},
		{"boss capped", 80, false, true, 0, 50},
		{"champion capped", 80, false, false, d2mapentity.MonTypeChampion, 50},
		{"unique capped", 80, false, false, d2mapentity.MonTypeUnique, 50},
		{"super unique capped", 80, false, false, d2mapentity.MonTypeSuperUnique, 75},
		{"super unique under cap", 60, false, false, d2mapentity.MonTypeSuperUnique, 60},
		{"minion is ordinary", 85, false, false, d2mapentity.MonTypeMinion, 85},
	}

	for _, c := range cases {
		if got := slowTargetAmount(c.stat, c.player, c.boss, c.flags); got != c.want {
			t.Errorf("%s: got %d want %d", c.name, got, c.want)
		}
	}
}
