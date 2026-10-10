package d2monster

import (
	"reflect"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
)

func TestPickUniqueMods(t *testing.T) {
	cands := []ModCandidate{{ID: 5, Weight: 10}, {ID: 6, Weight: 0}, {ID: 7, Weight: 5}, {ID: 8, Weight: 1}}

	for seed := uint32(1); seed < 50; seed++ {
		got := PickUniqueMods(d2rand.New(seed), cands, 3)
		if len(got) != 3 {
			t.Fatalf("seed %d: %v, want 3 distinct modifiers (the weight 0 row is never picked)", seed, got)
		}

		seen := map[int]bool{}

		for _, id := range got {
			if id == 6 || seen[id] {
				t.Fatalf("seed %d: %v has a repeat or the unpickable row", seed, got)
			}

			seen[id] = true
		}

		if again := PickUniqueMods(d2rand.New(seed), cands, 3); !reflect.DeepEqual(got, again) {
			t.Errorf("seed %d: the pick is not deterministic", seed)
		}
	}

	if got := PickUniqueMods(d2rand.New(1), cands, 10); len(got) != 3 {
		t.Errorf("asking for more than the pool holds gave %v", got)
	}

	if got := PickUniqueMods(d2rand.New(1), nil, 2); got != nil {
		t.Errorf("empty pool gave %v", got)
	}
}
