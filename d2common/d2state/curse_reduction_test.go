package d2state

import "testing"

func TestCurseStatReduction(t *testing.T) {
	base := func(v int) func(int) int { return func(int) int { return v } }

	tests := []struct {
		name   string
		stat   string
		value  int
		player bool
		base   int
		want   int
	}{
		{"lower resist on immune monster", "fireresist", -50, false, 100, -10},
		{"amplify damage on physical immune", "damageresist", -100, false, 100, -20},
		{"truncates towards zero", "coldresist", -9, false, 120, -1},
		{"not immune, unchanged", "fireresist", -50, false, 99, -50},
		{"player unchanged", "fireresist", -50, true, 100, -50},
		{"positive unchanged", "fireresist", 50, false, 100, 50},
		{"zero stays zero", "poisonresist", 0, false, 100, 0},
		{"other stat unchanged", "damagepercent", -50, false, 100, -50},
		{"lightning", "lightresist", -25, false, 100, -5},
		{"magic", "magicresist", -25, false, 100, -5},
	}

	for _, tc := range tests {
		if got := CurseStatReduction(tc.stat, tc.value, tc.player, base(tc.base)); got != tc.want {
			t.Errorf("%s: got %d, want %d", tc.name, got, tc.want)
		}
	}
}

func TestReduceCurseModsCopies(t *testing.T) {
	in := []StatMod{{Stat: "fireresist", Value: -50}, {Stat: "velocity", Value: -30}}
	out := ReduceCurseMods(in, false, func(int) int { return 100 })

	if out[0].Value != -10 || out[1].Value != -30 || in[0].Value != -50 {
		t.Errorf("out %v in %v", out, in)
	}
}

func TestCurseNullified(t *testing.T) {
	imm := func(int) int { return 100 }
	orig := []StatMod{{Stat: "fireresist", Value: -4}}

	if !CurseNullified(orig, ReduceCurseMods(orig, false, imm)) {
		t.Error("-4 on an immune monster is -4/5 = 0: no curse")
	}

	orig = []StatMod{{Stat: "fireresist", Value: -5}}
	if CurseNullified(orig, ReduceCurseMods(orig, false, imm)) {
		t.Error("-5 gives -1: the curse applies")
	}

	orig = []StatMod{{Stat: "fireresist", Value: -4}}
	if CurseNullified(orig, ReduceCurseMods(orig, false, func(int) int { return 50 })) {
		t.Error("not immune: unchanged")
	}

	if !CurseableMonster(true) || CurseableMonster(false) || WalkModeIndex != 2 {
		t.Error("walk mode gate")
	}
}

func TestBaseResistPerStat(t *testing.T) {
	// only the immune element is reduced
	base := func(i int) int { return [6]int{0, 0, 100, 0, 0, 0}[i] }

	if got := CurseStatReduction("fireresist", -50, false, base); got != -10 {
		t.Errorf("fire: %d", got)
	}

	if got := CurseStatReduction("coldresist", -50, false, base); got != -50 {
		t.Errorf("cold: %d", got)
	}
}
