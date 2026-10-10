package d2replay

import (
	"bytes"
	"sync"
	"testing"
)

// Independent sessions (own Recorder, own Sim) replayed in parallel must not
// share state and must all produce the same trace.
func TestReplayParallelSessions(t *testing.T) {
	rec := NewRecorder(77)
	for f := 0; f < 40; f += 3 {
		rec.Record(Input{Frame: f, Kind: "move", Actor: uint32(f%5 + 1), X: f, Y: f / 2})
	}

	var buf bytes.Buffer
	if err := rec.Finish(60).Write(&buf); err != nil {
		t.Fatal(err)
	}

	raw := buf.Bytes()

	first, err := Read(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}

	want := Run(newToy(true), first)

	var wg sync.WaitGroup

	for g := 0; g < 8; g++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for i := 0; i < 5; i++ {
				l, err := Read(bytes.NewReader(raw))
				if err != nil {
					t.Error(err)
					return
				}

				if f := Compare(want, Run(newToy(true), l)); f >= 0 {
					t.Errorf("parallel replay diverged at frame %d", f)
				}

				if err := Verify(newToy(true), l); err != nil {
					t.Error(err)
				}
			}
		}()
	}

	wg.Wait()
}
