package d2monstats

import "testing"

func TestParseElementKind(t *testing.T) {
	for tok, want := range map[string]ElementKind{"fire": ElemFire, "ltng": ElemLightning, "mag": ElemMagic, "cold": ElemCold,
		"pois": ElemPoison, "life": ElemLifeSteal, "mana": ElemManaSteal, "stam": ElemStaminaSteal, "stun": ElemStun,
		"rand": ElemRandom, "frze": ElemFreeze, "": ElemNone, "xyz": ElemNone} {
		if got := ParseElementKind(tok); got != want {
			t.Errorf("%q: %d, want %d", tok, got, want)
		}
	}

	if !ElemPoison.Damaging() || ElemStun.Damaging() || ElemNone.Damaging() {
		t.Error("Damaging set wrong")
	}
}

func TestScaleElement(t *testing.T) {
	tests := []struct {
		name         string
		dm           int
		noRatio      bool
		minP, maxP   int
		wantMin, max int
	}{
		{"ratio", 200, false, 25, 50, 50, 100},
		{"noRatio raw", 200, true, 25, 50, 25, 50},
		{"max below min lifted", 100, false, 50, 10, 50, 50},
		{"zero monlvl", 0, false, 25, 50, 0, 0},
	}

	for _, tc := range tests {
		e := ScaleElement(ElemFire, 100, tc.minP, tc.maxP, 7, tc.dm, tc.noRatio)
		if e.Min != tc.wantMin || e.Max != tc.max || e.Length != 7 {
			t.Errorf("%s: %+v", tc.name, e)
		}
	}
}

func TestElementFires(t *testing.T) {
	calls := 0
	roll := func(n int) int { calls++; return 49 }

	if (Element{Kind: ElemFire, Pct: 0}).Fires(roll) || calls != 0 {
		t.Error("zero chance fires or rolls")
	}

	if !(Element{Kind: ElemFire, Pct: 100}).Fires(roll) || !(Element{Kind: ElemFire, Pct: 140}).Fires(roll) || calls != 0 {
		t.Error("100 or more must always fire without a roll")
	}

	if !(Element{Kind: ElemFire, Pct: 50}).Fires(roll) || (Element{Kind: ElemFire, Pct: 49}).Fires(roll) || calls != 2 {
		t.Error("roll under the chance")
	}

	if (Element{Pct: 100}).Fires(roll) {
		t.Error("no kind")
	}
}

func TestPoisonTotal(t *testing.T) {
	e := Element{Kind: ElemPoison, Min: 13, Max: 26, Length: 200}

	min, max := e.PoisonTotal()
	if min != 13*10*400>>8 || max != 26*10*400>>8 {
		t.Errorf("%d %d", min, max)
	}
}
