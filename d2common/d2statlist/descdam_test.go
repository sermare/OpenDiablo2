package d2statlist

import "testing"

func armedTotals() *Totals {
	return &Totals{Weapon: WeaponParts{Armed: true, Min: 10, Max: 20}, DamagePct: 50}
}

func TestDescDam5(t *testing.T) {
	cases := []struct {
		name string
		t    *Totals
		in   DescDamInput
		want DamageBox
	}{
		// srcdam 0: no weapon part, the skill's own numbers only (>>8 of the elemental part), colour from the etype
		{"spell only", armedTotals(), DescDamInput{DescDam: 5, EType: 1, PhysMin: 3, PhysMax: 5, ElemMin: 10 << 8, ElemMax: 20 << 8},
			DamageBox{Min: 13, Max: 25, Colour: BoxRed}},
		// srcdam 128: weapon (10..20) * 150% = 15..30 added to the skill part
		{"full weapon", armedTotals(), DescDamInput{DescDam: 5, SrcDam: 128, PhysMin: 2, PhysMax: 2},
			DamageBox{Min: 17, Max: 32}},
		// srcdam 64 scales the finished weapon box (15..30) by half, truncating
		{"half weapon", armedTotals(), DescDamInput{DescDam: 5, SrcDam: 64},
			DamageBox{Min: 7, Max: 15}},
		// poison multiplies the elemental part by its length: 2/frame * 100 frames >> 8
		{"poison length", armedTotals(), DescDamInput{DescDam: 5, EType: 5, ElemMin: 2 << 8, ElemMax: 4 << 8, ElemLen: 100},
			DamageBox{Min: 200, Max: 400, Colour: BoxGreen}},
		{"blue state", armedTotals(), DescDamInput{DescDam: 5, EType: 4, DmgBlue: true, SrcDam: 128},
			DamageBox{Min: 15, Max: 30, Colour: BoxBlue}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := tc.t.DescDamBox(tc.in)
			if !ok || got != tc.want {
				t.Fatalf("got %+v ok=%v, want %+v", got, ok, tc.want)
			}
		})
	}
}

func TestDescDam7(t *testing.T) {
	tt := armedTotals()
	tt.Weapon.Flat = 4

	// pool 50 + calc1 20 = 70 -> 170%: 17..34, + flat 4 + calc2 6 = 27..44
	got, ok := tt.DescDamBox(DescDamInput{DescDam: 7, SrcDam: 128, Calc1: 20, Calc2: 6})
	if !ok || got.Min != 27 || got.Max != 44 {
		t.Fatalf("got %+v ok=%v, want 27-44", got, ok)
	}

	// srcdam 64 halves the whole box, then the skill part is added
	got, _ = tt.DescDamBox(DescDamInput{DescDam: 7, SrcDam: 64, Calc1: 20, Calc2: 6, PhysMin: 1, PhysMax: 2, ElemMin: 3 << 8, ElemMax: 3 << 8})
	if got.Min != 13+1+3 || got.Max != 22+2+3 {
		t.Fatalf("scaled: got %+v", got)
	}

	if _, ok := tt.DescDamBox(DescDamInput{DescDam: 3}); ok {
		t.Fatal("descdam 3 is not ported")
	}
}

func TestApplyElemental(t *testing.T) {
	tt := &Totals{}
	tt.Elem.Elem[0] = ElemStat{Min: 10, Max: 20, Pct: 50} // fire +50%: 15..30
	tt.Elem.Elem[2] = ElemStat{Min: 9, Max: 5}            // cold min above max: min drops to 5
	tt.Elem.Poison = ElemStat{Min: 1 << 8, Max: 2 << 8}
	tt.Elem.PoisonLen = 25

	acc := DamageAcc{}
	tt.ApplyElemental(&acc, false, nil, false, false)

	// 15+5 = 20, 30+5 = 35, poison 25*256>>8 = 25 and 50
	if acc.Min != 45 || acc.Max != 85 || acc.Colour != BoxGreen {
		t.Fatalf("got %+v", acc)
	}

	acc = DamageAcc{}
	tt.ApplyElemental(&acc, true, func(a *DamageAcc) { a.Min += 100 }, true, true)

	if acc.Min != 145 || acc.Colour != BoxRed {
		t.Fatalf("state/red: got %+v", acc)
	}

	// a lone max below min + 1 is lifted
	one := &Totals{}
	one.Elem.Elem[1] = ElemStat{Min: 0, Max: 0}
	one.Elem.Elem[3] = ElemStat{Min: 0, Max: 1}
	acc = DamageAcc{}
	one.ApplyElemental(&acc, false, nil, false, false)

	if acc.Min != 1 || acc.Max != 2 {
		t.Fatalf("clamp: got %+v", acc)
	}
}

func TestMinMaxText(t *testing.T) {
	for _, tc := range []struct {
		lo, hi int
		want   string
	}{{3, 9, "3-9"}, {5, 5, "5-6"}, {7, 2, "7-8"}} {
		if got := MinMaxText(tc.lo, tc.hi); got != tc.want {
			t.Fatalf("%d,%d: %q want %q", tc.lo, tc.hi, got, tc.want)
		}
	}
}

func TestStateBonusOf(t *testing.T) {
	got := StateBonusOf([]StateCharge{
		{PrgDam: 1, Param1: 10, Param2: 5, Level: 3, Count: 2, HasAuraStat: true}, // (10+10)*2
		{PrgDam: 2, Param1: 99, Level: 1, Count: 1, HasAuraStat: true},            // wrong prgdam
		{PrgDam: 1, Param1: 99, Level: 0, Count: 1, HasAuraStat: true},            // no level
	})
	if got != 40 {
		t.Fatalf("got %d, want 40", got)
	}
}
