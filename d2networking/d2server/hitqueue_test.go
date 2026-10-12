package d2server

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2networking/d2netpacket"
	"github.com/OpenDiablo2/OpenDiablo2/d2networking/d2netpacket/d2netpackettype"
)

func pvpCount(c *fakeConn) int {
	n := 0

	for _, p := range c.got {
		if p.PacketType == d2netpackettype.PvPHit {
			n++
		}
	}

	return n
}

// TestAttackerLeavingPurgesPendingHits: the hits a player had queued but not
// yet delivered are removed when he leaves (N16), the others stay in order.
func TestAttackerLeavingPurgesPendingHits(t *testing.T) {
	g, a, b := testServer()
	g.soc.hitQueueOn = true
	g.socialArea("a", 1)
	g.socialArea("b", 1)

	g.soc.mu.Lock()
	_ = g.setHostileLocked("a", "b", true)
	g.soc.mu.Unlock()

	g.socialArea("a", 2) // declared in town, fought in the field
	g.socialArea("b", 2)

	hit, _ := d2netpacket.CreatePvPHitPacket(d2netpacket.PvPHitPacket{Target: "b", Damage: 9, Raw: 50})
	for i := 0; i < 3; i++ {
		if err := g.onPvPHit(a, hit); err != nil {
			t.Fatal(err)
		}
	}

	if g.soc.hits.Pending("b") != 3 {
		t.Fatalf("pending %d", g.soc.hits.Pending("b"))
	}

	g.socialRemovePlayer("a") // the attacker leaves before the flush
	g.flushHits()

	if pvpCount(b) != 0 || g.soc.hits.Pending("b") != 0 {
		t.Fatalf("hits of a departed attacker were delivered (%d)", pvpCount(b))
	}
}
