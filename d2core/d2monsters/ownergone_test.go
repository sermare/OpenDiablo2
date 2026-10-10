package d2monsters

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2hero"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
)

func TestOwnerGone(t *testing.T) {
	alive := &d2mapentity.Player{Stats: &d2hero.HeroStatsState{Health: 10}}
	dead := &d2mapentity.Player{Stats: &d2hero.HeroStatsState{Health: 0}}

	for _, tc := range []struct {
		name string
		a    *allyState
		want bool
	}{
		{"nil state", nil, false},
		{"owner alive", &allyState{kind: "minion", owner: alive}, false},
		{"owner dead", &allyState{kind: "minion", owner: dead}, true},
		{"no owner", &allyState{kind: "minion"}, false},
		{"owner without stats", &allyState{kind: "minion", owner: &d2mapentity.Player{}}, false},
		{"totem released with its owner (exe 0x573980)", &allyState{kind: "totem", owner: dead}, true},
		{"totem of a live owner", &allyState{kind: "totem", owner: alive}, false},
		{"trap released with its owner (exe 0x573980)", &allyState{kind: "trap", owner: dead}, true},
		{"unknown kind kept", &allyState{kind: "hireling", owner: dead}, false},
	} {
		if got := ownerGone(tc.a); got != tc.want {
			t.Errorf("%s: got %v want %v", tc.name, got, tc.want)
		}
	}
}
