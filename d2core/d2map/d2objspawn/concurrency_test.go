package d2objspawn

import (
	"sync"
	"testing"
)

// Tables built by FromText are read-only afterwards; the engine shares one
// instance, so concurrent queries must be race free.
func TestTablesConcurrentReads(t *testing.T) {
	tb, err := FromText([]byte(objTxt), groupTxt())
	if err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup

	for g := 0; g < 8; g++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for i := 0; i < 300; i++ {
				tb.Def(i % 5)
				tb.WaypointBit(2, 1, i%2 == 0)

				room := &fakeRoom{tiles: 4096}
				PopulateRoom(tb, Level{Act: 1, Groups: [8]int{1}, Probs: [8]int{100}}, false, room, uint32(i))
			}
		}()
	}

	wg.Wait()
}
