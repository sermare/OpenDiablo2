package d2playertrade

import (
	"reflect"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2hero"
)

// wire is a fake server for one side: it applies the offers it receives and
// snapshots (offer, echoed seq) into updates that the client receives later.
type wire struct {
	srvOffer Offer
	srvSeq   uint32
	sent     []sentOffer // client -> server, in flight
	updates  []update    // server -> client, in flight
}

type sentOffer struct {
	o   Offer
	seq uint32
}

type update struct {
	yours Offer
	ack   uint32
}

func (w *wire) send(p *Pending) { w.sent = append(w.sent, sentOffer{p.Offer, p.Seq}) }

// deliver applies the oldest in-flight offer on the server.
func (w *wire) deliver() {
	s := w.sent[0]
	w.sent = w.sent[1:]
	w.srvOffer, w.srvSeq = s.o, s.seq
}

// snapshot makes the server emit an update now (e.g. because the other side
// changed its offer) as the server sees our offer at this moment.
func (w *wire) snapshot() { w.updates = append(w.updates, update{w.srvOffer, w.srvSeq}) }

// receive hands the oldest update to the client.
func (w *wire) receive(p *Pending) bool {
	u := w.updates[0]
	w.updates = w.updates[1:]

	return p.Apply(u.ack, u.yours)
}

func stored(code string, x int) d2hero.StoredItem { return d2hero.StoredItem{Code: code, X: x} }

func TestPendingInterleavings(t *testing.T) {
	tbk := stored("tbk", 0)
	both := Offer{Items: []d2hero.StoredItem{tbk}, Gold: 400}

	tests := []struct {
		name  string
		steps []string // add, gold, deliver, snapshot, receive
		want  Offer
	}{
		{"no update", []string{"add", "gold", "deliver", "deliver"}, both},
		{"update crosses add (the 9d-party-trade race)",
			// the other side's change makes the server emit yours=[] before it saw our add; it arrives between add and gold
			[]string{"snapshot", "add", "receive", "gold", "deliver", "deliver", "snapshot", "receive"}, both},
		{"update crosses gold",
			[]string{"add", "deliver", "snapshot", "gold", "receive", "deliver", "snapshot", "receive"}, both},
		{"update after add applied, before gold sent",
			[]string{"add", "deliver", "snapshot", "receive", "gold", "deliver"}, both},
		{"two stale updates then the echo",
			[]string{"snapshot", "snapshot", "add", "receive", "gold", "receive", "deliver", "deliver", "snapshot", "receive"}, both},
		{"echo of the first edit arrives after the second edit",
			[]string{"add", "deliver", "snapshot", "gold", "receive", "deliver", "snapshot", "receive"}, both},
	}

	for _, tt := range tests {
		var (
			p Pending
			w wire
		)

		for _, st := range tt.steps {
			switch st {
			case "add":
				p.Toggle(tbk)
				w.send(&p)
			case "gold":
				p.SetGold(400)
				w.send(&p)
			case "deliver":
				w.deliver()
			case "snapshot":
				w.snapshot()
			case "receive":
				w.receive(&p)
			}
		}

		if !reflect.DeepEqual(p.Offer, tt.want) {
			t.Errorf("%s: client offer %+v want %+v", tt.name, p.Offer, tt.want)
		}

		if !reflect.DeepEqual(w.srvOffer, tt.want) {
			t.Errorf("%s: server offer %+v want %+v", tt.name, w.srvOffer, tt.want)
		}
	}
}

func TestPendingStaleReportsNotTaken(t *testing.T) {
	var p Pending

	p.Toggle(stored("tbk", 0))

	if p.Apply(0, Offer{}) {
		t.Fatal("update older than the local edit was taken")
	}

	if len(p.Offer.Items) != 1 {
		t.Fatalf("stale update overwrote the offer: %+v", p.Offer)
	}

	if !p.Apply(1, Offer{Gold: 7}) || p.Offer.Gold != 7 {
		t.Fatalf("echo of the latest edit must be taken: %+v", p.Offer)
	}

	// without local edits the server's copy is always taken (fresh trade)
	p.Reset()

	if !p.Apply(0, Offer{Gold: 1}) || p.Offer.Gold != 1 {
		t.Fatal("update with no local edit must be taken")
	}
}

func TestSessionOfferSeq(t *testing.T) {
	s := NewSession("a", "b")

	s.SetOfferSeq("a", 3)
	s.SetOfferSeq("a", 2) // never goes back
	s.SetOfferSeq("x", 9) // not a party

	if s.OfferSeq("a") != 3 || s.OfferSeq("b") != 0 || s.OfferSeq("x") != 0 {
		t.Fatalf("seqs a=%d b=%d x=%d", s.OfferSeq("a"), s.OfferSeq("b"), s.OfferSeq("x"))
	}
}
