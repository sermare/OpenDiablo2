package d2monsters

import (
	"reflect"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2monster"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
)

func TestMemberTypeFlags(t *testing.T) {
	const (
		sup    = d2mapentity.MonTypeSuperUnique
		uni    = d2mapentity.MonTypeUnique
		champ  = d2mapentity.MonTypeChampion
		minion = d2mapentity.MonTypeMinion
	)

	tests := []struct {
		name         string
		plan         d2monster.Pack
		leader, rest uint16
	}{
		{"normal", d2monster.Pack{}, 0, 0},
		{"super unique", d2monster.Pack{SuperUnique: "x"}, sup, minion},
		{"champion", d2monster.Pack{Kind: d2monster.PackChampion}, champ, champ},
		{"unique", d2monster.Pack{Kind: d2monster.PackUnique}, uni, minion},
	}

	for _, tt := range tests {
		if g := memberTypeFlags(tt.plan, true); g != tt.leader {
			t.Errorf("%s leader = %#x, want %#x", tt.name, g, tt.leader)
		}

		if g := memberTypeFlags(tt.plan, false); g != tt.rest {
			t.Errorf("%s follower = %#x, want %#x", tt.name, g, tt.rest)
		}
	}
}

// Champion / unique plans draw the same members and consume the same RNG as
// PlanGroup, so fixed-seed spawn output only differs in TypeFlags.
func TestPlanKindsKeepMembers(t *testing.T) {
	info := d2monster.ClassInfo{Class: 3, MinGrp: 2, MaxGrp: 5, PartyMin: 1, PartyMax: 3, Minion1: 4, Minion2: 5}

	for _, seed := range []uint32{1, 7, 12345} {
		base := d2rand.New(seed)
		want := d2monster.PlanGroup(base, info)
		next := base.Roll(1000)

		for _, plan := range []func(*d2rand.Seed, d2monster.ClassInfo) d2monster.Pack{d2monster.PlanChampion, d2monster.PlanUnique} {
			r := d2rand.New(seed)
			got := plan(r, info)

			if !reflect.DeepEqual(got.Members, want.Members) {
				t.Errorf("seed %d members differ", seed)
			}

			if r.Roll(1000) != next {
				t.Errorf("seed %d RNG state differs", seed)
			}
		}
	}

	// Champions and minions are "plain" for the 0xa defense rule; uniques are not.
	if d2mapentity.MonTypeChampion&0xa != 0 || d2mapentity.MonTypeMinion&0xa != 0 {
		t.Error("champion / minion masks must not overlap 0xa")
	}
}
