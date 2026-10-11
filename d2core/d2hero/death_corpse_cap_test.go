package d2hero

import "testing"

func TestCorpseOverCap(t *testing.T) {
	tests := []struct {
		n    int
		want bool
	}{{0, false}, {14, false}, {15, false}, {16, true}, {100, true}}

	for _, tc := range tests {
		if got := CorpseOverCap(tc.n); got != tc.want {
			t.Errorf("CorpseOverCap(%d) = %v, want %v", tc.n, got, tc.want)
		}
	}
}

func TestCorpseExperience(t *testing.T) {
	tests := []struct{ in, want int }{
		{0, 0}, {-5, 0}, {1, 0}, {4, 3}, {100, 75}, {1000, 750}, {0x100000, 786432},
		{2000000000, 1500000000}, // no 32-bit overflow
	}

	for _, tc := range tests {
		if got := CorpseExperience(tc.in); got != tc.want {
			t.Errorf("CorpseExperience(%d) = %d, want %d", tc.in, got, tc.want)
		}
	}
}

func TestDieOverCorpseCap(t *testing.T) {
	var s DeathState

	out := s.Die(DeathInput{CorpseCount: 16, Gold: 50, PenaltyValue: 100})
	if !out.DropToFloor || out.Corpse != nil || s.Corpse != nil {
		t.Fatalf("over the cap: %+v, corpse %v", out, s.Corpse)
	}

	if s.Deaths != 1 || !s.Died || out.GoldDropped != 50 {
		t.Fatalf("death must still count: %+v %+v", s, out)
	}

	out = s.Die(DeathInput{CorpseCount: 15, PenaltyValue: 100})
	if out.DropToFloor || out.Corpse == nil || out.Corpse.Experience != 75 {
		t.Fatalf("at the cap a corpse is made with 75%%: %+v", out)
	}
}
