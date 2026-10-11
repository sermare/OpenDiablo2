package d2skills

import "testing"

func TestAuraUpkeepPay(t *testing.T) {
	cases := []struct {
		mana, frac, upkeep int
		wantMana, wantFrac int
		ok                 bool
	}{
		{10, 0, 256, 9, 0, true},    // Prayer level 1: exactly one mana
		{10, 0, 304, 9, 48, true},   // 1.1875 mana: the fraction is carried
		{10, 224, 304, 8, 16, true}, // the carried fraction completes a second point
		{0, 0, 256, 0, 0, false},    // not enough: nothing is paid
		{1, 0, 256, 0, 0, true},     // exactly enough
		{5, 7, 0, 5, 7, true},       // no upkeep
	}

	for i, c := range cases {
		m, f, ok := auraUpkeepPay(c.mana, c.frac, c.upkeep)
		if ok != c.ok || m != c.wantMana || f != c.wantFrac {
			t.Errorf("case %d: got %d,%d,%v want %d,%d,%v", i, m, f, ok, c.wantMana, c.wantFrac, c.ok)
		}
	}
}
