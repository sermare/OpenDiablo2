package d2gamescreen

import (
	"sync"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2asset"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2records"
)

// SaveActiveGame (window close, console command) may run while a game screen is
// being created or torn down.
func TestSaveActiveGameConcurrentWithSetClear(t *testing.T) {
	g := &Game{} // no hero, no client: saveBeforeExit is a no-op

	var wg sync.WaitGroup

	for i := 0; i < 4; i++ {
		wg.Add(2)

		go func() {
			defer wg.Done()

			for n := 0; n < 500; n++ {
				SaveActiveGame()
			}
		}()

		go func() {
			defer wg.Done()

			for n := 0; n < 500; n++ {
				setActiveGame(g)
				clearActiveGame(g)
			}
		}()
	}

	wg.Wait()
	SaveActiveGame()
}

func TestSpawnTablesLazyInitConcurrent(t *testing.T) {
	g := &Game{asset: &d2asset.AssetManager{Records: &d2records.RecordManager{}}}

	res := make([]interface{}, 16)

	var wg sync.WaitGroup

	for i := range res {
		wg.Add(1)

		go func(i int) {
			defer wg.Done()

			res[i] = g.spawnTables()
		}(i)
	}

	wg.Wait()

	for _, r := range res {
		if r != res[0] {
			t.Fatal("spawnTables returned different instances")
		}
	}
}
