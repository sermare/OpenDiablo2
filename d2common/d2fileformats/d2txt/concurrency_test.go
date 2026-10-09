package d2txt

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
)

// Header lookups that miss write the lazy "missing" set, so Has/Missing/col
// must be safe to call from several goroutines (the row cursor itself, Next,
// stays single-threaded by design).
func TestMissingHeaderConcurrent(t *testing.T) {
	d := LoadDataDictionary([]byte("Name\tVal\nfoo\t7\n"))

	var calls int64

	d.OnMissing = func(string) { atomic.AddInt64(&calls, 1) }

	var wg sync.WaitGroup

	for g := 0; g < 8; g++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for i := 0; i < 200; i++ {
				d.col(fmt.Sprintf("nope%d", i%10))
				d.Has("Val")
				_ = d.Missing()
			}
		}()
	}

	wg.Wait()

	if len(d.Missing()) != 10 || atomic.LoadInt64(&calls) != 10 {
		t.Fatalf("missing %d, OnMissing calls %d, want 10 each", len(d.Missing()), calls)
	}
}
