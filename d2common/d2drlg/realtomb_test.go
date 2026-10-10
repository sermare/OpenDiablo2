package d2drlg

import "testing"

// Tal Rasha's real tomb is TombA of the game seed: one of the seven tomb levels 66..72, never the
// second special tomb, and the same for the same seed.
func TestRealTomb(t *testing.T) {
	seen := map[int]bool{}

	for seed := uint32(1); seed <= 400; seed++ {
		a := RealTomb(seed)
		ex := DrawActExtras(seed, 1)

		if a < 66 || a > 72 || a != ex.TombA || a == ex.TombB {
			t.Fatalf("seed %d: real tomb %d, extras %+v", seed, a, ex)
		}

		if RealTomb(seed) != a {
			t.Fatalf("seed %d: not stable", seed)
		}

		seen[a] = true
	}

	if len(seen) != 7 {
		t.Errorf("400 seeds gave only %d different real tombs", len(seen))
	}
}
