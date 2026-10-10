package d2gamescreen

import (
	"reflect"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapengine"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
)

func TestApplyPlannedRank(t *testing.T) {
	const (
		mods   = d2mapentity.MonTypeModsRolled
		unique = d2mapentity.MonTypeUnique
		champ  = d2mapentity.MonTypeChampion
		minion = d2mapentity.MonTypeMinion
		super  = d2mapentity.MonTypeSuperUnique
	)

	cases := []struct {
		name  string
		pm    d2mapengine.PlannedMonster
		flags uint16
		mods  []int
		key   string
		idx   int
	}{
		{"plain monster", d2mapengine.PlannedMonster{Key: "zombie1"}, 0, nil, "", 0},
		{"rare leader", d2mapengine.PlannedMonster{Unique: true, Mods: []int{9}}, unique | mods, []int{9}, "", 0},
		{"champion leader", d2mapengine.PlannedMonster{Unique: true, Champion: true, Mods: []int{16}}, champ | mods, []int{16}, "", 0},
		{"champion minion", d2mapengine.PlannedMonster{Champion: true, Minion: true, Mods: []int{16}}, champ | minion, []int{16}, "", 0},
		{"rare minion", d2mapengine.PlannedMonster{Minion: true, Mods: []int{9}}, minion, []int{9}, "", 0},
		{"super unique", d2mapengine.PlannedMonster{SuperKey: "Bishibosh", SuperIdx: 0, Mods: []int{8, 9, 22}}, super | mods, []int{8, 9, 22}, "Bishibosh", 0},
		{"super unique follower", d2mapengine.PlannedMonster{Minion: true, Mods: []int{8}}, minion, []int{8}, "", 0},
		{"countess", d2mapengine.PlannedMonster{SuperKey: "The Countess", SuperIdx: 6}, super | mods, nil, "The Countess", 6},
	}

	for _, c := range cases {
		m := &d2mapentity.Monster{}
		applyPlannedRank(m, c.pm)

		if m.TypeFlags != c.flags {
			t.Errorf("%s: type flags %#x, want %#x", c.name, m.TypeFlags, c.flags)
		}

		if !reflect.DeepEqual(m.Modifiers, c.mods) {
			t.Errorf("%s: modifiers %v, want %v", c.name, m.Modifiers, c.mods)
		}

		if m.SuperUnique != c.key || m.SuperUniqueIdx != c.idx {
			t.Errorf("%s: super unique %q/%d, want %q/%d", c.name, m.SuperUnique, m.SuperUniqueIdx, c.key, c.idx)
		}
	}
}
