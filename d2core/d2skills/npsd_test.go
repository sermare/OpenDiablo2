package d2skills

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2state"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
)

func TestConvertScaled(t *testing.T) {
	tests := []struct {
		level, hp, max, caster int
		wl, whp, wmax          int
	}{
		{30, 300, 600, 15, 15, 150, 300},      // above the caster: halved
		{10, 100, 200, 15, 10, 100, 200},      // below: untouched
		{15, 100, 200, 15, 15, 100, 200},      // equal: untouched
		{90, 1, 50, 3, 3, 1, 1},               // never below 1
		{0, 100, 100, 15, 0, 100, 100},        // level 0 monsters are left alone
		{60, 5000, 10000, 30, 30, 2500, 5000}, // hell boss sized
	}

	for _, tc := range tests {
		l, hp, mx := ConvertScaled(tc.level, tc.hp, tc.max, tc.caster)
		if l != tc.wl || hp != tc.whp || mx != tc.wmax {
			t.Errorf("ConvertScaled(%d,%d,%d,%d) = %d,%d,%d want %d,%d,%d", tc.level, tc.hp, tc.max, tc.caster, l, hp, mx,
				tc.wl, tc.whp, tc.wmax)
		}
	}
}

func TestRevertConverted(t *testing.T) {
	// half life on a halved maximum returns as half life of the original
	if got := RevertConverted(150, 300, 600); got != 300 {
		t.Errorf("revert %d, want 300", got)
	}

	if got := RevertConverted(1, 1000, 10); got != 1 {
		t.Errorf("revert %d, want 1", got)
	}

	if got := RevertConverted(5, 0, 10); got != 1 {
		t.Errorf("revert with no max %d, want 1", got)
	}
}

func TestReviveCap(t *testing.T) {
	m := &d2mapentity.Monster{}
	m.Vitals.Level, m.Vitals.HP, m.Vitals.MaxHP = 40, 400, 400

	reviveCap(m, 40, 20)

	if m.Vitals.Level != 20 || m.Vitals.HP != 200 || m.Vitals.MaxHP != 200 {
		t.Errorf("revived %+v", m.Vitals)
	}

	m.Vitals.Level, m.Vitals.HP, m.Vitals.MaxHP = 10, 50, 50
	reviveCap(m, 10, 20)

	if m.Vitals.Level != 10 || m.Vitals.HP != 50 {
		t.Errorf("revive below the caster changed %+v", m.Vitals)
	}
}

func TestForcedByStat(t *testing.T) {
	for stat, want := range map[string]string{
		d2state.StatBlind: "blind", d2state.StatConfused: "confuse", d2state.StatAttract: "attract",
		d2state.StatTaunted: "taunt",
	} {
		if got := forcedByStat[stat].String(); got != want {
			t.Errorf("%s -> %s, want %s", stat, got, want)
		}
	}
}
