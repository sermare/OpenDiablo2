package d2rand

import "testing"

// Expected values were computed independently with Python big integers from
// the algorithm documented in the package comment.
func TestStepSequence(t *testing.T) {
	s := New(12345)

	want := []uint32{0x015B2E77, 0x8B57C5B0, 0x606EEEF7, 0xEEF1409D}

	if s.Hi != 0x29A {
		t.Fatalf("high word after Init = %#x", s.Hi)
	}

	for i, w := range want {
		if got := s.Step(); got != w {
			t.Fatalf("step %d = %#x, want %#x", i, got, w)
		}
	}
}

func TestRollRules(t *testing.T) {
	s := New(1)

	if s.Roll(0) != 0 || s.Roll(-5) != 0 || (*s != *New(1)) {
		t.Fatal("Roll(n<1) must return 0 without stepping")
	}

	// Roll(1) consumes a step and returns 0
	s = New(1)
	if s.Roll(1) != 0 || *s == *New(1) {
		t.Fatal("Roll(1) must advance the generator and return 0")
	}

	// power of two masks, other values use modulo
	a, b := New(777), New(777)
	v := b.Step()

	if got := a.Roll(8); got != v&7 {
		t.Fatalf("Roll(8) = %d, want %d", got, v&7)
	}

	a, b = New(777), New(777)
	v = b.Step()

	if got := a.Roll(7); got != v%7 {
		t.Fatalf("Roll(7) = %d, want %d", got, v%7)
	}
}

func TestSeedHierarchy(t *testing.T) {
	base, drlg := DrlgBaseSeed(0xDEADBEEF)
	if base != drlg.Lo {
		t.Fatal("base seed must be the low word after one step")
	}

	// a level seed depends only on base+id, not on generation order
	l1, l2 := LevelSeed(base, 3), LevelSeed(base, 3)
	if *l1 != *l2 || l1.Hi != 0x29A {
		t.Fatal("level seeds must be reproducible")
	}

	if *LevelSeed(base, 4) == *l1 {
		t.Fatal("different levels must have different seeds")
	}

	// rooms: level seed advances once per room
	level := LevelSeed(base, 1)
	before := *level
	room, value := NewRoomSeed(level)

	if *level == before || value != room.Lo {
		t.Fatalf("room derivation wrong: level=%+v room=%+v value=%#x", level, room, value)
	}
}
