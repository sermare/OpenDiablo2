package d2drop

import "testing"

// fixedRNG returns the same value for every roll and counts the calls.
type fixedRNG struct {
	v     uint32
	calls int
}

func (f *fixedRNG) Roll(n int32) uint32 {
	f.calls++

	return f.v
}

func (f *fixedRNG) Chance() bool { f.calls++; return f.v&1 == 1 }

func TestRollEthereal(t *testing.T) {
	ok := EtherealInput{WeaponOrArmor: true, Durability: true, Quality: QualityMagic}

	for _, c := range []struct {
		name  string
		in    EtherealInput
		roll  uint32
		want  bool
		calls int
	}{
		{"hit 4", ok, 4, true, 1},
		{"miss 5", ok, 5, false, 1},
		{"low quality", EtherealInput{true, true, false, QualityLow}, 0, false, 0},
		{"set", EtherealInput{true, true, false, QualitySet}, 0, false, 0},
		{"quest", EtherealInput{true, true, true, QualityNormal}, 0, false, 0},
		{"no durability", EtherealInput{true, false, false, QualityNormal}, 0, false, 0},
		{"not weapon or armor", EtherealInput{false, true, false, QualityNormal}, 0, false, 0},
		{"unique", EtherealInput{true, true, false, QualityUnique}, 0, true, 1},
	} {
		r := &fixedRNG{v: c.roll}
		if got := RollEthereal(r, c.in); got != c.want || r.calls != c.calls {
			t.Errorf("%s: %v with %d rolls, want %v %d", c.name, got, r.calls, c.want, c.calls)
		}
	}

	for base, want := range map[int]int{12: 7, 13: 7, 1: 1, 250: 126, 0: 1} {
		if got := EtherealMaxDurability(base); got != want {
			t.Errorf("EtherealMaxDurability(%d)=%d want %d", base, got, want)
		}
	}
}

func TestRollSockets(t *testing.T) {
	base := SocketInput{Quality: QualityNormal, HasInventory: true, MaxSockets: 6, Difficulty: 2, InitSeed: 10, Cells: 6}

	with := func(f func(*SocketInput)) SocketInput { in := base; f(&in); return in }

	for _, c := range []struct {
		name  string
		in    SocketInput
		roll  uint32
		want  int
		calls int
	}{
		{"hell 10%6+1", base, 32, 5, 1},
		{"roll 33 misses", base, 33, 0, 1},
		{"normal cap 3", with(func(i *SocketInput) { i.Difficulty = 0 }), 0, 2, 1},    // 10%3+1
		{"nightmare cap 4", with(func(i *SocketInput) { i.Difficulty = 1 }), 0, 3, 1}, // 10%4+1
		{"small area", with(func(i *SocketInput) { i.Cells = 4 }), 0, 4, 1},           // min(5,4)
		{"level limit", with(func(i *SocketInput) { i.MaxSockets = 2 }), 0, 1, 1},     // 10%2+1
		{"low quality", with(func(i *SocketInput) { i.Quality = QualityLow }), 0, 0, 0},
		{"stackable", with(func(i *SocketInput) { i.Stackable = true }), 0, 0, 0},
		{"no inventory", with(func(i *SocketInput) { i.HasInventory = false }), 0, 0, 0},
		{"no level limit", with(func(i *SocketInput) { i.MaxSockets = 0 }), 0, 0, 0},
		{"superior", with(func(i *SocketInput) { i.Quality = QualitySuperior }), 0, 5, 1},
	} {
		r := &fixedRNG{v: c.roll}
		if got := RollSockets(r, c.in); got != c.want || r.calls != c.calls {
			t.Errorf("%s: %d with %d rolls, want %d %d", c.name, got, r.calls, c.want, c.calls)
		}
	}
}

func TestClampSocketCount(t *testing.T) {
	for _, c := range []struct{ req, cells, lvl, want int }{
		{5, 6, 6, 5}, {5, 9, 6, 5}, {6, 9, 9, 6}, {9, 12, 12, 6}, {3, 4, 2, 2}, {0, 4, 4, 1}, {3, 0, 4, 0},
	} {
		if got := ClampSocketCount(c.req, c.cells, c.lvl); got != c.want {
			t.Errorf("ClampSocketCount(%d,%d,%d)=%d want %d", c.req, c.cells, c.lvl, got, c.want)
		}
	}
}
