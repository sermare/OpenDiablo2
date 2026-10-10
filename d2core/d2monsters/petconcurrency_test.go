package d2monsters

import (
	"fmt"
	"sync"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2skill"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2summon"
)

// One Summoner (own fake world and rng) per goroutine over shared read-only
// templates must be race free. A Summoner itself is driven only by the game
// loop, so it is deliberately not shared across goroutines.
func TestSummonersParallelSharedTemplates(t *testing.T) {
	tpl, err := d2summon.LoadTemplates([]byte(fakeMonstats))
	if err != nil {
		t.Fatal(err)
	}

	o := &d2skill.SummonOrder{Key: "necroskeleton", PetType: "skeleton", Max: 3, Count: 2, Kind: "minion", HPPct: 50, Frames: 100,
		Stats: []d2skill.StatMod{{Stat: "damagepercent", Value: 100}}}

	place := func(i, n int) (int, int) { return 10 + i, 20 }

	var wg sync.WaitGroup

	for g := 0; g < 8; g++ {
		wg.Add(1)

		go func(g int) {
			defer wg.Done()

			w := newFakePets()
			s := NewSummoner(w, tpl, d2summon.Normal, uint32(g+1))

			for i := 0; i < 100; i++ {
				w.frame = i * 10
				owner := fmt.Sprintf("o%d", i%4)

				s.Cast(owner, "necroskeleton", o, nil, place)
				s.Step()
				s.Roster(owner).Total()
			}
		}(g)
	}

	wg.Wait()
}
