package d2equip

import "testing"

func TestDepletedStackAction(t *testing.T) {
	stack := DepletedItem{Stackable: true}
	mod := func(f func(*DepletedItem)) DepletedItem {
		d := stack
		f(&d)

		return d
	}

	cases := []struct {
		name string
		in   DepletedItem
		want DepletedAction
	}{
		{"quantity left", mod(func(d *DepletedItem) { d.Quantity = 3 }), DepletedNone},
		{"not a stack", DepletedItem{}, DepletedNone},
		{"empty arrows", stack, DepletedUnequip},
		{"empty javelins", DepletedItem{Throwable: true}, DepletedUnequip},
		{"stat 7d keeps", mod(func(d *DepletedItem) { d.HasStat7d = true }), DepletedKeep},
		{"magic weapon breaks", mod(func(d *DepletedItem) { d.WeaponType, d.MagicQuality = true, true }), DepletedBreak},
		{"already broken", mod(func(d *DepletedItem) { d.WeaponType, d.MagicQuality, d.Broken = true, true, true }), DepletedNone},
		{"normal weapon stack unequips", mod(func(d *DepletedItem) { d.WeaponType = true }), DepletedUnequip},
	}

	for _, c := range cases {
		if got := DepletedStackAction(c.in); got != c.want {
			t.Errorf("%s: got %d want %d", c.name, got, c.want)
		}
	}
}

func TestReplacementStack(t *testing.T) {
	c := []StackCandidate{{Code: "hax"}, {Code: "jav", Broken: true}, {Code: "jav"}, {Code: "jav"}}

	if got := ReplacementStack("jav", c); got != 2 {
		t.Errorf("got %d want 2", got)
	}

	if got := ReplacementStack("bow", c); got != -1 {
		t.Errorf("got %d want -1", got)
	}
}

func TestWornItemBreaks(t *testing.T) {
	cases := []struct {
		weapon, applicable bool
		dur                int
		broken, want       bool
	}{
		{true, true, 0, false, true}, {true, true, -1, false, true}, {true, true, 1, false, false},
		{true, true, 0, true, false}, {false, true, 0, false, false}, {true, false, 0, false, false},
	}
	for _, c := range cases {
		if got := WornItemBreaks(c.weapon, c.applicable, c.dur, c.broken); got != c.want {
			t.Errorf("%+v got %v", c, got)
		}
	}
}

func TestReloadColumns(t *testing.T) {
	ty, err := ParseTypes([]byte("ItemType\tCode\tEquiv1\tEquiv2\tBody\tBodyLoc1\tBodyLoc2\tReload\tReEquip\n" +
		"Arrows\tbowq\t\t\t0\t\t\t1\t0\nPotion\tpotn\t\t\t0\t\t\t0\t1\n"))
	if err != nil {
		t.Fatal(err)
	}

	r := Rules{Types: ty}
	if !r.IsReload("bowq") || r.IsReload("potn") || !r.IsReEquip("potn") || r.IsReEquip("none") {
		t.Error("columns")
	}
}
