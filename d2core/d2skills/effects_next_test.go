package d2skills

import "testing"

func TestKnockDest(t *testing.T) {
	open := func(int, int) bool { return true }
	wall := func(x, _ int) bool { return x < 12 }

	cases := []struct {
		name         string
		hx, hy, x, y int
		dist         int
		walk         func(int, int) bool
		wx, wy       int
	}{
		{"east", 10, 10, 11, 10, 2, open, 13, 10},
		{"diagonal", 10, 10, 9, 9, 2, open, 7, 7},
		{"nil grid", 10, 10, 10, 12, 3, nil, 10, 15},
		{"wall stops it", 10, 10, 11, 10, 3, wall, 11, 10},
		{"same cell", 10, 10, 10, 10, 2, open, 10, 10},
	}

	for _, c := range cases {
		x, y := knockDest(c.hx, c.hy, c.x, c.y, c.dist, c.walk)
		if x != c.wx || y != c.wy {
			t.Errorf("%s: (%d,%d), want (%d,%d)", c.name, x, y, c.wx, c.wy)
		}
	}
}

func TestRedeemRoll(t *testing.T) {
	cases := []struct {
		roll, chance int
		want         bool
	}{{0, 1, true}, {49, 50, true}, {50, 50, false}, {99, 0, false}}

	for _, c := range cases {
		if got := redeemRoll(c.roll, c.chance); got != c.want {
			t.Errorf("redeemRoll(%d,%d) = %v", c.roll, c.chance, got)
		}
	}
}

func TestMonsterKnockClass(t *testing.T) {
	m := monAt(1, 1)
	if got := (&monsterTarget{m: m}).KnockClass(); got != 0 {
		t.Errorf("class %d, want normal", got)
	}
}
