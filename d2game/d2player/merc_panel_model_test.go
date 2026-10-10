package d2player

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2equip"
)

func TestMercViewValues(t *testing.T) {
	v := MercView{Name: "Greth", Level: 12, Exp: 5000, NextExp: 9000, Str: 60, Dex: 70, DmgMin: 4, DmgMax: 9,
		Defense: 120, Resist: [4]int{10, 20, 30, 40}, HP: 150, MaxHP: 300}

	want := map[string]string{
		"experience": "5000", "level": "12", "next_level": "9000", "strength": "60", "dexterity": "70",
		"damage": "4-9", "defense": "120", "fire": "10", "cold": "20", "lightning": "30", "poison": "40", "life": "150/300",
	}

	got := v.Values()
	for _, n := range MercRowOrder {
		if got[n] != want[n] {
			t.Errorf("%s = %q, want %q", n, got[n], want[n])
		}
	}

	if len(got) != len(MercRowOrder) || len(MercRowOrder) != len(mercValues) {
		t.Errorf("%d values, %d rows, %d table rows", len(got), len(MercRowOrder), len(mercValues))
	}

	v.Dead = true
	if v.Values()["life"] != "0/300" {
		t.Errorf("dead merc life %q", v.Values()["life"])
	}

	for i, n := range MercRowOrder {
		if mercValues[i].Name != n || mercLabels[i].Name != n {
			t.Errorf("row %d: order %s differs from the tables (%s, %s)", i, n, mercValues[i].Name, mercLabels[i].Name)
		}
	}
}

func TestMercSlotLoc(t *testing.T) {
	tests := []struct {
		slot string
		loc  d2equip.Loc
		ok   bool
	}{
		{"head", d2equip.LocHead, true}, {"torso", d2equip.LocTorso, true},
		{"weapon", d2equip.LocRightHand, true}, {"shield", d2equip.LocLeftHand, true},
		{"belt", 0, false}, {"", 0, false},
	}

	for _, c := range tests {
		loc, ok := MercSlotLoc(c.slot)
		if ok != c.ok || (ok && loc != c.loc) {
			t.Errorf("%q = %v,%v", c.slot, loc, ok)
		}

		if ok && !d2equip.MercHasLoc(loc) {
			t.Errorf("%q: a merc has no %v", c.slot, loc)
		}
	}
}
