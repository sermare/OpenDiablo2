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

func TestFalloffPhysical(t *testing.T) {
	cases := []struct {
		phys          int32
		distSq, limit int
		want          int32
	}{{500, 0, 36, 500}, {500, 36, 36, 500}, {500, 37, 36, 0}, {0, 100, 36, 0}}

	for _, c := range cases {
		if got := falloffPhysical(c.phys, c.distSq, c.limit); got != c.want {
			t.Errorf("falloffPhysical(%d,%d,%d) = %d, want %d", c.phys, c.distSq, c.limit, got, c.want)
		}
	}
}

func TestMonsterTargetHeal(t *testing.T) {
	m := monAt(1, 1)
	m.Vitals.HP, m.Vitals.MaxHP = 40, 100
	tg := &monsterTarget{m: m}

	tg.Heal(25 << 8)

	if m.Vitals.HP != 65 {
		t.Errorf("hp %d, want 65", m.Vitals.HP)
	}

	tg.Heal(500 << 8) // clamped to the maximum life

	if m.Vitals.HP != 100 {
		t.Errorf("hp %d, want 100", m.Vitals.HP)
	}
}

func TestMonsterKnockClass(t *testing.T) {
	m := monAt(1, 1)
	if got := (&monsterTarget{m: m}).KnockClass(); got != 0 {
		t.Errorf("class %d, want normal", got)
	}
}
