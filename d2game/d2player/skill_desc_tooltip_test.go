package d2player

import (
	"reflect"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2skill"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2skilldesc"
)

func TestSkillDescTexts(t *testing.T) {
	reg := d2skill.NewRegistry()
	sk := &d2skill.Skill{ID: 1, Name: "Jab", MaxLvl: 5}
	sk.Params[3], sk.Params[4], sk.Params[8] = 10, 2, 7
	sk.HitShift, sk.MinDam, sk.MaxDam = 8, 2, 4
	reg.Add(sk)

	desc := d2skilldesc.Desc{
		DescDam: 1,
		Lines:   []d2skilldesc.Row{{Kind: 2, TextA: "Dmg: ", TextB: "%", CalcA: "ln34"}, {Kind: 99}},
		Lines3:  []d2skilldesc.Row{{Kind: 63, TextA: "Syn", TextB: "Dmg", CalcA: "par8"}},
	}
	tr := func(s string) string { return s }

	tests := []struct {
		name   string
		points int
		want   descTexts
	}{
		{"level 3", 3, descTexts{
			Current: []string{"Damage: 2 to 4", "Dmg: +14%"},
			Next:    []string{"Next Level:", "Damage: 2 to 4", "Dmg: +16%"},
			Synergy: []string{"Syn: +7% Dmg"},
		}},
		{"not learned shows level 1 next", 0, descTexts{
			Next:    []string{"Next Level:", "Damage: 2 to 4", "Dmg: +10%"},
			Synergy: []string{"Syn: +7% Dmg"},
		}},
		{"at the cap there is no next block", 5, descTexts{
			Current: []string{"Damage: 2 to 4", "Dmg: +18%"},
			Synergy: []string{"Syn: +7% Dmg"},
		}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := skillDescTexts(tr, reg, sk, desc, tc.points, 20, map[int]int{1: tc.points})
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got %+v\nwant %+v", got, tc.want)
			}
		})
	}

	if got := skillDescTexts(tr, reg, nil, desc, 1, 1, nil); !reflect.DeepEqual(got, descTexts{}) {
		t.Fatalf("nil skill: %+v", got)
	}
}
