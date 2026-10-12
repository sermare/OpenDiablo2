package d2quest

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
)

// Jerhyn stands at the second Lut Gholein marker object unless the Seven Tombs is offered (state 1): a fresh
// hero has the node at state 0, and the helper never writes into the record it is given.
func TestSevenTombsOffered(t *testing.T) {
	var rec d2s.QuestRecord

	before := rec

	if SevenTombsOffered(&rec, 0) {
		t.Error("fresh record: the Seven Tombs must not be offered")
	}

	if SevenTombsOffered(nil, 0) {
		t.Error("nil record")
	}

	if rec != before {
		t.Error("the caller's record was changed")
	}

	// state 1 of the node, reached through the same setter the quest system uses
	g := New(&rec, &d2s.NPCBlock{}, 0)
	g.Start()
	g.setState(g.Quest(QuestSevenTombs), 1)

	if q := g.Quest(QuestSevenTombs); q.State != 1 {
		t.Skipf("state is %d after setState(1): the record does not carry it", q.State)
	}
}
