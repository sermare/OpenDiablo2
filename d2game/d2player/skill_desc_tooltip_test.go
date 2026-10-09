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
		Lines: []d2skilldesc.Row{
			{Kind: d2skilldesc.KindMana},
			{Kind: 2, TextA: "Dmg: ", TextB: "%", CalcA: "ln34"},
			{Kind: 99},
		},
		Lines2: []d2skilldesc.Row{{Kind: 3, TextA: "Rad: ", CalcA: "par8"}},
		Lines3: []d2skilldesc.Row{{Kind: 63, TextA: "Syn", TextB: "Dmg", CalcA: "par8"}},
	}
	// identity translation: every label falls back to the original's English
	tr := func(s string) string { return s }
	mana := func(level int) (string, bool) { return "Mana: " + string(rune('0'+level)), true }

	tests := []struct {
		name   string
		points int
		want   descTexts
	}{
		{"level 3", 3, descTexts{
			Top:     []string{"Rad: 7"},
			Current: []string{"Current Skill Level: 3", "Damage: 2-4", "Mana: 3", "Dmg: +14%"},
			Next:    []string{"Next Level", "Damage: 2-4", "Mana: 4", "Dmg: +16%"},
			Synergy: []string{"Syn: +7% Dmg"},
		}},
		{"not learned shows First Level and no current block", 0, descTexts{
			Top:     []string{"Rad: 7"},
			Next:    []string{"First Level", "Damage: 2-4", "Mana: 1", "Dmg: +10%"},
			Synergy: []string{"Syn: +7% Dmg"},
		}},
		{"at the cap there is no next block", 5, descTexts{
			Top:     []string{"Rad: 7"},
			Current: []string{"Current Skill Level: 5", "Damage: 2-4", "Mana: 5", "Dmg: +18%"},
			Synergy: []string{"Syn: +7% Dmg"},
		}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := skillDescTexts(tr, reg, sk, desc, tc.points, 20, map[int]int{1: tc.points}, mana)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got %+v\nwant %+v", got, tc.want)
			}
		})
	}

	if got := skillDescTexts(tr, reg, nil, desc, 1, 1, nil, nil); !reflect.DeepEqual(got, descTexts{}) {
		t.Fatalf("nil skill: %+v", got)
	}
}

func TestLabelKeys(t *testing.T) {
	tr := func(k string) string { return map[string]string{keyNextLevel: "Prochain niveau"}[k] }

	if got := label(tr, keyNextLevel, "Next Level"); got != "Prochain niveau" {
		t.Fatalf("translated label: %q", got)
	}

	if got := label(tr, keyFirstLevel, "First Level"); got != "First Level" {
		t.Fatalf("fallback label: %q", got)
	}

	// the keys are the original's string-table keys (ids 0x109d, 0x10ad, 0x109e, 0x10a0)
	for key, want := range map[string]string{
		keyNextLevel: "StrSkill1", keyFirstLevel: "StrSkill17", keyCurrentLevel: "StrSkill2", keyDamage: "StrSkill4",
	} {
		if key != want {
			t.Errorf("key %q want %q", key, want)
		}
	}
}
