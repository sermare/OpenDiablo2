package d2player

import (
	"reflect"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2skill"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2skilldesc"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2statlist"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2summon"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2hero"
)

const heroMonstats = "Id\thcIdx\tAI\tVelocity\tRun\tminHP\tmaxHP\tAC\tA1MinD\tA1MaxD\tA1TH\t" +
	"MinHP(N)\tMaxHP(N)\tAC(N)\tA1MinD(N)\tA1MaxD(N)\tA1TH(N)\tMinHP(H)\tMaxHP(H)\tAC(H)\tA1MinD(H)\tA1MaxD(H)\tA1TH(H)\n" +
	"claygolem\t289\tNecroPet\t8\t8\t100\t100\t100\t2\t5\t40\t175\t175\t100\t2\t6\t66\t275\t275\t100\t3\t7\t92\n"

func heroTemplates(t *testing.T) *d2summon.Templates {
	t.Helper()

	tpl, err := d2summon.LoadTemplates([]byte(heroMonstats))
	if err != nil {
		t.Fatal(err)
	}

	return tpl
}

// fakeHero is a hero whose last RecalcStats produced the given main-hand
// damage (nil totals: stats never computed).
func fakeHero(dmin, dmax, diff int, computed bool) *d2hero.HeroStatsState {
	st := &d2hero.HeroStatsState{Level: 20, Difficulty: diff}
	if computed {
		st.Totals = &d2statlist.Totals{DamageMin: dmin, DamageMax: dmax}
	}

	return st
}

func idTr(s string) string { return s }

func TestHeroInputsFrom(t *testing.T) {
	div := func(d int) int { return []int{1, 2, 4}[d] }

	tests := []struct {
		name string
		st   *d2hero.HeroStatsState
		want heroInputs
	}{
		{"nil hero", nil, heroInputs{}},
		{"stats never computed", fakeHero(0, 0, 0, false), heroInputs{CurseDiv: 1}},
		{"fists (engine floor 1-2)", fakeHero(1, 2, 0, true), heroInputs{Weapon: [2]int{1, 2}, CurseDiv: 1}},
		{"two-hand weapon with ED, nightmare", fakeHero(60, 130, 1, true),
			heroInputs{Weapon: [2]int{60, 130}, Diff: d2summon.Nightmare, CurseDiv: 2}},
		{"hell", fakeHero(5, 9, 2, true), heroInputs{Weapon: [2]int{5, 9}, Diff: d2summon.Hell, CurseDiv: 4}},
		{"bad difficulty falls back to normal", fakeHero(5, 9, 9, true), heroInputs{Weapon: [2]int{5, 9}, CurseDiv: 1}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var cd func(int) int
			if tc.st != nil {
				cd = div
			}

			if got := heroInputsFrom(tc.st, nil, cd); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got %+v want %+v", got, tc.want)
			}
		})
	}
}

func oneRowTexts(t *testing.T, sk *d2skill.Skill, rows []d2skilldesc.Row, level int, in heroInputs) []string {
	t.Helper()

	reg := d2skill.NewRegistry()
	reg.Add(sk)

	desc := d2skilldesc.Desc{Lines: rows}

	return skillDescTexts(idTr, reg, sk, desc, level, 20, map[int]int{sk.ID: level}, nil, in).Current
}

// Kind 9: weapon x SrcDam/128 + the skill's own damage, with the weapon range
// the engine computed for the hero.
func TestKind9WeaponFromHero(t *testing.T) {
	rows := []d2skilldesc.Row{{Kind: d2skilldesc.KindPhysDamage}}

	tests := []struct {
		name   string
		srcDam int
		hero   *d2hero.HeroStatsState
		want   string
	}{
		{"fists, full weapon share", 128, fakeHero(1, 2, 0, true), "Damage: 3-6"}, // +2..4 skill damage
		{"one-hand, half weapon", 64, fakeHero(10, 20, 0, true), "Damage: 7-14"},  // 5+2 .. 10+4
		{"two-hand enhanced", 128, fakeHero(60, 130, 0, true), "Damage: 62-134"},  // 60+2 .. 130+4
		{"item damage is in the hero range", 128, fakeHero(25, 41, 0, true), "Damage: 27-45"},
		{"stats not computed: skill damage only", 128, fakeHero(0, 0, 0, false), "Damage: 2-4"},
		{"no SrcDam: weapon ignored", 0, fakeHero(60, 130, 0, true), "Damage: 2-4"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sk := &d2skill.Skill{ID: 1, Name: "Strike", MaxLvl: 5}
			sk.HitShift, sk.MinDam, sk.MaxDam, sk.SrcDam = 8, 2, 4, tc.srcDam

			got := oneRowTexts(t, sk, rows, 1, heroInputsFrom(tc.hero, nil, nil))
			if want := []string{"Current Skill Level: 1", tc.want}; !reflect.DeepEqual(got, want) {
				t.Fatalf("got %v want %v", got, want)
			}
		})
	}
}

// Kind 13: "Life: " + avg + avg*calcA/100 + calcB of the summoned monster at
// the hero's difficulty.
func TestKind13SummonLife(t *testing.T) {
	tpl := heroTemplates(t)
	rows := []d2skilldesc.Row{{Kind: d2skilldesc.KindLife, CalcA: "50", CalcB: "10"}}

	tests := []struct {
		name   string
		summon string
		diff   int
		want   []string
	}{
		{"normal", "claygolem", 0, []string{"Current Skill Level: 1", "Life: 160"}}, // 100+50+10
		{"nightmare", "claygolem", 1, []string{"Current Skill Level: 1", "Life: 272"}},
		{"unknown monster: no line", "ghost", 0, nil},
		{"skill without summon: no line", "", 0, nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sk := &d2skill.Skill{ID: 1, Name: "Golem", MaxLvl: 5, Summon: tc.summon}

			got := oneRowTexts(t, sk, rows, 1, heroInputsFrom(fakeHero(1, 2, tc.diff, true), tpl, nil))
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}

	// no templates (monstats missing): the row gives no line
	sk := &d2skill.Skill{ID: 1, Name: "Golem", MaxLvl: 5, Summon: "claygolem"}
	if got := oneRowTexts(t, sk, rows, 1, heroInputs{}); got != nil {
		t.Fatalf("no templates: %v", got)
	}
}

// Kind 31: calcA frames divided by the difficulty's curse divisor (source
// column UNVERIFIED), shown as seconds.
func TestKind31CurseDivisor(t *testing.T) {
	rows := []d2skilldesc.Row{{Kind: d2skilldesc.KindCurseDuration, TextA: "Dur: ", CalcA: "200"}}
	div := func(d int) int { return []int{1, 2, 4}[d] }
	sk := &d2skill.Skill{ID: 1, Name: "Dim Vision", MaxLvl: 5}

	var lines [3][]string

	for diff := 0; diff < 3; diff++ {
		lines[diff] = oneRowTexts(t, sk, rows, 1, heroInputsFrom(fakeHero(1, 2, diff, true), nil, div))
		if len(lines[diff]) != 2 {
			t.Fatalf("difficulty %d: %v", diff, lines[diff])
		}
	}

	if lines[0][1] == lines[1][1] || lines[1][1] == lines[2][1] {
		t.Fatalf("divisor not applied: %v", lines)
	}

	// halving the frames equals dividing by 2
	half := []d2skilldesc.Row{{Kind: d2skilldesc.KindCurseDuration, TextA: "Dur: ", CalcA: "100"}}
	if got := oneRowTexts(t, sk, half, 1, heroInputsFrom(fakeHero(1, 2, 0, true), nil, div)); got[1] != lines[1][1] {
		t.Fatalf("200/2 = %q but 100/1 = %q", lines[1][1], got[1])
	}
}

// A hero with no weapon and no items must leave every non-damage line as it
// was when the tooltip passed no hero values at all.
func TestNoWeaponNoItemsKeepsNonDamageLines(t *testing.T) {
	sk := &d2skill.Skill{ID: 1, Name: "Jab", MaxLvl: 5}
	sk.Params[3], sk.Params[8] = 10, 7

	rows := []d2skilldesc.Row{
		{Kind: 2, TextA: "Dmg: ", TextB: "%", CalcA: "par3"},
		{Kind: 3, TextA: "Rad: ", CalcA: "par8"},
		{Kind: 15, TextA: "A", TextB: "B"},
		{Kind: 16, TextB: "x", CalcA: "50", CalcB: "75"},
		{Kind: 18, TextA: "wrapped"},
		{Kind: 63, TextA: "Syn", TextB: "Dmg", CalcA: "par8"},
		{Kind: 99},
	}

	before := oneRowTexts(t, sk, rows, 2, heroInputs{})

	for _, st := range []*d2hero.HeroStatsState{nil, fakeHero(0, 0, 0, false), fakeHero(1, 2, 0, true)} {
		got := oneRowTexts(t, sk, rows, 2, heroInputsFrom(st, nil, nil))
		if !reflect.DeepEqual(got, before) {
			t.Fatalf("hero %+v changed non-damage lines:\n got %v\nwant %v", st, got, before)
		}
	}

	if len(before) < 4 {
		t.Fatalf("expected several lines, got %v", before)
	}
}
