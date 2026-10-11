package d2steal

import "testing"

func it(c string) *Item { return &Item{Code: c, Mode: ModeBelt} }

func TestPasses(t *testing.T) {
	tests := []struct {
		seed uint32
		want bool
	}{{0, false}, {29, false}, {30, true}, {99, true}, {129, false}, {130, true}}
	for _, tc := range tests {
		if Passes(tc.seed) != tc.want {
			t.Errorf("seed %d", tc.seed)
		}
	}
}

func TestPassRate(t *testing.T) {
	n := 0

	for s := uint32(0); s < 100; s++ {
		if Passes(s) {
			n++
		}
	}

	if n != PassPercent {
		t.Fatalf("%d of 100", n)
	}
}

func TestStealTakesFirstCellAndCompacts(t *testing.T) {
	var b Belt
	b[1], b[5], b[9], b[2] = it("a"), it("b"), it("c"), it("d")

	r := Steal(&b, func() uint32 { return 50 }, true, true)

	if r.Stolen == nil || r.Stolen.Code != "a" || r.Cell != 1 {
		t.Fatalf("first occupied cell is 1: %+v", r)
	}

	// column 1 was a(1) b(5) c(9): now b(1) c(5); column 2 untouched
	if b[1].Code != "b" || b[5].Code != "c" || b[9] != nil || b[2].Code != "d" {
		t.Fatalf("%v", b)
	}
}

func TestStealRefusals(t *testing.T) {
	tests := []struct {
		name         string
		seed         uint32
		player, curs bool
		mode         int
	}{
		{"low roll", 10, true, true, ModeBelt},
		{"not a player", 50, false, true, ModeBelt},
		{"cursor busy", 50, true, false, ModeBelt},
		{"not in belt mode", 50, true, true, 4},
	}

	for _, tc := range tests {
		var b Belt
		b[3] = &Item{Code: "x", Mode: tc.mode}

		calls := 0
		r := Steal(&b, func() uint32 { calls++; return tc.seed }, tc.player, tc.curs)

		if r.Stolen != nil || b[3] == nil || calls != 1 {
			t.Errorf("%s: %+v calls=%d", tc.name, r, calls)
		}
	}

	var empty Belt
	if r := Steal(&empty, func() uint32 { return 99 }, true, true); r.Stolen != nil {
		t.Fatal("empty belt")
	}
}

func TestCompactColumnClamp(t *testing.T) {
	var b Belt
	b[7], b[15] = it("p"), it("q")
	Compact(&b, 9) // clamps to column 3
	Compact(&b, -1)

	if b[3] == nil || b[3].Code != "p" || b[7].Code != "q" || b[15] != nil {
		t.Fatalf("%v", b)
	}
}
