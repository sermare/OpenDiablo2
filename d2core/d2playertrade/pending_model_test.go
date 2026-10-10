package d2playertrade

import (
	"math/rand"
	"reflect"
	"testing"
)

// Randomised ordering check on top of the hand-written interleavings: whatever
// order edits, deliveries, server snapshots and receptions happen in (FIFO per
// direction, as on a TCP stream), once the wires are drained and one more
// update arrives the client and the server hold the same offer, and the client
// never shows an offer older than its latest edit in between.
func TestPendingRandomOrderingConverges(t *testing.T) {
	rng := rand.New(rand.NewSource(1))

	for run := 0; run < 2000; run++ {
		var (
			p Pending
			w wire
		)

		for step := 0; step < 30; step++ {
			switch rng.Intn(5) {
			case 0:
				p.Toggle(stored("tbk", rng.Intn(3)))
				w.send(&p)
			case 1:
				p.SetGold(rng.Intn(500))
				w.send(&p)
			case 2:
				if len(w.sent) > 0 {
					w.deliver()
				}
			case 3:
				w.snapshot()
			case 4:
				if len(w.updates) > 0 {
					before := p.Offer
					taken := w.receive(&p)

					if !taken && !reflect.DeepEqual(before, p.Offer) {
						t.Fatalf("run %d: stale update changed the local offer", run)
					}
				}
			}
		}

		for len(w.sent) > 0 {
			w.deliver()
		}

		w.snapshot()

		for len(w.updates) > 0 {
			w.receive(&p)
		}

		if !reflect.DeepEqual(p.Offer, w.srvOffer) && !(p.Offer.Empty() && w.srvOffer.Empty()) {
			t.Fatalf("run %d: client %+v != server %+v", run, p.Offer, w.srvOffer)
		}
	}
}
