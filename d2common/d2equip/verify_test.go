package d2equip

import "testing"

// Pins of rules checked against Game.exe 1.14b (0x62bd70, 0x62ebf0, 0x62b720,
// 0x557d90, 0x62f100, 0x63ec50). The table based ones need D2_TABLES.

func TestVerifiedMaxSockBorders(t *testing.T) {
	ty := &Type{MaxSock1: 1, MaxSock25: 2, MaxSock40: 3}

	for ilvl, want := range map[int]int{1: 1, 25: 1, 26: 2, 40: 2, 41: 3, 99: 3} {
		if got := ty.MaxSocketsByLevel(ilvl); got != want {
			t.Errorf("ilvl %d: %d want %d", ilvl, got, want)
		}
	}
}

func TestVerifiedMaxSocketsRealData(t *testing.T) {
	d := loadReal(t)

	// Plate Mail: gemsockets 2 in armor.txt, the tors MaxSock columns are
	// larger, so the base column is the binding one at every level.
	plt := d.bases["plt"]
	if plt.GemSockets != 2 {
		t.Fatalf("plt gemsockets %d", plt.GemSockets)
	}

	tors := d.rules.Types.Get("tors")
	if tors == nil || tors.MaxSock1 < 2 {
		t.Fatalf("tors %+v", tors)
	}

	for _, ilvl := range []int{1, 25, 26, 40, 41, 99} {
		if got := d.rules.Types.MaxSockets(plt, ilvl, 2); got != 2 {
			t.Errorf("plt ilvl %d: %d", ilvl, got)
		}
	}

	// the border at 40 is visible on a base whose gemsockets exceed the lower column
	b := d.bases["gsd"]
	got := [3]int{d.rules.Types.MaxSockets(b, 40, 2), d.rules.Types.MaxSockets(b, 41, 2), d.rules.Types.MaxSockets(b, 25, 2)}

	if got != [3]int{4, 6, 3} {
		t.Errorf("gsd 40/41/25 = %v", got)
	}
}

func TestVerifiedRequirementPercent(t *testing.T) {
	for _, c := range []struct {
		base, pct int
		eth       bool
		want      int
	}{
		{100, 0, false, 100}, {100, -20, false, 80}, {100, 25, false, 125}, {37, 15, false, 42}, {37, -15, false, 32},
		{100, 0, true, 90}, {100, -20, true, 70},
	} {
		if got := RequiredStat(c.base, c.pct, c.eth); got != c.want {
			t.Errorf("RequiredStat(%d,%d,%v)=%d want %d", c.base, c.pct, c.eth, got, c.want)
		}
	}
}

func TestVerifiedRequiredLevel(t *testing.T) {
	if got := RequiredLevelCrafted(99, 20, 30); got != 30+10+6 {
		t.Errorf("crafted %d", got)
	}

	if got := RequiredLevelCrafted(99, 80, 85); got != 98 {
		t.Errorf("crafted cap %d", got)
	}

	for _, c := range []struct {
		q, base int
		sock    []int
		skills  []int
		stat    int
		want    int
	}{
		{0, 30, nil, nil, 0, 30}, {45, 30, nil, nil, 0, 45}, {45, 30, []int{50}, nil, 0, 50},
		{0, 30, nil, []int{30}, 0, 30}, {0, 10, nil, []int{18}, 5, 23}, {0, 0, nil, nil, -5, 0},
	} {
		if got := RequiredLevelTotal(c.q, c.base, c.sock, c.skills, c.stat); got != c.want {
			t.Errorf("%+v: %d", c, got)
		}
	}
}

func TestVerifiedDurabilityChances(t *testing.T) {
	// armor 10, weapon 4 (0x557d90); itemgen.md had them swapped
	if ChanceArmor != 10 || ChanceWeapon != 4 || ChanceThrown != 10 {
		t.Fatal("durability chances")
	}
}

func TestVerifiedRepairNumerator(t *testing.T) {
	for _, c := range []struct {
		max, cur  int
		replenish bool
		want      int
	}{
		{100, 100, false, 0}, {100, 40, false, 60}, {100, 0, false, 100}, {0, 0, false, 0},
		{100, 99, true, 0}, {100, 100, true, 0}, {100, 98, true, 99}, {100, 0, true, 99}, {1, 0, true, 0},
	} {
		if got := RepairNumerator(c.max, c.cur, c.replenish); got != c.want {
			t.Errorf("%+v: %d", c, got)
		}
	}
}

func TestVerifiedAssassinClawsBothHands(t *testing.T) {
	d := loadReal(t)

	for _, c := range []struct {
		right, left string
		ok          bool
	}{
		{"ktr", "ktr", true}, {"ktr", "ssd", false}, {"ssd", "ktr", false}, {"ssd", "buc", true},
	} {
		dec := d.rules.CheckHands("ass", d.item(c.right), d.item(c.left))
		if (dec == nil) != c.ok {
			t.Errorf("sin %s + %s: %+v want ok=%v", c.right, c.left, dec, c.ok)
		}
	}
}
