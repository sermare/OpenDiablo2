package d2statlist

import "testing"

func TestSkillDamageBox(t *testing.T) {
	armed := &Totals{DamagePct: 50, Str: 100, Weapon: WeaponParts{Armed: true, Min: 10, Max: 20, BonusPct: 10, Flat: 5}}
	bare := &Totals{DamagePct: 0, Str: 30, Weapon: WeaponParts{Min: 1, Max: 2}}

	cases := []struct {
		name string
		t    *Totals
		sk   SkillBox
		want DamageBox
	}{
		// pool 60: 10 * 160 / 100 + 5 flat after the percent
		{"full weapon", armed, SkillBox{SrcDam: 128}, DamageBox{Min: 21, Max: 37}},
		// half the weapon: 5 and 10, then the pool
		{"half weapon", armed, SkillBox{SrcDam: 64}, DamageBox{Min: 13, Max: 21}},
		// srcdam 0: no weapon part, skill damage only, flat and extra still added
		{"no weapon part", armed, SkillBox{SkillMin: 10, SkillMax: 10, Extra: 2}, DamageBox{Min: 23, Max: 23}},
		{"state bonus bit 8", armed, SkillBox{SrcDam: 128, Bit8: true, StateBonus: 40}, DamageBox{Min: 25, Max: 45, Colour: BoxBlue}},
		{"state bonus needs bit 8", armed, SkillBox{SrcDam: 128, StateBonus: 40}, DamageBox{Min: 21, Max: 37}},
		{"red wins", armed, SkillBox{SrcDam: 128, DmgBlue: true, DmgRed: true}, DamageBox{Min: 21, Max: 37, Colour: BoxRed}},
		{"bare hands strength", bare, SkillBox{SrcDam: 128}, DamageBox{Min: 1, Max: 2}},
		{"pool floor", &Totals{DamagePct: -300, Weapon: WeaponParts{Armed: true, Min: 100, Max: 100}}, SkillBox{SrcDam: 128}, DamageBox{Min: 10, Max: 10}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.t.SkillDamageBox(tc.sk); got != tc.want {
				t.Fatalf("got %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestThrownBoxIgnoresFlat(t *testing.T) {
	tt := &Totals{DamagePct: 20, Weapon: WeaponParts{Armed: true, Throwing: true, ThrowMin: 10, ThrowMax: 30, BonusPct: 30, Flat: 99, Min: 1, Max: 2}}

	lo, hi, ok := tt.ThrownBox(0)
	if !ok || lo != 15 || hi != 45 {
		t.Fatalf("thrown box %d-%d ok=%v, want 15-45 (pool 50, no flat)", lo, hi, ok)
	}

	if _, _, ok := (&Totals{Weapon: WeaponParts{Armed: true, Min: 1, Max: 2}}).ThrownBox(0); ok {
		t.Fatal("a melee weapon has no thrown box")
	}
}

func TestSkillAttackRating(t *testing.T) {
	cases := []struct {
		name                   string
		flat, pct, mast, skill int
		typ26, blue, red       bool
		want                   int
		colour                 BoxColour
	}{
		{"gear only", 1000, 20, 0, 0, false, false, false, 1200, BoxWhite},
		{"skill bonus", 1000, 20, 0, 80, false, true, false, 2000, BoxBlue},
		{"mastery", 1000, 0, 10, 0, false, true, true, 1100, BoxRed},
		{"type 0x26", 1000, 20, 0, 80, true, true, false, 0, BoxWhite},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, col := SkillAttackRating(tc.flat, tc.pct, tc.mast, tc.skill, tc.typ26, tc.blue, tc.red)
			if got != tc.want || col != tc.colour {
				t.Fatalf("got %d colour %d, want %d colour %d", got, col, tc.want, tc.colour)
			}
		})
	}
}
