package d2rand

import (
	"sync"
	"testing"
)

// A Seed is a single-owner stream by design (every draw mutates it), so the
// supported concurrent pattern is: derive per-level / per-room streams from a
// shared base value, one stream per goroutine. That must be race free and
// give the same numbers as the sequential run.
func TestDerivedStreamsParallel(t *testing.T) {
	base, _ := DrlgBaseSeed(4242)

	seq := func(level uint32) [3]uint32 {
		ls := LevelSeed(base, level)
		room, v := NewRoomSeed(ls)

		return [3]uint32{v, room.Roll(100), ls.Step()}
	}

	want := map[uint32][3]uint32{}
	for l := uint32(1); l <= 40; l++ {
		want[l] = seq(l)
	}

	var wg sync.WaitGroup

	for g := 0; g < 8; g++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for l := uint32(1); l <= 40; l++ {
				if got := seq(l); got != want[l] {
					t.Errorf("level %d: %v want %v", l, got, want[l])
				}
			}
		}()
	}

	wg.Wait()
}
