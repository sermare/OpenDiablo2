package d2loader

import (
	"io"
	"sync"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2loader/asset/types"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2util"
)

// Streams of one MPQ are read from several goroutines (sounds load while the game runs). A shared
// Seek+Read on the archive file raced and made loads fail with "file not found" at random.
func TestLoader_ConcurrentMPQReads(t *testing.T) {
	loader, _ := NewLoader(d2util.LogLevelDefault)

	if err := loader.AddSource(sourcePathD, types.AssetSourceMPQ); err != nil {
		t.Fatal(err)
	}

	want, err := loader.Load(exclusiveD)
	if err != nil {
		t.Fatal(err)
	}

	wantData, _ := io.ReadAll(want)

	var wg sync.WaitGroup

	for g := 0; g < 8; g++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for i := 0; i < 300; i++ {
				name := exclusiveD
				if i%2 == 1 {
					name = subdirCommonD
				}

				r, err := loader.Load(name)
				if err != nil {
					t.Errorf("load %s: %v", name, err)
					return
				}

				data, err := io.ReadAll(r)
				if err != nil {
					t.Errorf("read %s: %v", name, err)
					return
				}

				if name == exclusiveD && string(data) != string(wantData) {
					t.Errorf("%s: data differs under concurrency", name)
					return
				}
			}
		}()
	}

	wg.Wait()
}
